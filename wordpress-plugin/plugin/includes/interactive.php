<?php
/**
 * Production Interactive "Visibility for My Location" module.
 *
 * Synthesized from the qwen3 exploratory drafts (v3 + v4) and brought to
 * production quality:
 *   - Smart defaults (Jerusalem + the next/closest new moon, like the web app)
 *   - Nonce-validated admin-ajax endpoints for dynamic year / new-moon loading
 *   - City list sourced from the imported `crescent_cities` table (with a
 *     locked Phase-0 fallback) so the front end stays in sync with the data
 *   - The exact atmospheric heuristic ported from main.go (used for the no-JS
 *     server-side fallback; an identical copy lives in assets/js/interactive.js)
 *   - A lightweight, version-gated schema upgrade for installs that imported
 *     data before the q_at_best / moon_age_at_best columns existed
 *
 * Accuracy First: every visibility category traces back to the reference
 * Yallop renderer via the offline JSON generator. Nothing is computed at
 * runtime here — the atmospheric heuristic only re-grades a pre-computed
 * category, exactly as the Go web app does.
 *
 * @package Crescent_Visibility
 */

if (!defined('ABSPATH')) {
    exit;
}

/**
 * Schema version for the interactive feature. Bump when the observations
 * table layout changes so existing installs upgrade automatically.
 */
if (!defined('CVI_SCHEMA_VERSION')) {
    define('CVI_SCHEMA_VERSION', '2');
}

/**
 * Cities hidden from the tool regardless of what the imported data contains.
 * Currently empty — the dropdown reflects EVERY city present in the data, with
 * no blacklist. (Kept as a single point of control should filtering ever be
 * needed again.)
 *
 * @return string[] lowercase slugs
 */
function cvi_excluded_cities() {
    return [];
}

/**
 * Cities the tool always offers, in display order. These are guaranteed to
 * appear in the dropdown and as quick-select buttons even if the imported
 * data does not include them. Data preparers can add MORE cities simply by
 * including them in the import JSON — those appear after these.
 *
 * @return array<int,array{slug:string,name:string,latitude:float,longitude:float}>
 */
function cvi_required_cities() {
    return [
        ['slug' => 'jerusalem', 'name' => 'Israel — Jerusalem',    'latitude' => 31.7683,  'longitude' => 35.2137],
        ['slug' => 'dallas',    'name' => 'USA — Dallas',          'latitude' => 32.7767,  'longitude' => -96.7970],
        ['slug' => 'melbourne', 'name' => 'Australia — Melbourne', 'latitude' => -37.8136, 'longitude' => 144.9631],
    ];
}

/**
 * Fallback city list when there is no data yet — just the required cities.
 *
 * @return array<int,array{slug:string,name:string,latitude:float,longitude:float}>
 */
function cvi_locked_cities() {
    return cvi_required_cities();
}

/**
 * Built-in display name ("Country — City") + coordinates for the curated
 * cities, keyed by slug. This is the authoritative source for how a city is
 * labelled and sorted in the dropdown (country first, then city). Cities the
 * data preparer adds that are not listed here fall back to the crescent_cities
 * table name, then a slug-derived name.
 *
 * @return array<string,array{0:string,1:float,2:float}> slug => [name, lat, lon]
 */
function cvi_known_cities() {
    return [
        // Middle East
        'jerusalem'    => ['Israel — Jerusalem',               31.7683,  35.2137],
        // Australia / Pacific
        'melbourne'    => ['Australia — Melbourne',            -37.8136, 144.9631],
        'adelaide'     => ['Australia — Adelaide',             -34.9285, 138.6007],
        'perth'        => ['Australia — Perth',                -31.9523, 115.8613],
        // Canada
        'princegeorge' => ['Canada — Prince George',           53.9171,  -122.7497],
        'regina'       => ['Canada — Regina',                  50.4452,  -104.6189],
        // Colombia
        'pasto'        => ['Colombia — Pasto',                 1.2136,   -77.2811],
        // Jamaica
        'montegobay'   => ['Jamaica — Montego Bay',            18.4762,  -77.8939],
        // Mexico
        'mexicocity'   => ['Mexico — Mexico City',             19.4326,  -99.1332],
        'merida'       => ['Mexico — Mérida',                  20.9674,  -89.5926],
        // Namibia
        'gobabis'      => ['Namibia — Gobabis',                -22.4500, 18.9667],
        // Panama
        'panamacity'   => ['Panama — Panama City',             8.9824,   -79.5199],
        // Puerto Rico
        'sanjuan'      => ['Puerto Rico — San Juan',           18.4655,  -66.1057],
        // South Africa
        'johannesburg' => ['South Africa — Johannesburg',      -26.2041, 28.0473],
        // Tanzania
        'dodoma'       => ['Tanzania — Dodoma',                -6.1630,  35.7516],
        // Trinidad & Tobago
        'portofspain'  => ['Trinidad & Tobago — Port of Spain', 10.6549, -61.5019],
        // Turkey
        'fethiye'      => ['Turkey — Fethiye',                 36.6213,  29.1162],
        // United Kingdom
        'london'       => ['United Kingdom — London',          51.5074,  -0.1278],
        // United States
        'atlanta'      => ['USA — Atlanta',                    33.7490,  -84.3880],
        'boston'       => ['USA — Boston',                     42.3601,  -71.0589],
        'chicago'      => ['USA — Chicago',                    41.8781,  -87.6298],
        'dallas'       => ['USA — Dallas',                     32.7767,  -96.7970],
        'denver'       => ['USA — Denver',                     39.7392,  -104.9903],
        'honolulu'     => ['USA — Honolulu',                   21.3069,  -157.8583],
        'kansascity'   => ['USA — Kansas City',                39.0997,  -94.5786],
        'losangeles'   => ['USA — Los Angeles',                34.0522,  -118.2437],
        'orlando'      => ['USA — Orlando',                    28.5383,  -81.3792],
        'phoenix'      => ['USA — Phoenix',                    33.4484,  -112.0740],
        'seattle'      => ['USA — Seattle',                    47.6062,  -122.3321],
        'washington'   => ['USA — Washington, DC',             38.9072,  -77.0369],
    ];
}

/**
 * Exact PHP port of main.go:applyAtmosphericAdjustment.
 *
 * IMPORTANT (Accuracy First): keep this byte-for-byte equivalent to the Go
 * source and to the JS copy in assets/js/interactive.js. It re-grades a
 * pre-computed raw category for the user's atmospheric inputs — it never
 * performs astronomy.
 *
 * Note: like the Go version, "F" and any unmapped value (including "J", "?",
 * "") resolve to category value 0, which is then floored to 1 ("E"). The
 * graceful "not a good window" handling for "J" / "?" / large ages lives in
 * the card renderer, mirroring main.go's handlePointQuery.
 *
 * @param string $raw_category Raw category letter from the renderer.
 * @param int    $cloud_percent Cloud cover 0-100.
 * @param float  $transparency  Transparency 1-10.
 * @return array{0:string,1:string} [effective_category, plain-English note]
 */
function cvi_apply_atmospheric_adjustment($raw_category, $cloud_percent, $transparency) {
    $category_value = ['A' => 5, 'B' => 4, 'C' => 3, 'D' => 2, 'E' => 1, 'F' => 0][$raw_category] ?? 0;
    if ($category_value === 0) {
        $category_value = 1;
    }

    $adjustment = 0;

    // Cloud cover penalty (very impactful for crescent visibility).
    if ($cloud_percent > 80) {
        $adjustment -= 3;
    } elseif ($cloud_percent > 60) {
        $adjustment -= 2;
    } elseif ($cloud_percent > 40) {
        $adjustment -= 1;
    }

    // Transparency adjustment.
    if ($transparency >= 9) {
        $adjustment += 1;
    } elseif ($transparency <= 4) {
        $adjustment -= 1;
    } elseif ($transparency <= 2) {
        $adjustment -= 2;
    }

    $final_value = max(1, min(5, $category_value + $adjustment));
    $effective   = [5 => 'A', 4 => 'B', 3 => 'C', 2 => 'D', 1 => 'E'][$final_value];

    if ($adjustment === 0) {
        $note = __('Atmospheric conditions have minimal impact on this prediction.', 'crescent-visibility');
    } elseif ($adjustment < 0) {
        /* translators: %d: number of visibility category levels lost */
        $note = sprintf(__('Conditions are reducing visibility by approximately %d category level(s).', 'crescent-visibility'), -$adjustment);
    } else {
        $note = __('Excellent atmospheric conditions are slightly improving the prediction.', 'crescent-visibility');
    }

    return [$effective, $note];
}

/**
 * Encapsulates the interactive experience: AJAX endpoints, smart defaults,
 * city/year/new-moon data access, and the schema upgrade check.
 */
class Crescent_Visibility_Interactive {

    const NONCE_ACTION = 'cvi_interactive_nonce';

    /** @var string */
    private $table_observations;

    /** @var string */
    private $table_cities;

    public function __construct() {
        global $wpdb;
        $this->table_observations = $wpdb->prefix . 'crescent_observations';
        $this->table_cities       = $wpdb->prefix . 'crescent_cities';

        // Read-only data endpoints (available to logged-in and anonymous users).
        add_action('wp_ajax_cvi_get_years',        [$this, 'ajax_get_years']);
        add_action('wp_ajax_nopriv_cvi_get_years', [$this, 'ajax_get_years']);

        add_action('wp_ajax_cvi_get_newmoons',        [$this, 'ajax_get_newmoons']);
        add_action('wp_ajax_nopriv_cvi_get_newmoons', [$this, 'ajax_get_newmoons']);

        // Upgrade the schema if the stored version is behind (idempotent, gated).
        add_action('admin_init', [$this, 'maybe_upgrade_schema']);
    }

    // =================================================================
    // Schema upgrade (for installs that predate the q/age columns)
    // =================================================================

    /**
     * Run dbDelta to add the q_at_best / moon_age_at_best / data_version
     * columns when an older install is detected. Cheap and idempotent: it
     * only does work when the stored schema version is behind.
     */
    public function maybe_upgrade_schema() {
        if (get_option('cvi_schema_version') === CVI_SCHEMA_VERSION) {
            return;
        }

        global $wpdb;
        require_once ABSPATH . 'wp-admin/includes/upgrade.php';

        $charset_collate = $wpdb->get_charset_collate();

        // dbDelta can add missing columns/indexes to an existing table.
        $sql = "CREATE TABLE {$this->table_observations} (
            id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
            city VARCHAR(50) NOT NULL,
            new_moon_date DATE NOT NULL,
            year SMALLINT NOT NULL,
            raw_day_0 CHAR(1) NOT NULL,
            raw_day_1 CHAR(1) NOT NULL,
            raw_day_2 CHAR(1) NOT NULL,
            best_raw CHAR(1) NOT NULL,
            best_effective CHAR(1) NOT NULL,
            q_at_best DECIMAL(6,4) NULL,
            moon_age_at_best DECIMAL(5,2) NULL,
            q_day_0 DECIMAL(7,4) NULL,
            q_day_1 DECIMAL(7,4) NULL,
            q_day_2 DECIMAL(7,4) NULL,
            age_day_0 DECIMAL(6,2) NULL,
            age_day_1 DECIMAL(6,2) NULL,
            age_day_2 DECIMAL(6,2) NULL,
            data_version VARCHAR(60) NULL,
            PRIMARY KEY (id),
            UNIQUE KEY city_newmoon (city, new_moon_date),
            KEY city_year (city, year),
            KEY city_year_date (city, year, new_moon_date)
        ) $charset_collate;";

        dbDelta($sql);

        update_option('cvi_schema_version', CVI_SCHEMA_VERSION);
    }

    // =================================================================
    // Smart defaults — makes the form feel like the web app on first load
    // =================================================================

    /**
     * Best new moon to show for a city: prefer the next future new moon we
     * have data for, then the most recent, then a hard fallback.
     *
     * @param string $city_slug
     * @return array{new_moon_date:?string,year:int,reason:string}
     */
    public function get_smart_default($city_slug = 'jerusalem') {
        $city_slug = $this->sanitize_city($city_slug);
        $today     = current_time('Y-m-d');

        $next = $this->get_next_future_new_moon($city_slug, $today);
        if ($next) {
            return [
                'new_moon_date' => $next['new_moon_date'],
                'year'          => (int) $next['year'],
                'reason'        => 'next_future',
            ];
        }

        $latest = $this->get_most_recent_new_moon($city_slug);
        if ($latest) {
            return [
                'new_moon_date' => $latest['new_moon_date'],
                'year'          => (int) $latest['year'],
                'reason'        => 'most_recent',
            ];
        }

        return [
            'new_moon_date' => null,
            'year'          => (int) current_time('Y'),
            'reason'        => 'no_data',
        ];
    }

    private function get_next_future_new_moon($city, $today) {
        global $wpdb;
        return $wpdb->get_row($wpdb->prepare(
            "SELECT new_moon_date, year
               FROM {$this->table_observations}
              WHERE city = %s AND new_moon_date >= %s
              ORDER BY new_moon_date ASC
              LIMIT 1",
            $city,
            $today
        ), ARRAY_A);
    }

    private function get_most_recent_new_moon($city) {
        global $wpdb;
        return $wpdb->get_row($wpdb->prepare(
            "SELECT new_moon_date, year
               FROM {$this->table_observations}
              WHERE city = %s
              ORDER BY new_moon_date DESC
              LIMIT 1",
            $city
        ), ARRAY_A);
    }

    // =================================================================
    // City list (from the imported data, with a locked fallback)
    // =================================================================

    /**
     * The city list for the dropdown, sorted alphabetically by display name
     * ("Country — City", so country first then city). Driven by the cities that
     * actually have observation data, with the required cities guaranteed
     * present even if the import omitted them. Excluded cities are dropped.
     *
     * @return array<int,array{slug:string,name:string,lat:float,lon:float}>
     */
    public function get_cities() {
        global $wpdb;

        // Source of truth: the slugs that actually have observation data. The
        // crescent_cities table is only used to look up names/coordinates and
        // may be incomplete, so we never depend on it for the list itself.
        $slugs_with_data = $wpdb->get_col(
            "SELECT DISTINCT city FROM {$this->table_observations} ORDER BY city ASC"
        );

        // Name/coordinate metadata from the cities table, where available.
        $meta = [];
        $rows = $wpdb->get_results(
            "SELECT slug, name, latitude, longitude FROM {$this->table_cities}",
            ARRAY_A
        );
        foreach ((array) $rows as $row) {
            $slug = isset($row['slug']) ? trim((string) $row['slug']) : '';
            if ($slug !== '') {
                $meta[$slug] = $row;
            }
        }

        $known    = cvi_known_cities();
        $excluded = cvi_excluded_cities();

        // Guarantee the required cities even if the import omitted them.
        $slugs = (array) $slugs_with_data;
        foreach (cvi_required_cities() as $req) {
            if (!in_array($req['slug'], $slugs, true)) {
                $slugs[] = $req['slug'];
            }
        }

        $cities = [];
        foreach ($slugs as $raw_slug) {
            $slug = trim((string) $raw_slug);
            if ($slug === '' || in_array($slug, $excluded, true)) {
                continue;
            }
            // Prefer the curated "Country — City" catalog (authoritative for
            // labelling/sorting), then the cities table, then a slug fallback.
            if (isset($known[$slug])) {
                $cities[$slug] = ['slug' => $slug, 'name' => $known[$slug][0], 'lat' => $known[$slug][1], 'lon' => $known[$slug][2]];
            } elseif (isset($meta[$slug]) && $meta[$slug]['name'] !== null && $meta[$slug]['name'] !== '') {
                $cities[$slug] = ['slug' => $slug, 'name' => $meta[$slug]['name'], 'lat' => (float) $meta[$slug]['latitude'], 'lon' => (float) $meta[$slug]['longitude']];
            } else {
                $cities[$slug] = ['slug' => $slug, 'name' => ucwords(str_replace('-', ' ', $slug)), 'lat' => 0.0, 'lon' => 0.0];
            }
        }

        $cities = array_values($cities);
        usort($cities, function ($a, $b) {
            return strcasecmp($a['name'], $b['name']);
        });

        return $cities;
    }

    /**
     * Compact, page-embeddable dataset for the given city slugs.
     *
     * Returned as slug => year => list of [date, [d0,d1,d2], q, age]. This is
     * inlined into the page so the front end needs no admin-ajax/REST call at
     * all — which matters on sites where /wp-admin is behind Cloudflare Access
     * or similar and the public cannot reach admin-ajax.php.
     *
     * @param string[] $slugs
     * @return array<string,array<string,array<int,array{0:string,1:array,2:?float,3:?float}>>>
     */
    public function get_embedded_dataset(array $slugs) {
        global $wpdb;

        $excluded = cvi_excluded_cities();
        $slugs = array_values(array_unique(array_filter(
            array_map([$this, 'sanitize_city'], $slugs),
            function ($s) use ($excluded) { return $s !== '' && !in_array($s, $excluded, true); }
        )));
        if (empty($slugs)) {
            return [];
        }

        $placeholders = implode(',', array_fill(0, count($slugs), '%s'));
        $rows = $wpdb->get_results($wpdb->prepare(
            "SELECT city, year, new_moon_date, raw_day_0, raw_day_1, raw_day_2,
                    q_day_0, q_day_1, q_day_2, age_day_0, age_day_1, age_day_2
               FROM {$this->table_observations}
              WHERE city IN ($placeholders)
              ORDER BY city ASC, new_moon_date ASC",
            $slugs
        ), ARRAY_A);

        $q = function ($v) { return $v !== null ? round((float) $v, 4) : null; };
        $a = function ($v) { return $v !== null ? round((float) $v, 2) : null; };

        // Compact row: [date, [d0,d1,d2], [q0,q1,q2], [age0,age1,age2]]
        $out = [];
        foreach ((array) $rows as $r) {
            $year = (string) (int) $r['year'];
            $out[$r['city']][$year][] = [
                $r['new_moon_date'],
                [$r['raw_day_0'], $r['raw_day_1'], $r['raw_day_2']],
                [$q($r['q_day_0'] ?? null), $q($r['q_day_1'] ?? null), $q($r['q_day_2'] ?? null)],
                [$a($r['age_day_0'] ?? null), $a($r['age_day_1'] ?? null), $a($r['age_day_2'] ?? null)],
            ];
        }
        return $out;
    }

    // =================================================================
    // AJAX endpoints (read-only; nonce-validated)
    // =================================================================

    public function ajax_get_years() {
        $this->verify_request();

        $city = $this->sanitize_city(wp_unslash($_GET['city'] ?? 'jerusalem'));
        if (in_array($city, cvi_excluded_cities(), true)) {
            wp_send_json_success(['years' => []]);
        }

        wp_send_json_success(['years' => $this->get_available_years($city)]);
    }

    public function ajax_get_newmoons() {
        $this->verify_request();

        $city = $this->sanitize_city(wp_unslash($_GET['city'] ?? 'jerusalem'));
        $year = isset($_GET['year']) ? absint($_GET['year']) : 0;

        if ($year < 2000 || $year > 2100) {
            wp_send_json_error(['message' => 'Invalid year'], 400);
        }

        if (in_array($city, cvi_excluded_cities(), true)) {
            wp_send_json_success(['new_moons' => []]);
        }

        wp_send_json_success(['new_moons' => $this->get_new_moons_with_details($city, $year)]);
    }

    /**
     * Validate the read-only AJAX request.
     *
     * SECURITY NOTE: these two endpoints are intentionally public (wp_ajax_nopriv)
     * and expose only pre-computed, public visibility data. They perform no writes
     * and require no capability. The nonce is therefore soft abuse-control, NOT an
     * authorization boundary — and is mostly a fallback: the front end normally
     * reads the dataset embedded in the page and never calls these at all. Failing
     * closed on a bad/expired nonce is safe (no data is sensitive).
     */
    private function verify_request() {
        if (!check_ajax_referer(self::NONCE_ACTION, 'nonce', false)) {
            wp_send_json_error(['message' => 'Invalid or expired security token'], 403);
        }
    }

    // =================================================================
    // Internal data access
    // =================================================================

    public function get_available_years($city) {
        global $wpdb;
        $years = $wpdb->get_col($wpdb->prepare(
            "SELECT DISTINCT year FROM {$this->table_observations}
              WHERE city = %s ORDER BY year ASC",
            $city
        ));
        return array_map('intval', $years);
    }

    /**
     * New moons for a city/year, normalized for the front end. Each row
     * carries the three raw day categories plus the q value and moon age at
     * the best evening (the values the web app cards display).
     *
     * @return array<int,array{new_moon_date:string,days:array<int,string>,q_at_best:?float,moon_age_at_best:?float}>
     */
    public function get_new_moons_with_details($city, $year) {
        global $wpdb;

        $rows = $wpdb->get_results($wpdb->prepare(
            "SELECT new_moon_date,
                    raw_day_0, raw_day_1, raw_day_2,
                    q_day_0, q_day_1, q_day_2,
                    age_day_0, age_day_1, age_day_2,
                    best_raw, best_effective,
                    q_at_best, moon_age_at_best
               FROM {$this->table_observations}
              WHERE city = %s AND year = %d
              ORDER BY new_moon_date ASC",
            $city,
            $year
        ), ARRAY_A);

        $f = function ($v) { return $v !== null ? (float) $v : null; };

        return array_map(function ($row) use ($f) {
            return [
                'new_moon_date'    => $row['new_moon_date'],
                'days'             => [$row['raw_day_0'], $row['raw_day_1'], $row['raw_day_2']],
                'day_q'            => [$f($row['q_day_0'] ?? null), $f($row['q_day_1'] ?? null), $f($row['q_day_2'] ?? null)],
                'day_age'          => [$f($row['age_day_0'] ?? null), $f($row['age_day_1'] ?? null), $f($row['age_day_2'] ?? null)],
                // Kept for backward compatibility / summaries.
                'q_at_best'        => $f($row['q_at_best']),
                'moon_age_at_best' => $f($row['moon_age_at_best']),
            ];
        }, $rows);
    }

    /**
     * Conservative city-slug sanitizer (lowercase letters, digits, hyphens).
     * Matches the slugs produced by the generator.
     */
    private function sanitize_city($value) {
        $value = sanitize_key((string) $value);
        return $value !== '' ? $value : 'jerusalem';
    }
}

// Instantiate and expose globally so the renderer can read smart defaults.
$GLOBALS['cvi_interactive'] = new Crescent_Visibility_Interactive();
