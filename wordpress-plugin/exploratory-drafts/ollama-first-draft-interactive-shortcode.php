<?php
/**
 * EXPLORATORY FIRST DRAFT - Ollama Style
 * 
 * This is a rough, volume-oriented first pass at the full interactive 
 * "Visibility for My Location" shortcode.
 * 
 * NOT production code. Lots of TODOs, multiple approaches noted,
 * and some simplifications for the first draft.
 * 
 * Goal: Feel as close as possible to the original web app's /point experience
 * using only imported precomputed data.
 */

if (!defined('ABSPATH')) exit;

// ============================================================
// SHORTCODE REGISTRATION (Exploratory)
// ============================================================
add_shortcode('crescent_visibility_interactive', 'crescent_visibility_interactive_shortcode');

function crescent_visibility_interactive_shortcode($atts) {
    $atts = shortcode_atts([
        'default_city' => 'jerusalem',
    ], $atts);

    // For the first draft, we render the form + a results container.
    // We pre-load some data for the default city to make it feel alive quickly.
    ob_start();
    ?>
    <div id="crescent-interactive" class="crescent-interactive" style="max-width: 900px; font-family: system-ui, sans-serif;">
        
        <h2 style="margin-bottom: 16px;">Visibility for My Location</h2>

        <!-- City Selection -->
        <div style="margin-bottom: 16px;">
            <label><strong>City</strong></label><br>
            <select id="city-select" style="width: 100%; padding: 8px; font-size: 16px;">
                <!-- Populated via JS from available data or hardcoded list for v1 -->
                <option value="jerusalem">Jerusalem</option>
                <option value="mecca">Mecca</option>
                <option value="karachi">Karachi</option>
                <option value="rabat">Rabat</option>
                <option value="cairo">Cairo</option>
                <option value="london">London</option>
                <option value="istanbul">Istanbul</option>
                <option value="mumbai">Mumbai</option>
                <option value="tokyo">Tokyo</option>
                <option value="rio">Rio de Janeiro</option>
                <option value="capetown">Cape Town</option>
                <option value="dallas">Dallas</option>
                <option value="melbourne">Melbourne</option>
            </select>
            <small style="color:#666;">(Currently limited to pre-computed cities for full accuracy)</small>
        </div>

        <!-- Simple Map Context (very basic for first draft) -->
        <div id="map-context" style="margin-bottom: 16px; padding: 10px; background:#f8f9fa; border-radius:6px; font-size:13px;">
            <!-- JS will update this with a simple note or static map idea -->
        </div>

        <!-- Year + New Moon -->
        <div style="display: grid; grid-template-columns: 1fr 2fr; gap: 12px; margin-bottom: 16px;">
            <div>
                <label><strong>Year</strong></label><br>
                <select id="year-select" style="width:100%; padding:8px;">
                    <!-- Populated by JS -->
                </select>
            </div>
            <div>
                <label><strong>New Moon</strong></label><br>
                <select id="newmoon-select" style="width:100%; padding:8px;">
                    <!-- Dynamically populated -->
                </select>
            </div>
        </div>

        <!-- Atmospheric Conditions (matching web app) -->
        <div style="margin-bottom: 16px; padding: 12px; background: #f8f9fa; border-radius: 6px;">
            <strong>Atmospheric Conditions</strong>
            
            <div style="margin-top: 8px;">
                <label>Cloud Cover: <span id="cloud-value">20</span>%</label>
                <input type="range" id="cloud-slider" min="0" max="100" value="20" style="width:100%;">
            </div>
            
            <div style="margin-top: 8px;">
                <label>Transparency (1–10): <span id="trans-value">7</span></label>
                <input type="range" id="trans-slider" min="1" max="10" value="7" style="width:100%;">
            </div>
        </div>

        <button id="check-visibility" style="width:100%; padding:12px; font-size:16px; background:#0d6efd; color:white; border:none; border-radius:6px; cursor:pointer;">
            Check Visibility
        </button>

        <!-- Results Area -->
        <div id="results-area" style="margin-top: 20px; display:none;">
            <h3 style="margin-bottom:12px;">Results</h3>
            <div id="result-cards"></div>
        </div>

    </div>

    <script>
    // ============================================================
    // EXPLORATORY JAVASCRIPT (Ollama-style first draft)
    // ============================================================
    (function() {
        const container = document.getElementById('crescent-interactive');
        if (!container) return;

        const citySelect   = document.getElementById('city-select');
        const yearSelect   = document.getElementById('year-select');
        const newMoonSelect= document.getElementById('newmoon-select');
        const cloudSlider  = document.getElementById('cloud-slider');
        const transSlider  = document.getElementById('trans-slider');
        const cloudValue   = document.getElementById('cloud-value');
        const transValue   = document.getElementById('trans-value');
        const checkBtn     = document.getElementById('check-visibility');
        const resultsArea  = document.getElementById('results-area');
        const resultCards  = document.getElementById('result-cards');
        const mapContext   = document.getElementById('map-context');

        // TODO: In a real version, load this from a localized script or AJAX
        // For this exploratory draft we hardcode the 13 cities
        const PRECOMPUTED_CITIES = [
            {slug:'jerusalem', name:'Jerusalem', lat:31.7683, lon:35.2137},
            {slug:'mecca', name:'Mecca', lat:21.3891, lon:39.8579},
            {slug:'karachi', name:'Karachi', lat:24.8607, lon:67.0011},
            {slug:'rabat', name:'Rabat', lat:33.9716, lon:-6.8498},
            {slug:'cairo', name:'Cairo', lat:30.0444, lon:31.2357},
            {slug:'london', name:'London', lat:51.5074, lon:-0.1278},
            {slug:'istanbul', name:'Istanbul', lat:41.0136, lon:28.9550},
            {slug:'mumbai', name:'Mumbai', lat:19.0760, lon:72.8777},
            {slug:'tokyo', name:'Tokyo', lat:35.6762, lon:139.6503},
            {slug:'rio', name:'Rio de Janeiro', lat:-22.9068, lon:-43.1729},
            {slug:'capetown', name:'Cape Town', lat:-33.9249, lon:18.4241},
            {slug:'dallas', name:'Dallas', lat:32.7767, lon:-96.7970},
            {slug:'melbourne', name:'Melbourne', lat:-37.8136, lon:144.9631}
        ];

        // Simple in-memory cache for this draft
        let currentData = {
            city: null,
            year: null,
            newMoons: []   // array of {new_moon_date, raw_day_0, raw_day_1, raw_day_2, best_raw, best_effective}
        };

        function updateCloudValue() {
            cloudValue.textContent = cloudSlider.value;
        }
        function updateTransValue() {
            transValue.textContent = transSlider.value;
        }

        cloudSlider.addEventListener('input', updateCloudValue);
        transSlider.addEventListener('input', updateTransValue);

        // Very rough client-side atmospheric adjustment (mirrors the PHP version)
        function adjustCategory(rawCat, cloud, trans) {
            const map = {A:5, B:4, C:3, D:2, E:1, F:0};
            let val = map[rawCat] || 1;
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

        // Load years for selected city (simulated from data we would fetch)
        async function loadYearsForCity(citySlug) {
            // In a real implementation this would be an AJAX call to
            // the PHP helper get_available_years_for_city()
            // For this first draft we fake it with a common range.
            yearSelect.innerHTML = '<option>Loading...</option>';

            // TODO: Replace with real AJAX to get_available_years_for_city
            const fakeYears = [2026,2027,2028]; // we only have this data so far

            yearSelect.innerHTML = '';
            fakeYears.forEach(y => {
                const opt = document.createElement('option');
                opt.value = y;
                opt.textContent = y;
                yearSelect.appendChild(opt);
            });

            // Auto-select first year and load new moons
            if (fakeYears.length > 0) {
                yearSelect.value = fakeYears[0];
                await loadNewMoons(citySlug, fakeYears[0]);
            }
        }

        async function loadNewMoons(citySlug, year) {
            newMoonSelect.innerHTML = '<option>Loading new moons...</option>';

            // TODO: Real AJAX call to get_new_moons_for_city_and_year
            // For now we simulate with the data we generated earlier
            // (In practice the server would return the actual rows from the DB)

            // Fake response for demo purposes
            const fakeNewMoons = [
                {new_moon_date: `${year}-01-18`, raw_day_0:'D', raw_day_1:'C', raw_day_2:'B', best_raw:'B', best_effective:'B'},
                {new_moon_date: `${year}-02-17`, raw_day_0:'E', raw_day_1:'D', raw_day_2:'C', best_raw:'C', best_effective:'C'},
                {new_moon_date: `${year}-03-18`, raw_day_0:'B', raw_day_1:'A', raw_day_2:'C', best_raw:'A', best_effective:'A'},
            ];

            newMoonSelect.innerHTML = '';
            fakeNewMoons.forEach(m => {
                const opt = document.createElement('option');
                opt.value = m.new_moon_date;
                opt.textContent = m.new_moon_date;
                opt.dataset.data = JSON.stringify(m); // store the raw data on the option
                newMoonSelect.appendChild(opt);
            });

            currentData.city = citySlug;
            currentData.year = year;
            currentData.newMoons = fakeNewMoons;
        }

        function updateMapContext(citySlug) {
            const city = PRECOMPUTED_CITIES.find(c => c.slug === citySlug);
            if (!city) {
                mapContext.innerHTML = 'Location context will appear here.';
                return;
            }
            mapContext.innerHTML = `
                <strong>${city.name}</strong> (${city.lat}, ${city.lon})<br>
                <small>Pre-computed data available for full accuracy.</small>
            `;
            // TODO: Add a tiny Leaflet map or static image here in a later pass
        }

        function showResults() {
            const selectedOption = newMoonSelect.options[newMoonSelect.selectedIndex];
            if (!selectedOption || !selectedOption.dataset.data) {
                alert('Please select a new moon.');
                return;
            }

            const moonData = JSON.parse(selectedOption.dataset.data);
            const cloud = parseInt(cloudSlider.value);
            const trans = parseFloat(transSlider.value);

            const days = [moonData.raw_day_0, moonData.raw_day_1, moonData.raw_day_2];
            const adjusted = days.map(d => adjustCategory(d, cloud, trans));

            let html = '<div style="display:grid; gap:12px; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));">';
            const labels = ['Day +0', 'Day +1', 'Day +2'];

            for (let i = 0; i < 3; i++) {
                html += `
                    <div style="background:white; padding:12px; border:1px solid #dee2e6; border-radius:6px;">
                        <div style="font-size:12px; color:#6c757d;">${labels[i]}</div>
                        <div style="font-size:11px; color:#6c757d;">Raw: <strong>${days[i]}</strong></div>
                        <div style="font-size:28px; font-weight:700; line-height:1; margin:4px 0;">${adjusted[i]}</div>
                        <div style="font-size:12px; color:#495057;">
                            ${getExplanation(adjusted[i])}
                        </div>
                    </div>
                `;
            }
            html += '</div>';

            resultCards.innerHTML = html;
            resultsArea.style.display = 'block';
        }

        function getExplanation(cat) {
            const map = {
                'A': 'Excellent — easily visible naked eye.',
                'B': 'Good — visible naked eye in clear conditions.',
                'C': 'Moderate — requires good conditions.',
                'D': 'Difficult — usually needs binoculars.',
                'E': 'Very difficult or not visible.'
            };
            return map[cat] || '';
        }

        // Event listeners
        citySelect.addEventListener('change', () => {
            const city = citySelect.value;
            updateMapContext(city);
            loadYearsForCity(city);
            resultsArea.style.display = 'none';
        });

        yearSelect.addEventListener('change', () => {
            const city = citySelect.value;
            const year = parseInt(yearSelect.value);
            loadNewMoons(city, year);
            resultsArea.style.display = 'none';
        });

        checkBtn.addEventListener('click', showResults);

        // Initial load
        citySelect.value = '<?php echo esc_js($atts['default_city']); ?>';
        updateMapContext(citySelect.value);
        loadYearsForCity(citySelect.value);
        updateCloudValue();
        updateTransValue();

        // TODO (Ollama note): 
        // - Replace fake data with real AJAX calls to get_available_years_for_city and get_new_moons_for_city_and_year
        // - Add proper error handling
        // - Consider live recalculation on slider move (debounced)
        // - Add a lightweight Leaflet map for the selected city
        // - Handle custom lat/lon with a warning that accuracy is reduced
    })();
    </script>
    <?php
    return ob_get_clean();
}

// Very rough client-side version of the adjustment (for live feel in the first draft)
function crescent_visibility_client_adjust($raw, $cloud, $trans) {
    // This is duplicated from the PHP version for the JS demo above.
    // In a real implementation we would keep the single source of truth in PHP
    // and call it via AJAX when needed.
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
