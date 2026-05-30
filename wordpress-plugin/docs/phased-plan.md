# Phased Plan: Minimal-Footprint WordPress Plugin for Pre-Computed City Visibility

**Project**: Crescent Moon Visibility – WordPress Plugin (Pre-computed Data Approach)  
**Date**: 2026-05-28  
**Goal**: Deliver a lightweight, installable WordPress plugin that displays accurate historical/future crescent visibility for a fixed set of major cities using pre-computed data (2006 onward, 10-year windows).  
**Primary Constraint**: Minimal runtime footprint on WordPress (no native binaries, no heavy computation at runtime).  
**Delegation Philosophy**: Follow the project's proven pattern — Ollama for volume/exploration/ideas, Claude for clean production code, Human (Grok + user) for direction, architecture, Accuracy First gatekeeping, and final integration.

---

## Success Criteria

- Plugin installs cleanly on standard WordPress (shared hosting friendly).
- Users can view visibility data for 8–12 major cities from 2006 onward.
- Data is verifiably generated from the project's reference renderer.
- Clear, honest presentation of what the data represents ("likelihood" is well-defined).
- Total human effort in the 25–50 hour range (AI doing the majority of coding and drafting).

---

## Phase Overview

| Phase | Focus | Human Hours (est.) | AI Load | Primary Risk |
|-------|-------|---------------------|---------|--------------|
| 0 | Scope Lock & Likelihood Definition | 6–10 | Low | Wrong metric chosen |
| 1 | Data Generator Tooling | 4–8 | High | Generator produces bad data |
| 2 | Generate & Validate Dataset | 4–6 | Medium | Data quality / provenance gaps |
| 3 | WordPress Plugin Foundation | 6–10 | High | Architecture that doesn't scale or violates WP standards |
| 4 | Import + Visualization UI | 6–10 | High | Poor UX for "likelihood" display |
| 5 | Docs, Polish & Release Prep | 4–8 | Medium | Incomplete documentation |
| 6 | Final Human Review & Accuracy Gate | 3–6 | Low | Last-minute accuracy or scope issues |

**Total Estimated Human Time**: 33–58 hours (realistic target: 35–45 hours with disciplined prompting).

---

## Phase 0: Scope Lock & Likelihood Definition (Human-Led)

**Objective**: Make the highest-stakes decisions before any code is written.

**Human Responsibilities** (you + Grok as integrator):
- Finalize exact list of cities (current proposed set is good starting point).
- Rigorously define "likelihood visibility" for a 10-year window.
  - Options to evaluate: Best category seen? % of windows with effective B or better? Average best category? Something else?
- Decide data granularity (per new moon + 3 days vs. yearly aggregates vs. both).
- Decide update model (how users will get future years of data).
- Confirm atmospheric assumption for pre-computed data (e.g., "standard clear skies").

**Ollama Role**: Low. Can be used for quick brainstorming of "likelihood" metric ideas if desired.

**Claude Role**: Low at this stage. Can be asked later to critique proposed definitions.

**Deliverables**:
- Locked city list (with coordinates).
- Written definition of "likelihood" (1–2 paragraphs + example calculations).
- Data schema v1.0 (fields per row).
- Decision log (why certain choices were made — important for provenance).

**Timebox**: 6–10 human hours + 1–2 days of thinking time.

**Exit Gate**: Human sign-off on the above before moving to Phase 1.

**Example Prompt (if using AI for options)**:
> "Act as a careful reviewer. We are building a WordPress plugin that shows crescent visibility likelihood for major cities over 10-year periods using pre-computed data. Propose 4–5 different ways to define and display 'likelihood' that are honest, observer-friendly, and not misleading. For each, list pros, cons, and what data would need to be stored."

---

## Phase 1: Data Generator Tooling (AI-Heavy)

**Objective**: Build a reliable, repeatable offline tool that produces the import files using the project's reference renderer.

**Division of Labor**:

**Ollama** (volume + exploration):
- First-draft generator script (multiple iterations).
- Ideas for command-line UX, error handling, progress reporting.
- Prototypes for different output formats (JSON, CSV, NDJSON).

**Claude** (production code):
- Refactor Ollama's drafts into clean, maintainable Go code.
- Proper use of existing `internal/jobspec` package.
- Robust error handling, logging, and validation.
- Final CLI design and documentation.

**Human**:
- Architecture review (does it reuse existing packages correctly?).
- Define exact output schema (must match what WP side expects).
- Spot-check generated data for a couple of cities/years against known good runs.
- Final approval of the generator.

**Specific Prompts**:

**Ollama Prompt (first iteration)**:
> "You are helping build a data generator for a WordPress plugin. We need to pre-compute crescent visibility using the existing crescent-moon-visibility tool for a fixed list of cities from 2006 onward. Write a Go program that:
> - Takes a list of cities (slug, name, lat, lon)
> - For each year, gets accurate new moon dates
> - For each new moon, runs the visibility renderer in 'point' mode for the following 3 days for each city
> - Records raw categories and computes effective category under standard conditions
> Output clean, importable JSON. Start with a simple working version."

**Claude Prompt (production version)**:
> "Review and rewrite the attached generator stub into production-quality Go code. It must:
> - Reuse `internal/jobspec` for new moon dates
> - Properly discover and execute the visibility renderer binary
> - Support command-line flags for cities, year range, output path
> - Include good error handling and progress reporting
> - Produce the exact JSON schema defined in [attach schema]
> - Be well-commented and follow the project's style
> Also write a short README for the script."

**Timebox**: 4–8 human hours (mostly review + running tests) + AI cycles.

**Deliverables**:
- Working `scripts/generate_city_visibility/main.go`
- Updated README in that directory
- Example command + sample (small) output

**Exit Gate**: Human can run the generator end-to-end for 1–2 cities and get valid output.

---

## Phase 2: Generate & Validate Master Dataset

**Objective**: Produce the actual high-quality dataset that will ship with (or be importable into) the plugin.

**Division of Labor**:

- **Ollama**: Help write validation scripts, spot-check scripts, or summary statistics generators.
- **Claude**: Write data validation + provenance tools (e.g., "generate a report showing coverage per city/year").
- **Human** (primary):
  - Actually run the generator (this can take significant CPU time).
  - Perform accuracy spot-checks against known good manual runs.
  - Write the provenance document (exactly which renderer binary + commit was used, atmospheric assumptions, etc.).
  - Decide final year range to ship (e.g., 2006–2028 with 3 years of projection).

**Timebox**: 4–6 human hours (mostly oversight + validation) + computer time.

**Important**: This phase is where Accuracy First is actively enforced by the human.

---

## Phase 3: WordPress Plugin Foundation (Claude Primary)

**Objective**: Create a proper, minimal, standards-compliant WordPress plugin skeleton.

**Division of Labor**:

**Claude** (main coder):
- Full plugin structure (`plugin-name.php`, `includes/`, `admin/`, `public/`).
- Activation/deactivation hooks.
- Custom table creation (or options-based storage decision).
- Admin menu + import page skeleton (file upload + basic validation).
- Security basics (nonces, capability checks, sanitization).

**Ollama** (exploration):
- Multiple variations of admin UI patterns.
- Ideas for storage (custom table vs. options + JSON vs. custom post types).
- First drafts of shortcode registration.

**Human**:
- Architecture decisions (storage strategy, update story, i18n approach).
- Security review of Claude's code.
- Decide on minimum supported WordPress version.
- Review against WordPress Plugin Handbook standards.

**Recommended Prompts**:

**Claude Prompt**:
> "Create a minimal but production-quality WordPress plugin for displaying pre-computed crescent visibility data. Requirements:
> - Single main plugin file + organized includes
> - Custom database table on activation
> - Admin page under Tools for importing JSON data
> - Basic shortcode `[crescent_visibility]` that accepts city and year-range parameters
> - Follow current WordPress coding standards and security best practices
> - Include uninstall cleanup
> Provide the full initial file structure and key code."

**Timebox**: 6–10 human hours.

**Exit Gate**: Plugin activates cleanly, creates its table, and has a working (if empty) admin import page and shortcode.

---

## Phase 4: Data Import + Visualization UI

**Objective**: Make the plugin actually useful — import real data and display it nicely.

**Division of Labor**:

**Ollama**:
- Exploration of different visualization approaches (tables, heatmaps, calendars, sparklines).
- Multiple UI mockups / first-pass JavaScript for the city selector + year range picker.
- Ideas for "likelihood" summary cards.

**Claude**:
- Production implementation of the chosen UI.
- Robust import logic (parsing, validation, bulk insert, progress feedback, error reporting).
- Shortcode / block rendering logic that queries the data and applies the "likelihood" rules.
- Basic Chart.js or pure CSS visualization (keep footprint small).

**Human**:
- Final choice of visualization approach (after seeing Ollama options).
- Review of import error handling and data integrity.
- Ensure the UI clearly communicates the limitations and provenance of the data.
- Accessibility and mobile review.

**Timebox**: 6–10 human hours.

**Deliverables**:
- Working import that can load the master dataset.
- Functional shortcode showing real data for at least one city + one 10-year window.
- Clear visual distinction between raw categories and effective categories.

---

## Phase 5: Documentation, Polish & Release Readiness

**Objective**: Make the plugin maintainable and credible.

**Division of Labor**:

**Ollama**:
- First drafts of user-facing documentation.
- Help text, tooltips, and FAQ content.
- Example usage scenarios.

**Claude**:
- Clean, accurate plugin README.md (following WordPress standards).
- Inline code documentation.
- Update mechanism notes.

**Human**:
- Write the critical "Data Provenance & Accuracy" section.
- Final review of all user-facing text for honesty.
- Decide on licensing, support policy, and update process for the data files.
- Create the initial release package checklist.

**Timebox**: 4–8 human hours.

**Also in this phase**: Update the main project `README.md` and `docs/basic-web-ui-design.md` (and the pre-computed data doc) to reference the new plugin.

---

## Phase 6: Final Human Review & Accuracy Gate

**Objective**: The last quality filter before any public release.

**Human-only** (with AI assistance only for mechanical tasks):

- Full end-to-end test on a fresh WordPress install.
- Manual review of a sample of the imported data against the generator output.
- Security audit (even light).
- Confirmation that Accuracy First and Verifiability are properly represented.
- Final decision: Is this ready to be published, or does it need another iteration?

**Timebox**: 3–6 hours.

---

## Overall Recommendations for Effective Delegation

1. **Always start phases with a strong human-written brief** (scope, constraints, success criteria).
2. **Use Ollama first** for exploration and volume when you want options or rough drafts.
3. **Use Claude second** for turning good ideas into clean, production code.
4. **Human review gates** after every major AI output — especially anything touching data accuracy or security.
5. **Document the generation environment** for every dataset release (this is non-negotiable for this project).
6. **Keep the plugin deliberately small** — resist feature creep. The value is in the trustworthy data + clear presentation, not in rich interactivity.

---

## Suggested Starting Order

1. Complete **Phase 0** (especially the "likelihood" definition) — do not skip.
2. Run **Phase 1** with the prompts above to get a working generator.
3. Generate a small test dataset (Phase 2 lite).
4. Proceed to Phase 3 only after you are happy with the data schema.

---

**Next Step**: If you approve this plan, we can begin with **Phase 0** immediately. I can help you draft the exact "likelihood" decision document and the city list finalization.

Would you like me to create a one-page "Phase 0 Brief" template you can fill out, or start drafting the likelihood metric options right now?