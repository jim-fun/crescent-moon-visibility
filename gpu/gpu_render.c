// gpu_render.c — OpenCL host for GPU-accelerated crescent moon visibility maps.
//
// This program replaces visibility.out for the "map" mode. It:
//   1. Precomputes dense ephemeris tables using the full-precision Astronomy Engine (CPU)
//   2. Uploads the tables to GPU via OpenCL
//   3. Dispatches the visibility_kernel.cl kernel across all pixels in parallel
//   4. Reads back the image and writes a PNG
//
// Build:
//   Linux:  gcc -O3 -o gpu_visibility.out gpu/gpu_render.c thirdparty/astronomy.c -lm -lOpenCL -I.
//   macOS:  gcc -O3 -o gpu_visibility.out gpu/gpu_render.c thirdparty/astronomy.c -lm -framework OpenCL -I.
//
// Usage (same CLI as visibility.out for map mode):
//   ./gpu_visibility.out 2026-01-18 map evening yallop output.png

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <time.h>

#ifdef __APPLE__
#include <OpenCL/opencl.h>
#else
#include <CL/cl.h>
#endif

#include "thirdparty/astronomy.h"
#include "gpu/chebyshev.h"

#define STB_IMAGE_WRITE_IMPLEMENTATION
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
#include "thirdparty/stb_image_write.h"
#pragma GCC diagnostic pop

#ifndef PIXEL_PER_DEGREE_LON
#define PIXEL_PER_DEGREE_LON 10
#endif
#ifndef PIXEL_PER_DEGREE_LAT
#define PIXEL_PER_DEGREE_LAT 12
#endif

// Chebyshev ephemeris window: 3 days centered on base_time (-1 to +2 days).
// Sampled at CHEB_N_COEFFS GL nodes; each quantity fits in a degree-24 polynomial.
#define EPH_T_START  (-1.0)
#define EPH_T_END    (+2.0)

static const int WIDTH  = 360 * PIXEL_PER_DEGREE_LON;
static const int HEIGHT = 180 * PIXEL_PER_DEGREE_LAT;

// ---------- GMST calculation ----------
// Greenwich Mean Sidereal Time at a given Julian UT date, in hours.
static double compute_gmst(double jd_ut) {
    double T = (jd_ut - 2451545.0) / 36525.0;
    double gmst = 280.46061837 + 360.98564736629 * (jd_ut - 2451545.0)
                + 0.000387933 * T * T - T * T * T / 38710000.0;
    gmst = fmod(gmst, 360.0);
    if (gmst < 0.0) gmst += 360.0;
    return gmst / 15.0;  // convert degrees to hours
}

// ---------- Read kernel source ----------
static char *read_kernel_source(const char *filename, size_t *length) {
    FILE *f = fopen(filename, "r");
    if (!f) {
        // Try relative to executable
        char path[512];
        snprintf(path, sizeof(path), "gpu/%s", filename);
        f = fopen(path, "r");
    }
    if (!f) { fprintf(stderr, "Cannot open kernel: %s\n", filename); return NULL; }
    fseek(f, 0, SEEK_END);
    *length = ftell(f);
    fseek(f, 0, SEEK_SET);
    char *src = (char *)malloc(*length + 1);
    fread(src, 1, *length, f);
    src[*length] = '\0';
    fclose(f);
    return src;
}

// ---------- OpenCL error checker ----------
static void check_cl(cl_int err, const char *msg) {
    if (err != CL_SUCCESS) {
        fprintf(stderr, "OpenCL error %d: %s\n", err, msg);
        exit(1);
    }
}

int main(int argc, const char **argv) {
    if (argc < 6) {
        printf("Usage: %s YYYY-MM-DD map <evening|morning> <yallop|odeh> output.png\n", argv[0]);
        return 1;
    }

    // Parse date
    int year, month, day;
    sscanf(argv[1], "%d-%d-%d", &year, &month, &day);
    astro_time_t base_time = Astronomy_MakeTime(year, month, day, 0, 0, 0);

    int is_evening = strcmp(argv[3], "evening") == 0;
    int is_yallop  = strcmp(argv[4], "yallop") == 0;

    printf("[GPU] Generating %s %s map for %s (%dx%d)\n",
           is_evening ? "evening" : "morning",
           is_yallop  ? "Yallop"  : "Odeh",
           argv[1], WIDTH, HEIGHT);

    // ---- Phase 1: Sample astronomy library at Chebyshev nodes and fit polynomials ----
    struct timespec t0, t1;
    clock_gettime(CLOCK_MONOTONIC, &t0);

    // Sample at CHEB_N_COEFFS Chebyshev-Gauss-Lobatto nodes covering [EPH_T_START, EPH_T_END].
    double node_times[CHEB_N_COEFFS];
    cheb_gl_nodes_in_range(CHEB_DEGREE, EPH_T_START, EPH_T_END, node_times);

    double v_sun_ra   [CHEB_N_COEFFS];
    double v_sun_dec  [CHEB_N_COEFFS];
    double v_moon_ra  [CHEB_N_COEFFS];
    double v_moon_dec [CHEB_N_COEFFS];
    double v_moon_sd  [CHEB_N_COEFFS];
    double v_moon_elong[CHEB_N_COEFFS];

    astro_observer_t geocentric = {0, 0, 0};

    // Additional vectors: the Moon's geocentric position in EQD (AU, x/y/z).
    // The GPU kernel will subtract the per-pixel observer position (via terra())
    // from this vector to get the topocentric apparent position — the missing
    // ~1 classification step of accuracy comes from this parallax correction.
    double v_moon_x[CHEB_N_COEFFS];
    double v_moon_y[CHEB_N_COEFFS];
    double v_moon_z[CHEB_N_COEFFS];

    for (int k = 0; k <= CHEB_DEGREE; k++) {
        astro_time_t t = Astronomy_AddDays(base_time, node_times[k]);

        astro_equatorial_t seq = Astronomy_Equator(BODY_SUN,  &t, geocentric, EQUATOR_OF_DATE, ABERRATION);
        astro_equatorial_t meq = Astronomy_Equator(BODY_MOON, &t, geocentric, EQUATOR_OF_DATE, ABERRATION);
        astro_libration_t  lib = Astronomy_Libration(t);

        v_sun_ra[k]     = seq.ra;
        v_sun_dec[k]    = seq.dec;
        v_moon_ra[k]    = meq.ra;
        v_moon_dec[k]   = meq.dec;
        v_moon_sd[k]    = lib.diam_deg * 60.0 / 2.0;
        v_moon_elong[k] = Astronomy_Elongation(BODY_MOON, t).elongation;

        // Moon's geocentric position vector — in EQJ from GeoVector, rotate to EQD.
        astro_vector_t   geomoon_eqj = Astronomy_GeoVector(BODY_MOON, t, ABERRATION);
        astro_rotation_t rot_eqj_eqd = Astronomy_Rotation_EQJ_EQD(&t);
        astro_vector_t   geomoon_eqd = Astronomy_RotateVector(rot_eqj_eqd, geomoon_eqj);
        v_moon_x[k] = geomoon_eqd.x;
        v_moon_y[k] = geomoon_eqd.y;
        v_moon_z[k] = geomoon_eqd.z;
    }

    // RA is reported in [0, 24) — over a 3-day window the Moon's RA can wrap
    // through 24h. Unwrap so the polynomial fit sees a smooth, monotone-ish
    // function. (Sun RA changes too slowly to wrap in 3 days, but unwrap
    // defensively as well.)
    for (int k = 1; k <= CHEB_DEGREE; k++) {
        while (v_moon_ra[k] - v_moon_ra[k-1] >  12.0) v_moon_ra[k] -= 24.0;
        while (v_moon_ra[k] - v_moon_ra[k-1] < -12.0) v_moon_ra[k] += 24.0;
        while (v_sun_ra[k]  - v_sun_ra[k-1]  >  12.0) v_sun_ra[k]  -= 24.0;
        while (v_sun_ra[k]  - v_sun_ra[k-1]  < -12.0) v_sun_ra[k]  += 24.0;
    }

    double sun_ra_c    [CHEB_N_COEFFS];
    double sun_dec_c   [CHEB_N_COEFFS];
    double moon_ra_c   [CHEB_N_COEFFS];
    double moon_dec_c  [CHEB_N_COEFFS];
    double moon_sd_c   [CHEB_N_COEFFS];
    double moon_elong_c[CHEB_N_COEFFS];
    double moon_x_c    [CHEB_N_COEFFS];
    double moon_y_c    [CHEB_N_COEFFS];
    double moon_z_c    [CHEB_N_COEFFS];
    cheb_compute_coeffs(CHEB_DEGREE, v_sun_ra,    sun_ra_c);
    cheb_compute_coeffs(CHEB_DEGREE, v_sun_dec,   sun_dec_c);
    cheb_compute_coeffs(CHEB_DEGREE, v_moon_ra,   moon_ra_c);
    cheb_compute_coeffs(CHEB_DEGREE, v_moon_dec,  moon_dec_c);
    cheb_compute_coeffs(CHEB_DEGREE, v_moon_sd,   moon_sd_c);
    cheb_compute_coeffs(CHEB_DEGREE, v_moon_elong,moon_elong_c);
    cheb_compute_coeffs(CHEB_DEGREE, v_moon_x,    moon_x_c);
    cheb_compute_coeffs(CHEB_DEGREE, v_moon_y,    moon_y_c);
    cheb_compute_coeffs(CHEB_DEGREE, v_moon_z,    moon_z_c);

    // Precompute new moon times
    astro_search_result_t nm_prev_r = Astronomy_SearchMoonPhase(0, base_time, -35);
    astro_search_result_t nm_next_r = Astronomy_SearchMoonPhase(0, base_time, +35);
    double new_moon_prev = nm_prev_r.time.ut - base_time.ut;
    double new_moon_next = nm_next_r.time.ut - base_time.ut;

    // GMST at base_time
    double gmst0 = compute_gmst(base_time.ut + 2451545.0);

    clock_gettime(CLOCK_MONOTONIC, &t1);
    double precomp_ms = (t1.tv_sec - t0.tv_sec) * 1000.0 + (t1.tv_nsec - t0.tv_nsec) / 1e6;
    printf("[GPU] Chebyshev fit (degree %d, %d samples) in %.1f ms\n",
           CHEB_DEGREE, CHEB_N_COEFFS, precomp_ms);

    // ---- Phase 2: Set up OpenCL ----
    clock_gettime(CLOCK_MONOTONIC, &t0);

    cl_int err;
    cl_platform_id platform;
    cl_uint n_platforms;
    err = clGetPlatformIDs(1, &platform, &n_platforms);
    check_cl(err, "clGetPlatformIDs");

    cl_device_id device;
    err = clGetDeviceIDs(platform, CL_DEVICE_TYPE_GPU, 1, &device, NULL);
    if (err != CL_SUCCESS) {
        printf("[GPU] No GPU found, falling back to CPU OpenCL device\n");
        err = clGetDeviceIDs(platform, CL_DEVICE_TYPE_CPU, 1, &device, NULL);
        check_cl(err, "clGetDeviceIDs (CPU fallback)");
    }

    char dev_name[256];
    clGetDeviceInfo(device, CL_DEVICE_NAME, sizeof(dev_name), dev_name, NULL);
    printf("[GPU] Using device: %s\n", dev_name);

    cl_context ctx = clCreateContext(NULL, 1, &device, NULL, NULL, &err);
    check_cl(err, "clCreateContext");

    cl_command_queue queue = clCreateCommandQueue(ctx, device, 0, &err);
    check_cl(err, "clCreateCommandQueue");

    // Load kernel source
    size_t src_len;
    char *src = read_kernel_source("visibility_kernel.cl", &src_len);
    if (!src) return 1;

    cl_program program = clCreateProgramWithSource(ctx, 1, (const char **)&src, &src_len, &err);
    check_cl(err, "clCreateProgramWithSource");

    // Build options:
    //   -cl-fast-relaxed-math        allow non-IEEE-strict math (we don't depend on signed zero / NaN semantics)
    //   -cl-mad-enable               permit MAD/FMA fusion of separate * and + ops
    //   -cl-no-signed-zeros          allow +0 and -0 to be treated identically (we never test for sign)
    //   -cl-unsafe-math-optimizations enable algebraic rewrites (a/b * c == a * c/b etc.)
    const char *build_opts = "-cl-fast-relaxed-math -cl-mad-enable "
                             "-cl-no-signed-zeros -cl-unsafe-math-optimizations";
    err = clBuildProgram(program, 1, &device, build_opts, NULL, NULL);
    if (err != CL_SUCCESS) {
        char log[16384];
        clGetProgramBuildInfo(program, device, CL_PROGRAM_BUILD_LOG, sizeof(log), log, NULL);
        fprintf(stderr, "Kernel build failed:\n%s\n", log);
        return 1;
    }

    cl_kernel kernel = clCreateKernel(program, "visibility_map", &err);
    check_cl(err, "clCreateKernel");

    // ---- Phase 3: Upload buffers ----
    size_t coeff_bytes = CHEB_N_COEFFS * sizeof(double);
    size_t img_bytes = WIDTH * HEIGHT * sizeof(cl_uint);

    cl_mem d_sun_ra_c    = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, coeff_bytes, sun_ra_c, &err);
    cl_mem d_sun_dec_c   = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, coeff_bytes, sun_dec_c, &err);
    cl_mem d_moon_ra_c   = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, coeff_bytes, moon_ra_c, &err);
    cl_mem d_moon_dec_c  = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, coeff_bytes, moon_dec_c, &err);
    cl_mem d_moon_sd_c   = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, coeff_bytes, moon_sd_c, &err);
    cl_mem d_moon_elong_c = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, coeff_bytes, moon_elong_c, &err);
    cl_mem d_moon_x_c    = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, coeff_bytes, moon_x_c, &err);
    cl_mem d_moon_y_c    = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, coeff_bytes, moon_y_c, &err);
    cl_mem d_moon_z_c    = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, coeff_bytes, moon_z_c, &err);
    cl_mem d_image     = clCreateBuffer(ctx, CL_MEM_WRITE_ONLY, img_bytes, NULL, &err);
    check_cl(err, "clCreateBuffer");

    // ---- Phase 4: Set kernel arguments and dispatch ----
    int w = WIDTH, h = HEIGHT;
    int ppd_lon = PIXEL_PER_DEGREE_LON, ppd_lat = PIXEL_PER_DEGREE_LAT;
    double eph_t_start = EPH_T_START;
    double eph_t_end   = EPH_T_END;
    int    cheb_degree = CHEB_DEGREE;

    clSetKernelArg(kernel,  0, sizeof(cl_mem), &d_sun_ra_c);
    clSetKernelArg(kernel,  1, sizeof(cl_mem), &d_sun_dec_c);
    clSetKernelArg(kernel,  2, sizeof(cl_mem), &d_moon_ra_c);
    clSetKernelArg(kernel,  3, sizeof(cl_mem), &d_moon_dec_c);
    clSetKernelArg(kernel,  4, sizeof(cl_mem), &d_moon_sd_c);
    clSetKernelArg(kernel,  5, sizeof(cl_mem), &d_moon_elong_c);
    clSetKernelArg(kernel,  6, sizeof(cl_mem), &d_moon_x_c);
    clSetKernelArg(kernel,  7, sizeof(cl_mem), &d_moon_y_c);
    clSetKernelArg(kernel,  8, sizeof(cl_mem), &d_moon_z_c);
    clSetKernelArg(kernel,  9, sizeof(double), &gmst0);
    clSetKernelArg(kernel, 10, sizeof(double), &base_time.ut);
    clSetKernelArg(kernel, 11, sizeof(double), &new_moon_prev);
    clSetKernelArg(kernel, 12, sizeof(double), &new_moon_next);
    clSetKernelArg(kernel, 13, sizeof(double), &eph_t_start);
    clSetKernelArg(kernel, 14, sizeof(double), &eph_t_end);
    clSetKernelArg(kernel, 15, sizeof(int),    &cheb_degree);
    clSetKernelArg(kernel, 16, sizeof(int),    &w);
    clSetKernelArg(kernel, 17, sizeof(int),    &h);
    clSetKernelArg(kernel, 18, sizeof(int),    &ppd_lon);
    clSetKernelArg(kernel, 19, sizeof(int),    &ppd_lat);
    clSetKernelArg(kernel, 20, sizeof(int),    &is_evening);
    clSetKernelArg(kernel, 21, sizeof(int),    &is_yallop);
    clSetKernelArg(kernel, 22, sizeof(cl_mem), &d_image);

    size_t global_work_size = (size_t)WIDTH * HEIGHT;
    // Round up to multiple of 256 for efficiency
    size_t local_work_size = 256;
    global_work_size = ((global_work_size + local_work_size - 1) / local_work_size) * local_work_size;

    clock_gettime(CLOCK_MONOTONIC, &t1);
    double setup_ms = (t1.tv_sec - t0.tv_sec) * 1000.0 + (t1.tv_nsec - t0.tv_nsec) / 1e6;
    printf("[GPU] OpenCL setup + upload in %.1f ms\n", setup_ms);

    clock_gettime(CLOCK_MONOTONIC, &t0);

    err = clEnqueueNDRangeKernel(queue, kernel, 1, NULL, &global_work_size, &local_work_size, 0, NULL, NULL);
    check_cl(err, "clEnqueueNDRangeKernel");
    clFinish(queue);

    clock_gettime(CLOCK_MONOTONIC, &t1);
    double kernel_ms = (t1.tv_sec - t0.tv_sec) * 1000.0 + (t1.tv_nsec - t0.tv_nsec) / 1e6;
    printf("[GPU] Kernel executed in %.1f ms (%.1f Mpx/s)\n",
           kernel_ms, (WIDTH * HEIGHT / 1e6) / (kernel_ms / 1000.0));

    // ---- Phase 5: Read back and write PNG ----
    cl_uint *image = (cl_uint *)calloc(WIDTH * HEIGHT, sizeof(cl_uint));
    clEnqueueReadBuffer(queue, d_image, CL_TRUE, 0, img_bytes, image, 0, NULL, NULL);

    stbi_write_png(argv[5], WIDTH, HEIGHT, 4, image, WIDTH * 4);
    printf("[GPU] Wrote %s (%.2f MB)\n", argv[5], (WIDTH * HEIGHT * 4) / (1024.0 * 1024.0));

    // Cleanup
    clReleaseMemObject(d_sun_ra_c);
    clReleaseMemObject(d_sun_dec_c);
    clReleaseMemObject(d_moon_ra_c);
    clReleaseMemObject(d_moon_dec_c);
    clReleaseMemObject(d_moon_sd_c);
    clReleaseMemObject(d_moon_elong_c);
    clReleaseMemObject(d_moon_x_c);
    clReleaseMemObject(d_moon_y_c);
    clReleaseMemObject(d_moon_z_c);
    clReleaseMemObject(d_image);
    clReleaseKernel(kernel);
    clReleaseProgram(program);
    clReleaseCommandQueue(queue);
    clReleaseContext(ctx);
    free(image); free(src);

    return 0;
}
