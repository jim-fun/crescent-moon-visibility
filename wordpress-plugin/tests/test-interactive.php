<?php
/**
 * Functional test harness for the production interactive shortcode.
 *
 * Mocks just enough of WordPress (plus an in-memory $wpdb seeded from the real
 * 2026-2028 dataset) to exercise the *actual* production code paths:
 *   - includes/interactive.php  (heuristic, smart defaults, cities, AJAX)
 *   - public/interactive-renderer.php (shortcode markup + <noscript> fallback)
 *
 * Run: php wordpress-plugin/tests/test-interactive.php
 */

error_reporting(E_ALL);

$FAILS = 0;
function check($label, $cond) {
    global $FAILS;
    echo ($cond ? "  PASS  " : "  FAIL  ") . $label . "\n";
    if (!$cond) { $FAILS++; }
}

// -----------------------------------------------------------------------------
// WordPress mocks
// -----------------------------------------------------------------------------
define('ABSPATH', __DIR__ . '/');

$GLOBALS['__actions'] = [];
$GLOBALS['__shortcodes'] = [];
$GLOBALS['__localized'] = null;
$GLOBALS['__options'] = [];

function add_action($h, $cb, $p = 10, $a = 1) { $GLOBALS['__actions'][] = $h; }
function add_shortcode($t, $cb) { $GLOBALS['__shortcodes'][$t] = $cb; }
function register_activation_hook($f, $cb) {}
function shortcode_atts($pairs, $atts, $shortcode = '') { return array_merge($pairs, (array) ($atts ?: [])); }
function sanitize_text_field($s) { return trim(strip_tags((string) $s)); }
function sanitize_key($s) { return preg_replace('/[^a-z0-9_\-]/', '', strtolower((string) $s)); }
function absint($v) { return abs((int) $v); }
function esc_html($s) { return htmlspecialchars((string) $s, ENT_QUOTES, 'UTF-8'); }
function esc_attr($s) { return htmlspecialchars((string) $s, ENT_QUOTES, 'UTF-8'); }
function __($s, $d = 'default') { return $s; }
function esc_html__($s, $d = 'default') { return esc_html($s); }
function esc_attr__($s, $d = 'default') { return esc_attr($s); }
function esc_html_e($s, $d = 'default') { echo esc_html($s); }
function esc_attr_e($s, $d = 'default') { echo esc_attr($s); }
function _e($s, $d = 'default') { echo $s; }
function _n($s, $p, $n, $d = 'default') { return $n == 1 ? $s : $p; }
function wp_kses($s, $allowed = []) { return $s; }
function current_user_can($c) { return true; }
function current_time($fmt) { return date($fmt, strtotime('2026-05-29')); } // matches project "today"
function admin_url($p = '') { return 'http://example.com/wp-admin/' . $p; }
function plugin_dir_path($f) { return dirname($f) . '/'; }
function plugin_dir_url($f) { return 'http://example.com/wp-content/plugins/crescent-visibility/'; }
function wp_enqueue_style($h, $src = '', $d = [], $v = false) {}
function wp_enqueue_script($h, $src = '', $d = [], $v = false, $f = false) {}
function wp_create_nonce($a) { return 'test-nonce'; }
function check_ajax_referer($a, $f = false, $die = true) { return true; }
function get_option($k, $d = false) { return $GLOBALS['__options'][$k] ?? $d; }
function update_option($k, $v) { $GLOBALS['__options'][$k] = $v; return true; }
function wp_localize_script($h, $name, $data) { $GLOBALS['__localized'] = $data; }
function wp_json_encode($data, $flags = 0, $depth = 512) { return json_encode($data, $flags, $depth); }
function wp_list_pluck($list, $field) { return array_map(function ($i) use ($field) { return is_array($i) ? ($i[$field] ?? null) : $i->$field; }, $list); }
function wp_unslash($v) { return is_string($v) ? stripslashes($v) : $v; }

class CviJsonExit extends Exception {
    public $payload;
    public function __construct($payload) { $this->payload = $payload; parent::__construct('json'); }
}
function wp_send_json_success($data) { throw new CviJsonExit(['success' => true, 'data' => $data]); }
function wp_send_json_error($data, $code = 200) { throw new CviJsonExit(['success' => false, 'data' => $data, 'code' => $code]); }

// -----------------------------------------------------------------------------
// In-memory $wpdb seeded from the real dataset
// -----------------------------------------------------------------------------
class MockWPDB {
    public $prefix = 'wp_';
    public $observations = [];
    public $cities = [];

    public function get_charset_collate() { return ''; }

    public function prepare($query, ...$args) {
        if (count($args) === 1 && is_array($args[0])) { $args = $args[0]; }
        foreach ($args as $a) {
            if (is_int($a)) {
                $query = preg_replace('/%d/', (string) $a, $query, 1);
            } elseif (is_float($a)) {
                $query = preg_replace('/%f/', (string) $a, $query, 1);
            } else {
                $query = preg_replace('/%s/', "'" . addslashes((string) $a) . "'", $query, 1);
            }
        }
        return $query;
    }

    private function city_of($q) {
        return preg_match("/city = '([^']+)'/", $q, $m) ? $m[1] : null;
    }

    public function get_col($q) {
        if (strpos($q, 'SELECT DISTINCT city') !== false) {
            $set = [];
            foreach ($this->observations as $o) { $set[$o['city']] = true; }
            $slugs = array_keys($set);
            sort($slugs);
            return $slugs;
        }
        if (strpos($q, 'SELECT DISTINCT year') !== false) {
            $city = $this->city_of($q);
            $years = [];
            foreach ($this->observations as $o) {
                if ($o['city'] === $city) { $years[$o['year']] = true; }
            }
            $years = array_map('intval', array_keys($years));
            sort($years);
            return $years;
        }
        return [];
    }

    public function get_row($q, $output = OBJECT) {
        $city = $this->city_of($q);
        $rows = array_values(array_filter($this->observations, fn($o) => $o['city'] === $city));
        usort($rows, fn($a, $b) => strcmp($a['new_moon_date'], $b['new_moon_date']));

        if (strpos($q, 'new_moon_date >=') !== false) {
            preg_match("/new_moon_date >= '([^']+)'/", $q, $m);
            $today = $m[1] ?? '2026-05-29';
            foreach ($rows as $r) {
                if ($r['new_moon_date'] >= $today) {
                    return ['new_moon_date' => $r['new_moon_date'], 'year' => $r['year']];
                }
            }
            return null;
        }
        if (strpos($q, 'ORDER BY new_moon_date DESC') !== false) {
            $r = end($rows);
            return $r ? ['new_moon_date' => $r['new_moon_date'], 'year' => $r['year']] : null;
        }
        return null;
    }

    public function get_results($q, $output = OBJECT) {
        // Cities-with-data join
        if (strpos($q, 'crescent_cities') !== false) {
            $out = [];
            foreach ($this->cities as $c) {
                $has = false;
                foreach ($this->observations as $o) {
                    if ($o['city'] === $c['slug']) { $has = true; break; }
                }
                if ($has) {
                    $out[] = ['slug' => $c['slug'], 'name' => $c['name'], 'latitude' => $c['latitude'], 'longitude' => $c['longitude']];
                }
            }
            usort($out, fn($a, $b) => strcmp($a['name'], $b['name']));
            return $out;
        }
        // Embedded dataset: WHERE city IN ('a','b',...)
        if (strpos($q, 'city IN (') !== false) {
            preg_match("/city IN \(([^)]*)\)/", $q, $m);
            $slugs = array_map(function ($s) { return trim($s, " '"); }, explode(',', $m[1] ?? ''));
            $out = array_values(array_filter($this->observations, fn($o) => in_array($o['city'], $slugs, true)));
            usort($out, fn($a, $b) => [$a['city'], $a['new_moon_date']] <=> [$b['city'], $b['new_moon_date']]);
            return $out;
        }
        // New moons with details (city + year)
        if (strpos($q, 'raw_day_0, raw_day_1, raw_day_2') !== false && preg_match('/year = (\d+)/', $q, $ym)) {
            $city = $this->city_of($q);
            $year = (int) $ym[1];
            $out = [];
            foreach ($this->observations as $o) {
                if ($o['city'] === $city && (int) $o['year'] === $year) { $out[] = $o; }
            }
            usort($out, fn($a, $b) => strcmp($a['new_moon_date'], $b['new_moon_date']));
            return $out;
        }
        return [];
    }

    public function get_var($q) {
        if (strpos($q, 'COUNT(*)') !== false) { return count($this->observations); }
        return 0;
    }

    public function replace($table, $data, $format = null) { return 1; }
}

if (!defined('OBJECT')) { define('OBJECT', 'OBJECT'); }
if (!defined('ARRAY_A')) { define('ARRAY_A', 'ARRAY_A'); }
if (!defined('CVI_VERSION')) { define('CVI_VERSION', 'test-version'); }
$GLOBALS['wpdb'] = new MockWPDB();

// -----------------------------------------------------------------------------
// Seed from the real dataset
// -----------------------------------------------------------------------------
// Seed from the production dataset (the only data file we keep — regenerate via
// the generator per wordpress-plugin/generator/). Assertions are data-driven so
// they hold regardless of how many cities/years it contains.
$json = json_decode(file_get_contents(__DIR__ . '/../data/visibility-2026-2075.json'), true);
foreach ($json['cities'] as $c) {
    $GLOBALS['wpdb']->cities[] = $c;
}
foreach ($json['observations'] as $o) {
    $dq = $o['day_q'] ?? [];
    $da = $o['day_age'] ?? [];
    $GLOBALS['wpdb']->observations[] = [
        'city' => $o['city'],
        'new_moon_date' => $o['new_moon'],
        'year' => (int) $o['year'],
        'raw_day_0' => $o['days'][0] ?? '?',
        'raw_day_1' => $o['days'][1] ?? '?',
        'raw_day_2' => $o['days'][2] ?? '?',
        'q_day_0' => $dq[0] ?? null, 'q_day_1' => $dq[1] ?? null, 'q_day_2' => $dq[2] ?? null,
        'age_day_0' => $da[0] ?? null, 'age_day_1' => $da[1] ?? null, 'age_day_2' => $da[2] ?? null,
        'best_raw' => $o['best_raw'] ?? '?',
        'best_effective' => $o['best_effective'] ?? '?',
        'q_at_best' => $o['q_at_best'] ?? null,
        'moon_age_at_best' => $o['moon_age_at_best'] ?? null,
        'data_version' => 'yallop',
    ];
}
// How many distinct cities the data actually contains (drives the count checks).
$EXPECTED_CITIES = count(array_unique(array_column($json['observations'], 'city')));
echo "Seeded " . count($GLOBALS['wpdb']->observations) . " observations, " . count($GLOBALS['wpdb']->cities) . " cities (distinct=$EXPECTED_CITIES).\n\n";

// -----------------------------------------------------------------------------
// Load production code
// -----------------------------------------------------------------------------
require_once __DIR__ . '/../plugin/includes/interactive.php';
require_once __DIR__ . '/../plugin/public/interactive-renderer.php';
$cvi = $GLOBALS['cvi_interactive'];

// =============================================================================
// 1. Heuristic parity with main.go (known reference outputs)
// =============================================================================
echo "[1] Atmospheric heuristic (exact main.go port)\n";
// A, clear (cloud 0, trans 7): adj 0 -> A, minimal-impact note
list($eff, $note) = cvi_apply_atmospheric_adjustment('A', 0, 7);
check("A @ clear stays A", $eff === 'A');
check("clear note = minimal impact", strpos($note, 'minimal impact') !== false);
// A, overcast (cloud 90 -> -3), trans 7: 5-3=2 -> D
list($eff) = cvi_apply_atmospheric_adjustment('A', 90, 7);
check("A @ 90% cloud -> D", $eff === 'D');
// C (3), cloud 50 (-1), trans 10 (+1) => 3 -> C
list($eff) = cvi_apply_atmospheric_adjustment('C', 50, 10);
check("C @ 50% cloud, trans 10 -> C", $eff === 'C');
// F maps to 0 -> floored to 1 -> E (no early return)
list($eff) = cvi_apply_atmospheric_adjustment('F', 0, 7);
check("F floored to E", $eff === 'E');
// Unknown 'J' also floored to 1 -> E (heuristic itself does NOT special-case)
list($eff) = cvi_apply_atmospheric_adjustment('J', 0, 7);
check("J floored to E in heuristic", $eff === 'E');
// trans <= 4 penalty: A (5) - 1 = 4 -> B
list($eff) = cvi_apply_atmospheric_adjustment('A', 0, 3);
check("A @ low transparency -> B", $eff === 'B');

// =============================================================================
// 2. Smart defaults (next future new moon from 2026-05-29)
// =============================================================================
echo "\n[2] Smart defaults\n";
$def = $cvi->get_smart_default('jerusalem');
check("reason = next_future", $def['reason'] === 'next_future');
check("default date is in the future (>= 2026-05-29)", $def['new_moon_date'] >= '2026-05-29');
check("default year set", $def['year'] >= 2026);
echo "    -> {$def['new_moon_date']} ({$def['reason']})\n";

// =============================================================================
// 3. Cities from DB
// =============================================================================
echo "\n[3] City list from data\n";
$cities = $cvi->get_cities();
check("all data cities returned (no blacklist)", count($cities) === $EXPECTED_CITIES);
$names = array_column($cities, 'name');
$sortedNames = $names;
usort($sortedNames, 'strcasecmp');
check("dropdown sorted by name (country then city)", $names === $sortedNames);
check("known cities use 'Country — City' labels", in_array('USA — Dallas', $names, true) && in_array('Israel — Jerusalem', $names, true));
check("required cities still present", !array_diff(['jerusalem', 'dallas', 'melbourne'], array_column($cities, 'slug')));
check("cities carry lat/lon floats", is_float($cities[0]['lat']) && is_float($cities[0]['lon']));

// =============================================================================
// 4. AJAX endpoints
// =============================================================================
echo "\n[4] AJAX endpoints\n";
$_GET = ['city' => 'jerusalem', 'nonce' => 'test-nonce'];
try { $cvi->ajax_get_years(); } catch (CviJsonExit $e) { $years = $e->payload; }
check("get_years success", $years['success'] === true);
$yrs = $years['data']['years'];
check("years cover 2026-2028", in_array(2026, $yrs) && in_array(2027, $yrs) && in_array(2028, $yrs));
check("years are ascending ints", $yrs === array_values($yrs) && $yrs[0] <= end($yrs));

$_GET = ['city' => 'jerusalem', 'year' => '2026', 'nonce' => 'test-nonce'];
try { $cvi->ajax_get_newmoons(); } catch (CviJsonExit $e) { $nm = $e->payload; }
check("get_newmoons success", $nm['success'] === true);
$rows = $nm['data']['new_moons'];
check("new moons returned", count($rows) > 0);
$r0 = $rows[0];
check("row has 3 days", count($r0['days']) === 3);
check("row carries per-day day_q (3 values)", isset($r0['day_q']) && count($r0['day_q']) === 3);
check("row carries per-day day_age (3 values)", isset($r0['day_age']) && count($r0['day_age']) === 3);
check("per-day age values differ (not duplicated)", $r0['day_age'][0] !== $r0['day_age'][1] && $r0['day_age'][1] !== $r0['day_age'][2]);
check("per-day q values differ (not duplicated)", $r0['day_q'][0] !== $r0['day_q'][1] && $r0['day_q'][1] !== $r0['day_q'][2]);

// invalid year rejected
$_GET = ['city' => 'jerusalem', 'year' => '1500', 'nonce' => 'test-nonce'];
try { $cvi->ajax_get_newmoons(); } catch (CviJsonExit $e) { $bad = $e->payload; }
check("invalid year rejected (success=false, 400)", $bad['success'] === false && $bad['code'] === 400);

// =============================================================================
// 5. Shortcode rendering
// =============================================================================
echo "\n[5] Shortcode markup\n";
$cb = $GLOBALS['__shortcodes']['crescent_visibility_interactive'] ?? null;
check("shortcode registered", is_callable($cb));
check("alias [crescent_visibility_point] registered", isset($GLOBALS['__shortcodes']['crescent_visibility_point']));
$html = $cb(['default_city' => 'jerusalem']);
check("renders root container", strpos($html, 'cvi-interactive-root') !== false);
check("has city/year/newmoon selects", strpos($html, 'id="cvi-city"') && strpos($html, 'id="cvi-year"') && strpos($html, 'id="cvi-newmoon"'));
check("has sliders", strpos($html, 'id="cvi-cloud"') && strpos($html, 'id="cvi-trans"'));
check("has map container", strpos($html, 'id="cvi-map"') !== false);
check("shows installed plugin version", strpos($html, 'Plugin v' . CVI_VERSION) !== false);
check("embeds precomputed dataset script (no AJAX needed)", strpos($html, 'class="cvi-dataset"') !== false);
// Extract and validate the embedded JSON dataset (strpos, not regex — the
// embed can be multi-MB and would blow the PCRE backtrack limit).
$ds_open = '<script type="application/json" class="cvi-dataset">';
$ds_start = strpos($html, $ds_open);
$ds_end   = $ds_start !== false ? strpos($html, '</script>', $ds_start) : false;
if ($ds_start !== false && $ds_end !== false) {
    $ds = json_decode(substr($html, $ds_start + strlen($ds_open), $ds_end - $ds_start - strlen($ds_open)), true);
    check("dataset has jerusalem with 2026", isset($ds['jerusalem']['2026']) && count($ds['jerusalem']['2026']) > 0);
    $sample = $ds['jerusalem']['2026'][0] ?? null;
    check("dataset row is [date,[d0,d1,d2],[q0,q1,q2],[a0,a1,a2]]",
        is_array($sample) && count($sample) === 4
        && is_array($sample[1]) && count($sample[1]) === 3
        && is_array($sample[2]) && count($sample[2]) === 3
        && is_array($sample[3]) && count($sample[3]) === 3);
    check("dataset covers all data cities", count($ds) === $EXPECTED_CITIES);
} else {
    check("dataset script parseable", false);
}
check("has 3 quick-select buttons (jerusalem/dallas/melbourne)",
    strpos($html, 'data-slug="jerusalem"') !== false
    && strpos($html, 'data-slug="dallas"') !== false
    && strpos($html, 'data-slug="melbourne"') !== false);
check("has legend", strpos($html, 'How to read the visibility rating') !== false);
check("renders <noscript> fallback cards", strpos($html, '<noscript>') !== false && strpos($html, 'cvi-card') !== false);
check("cities embedded in data-cities attribute", strpos($html, 'data-cities=') !== false && strpos($html, 'jerusalem') !== false);
check("defaults embedded in data-* attributes", strpos($html, 'data-default-city=') !== false && strpos($html, 'data-default-newmoon=') !== false);

// localized payload
$loc = $GLOBALS['__localized'];
check("localized ajaxUrl", !empty($loc['ajaxUrl']));
check("localized nonce", $loc['nonce'] === 'test-nonce');
check("localized all data cities", count($loc['cities']) === $EXPECTED_CITIES);
check("localized default city = jerusalem", $loc['defaultCity'] === 'jerusalem');
check("localized default new moon matches smart default", $loc['defaultNewMoon'] === $def['new_moon_date']);

// dark theme attribute
$darkHtml = $cb(['default_city' => 'london', 'theme' => 'dark']);
check("theme=dark applied", strpos($darkHtml, 'data-theme="dark"') !== false);

// =============================================================================
// 6. Adverse scenarios from the paleotimes.org HAR
// =============================================================================
echo "\n[6] HAR regression: junk cities row + empty data\n";

// (a) A blank cities row must NOT leak into the dropdown.
$wpdb = $GLOBALS['wpdb'];
array_unshift($wpdb->cities, ['slug' => '', 'name' => '', 'latitude' => 0, 'longitude' => 0]);
$citiesAfter = $cvi->get_cities();
check("blank cities row filtered out", count(array_filter($citiesAfter, fn($c) => $c['slug'] === '')) === 0);
check("real cities still present", count($citiesAfter) === $EXPECTED_CITIES);
$htmlJunk = $cb(['default_city' => 'jerusalem']);
check("rendered data-cities has no empty slug", strpos($htmlJunk, '"slug":""') === false);
array_shift($wpdb->cities); // restore

// (b) No observation data at all -> smart default null, but render must not fatal
$savedObs = $wpdb->observations;
$wpdb->observations = [];
$defEmpty = $cvi->get_smart_default('jerusalem');
check("no-data smart default reason = no_data", $defEmpty['reason'] === 'no_data');
check("no-data new_moon_date is null", $defEmpty['new_moon_date'] === null);
$citiesEmpty = $cvi->get_cities();
check("required cities present even with no data", count($citiesEmpty) === 3 && !array_diff(['jerusalem', 'dallas', 'melbourne'], array_column($citiesEmpty, 'slug')));
$htmlEmpty = $cb(['default_city' => 'jerusalem']);
check("empty-data render still produces a city dropdown (required fallback)", strpos($htmlEmpty, 'data-cities=') !== false && strpos($htmlEmpty, '"slug":""') === false);
check("empty-data noscript shows the enable-JS message", strpos($htmlEmpty, 'Enable JavaScript') !== false);
$wpdb->observations = $savedObs; // restore

// =============================================================================
echo "\n" . str_repeat('=', 50) . "\n";
if ($FAILS === 0) {
    echo "ALL CHECKS PASSED\n";
    exit(0);
}
echo "$FAILS CHECK(S) FAILED\n";
exit(1);
