=== Young Crescent Moon Visibility ===
Contributors: (your-wporg-username)
Tags: moon, crescent, astronomy, hijri, lunar
Requires at least: 5.8
Tested up to: 6.7
Requires PHP: 7.4
Stable tag: 0.5.1
License: GPLv2 or later
License URI: https://www.gnu.org/licenses/gpl-2.0.html

Show pre-computed young-crescent-moon visibility (Yallop criterion) for cities and dates, with a live atmospheric-conditions adjuster — no runtime astronomy.

== Description ==

Young Crescent Moon Visibility adds an interactive tool that estimates how likely the thin new crescent moon is to be seen from a city on each of the first three evenings after conjunction.

All astronomy is computed **offline** with the project's reference Yallop renderer and imported as a compact dataset, so the plugin does no runtime calculation and works on ordinary shared hosting. The whole dataset for the page is embedded inline, so the tool also works on sites where /wp-admin is locked down.

**Features**

* Interactive shortcode: `[crescent_visibility_interactive]` (alias `[crescent_visibility_point]`).
* City / year / new-moon selectors; the city list is built from your imported data, with Jerusalem, Dallas and Melbourne always available and quick-select buttons.
* Per-evening result cards: raw category, weather-adjusted "effective" category, and that evening's moon age and Q value.
* Live atmospheric adjustment (cloud cover + transparency sliders) that re-grades the prediction in the browser.
* Small, non-interactive context map (Leaflet, bundled — no external script).
* Light and dark (`theme="dark"`) themes; server-rendered no-JavaScript fallback.
* A simple static table shortcode: `[crescent_visibility city="jerusalem" years="2026-2035"]`.

A small sample dataset (3 cities, 2026–2028) is imported on activation so the tool works immediately. Replace it with your own dataset under **Tools → Crescent Visibility**.

== External services ==

This plugin loads **map tiles from OpenStreetMap** (https://tile.openstreetmap.org) at runtime to draw the small context map. When the tool is viewed, the visitor's browser requests tile images for the selected city's area, which sends the visitor's IP address and the requested map area to OpenStreetMap. No other personal data is sent.

* OpenStreetMap Terms: https://operations.osmfoundation.org/policies/tiles/
* OpenStreetMap Privacy Policy: https://wiki.osmfoundation.org/wiki/Privacy_Policy

The map is cosmetic; everything else works without it.

== Installation ==

1. Upload the plugin and activate it. A small sample dataset is imported automatically.
2. Add `[crescent_visibility_interactive]` to any page or post.
3. (Optional) Generate a larger dataset with the project's offline generator and import it under **Tools → Crescent Visibility**.

== Frequently Asked Questions ==

= Does it calculate moon visibility on my server? =
No. All values are pre-computed offline with the Yallop reference renderer and imported as data. The browser only re-grades the pre-computed category for the cloud/transparency you choose.

= How do I add more cities or years? =
Generate a new JSON dataset with the project's generator (years via `--start`/`--end`, cities via `--cities`) and import it. The dropdown then reflects whatever the data contains.

= Where does the data come from? =
The Yallop crescent-visibility criterion, computed by the project's reference renderer. See the plugin homepage.

== Screenshots ==

1. The interactive tool: city/date selectors, atmospheric sliders, and three per-evening result cards.
2. Dark theme.
3. Tools → Crescent Visibility data import screen.

== Changelog ==

= 0.5.1 =
* Internationalize the full front-end UI and JavaScript; add translation template (.pot).

= 0.5.0 =
* Bundle Leaflet locally (no external CDN).
* Per-evening Age and Q values on every card.
* Country-sorted city dropdown derived from the data.
* Bundled sample dataset imported on activation; uninstall cleanup; import size guard.
* License set to GPLv2-or-later; internationalization of the front-end UI.

== Upgrade Notice ==

= 0.5.0 =
First public release.
