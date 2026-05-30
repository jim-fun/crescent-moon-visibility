<?php
/**
 * EXPLORATORY FOCUSED VOLUME DRAFT v4 - qwen3-coder:30b style
 * Target: AJAX / Dynamic Loading + Smart Defaults (the critical "feels like the app" layer)
 *
 * This is a targeted follow-up pass after v3 (app-parity interactive).
 * Instead of another full shortcode rewrite, this file concentrates volume on the
 * data loading and initialization problems that currently make the interactive form
 * feel static or fake.
 *
 * Specific focus areas requested by user direction:
 * - Real AJAX sketch (wp_ajax + REST route options)
 * - Dynamic year → new moon population from actual DB data
 * - Smart defaults: Jerusalem + closest/next new moon based on current date
 * - Loading states, error handling, graceful degradation
 * - Multiple approaches documented (preload vs lazy fetch, admin-ajax vs REST)
 *
 * This file is meant to be combined with (or replace sections of) the v3 draft
 * during the Claude production pass.
 *
 * It deliberately does NOT re-implement the full card UI or heuristic again.
 */

// -----------------------------------------------------------------------------
// 1. PHP: AJAX HANDLERS (the missing piece for real interactivity)
// -----------------------------------------------------------------------------

/**
 * Approach A (recommended for minimal footprint WP plugin):
 * wp_ajax + wp_ajax_nopriv handlers.
 * Simple, no extra dependencies, works on almost every host.
 */
add_action('wp_ajax_cvi_get_years', 'cvi_ajax_get_years');
add_action('wp_ajax_nopriv_cvi_get_years', 'cvi_ajax_get_years');

function cvi_ajax_get_years() {
    $city = sanitize_text_field($_GET['city'] ?? 'jerusalem');

    // Use the plugin instance if available, otherwise fall back to direct query
    global $cv_plugin;
    $years = [];

    if ($cv_plugin && method_exists($cv_plugin, 'get_available_years_for_city')) {
        $years = $cv_plugin->get_available_years_for_city($city);
    } else {
        global $wpdb;
        $table = $wpdb->prefix . 'crescent_observations';
        $years = $wpdb->get_col($wpdb->prepare(
            "SELECT DISTINCT year FROM $table WHERE city = %s ORDER BY year ASC",
            $city
        ));
    }

    wp_send_json_success(['years' => array_map('intval', $years)]);
}

add_action('wp_ajax_cvi_get_newmoons', 'cvi_ajax_get_newmoons');
add_action('wp_ajax_nopriv_cvi_get_newmoons', 'cvi_ajax_get_newmoons');

function cvi_ajax_get_newmoons() {
    $city = sanitize_text_field($_GET['city'] ?? 'jerusalem');
    $year = intval($_GET['year'] ?? 0);

    if (!$year) {
        wp_send_json_error(['message' => 'Year required']);
    }

    global $cv_plugin;
    $rows = [];

    if ($cv_plugin && method_exists($cv_plugin, 'get_new_moons_for_city_and_year')) {
        $rows = $cv_plugin->get_new_moons_for_city_and_year($city, $year);
    } else {
        global $wpdb;
        $table = $wpdb->prefix . 'crescent_observations';
        $rows = $wpdb->get_results($wpdb->prepare(
            "SELECT new_moon_date, raw_day_0, raw_day_1, raw_day_2, 
                    best_raw, best_effective, q_at_best, moon_age_at_best
             FROM $table 
             WHERE city = %s AND year = %d 
             ORDER BY new_moon_date ASC",
            $city, $year
        ), ARRAY_A);
    }

    // Normalize for JS consumption (make sure q and age are numbers or null)
    $normalized = array_map(function($row) {
        return [
            'new_moon_date'   => $row['new_moon_date'],
            'days'            => [$row['raw_day_0'], $row['raw_day_1'], $row['raw_day_2']],
            'q_at_best'       => isset($row['q_at_best']) ? (float)$row['q_at_best'] : null,
            'moon_age_at_best'=> isset($row['moon_age_at_best']) ? (float)$row['moon_age_at_best'] : null,
        ];
    }, $rows);

    wp_send_json_success(['new_moons' => $normalized]);
}

// -----------------------------------------------------------------------------
// Approach B sketch: REST route (more modern, but slightly heavier)
// -----------------------------------------------------------------------------
add_action('rest_api_init', function() {
    register_rest_route('crescent/v1', '/newmoons/(?P<city>[a-z-]+)/(?P<year>\d{4})', [
        'methods'  => 'GET',
        'callback' => 'cvi_rest_get_newmoons',
        'permission_callback' => '__return_true',
    ]);

    register_rest_route('crescent/v1', '/years/(?P<city>[a-z-]+)', [
        'methods'  => 'GET',
        'callback' => 'cvi_rest_get_years',
        'permission_callback' => '__return_true',
    ]);
});

function cvi_rest_get_newmoons($request) {
    // Reuse the ajax logic or call the plugin method directly
    // For volume exploration we just note that this is the cleaner long-term path
    $city = sanitize_text_field($request['city']);
    $year = intval($request['year']);

    // In real code we would call the same logic as cvi_ajax_get_newmoons
    // and return new WP_REST_Response(...)
    return new WP_REST_Response(['new_moons' => []], 200); // placeholder
}

function cvi_rest_get_years($request) {
    // Similar placeholder
    return new WP_REST_Response(['years' => []], 200);
}

// -----------------------------------------------------------------------------
// 2. SMART DEFAULTS - "Jerusalem + closest/next new moon" like the web app
// -----------------------------------------------------------------------------

/**
 * Server-side helper to find the best default new moon for a city.
 * This is the piece the original web app does automatically.
 */
function cvi_get_smart_default_new_moon($city = 'jerusalem') {
    global $wpdb;
    $table = $wpdb->prefix . 'crescent_observations';

    $today = current_time('Y-m-d');

    // Strategy 1 (preferred): Find the next future new moon we have data for
    $next = $wpdb->get_row($wpdb->prepare(
        "SELECT new_moon_date, year 
         FROM $table 
         WHERE city = %s AND new_moon_date >= %s 
         ORDER BY new_moon_date ASC 
         LIMIT 1",
        $city, $today
    ), ARRAY_A);

    if ($next) {
        return [
            'new_moon_date' => $next['new_moon_date'],
            'year'          => (int)$next['year'],
            'reason'        => 'next_future',
        ];
    }

    // Strategy 2: Fall back to the most recent one we have
    $latest = $wpdb->get_row($wpdb->prepare(
        "SELECT new_moon_date, year 
         FROM $table 
         WHERE city = %s 
         ORDER BY new_moon_date DESC 
         LIMIT 1",
        $city
    ), ARRAY_A);

    if ($latest) {
        return [
            'new_moon_date' => $latest['new_moon_date'],
            'year'          => (int)$latest['year'],
            'reason'        => 'most_recent',
        ];
    }

    // Final fallback
    return ['new_moon_date' => null, 'year' => 2026, 'reason' => 'hardcoded_fallback'];
}

/**
 * Shortcode attribute helper so the interactive form can receive good defaults
 * from the server on initial page load (important for SEO + no-JS users).
 */
function cvi_get_initial_defaults($atts) {
    $city = sanitize_text_field($atts['default_city'] ?? 'jerusalem');

    $smart = cvi_get_smart_default_new_moon($city);

    return [
        'city'          => $city,
        'year'          => $smart['year'],
        'new_moon_date' => $smart['new_moon_date'],
        'reason'        => $smart['reason'],
    ];
}

// -----------------------------------------------------------------------------
// 3. JAVASCRIPT LOADING LAYER (the frontend half of the story)
// -----------------------------------------------------------------------------
// This section is written as a drop-in replacement / augmentation for the
// loading code in v3. It demonstrates clean, production-leaning patterns
// while still keeping the exploratory volume style (multiple options, TODOs).

/*
<script>
(function() {
    // ... existing v3 code ...

    // === DYNAMIC LOADING MANAGER (new in v4) ===
    const LoadingManager = {
        cache: {}, // city -> year -> newmoons

        async fetchYears(city) {
            const res = await fetch(`/wp-admin/admin-ajax.php?action=cvi_get_years&city=${city}`);
            const json = await res.json();
            if (!json.success) throw new Error(json.data?.message || 'Failed to load years');
            return json.data.years;
        },

        async fetchNewMoons(city, year) {
            const key = `${city}:${year}`;
            if (this.cache[key]) return this.cache[key];

            const res = await fetch(`/wp-admin/admin-ajax.php?action=cvi_get_newmoons&city=${city}&year=${year}`);
            const json = await res.json();
            if (!json.success) throw new Error('Failed to load new moons');

            this.cache[key] = json.data.new_moons;
            return json.data.new_moons;
        },

        // Alternative REST approach (Approach B)
        async fetchNewMoonsREST(city, year) {
            const res = await fetch(`/wp-json/crescent/v1/newmoons/${city}/${year}`);
            return (await res.json()).new_moons || [];
        }
    };

    // Improved year change handler with loading state
    async function onYearChange(city, year) {
        const select = document.getElementById('cvi-newmoon');
        select.innerHTML = '<option>Loading new moons...</option>';
        select.disabled = true;

        try {
            const moons = await LoadingManager.fetchNewMoons(city, year);

            select.innerHTML = '';
            moons.forEach(m => {
                const opt = document.createElement('option');
                opt.value = m.new_moon_date;
                opt.textContent = m.new_moon_date;
                opt.dataset.days = JSON.stringify(m.days);
                opt.dataset.q = m.q_at_best ?? '';
                opt.dataset.age = m.moon_age_at_best ?? '';
                select.appendChild(opt);
            });

            if (moons.length) {
                // Auto-select the first (or implement "closest to today" logic)
                select.selectedIndex = 0;
            }
        } catch (err) {
            select.innerHTML = `<option>Error loading data</option>`;
            console.error(err);
        } finally {
            select.disabled = false;
        }
    }

    // On initial load, use server-provided smart defaults when possible
    // (the PHP side already computed cvi_get_smart_default_new_moon)
    function bootstrapWithDefaults(defaultsFromServer) {
        // defaultsFromServer = { city, year, new_moon_date, reason }
        // ... populate selects and trigger first load ...
    }

    // TODO (for Claude): 
    // - Debounce rapid city/year changes
    // - Add abortController for in-flight fetches
    // - Show skeleton cards while loading
    // - Cache invalidation strategy when admin re-imports data
})();
</script>
*/

// -----------------------------------------------------------------------------
// 4. INITIALIZATION + DEFAULTS WIRING (how the shortcode should boot)
// -----------------------------------------------------------------------------
/**
 * Enhanced shortcode that now accepts and uses smart defaults.
 * This is the version that should be used after merging v3 + v4 work.
 */
function crescent_visibility_interactive_v4_shortcode($atts) {
    $atts = shortcode_atts([
        'default_city' => 'jerusalem',
    ], $atts);

    // NEW in v4: ask the server for intelligent defaults
    $initial = cvi_get_initial_defaults($atts);

    // We pass the smart defaults down to the JS via data attributes or inline script.
    // This is the key to feeling like the original web app on first load.

    ob_start();
    ?>
    <div id="cvi-app-v4" 
         data-default-city="<?= esc_attr($initial['city']) ?>"
         data-default-year="<?= esc_attr($initial['year']) ?>"
         data-default-newmoon="<?= esc_attr($initial['new_moon_date']) ?>"
         data-default-reason="<?= esc_attr($initial['reason']) ?>">

        <!-- The rest of the form is the same as v3, but the JS now reads the
             data-* attributes on mount and pre-selects the best new moon. -->

        <!-- (In real merged code this would include the full v3 markup + the
             new LoadingManager from above) -->

        <p><strong>Smart default reason (dev):</strong> <?= esc_html($initial['reason']) ?> — <?= esc_html($initial['new_moon_date']) ?></p>
    </div>

    <script>
    // Minimal bootstrap example (volume style)
    (function() {
        const root = document.getElementById('cvi-app-v4');
        if (!root) return;

        const defaults = {
            city: root.dataset.defaultCity,
            year: parseInt(root.dataset.defaultYear),
            newMoon: root.dataset.defaultNewmoon,
        };

        console.log('%c[v4] Smart defaults received from server:', 'color:#0d6efd', defaults);

        // TODO: Wire this into the real city/year/newmoon selects and trigger
        // the first fetch + render using the LoadingManager above.
    })();
    </script>
    <?php
    return ob_get_clean();
}

// Re-register under the same names so it can replace v3 during testing
add_shortcode('crescent_visibility_interactive_v4', 'crescent_visibility_interactive_v4_shortcode');

// -----------------------------------------------------------------------------
// CLAUDE NOTES SPECIFIC TO THIS v4 PASS
// -----------------------------------------------------------------------------
/*
For the final production implementation you should:

1. Merge the AJAX handlers from this file into the main plugin class or a
   dedicated includes/ajax.php.

2. Make the interactive shortcode (from v3) call cvi_get_initial_defaults()
   and seed the JS with those values on first render.

3. Choose ONE loading strategy (admin-ajax is simplest and recommended for
   minimal footprint; REST is nicer if you ever want to support blocks or
   headless).

4. Add the "next new moon" logic into a proper method on the plugin class
   so it can be unit tested.

5. Consider adding a small "data_version" filter so the frontend can warn
   the user if the imported data is old.

6. Loading states and optimistic UI updates are critical for feeling "like the app".

This v4 pass gives you the raw material for the data flow. Combine it with
the card rendering and heuristic work from v3.
*/
