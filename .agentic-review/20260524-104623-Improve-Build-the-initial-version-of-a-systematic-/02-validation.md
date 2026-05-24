# Validation Agent Output

**Date**: 2026-05-24
**Improvement Proposal**: Initial ICOP External Validation Harness (MVP)

## Validation Steps Performed

1. Reviewed the current project test surface:
   - `make test` (unit tests)
   - `make test-accuracy` (CPU vs GPU pixel match)
   - No existing external data validation

2. Inspected the proposed implementation:
   - New package `internal/validation/icop`
   - New command `cmd/validate-icop`
   - Data-only approach (no renderer modification)
   - Uses existing `internal/astro` CGO bindings for ephemeris

3. Built and ran the initial stub:
   - Compiles cleanly
   - Loads and validates the curated dataset successfully
   - No impact on existing tests

4. Assessed integration risk:
   - The harness is completely isolated from the renderers.
   - It will eventually call into astronomy calculations (via CGO) but does not modify them.
   - Adding this does not risk the 96.97% CPU/GPU regression gate.

## Quantitative Assessment

- Current state of external validation: **None**
- Proposed harness addresses the #1 gap identified in the previous agentic review.
- The MVP scope (curated dataset + basic comparison) is appropriately small and low-risk.

## Assessment Against Core Principles

- **Accuracy First**: Strongly positive. This is the most direct way to start proving (or challenging) our Yallop/Odeh predictions against real data.
- **Performance with Integrity**: Neutral. No changes to hot paths.
- **Verifiability & Reproducibility**: Excellent. A small, committed dataset + deterministic Go code is highly reproducible.
- **Minimalism & Portability**: Good. Keeps new surface area very small.

## Issues / Risks Discovered

- **Medium (mitigable)**: The full accurate Yallop calculation requires the same logic as `visibility.cc`. We must be careful not to create a second, divergent implementation. Recommendation: either share code or very carefully document that the harness uses a Go re-implementation of the published Yallop formulas.
- The current stub is honest about being incomplete.

## Recommendations to Improvement Agent / Judge

- Proceed with the MVP.
- In the next increment, connect real calculation logic.
- Strongly prefer using the existing Astronomy Engine + re-implementing only the Yallop q-test math in Go (rather than trying to drive the full C++ renderer for every validation run).
- Add a basic "make validate-icop" target (already done in the proposal).

Overall: The proposal is sound, low-risk, and correctly targeted at the project's biggest current weakness. I support moving forward.