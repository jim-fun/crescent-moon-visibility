# Ollama Prompt: Full Interactive "Visibility for My Location" Shortcode (Exploratory First Draft)

**Goal**: Produce a rough but functional first draft of an interactive shortcode that feels as close as possible to the original web app's /point page, powered by the imported precomputed data.

**Requirements for this exploratory pass**:
- The shortcode should render a complete form + results area.
- Form elements:
  - City selector (dropdown populated from the cities that have data in the DB, or hard-coded list of the 13 for simplicity in v1).
  - Year selector (dynamically loaded or from available data).
  - New Moon selector (populated when year or city changes — use AJAX or preload data).
  - Atmospheric sliders: Cloud cover (0-100%) and Transparency (1-10), with live or button-triggered recalculation.
  - "Check Visibility" button (or live updates).
- On submit / update:
  - Look up the 3 raw daily categories for the chosen new moon + city from the imported data.
  - Apply the atmospheric adjustment heuristic to each of the 3 days using the current slider values.
  - Display 3 result cards (one per day) showing:
    - Date
    - Large Raw category letter
    - Large Effective category letter (after adjustment)
    - Short explanation based on the effective category
    - Optionally the original raw daily values
- Make it reasonably nice visually (you can use inline Tailwind via CDN or clean custom CSS for a first draft).
- Include basic JavaScript for dynamic behavior (especially loading new moons and applying adjustment live).
- Handle the case where no data is imported yet with a friendly message pointing to the admin import page.
- Keep the code as self-contained as possible for a first draft (one file or clearly separated PHP + JS).

**Style for this Ollama pass**:
- Exploratory and volume-oriented: Provide a working skeleton even if it's a bit rough or has placeholder parts.
- Offer 2-3 variations on key UX decisions (e.g., live update vs button, how to display the new moon list, simple vs richer cards).
- Note any obvious limitations or "TODO" items for later refinement with Claude or human review.

**Output format requested**:
- The main PHP shortcode function / class method.
- The JavaScript (can be inline in a script tag for the first draft).
- Basic CSS (Tailwind via CDN is fine for v1).
- Example shortcode usage.
- Short notes on integration points with the existing main plugin class and database schema.

Focus on making it feel interactive and close to the original web app experience. We will refine accuracy, performance, and polish in later passes.