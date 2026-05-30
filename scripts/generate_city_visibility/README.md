# Pre-computed City Visibility Data Generator

This directory contains tools to generate the offline datasets used by a potential minimal-footprint WordPress plugin (see `docs/wordpress-plugin-precomputed-data.md`).

## Purpose

Instead of running the heavy visibility renderer live inside WordPress, we pre-compute accurate Yallop-based visibility categories for a fixed list of major cities across many years. The resulting structured data (JSON/CSV) is imported into the WordPress database once.

This approach gives:
- Full accuracy (uses the reference C++/Go pipeline)
- Truly minimal WP plugin (pure data + display, no binaries, no exec)
- Easy historical views from 2006 onward

## Recommended Cities (initial set)

See the full list and rationale in `docs/wordpress-plugin-precomputed-data.md`.

Current working set:
- Jerusalem, Dallas, Melbourne (existing presets)
- Cairo, London, Tokyo, Rio de Janeiro, Cape Town, Mumbai, Istanbul

## Data Generation Workflow

1. Build the project normally (so the renderer binary exists):
   ```bash
   make
   ```

2. Run the generator (to be implemented):
   ```bash
   go run scripts/generate_city_visibility/main.go \
     --cities jerusalem,cairo,london,tokyo \
     --start 2006 \
     --end 2030 \
     --output docs/visibility-data.json
   ```

3. The generator will:
   - Use `internal/jobspec` to get accurate new moon dates per year.
   - For each new moon + each city, call the renderer in `point` mode for the following 3 days.
   - Record raw categories + compute effective under standard clear conditions.
   - Output a clean import file.

## Output Format

See `docs/wordpress-plugin-precomputed-data-example.json` for the current proposed structure.

## Integration with WordPress Plugin

The generated JSON/CSV is imported via a WP admin page into a custom table. The plugin then offers:
- City selector (10-12 options)
- 10-year window picker (starting 2006)
- Tables, heatmaps, or summary statistics for visibility likelihood

No live astronomy computation happens inside WordPress.

## Status

This is currently a planning + tooling skeleton. The actual `main.go` generator has not been written yet.

If we decide to pursue the pre-computed WP plugin path, the first real implementation task would be writing this generator so we can produce a high-quality initial dataset.