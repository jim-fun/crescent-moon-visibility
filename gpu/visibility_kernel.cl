// visibility_kernel.cl — OpenCL kernel for crescent moon visibility computation.
//
// This kernel receives precomputed ephemeris lookup tables from the CPU host
// (sun/moon positions at fine time intervals) and performs per-pixel:
//   1. Horizon coordinate transforms (equatorial → local alt/az)
//   2. Rise/set time binary search
//   3. Visibility parameter computation
//   4. Yallop/Odeh category classification
//
// Compatible with: macOS (Metal/OpenCL), Linux NVIDIA (CUDA/OpenCL),
//                  Linux AMD (ROCm/OpenCL)

#pragma OPENCL EXTENSION cl_khr_fp64 : enable

#define DEG2RAD 0.017453292519943295
#define RAD2DEG 57.29577951308232
#define PI_VAL  3.14159265358979323846


// ---------- helpers ----------

// Compute local sidereal time (hours) for a given UT offset (days) and longitude (degrees).
// gmst0 is the GMST at the base_time epoch, precomputed on host.
double local_sidereal_time(double ut_offset_days, double longitude_deg, double gmst0_hours) {
    // Earth rotates 360.98564736629 deg/day relative to stars
    double gmst = gmst0_hours + ut_offset_days * 24.0 * 1.00273790935;
    double lst = gmst + longitude_deg / 15.0;
    // Normalize to [0,24)
    lst = fmod(lst, 24.0);
    if (lst < 0.0) lst += 24.0;
    return lst;
}

// Equatorial (RA hours, Dec degrees) → altitude (radians) for observer at lat_rad.
// Uses native_sin / native_cos because angle accuracy is bounded (input always
// in [-2π, 2π]) and the GPU hardware sin/cos units are 4–8× faster than the
// IEEE-rounded library versions. The final asin is unavoidable.
double eq_to_alt(double ra_hours, double dec_deg, double lat_rad, double lst_hours) {
    double ha_rad = (lst_hours - ra_hours) * 15.0 * DEG2RAD;
    double dec_rad = dec_deg * DEG2RAD;
    return asin(native_sin(lat_rad) * native_sin(dec_rad)
              + native_cos(lat_rad) * native_cos(dec_rad) * native_cos(ha_rad));
}

// Equatorial → azimuth (radians, measured from North through East)
double eq_to_az(double ra_hours, double dec_deg, double lat_rad, double lst_hours, double alt_rad) {
    double ha_rad = (lst_hours - ra_hours) * 15.0 * DEG2RAD;
    double dec_rad = dec_deg * DEG2RAD;
    double cos_lat = native_cos(lat_rad);
    double sin_az_num = -native_sin(ha_rad) * native_cos(dec_rad);
    double cos_az_num = native_sin(dec_rad) * cos_lat
                      - native_cos(dec_rad) * native_sin(lat_rad) * native_cos(ha_rad);
    double az = atan2(sin_az_num, cos_az_num);
    if (az < 0.0) az += 2.0 * PI_VAL;
    return az;
}

// Map a physical UT offset (days from base_time) to the Chebyshev abscissa
// x ∈ [-1, +1] used for polynomial evaluation. The fit was computed over
// [t_start, t_end] (typically [-1, +2] days from base_time).
double ut_to_cheb_x(double ut_offset, double t_start, double t_end) {
    double t_center   = 0.5 * (t_end + t_start);
    double t_halfspan = 0.5 * (t_end - t_start);
    return (ut_offset - t_center) / t_halfspan;
}

// Evaluate sum_{j=0}^{N} c_j * T_j(x) via Clenshaw recurrence.
// Stable for |x| <= 1 and ~N FMAs per call. Coefficients live in __constant
// memory so reads hit the constant cache (much lower latency than __global
// L2 reads on NVIDIA / AMD / Intel GPUs).
double cheb_eval(__constant const double *c, int N, double x) {
    double two_x = 2.0 * x;
    double b1 = 0.0, b2 = 0.0;
    for (int j = N; j >= 1; j--) {
        double bj = two_x * b1 - b2 + c[j];
        b2 = b1;
        b1 = bj;
    }
    return x * b1 - b2 + c[0];
}

// Binary search for the time when a body crosses the horizon (altitude = threshold).
// direction: +1 = search for setting (alt goes below threshold)
//           -1 = search for rising  (alt goes above threshold)
// Returns the UT offset (days) of the crossing, or -9999 on failure.
double search_rise_set(
    __constant const double *body_ra_c,
    __constant const double *body_dec_c,
    int cheb_degree, double eph_t_start, double eph_t_end,
    double lat_rad, double longitude_deg, double gmst0,
    double threshold_rad, int is_setting,
    double search_start, double search_end)
{
    // Coarse scan: 32 steps × ~45 minutes covers the 1-day window. The sun
    // and moon altitude change at ~15°/hour near the horizon, so a 45-minute
    // step easily brackets the threshold crossing while bisection refines.
    // (Old value was 200 — 9× more work for the same final precision.)
    int n_scan = 32;
    double dt = (search_end - search_start) / (double)n_scan;
    double prev_alt = 0.0;
    double t_lo = -9999.0, t_hi = -9999.0;
    int found = 0;

    for (int k = 0; k <= n_scan; k++) {
        double t = search_start + k * dt;
        double x = ut_to_cheb_x(t, eph_t_start, eph_t_end);
        double ra  = cheb_eval(body_ra_c,  cheb_degree, x);
        double dec = cheb_eval(body_dec_c, cheb_degree, x);
        double lst = local_sidereal_time(t, longitude_deg, gmst0);
        double alt = eq_to_alt(ra, dec, lat_rad, lst);
        double val = alt - threshold_rad;

        if (k > 0) {
            // For setting: look for val going from positive to negative
            // For rising:  look for val going from negative to positive
            if (is_setting && prev_alt > 0.0 && val <= 0.0) {
                t_lo = search_start + (k - 1) * dt;
                t_hi = t;
                found = 1;
                break;
            }
            if (!is_setting && prev_alt < 0.0 && val >= 0.0) {
                t_lo = search_start + (k - 1) * dt;
                t_hi = t;
                found = 1;
                break;
            }
        }
        prev_alt = val;
    }

    if (!found) return -9999.0;

    // Bisection from a 45-minute window: 12 iterations → ~0.7 second precision.
    // The Yallop value depends on moon altitude (changes ~30 arcsec/sec near
    // the horizon), so sub-second timing yields sub-arcsec altitude error,
    // far below the classification boundaries. (Old value was 20 — 8 extra
    // iterations buying nothing.)
    for (int iter = 0; iter < 12; iter++) {
        double t_mid = 0.5 * (t_lo + t_hi);
        double x = ut_to_cheb_x(t_mid, eph_t_start, eph_t_end);
        double ra  = cheb_eval(body_ra_c,  cheb_degree, x);
        double dec = cheb_eval(body_dec_c, cheb_degree, x);
        double lst = local_sidereal_time(t_mid, longitude_deg, gmst0);
        double alt = eq_to_alt(ra, dec, lat_rad, lst) - threshold_rad;

        if ((is_setting && alt > 0.0) || (!is_setting && alt < 0.0))
            t_lo = t_mid;
        else
            t_hi = t_mid;
    }
    return 0.5 * (t_lo + t_hi);
}

// ---------- main kernel ----------

__kernel void visibility_map(
    // Chebyshev coefficients for each ephemeris quantity (degree+1 doubles each)
    // Placed in __constant memory: ~9 × 25 × 8 = 1.8 KB total, well under the
    // 64 KB minimum constant-cache size on all conformant OpenCL devices.
    __constant const double *sun_ra_c,    // RA in hours (geocentric, for sunset search)
    __constant const double *sun_dec_c,   // Dec in degrees
    __constant const double *moon_ra_c,   // RA in hours (geocentric, for moonset search)
    __constant const double *moon_dec_c,  // Dec in degrees
    __constant const double *moon_sd_c,   // semi-diameter arcmin
    __constant const double *moon_elong_c,// geocentric elongation deg
    // Moon's geocentric position vector in EQD (AU) — fitted X/Y/Z components.
    // Used to derive topocentric Yallop ARCV from a geocentric vector at t_best.
    __constant const double *moon_x_c,
    __constant const double *moon_y_c,
    __constant const double *moon_z_c,
    // Scalar parameters
    double gmst0,           // GMST at base_time (hours)
    double base_ut,         // base_time UT value (days)
    double new_moon_prev,   // UT offset of previous new moon
    double new_moon_next,   // UT offset of next new moon
    double eph_t_start,     // start UT offset of Chebyshev fit window
    double eph_t_end,       // end   UT offset of Chebyshev fit window
    int    cheb_degree,     // polynomial degree (n_coeffs - 1)
    int    width,           // image width in pixels
    int    height,          // image height in pixels
    int    ppd_lon,         // pixels per degree longitude
    int    ppd_lat,         // pixels per degree latitude
    int    is_evening,      // 1 = evening (setting), 0 = morning (rising)
    int    is_yallop,       // 1 = yallop, 0 = odeh
    // Output
    __global uint *image    // [width * height] ABGR pixel colors
)
{
    int gid = get_global_id(0);
    if (gid >= width * height) return;

    int px = gid % width;
    int py = gid / width;

    double latitude  = ((double)(height - (py + 1)) / (double)ppd_lat) + (-90.0);
    double longitude = ((double)px / (double)ppd_lon) + (-180.0);

    // Latitude capping at ±60°
    if (latitude > 60.0 || latitude < -60.0) {
        image[gid] = 0x00000000;
        return;
    }

    double lat_rad = latitude * DEG2RAD;

    // Time adjustment for longitude
    double lon_offset = -longitude / 360.0;  // days

    // Sun threshold: -0.8333 degrees (standard refraction + solar semi-diameter)
    double sun_thresh = -0.8333 * DEG2RAD;
    // Moon threshold: approximate (ignoring parallax for rise/set search)
    double moon_thresh = 0.125 * DEG2RAD;

    // Search window: forward 1 day from longitude-adjusted base. This matches
    // the CPU renderer's Astronomy_SearchRiseSet(time, 1) call exactly. A
    // symmetric ±0.75-day window would otherwise pick up the *previous*
    // day's sunset (the first sign change in the window) and shift the
    // entire computation 24 h backwards.
    double search_lo = lon_offset;
    double search_hi = lon_offset + 1.0;

    int setting = is_evening ? 1 : 0;

    double t_sun = search_rise_set(sun_ra_c, sun_dec_c, cheb_degree, eph_t_start, eph_t_end,
                                   lat_rad, longitude, gmst0, sun_thresh, setting,
                                   search_lo, search_hi);
    double t_moon = search_rise_set(moon_ra_c, moon_dec_c, cheb_degree, eph_t_start, eph_t_end,
                                    lat_rad, longitude, gmst0, moon_thresh, setting,
                                    search_lo, search_hi);

    if (t_sun < -9000.0 || t_moon < -9000.0) {
        image[gid] = 0x00000000; // 'H' — no rise/set
        return;
    }

    double lag_time = (t_moon - t_sun) * (is_evening ? 1.0 : -1.0);

    double t_best = (lag_time < 0.0)
        ? t_sun
        : t_sun + lag_time * 4.0 / 9.0 * (is_evening ? 1.0 : -1.0);

    // New moon determination
    double nm_nearest = (fabs(t_sun - new_moon_prev) <= fabs(new_moon_next - t_sun))
                        ? new_moon_prev : new_moon_next;

    // Moon age line
    int draw_moon_line = (((int)round((t_best - nm_nearest) * 24.0 * 20.0)) % 20) == 0;

    int before_new_moon = ((t_sun - nm_nearest) * (is_evening ? 1.0 : -1.0)) < 0.0;

    // Early exits — bytes in memory (LE): R, G, B, A
    uint color = 0x00000000;
    if (lag_time < 0.0 && before_new_moon) {
        color = 0x00000000; // 'J' — pre-conjunction + moonset before sunset, transparent
        if (draw_moon_line) color = 0xFFFFFFFF;
        image[gid] = color;
        return;
    }
    if (lag_time < 0.0) {
        // 'I' — moonset before sunset
        if (draw_moon_line) color = 0xFFFFFFFF;
        image[gid] = color;
        return;
    }
    if (before_new_moon) {
        color = 0x00000000; // 'G' — pre-conjunction, transparent
        if (draw_moon_line) color = 0xFFFFFFFF;
        image[gid] = color;
        return;
    }

    // Compute astronomical parameters at best time via Chebyshev evaluation.
    double x_best = ut_to_cheb_x(t_best, eph_t_start, eph_t_end);

    double s_ra  = cheb_eval(sun_ra_c,    cheb_degree, x_best);
    double s_dec = cheb_eval(sun_dec_c,   cheb_degree, x_best);
    double SD    = cheb_eval(moon_sd_c,   cheb_degree, x_best);
    double ARCL  = cheb_eval(moon_elong_c,cheb_degree, x_best);

    double lst_best = local_sidereal_time(t_best, longitude, gmst0);

    // For Yallop the CPU uses GEOCENTRIC moon RA/Dec (visibility.cc:116-125 —
    // Astronomy_GeoVector → EQD → EquatorFromVector → Astronomy_Horizon with
    // observer lat/lon for the alt/az transform but *not* for parallax).
    // Derive geocentric RA/Dec from the polynomial-fitted EQD position vector.
    double moon_geo_x = cheb_eval(moon_x_c, cheb_degree, x_best);
    double moon_geo_y = cheb_eval(moon_y_c, cheb_degree, x_best);
    double moon_geo_z = cheb_eval(moon_z_c, cheb_degree, x_best);

    double moon_geo_r = sqrt(moon_geo_x * moon_geo_x + moon_geo_y * moon_geo_y + moon_geo_z * moon_geo_z);
    double m_dec = asin(moon_geo_z / moon_geo_r) * RAD2DEG;
    double m_ra  = atan2(moon_geo_y, moon_geo_x) * RAD2DEG / 15.0;  // hours
    if (m_ra < 0.0) m_ra += 24.0;

    double sun_alt  = eq_to_alt(s_ra, s_dec, lat_rad, lst_best);
    double sun_az   = eq_to_az(s_ra, s_dec, lat_rad, lst_best, sun_alt);
    double moon_alt = eq_to_alt(m_ra, m_dec, lat_rad, lst_best);
    double moon_az  = eq_to_az(m_ra, m_dec, lat_rad, lst_best, moon_alt);

    double lunar_parallax = SD / 0.27245;
    double SD_topo = SD * (1.0 + native_sin(moon_alt) * native_sin(lunar_parallax / 60.0 * DEG2RAD));
    double DAZ = sun_az * RAD2DEG - moon_az * RAD2DEG;
    double W_topo = SD_topo * (1.0 - native_cos(ARCL * DEG2RAD));

    double ARCV;
    if (is_yallop) {
        // Yallop uses geocentric positions for ARCV
        ARCV = moon_alt * RAD2DEG - sun_alt * RAD2DEG;
    } else {
        double COSARCV = native_cos(ARCL * DEG2RAD) / native_cos(DAZ * DEG2RAD);
        COSARCV = clamp(COSARCV, -1.0, 1.0);
        ARCV = acos(COSARCV) * RAD2DEG;
    }

    // Classification
    double value;
    char result;
    if (is_yallop) {
        value = (ARCV - (11.8371 - 6.3226 * W_topo + 0.7319 * W_topo * W_topo - 0.1018 * W_topo * W_topo * W_topo)) / 10.0;
        if      (value >  0.216) result = 'A';
        else if (value > -0.014) result = 'B';
        else if (value > -0.160) result = 'C';
        else if (value > -0.232) result = 'D';
        else if (value > -0.293) result = 'E';
        else                     result = 'F';
    } else {
        value = ARCV - (7.1651 - 6.3226 * W_topo + 0.7319 * W_topo * W_topo - 0.1018 * W_topo * W_topo * W_topo);
        if      (value >= 5.65)  result = 'A';
        else if (value >= 2.00)  result = 'C';
        else if (value >= -0.96) result = 'E';
        else                     result = 'F';
    }

    // ABGR layout — bytes in memory (LE): R, G, B, A.  Matches CPU renderer.
    if      (result == 'A') color = 0xFFCCCC00; // A: Cyan — easily visible
    else if (result == 'B') color = 0xFFB3B300; // B: Darker cyan — perfect conditions
    else if (result == 'C') color = 0xFF1AFFFF; // C: Light cyan — may need optical aid
    else if (result == 'D') color = 0xFF00E6E6; // D: Bright cyan — will need optical aid
    else if (result == 'E') color = 0xFF00B3B3; // E: Darker cyan — not visible w/o telescope
    else                    color = 0x00000000; // F: Not visible — transparent

    if (draw_moon_line) color = 0xFFFFFFFF;
    image[gid] = color;
}
