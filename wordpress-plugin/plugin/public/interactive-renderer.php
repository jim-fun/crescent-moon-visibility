<?php
/**
 * Production renderer for [crescent_visibility_interactive] / [crescent_visibility_point].
 *
 * Renders the interactive "Visibility for My Location" experience: a city /
 * year / new-moon selector, atmospheric sliders, three rich result cards, a
 * non-interactive context map, and a legend — all powered by pre-computed
 * data via the AJAX endpoints in includes/interactive.php.
 *
 * Markup uses CSS classes from assets/css/interactive.css (light default with
 * an optional data-theme="dark" zinc variant). A server-rendered <noscript>
 * fallback shows the smart-default new moon so the page is useful without JS.
 *
 * @package Crescent_Visibility
 */

if (!defined('ABSPATH')) {
    exit;
}

/**
 * Effective-category colors, matching the web app's Tailwind palette
 * (text-cyan-400 / text-cyan-300 / text-yellow-400 / text-yellow-300 /
 * text-amber-500). Shared by the server-side fallback and mirrored in JS.
 *
 * @return array<string,string>
 */
function cvi_category_colors() {
    return ['A' => '#22d3ee', 'B' => '#67e8f9', 'C' => '#facc15', 'D' => '#fde047', 'E' => '#f59e0b'];
}

/**
 * Render one result card server-side (used for the no-JS fallback). Mirrors
 * the JS card markup and the web app's graceful handling of "J" / "?" /
 * large moon ages.
 *
 * @param string $label Day label, e.g. "Day +0".
 * @param string $raw   Raw category letter.
 * @param int    $cloud Cloud cover 0-100.
 * @param float  $trans Transparency 1-10.
 * @param float  $age   Moon age in hours (best evening).
 * @param float  $q     Q value (best evening).
 */
function cvi_render_card($label, $raw, $cloud, $trans, $age, $q) {
    $raw = strtoupper((string) $raw);

    if ($raw === 'J' || $raw === '?' || $raw === '' || $age > 100) {
        return '<div class="cvi-card cvi-card--empty">'
            . '<div class="cvi-card__label">' . esc_html($label) . '</div>'
            . '<div class="cvi-card__empty-title">' . esc_html__('Not a good crescent window', 'crescent-visibility') . '</div>'
            . '<div class="cvi-card__empty-note">' . esc_html__('The selected date is too far from actual new moon conjunction for reliable prediction.', 'crescent-visibility') . '</div>'
            . '</div>';
    }

    list($eff, $note) = cvi_apply_atmospheric_adjustment($raw, $cloud, $trans);
    $colors = cvi_category_colors();
    $color  = $colors[$eff] ?? '#64748b';

    /* translators: 1: moon age in hours, 2: Q value */
    $age_q = sprintf(__('Age: %1$s h • Q: %2$s', 'crescent-visibility'), number_format((float) $age, 1), number_format((float) $q, 3));

    return '<div class="cvi-card cvi-card--cat" style="--cat:' . esc_attr($color) . ';">'
        . '<div class="cvi-card__label">' . esc_html($label) . '</div>'
        . '<div class="cvi-card__head">'
        . '<div><div class="cvi-card__sub">' . esc_html__('Effective', 'crescent-visibility') . '</div>'
        . '<div class="cvi-card__big">' . esc_html($eff) . '</div></div>'
        . '<div class="cvi-card__meta">'
        . '<div>' . esc_html__('Raw', 'crescent-visibility') . ' <span class="cvi-mono">' . esc_html($raw) . '</span></div>'
        . '<div class="cvi-card__sub">' . esc_html($age_q) . '</div>'
        . '</div></div>'
        . '<div class="cvi-card__note">' . esc_html($note) . '</div>'
        . '</div>';
}

/**
 * Build the static <noscript> fallback for the smart-default new moon.
 */
function cvi_render_noscript_fallback($interactive, $defaults) {
    if (!$interactive || empty($defaults['new_moon_date'])) {
        return '<p class="cvi-noscript">' . wp_kses(
            /* translators: %s: shortcode name wrapped in <code> */
            sprintf(__('Enable JavaScript to use the interactive visibility tool, or see the %s shortcode for a static table.', 'crescent-visibility'), '<code>[crescent_visibility]</code>'),
            ['code' => []]
        ) . '</p>';
    }

    $rows = $interactive->get_new_moons_with_details($defaults['city'], $defaults['year']);
    $row  = null;
    foreach ($rows as $candidate) {
        if ($candidate['new_moon_date'] === $defaults['new_moon_date']) {
            $row = $candidate;
            break;
        }
    }

    if (!$row) {
        return '<p class="cvi-noscript">' . esc_html__('Enable JavaScript to use the interactive visibility tool.', 'crescent-visibility') . '</p>';
    }

    // Default atmospheric assumption (clear-ish sky), matching the UI defaults.
    $cloud   = 20;
    $trans   = 7;
    $day_q   = isset($row['day_q']) && is_array($row['day_q']) ? $row['day_q'] : [];
    $day_age = isset($row['day_age']) && is_array($row['day_age']) ? $row['day_age'] : [];
    $labels  = [
        __('Day +0', 'crescent-visibility'),
        __('Day +1', 'crescent-visibility'),
        __('Day +2', 'crescent-visibility'),
    ];

    $cards = '';
    foreach ($row['days'] as $i => $raw) {
        $age = $day_age[$i] ?? 0;
        $q   = $day_q[$i] ?? 0;
        $cards .= cvi_render_card($labels[$i] ?? ('Day +' . $i), $raw, $cloud, $trans, $age, $q);
    }

    return '<div class="cvi-noscript">'
        . '<p>' . sprintf(
            /* translators: %s: a new moon date (YYYY-MM-DD) */
            esc_html__('Showing the default window (%s) at clear-sky conditions. Enable JavaScript for the full interactive tool.', 'crescent-visibility'),
            esc_html($defaults['new_moon_date'])
        ) . '</p>'
        . '<div class="cvi-cards">' . $cards . '</div>'
        . '</div>';
}

/**
 * Shortcode handler for the interactive experience.
 */
function crescent_visibility_render_interactive($atts) {
    $atts = shortcode_atts([
        'default_city' => 'jerusalem',
        'theme'        => 'light', // "light" | "dark"
    ], $atts, 'crescent_visibility_interactive');

    /** @var Crescent_Visibility_Interactive|null $interactive */
    $interactive = $GLOBALS['cvi_interactive'] ?? null;

    $default_city = sanitize_key($atts['default_city']);
    $theme        = $atts['theme'] === 'dark' ? 'dark' : 'light';

    // Smart defaults + the city list straight from the imported data.
    $defaults = ['city' => $default_city, 'year' => (int) current_time('Y'), 'new_moon_date' => null, 'reason' => 'no_data'];
    $cities   = cvi_locked_cities();

    if ($interactive) {
        $defaults = array_merge(['city' => $default_city], $interactive->get_smart_default($default_city));
        $cities   = $interactive->get_cities();
    }

    // Defensive: strip any malformed entry (empty slug), and if nothing usable
    // remains fall back to the locked Phase-0 list. A single blank cities row in
    // the DB previously blanked the whole dropdown (paleotimes.org HAR).
    $cities = array_values(array_filter((array) $cities, function ($c) {
        return is_array($c) && !empty($c['slug']);
    }));
    if (empty($cities)) {
        $cities = cvi_locked_cities();
    }

    // Embed the full precomputed dataset for these cities directly in the page
    // so the tool works with NO admin-ajax/REST call. This is essential where
    // /wp-admin is behind Cloudflare Access (admin-ajax.php is unreachable for
    // logged-out visitors). AJAX remains as a fallback when the embed is absent.
    $dataset = $interactive
        ? $interactive->get_embedded_dataset(wp_list_pluck($cities, 'slug'))
        : [];

    // Enqueue assets only when the shortcode actually renders.
    $base_url = plugin_dir_url(dirname(__FILE__, 2) . '/crescent-visibility.php');
    $asset_ver = defined('CVI_VERSION') ? CVI_VERSION : '0.5.2';

    // Leaflet is bundled locally (no external CDN) per WordPress.org guidelines.
    wp_enqueue_style('cvi-leaflet', $base_url . 'assets/vendor/leaflet/leaflet.css', [], '1.9.4');
    wp_enqueue_script('cvi-leaflet', $base_url . 'assets/vendor/leaflet/leaflet.js', [], '1.9.4', true);

    wp_enqueue_style('cvi-interactive', $base_url . 'assets/css/interactive.css', ['cvi-leaflet'], $asset_ver);
    wp_enqueue_script('cvi-interactive', $base_url . 'assets/js/interactive.js', ['cvi-leaflet', 'wp-i18n'], $asset_ver, true);
    if (function_exists('wp_set_script_translations')) {
        wp_set_script_translations('cvi-interactive', 'crescent-visibility');
    }

    wp_localize_script('cvi-interactive', 'cviInteractiveData', [
        'ajaxUrl'        => admin_url('admin-ajax.php'),
        'nonce'          => wp_create_nonce(Crescent_Visibility_Interactive::NONCE_ACTION),
        'cities'         => $cities,
        'defaultCity'    => $defaults['city'],
        'defaultYear'    => (int) $defaults['year'],
        'defaultNewMoon' => $defaults['new_moon_date'],
    ]);

    ob_start();

    // Robust data passing: put key data in attributes so minifiers can't break wp_localize_script
    $cities_json = wp_json_encode( $cities );
    ?>
    <div class="cvi-interactive-root" 
         data-theme="<?php echo esc_attr($theme); ?>"
         data-cities="<?php echo esc_attr( $cities_json ); ?>"
         data-default-city="<?php echo esc_attr( $defaults['city'] ); ?>"
         data-default-year="<?php echo esc_attr( $defaults['year'] ); ?>"
         data-default-newmoon="<?php echo esc_attr( $defaults['new_moon_date'] ); ?>">

        <script type="application/json" class="cvi-dataset"><?php echo wp_json_encode($dataset, JSON_HEX_TAG | JSON_HEX_AMP | JSON_HEX_APOS | JSON_HEX_QUOT); ?></script>

        <h2 class="cvi-title"><?php esc_html_e('Young Crescent Moon Visibility', 'crescent-visibility'); ?></h2>
        <p class="cvi-intro">
            <?php esc_html_e('Select your nearest city and a New Moon date, then adjust the sky conditions to explore your chances of spotting the delicate thin crescent on each of the first three evenings.', 'crescent-visibility'); ?>
        </p>

        <div class="cvi-field">
            <div class="cvi-quick" id="cvi-quick" role="group" aria-label="<?php esc_attr_e('Quick city selection', 'crescent-visibility'); ?>">
                <span class="cvi-quick__label"><?php esc_html_e('Quick select:', 'crescent-visibility'); ?></span>
                <button type="button" class="cvi-quick__btn" data-slug="jerusalem">Jerusalem</button>
                <button type="button" class="cvi-quick__btn" data-slug="dallas">Dallas</button>
                <button type="button" class="cvi-quick__btn" data-slug="melbourne">Melbourne</button>
            </div>
            <label class="cvi-label" for="cvi-city"><?php esc_html_e('City', 'crescent-visibility'); ?></label>
            <select id="cvi-city" class="cvi-select"></select>
            <div id="cvi-city-context" class="cvi-hint"></div>
        </div>

        <div class="cvi-field cvi-field--row">
            <div>
                <label class="cvi-label" for="cvi-year"><?php esc_html_e('Year', 'crescent-visibility'); ?></label>
                <select id="cvi-year" class="cvi-select"></select>
            </div>
            <div>
                <label class="cvi-label" for="cvi-newmoon"><?php esc_html_e('New Moon (3-day window)', 'crescent-visibility'); ?></label>
                <select id="cvi-newmoon" class="cvi-select">
                    <option value=""><?php esc_html_e('Select a new moon…', 'crescent-visibility'); ?></option>
                </select>
            </div>
        </div>

        <div class="cvi-controls">
            <div class="cvi-controls__title"><?php esc_html_e('Atmospheric Conditions', 'crescent-visibility'); ?></div>
            <div class="cvi-controls__grid">
                <div>
                    <div class="cvi-slider-head">
                        <span><?php esc_html_e('Cloud Cover', 'crescent-visibility'); ?></span>
                        <span><strong id="cvi-cloud-val">20</strong>%</span>
                    </div>
                    <input type="range" id="cvi-cloud" class="cvi-range" min="0" max="100" value="20">
                    <div class="cvi-hint"><?php esc_html_e('0% clear → 100% overcast', 'crescent-visibility'); ?></div>
                </div>
                <div>
                    <label class="cvi-label" for="cvi-trans"><?php esc_html_e('Transparency', 'crescent-visibility'); ?></label>
                    <div class="cvi-slider-inline">
                        <input type="range" id="cvi-trans" class="cvi-range" min="1" max="10" step="0.5" value="7">
                        <input type="number" id="cvi-trans-num" class="cvi-number" min="1" max="10" step="0.5" value="7">
                    </div>
                </div>
            </div>
        </div>

        <div id="cvi-status" class="cvi-status" role="status" aria-live="polite"></div>

        <div id="cvi-results" class="cvi-results" hidden>
            <div class="cvi-results__title"><?php esc_html_e('Results for the three evenings after conjunction', 'crescent-visibility'); ?></div>
            <div id="cvi-cards" class="cvi-cards"></div>
        </div>

        <div class="cvi-map-wrap">
            <div class="cvi-results__title"><?php esc_html_e('Location context', 'crescent-visibility'); ?></div>
            <div id="cvi-map" class="cvi-map"></div>
            <div class="cvi-hint"><?php esc_html_e('Map shown for geographic context only (marker at the selected city).', 'crescent-visibility'); ?></div>
        </div>

        <details class="cvi-legend" open>
            <summary><?php esc_html_e('How to read the visibility rating', 'crescent-visibility'); ?></summary>
            <div class="cvi-legend__grid">
                <div>
                    <span class="cvi-legend__key" style="color:#22d3ee;">A</span> <?php esc_html_e('Easily visible to the naked eye', 'crescent-visibility'); ?><br>
                    <span class="cvi-legend__key" style="color:#67e8f9;">B</span> <?php esc_html_e('Visible naked eye under good conditions', 'crescent-visibility'); ?><br>
                    <span class="cvi-legend__key" style="color:#facc15;">C</span> <?php esc_html_e('Visible naked eye, but requires effort', 'crescent-visibility'); ?>
                </div>
                <div>
                    <span class="cvi-legend__key" style="color:#fde047;">D</span> <?php esc_html_e('Usually needs binoculars or a telescope', 'crescent-visibility'); ?><br>
                    <span class="cvi-legend__key" style="color:#f59e0b;">E</span> <?php esc_html_e('Very difficult or not visible even with aid', 'crescent-visibility'); ?>
                </div>
            </div>
            <div class="cvi-legend__notes">
                <div><strong><?php esc_html_e('Raw', 'crescent-visibility'); ?></strong>: <?php esc_html_e("pure astronomical prediction using the Yallop criterion at the city's location and time.", 'crescent-visibility'); ?></div>
                <div><strong><?php esc_html_e('Effective', 'crescent-visibility'); ?></strong>: <?php esc_html_e('adjusted for the cloud cover and transparency you entered.', 'crescent-visibility'); ?></div>
                <div><strong><?php esc_html_e('Age', 'crescent-visibility'); ?></strong>: <?php esc_html_e('hours since the exact moment of new moon (conjunction).', 'crescent-visibility'); ?></div>
                <div><strong><?php esc_html_e('Q', 'crescent-visibility'); ?></strong>: <?php esc_html_e('model quality factor (higher generally indicates more favorable geometry).', 'crescent-visibility'); ?></div>
            </div>
        </details>

        <p class="cvi-footnote">
            <?php esc_html_e('Yallop criterion (clear-sky baseline). Atmospheric adjustment applied live in the browser. No astronomy is computed at runtime.', 'crescent-visibility'); ?>
            <span class="cvi-version"><?php echo esc_html(sprintf(/* translators: %s: plugin version */ __('Plugin v%s', 'crescent-visibility'), defined('CVI_VERSION') ? CVI_VERSION : '?')); ?></span>
        </p>

        <noscript>
            <?php echo cvi_render_noscript_fallback($interactive, $defaults); // already escaped ?>
        </noscript>
    </div>
    <?php
    return ob_get_clean();
}

add_shortcode('crescent_visibility_interactive', 'crescent_visibility_render_interactive');
add_shortcode('crescent_visibility_point', 'crescent_visibility_render_interactive');
