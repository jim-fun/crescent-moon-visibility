// MIT, @ebraminio and @hidp123

#include <cstdio>
#include <cstdlib>
#include <cmath>
#include <cstdint>
#include <vector>

#include "thirdparty/astronomy.h"
#define STB_IMAGE_WRITE_IMPLEMENTATION

#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
#include "thirdparty/stb_image_write.h"
#pragma GCC diagnostic pop

// To be passed compiled time
#ifndef PIXEL_PER_DEGREE_LON
#define PIXEL_PER_DEGREE_LON 4
#endif
#ifndef PIXEL_PER_DEGREE_LAT
#define PIXEL_PER_DEGREE_LAT 4
#endif

const unsigned pixelsPerDegreeLon = PIXEL_PER_DEGREE_LON;
const unsigned pixelsPerDegreeLat = PIXEL_PER_DEGREE_LAT;
const int minLatitude = -90;
const int maxLatitude = +90;
const int minLongitude = -180;
const int maxLongitude = +180;
const unsigned width = (maxLongitude - minLongitude) * pixelsPerDegreeLon;
const unsigned height = (maxLatitude - minLatitude) * pixelsPerDegreeLat;

struct details_t
{
    astro_time_t sunset_sunrise, moonset_moonrise, best_time, new_moon_prev, new_moon_next;
    double lag_time, moon_age_prev, moon_age_next;
    double sd, lunar_parallax, arcl, arcv, daz, w_topo, sd_topo, value;
    double moon_azimuth, moon_altitude, moon_ra, moon_dec;
    double sun_azimuth, sun_altitude, sun_ra, sun_dec;
};

template <bool evening, bool yallop>
static char calculate(
    double latitude, double longitude, double altitude, astro_time_t base_time,
    /* optional, used for table extra results */ details_t *details = nullptr,
    /* optional, used as an option in table results */ bool ignore_besttime = false,
    /* optional, used for moon ages lines in map */ bool *draw_moon_line = nullptr,
    /* optional, used for first visibility points in map */ double *result_time = nullptr,
    /* optional, used for red line in map */ double *q_value = nullptr)
{
    astro_time_t time = Astronomy_AddDays(base_time, -longitude / 360);
    astro_observer_t observer = {.latitude = latitude, .longitude = longitude, .height = altitude};

    astro_direction_t direction = evening ? DIRECTION_SET : DIRECTION_RISE;
    astro_search_result_t sunset_sunrise = Astronomy_SearchRiseSet(BODY_SUN, observer, direction, time, 1);
    astro_search_result_t moonset_moonrise = Astronomy_SearchRiseSet(BODY_MOON, observer, direction, time, 1);
    if (sunset_sunrise.status != ASTRO_SUCCESS || moonset_moonrise.status != ASTRO_SUCCESS)
        return 'H'; // No sun{set,rise} or moon{set,rise}
    double lag_time = (moonset_moonrise.time.ut - sunset_sunrise.time.ut) * (evening ? 1 : -1);
    if (details)
    {
        details->lag_time = lag_time;
        details->moonset_moonrise = moonset_moonrise.time;
        details->sunset_sunrise = sunset_sunrise.time;
    }
    astro_time_t best_time = (lag_time < 0 || ignore_besttime)
                                 ? sunset_sunrise.time
                                 : Astronomy_AddDays(sunset_sunrise.time, lag_time * 4 / 9 * (evening ? 1 : -1));
    if (result_time)
        *result_time = best_time.ut;
    astro_time_t new_moon_prev = Astronomy_SearchMoonPhase(0, sunset_sunrise.time, -35).time;
    astro_time_t new_moon_next = Astronomy_SearchMoonPhase(0, sunset_sunrise.time, +35).time;
    astro_time_t new_moon_nearest = (sunset_sunrise.time.ut - new_moon_prev.ut) <= (new_moon_next.ut - sunset_sunrise.time.ut)
                                        ? new_moon_prev
                                        : new_moon_next;
    if (details)
    {
        details->new_moon_prev = new_moon_prev;
        details->new_moon_next = new_moon_next;
    }
    if (draw_moon_line)
        *draw_moon_line = ((int)round((best_time.ut - new_moon_nearest.ut) * 24 * 20) % 20) == 0;
    if (details)
    {
        details->moon_age_prev = best_time.ut - new_moon_prev.ut;
        details->moon_age_next = best_time.ut - new_moon_next.ut;
    }
    bool before_new_moon = (sunset_sunrise.time.ut - new_moon_nearest.ut) * (evening ? 1 : -1) < 0;
    if (lag_time < 0 && before_new_moon)
        return 'J'; // Checks both of the conditions on the two below lines, shows a mixed color
    if (lag_time < 0)
        return 'I'; // Moonset before sunset / Moonrise after sunrise
    if (before_new_moon)
        return 'G'; // Sunset is before new moon / Sunrise is after new moon

    astro_equatorial_t sun_equator = Astronomy_Equator(BODY_SUN, &best_time, observer, EQUATOR_OF_DATE, ABERRATION);
    astro_horizon_t sun_horizon = Astronomy_Horizon(&best_time, observer, sun_equator.ra, sun_equator.dec, REFRACTION_NONE);
    astro_equatorial_t moon_equator = Astronomy_Equator(BODY_MOON, &best_time, observer, EQUATOR_OF_DATE, ABERRATION);
    astro_horizon_t moon_horizon = Astronomy_Horizon(&best_time, observer, moon_equator.ra, moon_equator.dec, REFRACTION_NONE);
    astro_libration_t liberation = Astronomy_Libration(best_time);

    double SD = liberation.diam_deg * 60 / 2; // Semi-diameter of the Moon in arcminutes, geocentric
    double lunar_parallax = SD / 0.27245;     // In arcminutes
    // As SD_topo should be in arcminutes as SD, but moon_alt and lunar_parallax are in degrees, it is divided by 60.
    double SD_topo = SD * (1 + sin(moon_horizon.altitude * DEG2RAD) * sin(lunar_parallax / 60 * DEG2RAD));

    double ARCL = yallop
                      ? Astronomy_Elongation(BODY_MOON, best_time).elongation            // Geocentric elongation in Yallop
                      : Astronomy_AngleBetween(sun_equator.vec, moon_equator.vec).angle; // Topocentric elongation in Odeh

    double DAZ = sun_horizon.azimuth - moon_horizon.azimuth;
    double ARCV;
    if (yallop)
    {
        astro_vector_t geomoon = Astronomy_GeoVector(BODY_MOON, best_time, ABERRATION);
        astro_vector_t geosun = Astronomy_GeoVector(BODY_SUN, best_time, ABERRATION);
        astro_rotation_t rot = Astronomy_Rotation_EQJ_EQD(&best_time);
        astro_vector_t rotmoon = Astronomy_RotateVector(rot, geomoon);
        astro_vector_t rotsun = Astronomy_RotateVector(rot, geosun);
        astro_equatorial_t meq = Astronomy_EquatorFromVector(rotmoon);
        astro_equatorial_t seq = Astronomy_EquatorFromVector(rotsun);
        astro_horizon_t mhor = Astronomy_Horizon(&best_time, observer, meq.ra, meq.dec, REFRACTION_NONE);
        astro_horizon_t shor = Astronomy_Horizon(&best_time, observer, seq.ra, seq.dec, REFRACTION_NONE);
        ARCV = mhor.altitude - shor.altitude;
    }
    else
    { // Odeh
        double COSARCV = cos(ARCL * DEG2RAD) / cos(DAZ * DEG2RAD);
        if (COSARCV < -1)
            COSARCV = -1;
        else if (COSARCV > +1)
            COSARCV = +1;
        ARCV = acos(COSARCV) * RAD2DEG;
    }
    double W_topo = SD_topo * (1 - cos(ARCL * DEG2RAD)); // In arcminutes

    char result = ' ';
    double value;
    if (yallop)
    {
        value = (ARCV - (11.8371 - 6.3226 * W_topo + .7319 * pow(W_topo, 2) - .1018 * pow(W_topo, 3))) / 10;
        if (value > +.216)
            result = 'A'; // Crescent easily visible
        else if (value > -.014)
            result = 'B'; // Crescent visible under perfect conditions
        else if (value > -.160)
            result = 'C'; // May need optical aid to find crescent
        else if (value > -.232)
            result = 'D'; // Will need optical aid to find crescent
        else if (value > -.293)
            result = 'E'; // Crescent not visible with telescope
        else
            result = 'F';
    }
    else
    { // Odeh
        value = ARCV - (7.1651 - 6.3226 * W_topo + .7319 * pow(W_topo, 2) - .1018 * pow(W_topo, 3));
        if (value >= 5.65)
            result = 'A'; // Crescent is visible by naked eye
        else if (value >= 2.00)
            result = 'C'; // Crescent is visible by optical aid
        else if (value >= -.96)
            result = 'E'; // Crescent is visible only by optical aid
        else
            result = 'F';
    }
    if (q_value)
        *q_value = value;

    if (details)
    {
        details->best_time = best_time;
        details->sd = SD;
        details->lunar_parallax = lunar_parallax;
        details->arcl = ARCL;
        details->arcv = ARCV;
        details->daz = DAZ;
        details->w_topo = W_topo;
        details->sd_topo = SD_topo;
        details->value = value;
        details->moon_azimuth = moon_horizon.azimuth, details->moon_altitude = moon_horizon.altitude;
        details->moon_ra = moon_horizon.ra;
        details->moon_dec = moon_horizon.dec;
        details->sun_azimuth = sun_horizon.azimuth;
        details->sun_altitude = sun_horizon.altitude;
        details->sun_ra = sun_horizon.ra;
        details->sun_dec = sun_horizon.dec;
    }

    return result;
}

template <bool evening, bool yallop>
static void render(uint32_t *image, astro_time_t base_time)
{
    // First-visibility diamonds are drawn later by the Go compositor (blend.go)
    // for both the CPU and GPU renderers, so this renderer only paints the A–E
    // visibility zones and moon-age contour lines.
#if defined(_OPENMP)
#pragma omp parallel for
#endif
    for (unsigned i = 0; i < width; ++i)
    {
        // double max_q_value = -INFINITY; unsigned max_q_value_x = 0, max_q_value_y = 0;
        for (unsigned j = 0; j < height; ++j)
        {
            double latitude = ((height - (j + 1)) / (double)pixelsPerDegreeLat) + minLatitude;
            double longitude = (i / (double)pixelsPerDegreeLon) + minLongitude;

            if (latitude > 60 || latitude < -60) {
                image[i + j * width] = 0x00000000;
                continue;
            }

            bool draw_moon_line = false;
            double result_time = 0;
            double q_value = -INFINITY;
            char q_code = calculate<evening, yallop>(latitude, longitude, 0, base_time, nullptr, false, &draw_moon_line, &result_time, &q_value);
            uint32_t color = 0x00000000;
            if (q_code == 'A')
                color = 0xFFCCCC00; // Cyan - Crescent easily visible
            else if (q_code == 'B')
                color = 0xFFB3B300; // Darker cyan - Visible under perfect conditions
            else if (q_code == 'C')
                color = 0xFF1AFFFF; // Light blue/cyan - May need optical aid
            else if (q_code == 'D')
                color = 0xFF00E6E6; // Bright cyan - Will need optical aid
            else if (q_code == 'E')
                color = 0xFF00B3B3; // Darker cyan - Not visible with telescope
            else if (q_code == 'F')
                color = 0x00000000; // Black/transparent - Not visible
            else if (q_code == 'G')
                color = 0x00000000; // Transparent - Pre-conjunction
            else if (q_code == 'H')
                color = 0x00000000;
            else if (q_code == 'I')
                color = 0x00000000;
            else if (q_code == 'J')
                color = 0x00000000; // Transparent - Pre-conjunction + moonset before sunset
            if (draw_moon_line)
                color = 0xFFFFFFFF;
            image[i + j * width] = color;

            //             if (q_value > max_q_value)
            // #if defined(_OPENMP)
            //                 #pragma omp critical
            // #endif
            //             { max_q_value_x = i; max_q_value_y = j; max_q_value = q_value; }
        }

        // if (max_q_value_x != 0 && max_q_value_y != 0)
        //     image[max_q_value_x + max_q_value_y * width] = 0xFF0000FF;
    }
}

int main(int argc, const char **argv)
{
    if (argc >= 2 && (!strcmp(argv[1], "-version") || !strcmp(argv[1], "--version")))
    {
#ifdef VERSION_STR
// Stringize the version macro so it can be passed UNQUOTED on the command line
// (-DVERSION_STR=0.5.2). Passing it quoted (-DVERSION_STR="0.5.2") is fragile:
// shells (PowerShell, bash) strip the quotes, leaving g++ to choke on the bare
// pp-number "0.5.2" ("too many decimal points in number"). Two-step indirection
// expands VERSION_STR to its value before stringizing.
#define CMV_STRINGIZE2(x) #x
#define CMV_STRINGIZE(x) CMV_STRINGIZE2(x)
        printf("visibility (CPU renderer) version %s\n", CMV_STRINGIZE(VERSION_STR));
#elif defined(VERSION)
        printf("visibility (CPU renderer) version %s\n", VERSION);
#else
        printf("visibility (CPU renderer) version unknown\n");
#endif
        return 0;
    }

    if (argc == 1)
    {
        printf("Run like this:\n"
               "./visibility 2022-08-27 map evening yallop out.png\n"
               "./visibility 2022-08-27 table 34.23,23.3,0 100 > results.tsv\n"
               "./visibility 2022-08-27 point 31.95 35.23 yallop\n");
        return 1;
    }

    // New "point" mode for external validation harness (ICOP, etc.)
    int year = atoi(strtok((char *)argv[1], "-"));
    int month = atoi(strtok(nullptr, "-"));
    int day = atoi(strtok(nullptr, "-"));
    astro_time_t time = Astronomy_MakeTime(year, month, day, 0, 0, 0);

    if (argc >= 6 && !strcmp(argv[2], "point"))
    {
        double latitude = atof(argv[3]);
        double longitude = atof(argv[4]);
        bool yallop = (strcmp(argv[5], "yallop") == 0);

        details_t details = {};
        // We focus on evening crescents for most validation use cases
        char result = calculate<true, true>(latitude, longitude, 0.0, time, &details, false);

        if (!yallop) {
            result = calculate<true, false>(latitude, longitude, 0.0, time, &details, false);
        }

        // Emit moon age (hours since previous conjunction at best time) for validation harness alignment diagnostics.
        // This is the *exact* age used internally for the category/q decision (high fidelity for PR2 ICOP).
        double moon_age_h = details.moon_age_prev * 24.0;
        printf("date=%s lat=%.4f lon=%.4f criterion=%s category=%c q=%.4f arcv=%.2f w=%.2f age=%.2f\n",
               argv[1], latitude, longitude, argv[5], result, details.value, details.arcv, details.w_topo, moon_age_h);
        return 0;
    }

    if (!strcmp(argv[2], "map"))
    {
        bool evening;
        if (!strcmp(argv[3], "evening"))
            evening = true;
        else if (!strcmp(argv[3], "morning"))
            evening = false;
        else
            return 1;

        bool yallop;
        if (!strcmp(argv[4], "yallop"))
            yallop = true;
        else if (!strcmp(argv[4], "odeh"))
            yallop = false;
        else
            return 1;

        // Allocate image buffer and run render
        std::vector<uint32_t> image(width * height);
        if (yallop) {
            if (evening) render<true, true>(image.data(), time);
            else         render<false, true>(image.data(), time);
        } else {
            if (evening) render<true, false>(image.data(), time);
            else         render<false, false>(image.data(), time);
        }

        // Write raw RGBA binary for Python to load
        // First save raw RGBA pixel data as binary for Python to load
        // Format: W*H rows, each row = width*4 bytes (RGBA per pixel)
        // We use a .bin extension to distinguish from the final PNG
        char bin_file[1024];
        snprintf(bin_file, sizeof(bin_file), "%s.bin", argv[5]);

        FILE *f = fopen(bin_file, "wb");
        if (!f) {
            fprintf(stderr, "Cannot open %s\n", bin_file);
            return 1;
        }

        // Write 8-byte metadata header (little-endian): width (4 bytes) + height (4 bytes)
        // This allows the Go compositor to load the file without hardcoded dimension probing.
        uint32_t header[2] = { width, height };
        fwrite(header, sizeof(uint32_t), 2, f);

        // The image array is in RGBA format (bytes are already ABGR, converted to RGBA above)
        // Write each pixel as R, G, B, A (RGBA order for PNG)
        for (unsigned j = 0; j < height; ++j) {
            // Write one row of pixels as RGBA
            unsigned char row_bytes[4 * width];
            for (unsigned i = 0; i < width; ++i) {
                uint32_t px = image[i + j * width];
                // Extract RGBA components (image is stored as RGBA uint32: a<<24 | b<<16 | g<<8 | r)
                row_bytes[i * 4 + 0] = (px >> 0) & 0xFF;  // R
                row_bytes[i * 4 + 1] = (px >> 8) & 0xFF;  // G
                row_bytes[i * 4 + 2] = (px >> 16) & 0xFF; // B
                row_bytes[i * 4 + 3] = (px >> 24) & 0xFF; // A
            }
            fwrite(row_bytes, 4 * width, 1, f);
        }
        fclose(f);

        printf("Map written to %s (use gpu_blend.py to compose with base map)\n", argv[5]);
        return 0;
    }
    else if (!strcmp(argv[2], "table") || !strcmp(argv[2], "table-ignore-besttime"))
    {
        details_t details;
        bool ignore_besttime = !strcmp(argv[2], "table-ignore-besttime");
        double latitude = atof(strtok((char *)argv[3], ","));
        double longitude = atof(strtok(nullptr, ","));
        double altitude = atof(strtok(nullptr, ","));
        unsigned days = atoi(argv[4]);
        printf("UTC Date\tLatitude\tLongitude\tAltitude\t");

        printf("Sunset\tMoonset%s\tPrev New Moon\tNext New Moon\tMoon age from prev\tMoon age to next\tLag time\t",
               ignore_besttime ? "" : "\tBest time");
        printf("Evening (Yallop)\tq value\t");
        printf("Moon azimuth\tMoon altitude\tMoon ra\tMoon dec\t");
        printf("Sun azimuth\tSun altitude\tSun ra\tSun dec\t");
        printf("Moon sd\tlunar parallax\tarcl geo\tarcv yallop\tdaz\tw topo\tsd topo\t");

        printf("Evening (Odeh)\tV value\t");
        printf("Moon sd\tlunar parallax\tarcl topo\tarcv odeh\tdaz\tw topo\tsd topo\t");

        printf("Sunrise\tMoonrise%s\tPrev New Moon\tNext New Moon\tMoon age from prev\tMoon age to next\tlag time\t",
               ignore_besttime ? "" : "\tBest time");
        printf("Morning (Yallop)\tq value\t");
        printf("Moon azimuth\tMoon altitude\tMoon ra\tMoon dec\t");
        printf("Sun azimuth\tSun altitude\tSun ra\tSun dec\t");
        printf("Moon sd\tlunar parallax\tarcl geo\tarcv yallop\tdaz\tw topo\tsd topo\t");

        printf("Morning (Odeh)\tV value\t");
        printf("Moon sd\tlunar parallax\tarcl topo\tarcv odeh\tdaz\tw topo\tsd topo\t");

        printf("\n");
        for (unsigned i = 0; i < days; ++i)
        {
            char result;
            astro_utc_t utc = Astronomy_UtcFromTime(time);
            printf("%d-%d-%d\t%f\t%f\t%f\t", utc.year, utc.month, utc.day, latitude, longitude, altitude);
#define LOG(v) printf("%f\t", details.v)
#define TIME(t)                             \
    utc = Astronomy_UtcFromTime(details.t); \
    printf("%d-%02d-%02d %02d:%02d:%02.2f\t", utc.year, utc.month, utc.day, utc.hour, utc.minute, utc.second)
#define TIME_DIFF(t) printf("%s%d:%02d:%02d\t", details.t < 0 ? "-" : "", (int)floor(abs(details.t) * 24), (int)floor(abs(details.t) * 24 * 60 - floor(abs(details.t) * 24) * 60), (int)floor(abs(details.t) * 24 * 60 * 60 - floor(abs(details.t) * 24 * 60) * 60))
            memset(&details, 0, sizeof(details_t));
            result = calculate<true, true>(latitude, longitude, altitude, time, &details, ignore_besttime);
            TIME(sunset_sunrise);
            TIME(moonset_moonrise);
            if (!ignore_besttime)
            {
                TIME(best_time);
            }
            TIME(new_moon_prev);
            TIME(new_moon_next);
            TIME_DIFF(moon_age_prev);
            TIME_DIFF(moon_age_next);
            TIME_DIFF(lag_time);
            printf("%c\t", result);
            LOG(value);
            LOG(moon_azimuth);
            LOG(moon_altitude);
            LOG(moon_ra);
            LOG(moon_dec);
            LOG(sun_azimuth);
            LOG(sun_altitude);
            LOG(sun_ra);
            LOG(sun_dec);
            LOG(sd);
            LOG(lunar_parallax);
            LOG(arcl);
            LOG(arcv);
            LOG(daz);
            LOG(w_topo);
            LOG(sd_topo);

            memset(&details, 0, sizeof(details_t));
            printf("%c\t", calculate<true, false>(latitude, longitude, altitude, time, &details, ignore_besttime));
            LOG(value);
            LOG(sd);
            LOG(lunar_parallax);
            LOG(arcl);
            LOG(arcv);
            LOG(daz);
            LOG(w_topo);
            LOG(sd_topo);

            memset(&details, 0, sizeof(details_t));
            result = calculate<false, true>(latitude, longitude, altitude, time, &details, ignore_besttime);
            TIME(sunset_sunrise);
            TIME(moonset_moonrise);
            if (!ignore_besttime)
            {
                TIME(best_time);
            }
            TIME(new_moon_prev);
            TIME(new_moon_next);
            TIME_DIFF(moon_age_prev);
            TIME_DIFF(moon_age_next);
            TIME_DIFF(lag_time);
            printf("%c\t", result);
            LOG(value);
            LOG(moon_azimuth);
            LOG(moon_altitude);
            LOG(moon_ra);
            LOG(moon_dec);
            LOG(sun_azimuth);
            LOG(sun_altitude);
            LOG(sun_ra);
            LOG(sun_dec);
            LOG(sd);
            LOG(lunar_parallax);
            LOG(arcl);
            LOG(arcv);
            LOG(daz);
            LOG(w_topo);
            LOG(sd_topo);

            memset(&details, 0, sizeof(details_t));
            printf("%c\t", calculate<false, false>(latitude, longitude, altitude, time, &details, ignore_besttime));
            LOG(value);
            LOG(sd);
            LOG(lunar_parallax);
            LOG(arcl);
            LOG(arcv);
            LOG(daz);
            LOG(w_topo);
            LOG(sd_topo);
#undef TIME
#undef LOG
            printf("\n");
            time = Astronomy_AddDays(time, 1);
        }
        return 0;
    }
    else
    {
        printf("Invalid command\n");
        return 1;
    }
}
