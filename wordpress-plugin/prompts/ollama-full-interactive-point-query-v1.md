# Ollama Prompt: Full Interactive "Visibility for My Location" Shortcode (Exploratory First Draft)

**Role**: You are an exploratory coding assistant (Ollama-style). Produce a rough but functional first-draft implementation. Prioritize volume of ideas and working skeleton over perfection. Include multiple approaches where there are meaningful UX or technical decisions. Use lots of comments and TODOs.

**Project Context**:
- We have a minimal-footprint WordPress plugin for crescent visibility.
- Data is pre-computed offline for 13 major cities and imported via the admin as JSON into two tables:
  - crescent_cities
  - crescent_observations (contains per-new-moon: new_moon_date, year, raw_day_0/1/2, best_raw, best_effective)
- We already have these PHP helper methods available in the main plugin class:
  - get_available_years_for_city($city_slug)
  - get_new_moons_for_city_and_year($city_slug, $year)
  - apply_atmospheric_adjustment($raw_category, $cloud_percent, $transparency)
- The plugin already has a basic manual import system and a simple shortcode that currently just shows a table.

**Goal**:
Create a rich, interactive "Visibility for My Location" experience inside a WordPress shortcode that feels as close as possible to the original Go web app's /point tool, but powered entirely by the imported precomputed data (no live heavy renderer calls).

**Required Features (match the web app experience as closely as possible)**:

1. **City Selection**
   - Dropdown or nice buttons/cards for the imported cities (at minimum the 13 we chose).
   - Optional: Show a simple map or location context for the selected city.

2. **Year + New Moon Selection** (dynamic)
   - Year dropdown populated from the data for the selected city.
   - When year changes, dynamically load the list of available new moons for that city + year (via AJAX or by pre-loading data).
   - Allow the user to pick one specific new moon.

3. **Atmospheric Conditions**
   - Cloud cover slider (0–100%)
   - Transparency (1–10)
   - These must actually affect the displayed Effective categories by applying the atmospheric heuristic to the 3 stored raw daily values.

4. **Results**
   - On "Check" (or live update), display 3 result cards (one per day after the chosen new moon).
   - Each card should show:
     - Date
     - Large Raw category
     - Large Effective category (after current atmospheric settings)
     - Short explanation based on the effective category
     - Ideally the original raw daily values too

**Technical Constraints**:
- Must work as a normal WordPress shortcode.
- Keep external dependencies minimal (vanilla JS is preferred; Tailwind via CDN is acceptable for v1).
- All heavy computation must come from the precomputed data + the lightweight PHP/JS version of the atmospheric heuristic.
- Handle the case where no data has been imported yet with a friendly message.

**Output Expectations**:
- Provide a complete (but rough) first-draft implementation.
- Include the PHP shortcode function.
- Include the JavaScript (can be inline in a <script> tag for the first draft).
- Include basic accompanying CSS.
- Show example shortcode usage.
- Include notes on integration points with the existing plugin class.
- Offer 2–3 different approaches for at least two key decisions (e.g., live update vs button-triggered results, how to handle custom locations vs only precomputed cities).
- Use lots of TODO comments for things that should be cleaned up later.

**Style**: Exploratory and volume-oriented. It is okay if this draft is a bit messy or has placeholder pieces, as long as the core flow works. We will refine it later with Claude or human review.

Start coding. Prioritize making the interactive flow feel good.