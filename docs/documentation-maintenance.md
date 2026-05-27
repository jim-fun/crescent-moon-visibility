# Documentation & Architecture Maintenance

**Crescent Moon Visibility Maps Generator — Public Process for Consistency**

This document defines the triggers, ownership, and Sync Checklist for the Documentation & Architecture Agent role. It ensures README, docs/*.md, TODO.md, CHANGELOG, and related files remain accurate and mutually consistent as architecture, platform support (e.g., Windows CPU), accuracy claims, and validation results evolve.

## Triggers for Mandatory Documentation & Architecture Review

Any PR or change hitting one or more of the following must include a Documentation & Architecture Agent pass (integrated with the project's 4-stage agentic workflow):

- Changes to visibility math (CPU/GPU kernels, `visibility.cc`, Yallop/Odeh logic, Chebyshev).
- Platform support changes (Windows instructions, Makefile detection, runtime.GOOS / renderer discovery in `main.go`).
- External validation harness results, ICOP/HMNAO baselines, or new accuracy data.
- Release process, packaging layout, or artifact changes (`release.yml`, `scripts/release.sh`).
- Architecture descriptions, performance numbers, or diagram updates.
- Any addition/modification to public documentation files.

## Sync Checklist (Enforced on Every Triggering Change)

The Documentation & Architecture Agent (human or subagent) must verify and apply updates to **all** of the following before Judge review. Use `todo_write` (or equivalent manual tracking) for multi-phase doc work. Always re-run `make`, `make test`, `make test-accuracy` (and `make validate-icop` when relevant) plus the anchor audit.

- **README.md**: Architecture, Setup & Installation (incl. Windows build subsection), Release Process, Testing & Validation, and any cross-refs updated. GPU dependency table reviewed for platform notes.
- **All three technical `docs/*.md` files** (`performance-accuracy.md`, `yallop-criteria-and-external-validation.md`, `visibility-criteria-comparison.md`): Footers ("Document version"/"status"/"Maintained by") refreshed with date + context; cross-refs added/updated (including to this document); new results or process changes recorded.
- **This file** (`docs/documentation-maintenance.md`): Itself updated if triggers or checklist evolve.
- **TODO.md**: Affected items moved to Completed, status text updated (e.g., Windows phases), "Last Updated" refreshed.
- **CHANGELOG.md** (under [Unreleased] or appropriate version) + release notes body in `.github/workflows/release.yml` (when release-related).
- **Makefile** comments (build targets, agentic workflow section, platform notes) + `make agentic-review` output.
- **`scripts/release.sh`** comments + the release/packaging workflow `.github/workflows/release.yml` body (when touched). Note: there is no separate `package.sh`; all release packaging and artifact policy (including the "GPU never shipped on any platform" restatement) lives in `release.yml`.
- **CONTRIBUTING.md**: Explicit reference to this public process added/updated.
- **Cross-references** to the `crescent-moon-visibility-engineering` skill preserved in public-facing locations.
- **Markdown anchor audit (mandatory explicit step)**: Verify *every* internal link of form `](#...` resolves to an existing heading. No dangling anchors (e.g., the pre-existing `#building-from-source` reference in README was repaired in PR 5 by updating the link target and adding the Windows subsection). Grep for `](#` + manual heading cross-check is sufficient.

## Public vs. Internal Split

**Public surface** (human contributors, external PRs, self-service): This document, README cross-refs, CONTRIBUTING.md, TODO.md, the three technical `docs/*.md`, and CHANGELOG. Follow the triggers + checklist above.

**Internal** (agentic sessions, maintainers): Full Documentation & Architecture Agent role definition, detailed prompts, subagent patterns, 4-stage workflow mechanics (Improvement → Validation → Security Review → Judge with Core Principles Scorecard), JUDGE_DECISION_TEMPLATE, and the authoritative `crescent-moon-visibility-engineering` skill remain in the private/maintainer workspace only (consistent with public mirror policy documented in TODO.md "Public GitHub Repository Restart & Documentation Polish (2026)" and `scripts/push-to-github.sh`).

Human contributors: Open issues or PRs referencing this checklist. Do not expect internal skill details in the public tree.

## Performing a Sync (Practical Guidance)

1. Identify trigger(s).
2. Apply `todo_write` (agentic) or manual checklist for the multi-file pass.
3. Update files per the Sync Checklist (anchor audit last).
4. Run verification commands.
5. Document the sync in the PR description and CHANGELOG.

See the approved roadmap (`docs/roadmap-execution-plan-f01edaab.md`, PR 5 and PR 6 sections) for full context — the Windows CPU Phase 3 example that introduced this process, and the Windows GPU Phase 4 example that exercised it by restating the local-build-only / never-shipped GPU policy across README, `release.yml`, `Makefile`, TODO, and CHANGELOG.

---
**Document status**: 2026-05 (introduced via PR 5 to codify the Documentation & Architecture Agent process and complete Windows CPU Phase 3 documentation; reaffirmed in PR 6 for Windows GPU Phase 4 — local-build-only, never shipped in releases on any platform — per roadmap-execution-plan-f01edaab.md).

**Maintained by**: Project maintainers + the crescent-moon-visibility-engineering skill and agentic workflow (especially the Documentation & Architecture Agent).
