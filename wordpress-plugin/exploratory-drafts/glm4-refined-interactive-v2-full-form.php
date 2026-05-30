<?php
/**
 * EXPLORATORY VOLUME DRAFT v2 - glm-4.7-flash style (refined after first pass)
 *
 * "go" iteration following user selection of Option B + glm-4.7-flash:latest.
 *
 * Purpose: Deliver a single-file, high-volume, rough-but-functional sketch of the
 * FULL interactive "Visibility for My Location" experience that matches the
 * original Go web app /point tool — powered 100% by pre-computed MariaDB data.
 *
 * NO image generation, NO CGO, NO live astronomy at runtime. Pure PHP + vanilla JS.
 *
 * This is deliberately exploratory / Ollama-volume output:
 *   - Multiple data-loading approaches documented with code sketches
 *   - TODOs and rough edges everywhere (expected)
 *   - Two different card styles explored
 *   - Live vs. button-triggered recalculation both shown
 *   - "Likelihood" visualization options sketched
 *   - Exact heuristic from main.go: applyAtmosphericAdjustment
 *   - Handles real data oddities ("J" values, missing rows, etc.)
 *
 * Next consumer: hand the best parts to Claude for a clean, WP-standards-compliant
 * production version (enqueue, REST/ AJAX endpoints, separate JS, tests).
 *
 * How to use in current plugin (quick test):
 *   1. Include/require this file from crescent-visibility.php (dev only)
 *   2. The shortcode [crescent_visibility_interactive] or [crescent_visibility_full]
 *   3. For real data: import via Tools page first. For standalone demo this file
 *      contains a compact embedded sample of real 2026 observations for Jerusalem + Mecca.
 *
 * Data shape expected from DB (matches generator + real JSON):
 *   city, new_moon_date, year, raw_day_0, raw_day_1, raw_day_2,
 *   best_raw, best_effective, q_at_best, moon_age_at_best, data_version
 *
 * Cities (13, locked Phase 0):
 * Jerusalem, Mecca, Karachi, Rabat, Cairo, London, Istanbul, Mumbai, Tokyo,
 * Rio de Janeiro, Cape Town, Dallas, Melbourne.
 */

// Prevent direct access
if (!defined('ABSPATH')) exit;

// -----------------------------------------------------------------------------
// SHORTCODE REGISTRATION (exploratory — production will be cleaner)
// -----------------------------------------------------------------------------
add_shortcode('crescent_visibility_interactive', 'crescent_visibility_interactive_shortcode');
add_shortcode('crescent_visibility_full', 'crescent_visibility_interactive_shortcode'); // alias

/**
 * The main interactive shortcode.
 * Renders a complete self-contained form + live results area.
 */
function crescent_visibility_interactive_shortcode($atts) {
    $atts = shortcode_atts([
        'default_city' => 'jerusalem',
        'years'        => '2026-2028',   // future: allow "2006-2035" etc once more data imported
        'live'         => 'true',        // live slider updates (debounced)
        'show_likelihood' => 'true',
        'show_map'     => 'true',
    ], $atts);

    $default_city = sanitize_text_field($atts['default_city']);
    $year_range   = sanitize_text_field($atts['years']);
    $live_mode    = filter_var($atts['live'], FILTER_VALIDATE_BOOLEAN);
    $show_like    = filter_var($atts['show_likelihood'], FILTER_VALIDATE_BOOLEAN);
    $show_map     = filter_var($atts['show_map'], FILTER_VALIDATE_BOOLEAN);

    // -------------------------------------------------------------------------
    // EMBEDDED SAMPLE DATA (for instant demo / standalone testing of the JS)
    // In production this would come from the DB via the helper functions below.
    // This is a tiny slice of the real 2026-2028-real.json for Jerusalem + Mecca.
    // -------------------------------------------------------------------------
    $sample_data_json = json_encode([
        'jerusalem' => [
            2026 => [
                ['new_moon_date' => '2026-01-18', 'days' => ['J','E','A'], 'q' => 1.3937, 'age' => 43.88],
                ['new_moon_date' => '2026-02-17', 'days' => ['F','A','A'],  'q' => 0.5265, 'age' => 27.92],
                ['new_moon_date' => '2026-03-19', 'days' => ['E','A','A'],  'q' => 1.6498, 'age' => 39.19],
                ['new_moon_date' => '2026-04-17', 'days' => ['F','A','A'],  'q' => 0.9875, 'age' => 28.92],
                ['new_moon_date' => '2026-09-11', 'days' => ['F','B','A'],  'q' => 1.426,  'age' => 60.82],
            ],
            2027 => [
                ['new_moon_date' => '2027-01-07', 'days' => ['D','C','B'], 'q' => 0.8, 'age' => 32.1],
            ],
        ],
        'mecca' => [
            2026 => [
                ['new_moon_date' => '2026-01-18', 'days' => ['E','D','C'], 'q' => 1.1, 'age' => 41.2],
                ['new_moon_date' => '2026-02-17', 'days' => ['A','A','B'], 'q' => 0.4, 'age' => 25.5],
            ],
        ],
        // Add more cities/years as needed for demo volume — the real import has hundreds of rows
    ]);

    // Full city metadata (lat/lon) for map + context — matches the 13 locked cities
    $cities_json = json_encode([
        ['slug'=>'jerusalem','name'=>'Jerusalem','lat'=>31.7683,'lon'=>35.2137],
        ['slug'=>'mecca','name'=>'Mecca','lat'=>21.3891,'lon'=>39.8579],
        ['slug'=>'karachi','name'=>'Karachi','lat'=>24.8607,'lon'=>67.0011],
        ['slug'=>'rabat','name'=>'Rabat','lat'=>33.9716,'lon'=>-6.8498],
        ['slug'=>'cairo','name'=>'Cairo','lat'=>30.0444,'lon'=>31.2357],
        ['slug'=>'london','name'=>'London','lat'=>51.5074,'lon'=>-0.1278],
        ['slug'=>'istanbul','name'=>'Istanbul','lat'=>41.0136,'lon'=>28.955],
        ['slug'=>'mumbai','name'=>'Mumbai','lat'=>19.076,'lon'=>72.8777],
        ['slug'=>'tokyo','name'=>'Tokyo','lat'=>35.6762,'lon'=>139.6503],
        ['slug'=>'rio','name'=>'Rio de Janeiro','lat'=>-22.9068,'lon'=>-43.1729],
        ['slug'=>'capetown','name'=>'Cape Town','lat'=>-33.9249,'lon'=>18.4241],
        ['slug'=>'dallas','name'=>'Dallas','lat'=>32.7767,'lon'=>-96.797],
        ['slug'=>'melbourne','name'=>'Melbourne','lat'=>-37.8136,'lon'=>144.9631],
    ]);

    ob_start();
    ?>
    <div id="cvi-<?= esc_attr(uniqid()) ?>" class="crescent-interactive-v2"
         style="max-width:980px; font-family:system-ui,-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif; margin:20px 0; color:#111;">

        <style>
            /* Self-contained light styles — WP friendly, easy to theme later */
            .crescent-interactive-v2 { --cvi-accent:#0d6efd; }
            .cvi-card { background:#fff; border:1px solid #e5e7eb; border-radius:12px; padding:16px; box-shadow:0 1px 3px rgba(0,0,0,0.05); }
            .cvi-big { font-size:42px; font-weight:800; line-height:1; letter-spacing:-1px; }
            .cvi-raw { font-family:ui-monospace,SFMono-Regular,Menlo,monospace; font-size:15px; }
            .cvi-slider-wrap { background:#f8fafc; border:1px solid #e5e7eb; border-radius:10px; padding:14px; }
            .cvi-meta { font-size:12px; color:#475569; }
            .cvi-explain { font-size:13px; line-height:1.35; background:#f1f5f9; padding:8px 10px; border-radius:6px; }
            .cvi-day-label { font-size:12px; color:#64748b; font-weight:600; letter-spacing:0.5px; }
            .cvi-likely { font-size:13px; background:#f8fafc; border-radius:8px; padding:12px; }
            #cvi-map { height: 220px; border-radius: 10px; border:1px solid #cbd5e1; background:#f1f5f9; }
        </style>

        <h2 style="margin:0 0 4px; font-size:22px; font-weight:700;">Visibility for My Location</h2>
        <p style="margin:0 0 16px; color:#475569; font-size:14px;">
            Pre-computed Yallop data. Adjust conditions to see effective visibility.
            <span style="color:#64748b;">(Data imported via Tools → Crescent Visibility)</span>
        </p>

        <!-- CITY SELECTOR -->
        <div style="margin-bottom:14px;">
            <label style="display:block; font-weight:600; font-size:13px; margin-bottom:4px;">City</label>
            <select id="cvi-city" style="width:100%; padding:10px; font-size:15px; border:1px solid #cbd5e1; border-radius:8px; background:#fff;">
                <!-- Populated by JS from cities_json -->
            </select>
            <div id="cvi-city-context" class="cvi-meta" style="margin-top:4px;"></div>
        </div>

        <!-- YEAR + NEW MOON (the dynamic pair that powers the web-app UX) -->
        <div style="display:grid; grid-template-columns: 140px 1fr; gap:12px; margin-bottom:14px;">
            <div>
                <label style="display:block; font-weight:600; font-size:13px; margin-bottom:4px;">Year</label>
                <select id="cvi-year" style="width:100%; padding:10px; font-size:15px; border:1px solid #cbd5e1; border-radius:8px;">
                    <!-- Populated dynamically -->
                </select>
            </div>
            <div>
                <label style="display:block; font-weight:600; font-size:13px; margin-bottom:4px;">New Moon (3-day window)</label>
                <select id="cvi-newmoon" style="width:100%; padding:10px; font-size:15px; border:1px solid #cbd5e1; border-radius:8px;">
                    <option value="">Select a new moon…</option>
                </select>
            </div>
        </div>

        <!-- ATMOSPHERIC CONTROLS (exact same semantics as web app) -->
        <div class="cvi-slider-wrap" style="margin-bottom:16px;">
            <div style="font-weight:600; font-size:13px; margin-bottom:10px;">Atmospheric Conditions (affects Effective rating live)</div>

            <div style="display:grid; grid-template-columns:1fr 110px; gap:16px; align-items:end;">
                <div>
                    <div style="display:flex; justify-content:space-between; font-size:13px; margin-bottom:3px;">
                        <span>Cloud Cover</span>
                        <span><strong id="cvi-cloud-val">20</strong>%</span>
                    </div>
                    <input type="range" id="cvi-cloud" min="0" max="100" step="1" value="20"
                           style="width:100%; accent-color:#0d6efd;">
                    <div style="font-size:10px; color:#64748b;">0% = perfectly clear … 100% = overcast</div>
                </div>
                <div>
                    <div style="font-size:13px; margin-bottom:3px;">Transparency</div>
                    <div style="display:flex; gap:6px; align-items:center;">
                        <input type="range" id="cvi-trans" min="1" max="10" step="0.5" value="7" style="flex:1; accent-color:#0d6efd;">
                        <input type="number" id="cvi-trans-num" min="1" max="10" step="0.5" value="7"
                               style="width:58px; padding:4px 6px; font-size:13px; border:1px solid #cbd5e1; border-radius:4px;">
                    </div>
                </div>
            </div>
        </div>

        <div style="display:flex; gap:10px; margin-bottom:18px;">
            <button id="cvi-check" type="button"
                    style="flex:1; background:#0d6efd; color:#fff; border:none; border-radius:10px; padding:13px; font-size:15px; font-weight:600; cursor:pointer;">
                Check Visibility
            </button>
            <button id="cvi-reset" type="button"
                    style="background:#f1f5f9; color:#334155; border:1px solid #cbd5e1; border-radius:10px; padding:13px 18px; font-size:14px; cursor:pointer;">
                Reset
            </button>
        </div>

        <!-- RESULTS: 3-DAY CARDS (high fidelity match to web app) -->
        <div id="cvi-results" style="display:none; margin-bottom:20px;">
            <div style="font-weight:600; font-size:14px; margin-bottom:8px; color:#334155;">
                Results for the three evenings after conjunction
            </div>
            <div id="cvi-cards" style="display:grid; grid-template-columns:repeat(auto-fit, minmax(260px, 1fr)); gap:12px;"></div>
        </div>

        <!-- LIKELIHOOD SUMMARY (optional, one of several "likelihood" visualizations) -->
        <?php if ($show_like): ?>
        <div id="cvi-likelihood" class="cvi-likely" style="display:none; margin-bottom:16px;">
            <div style="font-weight:600; font-size:13px; margin-bottom:6px;">Likelihood this year (under current conditions)</div>
            <div id="cvi-like-content" style="font-size:13px; line-height:1.4;"></div>
            <div style="margin-top:6px; font-size:10px; color:#64748b;">
                Computed client-side from imported observations. Experimental — exact metric can be tuned later.
            </div>
        </div>
        <?php endif; ?>

        <!-- CONTEXT MAP (minimal, non-interactive pin for geographic sense) -->
        <?php if ($show_map): ?>
        <div style="margin-bottom:12px;">
            <div style="font-size:12px; font-weight:600; color:#475569; margin-bottom:4px;">Location context</div>
            <div id="cvi-map"></div>
            <div class="cvi-meta" style="margin-top:3px;">Map for reference only (Leaflet via CDN). Marker shows selected city.</div>
        </div>
        <?php endif; ?>

        <div style="font-size:11px; color:#64748b; border-top:1px solid #e5e7eb; padding-top:10px; margin-top:8px;">
            Yallop criterion (clear-sky baseline). Atmospheric adjustment applied in the browser.
            Data version shown after import. <a href="#" style="color:inherit;">Learn more</a>
        </div>

    </div>

    <script>
    (function() {
        const root = document.currentScript ? document.currentScript.parentElement : document.body;
        const container = root.querySelector('.crescent-interactive-v2') || document.getElementById('cvi-<?= esc_attr(uniqid()) ?>');
        if (!container) return;

        // ---------------------------------------------------------------------
        // CONFIG + DATA (glm-4 volume: two loading strategies sketched)
        // ---------------------------------------------------------------------
        const CITIES = <?= $cities_json ?>;
        const SAMPLE = <?= $sample_data_json ?>;   // embedded real-ish observations for demo

        // Approach A (simplest for first production cut):
        //   Every time city or year changes → fetch via a lightweight AJAX endpoint
        //   (wp_ajax_get_new_moons_for_city_year or a REST route).
        //
        // Approach B (faster after first load, good for small datasets):
        //   On shortcode mount, preload ALL observations for the default city (or all 13)
        //   into a JS cache, then filter client-side. Great for 2006-2035 range.
        //
        // For this exploratory draft we demonstrate BOTH with a flag.
        // Real implementation: the PHP side will provide the data via one of these.
        let DATA_LOADING_STRATEGY = 'preload-sample'; // 'ajax' | 'preload-sample' | 'preload-full'

        let currentCityData = {};   // will hold { year: [ {new_moon_date, days:[r0,r1,r2], q, age}, ... ] }
        let leafletMap = null;
        let leafletMarker = null;
        let debounceTimer = null;

        // Exact JavaScript port of main.go:applyAtmosphericAdjustment (Accuracy First)
        function applyAtmosphericAdjustment(rawCategory, cloudPercent, transparency) {
            const map = { 'A':5, 'B':4, 'C':3, 'D':2, 'E':1, 'F':0 };
            let val = map[rawCategory];
            if (val === undefined) {
                // "J" and other odd output from renderer → treat as very poor (like Go code does)
                if (rawCategory === 'J' || rawCategory === '?' || !rawCategory) {
                    return ['E', 'Not a reliable crescent window (renderer returned ' + (rawCategory || 'unknown') + ')'];
                }
                val = 1;
            }

            let adj = 0;
            if (cloudPercent > 80) adj -= 3;
            else if (cloudPercent > 60) adj -= 2;
            else if (cloudPercent > 40) adj -= 1;

            if (transparency >= 9) adj += 1;
            else if (transparency <= 4) adj -= 1;
            else if (transparency <= 2) adj -= 2;

            let final = val + adj;
            if (final < 1) final = 1;
            if (final > 5) final = 5;

            const rev = {5:'A',4:'B',3:'C',2:'D',1:'E'};
            const effective = rev[final];

            let note = 'Atmospheric conditions have minimal impact on this prediction.';
            if (adj < 0) {
                note = 'Conditions are reducing visibility by approximately ' + (-adj) + ' category level(s).';
            } else if (adj > 0) {
                note = 'Excellent atmospheric conditions are slightly improving the prediction.';
            }
            return [effective, note];
        }

        // ---------------------------------------------------------------------
        // DOM references
        // ---------------------------------------------------------------------
        const citySel   = container.querySelector('#cvi-city');
        const yearSel   = container.querySelector('#cvi-year');
        const moonSel   = container.querySelector('#cvi-newmoon');
        const cloudR    = container.querySelector('#cvi-cloud');
        const cloudVal  = container.querySelector('#cvi-cloud-val');
        const transR    = container.querySelector('#cvi-trans');
        const transNum  = container.querySelector('#cvi-trans-num');
        const checkBtn  = container.querySelector('#cvi-check');
        const resetBtn  = container.querySelector('#cvi-reset');
        const results   = container.querySelector('#cvi-results');
        const cardsBox  = container.querySelector('#cvi-cards');
        const ctxDiv    = container.querySelector('#cvi-city-context');
        const likeBox   = container.querySelector('#cvi-likelihood');
        const likeContent = container.querySelector('#cvi-like-content');
        const mapDiv    = container.querySelector('#cvi-map');

        // ---------------------------------------------------------------------
        // Helper: populate cities (static list from Phase 0 lock)
        // ---------------------------------------------------------------------
        function populateCities(defaultSlug) {
            citySel.innerHTML = '';
            CITIES.forEach(c => {
                const o = document.createElement('option');
                o.value = c.slug;
                o.textContent = c.name;
                citySel.appendChild(o);
            });
            citySel.value = defaultSlug || 'jerusalem';
        }

        function updateCityContext(slug) {
            const c = CITIES.find(x => x.slug === slug);
            if (!c) { ctxDiv.innerHTML = ''; return; }
            ctxDiv.innerHTML = `${c.name} • ${c.lat.toFixed(4)}°, ${c.lon.toFixed(4)}° — pre-computed data`;
            updateContextMap(c);
        }

        // ---------------------------------------------------------------------
        // Very small Leaflet context map (matches web-app intent, minimal footprint)
        // ---------------------------------------------------------------------
        function ensureLeaflet(cb) {
            if (window.L) { cb(); return; }
            // In real plugin we would enqueue properly; here we tolerate CDN for the draft
            const s = document.createElement('script');
            s.src = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js';
            s.onload = () => {
                const link = document.createElement('link');
                link.rel = 'stylesheet';
                link.href = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css';
                document.head.appendChild(link);
                cb();
            };
            s.onerror = () => console.warn('Leaflet CDN failed — map disabled in this draft');
            document.head.appendChild(s);
        }

        function updateContextMap(city) {
            if (!mapDiv || !city) return;
            ensureLeaflet(() => {
                if (!window.L) return;
                if (!leafletMap) {
                    leafletMap = L.map(mapDiv, { zoomControl: false, attributionControl: false }).setView([city.lat, city.lon], 5);
                    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', { opacity: 0.85 }).addTo(leafletMap);
                } else {
                    leafletMap.setView([city.lat, city.lon], 5);
                }
                if (leafletMarker) leafletMarker.remove();
                leafletMarker = L.marker([city.lat, city.lon]).addTo(leafletMap);
                // Give the map time to size correctly inside the WP page
                setTimeout(() => leafletMap.invalidateSize(), 180);
            });
        }

        // ---------------------------------------------------------------------
        // DATA LOADING — Approach sketches (glm-4 exploratory)
        // ---------------------------------------------------------------------

        // Real helper that would live in PHP (see renderer-improved etc.)
        // For now we simulate from the embedded SAMPLE or later from AJAX.
        function getObservationsForCityYear(citySlug, year) {
            if (DATA_LOADING_STRATEGY === 'preload-sample') {
                const byCity = SAMPLE[citySlug] || {};
                return byCity[year] || [];
            }
            // TODO Approach A: real AJAX
            // return fetch(`/wp-admin/admin-ajax.php?action=cvi_get_newmoons&city=${citySlug}&year=${year}`)
            //            .then(r => r.json()).then(d => d.observations || []);
            return [];
        }

        async function loadYearsForCity(citySlug) {
            yearSel.innerHTML = '<option>Loading…</option>';
            // In a full preload we would already know all years.
            // For demo we hardcode the range we have data for (easy to make dynamic later).
            const years = [2026, 2027, 2028];   // TODO: compute from imported data or meta table
            yearSel.innerHTML = '';
            years.forEach(y => {
                const o = document.createElement('option');
                o.value = y;
                o.textContent = y;
                yearSel.appendChild(o);
            });
            if (years.length) {
                yearSel.value = years[0];
                await loadNewMoons(citySlug, years[0]);
            }
        }

        async function loadNewMoons(citySlug, year) {
            moonSel.innerHTML = '<option>Loading new moons…</option>';
            const obs = getObservationsForCityYear(citySlug, parseInt(year, 10));

            moonSel.innerHTML = '';
            if (!obs.length) {
                const o = document.createElement('option');
                o.textContent = 'No data for this city/year (import more?)';
                o.disabled = true;
                moonSel.appendChild(o);
                return;
            }

            obs.forEach(item => {
                const o = document.createElement('option');
                o.value = item.new_moon_date;
                o.textContent = item.new_moon_date;
                // Store the raw triple + q/age on the option for fast access
                o.dataset.days = JSON.stringify(item.days || []);
                o.dataset.q = item.q || '';
                o.dataset.age = item.age || '';
                moonSel.appendChild(o);
            });

            // Auto-select first
            if (obs.length) {
                moonSel.selectedIndex = 0;
                if (results.style.display !== 'none') {
                    renderResultsLive(); // if already showing, refresh
                }
            }
        }

        // ---------------------------------------------------------------------
        // RENDERING — high-fidelity 3-day cards (web-app parity target)
        // ---------------------------------------------------------------------
        function renderResultsLive() {
            const opt = moonSel.options[moonSel.selectedIndex];
            if (!opt || !opt.dataset.days) {
                results.style.display = 'none';
                return;
            }

            const days = JSON.parse(opt.dataset.days || '["?","?","?"]');
            const age = parseFloat(opt.dataset.age || '0');
            const q   = parseFloat(opt.dataset.q || '0');

            const cloud = parseInt(cloudR.value, 10);
            const trans = parseFloat(transNum.value);

            const labels = ['Day +0 (evening of conjunction +0)', 'Day +1', 'Day +2'];
            let html = '';

            for (let i = 0; i < 3; i++) {
                const raw = (days[i] || '?').toUpperCase();
                const [eff, note] = applyAtmosphericAdjustment(raw, cloud, trans);

                // Color classes (close to web-app Tailwind mapping)
                const effColor = {A:'#22d3ee', B:'#67e8f9', C:'#fde047', D:'#fbbf24', E:'#d97706'}[eff] || '#64748b';

                // Graceful handling for "J" / bad windows (exact same logic as Go renderer)
                if (raw === 'J' || raw === '?' || age > 80) {
                    html += `
                    <div class="cvi-card" style="opacity:0.65;">
                        <div class="cvi-day-label">${labels[i]}</div>
                        <div style="margin-top:8px; font-size:15px; color:#64748b;">Not a good crescent window</div>
                        <div class="cvi-meta" style="margin-top:4px;">Renderer returned “${raw}” — date too far from true new moon.</div>
                    </div>`;
                    continue;
                }

                html += `
                <div class="cvi-card">
                    <div class="cvi-day-label">${labels[i]}</div>

                    <div style="display:flex; justify-content:space-between; align-items:flex-end; margin-top:4px;">
                        <div>
                            <div style="font-size:11px; color:#64748b;">Effective</div>
                            <div class="cvi-big" style="color:${effColor};">${eff}</div>
                        </div>
                        <div style="text-align:right;">
                            <div class="cvi-meta">Raw <span class="cvi-raw" style="font-weight:700;">${raw}</span></div>
                            <div class="cvi-meta">Age: ${age.toFixed(1)} h • Q: ${q.toFixed(3)}</div>
                        </div>
                    </div>

                    <div class="cvi-explain" style="margin-top:10px;">
                        ${note}
                    </div>
                </div>`;
            }

            cardsBox.innerHTML = html;
            results.style.display = 'block';

            // Optional live likelihood (cheap client-side aggregation)
            if (likeBox) updateLikelihood(cloud, trans);
        }

        function updateLikelihood(cloud, trans) {
            if (!likeContent) return;
            // Very rough "likelihood" metric for the chosen year:
            // % of new moons where at least one of the three days reaches B or better under current conditions.
            const year = parseInt(yearSel.value, 10);
            const city = citySel.value;
            const obsList = (SAMPLE[city] && SAMPLE[city][year]) || [];

            if (!obsList.length) {
                likeContent.textContent = 'No data loaded for likelihood calculation.';
                return;
            }

            let good = 0;
            obsList.forEach(item => {
                const ds = item.days || [];
                for (let i=0; i<3; i++) {
                    const [eff] = applyAtmosphericAdjustment(ds[i], cloud, trans);
                    if (eff === 'A' || eff === 'B') { good++; break; }
                }
            });

            const pct = Math.round((good / obsList.length) * 100);
            likeContent.innerHTML = `
                <strong>${pct}%</strong> of new moon windows this year have at least one evening rated <strong>B or better</strong>
                under the current atmospheric settings.
                <span style="color:#64748b;">(${good}/${obsList.length} windows)</span>
            `;
            likeBox.style.display = 'block';
        }

        // ---------------------------------------------------------------------
        // SLIDER SYNC + LIVE MODE
        // ---------------------------------------------------------------------
        function syncSliders(from) {
            if (from === 'range') {
                transNum.value = transR.value;
            } else {
                transR.value = transNum.value;
            }
            cloudVal.textContent = cloudR.value;
        }

        function scheduleLiveUpdate() {
            if (!<?= $live_mode ? 'true' : 'false' ?>) return;
            clearTimeout(debounceTimer);
            debounceTimer = setTimeout(() => {
                if (results.style.display !== 'none') renderResultsLive();
            }, 140);
        }

        // Wire everything
        cloudR.addEventListener('input', () => { syncSliders('range'); scheduleLiveUpdate(); });
        transR.addEventListener('input', () => { syncSliders('range'); scheduleLiveUpdate(); });
        transNum.addEventListener('input', () => { syncSliders('num'); scheduleLiveUpdate(); });

        citySel.addEventListener('change', async () => {
            updateCityContext(citySel.value);
            await loadYearsForCity(citySel.value);
            results.style.display = 'none';
            if (likeBox) likeBox.style.display = 'none';
        });

        yearSel.addEventListener('change', async () => {
            await loadNewMoons(citySel.value, yearSel.value);
            results.style.display = 'none';
            if (likeBox) likeBox.style.display = 'none';
        });

        moonSel.addEventListener('change', () => {
            results.style.display = 'none';
        });

        checkBtn.addEventListener('click', () => {
            renderResultsLive();
        });

        resetBtn.addEventListener('click', () => {
            cloudR.value = 20;
            transR.value = 7;
            transNum.value = 7;
            cloudVal.textContent = '20';
            results.style.display = 'none';
            if (likeBox) likeBox.style.display = 'none';
        });

        // ---------------------------------------------------------------------
        // BOOTSTRAP
        // ---------------------------------------------------------------------
        populateCities('<?= esc_js($default_city) ?>');
        updateCityContext(citySel.value);

        // Initial years + moons (simulated preload)
        loadYearsForCity(citySel.value).then(() => {
            // If live mode, we can optionally show the first result immediately
            // renderResultsLive();   // commented — user usually wants to adjust first
        });

        // Keyboard niceties
        container.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && document.activeElement.tagName === 'SELECT') {
                renderResultsLive();
            }
        });

        // Public debug hook (glm-4 volume style)
        window.CVI_DEBUG = { container, SAMPLE, applyAtmosphericAdjustment };

        console.log('%c[CVI exploratory v2] Interactive form mounted. Strategy: ' + DATA_LOADING_STRATEGY, 'color:#64748b');
    })();
    </script>
    <?php
    $html = ob_get_clean();

    // -------------------------------------------------------------------------
    // PHP-side data helpers (stubs for the real implementation)
    // These would live in renderer.php or a dedicated includes/data.php in prod.
    // They are here for completeness of the exploratory draft.
    // -------------------------------------------------------------------------
    /*
    function cvi_get_available_cities() { ... query wp_crescent_cities ... }
    function cvi_get_years_for_city($slug) { ... distinct year ... }
    function cvi_get_observations($city, $new_moon_date) {
        // returns array with raw_day_0/1/2, q_at_best, moon_age_at_best
    }
    */

    // For the test harness + future AJAX we also expose a tiny action sketch
    /*
    add_action('wp_ajax_cvi_get_newmoons', 'cvi_ajax_get_newmoons');
    add_action('wp_ajax_nopriv_cvi_get_newmoons', 'cvi_ajax_get_newmoons');
    function cvi_ajax_get_newmoons() {
        // nonce check, sanitize, query DB, wp_send_json
    }
    */

    return $html;
}

// -----------------------------------------------------------------------------
// EXACT PHP PORT OF THE HEURISTIC (for any server-side shortcode variants)
// -----------------------------------------------------------------------------
if (!function_exists('cvi_apply_atmospheric_adjustment')) {
    function cvi_apply_atmospheric_adjustment($raw, $cloud, $trans) {
        $map = ['A'=>5,'B'=>4,'C'=>3,'D'=>2,'E'=>1,'F'=>0];
        $val = $map[$raw] ?? 1;
        if ($raw === 'J' || $raw === '?' || !$raw) {
            return ['E', 'Not a reliable crescent window (renderer returned ' . ($raw ?: 'unknown') . ')'];
        }
        $adj = 0;
        if ($cloud > 80) $adj -= 3;
        elseif ($cloud > 60) $adj -= 2;
        elseif ($cloud > 40) $adj -= 1;

        if ($trans >= 9) $adj += 1;
        elseif ($trans <= 4) $adj -= 1;
        elseif ($trans <= 2) $adj -= 2;

        $final = max(1, min(5, $val + $adj));
        $rev = [5=>'A',4=>'B',3=>'C',2=>'D',1=>'E'];
        $eff = $rev[$final];

        if ($adj === 0) $note = 'Atmospheric conditions have minimal impact on this prediction.';
        elseif ($adj < 0) $note = 'Conditions are reducing visibility by approximately ' . (-$adj) . ' category level(s).';
        else $note = 'Excellent atmospheric conditions are slightly improving the prediction.';
        return [$eff, $note];
    }
}
