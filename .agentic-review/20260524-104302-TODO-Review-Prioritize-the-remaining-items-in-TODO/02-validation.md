# Validation Agent Output

**Date**: 2026-05-24
**Context**: Improvement Agent proposal for re-prioritizing TODO items, with strong emphasis on external validation work.

## Validation Steps Performed

1. Reviewed current automated test surface:
   - `make test` (unit tests in `internal/blend`, `main_test.go`)
   - `make test-accuracy` / `TestRendererAccuracy` (CPU vs GPU per-pixel match ≥96%)
   - No existing external data comparison harness

2. Evaluated the proposed "ICOP sighting comparison harness":
   - This would be new test infrastructure, not covered by existing renderer accuracy tests.
   - The existing `TestRendererAccuracy` only validates internal numerical consistency between CPU and GPU renderers. It would not catch systematic bias vs real-world observations.
   - Adding a golden sighting dataset + comparison harness is a net positive for Verifiability, but requires new test design.

3. Evaluated "Add additional criteria" proposal:
   - Feasible to add without breaking the current 96.97% gate if we extend the accuracy test matrix.
   - Risk: New criterion code paths could introduce subtle divergence between CPU reference and GPU if not implemented with the same discipline as Yallop/Odeh.

4. Evaluated "Golden sighting test dataset":
   - Excellent idea for long-term regression protection.
   - Would complement (not replace) the pixel-match test.

## Quantitative / Empirical Assessment

- Current test suite strength on Accuracy First: **Strong for internal consistency**, **Weak for external truth**.
- The 96.97% number is very valuable but is only a self-consistency metric.
- Adding real sighting data validation directly addresses the largest remaining gap in the project's accuracy claims.

## Assessment Against Core Principles

- **Accuracy First**: The Improvement proposal is strongly aligned. Prioritizing external validation work is the correct focus.
- **Performance with Integrity**: No proposed changes threaten performance or the existing fidelity bar.
- **Verifiability & Reproducibility**: Significantly improved by the proposed harness and golden dataset. This is one of the highest-leverage improvements for this principle.
- **Minimalism & Portability**: The validation work can be implemented with very low dependency footprint (data files + simple comparison logic in Go).

## Issues Discovered

- **Medium**: No automated way today to answer "how accurate is our Yallop implementation against reality?"
- **Low**: The older TODO items (web UI, projections, etc.) are still listed at Medium/Low priority even though they score much lower on the current principles than validation work.

## Recommendations

- Strongly support elevating external validation (ICOP + HMNAO) to High Priority.
- The "golden sighting dataset" idea should be implemented as part of the ICOP harness work rather than as a separate item.
- Deprioritize or move the older feature-addition items (terrain, extinction modeling, web UI, etc.) to a new "Future / Stretch Goals" section so the TODO stays focused.
- When the ICOP harness is built, it should be runnable via `make validate-external` or similar and produce clear pass/fail or statistical reports.

## Suggested Additional Tests

- New test target: `make validate-icop` (or equivalent) that can be run on demand with a curated sighting dataset.
- Extend `TestRendererAccuracy` to become a matrix that also exercises any new criteria added later.
- Add a small number of "known hard cases" (very young crescents, high-latitude cases) as golden tests.

Overall: The Improvement Agent's prioritization is sound and well-aligned with the project's stated principles. I recommend the Judge endorse the High Priority items with minor refinement on scope.