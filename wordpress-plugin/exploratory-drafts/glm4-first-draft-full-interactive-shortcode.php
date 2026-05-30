<?php
/**
 * EXPLORATORY / VOLUME DRAFT - glm-4.7-flash style
 *
 * Goal: One file that gives a reasonably complete "interactive Visibility for My Location"
 * experience similar to the original web app, but powered by precomputed data only.
 *
 * This is deliberately rough and exploratory. Multiple approaches are noted.
 * Use as raw material for Ollama iteration or to hand to Claude for cleanup.
 */

if (!defined('ABSPATH')) exit;

// ------------------------------------------------------------------
// SHORTCODE
// ------------------------------------------------------------------
add_shortcode('crescent_visibility_interactive', 'crescent_visibility_interactive_shortcode');

function crescent_visibility_interactive_shortcode($atts) {
    $atts = shortcode_atts([
        'default_city' => 'jerusalem',
    ], $atts);

    // For this draft we render a self-contained form + results area.
    // In a real version this would be cleaner (enqueue assets, separate templates, etc.)

    ob_start();
    ?>
    <div id="crescent-interactive-v1" class="crescent-interactive" style="max-width: 920px; font-family: system-ui, -apple-system, sans-serif;">

        <h2 style="margin-bottom: 16px;">Visibility for My Location (Pre-computed Data)</h2>

        <!-- City -->
        <div style="margin-bottom: 12px;">
            <label style="display:block; font-weight: 600; margin-bottom: 4px;">City</label>
            <select id="city" style="width: 100%; padding: 8px; font-size: 15px;">
                <!-- Populated by JS from the 13 cities we support -->
            </select>
            <div id="city-context" style="margin-top: 4px; font-size: 13px; color: #555;"></div>
        </div>

        <!-- Year + New Moon -->
        <div style="display: grid; grid-template-columns: 1fr 2fr; gap: 12px; margin-bottom: 12px;">
            <div>
                <label style="display:block; font-weight: 600; margin-bottom: 4px;">Year</label>
                <select id="year" style="width:100%; padding:8px; font-size:15px;">
                    <option value="">Select year...</option>
                </select>
            </div>
            <div>
                <label style="display:block; font-weight: 600; margin-bottom: 4px;">New Moon</label>
                <select id="new_moon" style="width:100%; padding:8px; font-size:15px;">
                    <option value="">Select new moon...</option>
                </select>
            </div>
        </div>

        <!-- Atmospheric Conditions -->
        <div style="margin-bottom: 16px; padding: 12px; background:#f8f9fa; border-radius:6px;">
            <div style="font-weight:600; margin-bottom:8px;">Atmospheric Conditions</div>

            <div style="margin-bottom:8px;">
                <label style="font-size:13px; display:flex; justify-content:space-between;">
                    <span>Cloud Cover</span> <span><span id="cloud_val">20</span>%</span>
                </label>
                <input type="range" id="cloud" min="0" max="100" value="20" style="width:100%;">
            </div>

            <div>
                <label style="font-size:13px; display:flex; justify-content:space-between;">
                    <span>Transparency (1–10)</span> <span><span id="trans_val">7</span></span>
                </label>
                <input type="range" id="trans" min="1" max="10" value="7" style="width:100%;">
            </div>
        </div>

        <button id="check" style="width:100%; padding:11px; background:#0d6efd; color:white; border:none; border-radius:6px; font-size:15px; cursor:pointer;">
            Check Visibility
        </button>

        <!-- Results -->
        <div id="results" style="margin-top: 18px; display:none;">
            <h3 style="margin-bottom:10px;">Results – 3 days after new moon</h3>
            <div id="cards" style="display:grid; gap:12px; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));"></div>
        </div>

    </div>

    <script>
    (function() {
        const root = document.getElementById('crescent-interactive-v1');
        if (!root) return;

        const citySel   = root.querySelector('#city');
        const yearSel   = root.querySelector('#year');
        const moonSel   = root.querySelector('#new_moon');
        const cloudS    = root.querySelector('#cloud');
        const transS    = root.querySelector('#trans');
        const cloudV    = root.querySelector('#cloud_val');
        const transV    = root.querySelector('#trans_val');
        const checkBtn  = root.querySelector('#check');
        const results   = root.querySelector('#results');
        const cardsBox  = root.querySelector('#cards');
        const context   = root.querySelector('#city-context');

        let currentMoons = [];

        // Very rough client-side version of the atmospheric heuristic
        function adjust(raw, cloud, trans) {
            const m = {A:5,B:4,C:3,D:2,E:1,F:0};
            let v = m[raw] || 1;
            let a = 0;
            if (cloud > 80) a -= 3;
            else if (cloud > 60) a -= 2;
            else if (cloud > 40) a -= 1;
            if (trans >= 9) a += 1;
            else if (trans <= 4) a -= 1;
            else if (trans <= 2) a -= 2;
            const f = Math.max(1, Math.min(5, v + a));
            return {5:'A',4:'B',3:'C',2:'D',1:'E'}[f];
        }

        function updateVals() {
            cloudV.textContent = cloudS.value;
            transV.textContent = transS.value;
        }
        cloudS.addEventListener('input', updateVals);
        transS.addEventListener('input', updateVals);

        // Hardcoded list for this exploratory draft (matches our 13 cities)
        const CITIES = [
            {slug:'jerusalem', name:'Jerusalem'},
            {slug:'mecca', name:'Mecca'},
            {slug:'karachi', name:'Karachi'},
            {slug:'rabat', name:'Rabat'},
            {slug:'cairo', name:'Cairo'},
            {slug:'london', name:'London'},
            {slug:'istanbul', name:'Istanbul'},
            {slug:'mumbai', name:'Mumbai'},
            {slug:'tokyo', name:'Tokyo'},
            {slug:'rio', name:'Rio de Janeiro'},
            {slug:'capetown', name:'Cape Town'},
            {slug:'dallas', name:'Dallas'},
            {slug:'melbourne', name:'Melbourne'}
        ];

        function populateCities(defaultSlug) {
            citySel.innerHTML = '';
            CITIES.forEach(c => {
                const o = document.createElement('option');
                o.value = c.slug;
                o.textContent = c.name;
                citySel.appendChild(o);
            });
            citySel.value = defaultSlug;
            updateCityContext(defaultSlug);
        }

        function updateCityContext(slug) {
            const c = CITIES.find(x => x.slug === slug);
            context.innerHTML = c ? `<small>Pre-computed data available for accurate results. Location: ${c.name}</small>` : '';
            // TODO: Add tiny static map or Leaflet here in a later pass
        }

        // ----------------------------------------------------------
        // Dynamic loading (Approach A for first draft - simple fetch)
        // In reality these would hit lightweight AJAX endpoints that call the PHP helpers
        // ----------------------------------------------------------
        async function loadYears(city) {
            yearSel.innerHTML = '<option>Loading...</option>';

            // TODO: Replace with real call to get_available_years_for_city
            // For now we hardcode the years we actually have data for
            const years = [2026, 2027, 2028];

            yearSel.innerHTML = '';
            years.forEach(y => {
                const o = document.createElement('option');
                o.value = y;
                o.textContent = y;
                yearSel.appendChild(o);
            });

            if (years.length) {
                yearSel.value = years[0];
                await loadNewMoons(city, years[0]);
            }
        }

        async function loadNewMoons(city, year) {
            moonSel.innerHTML = '<option>Loading new moons...</option>';

            // TODO: Real AJAX → get_new_moons_for_city_and_year(city, year)
            // Fake data for this exploratory draft
            const fake = [
                {new_moon_date: `${year}-01-18`, raw_day_0:'D', raw_day_1:'C', raw_day_2:'B', best_raw:'B', best_effective:'B'},
                {new_moon_date: `${year}-02-17`, raw_day_0:'E', raw_day_1:'D', raw_day_2:'C', best_raw:'C', best_effective:'C'},
                {new_moon_date: `${year}-03-18`, raw_day_0:'B', raw_day_1:'A', raw_day_2:'C', best_raw:'A', best_effective:'A'},
            ];

            moonSel.innerHTML = '';
            fake.forEach(m => {
                const o = document.createElement('option');
                o.value = m.new_moon_date;
                o.textContent = m.new_moon_date;
                o.dataset.data = JSON.stringify(m);
                moonSel.appendChild(o);
            });

            currentMoons = fake;
        }

        function showResults() {
            const opt = moonSel.options[moonSel.selectedIndex];
            if (!opt || !opt.dataset.data) {
                alert('Please select a new moon');
                return;
            }

            const data = JSON.parse(opt.dataset.data);
            const cloud = parseInt(cloudS.value);
            const trans = parseFloat(transS.value);

            const raws = [data.raw_day_0, data.raw_day_1, data.raw_day_2];
            const adjusted = raws.map(r => adjustCategory(r, cloud, trans));

            let html = '';
            const labels = ['Day +0', 'Day +1', 'Day +2'];
            for (let i = 0; i < 3; i++) {
                html += `
                    <div style="background:white;border:1px solid #dee2e6;border-radius:8px;padding:12px;">
                        <div style="font-size:12px;color:#666;margin-bottom:2px;">${labels[i]}</div>
                        <div style="font-size:12px;">Raw: <strong>${raws[i]}</strong></div>
                        <div style="font-size:26px;font-weight:700;line-height:1;margin:4px 0;">${adjusted[i]}</div>
                        <div style="font-size:12px;color:#495057;">
                            ${getExplanation(adjusted[i])}
                        </div>
                    </div>`;
            }
            cards.innerHTML = html;
            results.style.display = 'block';
        }

        function getExplanation(cat) {
            const map = {
                A: 'Excellent — easily visible to the naked eye.',
                B: 'Good — visible naked eye under clear conditions.',
                C: 'Moderate — visible but needs decent conditions.',
                D: 'Difficult — binoculars usually required.',
                E: 'Very difficult or not visible even with aid.'
            };
            return map[cat] || '';
        }

        // Event wiring
        citySel.addEventListener('change', () => {
            loadYears(citySel.value);
            results.style.display = 'none';
        });

        yearSel.addEventListener('change', () => {
            loadNewMoons(citySel.value, parseInt(yearSel.value));
            results.style.display = 'none';
        });

        checkBtn.addEventListener('click', showResults);

        // Initial bootstrap
        populateCities('<?php echo esc_js($atts['default_city']); ?>');
        loadYears(citySel.value);
        updateSliders(); // helper not shown here for brevity

        // TODO (glm-4 notes):
        // - Replace fake data with real AJAX to the PHP helpers we already have
        // - Add proper loading states and error handling
        // - Add a lightweight Leaflet map for the selected city
        // - Support "custom location" with a warning that accuracy drops
        // - Make the result cards look even closer to the original web app
        // - Consider live recalculation on slider move (debounced)
    })();
    </script>
    <?php
    return ob_get_clean();
}
