# WordPress Plugin: Pre-computed Visibility for Major Cities

**Date**: 2026 (current session)
**Status**: Proposal / Feasibility exploration

## Goal

Create a very lightweight WordPress plugin (minimal PHP/JS footprint, no native binaries at runtime) that lets users see historical and near-future crescent visibility likelihood for a fixed set of major cities.

Instead of on-demand heavy computation (current Go web app behavior), we pre-compute accurate data offline using the authoritative renderer and load it into the WordPress database.

## Key Constraints for "Minimal Footprint" Plugin

- No `exec()` of C++/Go renderer on the WP server
- Works on typical shared hosting
- Small plugin size (< 1MB including data)
- Easy data updates via admin import
- No external API calls required for core functionality

## Proposed Scope

### Cities (initial set)

Start with ~10 major cities, chosen for geographic/cultural diversity:

1. Jerusalem (31.7683, 35.2137) — existing preset
2. Dallas (32.7767, -96.7970) — existing preset
3. Melbourne (-37.8136, 144.9631) — existing preset
4. Cairo (30.0444, 31.2357)
5. London (51.5074, -0.1278)
6. Tokyo (35.6762, 139.6503)
7. Rio de Janeiro (-22.9068, -43.1729)
8. Cape Town (-33.9249, 18.4241)
9. Mumbai (19.0760, 72.8777)
10. Istanbul (41.0136, 28.9550)

Additional candidates for later: Mecca, New York, Sydney, Beijing, Buenos Aires.

### Time Range

- Start: 2006
- "Up to 10 years" windows: Users can select any 10-year span beginning 2006 (e.g., 2006-2015, 2015-2024, 2018-2027, etc.)
- Future years: Pre-compute 3-5 years ahead so the data stays useful without constant updates.

### What Data to Store (per city + new moon)

For each new moon date + city:

- New moon date (YYYY-MM-DD)
- Year
- Three daily raw categories (day 0, 1, 2 after conjunction)
- Best raw category in the window
- Best *effective* category (using a standard "clear" atmospheric assumption, e.g. 0% cloud, transparency 7-8)
- Optional: Moon age at best time, Q value

Aggregated per year (for quick "likelihood" views):
- Number of new moons that year
- Best visibility seen that year
- Number of windows with best effective >= B, >= C, etc.
- Simple "good visibility" percentage

## Data Generation Process (Offline)

Use the existing project tooling:

1. Build the full renderer + Go binary.
2. Write a small generator (suggested location: `scripts/generate_city_visibility/main.go` or a one-off command).
3. For each city + each year from 2006 onward:
   - Get accurate new moon dates via `jobspec.GetNewMoonsForYear(year)` (uses real astro when available).
   - For each new moon, call the renderer in `point` mode for the 3 days.
   - Record raw categories + compute effective under standard conditions.
4. Export as JSON or CSV (one file per city or one big file with city column).

Example output row:

```json
{
  "city": "jerusalem",
  "new_moon": "2025-03-29",
  "year": 2025,
  "days": ["C", "B", "A"],
  "best_raw": "A",
  "best_effective": "B",
  "q_at_best": 0.123,
  "age_at_best": 18.4
}
```

This data is generated once using the high-accuracy path. The WP plugin never runs this code.

## WordPress Plugin Architecture (Minimal)

- **Custom table** (or use `wp_options` + JSON for extreme simplicity):
  `wp_crescent_visibility` with columns: `id, city, new_moon_date, year, data_json`
- **Admin page**: "Import Visibility Data" – upload JSON/CSV, validate, bulk insert.
- **Shortcode / Block**: `[crescent_visibility city="jerusalem" years="2015-2024"]`
  - Dropdown or tabs for city
  - Year range selector (limited to 10-year windows starting 2006+)
  - Table or simple heatmap/calendar showing per new moon the best category
  - Summary stats for the selected period ("In this 10-year window, Jerusalem had X windows with effective A or B")

- **No live computation** — everything is lookup + display.
- **JavaScript**: Only for interactivity (filtering, charts). Can use Chart.js (small) or pure CSS tables.

## Advantages for Minimal Footprint

- Plugin size dominated by data (still small — ~10 cities × ~25 new moons/year × 20 years ≈ 5,000 rows. Very manageable).
- Pure PHP + JS + database.
- Updates are deliberate data imports, not code changes.
- Preserves full Accuracy First for the pre-computed data (generated with the reference renderer).

## Challenges & Open Questions

1. **"Likelihood" Definition**
   - What exactly do we show? Best category per window? Percentage of days rated C or better? Something else?
   - Need a clear, observer-friendly metric.

2. **Data Freshness**
   - How often do we expect users to re-import newer years?
   - Should we ship the plugin with a base dataset up to the current year + 3 years projected?

3. **Atmospheric Conditions**
   - Pre-computed data assumes "standard" clear skies. The live tool allows per-query adjustment.
   - In the plugin we can still let users apply a simple post-hoc adjustment (the existing heuristic is tiny JS).

4. **Granularity**
   - Per new moon + 3 days is rich but verbose.
   - Yearly summary + drill-down might be friendlier for 10-year views.

5. **Cultural/Religious Use**
   - Many users care about specific criteria (naked eye only, or specific moon sighting traditions). Pre-computed data should note the criterion used (Yallop).

## Recommended Next Steps (if pursuing)

1. Define the exact output schema and "likelihood" metric.
2. Implement a data generator script in this repo that can produce the import files for the chosen cities.
3. Create a minimal WP plugin skeleton (single file or well-organized) with:
   - Admin import page
   - Shortcode renderer
   - Basic table + year range picker
4. Generate initial dataset (2006–2026 + 3 years ahead) for the 10 cities.
5. Decide on packaging: Ship base data with the plugin, or require one-time import?

## Relation to Existing Work

This approach is highly compatible with the Claude-style review concerns about accuracy vs. distribution. By moving computation offline, we get the best of both worlds: authoritative data + tiny runtime footprint.

It also complements (rather than replaces) the full Go web app for users who need arbitrary locations or live atmospheric adjustment.

---

*This document was created to explore the user's specific proposal for a pre-computed, database-backed WordPress plugin experience.*