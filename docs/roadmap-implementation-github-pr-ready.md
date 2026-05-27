# GitHub PR — Ready to Open

**From branch**: `roadmap-implementation/f01edaab`
**Into**: `main`

---

## Suggested PR Title

Roadmap Implementation: Consolidated PR1–PR8 (ICOP 100%, PR4 Golden Foundation, PR3 HMNAO Skeleton, Windows Parity, Documentation Process)

---

## PR Body (Copy-Paste Ready)

This PR merges the complete, consolidated implementation of the approved roadmap (PR1 through PR8) onto a single branch.

### Key Deliveries

- **PR2**: Hardened ICOP validation harness with `InstrumentAwareMatch`, 12 real Ramadan 1446 records, **100% match rate** on the exact CPU renderer, `--report=json`, and golden support foundation.
- **PR3**: HMNAO/UKHO comparison skeleton + research sources memo + live baseline mode (`--baseline=hmnao`). Real curated data population is explicitly deferred per Option 3.
- **PR4**: Native `--update-golden` support and guarded `validate-icop-golden-*` Makefile targets for strict regression protection of the 100% baseline. The golden check now runs on every Linux build in the CI matrix (no continue-on-error; mismatch fails the Linux job). **Long-term goal (explicitly tracked)**: make the Linux build (or dedicated golden job) a required status in GitHub branch protection for `main`.
- **PR5/6**: Windows CPU build support + comprehensive local-only GPU documentation and Makefile targets (never released).
- **PR7/8**: `validate-icop-ci` target, formalized documentation maintenance process, and multiple hygiene passes.

All changes have passed repeated verification (`make validate-icop-ci` at 100%).

This establishes a stable, verifiable foundation for future work while strictly honoring the project's Core Principles (Accuracy First non-negotiable).

### Working Documents (included in this PR)

- Final PR description: `docs/roadmap-implementation-final-pr-description.md`
- Body draft: `docs/roadmap-implementation-pr-body-draft.md`
- Ready-to-execute checklist: `docs/roadmap-implementation-pr-creation-checklist.md`
- Final hygiene + opening notes package: `docs/roadmap-implementation-closing-package.md`

### How to Review / Test Locally

1. `git checkout roadmap-implementation/f01edaab`
2. Core validation: `make validate-icop-ci`
3. Golden regression check: `make validate-icop-golden-check ICOP_MIN_MATCH_RATE=100`
4. PR3 HMNAO baseline: `go run ./cmd/validate-icop --baseline=hmnao`
5. Golden update (guarded): `ICOP_GOLDEN_UPDATE=1 make validate-icop-golden-update`

All changes must pass `make validate-icop-ci` and follow the documentation maintenance process in `docs/documentation-maintenance.md`.

---

## Suggested Labels (optional)

- `roadmap`
- `consolidated-pr`
- `accuracy`
- `documentation`

## Suggested Reviewers

(You can add any core reviewers here when opening)

---

**Status**: This package is ready to be used when creating the PR on GitHub. Copy the title and body above directly into the PR form.

### Known Remaining / Deferred Items (documented for transparency)

- Real curated HMNAO/UKHO data population (PR3) — explicitly deferred per Option 3 decision.
- Making golden regression a required pre-merge status check via GitHub branch protection (currently runs and fails Linux jobs on mismatch; tracked as next enforcement step).

---

## Suggested Merge Commit Message (for when merging this PR into `main`)

```
Roadmap Implementation: Consolidated PR1–PR8 (f01edaab)

Merges the complete approved roadmap execution onto a single branch.

- PR1: Release tooling fixes
- PR2: ICOP 100% with 12 real records + hardened harness
- PR3: HMNAO/UKHO skeleton + baseline mode (real data deferred per Option 3)
- PR4: Native guarded golden regression targets
- PR5/6: Windows + local GPU support (never released)
- PR7/8: validate-icop-ci + documentation process + hygiene

All changes verified at 100%. Prep artifacts included for future work.

Closes roadmap-execution-plan-f01edaab execution phase.
```