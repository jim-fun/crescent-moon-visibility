/**
 * EXPLORATORY FIRST DRAFT - glm-4.7-flash style
 *
 * Dynamic behavior for the interactive "Visibility for My Location" form.
 * This is rough, volume-oriented code meant for iteration.
 *
 * Two approaches are sketched:
 *   A) Simple fetch-every-time (easiest to start with)
 *   B) Preload data for the city then filter client-side (faster after first load)
 *
 * Also includes a JavaScript version of the atmospheric adjustment heuristic.
 */

// ============================================================
// 1. JavaScript version of the atmospheric adjustment (mirrors PHP)
// ============================================================
function adjustCategory(raw, cloudPercent, transparency) {
    const map = { A: 5, B: 4, C: 3, D: 2, E: 1, F: 0 };
    let val = map[raw] || 1;
    let adj = 0;

    if (cloudPercent > 80) adj -= 3;
    else if (cloudPercent > 60) adj -= 2;
    else if (cloudPercent > 40) adj -= 1;

    if (transparency >= 9) adj += 1;
    else if (transparency <= 4) adj -= 1;
    else if (transparency <= 2) adj -= 2;

    const final = Math.max(1, Math.min(5, val + adj));
    const reverse = { 5: 'A', 4: 'B', 3: 'C', 2: 'D', 1: 'E' };
    return reverse[final];
}

// ============================================================
// 2. Approach A: Simple fetch on every change (recommended for first version)
// ============================================================
async function loadNewMoonsSimple(citySlug, year) {
    // TODO: Replace with real AJAX endpoint that calls get_new_moons_for_city_and_year
    const res = await fetch(`/wp-admin/admin-ajax.php?action=get_new_moons&city=${citySlug}&year=${year}`);
    if (!res.ok) throw new Error('Failed to load new moons');

    const data = await res.json();
    return data.new_moons || [];
}

// ============================================================
// 3. Approach B: Preload all data for a city (more advanced)
// ============================================================
let cityDataCache = {};

async function preloadCityData(citySlug) {
    if (cityDataCache[citySlug]) return cityDataCache[citySlug];

    // TODO: Create a proper endpoint that returns all years + new moons for a city
    const res = await fetch(`/wp-admin/admin-ajax.php?action=get_city_visibility_data&city=${citySlug}`);
    const data = await res.json();

    cityDataCache[citySlug] = data;
    return data;
}

function getNewMoonsFromCache(citySlug, year) {
    const cityData = cityDataCache[citySlug];
    if (!cityData) return [];

    return cityData[year] || [];
}

// ============================================================
// 4. Live atmospheric adjustment + result rendering (core UX)
// ============================================================
function recalculateAndRenderResults(selectedMoonData, cloud, trans, container) {
    if (!selectedMoonData) return;

    const rawDays = [
        selectedMoonData.raw_day_0,
        selectedMoonData.raw_day_1,
        selectedMoonData.raw_day_2
    ];

    const adjusted = rawDays.map(day => adjustCategory(day, cloud, trans));

    // Very basic card rendering (will be replaced by nicer component later)
    let html = '<div style="display: grid; gap: 12px; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));">';

    const labels = ['Day +0', 'Day +1', 'Day +2'];
    for (let i = 0; i < 3; i++) {
        html += `
            <div style="background:white; border:1px solid #ddd; border-radius:8px; padding:12px;">
                <div style="font-size:12px; color:#666;">${labels[i]}</div>
                <div style="font-size:13px;">Raw: <strong>${rawDays[i]}</strong></div>
                <div style="font-size:26px; font-weight:700; line-height:1; margin:4px 0;">${adjusted[i]}</div>
                <div style="font-size:12px; color:#555;">
                    ${getSimpleExplanation(adjusted[i])}
                </div>
            </div>
        `;
    }
    html += '</div>';

    container.innerHTML = html;
}

function getSimpleExplanation(cat) {
    const map = {
        A: 'Excellent — easily visible naked eye.',
        B: 'Good — visible under clear conditions.',
        C: 'Moderate — visible but needs decent skies.',
        D: 'Difficult — binoculars usually required.',
        E: 'Very difficult or not visible.'
    };
    return map[cat] || '';
}

// ============================================================
// 5. Main controller (rough wiring example)
// ============================================================
function initInteractiveVisibility(containerId) {
    const root = document.getElementById(containerId);
    if (!root) return;

    // TODO: Build the actual form elements here or assume they exist with these IDs
    const citySelect   = root.querySelector('#city');
    const yearSelect   = root.querySelector('#year');
    const moonSelect   = root.querySelector('#new_moon');
    const cloudSlider  = root.querySelector('#cloud');
    const transSlider  = root.querySelector('#trans');
    const resultsBox   = root.querySelector('#results');

    let currentMoonData = null;

    function updateResults() {
        if (!currentMoonData || !cloudSlider || !transSlider) return;

        const cloud = parseInt(cloudSlider.value);
        const trans = parseFloat(transSlider.value);

        recalculateAndRenderResults(currentMoonData, cloud, trans, resultsBox);
    }

    // When city or year changes → reload new moons
    async function refreshNewMoons() {
        const city = citySelect.value;
        const year = yearSelect.value;
        if (!city || !year) return;

        moonSelect.innerHTML = '<option>Loading...</option>';

        // Using Approach A for simplicity in this first draft
        const newMoons = await loadNewMoonsSimple(city, parseInt(year));

        moonSelect.innerHTML = '';
        newMoons.forEach(m => {
            const opt = document.createElement('option');
            opt.value = m.new_moon_date;
            opt.textContent = m.new_moon_date;
            opt.dataset.data = JSON.stringify(m);
            moonSelect.appendChild(opt);
        });

        if (newMoons.length > 0) {
            moonSelect.value = newMoons[0].new_moon_date;
            currentMoonData = newMoons[0];
            updateResults();
        }
    }

    // Wire events
    citySelect.addEventListener('change', refreshNewMoons);
    yearSelect.addEventListener('change', refreshNewMoons);

    moonSelect.addEventListener('change', () => {
        const opt = moonSelect.options[moonSelect.selectedIndex];
        currentMoonData = opt ? JSON.parse(opt.dataset.data) : null;
        updateResults();
    });

    if (cloudSlider) cloudSlider.addEventListener('input', updateResults);
    if (transSlider) transSlider.addEventListener('input', updateResults);

    // Initial load
    // In real usage we would trigger this after the form is ready
    // refreshNewMoons();
}

// Example usage in a shortcode:
// initInteractiveVisibility('crescent-full-interactive');

console.log('[glm4-draft] Exploratory dynamic JS module loaded. Replace loadNewMoonsSimple with real AJAX.');