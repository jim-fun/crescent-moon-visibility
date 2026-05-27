# HMNAO / UKHO Baseline Validation Data

This directory will contain curated excerpts from HM Nautical Almanac Office (HMNAO / UKHO) published lunar crescent visibility predictions and tables for comparison against the project's Yallop implementation.

## Purpose
Provide an independent published baseline (distinct from ICOP observational reports) for:
- First-visibility longitude boundaries
- ARCV / elongation thresholds per lunation
- Category boundary comparisons

This supports the external validation story (Accuracy First + Verifiability).

## Selection Criteria (Strict)
- Direct from official HMNAO/UKHO publications or almanacs (with year, page/table reference).
- Dates with clear geocentric or topocentric predictions.
- Preference for 2015–2026 range where possible for modern comparison.
- Only unambiguous tabulated values (no derived interpolations without source).

## Current Status
**PR 3 initialization** (roadmap-execution-plan-f01edaab). Directory and skeleton created. Real curated excerpts (10–20 lunations) and comparison harness logic to be added in the body of PR 3.

See `docs/yallop-criteria-and-external-validation.md` §4.3 and the main roadmap plan for targets.

## Format
(Planned) Small JSON or CSV + provenance notes. Initial entries will record date, predicted first-visibility longitude or conditions, source table, and our computed equivalent for delta analysis.

## Provenance
All data will be from official HMNAO/UKHO sources (almanacs, technical notes, or public tables) with full citation. No scraping of paywalled or non-attributed material.

## License / Attribution
Used under fair use / research context for scientific validation of the Yallop criterion implementation. Full attribution will be maintained in every record and in the yallop comparison document.

---

**Maintained as part of PR 3+ of the Accuracy First roadmap.** Follows the Documentation & Architecture Maintenance process.
