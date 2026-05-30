# GLM-4.7-flash Prompt: Dynamic New Moon Loading + Live Atmospheric Adjustment (JS)

You are generating exploratory first-draft JavaScript for a minimal WordPress plugin.

Context:
- We have precomputed visibility data in the WordPress database for 13 cities.
- We already have PHP helper methods available:
  - get_available_years_for_city($city_slug)
  - get_new_moons_for_city_and_year($city_slug, $year)  → returns array of {new_moon_date, raw_day_0, raw_day_1, raw_day_2, best_raw, best_effective}
  - apply_atmospheric_adjustment($raw_category, $cloud_percent, $transparency)

Task:
Write a self-contained JavaScript module (can be used inside a WordPress shortcode) that handles:

1. When the user changes the City or Year:
   - Fetch the list of available new moons for that city + year (via AJAX to a lightweight endpoint that calls the PHP helper).
   - Populate the New Moon dropdown.

2. When the user changes the atmospheric sliders (cloud cover + transparency):
   - For the currently selected new moon, take its 3 raw daily values.
   - Apply the atmospheric adjustment heuristic **on the client side** (you must also provide a JavaScript version of the heuristic).
   - Update the displayed Effective categories live (or with light debouncing).

3. When the user clicks "Check Visibility":
   - Show 3 result cards (one per day).
   - Each card should display: Date, Raw category, Adjusted Effective category, and a short explanation.

Requirements for this draft:
- Use vanilla JavaScript + fetch (no heavy frameworks).
- Assume we will add a simple AJAX endpoint later (you can suggest the endpoint name and what it should return).
- Provide a clean JavaScript version of the atmospheric adjustment function.
- Include loading states and basic error handling.
- Make it reasonably easy to drop into a WordPress shortcode.

Offer 2 different approaches for the dynamic loading part:
- Approach A: Simple fetch on every change (easiest).
- Approach B: Preload all years + new moons for the selected city on city change, then filter client-side (faster after first load).

Also include comments on how this would integrate with the existing PHP plugin class.