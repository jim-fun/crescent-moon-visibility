# Improvement Agent Output — TODO Prioritization Review

**Date**: 2026-05-24
**Area reviewed**: Remaining items in TODO.md, prioritized strictly against the four Core Principles (Accuracy First dominant).

## Summary

After reviewing the current TODO.md against the project's Core Principles, I am proposing a re-prioritization and some new/refined items. The current list still contains many medium/low priority feature additions that were defined before the major 2026 engineering discipline work (dual renderers, 96.97% accuracy bar, agentic workflow, professional releases).

The highest-leverage work right now is anything that strengthens **external, reproducible validation** of the existing high-fidelity Yallop + Odeh implementation. We now have excellent internal validation (`TestRendererAccuracy`) and a detailed comparison document (`docs/yallop-criteria-and-external-validation.md`), but almost no systematic external grounding against real observations.

I am ruthless: many of the older "Medium/Low" items (web UI, multi-language, configurable projections) are nice-to-have but do not meaningfully advance Accuracy First or Verifiability in the near term. They should be deprioritized or moved to a "Future / Stretch" section.

## Curated TODO Items (ready for TODO.md)

### High Priority (Accuracy First + Verifiability)

- [ ] **High** - Build systematic external validation harness against ICOP sighting database
  Rationale: We have a high-quality computational implementation of the Yallop (1997) q-test and Odeh (2004) criteria, plus a detailed comparison document. However, we lack quantitative evidence of how well the current predictions match real-world naked-eye and telescopic observations. This is the single largest gap in the "Accuracy First" claim.
  Ties to Core Principles: Directly strengthens Accuracy First (non-negotiable) and Verifiability & Reproducibility. Without this, all internal 96.97% numbers are only self-consistency, not external truth.
  Suggested validation: Curated set of 50–100 ICOP positive/negative sightings from 2015–2025. Automated comparison script that ingests sighting reports (lat/lon, date, instrument, success/failure) and runs the renderer at those locations/times. Report precision/recall per category (A/B vs C/D/E) and confusion matrix. Add as `make validate-icop` or similar.

- [ ] **High** - Add HMNAO / UKHO lunar crescent visibility predictions as a comparison baseline
  Rationale: HMNAO publishes official predictions using (a version of) the Yallop method. Comparing our maps against their published tables for the same dates provides an independent implementation check and increases credibility.
  Ties to Core Principles: Strong Verifiability & Reproducibility + Accuracy First. Helps prove our Chebyshev + rise/set + Yallop logic produces equivalent results to the official source.
  Suggested validation: Manually or semi-automatically compare 10–20 dates against published HMNAO visibility tables. Document any systematic differences in first-visibility longitude or category boundaries.

### Medium Priority (supporting Accuracy + Performance)

- [ ] **Medium** - Implement at least one additional modern visibility criterion (start with Odeh as baseline, then Schaefer or a recent paper)
  Rationale: Having only Yallop + Odeh limits users who prefer other calibrated methods. Adding a third (especially a more physically-based one) increases the tool's scientific utility without compromising the primary Yallop path.
  Ties to Core Principles: Supports Accuracy First by allowing direct side-by-side comparison on the same dates. Improves Verifiability.
  Suggested validation: New `TestRendererAccuracy` variant that also exercises the new criterion. Ensure CPU/GPU match remains ≥96% for the new path as well.

- [ ] **Medium** - Create golden sighting test dataset and regression harness
  Rationale: As we add new criteria or modeling improvements, we need reproducible test cases that protect the accuracy bar. A small, curated, version-controlled set of "known good" real-world sighting outcomes (with location, time, instrument, and result) would be extremely valuable.
  Ties to Core Principles: Excellent for Verifiability & Reproducibility and long-term protection of Accuracy First.
  Suggested validation: 20–30 high-quality ICOP or published sightings stored as JSON/CSV. Simple Go test or script that runs the renderer and asserts category or first-visibility time is within tolerance.

### Low / Deprioritize or Move to Stretch

(The following are valuable eventually but score lower on Accuracy First and Verifiability right now compared to validation work.)

- Terrain elevation, atmospheric extinction modeling, observer experience factors — move to "Future Modeling Enhancements" section. These are important for real-world fidelity but should come after we have proven the current model against observations.
- Web UI, real-time notifications, multi-language — these are usability / distribution features. They do not strengthen the core accuracy claims.

## Assessment Against Core Principles

- **Accuracy First**: Strongly addressed by elevating external validation work.
- **Performance with Integrity**: No regression risk proposed.
- **Verifiability & Reproducibility**: Significantly improved by the proposed validation harness and golden dataset.
- **Minimalism & Portability**: The validation work can be done with minimal new dependencies (pure Go + data files).

## Risks / Trade-offs

- External validation work requires sourcing and cleaning real sighting data (non-trivial effort).
- Risk of discovering that current predictions have systematic biases vs real observations (this would actually be valuable information, not a failure).

## Suggested Validation Steps for Validation Agent

- Review whether `make test-accuracy` and the existing unit tests would catch issues in a new ICOP comparison harness.
- Check that adding new criteria paths would not break the existing 96.97% CPU/GPU regression gate.
- Confirm that the proposed golden dataset approach is maintainable.

This prioritization is deliberately focused on protecting and strengthening the project's hardest-won achievement: a trustworthy, high-accuracy implementation of established visibility criteria.