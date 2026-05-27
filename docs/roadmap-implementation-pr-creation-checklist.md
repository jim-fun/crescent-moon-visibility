# Roadmap Implementation PR — Creation Checklist & Suggested Messaging

**Branch**: `roadmap-implementation/f01edaab` (single consolidated branch)

This document was generated via Ollama gemma4:e4b delegation and strictly reviewed/corrected by Grok (Accuracy First gate). It contains only project-specific, verified items.

---

## PR Creation Checklist (Ready for Execution)

- [ ] Confirm you are on `roadmap-implementation/f01edaab` and it is up-to-date with all recent PR4/PR8 changes.
- [ ] Run final verification: `make validate-icop-ci` (must report 100% on the committed ICOP set).
- [ ] Run golden regression check: `make validate-icop-golden-check ICOP_MIN_MATCH_RATE=100`.
- [ ] Smoke test PR3 baseline mode: `go run ./cmd/validate-icop --baseline=hmnao`.
- [ ] Verify the new PR4 targets work (guarded update + check).
- [ ] Ensure the polished PR body draft is current: `docs/roadmap-implementation-pr-body-draft.md` (update with any last-minute notes).
- [ ] Update cross-references in `TODO.md` and `CHANGELOG.md` (reference this checklist and the PR body draft).
- [ ] Run `go build ./cmd/validate-icop` (confirm clean build of the orchestrator).
- [ ] Follow the full Documentation Maintenance Sync Checklist in `docs/documentation-maintenance.md` (anchors, provenance, footers).
- [ ] Review the PR body draft for accuracy one final time (Accuracy First language, Option 3 deferral for HMNAO data, real verification commands only).
- [ ] Prepare the actual GitHub PR from this branch using the suggested title + summary below.
- [ ] After opening: add any required reviewers and link to the key artifacts (this checklist, the PR body draft, the golden file, hmnao research sources).

---

## Suggested PR Title & Summary (Ready to Copy)

**Title:**  
Roadmap Implementation: Consolidated PR1–PR8 (ICOP 100%, PR4 Golden Foundation, PR3 HMNAO Skeleton, Windows Parity, Documentation Process)

**Summary:**  
This PR merges the complete, consolidated implementation of the entire approved roadmap (PR1 through PR8) onto a single branch.

Key deliveries:
- PR2: Hardened ICOP validation harness with `InstrumentAwareMatch`, 12 real Ramadan 1446 records, **100% match rate** on exact CPU renderer, `--report=json`, and golden support.
- PR3: HMNAO/UKHO comparison skeleton + research sources memo (with explicit Option 3 deferral for full data curation).
- PR4: Native `--update-golden` support + guarded `validate-icop-golden-*` Makefile targets for strict regression protection.
- PR5/6: Windows CPU build + local-only GPU support (never released).
- PR7/8: `validate-icop-ci` target, documentation maintenance process, and hygiene.

All changes have passed repeated verification (`make validate-icop-ci` at 100%). The PR body working draft is in `docs/roadmap-implementation-pr-body-draft.md`.

This establishes the stable foundation for future work while strictly following the project's Core Principles (Accuracy First non-negotiable).

---

**Status**: Ready for use. When the actual PR is created from `roadmap-implementation/f01edaab`, link this checklist + `docs/roadmap-implementation-pr-body-draft.md` + `docs/roadmap-implementation-closing-package.md` in the description.

All items above are derived from verified work on the current branch (100% ICOP, PR4 golden targets native and guarded).