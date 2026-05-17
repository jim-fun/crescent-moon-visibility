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

// Ephemeris table: 3 days at 30-second intervals = 8640 steps
#define EPH_DAYS      3.0
#define EPH_STEP_SEC  30.0
#define EPH_N_STEPS   ((int)(EPH_DAYS * 86400.0 / EPH_STEP_SEC))

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

    // ---- Phase 1: Precompute ephemeris on CPU ----
    struct timespec t0, t1;
    clock_gettime(CLOCK_MONOTONIC, &t0);

    double eph_start = -1.0;  // start 1 day before base_time
    double eph_step  = EPH_STEP_SEC / 86400.0;  // in days

    double *sun_ra    = (double *)malloc(EPH_N_STEPS * sizeof(double));
    double *sun_dec   = (double *)malloc(EPH_N_STEPS * sizeof(double));
    double *moon_ra   = (double *)malloc(EPH_N_STEPS * sizeof(double));
    double *moon_dec  = (double *)malloc(EPH_N_STEPS * sizeof(double));
    double *moon_sd   = (double *)malloc(EPH_N_STEPS * sizeof(double));
    double *moon_elong = (double *)malloc(EPH_N_STEPS * sizeof(double));

    astro_observer_t geocentric = {0, 0, 0};

    for (int i = 0; i < EPH_N_STEPS; i++) {
        double offset = eph_start + i * eph_step;
        astro_time_t t = Astronomy_AddDays(base_time, offset);

        astro_equatorial_t seq = Astronomy_Equator(BODY_SUN,  &t, geocentric, EQUATOR_OF_DATE, ABERRATION);
        astro_equatorial_t meq = Astronomy_Equator(BODY_MOON, &t, geocentric, EQUATOR_OF_DATE, ABERRATION);
        astro_libration_t  lib = Astronomy_Libration(t);

        sun_ra[i]     = seq.ra;
        sun_dec[i]    = seq.dec;
        moon_ra[i]    = meq.ra;
        moon_dec[i]   = meq.dec;
        moon_sd[i]    = lib.diam_deg * 60.0 / 2.0;  // arcminutes
        moon_elong[i] = Astronomy_Elongation(BODY_MOON, t).elongation;
    }

    // Precompute new moon times
    astro_search_result_t nm_prev_r = Astronomy_SearchMoonPhase(0, base_time, -35);
    astro_search_result_t nm_next_r = Astronomy_SearchMoonPhase(0, base_time, +35);
    double new_moon_prev = nm_prev_r.time.ut - base_time.ut;
    double new_moon_next = nm_next_r.time.ut - base_time.ut;

    // GMST at base_time
    double gmst0 = compute_gmst(base_time.ut + 2451545.0);

    clock_gettime(CLOCK_MONOTONIC, &t1);
    double precomp_ms = (t1.tv_sec - t0.tv_sec) * 1000.0 + (t1.tv_nsec - t0.tv_nsec) / 1e6;
    printf("[GPU] Ephemeris precomputed (%d steps) in %.1f ms\n", EPH_N_STEPS, precomp_ms);

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

    err = clBuildProgram(program, 1, &device, "-cl-fast-relaxed-math", NULL, NULL);
    if (err != CL_SUCCESS) {
        char log[16384];
        clGetProgramBuildInfo(program, device, CL_PROGRAM_BUILD_LOG, sizeof(log), log, NULL);
        fprintf(stderr, "Kernel build failed:\n%s\n", log);
        return 1;
    }

    cl_kernel kernel = clCreateKernel(program, "visibility_map", &err);
    check_cl(err, "clCreateKernel");

    // ---- Phase 3: Upload buffers ----
    size_t eph_bytes = EPH_N_STEPS * sizeof(double);
    size_t img_bytes = WIDTH * HEIGHT * sizeof(cl_uint);

    cl_mem d_sun_ra    = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, eph_bytes, sun_ra, &err);
    cl_mem d_sun_dec   = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, eph_bytes, sun_dec, &err);
    cl_mem d_moon_ra   = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, eph_bytes, moon_ra, &err);
    cl_mem d_moon_dec  = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, eph_bytes, moon_dec, &err);
    cl_mem d_moon_sd   = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, eph_bytes, moon_sd, &err);
    cl_mem d_moon_elong = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR, eph_bytes, moon_elong, &err);
    cl_mem d_image     = clCreateBuffer(ctx, CL_MEM_WRITE_ONLY, img_bytes, NULL, &err);
    check_cl(err, "clCreateBuffer");

    // ---- Phase 4: Set kernel arguments and dispatch ----
    int w = WIDTH, h = HEIGHT;
    int ppd_lon = PIXEL_PER_DEGREE_LON, ppd_lat = PIXEL_PER_DEGREE_LAT;

    clSetKernelArg(kernel,  0, sizeof(cl_mem), &d_sun_ra);
    clSetKernelArg(kernel,  1, sizeof(cl_mem), &d_sun_dec);
    clSetKernelArg(kernel,  2, sizeof(cl_mem), &d_moon_ra);
    clSetKernelArg(kernel,  3, sizeof(cl_mem), &d_moon_dec);
    clSetKernelArg(kernel,  4, sizeof(cl_mem), &d_moon_sd);
    clSetKernelArg(kernel,  5, sizeof(cl_mem), &d_moon_elong);
    clSetKernelArg(kernel,  6, sizeof(double), &gmst0);
    clSetKernelArg(kernel,  7, sizeof(double), &base_time.ut);
    clSetKernelArg(kernel,  8, sizeof(double), &new_moon_prev);
    clSetKernelArg(kernel,  9, sizeof(double), &new_moon_next);
    clSetKernelArg(kernel, 10, sizeof(double), &eph_start);
    clSetKernelArg(kernel, 11, sizeof(double), &eph_step);
    int n_steps = EPH_N_STEPS;
    clSetKernelArg(kernel, 12, sizeof(int),    &n_steps);
    clSetKernelArg(kernel, 13, sizeof(int),    &w);
    clSetKernelArg(kernel, 14, sizeof(int),    &h);
    clSetKernelArg(kernel, 15, sizeof(int),    &ppd_lon);
    clSetKernelArg(kernel, 16, sizeof(int),    &ppd_lat);
    clSetKernelArg(kernel, 17, sizeof(int),    &is_evening);
    clSetKernelArg(kernel, 18, sizeof(int),    &is_yallop);
    clSetKernelArg(kernel, 19, sizeof(cl_mem), &d_image);

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
    clReleaseMemObject(d_sun_ra);
    clReleaseMemObject(d_sun_dec);
    clReleaseMemObject(d_moon_ra);
    clReleaseMemObject(d_moon_dec);
    clReleaseMemObject(d_moon_sd);
    clReleaseMemObject(d_moon_elong);
    clReleaseMemObject(d_image);
    clReleaseKernel(kernel);
    clReleaseProgram(program);
    clReleaseCommandQueue(queue);
    clReleaseContext(ctx);
    free(sun_ra); free(sun_dec); free(moon_ra); free(moon_dec);
    free(moon_sd); free(moon_elong); free(image); free(src);

    return 0;
}
