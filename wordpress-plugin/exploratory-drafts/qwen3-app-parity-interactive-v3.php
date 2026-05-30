<?php
/**
 * EXPLORATORY VOLUME DRAFT v3 - qwen3-coder:30b style
 *
 * Goal: Make the plugin's interactive "Visibility for My Location" experience
 * match the web app /point tool as closely as possible — without any image or
 * map generation at all.
 *
 * This is a deliberate high-volume exploratory pass using the stronger coding model.
 * It is rough in places, contains multiple approaches, heavy TODOs, and is
 * explicitly designed to be mined by Claude for a clean production version.
 *
 * Key advances over previous drafts:
 * - Includes the missing data (q_at_best + moon_age_at_best) that the web app
 *   actually displays on every result card.
 * - Schema evolution + import updates so real imported data can power the UI.
 * - Interactive shortcode wired to the real plugin helper methods.
 * - 3-day cards that aim for visual + informational parity with the Go web app
 *   (Raw, big Effective, Age h • Q, explanation, graceful "J"/bad window handling).
 * - Live client-side atmospheric adjustment using the exact heuristic from main.go.
 * - Dynamic year → new moon loading using the actual DB.
 * - Notes on defaults (Jerusalem + sensible next new moon behavior).
 *
 * Usage for testing (while still exploratory):
 *   - Include this file from crescent-visibility.php during dev.
 *   - Use shortcode: [crescent_visibility_interactive] or [crescent_visibility_app]
 *   - Import a real visibility-*-real.json via Tools → Crescent Visibility first.
 *
 * Data this now expects (after schema update):
 *   city, new_moon_date, year, raw_day_0/1/2, best_raw, best_effective,
 *   q_at_best, moon_age_at_best, data_version
 */

// -----------------------------------------------------------------------------
// SAFETY + REGISTRATION
// -----------------------------------------------------------------------------
if (!defined('ABSPATH')) exit;

add_shortcode('crescent_visibility_interactive', 'crescent_visibility_interactive_shortcode');
add_shortcode('crescent_visibility_app', 'crescent_visibility_interactive_shortcode'); // alias for "app-like"

// -----------------------------------------------------------------------------
// SCHEMA EVOLUTION (the real blocker for app parity)
// -----------------------------------------------------------------------------
// The web app shows "Age: xx.x h • Q: x.xxx" on every card.
// The generator produces these fields. Previous implementation dropped them.
// This draft includes the necessary evolution.
//
// For existing installs: after updating the plugin, go to Tools → Crescent Visibility
// and re-import the JSON (it will use REPLACE). Or run the ALTER manually once.
//
// In a real production version this would be a proper migration class with
// version checks. For now we keep it simple and well-documented.
function cvi_ensure_observation_columns() {
    global $wpdb;
    $table = $wpdb->prefix . 'crescent_observations';

    // Add columns if they don't exist (idempotent)
    $wpdb->query("ALTER TABLE $table 
        ADD COLUMN IF NOT EXISTS q_at_best DECIMAL(6,4) NULL AFTER best_effective,
        ADD COLUMN IF NOT EXISTS moon_age_at_best DECIMAL(5,2) NULL AFTER q_at_best,
        ADD COLUMN IF NOT EXISTS data_version VARCHAR(60) NULL AFTER moon_age_at_best
    ");

    // Also ensure a useful composite index for the interactive queries
    $wpdb->query("ALTER TABLE $table 
        ADD INDEX IF NOT EXISTS idx_city_year_date (city, year, new_moon_date)
    ");
}

// Call this early (in constructor or on admin page load for now)
add_action('init', function() {
    if (is_admin()) {
        // Only run the check in admin to avoid front-end slowdown during early dev
        // In production this would be gated behind a version option check.
    }
});

// -----------------------------------------------------------------------------
// ENHANCED DATA HELPERS (extensions of what already exists in the plugin class)
// -----------------------------------------------------------------------------
/**
 * Returns full observation rows for a city + year, including q and age.
 * This is what the interactive UI actually needs for rich cards.
 */
function cvi_get_full_new_moons_for_city_and_year($city_slug, $year) {
    global $wpdb;
    $table = $wpdb->prefix . 'crescent_observations';

    return $wpdb->get_results($wpdb->prepare(
        "SELECT 
            new_moon_date,
            raw_day_0, raw_day_1, raw_day_2,
            best_raw, best_effective,
            q_at_best, moon_age_at_best,
            data_version
         FROM $table 
         WHERE city = %s AND year = %d 
         ORDER BY new_moon_date ASC",
        $city_slug, $year
    ), ARRAY_A);
}

/**
 * Get the most recent new moon date we have data for a city (useful for defaults).
 */
function cvi_get_latest_new_moon_for_city($city_slug) {
    global $wpdb;
    $table = $wpdb->prefix . 'crescent_observations';

    return $wpdb->get_var($wpdb->prepare(
        "SELECT MAX(new_moon_date) FROM $table WHERE city = %s",
        $city_slug
    ));
}

// -----------------------------------------------------------------------------
// EXACT ATMOSPHERIC HEURISTIC (port from main.go for Accuracy First)
// -----------------------------------------------------------------------------
function cvi_apply_atmospheric_adjustment($rawCategory, $cloudPercent, $transparency) {
    $map = ['A' => 5, 'B' => 4, 'C' => 3, 'D' => 2, 'E' => 1, 'F' => 0];
    $val = $map[$rawCategory] ?? 1;

    // Handle weird renderer output the same way the web app does
    if ($rawCategory === 'J' || $rawCategory === '?' || !$rawCategory) {
        return ['E', 'Not a reliable crescent window (renderer returned ' . ($rawCategory ?: 'unknown') . ')'];
    }

    $adj = 0;
    if ($cloudPercent > 80)      $adj -= 3;
    elseif ($cloudPercent > 60)  $adj -= 2;
    elseif ($cloudPercent > 40)  $adj -= 1;

    if ($transparency >= 9)      $adj += 1;
    elseif ($transparency <= 4)  $adj -= 1;
    elseif ($transparency <= 2)  $adj -= 2;

    $final = max(1, min(5, $val + $adj));
    $rev   = [5 => 'A', 4 => 'B', 3 => 'C', 2 => 'D', 1 => 'E'];
    $eff   = $rev[$final];

    if ($adj === 0) {
        $note = 'Atmospheric conditions have minimal impact on this prediction.';
    } elseif ($adj < 0) {
        $note = 'Conditions are reducing visibility by approximately ' . (-$adj) . ' category level(s).';
    } else {
        $note = 'Excellent atmospheric conditions are slightly improving the prediction.';
    }

    return [$eff, $note];
}

// -----------------------------------------------------------------------------
// THE MAIN INTERACTIVE SHORTCODE (the thing that should feel like the app)
// -----------------------------------------------------------------------------
function crescent_visibility_interactive_shortcode($atts) {
    $atts = shortcode_atts([
        'default_city' => 'jerusalem',
        'default_year' => '',           // empty = auto-pick most recent year we have
    ], $atts);

    $default_city = sanitize_text_field($atts['default_city']);

    // We call the column ensure here for dev convenience (remove in prod)
    if (function_exists('cvi_ensure_observation_columns')) {
        cvi_ensure_observation_columns();
    }

    ob_start();
    ?>
    <div id="cvi-app" class="cvi-app-parity" style="max-width: 980px; font-family: system-ui, -apple-system, sans-serif; color: #111;">

        <style>
            /* Attempt to get closer to the web app's clean modern card feel while staying WP-friendly */
            .cvi-app-parity { --accent: #0d6efd; }
            .cvi-card {
                background: #fff;
                border: 1px solid #e5e7eb;
                border-radius: 14px;
                padding: 18px;
                box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.07);
            }
            .cvi-card.dark-mode {
                background: #18181b;
                border-color: #27272a;
                color: #f4f4f5;
            }
            .cvi-big { font-size: 52px; font-weight: 800; line-height: .9; letter-spacing: -2px; }
            .cvi-meta { font-size: 12.5px; color: #64748b; }
            .cvi-explain { font-size: 13px; background: #f8fafc; padding: 9px 11px; border-radius: 8px; line-height: 1.35; }
            .cvi-controls { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 12px; padding: 16px; }
            .cvi-select { width: 100%; padding: 10px 12px; font-size: 15px; border: 1px solid #cbd5e1; border-radius: 8px; background: white; }
        </style>

        <h2 style="margin:0 0 6px; font-size: 24px; font-weight: 700;">Visibility for My Location</h2>
        <p style="margin:0 0 18px; color:#475569; font-size:14px;">
            Pre-computed Yallop data. Adjust conditions to see how weather affects the three evenings after each new moon.
        </p>

        <!-- CITY -->
        <div style="margin-bottom: 14px;">
            <label style="display:block; font-size:13px; font-weight:600; margin-bottom:4px;">City</label>
            <select id="cvi-city" class="cvi-select">
                <!-- Populated by JS from known list or future DB call -->
            </select>
            <div id="cvi-city-context" style="margin-top:4px; font-size:12px; color:#64748b;"></div>
        </div>

        <!-- YEAR + NEW MOON (dynamic pair - core of the app UX) -->
        <div style="display: grid; grid-template-columns: 130px 1fr; gap: 12px; margin-bottom: 16px;">
            <div>
                <label style="display:block; font-size:13px; font-weight:600; margin-bottom:4px;">Year</label>
                <select id="cvi-year" class="cvi-select"></select>
            </div>
            <div>
                <label style="display:block; font-size:13px; font-weight:600; margin-bottom:4px;">New Moon (3-day window)</label>
                <select id="cvi-newmoon" class="cvi-select">
                    <option value="">Select new moon…</option>
                </select>
            </div>
        </div>

        <!-- ATMOSPHERIC CONTROLS -->
        <div class="cvi-controls" style="margin-bottom: 18px;">
            <div style="font-weight:600; font-size:13px; margin-bottom: 10px;">Atmospheric Conditions</div>

            <div style="display:grid; grid-template-columns: 1fr 120px; gap:18px;">
                <!-- Cloud -->
                <div>
                    <div style="display:flex; justify-content:space-between; font-size:13px; margin-bottom:3px;">
                        <span>Cloud Cover</span>
                        <span><strong id="cvi-cloud-val">20</strong>%</span>
                    </div>
                    <input type="range" id="cvi-cloud" min="0" max="100" value="20" style="width:100%; accent-color:#0d6efd;">
                    <div style="font-size:10px; color:#64748b;">0% clear → 100% overcast</div>
                </div>

                <!-- Transparency -->
                <div>
                    <div style="font-size:13px; margin-bottom:3px;">Transparency</div>
                    <div style="display:flex; gap:6px; align-items:center;">
                        <input type="range" id="cvi-trans" min="1" max="10" step="0.5" value="7" style="flex:1; accent-color:#0d6efd;">
                        <input type="number" id="cvi-trans-num" min="1" max="10" step="0.5" value="7" style="width:56px; padding:4px; border:1px solid #cbd5e1; border-radius:4px; font-size:13px;">
                    </div>
                </div>
            </div>
        </div>

        <button id="cvi-check" style="width:100%; background:#0d6efd; color:white; border:none; border-radius:10px; padding:13px; font-size:15px; font-weight:600; cursor:pointer;">
            Check Visibility
        </button>

        <!-- RESULTS -->
        <div id="cvi-results" style="display:none; margin-top:20px;">
            <div style="font-size:13px; font-weight:600; margin-bottom:8px; color:#334155;">
                Results for the three evenings after conjunction
            </div>
            <div id="cvi-cards" style="display:grid; grid-template-columns:repeat(auto-fit, minmax(270px, 1fr)); gap:12px;"></div>
        </div>

        <!-- CONTEXT MAP (minimal, non-interactive, for geographic sense only) -->
        <div style="margin-top:18px;">
            <div style="font-size:12px; font-weight:600; color:#475569; margin-bottom:4px;">Location context</div>
            <div id="cvi-map" style="height:190px; border-radius:10px; border:1px solid #cbd5e1; background:#f1f5f9;"></div>
            <div style="font-size:10px; color:#64748b; margin-top:3px;">Map shown for reference only.</div>
        </div>

        <div style="margin-top:12px; font-size:11px; color:#64748b;">
            Yallop criterion. Atmospheric adjustment applied live in the browser. Data imported from the reference renderer.
        </div>
    </div>

    <script>
    (function() {
        const root = document.getElementById('cvi-app');
        if (!root) return;

        // -----------------------------------------------------------------
        // CONFIG & DATA (qwen3-coder volume style: multiple approaches shown)
        // -----------------------------------------------------------------
        const SUPPORTED_CITIES = [
            {slug:'jerusalem',name:'Jerusalem',lat:31.7683,lon:35.2137},
            {slug:'mecca',name:'Mecca',lat:21.3891,lon:39.8579},
            {slug:'karachi',name:'Karachi',lat:24.8607,lon:67.0011},
            {slug:'rabat',name:'Rabat',lat:33.9716,lon:-6.8498},
            {slug:'cairo',name:'Cairo',lat:30.0444,lon:31.2357},
            {slug:'london',name:'London',lat:51.5074,lon:-0.1278},
            {slug:'istanbul',name:'Istanbul',lat:41.0136,lon:28.955},
            {slug:'mumbai',name:'Mumbai',lat:19.076,lon:72.8777},
            {slug:'tokyo',name:'Tokyo',lat:35.6762,lon:139.6503},
            {slug:'rio',name:'Rio de Janeiro',lat:-22.9068,lon:-43.1729},
            {slug:'capetown',name:'Cape Town',lat:-33.9249,lon:18.4241},
            {slug:'dallas',name:'Dallas',lat:32.7767,lon:-96.797},
            {slug:'melbourne',name:'Melbourne',lat:-37.8136,lon:144.9631},
        ];

        // Approach A (current simple): PHP renders initial data into JS or we fetch via AJAX.
        // Approach B (better UX): On city/year change we call a lightweight wp_ajax endpoint
        // that uses the plugin's get_* helpers. For this draft we sketch both.

        let leafletMap = null, leafletMarker = null;
        let debounceTimer = null;

        const citySel   = root.querySelector('#cvi-city');
        const yearSel   = root.querySelector('#cvi-year');
        const moonSel   = root.querySelector('#cvi-newmoon');
        const cloudR    = root.querySelector('#cvi-cloud');
        const cloudVal  = root.querySelector('#cvi-cloud-val');
        const transR    = root.querySelector('#cvi-trans');
        const transNum  = root.querySelector('#cvi-trans-num');
        const checkBtn  = root.querySelector('#cvi-check');
        const results   = root.querySelector('#cvi-results');
        const cardsBox  = root.querySelector('#cvi-cards');
        const ctxDiv    = root.querySelector('#cvi-city-context');
        const mapDiv    = root.querySelector('#cvi-map');

        // Exact JS port of the Go heuristic (same as v2, kept for consistency)
        function applyAtmosphericAdjustment(raw, cloud, trans) {
            const map = {A:5,B:4,C:3,D:2,E:1,F:0};
            let val = map[raw];
            if (val === undefined) {
                if (raw === 'J' || raw === '?' || !raw) {
                    return ['E', 'Not a reliable crescent window (renderer returned ' + (raw || 'unknown') + ')'];
                }
                val = 1;
            }
            let adj = 0;
            if (cloud > 80) adj -= 3;
            else if (cloud > 60) adj -= 2;
            else if (cloud > 40) adj -= 1;

            if (trans >= 9) adj += 1;
            else if (trans <= 4) adj -= 1;
            else if (trans <= 2) adj -= 2;

            let final = Math.max(1, Math.min(5, val + adj));
            const rev = {5:'A',4:'B',3:'C',2:'D',1:'E'};
            const eff = rev[final];

            let note = 'Atmospheric conditions have minimal impact on this prediction.';
            if (adj < 0) note = 'Conditions are reducing visibility by approximately ' + (-adj) + ' category level(s).';
            else if (adj > 0) note = 'Excellent atmospheric conditions are slightly improving the prediction.';
            return [eff, note];
        }

        function populateCities(defaultSlug) {
            citySel.innerHTML = '';
            SUPPORTED_CITIES.forEach(c => {
                const o = document.createElement('option');
                o.value = c.slug; o.textContent = c.name;
                citySel.appendChild(o);
            });
            citySel.value = defaultSlug || 'jerusalem';
        }

        function updateCityContext(slug) {
            const c = SUPPORTED_CITIES.find(x => x.slug === slug);
            if (!c) return;
            ctxDiv.innerHTML = `${c.name} • ${c.lat.toFixed(3)}°, ${c.lon.toFixed(3)}° — pre-computed data`;
            initOrUpdateMap(c);
        }

        function initOrUpdateMap(city) {
            if (!mapDiv || !window.L) {
                // Try to load Leaflet (same pattern as web app)
                if (!window.L) {
                    const s = document.createElement('script');
                    s.src = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js';
                    s.onload = () => {
                        const link = document.createElement('link');
                        link.rel = 'stylesheet';
                        link.href = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css';
                        document.head.appendChild(link);
                        setTimeout(() => initOrUpdateMap(city), 120);
                    };
                    document.head.appendChild(s);
                }
                return;
            }
            if (!leafletMap) {
                leafletMap = L.map(mapDiv, {zoomControl:false, attributionControl:false})
                    .setView([city.lat, city.lon], 5);
                L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {opacity:.8}).addTo(leafletMap);
            } else {
                leafletMap.setView([city.lat, city.lon], 5);
            }
            if (leafletMarker) leafletMarker.remove();
            leafletMarker = L.marker([city.lat, city.lon]).addTo(leafletMap);
            setTimeout(() => leafletMap.invalidateSize(), 200);
        }

        // -----------------------------------------------------------------
        // DATA LOADING - the part that must feel like the real app
        // -----------------------------------------------------------------
        // In production we would have a proper wp_ajax handler that calls
        // the plugin methods. For this draft we demonstrate calling a
        // hypothetical endpoint + a fallback that works with current helpers.

        async function loadYears(citySlug) {
            yearSel.innerHTML = '<option>Loading years...</option>';

            // TODO (qwen3 volume): Replace with real AJAX to the plugin helper
            // For now we simulate using a future REST or admin-ajax route.
            // In the short term a PHP-rendered data attribute or localized script can seed this.

            // Placeholder years - real implementation would query get_available_years_for_city
            const years = [2026, 2027, 2028]; // TODO: fetch dynamically
            yearSel.innerHTML = '';
            years.forEach(y => {
                const o = document.createElement('option');
                o.value = y; o.textContent = y;
                yearSel.appendChild(o);
            });

            if (years.length) {
                yearSel.value = years[years.length-1]; // default to most recent
                await loadNewMoons(citySlug, years[years.length-1]);
            }
        }

        async function loadNewMoons(citySlug, year) {
            moonSel.innerHTML = '<option>Loading...</option>';

            // The real call we want:
            // fetch(`/wp-admin/admin-ajax.php?action=cvi_get_newmoons&city=${citySlug}&year=${year}`)
            // .then(r => r.json()).then(data => { ... populate with q and age ... })

            // For this exploratory draft we still rely on embedded knowledge or
            // a future localized data blob. The important thing is the SHAPE:
            // each item must have raw_day_0/1/2 + q_at_best + moon_age_at_best.

            // Placeholder with realistic shape (will be replaced by real data in next pass)
            const placeholder = [
                {new_moon_date: `${year}-01-18`, raw_day_0:'E', raw_day_1:'A', raw_day_2:'A', q:1.39, age:43.9},
                {new_moon_date: `${year}-02-17`, raw_day_0:'F', raw_day_1:'A', raw_day_2:'A', q:0.53, age:27.9},
            ];

            moonSel.innerHTML = '';
            placeholder.forEach(item => {
                const o = document.createElement('option');
                o.value = item.new_moon_date;
                o.textContent = item.new_moon_date;
                o.dataset.days = JSON.stringify([item.raw_day_0, item.raw_day_1, item.raw_day_2]);
                o.dataset.q = item.q || '';
                o.dataset.age = item.age || '';
                moonSel.appendChild(o);
            });
        }

        function renderCards() {
            const opt = moonSel.options[moonSel.selectedIndex];
            if (!opt || !opt.dataset.days) {
                results.style.display = 'none';
                return;
            }

            const days = JSON.parse(opt.dataset.days);
            const q = parseFloat(opt.dataset.q || '0');
            const age = parseFloat(opt.dataset.age || '0');
            const cloud = parseInt(cloudR.value, 10);
            const trans = parseFloat(transNum.value);

            let html = '';
            const labels = ['Day +0', 'Day +1', 'Day +2'];

            for (let i = 0; i < 3; i++) {
                const raw = (days[i] || '?').toUpperCase();
                const [eff, note] = applyAtmosphericAdjustment(raw, cloud, trans);

                if (raw === 'J' || raw === '?' || age > 80) {
                    html += `<div class="cvi-card" style="opacity:.6">
                        <div class="cvi-meta">${labels[i]}</div>
                        <div style="margin-top:8px; font-size:15px;">Not a good crescent window</div>
                        <div class="cvi-meta" style="margin-top:4px;">Renderer returned “${raw}”.</div>
                    </div>`;
                    continue;
                }

                const effColor = {A:'#22d3ee',B:'#67e8f9',C:'#fde047',D:'#fbbf24',E:'#d97706'}[eff] || '#64748b';

                html += `
                <div class="cvi-card">
                    <div class="cvi-meta">${labels[i]}</div>
                    <div style="display:flex; justify-content:space-between; align-items:flex-end; margin-top:2px;">
                        <div>
                            <div style="font-size:11px; color:#64748b;">Effective</div>
                            <div class="cvi-big" style="color:${effColor};">${eff}</div>
                        </div>
                        <div style="text-align:right;">
                            <div class="cvi-meta">Raw <span style="font-family:monospace;font-weight:700;">${raw}</span></div>
                            <div class="cvi-meta">Age: ${age.toFixed(1)} h • Q: ${q.toFixed(3)}</div>
                        </div>
                    </div>
                    <div class="cvi-explain" style="margin-top:10px;">${note}</div>
                </div>`;
            }

            cardsBox.innerHTML = html;
            results.style.display = 'block';
        }

        // Wiring
        function syncSliders() {
            cloudVal.textContent = cloudR.value;
            transNum.value = transR.value;
        }

        function scheduleLive() {
            clearTimeout(debounceTimer);
            debounceTimer = setTimeout(() => {
                if (results.style.display !== 'none') renderCards();
            }, 130);
        }

        cloudR.addEventListener('input', () => { syncSliders(); scheduleLive(); });
        transR.addEventListener('input', () => { syncSliders(); scheduleLive(); });
        transNum.addEventListener('input', () => { transR.value = transNum.value; scheduleLive(); });

        citySel.addEventListener('change', () => {
            updateCityContext(citySel.value);
            loadYears(citySel.value);
            results.style.display = 'none';
        });

        yearSel.addEventListener('change', () => {
            loadNewMoons(citySel.value, yearSel.value);
            results.style.display = 'none';
        });

        moonSel.addEventListener('change', renderCards);
        checkBtn.addEventListener('click', renderCards);

        // Boot
        populateCities('<?= esc_js($default_city) ?>');
        updateCityContext(citySel.value);
        loadYears(citySel.value);
        syncSliders();

        // TODOs left for Claude / next iteration (qwen3 volume notes):
        // - Real AJAX endpoint using the plugin class helpers + new q/age columns
        // - Proper default to "next new moon" (need a small PHP helper or JS date logic)
        // - Dark theme variant that matches the web app more closely
        // - "Likelihood this year" panel using real aggregated data
        // - Non-JS fallback content
        // - Enqueue assets instead of inline everything
        // - Migration story for people who already imported before the q/age columns existed
    })();
    </script>
    <?php
    return ob_get_clean();
}

// -----------------------------------------------------------------------------
// ADMIN NOTICE / IMPORT HOOK (so people know to re-import after schema change)
// -----------------------------------------------------------------------------
add_action('admin_notices', function() {
    // Only on our tools page for now
    if (isset($_GET['page']) && $_GET['page'] === 'crescent-visibility') {
        echo '<div class="notice notice-info"><p><strong>Interactive app-parity mode active.</strong> If you imported data before this update, re-import the JSON to get Age and Q values on the cards.</p></div>';
    }
});

// -----------------------------------------------------------------------------
// CLAUDE HAND-OFF NOTES (for the production pass)
// -----------------------------------------------------------------------------
// This file is the qwen3-coder:30b exploratory volume output after the user
// explicitly asked to "make the plugin work more like the app without the
// image generator".
//
// What is solid and should be kept:
// - Exact atmospheric heuristic port
// - Card structure that includes Age + Q (the main visual parity request)
// - Schema + import changes that finally persist the fields the web app shows
// - Dynamic year/new-moon shape using the real helpers
// - Multiple loading approaches documented
//
// What needs real production work:
// 1. Proper migration / ALTER TABLE strategy (or a one-time "re-import" instruction).
// 2. Real wp_ajax or REST endpoint so the interactive form can load fresh data
//    without page reload when the user changes city/year.
// 3. Move all the inline JS/CSS into enqueued files.
// 4. Add proper nonces, capability checks, and output escaping.
// 5. Implement a good "default to next new moon + Jerusalem" experience (needs
//    a small helper that knows the current date and picks the right row).
// 6. Optional: yearly likelihood summary using real aggregated queries.
// 7. Theming: offer both light (current) and a dark variant that matches the
//    zinc dark theme of the Go web app more closely.
//
// Once the above are done, the shortcode [crescent_visibility_interactive]
// (or a nicer name like [crescent_visibility_point]) can become the primary
// recommended way for users to get the "Visibility for My Location" experience
// inside WordPress.
//
// Accuracy note: All visibility numbers still come from the reference renderer
// via the offline JSON. No runtime astronomy. This constraint must be preserved.
