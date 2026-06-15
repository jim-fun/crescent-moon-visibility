<?php
/**
 * Plugin Name:       Young Crescent Moon Visibility
 * Plugin URI:        https://github.com/jim-fun/crescent-moon-visibility
 * Description:       Minimal-footprint plugin showing pre-computed crescent visibility data for major cities (2026 onward). Uses accurate Yallop data generated offline.
 * Version:           0.5.5
 * Requires at least: 5.8
 * Requires PHP:      7.4
 * Author:            Crescent Moon Visibility Project
 * License:           GPL-2.0-or-later
 * License URI:       https://www.gnu.org/licenses/gpl-2.0.html
 * Text Domain:       crescent-visibility
 * Domain Path:       /languages
 *
 * Production release synthesized from exploratory drafts (v3 + v4) via the
 * dedicated Claude production prompt. Clean module loading, schema versioning,
 * and full interactive "Visibility for My Location" experience.
 */

if (!defined('ABSPATH')) {
    exit;
}

// Plugin version — bump on every package update. Must match the "Version:"
// header above. Also used to cache-bust the enqueued CSS/JS assets.
if (!defined('CVI_VERSION')) {
    define('CVI_VERSION', '0.5.5');
}

// Production schema version for auto-upgrade logic
if (!defined('CVI_SCHEMA_VERSION')) {
    define('CVI_SCHEMA_VERSION', '2026.05.2');
}

class Crescent_Visibility_Plugin {

    private $table_observations;
    private $table_summaries;
    private $table_cities;

    public function __construct() {
        global $wpdb;
        $this->table_cities       = $wpdb->prefix . 'crescent_cities';
        $this->table_observations = $wpdb->prefix . 'crescent_observations';
        $this->table_summaries    = $wpdb->prefix . 'crescent_yearly_summary';

        add_action('init', [$this, 'load_textdomain']);
        add_action('admin_menu', [$this, 'add_admin_menu']);
        add_shortcode('crescent_visibility', [$this, 'render_shortcode']);

        add_action('admin_enqueue_scripts', [$this, 'enqueue_admin_assets']);

        // Self-healing upgrade: uploading a new plugin version over an old one
        // (the common workflow) does NOT run activate(), so a schema change
        // would be missed. Detect a version change and bring the tables up to
        // date once — idempotent dbDelta, non-destructive (data is preserved).
        add_action('admin_init', [$this, 'maybe_upgrade']);
    }

    /**
     * Run pending upgrades when the installed version differs from the code
     * version. Runs at most once per version bump (gated by an option).
     */
    public function maybe_upgrade() {
        if (!defined('CVI_VERSION') || get_option('cvi_plugin_version') === CVI_VERSION) {
            return;
        }
        $this->create_or_update_tables(); // dbDelta: adds missing columns only, never drops data
        update_option('cvi_plugin_version', CVI_VERSION);
    }

    public function load_textdomain() {
        load_plugin_textdomain('crescent-visibility', false, dirname(plugin_basename(__FILE__)) . '/languages');
    }

    public function add_admin_menu() {
        add_management_page(
            __('Crescent Visibility Data', 'crescent-visibility'),
            __('Crescent Visibility', 'crescent-visibility'),
            'manage_options',
            'crescent-visibility',
            [$this, 'render_admin_page']
        );
    }

    public function enqueue_admin_assets($hook) {
        if ($hook !== 'tools_page_crescent-visibility') {
            return;
        }
        wp_enqueue_style('crescent-admin', plugin_dir_url(__FILE__) . 'admin/admin.css', [], '0.1.0');
    }

    public function render_admin_page() {
        if (!current_user_can('manage_options')) {
            return;
        }

        if (isset($_POST['crescent_import']) && check_admin_referer('crescent_import_nonce')) {
            $this->handle_import();
        }

        if (isset($_POST['crescent_drop']) && check_admin_referer('crescent_drop_nonce')) {
            $this->handle_drop_tables();
        }

        include plugin_dir_path(__FILE__) . 'admin/admin-page.php';
    }

    /**
     * Drop and recreate the plugin tables (empty). Useful when the table
     * structure is corrupted (e.g. lost AUTO_INCREMENT) so a re-import lands
     * on a clean schema. Surfaces a DROP-privilege error if the DB user can't
     * drop tables — which is itself a useful diagnostic.
     */
    private function handle_drop_tables() {
        global $wpdb;

        $wpdb->query("DROP TABLE IF EXISTS {$this->table_observations}");
        $drop_err_obs = $wpdb->last_error;
        $wpdb->query("DROP TABLE IF EXISTS {$this->table_cities}");
        $drop_err_city = $wpdb->last_error;

        delete_option('cvi_last_import');
        delete_option('cvi_schema_version');

        // Recreate fresh, correctly-keyed empty tables.
        $this->create_or_update_tables();

        $err = $drop_err_obs ?: $drop_err_city;
        if ($err) {
            echo '<div class="notice notice-error"><p><strong>' . esc_html__('Could not drop tables.', 'crescent-visibility') . '</strong> ';
            /* translators: %s: database error message */
            echo wp_kses(sprintf(__('Database error: %s.', 'crescent-visibility'), '<code>' . esc_html($err) . '</code>'), ['code' => []]) . ' ';
            echo esc_html__('Your database user likely lacks the DROP privilege.', 'crescent-visibility') . ' ';
            /* translators: %s: database table name */
            echo wp_kses(sprintf(__('Ask your host to grant DROP, or use phpMyAdmin to drop %s manually.', 'crescent-visibility'), '<code>' . esc_html($this->table_observations) . '</code>'), ['code' => []]);
            echo '</p></div>';
            return;
        }

        // Confirm the recreated table actually has AUTO_INCREMENT.
        $id_col   = $wpdb->get_row("SHOW COLUMNS FROM {$this->table_observations} LIKE 'id'", ARRAY_A);
        $id_extra = is_array($id_col) ? ($id_col['Extra'] ?? '') : 'no id column';

        echo '<div class="notice notice-success"><p><strong>' . esc_html__('Tables dropped and recreated empty.', 'crescent-visibility') . '</strong> ';
        /* translators: %s: id column definition */
        echo wp_kses(sprintf(__('id column is now: %s.', 'crescent-visibility'), '<code>' . esc_html($id_extra !== '' ? $id_extra : __('(no AUTO_INCREMENT)', 'crescent-visibility')) . '</code>'), ['code' => []]) . ' ';
        echo esc_html__('Now re-import your JSON to repopulate.', 'crescent-visibility');
        echo '</p></div>';
    }

    private function handle_import() {
        // Large datasets: don't let a slow host abort mid-import.
        @set_time_limit(0);

        if (empty($_FILES['import_file']['tmp_name'])) {
            $this->record_import_result(['error' => __('No file was received by the server. Check your upload size limits (post_max_size / upload_max_filesize).', 'crescent-visibility')]);
            $this->print_import_notice();
            return;
        }

        $upload = $_FILES['import_file'];
        $size   = (int) ($upload['size'] ?? 0);

        if (!empty($upload['error'])) {
            /* translators: %d: PHP upload error code */
            $this->record_import_result(['error' => sprintf(__('Upload error code %d (often a server file-size limit).', 'crescent-visibility'), intval($upload['error'])), 'file_size' => $size]);
            $this->print_import_notice();
            return;
        }

        // Guard against a runaway upload exhausting memory during json_decode.
        // 64 MB is far above any realistic dataset (the 30-city, 50-year file is
        // ~8 MB) while keeping the parse bounded.
        $max_bytes = (int) apply_filters('cvi_max_import_bytes', 64 * 1024 * 1024);
        if ($size > $max_bytes) {
            $this->record_import_result([
                /* translators: 1: uploaded size in bytes, 2: limit in bytes */
                'error'     => sprintf(__('File too large (%1$s bytes). The limit is %2$s bytes. Generate a smaller dataset (fewer years/cities).', 'crescent-visibility'), number_format($size), number_format($max_bytes)),
                'file_size' => $size,
            ]);
            $this->print_import_notice();
            return;
        }

        $raw  = file_get_contents($upload['tmp_name']);
        $data = json_decode($raw, true);

        if (!is_array($data) || empty($data['observations']) || !is_array($data['observations'])) {
            $hint = (json_last_error() !== JSON_ERROR_NONE)
                /* translators: %s: JSON parser error message */
                ? sprintf(__('JSON parse error: %s (the file may be truncated by an upload limit).', 'crescent-visibility'), json_last_error_msg())
                : __('The file must contain a non-empty "observations" array.', 'crescent-visibility');
            $this->record_import_result(['error' => $hint, 'file_size' => $size]);
            $this->print_import_notice();
            return;
        }

        global $wpdb;

        // Rebuild the tables from scratch before importing. dbDelta cannot fix a
        // pre-existing table whose PRIMARY KEY lost AUTO_INCREMENT or whose unique
        // index is wrong — which makes every REPLACE collide onto a single row
        // (observed live: 1612 inserts, 1 surviving row). An import always carries
        // the full dataset, so a clean drop+recreate is safe and reliable.
        $wpdb->query("DROP TABLE IF EXISTS {$this->table_observations}");
        $wpdb->query("DROP TABLE IF EXISTS {$this->table_cities}");
        $this->create_or_update_tables();

        // Capture the real id-column definition so we can SEE whether the table
        // actually has AUTO_INCREMENT (some hosts/legacy tables don't, which is
        // what made every insert collapse onto id=0).
        $id_col  = $wpdb->get_row("SHOW COLUMNS FROM {$this->table_observations} LIKE 'id'", ARRAY_A);
        $id_extra = is_array($id_col) ? ($id_col['Extra'] ?? '') : 'no id column';
        $ddl_error = $wpdb->last_error;

        $cities_imported = 0;
        $city_id = 1; // Explicit id — does NOT rely on AUTO_INCREMENT (same reason as observations).
        if (!empty($data['cities']) && is_array($data['cities'])) {
            foreach ($data['cities'] as $city) {
                $slug = sanitize_text_field($city['slug'] ?? '');
                if ($slug === '') {
                    continue;
                }
                $ok = $wpdb->insert($this->table_cities, [
                    'id'        => $city_id++,
                    'slug'      => $slug,
                    'name'      => sanitize_text_field($city['name'] ?? ''),
                    'latitude'  => floatval($city['latitude'] ?? 0),
                    'longitude' => floatval($city['longitude'] ?? 0),
                ]);
                if ($ok !== false) {
                    $cities_imported++;
                }
            }
        }

        $total       = count($data['observations']);
        $obs_imported = 0;
        $obs_failed   = 0;
        $obs_skipped  = 0;
        $first_error  = '';
        $first_skip   = '';
        $next_id      = 1; // Explicit, guaranteed-unique id — does NOT rely on AUTO_INCREMENT.

        foreach ($data['observations'] as $obs) {
            $city  = isset($obs['city']) ? sanitize_text_field($obs['city']) : '';
            $moon  = isset($obs['new_moon']) ? sanitize_text_field($obs['new_moon']) : '';

            // Skip rows missing the keys that form the unique index — otherwise
            // they all collapse onto one empty-keyed row (observed live: 1 row).
            if ($city === '' || $moon === '') {
                $obs_skipped++;
                if ($first_skip === '') {
                    $first_skip = 'keys present: ' . implode(', ', array_keys((array) $obs));
                }
                continue;
            }

            $days = isset($obs['days']) && is_array($obs['days']) ? $obs['days'] : [];
            $dq   = isset($obs['day_q']) && is_array($obs['day_q']) ? $obs['day_q'] : [];
            $da   = isset($obs['day_age']) && is_array($obs['day_age']) ? $obs['day_age'] : [];
            // INSERT (not REPLACE) into the freshly-recreated empty table: if rows
            // are colliding on the unique key, INSERT fails with the exact
            // "Duplicate entry 'X-Y' for key 'city_newmoon'" message instead of
            // silently overwriting — turning the collapse into a precise error.
            $ok = $wpdb->insert($this->table_observations, [
                'id'              => $next_id++,
                'city'            => $city,
                'new_moon_date'   => $moon,
                'year'            => intval($obs['year'] ?? 0),
                'raw_day_0'       => sanitize_text_field($days[0] ?? '?'),
                'raw_day_1'       => sanitize_text_field($days[1] ?? '?'),
                'raw_day_2'       => sanitize_text_field($days[2] ?? '?'),
                'best_raw'        => sanitize_text_field($obs['best_raw'] ?? '?'),
                'best_effective'  => sanitize_text_field($obs['best_effective'] ?? '?'),
                'q_at_best'       => isset($obs['q_at_best']) ? floatval($obs['q_at_best']) : null,
                'moon_age_at_best'=> isset($obs['moon_age_at_best']) ? floatval($obs['moon_age_at_best']) : null,
                'q_day_0'         => isset($dq[0]) ? floatval($dq[0]) : null,
                'q_day_1'         => isset($dq[1]) ? floatval($dq[1]) : null,
                'q_day_2'         => isset($dq[2]) ? floatval($dq[2]) : null,
                'age_day_0'       => isset($da[0]) ? floatval($da[0]) : null,
                'age_day_1'       => isset($da[1]) ? floatval($da[1]) : null,
                'age_day_2'       => isset($da[2]) ? floatval($da[2]) : null,
                'data_version'    => sanitize_text_field($data['meta']['generator'] ?? 'yallop'),
            ]);

            if ($ok === false) {
                $obs_failed++;
                if ($first_error === '' && !empty($wpdb->last_error)) {
                    $first_error = $wpdb->last_error;
                }
            } else {
                $obs_imported++;
            }
        }

        // How many distinct rows are now actually stored (catches REPLACE collapse).
        $stored_total = (int) $wpdb->get_var("SELECT COUNT(*) FROM {$this->table_observations}");

        // Empirical diagnostics: read back what actually landed + the real index
        // layout, so we can see WHY rows collapse instead of guessing.
        $sample_rows = '';
        $rows_back = $wpdb->get_results("SELECT id, city, new_moon_date, year FROM {$this->table_observations} ORDER BY id LIMIT 6", ARRAY_A);
        foreach ((array) $rows_back as $rb) {
            $sample_rows .= '#' . $rb['id'] . ' ' . $rb['city'] . '/' . $rb['new_moon_date'] . '/' . $rb['year'] . '  ';
        }

        $index_info = '';
        $idx = $wpdb->get_results("SHOW INDEX FROM {$this->table_observations}", ARRAY_A);
        foreach ((array) $idx as $ix) {
            $index_info .= $ix['Key_name'] . '(' . $ix['Column_name'] . ') ';
        }

        // Distinct counts straight from the DB tell us if the *columns* collapsed.
        $distinct_pairs = (int) $wpdb->get_var("SELECT COUNT(DISTINCT city, new_moon_date) FROM {$this->table_observations}");

        $this->record_import_result([
            'cities'         => $cities_imported,
            'found'          => $total,
            'imported'       => $obs_imported,
            'skipped'        => $obs_skipped,
            'failed'         => $obs_failed,
            'stored_total'   => $stored_total,
            'first_error'    => $first_error,
            'first_skip'     => $first_skip,
            'file_size'      => $size,
            'id_column'      => $id_extra,
            'ddl_error'      => $ddl_error,
            'sample_rows'    => $sample_rows,
            'indexes'        => $index_info,
            'distinct_pairs' => $distinct_pairs,
        ]);

        $this->print_import_notice();
    }

    /**
     * Persist the outcome of the last import so it is visible on every Tools
     * page load (not just on the POST response). This makes remote diagnosis
     * possible from a simple screenshot of the page.
     */
    private function record_import_result(array $result) {
        $result['time'] = current_time('mysql');

        // Bound stored diagnostic strings (DB error text, sample rows) so a
        // hostile/huge error message can't bloat the option.
        foreach ($result as $k => $v) {
            if (is_string($v) && strlen($v) > 500) {
                $result[$k] = substr($v, 0, 500) . '…';
            }
        }

        // autoload=no: this is only read on the Tools admin page, never on the front end.
        update_option('cvi_last_import', $result, false);
    }

    /**
     * Render an admin notice describing the last import result.
     */
    private function print_import_notice() {
        $r = get_option('cvi_last_import');
        if (!is_array($r)) {
            return;
        }

        if (!empty($r['error'])) {
            echo '<div class="notice notice-error"><p><strong>' . esc_html__('Import failed.', 'crescent-visibility') . '</strong> ' . esc_html($r['error']) . '</p></div>';
            return;
        }

        $imported = intval($r['imported'] ?? 0);
        $found    = intval($r['found'] ?? 0);
        $skipped  = intval($r['skipped'] ?? 0);
        $failed   = intval($r['failed'] ?? 0);

        if ($imported === 0) {
            echo '<div class="notice notice-error"><p><strong>' . esc_html__('Import failed.', 'crescent-visibility') . '</strong> ';
            /* translators: 1: observations found, 2: skipped count, 3: db error count */
            echo esc_html(sprintf(__('Found %1$d observations in the file but stored none (%2$d skipped, %3$d db errors).', 'crescent-visibility'), $found, $skipped, $failed)) . '</p>';
            if (!empty($r['first_error'])) {
                /* translators: %s: database error message */
                echo '<p>' . wp_kses(sprintf(__('Database error: %s', 'crescent-visibility'), '<code>' . esc_html($r['first_error']) . '</code>'), ['code' => []]) . '</p>';
            }
            if (!empty($r['first_skip'])) {
                /* translators: %s: list of JSON keys present on the row */
                echo '<p>' . esc_html(sprintf(__('First skipped row %s — the file rows are missing city/new_moon.', 'crescent-visibility'), $r['first_skip'])) . '</p>';
            }
            echo '</div>';
            return;
        }

        $stored = intval($r['stored_total'] ?? $imported);

        // Collapse detection: many inserts but few surviving rows means the table
        // keys are broken (lost AUTO_INCREMENT / stale unique index).
        if ($stored < $imported) {
            echo '<div class="notice notice-error"><p><strong>';
            /* translators: 1: rows stored, 2: rows attempted */
            echo esc_html(sprintf(__('Import stored only %1$d of %2$d rows.', 'crescent-visibility'), $stored, $imported)) . '</strong> ';
            echo esc_html__('The database table keys are corrupted (rows are overwriting each other).', 'crescent-visibility') . ' ';
            /* translators: %s: database table name */
            echo wp_kses(sprintf(__('Deactivate the plugin, then in your database drop the %s table, reactivate, and import again.', 'crescent-visibility'), '<code>' . esc_html($this->table_observations) . '</code>'), ['code' => []]);
            echo '</p></div>';
            return;
        }

        $class = ($skipped > 0 || $failed > 0) ? 'notice-warning' : 'notice-success';
        $headline = ($skipped || $failed) ? __('Import completed with warnings!', 'crescent-visibility') : __('Import successful!', 'crescent-visibility');
        echo '<div class="notice ' . esc_attr($class) . '"><p><strong>' . esc_html($headline) . '</strong> ';
        if (intval($r['cities'] ?? 0)) {
            /* translators: 1: observations stored, 2: total observations, 3: cities stored */
            echo esc_html(sprintf(__('Stored %1$d of %2$d observations and %3$d cities.', 'crescent-visibility'), $stored, $found, intval($r['cities'])));
        } else {
            /* translators: 1: observations stored, 2: total observations */
            echo esc_html(sprintf(__('Stored %1$d of %2$d observations.', 'crescent-visibility'), $stored, $found));
        }
        if ($skipped > 0) {
            /* translators: %d: skipped row count */
            echo ' ' . esc_html(sprintf(__('Skipped %d (missing city/new_moon).', 'crescent-visibility'), $skipped));
        }
        if ($failed > 0) {
            /* translators: %d: db error count */
            echo ' ' . esc_html(sprintf(__('%d db errors.', 'crescent-visibility'), $failed));
        }
        echo '</p>';
        if (!empty($r['first_error'])) {
            /* translators: %s: database error message */
            echo '<p>' . wp_kses(sprintf(__('First db error: %s', 'crescent-visibility'), '<code>' . esc_html($r['first_error']) . '</code>'), ['code' => []]) . '</p>';
        }
        /* translators: %s: shortcode wrapped in <code> */
        echo '<p>' . wp_kses(sprintf(__('You can now use %s.', 'crescent-visibility'), '<code>[crescent_visibility_interactive]</code>'), ['code' => []]) . '</p></div>';
    }

    public function render_shortcode($atts) {
        $atts = shortcode_atts([
            'city'          => 'jerusalem',
            'years'         => '2026-2035',
            'cloud_cover'   => 0,
            'transparency'  => 7,
        ], $atts);

        return crescent_visibility_render($atts);
    }

    /**
     * Helper: Get distinct years that have data for a given city.
     */
    public function get_available_years_for_city($city_slug) {
        global $wpdb;
        $table = $wpdb->prefix . 'crescent_observations';

        return $wpdb->get_col($wpdb->prepare(
            "SELECT DISTINCT year FROM $table WHERE city = %s ORDER BY year ASC",
            $city_slug
        ));
    }

    /**
     * Helper: Get new moon dates + raw categories for a city + year.
     * Now also returns q_at_best and moon_age_at_best for rich cards (app parity).
     */
    public function get_new_moons_for_city_and_year($city_slug, $year) {
        global $wpdb;
        $table = $wpdb->prefix . 'crescent_observations';

        return $wpdb->get_results($wpdb->prepare(
            "SELECT 
                new_moon_date, 
                best_raw, best_effective, 
                raw_day_0, raw_day_1, raw_day_2,
                q_at_best, moon_age_at_best,
                data_version
             FROM $table 
             WHERE city = %s AND year = %d 
             ORDER BY new_moon_date ASC",
            $city_slug, $year
        ), ARRAY_A);
    }

    /**
     * Create or update the plugin tables. Idempotent (dbDelta), so it is safe
     * to call on activation and again right before an import to guarantee the
     * q_at_best / moon_age_at_best / data_version columns exist before we try
     * to insert into them.
     */
    public function create_or_update_tables() {
        global $wpdb;
        require_once ABSPATH . 'wp-admin/includes/upgrade.php';

        $charset_collate = $wpdb->get_charset_collate();

        $sql_cities = "CREATE TABLE {$this->table_cities} (
            id INT UNSIGNED NOT NULL AUTO_INCREMENT,
            slug VARCHAR(50) NOT NULL,
            name VARCHAR(100) NOT NULL,
            latitude DECIMAL(10,6) NOT NULL,
            longitude DECIMAL(10,6) NOT NULL,
            is_active TINYINT(1) NOT NULL DEFAULT 1,
            sort_order SMALLINT NOT NULL DEFAULT 100,
            PRIMARY KEY (id),
            UNIQUE KEY slug (slug)
        ) $charset_collate;";

        $sql_obs = "CREATE TABLE {$this->table_observations} (
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

        dbDelta($sql_cities);
        dbDelta($sql_obs);

        if (defined('CVI_SCHEMA_VERSION')) {
            update_option('cvi_schema_version', CVI_SCHEMA_VERSION);
        }
        if (defined('CVI_VERSION')) {
            update_option('cvi_plugin_version', CVI_VERSION);
        }
    }

    public function activate() {
        $this->create_or_update_tables();
        $this->seed_sample_data();
    }

    /**
     * On a fresh install, load the bundled small sample dataset (3 cities,
     * 2026-2028) so the shortcode works immediately. Skips silently if the
     * table already has data or the sample file is missing.
     */
    private function seed_sample_data() {
        global $wpdb;

        $existing = (int) $wpdb->get_var("SELECT COUNT(*) FROM {$this->table_observations}");
        if ($existing > 0) {
            return;
        }

        $file = plugin_dir_path(__FILE__) . 'data/sample.json';
        if (!is_readable($file)) {
            return;
        }
        $data = json_decode(file_get_contents($file), true);
        if (!is_array($data) || empty($data['observations'])) {
            return;
        }

        $city_id = 1;
        foreach (($data['cities'] ?? []) as $c) {
            $slug = sanitize_text_field($c['slug'] ?? '');
            if ($slug === '') {
                continue;
            }
            $wpdb->insert($this->table_cities, [
                'id'        => $city_id++,
                'slug'      => $slug,
                'name'      => sanitize_text_field($c['name'] ?? ''),
                'latitude'  => floatval($c['latitude'] ?? 0),
                'longitude' => floatval($c['longitude'] ?? 0),
            ]);
        }

        $obs_id = 1;
        foreach ($data['observations'] as $o) {
            $city = sanitize_text_field($o['city'] ?? '');
            $moon = sanitize_text_field($o['new_moon'] ?? '');
            if ($city === '' || $moon === '') {
                continue;
            }
            $days = isset($o['days']) && is_array($o['days']) ? $o['days'] : [];
            $dq   = isset($o['day_q']) && is_array($o['day_q']) ? $o['day_q'] : [];
            $da   = isset($o['day_age']) && is_array($o['day_age']) ? $o['day_age'] : [];
            $wpdb->insert($this->table_observations, [
                'id'              => $obs_id++,
                'city'            => $city,
                'new_moon_date'   => $moon,
                'year'            => intval($o['year'] ?? 0),
                'raw_day_0'       => sanitize_text_field($days[0] ?? '?'),
                'raw_day_1'       => sanitize_text_field($days[1] ?? '?'),
                'raw_day_2'       => sanitize_text_field($days[2] ?? '?'),
                'best_raw'        => sanitize_text_field($o['best_raw'] ?? '?'),
                'best_effective'  => sanitize_text_field($o['best_effective'] ?? '?'),
                'q_at_best'       => isset($o['q_at_best']) ? floatval($o['q_at_best']) : null,
                'moon_age_at_best'=> isset($o['moon_age_at_best']) ? floatval($o['moon_age_at_best']) : null,
                'q_day_0'         => isset($dq[0]) ? floatval($dq[0]) : null,
                'q_day_1'         => isset($dq[1]) ? floatval($dq[1]) : null,
                'q_day_2'         => isset($dq[2]) ? floatval($dq[2]) : null,
                'age_day_0'       => isset($da[0]) ? floatval($da[0]) : null,
                'age_day_1'       => isset($da[1]) ? floatval($da[1]) : null,
                'age_day_2'       => isset($da[2]) ? floatval($da[2]) : null,
                'data_version'    => 'sample',
            ]);
        }
    }
}

// -----------------------------------------------------------------------------
// Module loading — clean dependency order (production structure)
// 1. public/renderer.php             — original static shortcode
// 2. includes/interactive.php        — AJAX handlers, smart defaults, PHP heuristic
// 3. public/interactive-renderer.php — interactive UI + asset registration

require_once plugin_dir_path(__FILE__) . 'public/renderer.php';
require_once plugin_dir_path(__FILE__) . 'includes/interactive.php';
require_once plugin_dir_path(__FILE__) . 'public/interactive-renderer.php';

$cv_plugin = new Crescent_Visibility_Plugin();

register_activation_hook(__FILE__, [$cv_plugin, 'activate']);