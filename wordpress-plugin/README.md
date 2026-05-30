# WordPress Plugin – Pre-computed Crescent Visibility for Major Cities

This directory contains all work for building a **minimal-footprint WordPress plugin** that displays historical and near-future crescent moon visibility data for a fixed set of major cities.

## Philosophy & Constraints

- **Minimal runtime footprint**: The plugin must work on typical shared WordPress hosting. No native binaries, no `exec()`, no heavy computation at runtime.
- **Accuracy First**: All visibility data is generated offline using the project's authoritative C++/Go reference renderer (Yallop criterion).
- **Pre-computed data model**: We calculate results for ~10 major cities from 2006 onward. WordPress only stores and displays the data.
- **Delegation model**: We follow the project's established pattern:
  - **Ollama** (cheapest): Volume work, exploration, rough drafts, multiple options.
  - **Claude** (medium cost): Clean production code, refactoring, structured implementation.
  - **Human (Grok + user)** (most expensive): Direction, architecture, "likelihood" definition, Accuracy First gatekeeping, security/WP standards review, final integration.

## Folder Structure

```
wordpress-plugin/
├── data/                  # Pre-computed JSON/CSV datasets (generated offline)
├── generator/             # Go tooling to produce the datasets from the main project
├── plugin/                # The actual WordPress plugin code
│   ├── admin/             # Admin pages (import, settings)
│   ├── public/            # Frontend shortcode / block
│   └── includes/          # Core logic, database handling
├── docs/                  # Plugin-specific documentation
├── scripts/               # Helper scripts (import, validation, etc.)
└── README.md
```

## Key Documents

- `docs/phased-plan.md` — The full phased execution plan with Ollama/Claude/human division of labor.
- `docs/implementation-status.md` — Current progress across all phases (updated after each phase).

## Goals

Deliver a lightweight, credible WordPress plugin that:
- Lets users explore visibility data for major cities from 2006 onward.
- Uses 10-year windows (user-selectable, starting from 2006).
- Clearly communicates what the data represents and its limitations.
- Has an extremely small runtime footprint on WordPress.

## Current Branch

All work in this folder lives on the `wp-plugin-precomputed-cities` branch to avoid impacting the main `dev` work.

## How to Contribute (Delegation Flow)

1. Human defines scope / reviews critical decisions.
2. Ollama is used first for exploration and volume.
3. Claude is used for production-quality code.
4. Human reviews, gates, and integrates.

See `docs/phased-plan.md` for the detailed phase-by-phase breakdown and specific prompts.

---

**Status**: Basic import + static shortcode foundation complete. Full interactive "Visibility for My Location" experience (matching web-app `/point` tool UX) is now the active focus.

**Latest exploratory work (qwen3-coder:30b)**:
- v3: `exploratory-drafts/qwen3-app-parity-interactive-v3.php` (cards + heuristic + schema fix for Age/Q)
- v4 (targeted): `exploratory-drafts/qwen3-ajax-defaults-dynamic-loading-v4.php` (real AJAX + smart defaults: Jerusalem + next new moon)

**v0.2.0 Production Release**

The plugin is now packaged and ready:

- **Distributable zip**: `crescent-visibility.zip` (install via WordPress Plugins → Add New → Upload)
- **Bundled data**: Includes `data/visibility-2026-2028-real.json` so you can start immediately after activation.
- **Functional test**: `php wordpress-plugin/tests/test-interactive.php` (33 checks against real data — run from the repo root).

### Quick Verification After Install
1. Activate the plugin.
2. Go to **Tools → Crescent Visibility** and import the bundled `visibility-2026-2028-real.json` (or any generated file).
3. Add this shortcode to a page:
   ```html
   [crescent_visibility_interactive]
   ```
   (or the alias `[crescent_visibility_point]`)
4. You should see:
   - City selector defaulting to Jerusalem + the next new moon
   - Working year → new moon dropdowns
   - Live-updating 3-day cards when you move the atmospheric sliders
   - A small context map

For the full manual testing checklist (including AJAX and browser behavior), see `production-output/MIGRATION-AND-TESTING-NOTES.md`.

---

**Ready for production (development history)**: 
- `prompts/claude-production-app-parity-interactive.md` — comprehensive prompt synthesizing v3 + v4.
- `scripts/feed-v3-v4-to-claude.sh` — one-command launcher (run after `claude login`) that feeds the prompt + the two drafts to Claude Code for the merged production implementation.