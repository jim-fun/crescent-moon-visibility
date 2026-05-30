# Ollama Prompt: Interactive "Visibility for My Location" Form for WordPress Plugin

**Context**:
We are building a minimal-footprint WordPress plugin that replicates the "Visibility for My Location" (/point) experience from the original Go web app (`crescent_maps web`).

The plugin uses **pre-computed data** (imported via admin as JSON) for a fixed set of 13 major cities. This avoids needing the heavy C++/Go renderer at runtime on the WordPress server.

**Current State**:
- Data is imported into custom tables (`crescent_observations` and `crescent_cities`).
- We have helper methods in the main plugin class:
  - `get_available_years_for_city($city_slug)`
  - `get_new_moons_for_city_and_year($city_slug, $year)`
  - `apply_atmospheric_adjustment($raw_category, $cloud_percent, $transparency)` (ported heuristic)
- The current shortcode is basic (just dumps a table for fixed city + year range).
- We have a real 2026-2028 dataset with raw daily categories for each new moon.

**Goal for this task**:
Create a **rich, interactive frontend** (shortcode or Gutenberg block + supporting PHP/JS) that feels as close as possible to the original web app's point query tool, but powered entirely by the imported precomputed data.

**Required Features** (match the web app as closely as possible):

1. **City Selection**
   - Dropdown populated from the imported cities in the database (the 13 cities).
   - Optional: Show a simple static map image or Leaflet map (read-only) centered on the selected city for context.
   - Presets/buttons for quick selection (at minimum the original three: Jerusalem, Dallas, Melbourne, plus the new ones like Mecca if possible).

2. **Year + New Moon Selection**
   - Year dropdown (populated from available years in the data for the selected city).
   - When year changes (AJAX or JS fetch), dynamically populate a "New Moon" dropdown with the available new moons for that city + year.
   - Display the new moon date clearly.

3. **Atmospheric Conditions** (key feature from web app)
   - Cloud cover slider (0-100%)
   - Transparency number input or slider (1-10)
   - On change, live or on-submit recalculation of Effective categories using the stored raw daily values + the `apply_atmospheric_adjustment` heuristic.

4. **Results Display**
   - On "Check Visibility" (or live update):
     - Show the 3 days after the chosen new moon.
     - For each day: Date, Raw category (large letter), Effective category (large letter, color-coded), Age (if available), Q (if available), and the atmospheric note.
   - Use a card-style layout similar to the original web app results (nice Tailwind-like cards or clean CSS).
   - Include the explanations for A-E categories.

5. **Technical Constraints (Minimal Footprint)**
   - No heavy server computation on every request.
   - All visibility math uses the precomputed raw data + lightweight PHP/JS heuristic.
   - Prefer vanilla JS + minimal dependencies where possible (or jQuery if WP already loads it).
   - AJAX endpoints should be lightweight (just query the DB for new moons or compute adjustment).

**Output Expected from You (Ollama)**:
- A complete, well-commented PHP class or functions for the shortcode/block registration and rendering.
- The JavaScript needed for:
  - Dynamic loading of years/new moons when city/year changes.
  - Handling atmospheric sliders and recalculating Effective categories client-side or via AJAX.
  - Rendering the result cards.
- Any necessary enqueues (JS/CSS).
- Example usage of the shortcode.
- Notes on how to integrate with the existing main plugin class and helper methods.
- Suggestions for graceful degradation (e.g., what if no data is imported yet).

**Style Notes**:
- Keep the code clean, maintainable, and following WordPress coding standards where applicable.
- Make the UI feel modern and close to the original Tailwind-based web app.
- Prioritize usability for observers (clear labels, good defaults like Jerusalem + next available new moon if possible).
- Include helpful comments explaining the data flow.

**Reference Materials** (you should imagine or recall):
- The original web app's /point HTML/JS form and result cards.
- The existing plugin structure (main class with helpers, admin import page, current basic renderer).
- The database schema (raw_day_0/1/2 per observation row).

Please produce production-ready starter code + JS that can be dropped into the existing plugin with minimal changes. If you need to suggest small additions to the main plugin class, note them clearly.

Start with the PHP shortcode output (the form HTML), then the JavaScript, then any supporting PHP methods.