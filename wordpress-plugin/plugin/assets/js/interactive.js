/**
 * Crescent Visibility — interactive "Visibility for My Location" front end.
 *
 * Powers the [crescent_visibility_interactive] shortcode:
 *   - Loads the city list, years, and new moons from the nonce-protected
 *     admin-ajax endpoints (with in-flight request cancellation + caching).
 *   - Re-grades pre-computed categories live as the atmospheric sliders move,
 *     using an exact JS port of main.go:applyAtmosphericAdjustment.
 *   - Renders three rich result cards matching the web app, plus a small
 *     non-interactive Leaflet context map (Leaflet is bundled with the plugin).
 *
 * Accuracy First: no astronomy happens here. Categories come from the
 * pre-computed data; the heuristic only adjusts for weather, identically to
 * the Go web app.
 */
(function () {
    'use strict';

    var data = window.cviInteractiveData || {};
    var root = document.querySelector('.cvi-interactive-root');
    if (!root) {
        return;
    }

    // i18n via @wordpress/i18n (enqueued as a dependency). Falls back to the
    // English source string if wp.i18n is unavailable.
    var wpI18n = (window.wp && window.wp.i18n) ? window.wp.i18n : null;
    function __(s) { return wpI18n ? wpI18n.__(s, 'crescent-visibility') : s; }
    function _sprintf(fmt, a, b) {
        if (wpI18n && wpI18n.sprintf) { return wpI18n.sprintf(fmt, a, b); }
        return fmt.replace('%1$s', a).replace('%2$s', b).replace('%d', a);
    }

    var citySel  = document.getElementById('cvi-city');
    var yearSel   = document.getElementById('cvi-year');
    var moonSel   = document.getElementById('cvi-newmoon');
    var cloudR    = document.getElementById('cvi-cloud');
    var cloudVal  = document.getElementById('cvi-cloud-val');
    var transR    = document.getElementById('cvi-trans');
    var transNum  = document.getElementById('cvi-trans-num');
    var statusBox = document.getElementById('cvi-status');
    var results   = document.getElementById('cvi-results');
    var cardsBox  = document.getElementById('cvi-cards');
    var ctxDiv    = document.getElementById('cvi-city-context');
    var mapDiv    = document.getElementById('cvi-map');

    // Required cities — always available even if every server source fails.
    var REQUIRED_CITIES = [
        { slug: 'jerusalem', name: 'Jerusalem', lat: 31.7683, lon: 35.2137 },
        { slug: 'dallas', name: 'Dallas', lat: 32.7767, lon: -96.7970 },
        { slug: 'melbourne', name: 'Melbourne', lat: -37.8136, lon: 144.9631 }
    ];

    // City list: prefer data attribute (resilient to minifiers), then localized var, then required fallback.
    var CITIES;
    try {
        var fromAttr = root.dataset.cities ? JSON.parse(root.dataset.cities) : null;
        if (fromAttr && fromAttr.length > 0 && fromAttr[0].slug) {
            CITIES = fromAttr;
        }
    } catch (e) {}

    if (!CITIES || !CITIES.length) {
        CITIES = (data.cities && data.cities.length) ? data.cities : REQUIRED_CITIES.slice();
    }

    // Drop any malformed entry (blank slug) so a junk DB row can't create an
    // empty <option> that gets submitted as city="" (paleotimes.org HAR).
    CITIES = CITIES.filter(function (c) { return c && c.slug; });
    if (!CITIES.length) {
        CITIES = REQUIRED_CITIES.slice();
    }

    // Precomputed dataset embedded in the page (slug -> year -> [[date,[d0,d1,d2],q,age],...]).
    // When present, the tool runs entirely offline — no admin-ajax/REST needed
    // (works even when /wp-admin is behind Cloudflare Access). AJAX is fallback.
    var DATASET = {};
    try {
        var dsEl = root.querySelector('script.cvi-dataset');
        if (dsEl && dsEl.textContent.trim()) {
            DATASET = JSON.parse(dsEl.textContent);
        }
    } catch (e) { DATASET = {}; }

    function datasetRows(city, year) {
        var byYear = DATASET[city];
        if (!byYear) { return null; }
        var rows = byYear[String(year)];
        if (!rows) { return null; }
        return rows.map(function (r) {
            return { new_moon_date: r[0], days: r[1], day_q: r[2], day_age: r[3] };
        });
    }

    // Which year (in the embedded data) contains a given new-moon date.
    function yearOfDate(city, date) {
        var byYear = DATASET[city];
        if (!byYear || !date) { return null; }
        for (var y in byYear) {
            if (Object.prototype.hasOwnProperty.call(byYear, y)) {
                for (var i = 0; i < byYear[y].length; i++) {
                    if (byYear[y][i][0] === date) { return parseInt(y, 10); }
                }
            }
        }
        return null;
    }

    // Closest upcoming new moon for a city (first date >= today, else the last
    // available). Used as the default landing date.
    function closestNewMoonDate(city) {
        var byYear = DATASET[city];
        if (!byYear) { return null; }
        var all = [];
        for (var y in byYear) {
            if (Object.prototype.hasOwnProperty.call(byYear, y)) {
                byYear[y].forEach(function (r) { all.push(r[0]); });
            }
        }
        if (!all.length) { return null; }
        all.sort();
        var today = new Date().toISOString().slice(0, 10);
        for (var i = 0; i < all.length; i++) {
            if (all[i] >= today) { return all[i]; }
        }
        return all[all.length - 1];
    }

    var CATEGORY_COLORS = { A: '#22d3ee', B: '#67e8f9', C: '#facc15', D: '#fde047', E: '#f59e0b' };

    var liveTimer = null;          // debounce handle for slider-driven re-render
    var moonAbort = null;          // AbortController for the in-flight newmoons request
    var moonCache = {};            // `${city}:${year}` -> normalized rows
    var leafletMap = null;
    var leafletMarker = null;

    // -----------------------------------------------------------------
    // Exact JS port of main.go:applyAtmosphericAdjustment
    // -----------------------------------------------------------------
    function applyAtmosphericAdjustment(raw, cloud, trans) {
        var categoryValue = ({ A: 5, B: 4, C: 3, D: 2, E: 1, F: 0 })[raw];
        if (categoryValue === undefined || categoryValue === 0) {
            categoryValue = 1;
        }

        var adjustment = 0;
        if (cloud > 80) { adjustment -= 3; }
        else if (cloud > 60) { adjustment -= 2; }
        else if (cloud > 40) { adjustment -= 1; }

        if (trans >= 9) { adjustment += 1; }
        else if (trans <= 4) { adjustment -= 1; }
        else if (trans <= 2) { adjustment -= 2; }

        var finalValue = Math.max(1, Math.min(5, categoryValue + adjustment));
        var effective = ({ 5: 'A', 4: 'B', 3: 'C', 2: 'D', 1: 'E' })[finalValue];

        var note;
        if (adjustment === 0) {
            note = __('Atmospheric conditions have minimal impact on this prediction.');
        } else if (adjustment < 0) {
            note = _sprintf(__('Conditions are reducing visibility by approximately %d category level(s).'), -adjustment);
        } else {
            note = __('Excellent atmospheric conditions are slightly improving the prediction.');
        }
        return [effective, note];
    }

    // -----------------------------------------------------------------
    // Small DOM / fetch helpers
    // -----------------------------------------------------------------
    function setStatus(message, isError) {
        statusBox.textContent = message || '';
        statusBox.classList.toggle('cvi-status--error', !!isError);
    }

    function escapeHtml(s) {
        return String(s).replace(/[&<>"']/g, function (c) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
        });
    }

    // Find a <select> <option> by exact value WITHOUT building a CSS selector
    // from data (avoids selector-injection edge cases if a value ever contained
    // quotes/brackets).
    function findOption(sel, value) {
        if (value === null || value === undefined) { return null; }
        var opts = sel.options;
        for (var i = 0; i < opts.length; i++) {
            if (opts[i].value === String(value)) { return opts[i]; }
        }
        return null;
    }

    function ajax(action, params, signal) {
        var qs = new URLSearchParams(Object.assign({ action: action, nonce: data.nonce || '' }, params));
        return fetch(data.ajaxUrl + '?' + qs.toString(), { signal: signal })
            .then(function (r) { return r.json(); })
            .then(function (json) {
                if (!json || !json.success) {
                    throw new Error((json && json.data && json.data.message) || 'Request failed');
                }
                return json.data;
            });
    }

    // -----------------------------------------------------------------
    // City + context map
    // -----------------------------------------------------------------
    function populateCities(defaultSlug) {
        citySel.innerHTML = '';
        CITIES.forEach(function (c) {
            var o = document.createElement('option');
            o.value = c.slug;
            o.textContent = c.name;
            citySel.appendChild(o);
        });
        if (defaultSlug && findOption(citySel, defaultSlug)) {
            citySel.value = defaultSlug;
        } else if (CITIES.length) {
            // Never leave the selection empty — that would send city="".
            citySel.value = CITIES[0].slug;
        }
    }

    function currentCity() {
        return CITIES.find(function (c) { return c.slug === citySel.value; }) || CITIES[0];
    }

    var quickBtns = root.querySelectorAll('.cvi-quick__btn');

    // Disable a quick button if its city isn't in the current dropdown (e.g. the
    // imported dataset doesn't include it), and reflect the active selection.
    function syncQuickButtons() {
        quickBtns.forEach(function (btn) {
            var present = !!findOption(citySel, btn.dataset.slug);
            btn.disabled = !present;
            btn.classList.toggle('is-active', present && btn.dataset.slug === citySel.value);
        });
    }

    function updateCityContext() {
        var c = currentCity();
        if (!c) { return; }
        ctxDiv.textContent = c.name + ' • ' + c.lat.toFixed(3) + '°, ' + c.lon.toFixed(3) + '° — pre-computed Yallop data';
        updateMap(c);
        syncQuickButtons();
    }

    function updateMap(city) {
        if (!mapDiv) { return; }

        // Leaflet is bundled and enqueued as a dependency, so window.L is ready.
        // If it somehow failed to load, hide the (purely cosmetic) map.
        if (!window.L) { mapDiv.style.display = 'none'; return; }

        if (!leafletMap) {
            leafletMap = window.L.map(mapDiv, {
                attributionControl: false, zoomControl: false, dragging: false,
                scrollWheelZoom: false, doubleClickZoom: false, touchZoom: false
            }).setView([city.lat, city.lon], 6);
            window.L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png').addTo(leafletMap);
        } else {
            leafletMap.setView([city.lat, city.lon], 6);
        }
        if (leafletMarker) { leafletMarker.remove(); }
        leafletMarker = window.L.marker([city.lat, city.lon]).addTo(leafletMap);
        setTimeout(function () { leafletMap.invalidateSize(); }, 150);
    }

    // -----------------------------------------------------------------
    // Data loading
    // -----------------------------------------------------------------
    function loadYears(city) {
        if (!city) {
            // Guard against the empty-city request that produced years:[] live.
            yearSel.innerHTML = '<option value="">' + escapeHtml(__('No city selected')) + '</option>';
            setStatus(__('No city available. Check that data has been imported.'), true);
            return Promise.resolve([]);
        }
        // Prefer the embedded dataset (no network — survives Cloudflare Access).
        if (DATASET[city]) {
            var dsYears = Object.keys(DATASET[city]).map(Number).sort(function (a, b) { return a - b; });
            yearSel.innerHTML = '';
            dsYears.forEach(function (y) {
                var o = document.createElement('option');
                o.value = y; o.textContent = y;
                yearSel.appendChild(o);
            });
            if (!dsYears.length) {
                yearSel.innerHTML = '<option value="">' + escapeHtml(__('No data')) + '</option>';
                moonSel.innerHTML = '<option value="">' + escapeHtml(__('No data')) + '</option>';
                results.hidden = true;
                setStatus(__('No visibility data for this city yet.'), true);
            } else {
                setStatus('');
            }
            return Promise.resolve(dsYears);
        }

        yearSel.innerHTML = '<option>' + escapeHtml(__('Loading…')) + '</option>';
        return ajax('cvi_get_years', { city: city }).then(function (d) {
            var years = d.years || [];
            yearSel.innerHTML = '';
            years.forEach(function (y) {
                var o = document.createElement('option');
                o.value = y; o.textContent = y;
                yearSel.appendChild(o);
            });
            if (!years.length) {
                // No observations for this city yet — tell the user why the
                // year/new-moon dropdowns are empty instead of failing silently.
                yearSel.innerHTML = '<option value="">' + escapeHtml(__('No data')) + '</option>';
                moonSel.innerHTML = '<option value="">' + escapeHtml(__('No data')) + '</option>';
                results.hidden = true;
                setStatus(__('No visibility data for this city yet. An administrator needs to import a dataset under Tools → Crescent Visibility.'), true);
            } else {
                setStatus('');
            }
            return years;
        }).catch(function () {
            yearSel.innerHTML = '<option value="">' + escapeHtml(__('No data')) + '</option>';
            setStatus(__('Could not reach the data endpoint. Please try again.'), true);
            return [];
        });
    }

    function loadNewMoons(city, year, preferDate) {
        var key = city + ':' + year;
        moonSel.disabled = true;
        moonSel.innerHTML = '<option>' + escapeHtml(__('Loading new moons…')) + '</option>';

        var apply = function (moons) {
            moonSel.innerHTML = '';
            if (!moons.length) {
                moonSel.innerHTML = '<option value="">' + escapeHtml(__('No new moons for this year')) + '</option>';
                moonSel.disabled = false;
                return;
            }
            moons.forEach(function (m) {
                var o = document.createElement('option');
                o.value = m.new_moon_date;
                o.textContent = m.new_moon_date;
                o.dataset.days = JSON.stringify(m.days);
                o.dataset.dayq = JSON.stringify(m.day_q || []);
                o.dataset.dayage = JSON.stringify(m.day_age || []);
                moonSel.appendChild(o);
            });

            // Selection priority: the caller's preferred date (e.g. the date
            // carried over from the previous city), then the server smart
            // default, then the first option.
            var preferred = preferDate || data.defaultNewMoon;
            if (preferred && findOption(moonSel, preferred)) {
                moonSel.value = preferred;
            } else {
                moonSel.selectedIndex = 0;
            }
            moonSel.disabled = false;
            renderResults();
        };

        // Embedded dataset first (no network).
        var embedded = datasetRows(city, year);
        if (embedded) {
            moonCache[key] = embedded;
            apply(embedded);
            return Promise.resolve();
        }

        if (moonCache[key]) {
            apply(moonCache[key]);
            return Promise.resolve();
        }

        if (moonAbort) { moonAbort.abort(); }
        moonAbort = ('AbortController' in window) ? new AbortController() : null;

        return ajax('cvi_get_newmoons', { city: city, year: year }, moonAbort ? moonAbort.signal : undefined)
            .then(function (d) {
                moonCache[key] = d.new_moons || [];
                apply(moonCache[key]);
            })
            .catch(function (err) {
                if (err && err.name === 'AbortError') { return; }
                moonSel.innerHTML = '<option value="">' + escapeHtml(__('Error loading data')) + '</option>';
                moonSel.disabled = false;
                setStatus(__('Could not load new moon data. Please try again.'), true);
            });
    }

    // -----------------------------------------------------------------
    // Result cards
    // -----------------------------------------------------------------
    function renderResults() {
        var opt = moonSel.options[moonSel.selectedIndex];
        if (!opt || !opt.dataset.days) {
            results.hidden = true;
            return;
        }

        var days = JSON.parse(opt.dataset.days || '["?","?","?"]');
        var dayQ = JSON.parse(opt.dataset.dayq || '[]');
        var dayAge = JSON.parse(opt.dataset.dayage || '[]');
        var cloud = parseInt(cloudR.value, 10);
        var trans = parseFloat(transNum.value);
        var labels = [__('Day +0'), __('Day +1'), __('Day +2')];
        var html = '';

        for (var i = 0; i < 3; i++) {
            var raw = String(days[i] || '?').toUpperCase();
            // Each evening has its own age + Q (not the best evening's).
            var age = (dayAge[i] === null || dayAge[i] === undefined) ? 0 : Number(dayAge[i]);
            var q = (dayQ[i] === null || dayQ[i] === undefined) ? 0 : Number(dayQ[i]);

            // Same graceful handling as main.go's handlePointQuery.
            if (raw === 'J' || raw === '?' || age > 100) {
                html += '<div class="cvi-card cvi-card--empty">'
                    + '<div class="cvi-card__label">' + escapeHtml(labels[i]) + '</div>'
                    + '<div class="cvi-card__empty-title">' + escapeHtml(__('Not a good crescent window')) + '</div>'
                    + '<div class="cvi-card__empty-note">' + escapeHtml(__('The selected date is too far from actual new moon conjunction for reliable prediction.')) + '</div>'
                    + '</div>';
                continue;
            }

            var pair = applyAtmosphericAdjustment(raw, cloud, trans);
            var eff = pair[0];
            var note = pair[1];
            var color = CATEGORY_COLORS[eff] || '#64748b';
            var ageQ = _sprintf(__('Age: %1$s h • Q: %2$s'), age.toFixed(1), q.toFixed(3));

            html += '<div class="cvi-card cvi-card--cat" style="--cat:' + color + ';">'
                + '<div class="cvi-card__label">' + escapeHtml(labels[i]) + '</div>'
                + '<div class="cvi-card__head">'
                + '<div><div class="cvi-card__sub">' + escapeHtml(__('Effective')) + '</div>'
                + '<div class="cvi-card__big">' + escapeHtml(eff) + '</div></div>'
                + '<div class="cvi-card__meta">'
                + '<div>' + escapeHtml(__('Raw')) + ' <span class="cvi-mono">' + escapeHtml(raw) + '</span></div>'
                + '<div class="cvi-card__sub">' + escapeHtml(ageQ) + '</div>'
                + '</div></div>'
                + '<div class="cvi-card__note">' + escapeHtml(note) + '</div>'
                + '</div>';
        }

        cardsBox.innerHTML = html;
        results.hidden = false;
    }

    // -----------------------------------------------------------------
    // Wiring
    // -----------------------------------------------------------------
    function syncSliders() {
        cloudVal.textContent = cloudR.value;
        transNum.value = transR.value;
    }

    function scheduleLiveRender() {
        clearTimeout(liveTimer);
        liveTimer = setTimeout(function () {
            if (!results.hidden) { renderResults(); }
        }, 130);
    }

    cloudR.addEventListener('input', function () { syncSliders(); scheduleLiveRender(); });
    transR.addEventListener('input', function () { syncSliders(); scheduleLiveRender(); });
    transNum.addEventListener('input', function () {
        transR.value = transNum.value;
        scheduleLiveRender();
    });

    quickBtns.forEach(function (btn) {
        btn.addEventListener('click', function () {
            var slug = btn.dataset.slug;
            if (citySel.value === slug) { return; }
            if (findOption(citySel, slug)) {
                citySel.value = slug;
                citySel.dispatchEvent(new Event('change'));
            }
        });
    });

    // Select a city and land on a sensible new-moon date: keep the requested
    // date if that city has it (new moons are global, so it normally does),
    // otherwise the closest upcoming one — never jump to the last year.
    function selectCity(city, preferredDate) {
        updateCityContext();
        results.hidden = true;
        return loadYears(city).then(function (years) {
            if (!years.length) { return; }

            var targetDate = (preferredDate && yearOfDate(city, preferredDate))
                ? preferredDate
                : closestNewMoonDate(city);

            var targetYear = targetDate ? yearOfDate(city, targetDate) : null;
            if (!targetYear || years.indexOf(targetYear) === -1) {
                // Fall back to the server default year, else the earliest year.
                targetYear = (data.defaultYear && years.indexOf(data.defaultYear) !== -1)
                    ? data.defaultYear
                    : years[0];
            }
            yearSel.value = targetYear;
            loadNewMoons(city, targetYear, targetDate);
        });
    }

    citySel.addEventListener('change', function () {
        // Carry the currently selected date to the new city.
        selectCity(citySel.value, moonSel.value || data.defaultNewMoon);
    });

    yearSel.addEventListener('change', function () {
        results.hidden = true;
        loadNewMoons(citySel.value, yearSel.value);
    });

    moonSel.addEventListener('change', renderResults);

    // -----------------------------------------------------------------
    // Boot: seed from the server-provided smart defaults
    // Priority: data attributes (robust) > localized var > hard defaults
    // -----------------------------------------------------------------
    var initialCity = root.dataset.defaultCity || data.defaultCity || 'jerusalem';
    populateCities(initialCity);
    syncSliders();

    // Land on the server smart default (closest upcoming) if we have one,
    // otherwise the city's closest upcoming new moon.
    var bootDate = data.defaultNewMoon || closestNewMoonDate(citySel.value);
    selectCity(citySel.value, bootDate).then(function () {
        if (!Object.keys(DATASET).length && !yearSel.value) {
            setStatus(__('No data imported yet. Import a dataset under Tools → Crescent Visibility.'), true);
        }
    });
})();
