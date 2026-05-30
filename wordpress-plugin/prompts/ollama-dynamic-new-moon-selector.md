# Ollama Prompt: Dynamic New Moon Selector + City + Atmospheric UI for WP Plugin

**Context**:
We have a WordPress plugin with precomputed visibility data imported into custom tables for 13 major cities (including Mecca, Karachi, Rabat, Jerusalem, etc.).

We have PHP helper methods available:
- get_available_years_for_city($city_slug)
- get_new_moons_for_city_and_year($city_slug, $year)  // returns rows with new_moon_date + raw daily categories
- apply_atmospheric_adjustment($raw_category, $cloud_percent, $transparency)

**Task**:
Create the frontend (PHP + JavaScript) for a rich, interactive "Visibility for My Location" experience similar to the original Go web app's /point tool, but using only the imported precomputed data.

Requirements:

1. **City Selection**
   - Dropdown populated dynamically from the cities that have data in the database.
   - Optional quick preset buttons for the most important ones (Jerusalem, Mecca, etc.).

2. **Year + New Moon Selection** (the dynamic part)
   - Year dropdown (fetched via AJAX from the helper or a lightweight endpoint).
   - When the user changes the city or year, use AJAX (or fetch) to load the list of available new moons for that city + year.
   - Display them nicely (e.g., "2026-03-20 (New Moon)" or just the date, with ability to select one).

3. **Atmospheric Conditions**
   - Cloud cover slider (0–100%)
   - Transparency (1–10)
   - These should affect the displayed "Effective" categories live or on submit, by calling the PHP heuristic on the stored raw daily values for the selected new moon.

4. **Results Display**
   - On "Check" or live update, show the 3 days after the chosen new moon.
   - For each day: Date, Raw category (large), Effective category after current atmospheric settings (large, color-coded), plus a short explanation.
   - Show the original raw daily values too.
   - Nice card layout matching the style of the original web app results.

5. **Technical Notes**
   - Use vanilla JS + fetch for AJAX (keep dependencies minimal).
   - Create lightweight AJAX endpoints (or use admin-ajax) that call the existing helper methods.
   - Handle the case where no data is imported yet with a friendly message.
   - Make it work as a shortcode that can be placed on any page.

Output the following:
- PHP code for registering the shortcode and enqueuing JS/CSS.
- The JavaScript that powers the dynamic behavior (year change → load new moons, atmospheric change → recalc effective, submit → show results).
- Any necessary CSS for a clean modern look (Tailwind via CDN is acceptable for minimal footprint).
- Example shortcode usage.
- Notes on how to wire the AJAX actions to the existing plugin class methods.

Prioritize clean, readable code that can be iterated on. Provide multiple options where there are UX decisions (e.g., live update vs button, how to display the new moon list).