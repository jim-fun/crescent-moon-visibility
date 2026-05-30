# Ollama Prompt: Nice Result Cards Matching the Web App Style

**Context**:
We have precomputed raw daily visibility data (3 days per new moon) plus the ability to apply the atmospheric adjustment heuristic.

We want result cards that feel very close to the original web app's output for the point query tool.

Typical card from the web app includes:
- Date
- Large, prominent Effective category letter (color-coded: A cyan, B lighter cyan, C yellow, D lighter yellow, E amber)
- Raw category shown smaller
- Age in hours and Q value (if available from data)
- Short plain-English explanation based on the Effective category (and atmospheric note if adjusted)

**Task**:
Create clean, reusable HTML + CSS (Tailwind via CDN is acceptable for minimal footprint, or plain modern CSS) for attractive, accessible result cards.

Provide:
- A single card component that can be reused for each of the 3 days.
- Color mapping for A-E (matching the original web app as closely as possible).
- Variations: one for when atmospheric adjustment has been applied (show both Raw and Effective clearly, plus the adjustment note).
- Responsive and clean layout.
- Optional: small icons or visual indicators if it improves clarity without adding dependencies.

Include example usage (how to render 3 cards for a chosen new moon after looking up the raw values and applying adjustment).

Keep it lightweight and easy to integrate into a WordPress shortcode output.