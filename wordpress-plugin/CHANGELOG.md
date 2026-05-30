# Changelog – Crescent Visibility (WordPress Plugin)

## [0.2.0] – 2026-05-29

### Added
- Full interactive "Visibility for My Location" experience matching the web app `/point` tool (no image generation).
  - City selector (13 locked cities, loaded from imported data with fallback)
  - Year → dynamic New Moon selector
  - Atmospheric sliders (cloud cover + transparency) with live client-side re-grading
  - Rich 3-day result cards (Raw + Effective + Age hours + Q value + plain-English explanations)
  - Small on-demand Leaflet context map
  - Server-rendered `<noscript>` fallback
  - Dark theme support (`theme="dark"` attribute)
- Production-grade AJAX endpoints (`cvi_get_years`, `cvi_get_newmoons`) with nonce protection.
- Smart defaults: automatically selects Jerusalem + the next future (or most recent) new moon.
- Exact port of the atmospheric adjustment heuristic from `main.go` (both PHP and JS versions).
- Automatic schema upgrade for the `q_at_best` / `moon_age_at_best` / `data_version` columns.
- Functional test harness: `tests/test-interactive.php` (runs against real 2026-2028 dataset, 33 checks).

### Packaging
- Clean distributable zip: `crescent-visibility.zip` (correct WordPress plugin layout).
- Bundles `data/visibility-2026-2028-real.json` for immediate use after activation.
- Excludes unused exploratory files.

### Changed
- Main plugin version bumped to 0.2.0.
- Admin Tools page now prominently documents the new interactive shortcodes:
  - `[crescent_visibility_interactive]`
  - `[crescent_visibility_point]`
- Improved documentation around auto-upgrade vs. re-import for historical Age/Q data.

### Fixed
- Multiple fidelity gaps vs. the original web app (heuristic behavior for "J"/unmapped values, bad-window threshold `age > 100`, color palette).
- Asset URL bug in renderer.
- Proper nonce validation on all AJAX calls.

### Notes
- All production PHP files pass `php -l`.
- The new functional test (`php wordpress-plugin/tests/test-interactive.php`) passes completely in the source tree.
- For live WordPress testing (AJAX round-trips, Leaflet, slider behavior), see `production-output/MIGRATION-AND-TESTING-NOTES.md`.

---

## [0.1.0] – Earlier

Initial release with basic import and static `[crescent_visibility]` shortcode.