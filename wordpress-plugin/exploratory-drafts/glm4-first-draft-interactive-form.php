<?php
/**
 * EXPLORATORY FIRST DRAFT - glm-4.7-flash style
 *
 * This is a rough, high-volume exploratory pass at the full interactive
 * "Visibility for My Location" experience, powered by precomputed data.
 *
 * Goal: Get something that feels like the original web app's /point tool
 * as quickly as possible, using only imported data.
 *
 * NOT production ready. Expect TODOs, rough edges, and multiple approaches.
 */

if (!defined('ABSPATH')) exit;

// ------------------------------------------------------------------
// SHORTCODE
// ------------------------------------------------------------------
add_shortcode('crescent_visibility_full', 'crescent_visibility_full_shortcode');

function crescent_visibility_full_shortcode($atts) {
    $atts = shortcode_atts([
        'default_city' => 'jerusalem',
    ], $atts);

    ob_start();
    ?>
    <div id="crescent-full-interactive" class="crescent-full-interactive" style="max-width: 960px; font-family: system-ui, -apple-system, sans-serif;">

        <h2 style="margin-bottom: 12px;">Visibility for My Location</h2>

        <!-- City -->
        <div style="margin-bottom: 16px;">
            <label style="display:block; font-weight:600; margin-bottom:4px;">City</label>
            <select id="city" style="width:100%; padding:8px; font-size:15px;">
                <!-- Populated by JS from imported data -->
            </select>
            <div id="city-context" style="margin-top:6px; font-size:13px; color:#555;"></div>
        </div>

        <!-- Year + New Moon -->
        <div style="display:grid; grid-template-columns: 1fr 2fr; gap:12px; margin-bottom:16px;">
            <div>
                <label style="display:block; font-weight:600; margin-bottom:4px;">Year</label>
                <select id="year" style="width:100%; padding:8px; font-size:15px;">
                    <option value="">Select year...</option>
                </select>
            </div>
            <div>
                <label style="display:block; font-weight:600; margin-bottom:4px;">New Moon</label>
                <select id="new_moon" style="width:100%; padding:8px; font-size:15px;">
                    <option value="">Select new moon...</option>
                </select>
            </div>
        </div>

        <!-- Atmospheric Conditions -->
        <div style="margin-bottom: 18px; padding: 12px; background:#f8f9fa; border-radius:6px;">
            <div style="font-weight:600; margin-bottom:8px;">Atmospheric Conditions</div>

            <div style="margin-bottom:10px;">
                <label style="display:flex; justify-content:space-between; font-size:14px;">
                    <span>Cloud Cover</span>
                    <span><span id="cloud_val">20</span>%</span>
                </label>
                <input type="range" id="cloud" min="0" max="100" value="20" style="width:100%;">
            </div>

            <div>
                <label style="display:flex; justify-content:space-between; font-size:14px;">
                    <span>Transparency</span>
                    <span><span id="trans_val">7</span></span>
                </label>
                <input type="range" id="trans" min="1" max="10" value="7" style="width:100%;">
            </div>
        </div>

        <button id="check_btn" style="width:100%; padding:12px; background:#0d6efd; color:white; border:none; border-radius:6px; font-size:16px; cursor:pointer;">
            Check Visibility
        </button>

        <!-- Results -->
        <div id="results" style="margin-top:20px; display:none;">
            <h3 style="margin-bottom:12px;">Results for the 3 days after new moon</h3>
            <div id="cards" style="display:grid; gap:12px; grid-template-columns:repeat(auto-fit, minmax(260px, 1fr));"></div>
        </div>

    </div>

    <script>
    (function() {
        const container = document.getElementById('crescent-full-interactive');
        if (!container) return;

        const citySelect   = document.getElementById('city');
        const yearSelect   = document.getElementById('year');
        const newMoonSelect= document.getElementById('new_moon');
        const cloudSlider  = document.getElementById('cloud');
        const transSlider  = document.getElementById('trans');
        const cloudVal     = document.getElementById('cloud_val');
        const transVal     = document.getElementById('trans_val');
        const checkBtn     = document.getElementById('check_btn');
        const resultsDiv   = document.getElementById('results');
        const cardsContainer = document.getElementById('cards');
        const contextDiv   = document.getElementById('city-context');

        let currentNewMoons = []; // cache for current city+year

        // ----------------------------------------------------------
        // Very rough client-side atmospheric adjustment
        // (matches the PHP version we have)
        // ----------------------------------------------------------
        function adjustCategory(raw, cloud, trans) {
            const map = {A:5, B:4, C:3, D:2, E:1, F:0};
            let val = map[raw] || 1;
            let adj = 0;

            if (cloud > 80) adj -= 3;
            else if (cloud > 60) adj -= 2;
            else if (cloud > 40) adj -= 1;

            if (trans >= 9) adj += 1;
            else if (trans <= 4) adj -= 1;
            else if (trans <= 2) adj -= 2;

            const final = Math.max(1, Math.min(5, val + adj));
            const rev = {5:'A',4:'B',3:'C',2:'D',1:'E'};
            return rev[final];
        }

        function updateSliders() {
            cloudVal.textContent = cloudSlider.value;
            transVal.textContent = transSlider.value;
        }
        cloudSlider.addEventListener('input', updateSliders);
        transSlider.addEventListener('input', updateSliders);

        // ----------------------------------------------------------
        // Load cities from the database (via a lightweight AJAX endpoint we will add)
        // For this first draft we hardcode the known 13 cities.
        // TODO: Replace with real AJAX call to a new endpoint.
        // ----------------------------------------------------------
        const SUPPORTED_CITIES = [
            {slug: 'jerusalem', name: 'Jerusalem'},
            {slug: 'mecca', name: 'Mecca'},
            {slug: 'karachi', name: 'Karachi'},
            {slug: 'rabat', name: 'Rabat'},
            {slug: 'cairo', name: 'Cairo'},
            {slug: 'london', name: 'London'},
            {slug: 'istanbul', name: 'Istanbul'},
            {slug: 'mumbai', name: 'Mumbai'},
            {slug: 'tokyo', name: 'Tokyo'},
            {slug: 'rio', name: 'Rio de Janeiro'},
            {slug: 'capetown', name: 'Cape Town'},
            {slug: 'dallas', name: 'Dallas'},
            {slug: 'melbourne', name: 'Melbourne'},
        ];

        function populateCities(defaultSlug) {
            citySelect.innerHTML = '';
            SUPPORTED_CITIES.forEach(city => {
                const opt = document.createElement('option');
                opt.value = city.slug;
                opt.textContent = city.name;
                citySelect.appendChild(opt);
            });
            citySelect.value = defaultSlug;
        }

        function updateCityContext(slug) {
            const city = SUPPORTED_CITIES.find(c => c.slug === slug);
            if (!city) {
                contextDiv.innerHTML = '';
                return;
            }
            contextDiv.innerHTML = `<small>Pre-computed data available for full accuracy. Location: ${city.name}</small>`;
            // TODO: Add a tiny static map or Leaflet snippet here later
        }

        // ----------------------------------------------------------
        // Load years for selected city
        // TODO: Replace with real AJAX to get_available_years_for_city
        // ----------------------------------------------------------
        async function loadYears(citySlug) {
            yearSelect.innerHTML = '<option>Loading years...</option>';

            // For now we fake it with the years we actually have data for
            // In a real version this would come from the DB via AJAX
            const years = [2026, 2027, 2028];

            yearSelect.innerHTML = '';
            years.forEach(y => {
                const opt = document.createElement('option');
                opt.value = y;
                opt.textContent = y;
                yearSelect.appendChild(opt);
            });

            // Auto-load first year
            if (years.length > 0) {
                yearSelect.value = years[0];
                await loadNewMoons(citySlug, years[0]);
            }
        }

        // ----------------------------------------------------------
        // Load new moons for city + year
        // TODO: Replace with real AJAX to get_new_moons_for_city_and_year
        // ----------------------------------------------------------
        async function loadNewMoons(citySlug, year) {
            newMoonSelect.innerHTML = '<option>Loading new moons...</option>';

            // Fake data for this draft. Real version would fetch from DB.
            // Structure matches what our generator produces.
            const fakeNewMoons = [
                {new_moon_date: `${year}-01-18`, raw_day_0:'D', raw_day_1:'C', raw_day_2:'B', best_raw:'B', best_effective:'B'},
                {new_moon_date: `${year}-02-17`, raw_day_0:'E', raw_day_1:'D', raw_day_2:'C', best_raw:'C', best_effective:'C'},
                {new_moon_date: `${year}-03-18`, raw_day_0:'B', raw_day_1:'A', raw_day_2:'C', best_raw:'A', best_effective:'A'},
                {new_moon_date: `${year}-04-16`, raw_day_0:'C', raw_day_1:'B', raw_day_2:'B', best_raw:'B', best_effective:'B'},
            ];

            newMoonSelect.innerHTML = '';
            fakeNewMoons.forEach(m => {
                const opt = document.createElement('option');
                opt.value = m.new_moon_date;
                opt.textContent = m.new_moon_date;
                opt.dataset.raw = JSON.stringify(m);
                newMoonSelect.appendChild(opt);
            });

            currentNewMoons = fakeNewMoons;
        }

        function calculateAndShowResults() {
            const selectedOption = newMoonSelect.options[newMoonSelect.selectedIndex];
            if (!selectedOption || !selectedOption.dataset.raw) {
                alert('Please select a new moon first.');
                return;
            }

            const moonData = JSON.parse(selectedOption.dataset.raw);
            const cloud = parseInt(cloudSlider.value);
            const trans = parseFloat(transSlider.value);

            const rawDays = [moonData.raw_day_0, moonData.raw_day_1, moonData.raw_day_2];
            const adjusted = rawDays.map(day => adjustCategory(day, cloud, trans));

            // Build nice cards (trying to match original web app style)
            let html = '';
            const labels = ['Day +0', 'Day +1', 'Day +2'];

            for (let i = 0; i < 3; i++) {
                const raw = rawDays[i];
                const eff = adjusted[i];

                html += `
                    <div style="background:white; border:1px solid #dee2e6; border-radius:8px; padding:14px;">
                        <div style="font-size:13px; color:#6c757d; margin-bottom:4px;">${labels[i]}</div>
                        
                        <div style="display:flex; justify-content:space-between; align-items:baseline; margin-bottom:6px;">
                            <div>
                                <div style="font-size:11px; color:#6c757d;">Raw</div>
                                <div style="font-size:26px; font-weight:700; line-height:1;">${raw}</div>
                            </div>
                            <div style="text-align:right;">
                                <div style="font-size:11px; color:#6c757d;">Effective</div>
                                <div style="font-size:26px; font-weight:700; line-height:1; color:#0d6efd;">${eff}</div>
                            </div>
                        </div>

                        <div style="font-size:12px; background:#f8f9fa; padding:6px; border-radius:4px;">
                            ${getExplanation(eff)}
                        </div>
                    </div>
                `;
            }

            cardsContainer.innerHTML = html;
            results.style.display = 'block';
        }

        function getExplanation(cat) {
            const map = {
                'A': 'Excellent — easily visible to the naked eye.',
                'B': 'Good — visible naked eye under clear conditions.',
                'C': 'Moderate — visible but requires decent conditions.',
                'D': 'Difficult — usually needs binoculars or a telescope.',
                'E': 'Very difficult or not visible even with aid.'
            };
            return map[cat] || 'Conditions vary.';
        }

        // Event wiring
        citySelect.addEventListener('change', () => {
            loadYears(citySelect.value);
            results.style.display = 'none';
        });

        yearSelect.addEventListener('change', () => {
            loadNewMoons(citySelect.value, parseInt(yearSelect.value));
            results.style.display = 'none';
        });

        checkBtn.addEventListener('click', calculateAndShowResults);

        // Initial load
        populateCities('<?php echo esc_js($atts['default_city']); ?>');
        loadYears(citySelect.value);
        updateSliders(); // if we had the function here

        // TODO (glm-4 style notes):
        // - Replace fake data with real AJAX calls to the PHP helpers
        // - Add proper error handling and loading states
        // - Add a lightweight Leaflet map for context
        // - Consider live recalculation on slider move (debounced)
        // - Add support for custom lat/lon with reduced accuracy warning
        // - Make the result cards look even closer to the original web app
    })();
    </script>
    <?php
    return ob_get_clean();
}

// Very basic client-side adjustment (for the JS demo above)
function adjustCategory($raw, $cloud, $trans) {
    $map = ['A'=>5,'B'=>4,'C'=>3,'D'=>2,'E'=>1,'F'=>0];
    $val = $map[$raw] ?? 1;
    $adj = 0;

    if ($cloud > 80) $adj -= 3;
    elseif ($cloud > 60) $adj -= 2;
    elseif ($cloud > 40) $adj -= 1;

    if ($trans >= 9) $adj += 1;
    elseif ($trans <= 4) $adj -= 1;
    elseif ($trans <= 2) $adj -= 2;

    $final = max(1, min(5, $val + $adj));
    $rev = [5=>'A',4=>'B',3=>'C',2=>'D',1=>'E'];
    return $rev[$final];
}
