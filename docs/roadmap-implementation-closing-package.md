# Roadmap Implementation — Final Hygiene & PR Opening Package

**Branch**: `roadmap-implementation/f01edaab`

This document was generated via Ollama gemma4:e4b delegation and strictly reviewed/corrected by Grok (Accuracy First gate + project fidelity). It contains only realistic, specific items for our actual state.

---

## Final Hygiene Sweep Checklist (Ready to Execute)

- [ ] **README.md**: Confirm the "Testing & Validation" section (including PR3 baseline mode and PR4 golden targets) and the PR prep artifacts links are up to date.
- [x] **TODO.md + docs**: Golden check language synchronized. It now runs unconditionally on Linux in the build matrix (job fails on mismatch). Documentation updated across TODO, README, PR prep artifacts, workflow comments, and Makefile to reflect current behavior + future branch-protection enforcement. Checklist item complete.
- [ ] **CHANGELOG.md**: Ensure the [Unreleased] section accurately reflects PR3 skeleton + improved baseline, PR4 native golden support, and the documentation process work.
- [ ] **PR Prep Artifacts**: Verify both `docs/roadmap-implementation-pr-body-draft.md` and `docs/roadmap-implementation-pr-creation-checklist.md` have consistent "Remaining/Deferred" language (especially Option 3 for HMNAO data).
- [ ] **Key Docs Cross-References**: Spot-check `docs/documentation-maintenance.md`, `docs/yallop-criteria-and-external-validation.md`, and `data/validation/*/README.md` files for accurate links to the new validation tools and PR prep files.
- [ ] **Code Comments**: Quick pass over the PR3 baseline mode and PR4 golden logic in `cmd/validate-icop/main.go` and the Makefile to ensure the "why" (Accuracy First, guarded updates, deferral) is clear.
- [ ] **Verification Commands**: Confirm the commands listed in the PR body draft and checklist (`make validate-icop-ci`, golden targets, `--baseline=hmnao`) are still accurate and working.
- [ ] **No Generic Debt**: Remove or move any lingering vague "TODO" comments that no longer apply after the roadmap work.

---

## PR Opening Notes (Ready to Copy)

**Suggested Title:**  
Roadmap Implementation: Consolidated PR1–PR8 (ICOP 100%, PR4 Golden Foundation, PR3 HMNAO Skeleton, Windows Parity, Documentation Process)

**Summary:**  
This PR merges the complete, consolidated implementation of the approved roadmap (PR1 through PR8) on a single branch.

Key deliveries include:
- PR2: Hardened ICOP validation harness with `InstrumentAwareMatch`, 12 real Ramadan 1446 records, **100% match rate** on the exact CPU renderer, `--report=json`, and golden support.
- PR3: HMNAO/UKHO comparison skeleton + research sources memo + live baseline mode (`--baseline=hmnao`). Real curated data population is explicitly deferred per Option 3.
- PR4: Native `--update-golden` support and guarded `validate-icop-golden-*` Makefile targets for strict regression protection of the 100% baseline.
- PR5/6: Windows CPU build support + comprehensive local-only GPU documentation and Makefile targets (never released).
- PR7/8: `validate-icop-ci` target, formalized documentation maintenance process, and multiple hygiene passes.

All changes have passed repeated verification (`make validate-icop-ci` at 100%). Working PR description and ready-to-execute checklist are in `docs/`.

This establishes a stable, verifiable foundation for future work while strictly honoring the project's Core Principles (Accuracy First non-negotiable).

**Key Links (for the PR description):**
- Working PR body draft: `docs/roadmap-implementation-pr-body-draft.md`
- PR Creation Checklist + suggested title/summary: `docs/roadmap-implementation-pr-creation-checklist.md`
- Final Hygiene & Opening Package: `docs/roadmap-implementation-closing-package.md`
- PR3 HMNAO sources & pending data: `data/validation/hmnao/README.md`
- PR4 Golden implementation: `cmd/validate-icop/main.go` + Makefile targets
- Documentation process: `docs/documentation-maintenance.md`

---

**Status**: Ready for use when opening the final consolidated PR from this branch. All content has been reviewed for accuracy against the actual work delivered.

---

## Suggested Commit Message (for the final merge to main)

```
Roadmap Implementation: Consolidated PR1–PR8 on single branch (f01edaab)

- PR1: Release tooling fixes (prerelease version bumping)
- PR2: Hardened ICOP harness + 12 real Ramadan 1446 records + 100% match on exact renderer + --report=json
- PR3: HMNAO/UKHO skeleton + research sources + improved --baseline=hmnao mode (real data deferred per Option 3)
- PR4: Native --update-golden + guarded golden-check/update Makefile targets
- PR5/6: Windows CPU support + local-only GPU docs & Makefile (never released)
- PR7/8: validate-icop-ci target + documentation maintenance process + hygiene

All changes verified at 100% on the ICOP foundation set. Prep artifacts included for future work.

Closes roadmap-execution-plan-f01edaab execution on consolidated branch.
```

---

## Polished One-Paragraph PR Summary (for GitHub PR body)

This PR merges the complete consolidated implementation of the approved roadmap (PR1–PR8) onto a single branch. It delivers a hardened ICOP validation harness with 100% match on 12 real Ramadan 1446 records, the PR3 HMNAO/UKHO baseline skeleton and research sources (real data population deferred per Option 3), native PR4 golden regression support with guarded Makefile targets, Windows CPU + local-only GPU build support, and the formal documentation maintenance process. All work has been repeatedly verified at 100% and is ready for the next phase of the project while strictly following Accuracy First principles.

**Working documents** (included in the PR):
- Body draft + checklist + closing package in `docs/`
```

---

**Next step recommendation**: When ready, create the PR from `roadmap-implementation/f01edaab` into `main` using the title and summary above, and link the three docs files in the PR description.