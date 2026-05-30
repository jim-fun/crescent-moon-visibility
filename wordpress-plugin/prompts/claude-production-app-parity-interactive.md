# Claude Production Prompt — Crescent Visibility Interactive (App Parity)

> **Quick start for this task**  
> After logging into Claude Code (`claude login`), run:  
> `bash wordpress-plugin/scripts/feed-v3-v4-to-claude.sh`  
>  
> This script will feed this prompt + the two key exploratory drafts (v3 and v4) with the correct directory context.

---

**Role**: You are an expert WordPress plugin developer who also deeply understands the original Go web application in this repository. Your job is to produce **clean, production-grade, WP-coding-standards-compliant code** that delivers the full interactive "Visibility for My Location" experience from the web app inside the minimal-footprint precomputed-data WordPress plugin.

## Non-Negotiable Constraints

- **Zero runtime heavy computation or external binaries**. Everything must come from precomputed data already imported into MariaDB tables (`crescent_cities` and `crescent_observations`).
- **No CGO, no `exec()`, no image generation**. The plugin must work on ordinary shared hosting.
- **Accuracy First**: All visibility categories must trace back to the reference Yallop renderer via the offline JSON generator. The exact atmospheric adjustment heuristic from `main.go` must be preserved on both server and client.
- **Minimal footprint**: Prefer admin-ajax over full REST routes when possible. Keep the number of files low. No heavy dependencies.
- **Data comes from the existing import flow**. The plugin already has an admin import page that accepts the JSON produced by `wordpress-plugin/generator/generate_visibility_data.go`.

## Current State (What You Are Improving)

We have two recent high-volume exploratory drafts created with qwen3-coder:30b:

1. `wordpress-plugin/exploratory-drafts/qwen3-app-parity-interactive-v3.php` (the main UI + cards + heuristic + schema evolution)
2. `wordpress-plugin/exploratory-drafts/qwen3-ajax-defaults-dynamic-loading-v4.php` (the critical AJAX layer + smart defaults)

You must synthesize the best parts of both into clean, maintainable production code.

### Key Data Reality

After the v3 changes, the `crescent_observations` table (and import) now supports:
- `raw_day_0`, `raw_day_1`, `raw_day_2`
- `best_raw`, `best_effective`
- `q_at_best`, `moon_age_at_best`   ← these were the missing fields needed for app parity
- `data_version`

The existing `Crescent_Visibility_Plugin` class already has useful helpers (`get_available_years_for_city`, `get_new_moons_for_city_and_year`).

## Core User Experience to Replicate

The experience must feel like the web app's `/point` tool (minus any map/image generation):

- City selector (13 locked cities from Phase 0)
- Year dropdown → dynamically populates New Moon dates (the 3-day windows)
- Atmospheric sliders (Cloud Cover 0-100 + Transparency 1-10) with live or near-live updates
- Results: Three rich cards (Day +0, +1, +2) showing:
  - Large Effective category letter (with color)
  - Raw category
  - Age in hours + Q value (when available)
  - Plain-English explanation from the atmospheric heuristic
- Graceful handling of "J" / invalid renderer output (same logic as `main.go`)
- Small non-interactive context map (Leaflet via CDN is acceptable for now)
- Good defaults: Jerusalem + the closest/next new moon relative to today (the web app does this automatically)

## Exact Atmospheric Heuristic (must be preserved)

Copy the logic exactly from `main.go:applyAtmosphericAdjustment` (also present in the v3 draft). Both a PHP version and an identical JS version are required so the UI can update live without round-trips.

## Required Deliverables (in priority order)

1. **Main Plugin Updates** (`crescent-visibility.php` or split into logical includes)
   - Register the primary shortcode (recommend name: `[crescent_visibility_interactive]` or `[crescent_visibility_point]`)
   - Add the two AJAX actions (or one REST route + the ajax fallback)
   - Update `activate()` if needed for the new columns (already done in exploratory work)
   - Add a helper method for smart defaults (`get_smart_default_new_moon_for_city`)

2. **Public Renderer** (`plugin/public/interactive-renderer.php` or similar)
   - The full interactive form markup (clean, accessible, WP-friendly)
   - Server-side injection of smart defaults via data attributes or localized script
   - Include or require the heuristic function

3. **Frontend Assets**
   - One clean JS file (or two: core + ajax) that handles:
     - Dynamic loading of years and new moons using the AJAX endpoints
     - Live atmospheric adjustment on slider input (debounced)
     - Rich card rendering that matches the web app fidelity
     - Loading states and error handling
   - Optional: a small CSS file or scoped styles

4. **Admin / Documentation Updates**
   - Update the Tools page to mention the new interactive shortcode
   - Clear instructions for users who imported data before the q/age columns existed (re-import is the pragmatic solution)

## Styling Guidance

- The web app uses a dark zinc/Tailwind aesthetic (`bg-zinc-900`, `text-zinc-200`, etc.).
- For WordPress compatibility, provide a light default that looks good in most themes, plus a `data-theme="dark"` or class-based dark variant that closely matches the web app.
- Cards should be prominent, readable, and feel "app-like".

## Security & Quality Requirements

- All inputs sanitized and validated.
- Nonces on any AJAX that modifies state (read-only data endpoints can be more permissive but still validate).
- Proper `wp_send_json_success` / `wp_send_json_error`.
- Output escaping everywhere in the renderer.
- Follow WordPress Coding Standards.
- No direct SQL outside of the existing plugin class pattern.
- Graceful degradation when JavaScript is disabled (at minimum show a message directing users to the static shortcode or admin import status).

## File & Architecture Preferences

Keep the plugin structure simple:
- `crescent-visibility.php` (main class + shortcode registration)
- `plugin/public/interactive-renderer.php`
- `plugin/includes/ajax.php` or `includes/data.php` (optional)
- `plugin/assets/js/interactive.js` (and css if needed)
- Enqueue assets only on pages that actually contain the shortcode (use `has_shortcode` check or `do_shortcode` detection pattern).

## What Success Looks Like

A user installs the plugin, imports a real dataset (2026-2028 or larger), drops `[crescent_visibility_interactive]` on a page, and gets an experience that feels very close to visiting the web app's "Visibility for My Location" tool — but powered entirely by the precomputed database with zero external computation.

## Reference Materials You Should Use

- The two exploratory drafts mentioned above (v3 for UI/cards/heuristic, v4 for AJAX + defaults)
- `main.go` (especially `handlePointQuery`, `applyAtmosphericAdjustment`, and the result card HTML generation around line 1227+)
- `wordpress-plugin/data/visibility-2026-2028-real.json` (example data shape)
- `wordpress-plugin/docs/database-schema.md` (the more complete planned schema)
- `wordpress-plugin/generator/generate_visibility_data.go` (what fields the JSON contains)
- Existing `plugin/public/renderer.php` and the import logic for style consistency

## Output Instructions

Produce the complete set of updated or new files needed. For each file, give the full content.

At the end, include a short "Migration & Testing Notes" section for the human maintainer (how to test the AJAX, what to tell existing users about re-importing data, how to switch from the old static shortcode, etc.).

Begin.