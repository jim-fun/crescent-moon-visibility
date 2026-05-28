# Basic Web UI Design for Crescent Moon Visibility Maps Generator

**Author**: Grok (following crescent-moon-visibility patterns)  
**Date**: 2026-05-27  
**Status**: Revised v2 (after Claude-style review feedback — 2026-05-27). Integration strategy clarified, CGO risk called out, Documentation Impact section added, form coverage expanded, job persistence nuance added. Ready for small spike.

**Current Implementation Status (late 2026)**: A working basic web server (`crescent_maps web`) has been implemented. The primary delivered feature is the **"Visibility for My Location"** observer tool (`/point`):
- Year → all new moons for that year dropdown with intelligent "next new moon" defaults
- Interactive Leaflet map + presets + geolocation
- Atmospheric conditions with heuristic Effective category adjustment
- 3-day result cards + clear A–E explanations and Raw vs Effective guidance
- Contextual map now shown on results page
- Supports `-tags=web` builds

The original broader map-generation job gallery remains future work. See the new section in README.md for usage.  
**Related**: TODO.md (Future / Stretch Goals), README.md, main.go, Makefile, docs/performance-accuracy.md

---

## Overview

This document outlines a **minimal, practical web UI** that makes the existing Crescent Moon Visibility Maps Generator accessible without requiring users to install Go, build C++/OpenCL components, or run commands.

The goal is to deliver a useful "generate and view maps" experience while preserving the project's core values: **Accuracy First**, verifiability (all maps still come from the exact same CPU/GPU renderer code), and minimalism.

This is scoped as a **basic / MVP** implementation. It deliberately stays far away from full client-side WASM computation or complex interactive visualization in v1.

---

## Motivation & Context

- The CLI tool (`crescent_maps`) is powerful but has a high barrier for occasional users, researchers, or educators.
- The project has now completed its major roadmap phase (external validation, golden regression, Windows support, documentation process). A web interface was listed as a Future / Stretch goal in TODO.md.
- Many users simply want to answer: "What will the new crescent look like in March 2027 for my location?" or "Show me the maps for 2027."
- A basic web UI provides immediate value while the heavy scientific computation remains in the battle-tested C++/Go code.

---

## Goals (MVP Scope)

- Allow a user to specify a year (or small year range) and optionally filter months.
- Trigger map generation using the existing high-accuracy renderer (CPU by default; GPU when available on the server).
- Display generated maps in a simple gallery or timeline view with clear date labeling.
- Provide basic controls and progress feedback.
- Let users download individual maps or all maps for a run.
- Keep the implementation small, maintainable, and aligned with the existing codebase.

---

## Non-Goals / Explicit Boundaries (MVP)

- No full client-side WASM port of the renderer (extremely complex due to CGO, OpenCL, astronomy.c, and large Chebyshev data).
- No interactive map (zoomable canvas with per-pixel data in the browser).
- No user accounts, saved generations, or sharing links in v1.
- No side-by-side criterion comparison UI (Yallop vs Odeh) — can be added later.
- No real-time streaming of individual pixels or live GPU rendering in the browser.
- Server will not accept completely unbounded year ranges without limits (resource protection).
- No new visibility math — all output continues to come from the exact same `cmd/visibility/visibility.cc` and `internal/blend`.

### Known Risk / Prerequisite (CGO Build Reliability)

As of late May 2026, the Linux CGO build path for the Go orchestrator has shown repeated failures in CI (`build-linux-amd64`) and local reproduction (symbol resolution errors for symbols from `thirdparty/astronomy.c` during `go build` with CGO). The CPU renderer builds cleanly, but the CGO-dependent orchestrator does not in some environments.

**Before investing significant effort in the web UI**, one of the following must be true:
- The CGO build fragility is resolved on the main Linux path, or
- The web UI plan explicitly documents that the server mode is a "best-effort / self-hosted only" feature with known deployment constraints on some Linux environments.

This risk should be re-evaluated before any Phase 2 work.

---

## Recommended Architecture

### High-Level Approach (Chosen for MVP)

**Internal Package + Subcommand Model (Preferred)**

Instead of shelling out to the binary, the web mode should be integrated into the existing Go orchestrator:

- Introduce a new subcommand: `crescent_maps web` (or `crescent_maps serve`).
- Extract reusable logic from `main.go` (task structs, renderer discovery via `getRendererCandidates`, job orchestration, GPU/CPU selection, day selection rules) into internal packages (e.g. `internal/server`, `internal/job`).
- The web server reuses the same high-accuracy paths as the CLI. No duplication and no version skew risk.
- This keeps the project as "one orchestrator binary that can act as CLI or lightweight server".

**Job Execution**
- For MVP, jobs can still be executed by calling the existing generation logic (refactored into a callable function) rather than `exec.Command`.
- The server manages a job queue, status, and output files.

**Frontend**
- Lightweight static site (HTML + Tailwind via CDN for zero-build start, or tiny Vite build later) + vanilla JS + HTMX.

**Why this approach?**
- Preserves 100% of the accuracy guarantees.
- Reuses all existing code paths (`main.go`, renderer selection, GPU detection, blending, day selection rules from `DaysToProcess` + 0.2% illumination logic).
- Much faster to deliver value than attempting WASM.
- Easy to self-host.
- Consistent with the project's architecture (one binary, multiple modes).

**Alternative Considered (Rejected for v1)**
- Pure client-side WASM + WebGPU/WebCL: Too complex, loss of OpenCL maturity, huge binary size, accuracy verification nightmare.
- Shelling out to the binary from inside the same process: Fragile (PATH issues, Windows `.exe` differences, signal handling, version skew during development).

---

## User Experience & UI Flow (Basic)

### Primary Flow (Happy Path)

1. **Landing / Home**
   - Clean hero section: "Crescent Moon Visibility Maps"
   - Short one-sentence description + link to scientific background.
   - Prominent "Generate Maps" button.

2. **Generation Form** (simple, not overwhelming)
   - Year range: `Start Year` + `End Year` (default: current year) — maps to `-start` / `-end` or `-years`
   - Optional: Months (comma-separated or multi-select) — maps to `-months`
   - Toggle: "Use GPU acceleration" (only shown/enabled if server reports GPU support) — maps to `-gpu`
   - Checkbox (advanced): "Raw overlays only (skip blending)" — maps to `-noblend` (useful when the server lacks the base map or blending deps)
   - Advanced (collapsible): 
     - Workers count (maps to `-workers`)
     - Custom output directory name (for reference; server manages actual storage)
   - "Start Generation" button with clear warning about time (multi-year runs can take minutes even on GPU) and the 3-day + 0.2% illumination rule (see Day Selection in README).

The UI should surface the exact CLI equivalent that will be executed (or the internal equivalent) for transparency and power users.

3. **Job Progress View**
   - Job ID / status
   - Simple progress bar or step list ("Selecting days", "Rendering day 1/3", "Blending", "Done")
   - Estimated time (rough)
   - Option to cancel (if implemented)

4. **Results View**
   - Title: "Maps for March–April 2027"
   - Grid or horizontal scrollable timeline of maps
   - Each map card shows:
     - Date / New Moon label
     - The WEBP image (click to enlarge/lightbox)
     - Download button
   - "Download All (ZIP)" button
   - Link back to "Generate another set"
   - Small note: "Generated with vX.Y.Z using [CPU|GPU] renderer"

5. **Secondary Pages / Sections**
   - "How it works" (links to existing performance-accuracy.md and yallop doc)
   - Legend explanation (static image or SVG matching the vector legend)
   - "Command line equivalent" (shows the exact CLI command that was run — great for transparency and power users)

---

## Proposed Backend API (Minimal)

Base path: `/api/v1`

- `POST /generate`
  - Request body (JSON):
    ```json
    {
      "start_year": 2027,
      "end_year": 2027,
      "months": [3,4],
      "use_gpu": false,
      "workers": 4
    }
    ```
  - Response: `{ "job_id": "uuid-or-short-id", "estimated_minutes": 3 }`

- `GET /jobs/{job_id}/status`
  - Returns: `{ "status": "running|completed|failed", "progress": 0.45, "current_step": "Rendering maps", "maps_generated": 2 }`

- `GET /jobs/{job_id}/maps`
  - Returns list of maps with URLs and metadata.

- Static file serving for generated outputs under a safe path (e.g. `/outputs/{job_id}/...` with proper cleanup).

Additional nice-to-have:
- `GET /health` (reports whether GPU is available on this server instance)
- `GET /version`

**Job Storage (MVP)**: For the absolute minimal start, simple in-memory + filesystem is acceptable, with the explicit limitation that jobs do not survive server restart. This is reasonable for self-hosted / internal use.

For a slightly stronger MVP, a tiny embedded approach (JSON file + file locking, or `modernc.org/sqlite` with no external dependencies) is preferred over pure in-memory. Cleanup of old generations remains mandatory.

---

## Frontend Technology Choices (Keep It Light)

- **HTML + Tailwind CSS** (via CDN for zero-build MVP, or a tiny Vite build later)
- **Vanilla JavaScript** + **HTMX** (for partial updates without writing a full SPA framework)
- No React/Vue/Svelte in v1 unless the team wants to adopt one.
- Responsive by default (works on desktop + tablet; phone is secondary)

This keeps the frontend bundle tiny and the whole thing easy to maintain by the existing Go-heavy team.

---

## Resource & Security Considerations (Critical)

Because map generation is computationally expensive:

- Hard limits on year span (e.g. max 5 years in one request for MVP).
- Rate limiting per IP or API key (even for self-hosted).
- Queue with bounded concurrency (don't run 10 generations at once on one GPU).
- Automatic cleanup of generated files.
- Input validation (reject obviously malicious year values).
- Consider requiring a simple "I understand this may take time" confirmation for large jobs.

For public demos, strongly consider a "demo mode" that only allows small single-year generations.

---

## Phased Implementation Plan (Suggested)

**Phase 1 – Skeleton (1–2 days of focused work)**
- Add a `web` subcommand (`crescent_maps web`) using internal package extraction from the existing orchestrator logic.
- Basic HTTP server + one end-to-end job path that reuses refactored generation code.
- Minimal static HTML form that hits the endpoint (plain form + full reload is acceptable to start).

**Phase 2 – Usable MVP**
- Progress polling + results gallery.
- Proper image serving + download links.
- Basic Tailwind styling and legend.
- Display of the exact command-line equivalent (or internal equivalent) used for the job.

**Phase 3 – Polish & Safety**
- Job cleanup policy, input limits, error states, "command line equivalent" display.
- GPU availability reporting (`/health`).
- Dockerfile / deployment notes for easy self-hosting.

**Phase 4 (optional later)**
- HTMX for smoother UX without a heavy framework.
- ZIP download.
- Better progress reporting.
- Simple theming / branding.

---

## Documentation & Architecture Impact

Adding a web UI is a significant new public surface and deployment story. Per `docs/documentation-maintenance.md`, this change triggers the full Documentation & Architecture Sync Checklist.

On the first real implementation PR, the following must be updated (in addition to this design doc itself):

- **README.md**: Architecture section (new "Web Server Mode" subsection), Usage examples, Testing & Validation, and cross-references.
- **TODO.md**: Update the web UI item status.
- **CHANGELOG.md**: New entry under [Unreleased].
- **Makefile**: New targets/comments if appropriate.
- All existing `docs/*.md` files that mention architecture or usage: footers + cross-refs refreshed.
- **Anchor audit** (mandatory explicit step).

The Documentation & Architecture Agent role must be involved before the PR reaches Judge review.

---

## "Visibility for My Location" (Point Query Mode) + Atmospheric Conditions

One natural and high-value future extension (explicitly requested) is allowing users to ask **"What is the visibility like for my specific area?"** — i.e., point-based predictions for an observer at a given lat/lon.

### Atmospheric / Weather Integration
It is **possible and planned** to incorporate weather and atmospheric conditions. The current astronomical model (Yallop/Odeh) does not yet include them, but we have begun laying groundwork:

- The point query form (`/point`) now accepts **Cloud Cover (%)** and **Transparency (1-10)** inputs.
- A real heuristic adjustment function (`applyAtmosphericAdjustment`) is implemented and active. It downgrades the astronomical category based on clouds and transparency.
- The result page now prominently displays both the raw astronomical category and the **Effective Visibility** after atmospheric conditions.
- Short-term improvements (next steps): nicer explanations, more atmospheric fields, auto weather lookup.
- Medium-term: Integrate real weather APIs (Open-Meteo recommended – free, no key required for basic use).
- Long-term: Extend the core C++ model with proper extinction physics.

A basic proof-of-concept for the UI + data capture is already implemented.

### Why This Matters
- The current core product excels at wide-area maps.
- Many users ultimately care about one specific location (their city, a favorite observing spot, etc.).
- The C++ renderer already has excellent support for this via "point" mode:
  ```
  ./visibility 2027-03-20 point 31.95 35.23 yallop
  ```
  This outputs precise category, q-value, ARCV, width, moon age, etc. for an observer at that lat/lon.

### Proposed Addition to Web UI (Phase 3+)
- A second mode/tab: **"My Location"** (or "Point Query").
- Inputs:
  - Date (single date or small range)
  - Latitude / Longitude (manual entry + "Use my current location" button via browser Geolocation API)
  - Optional: Altitude, criterion (Yallop default)
- Output:
  - Clear visibility category (A–E with color)
  - Best time for observation
  - Moon age at best time
  - q-value and other diagnostics (for more advanced users)
  - Simple explanation ("Easily visible to the naked eye", "May require binoculars", etc.)
  - Link to the corresponding full map for that day if it exists

### Technical Approach
- The web backend can exec the renderer binary in "point" mode (low overhead, reuses the exact same trusted code).
  - Proof-of-concept endpoint already added: `GET /point?date=2027-03-20&lat=31.95&lon=35.23`
- Or (better long-term): expose a clean Go wrapper around the point calculation.
- This feature pairs extremely well with future items like terrain/horizon modeling and observer factors.

This turns the web UI from "generate pretty maps" into a practical daily tool for observers.

---

## Open Questions

1. Should the initial implementation accept that jobs are best-effort and do not survive server restart (acceptable for self-hosted use), or invest in minimal persistence (e.g. JSON file + file locking) from the start?
2. Will there ever be an official hosted demo, or is this strictly for self-hosting?
3. How important is real-time progress vs. "started... done" for the basic version? (Note: structured progress from the C++ renderer is non-trivial.)
4. Should we support `-noblend` and custom output handling in the basic UI form, or keep the initial form deliberately minimal?
5. Relationship to future WASM work — should the web UI be designed to allow swapping in a WASM backend later?

---

## Next Steps (if this plan is approved)

1. Decide on the integration approach (internal packages + `crescent_maps web` subcommand) and document the decision here.
2. Add the explicit "Documentation Impact" section (done in this revision).
3. Create a small spike / prototype of the Go HTTP server skeleton + one refactored job path.
4. Re-evaluate the CGO build reliability risk before committing to Phase 2.
5. Link this document from TODO.md and README.md under Future Work.

---

## Alignment with Project Principles

- **Accuracy First**: All maps are still produced by the exact reference implementation. No shortcuts.
- **Verifiability & Reproducibility**: The UI will display the exact command-line (or internal) invocation used.
- **Minimalism & Portability**: Keeps new dependencies very low (std HTTP + minimal frontend).
- **Documentation**: This work will follow the existing Documentation & Architecture Maintenance process (see the dedicated section above).

This plan deliberately defers the hardest technical challenges (WASM) while delivering real user value quickly.

---

**Status**: Ready for discussion / refinement before any implementation begins.

---

## Post-Implementation Status Note (2026-05)

The core "Visibility for My Location" observer tool (`/point` handler) was implemented and stabilized as part of the basic web UI spike:

- Full year → list of all new moons for that year dropdown (powered by `jobspec.GetNewMoonsForYear` + stub approximation under `-tags=web`).
- Intelligent defaults: on first load with no params, the server computes the closest future new moon (current year + next year search) and pre-selects it; location defaults to Jerusalem (31.7683, 35.2137) with map centered and inputs pre-filled.
- Preset city buttons (Jerusalem, Dallas, Melbourne) + "Use my current location" (geolocation) + fully draggable Leaflet interactive map ("virtual glob") with live approximate sunset/moonset.
- Atmospheric conditions (cloud cover 0-100% + transparency 1-10) with `applyAtmosphericAdjustment` heuristic producing Effective Visibility category + plain-English note.
- 3-day result cards for the chosen new moon + following evenings, with raw vs. effective categories, graceful fallback for out-of-window dates.
- Error logging to `web_errors.log` for all `/point` and renderer invocations.
- Build path: `go run -tags=web . web` (or `go build -tags=web`) works on machines where the full CGO `internal/astro` path is broken; the non-web path remains unchanged for releases.

The implementation lives primarily in the large HTML/JS literal inside `handlePointQuery` in `main.go`. This matches the "server-driven hybrid" approach from the original plan but (as flagged in the Claude-style review) should eventually be refactored into reusable internal packages (`internal/web`, `internal/jobspec` already exists and is shared).

The `-tags=web` + `internal/astro/astro_stub.go` technique successfully bypassed the pre-existing CGO fragility for web UI development.

All work preserved Accuracy First (still shells out to the same reference renderer for point queries) and the Documentation Maintenance discipline.

Usage example (after `go run -tags=web . web`):

  Open http://localhost:8080/point
  → Year and New Moon dropdown default to the next upcoming new moon.
  → Latitude/Longitude and map default to Jerusalem.
  → Pick any new moon date or use a Quick location preset, adjust atmospheric sliders if desired, submit to see the 3-day effective visibility forecast.

Future work per the review: extract the point-query + jobspec logic so the web path does not rely on exec to `bin/visibility.out`.