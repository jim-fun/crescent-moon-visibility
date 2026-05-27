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
**PR 3 initialization complete** (roadmap-execution-plan-f01edaab). 
- Directory + harness skeleton created.
- Baseline comparison mode (`go run ./cmd/validate-icop --baseline=hmnao`) is live and improved.
- Research sources memo + pending example data added.

**Real curated HMNAO/UKHO data population** (10–20 lunations with quantitative deltas) is **explicitly deferred** per the project’s Option 3 decision. It will be addressed in a later phase of PR3. Current work is limited to the comparison harness and source research only.

See `docs/yallop-criteria-and-external-validation.md` §4.3 and the main roadmap plan for targets.

## Format
(Planned) Small JSON or CSV + provenance notes. Initial entries will record date, predicted first-visibility longitude or conditions, source table, and our computed equivalent for delta analysis.

## Provenance
All data will be from official HMNAO/UKHO sources (almanacs, technical notes, or public tables) with full citation. No scraping of paywalled or non-attributed material.

## License / Attribution
Used under fair use / research context for scientific validation of the Yallop criterion implementation. Full attribution will be maintained in every record and in the yallop comparison document.

---

**Maintained as part of PR 3+ of the Accuracy First roadmap.** Follows the Documentation & Architecture Maintenance process.

## Research Sources (2026-05-27 Grok web-assisted fallback after Claude auth blocker)

Primary public verifiable sources for HMNAO/UKHO lunar crescent visibility predictions (per-lunation guidance with city tables + global maps). No single multi-year table; releases are event-driven. All data for observer interpretation only. Full real curation of 10–20 lunations remains deferred per user Option 3 decision (taxonomy/map accepted as planning guidance; numeric population is later manual/web-assisted work inside the PR 3 body).

Key exact sources (Open Government Licence):
- Main hub: https://www.gov.uk/government/organisations/hm-nautical-almanac-office
- February 2026 example (new moon 17 Feb 2026 12:01 UT): https://www.gov.uk/government/publications/crescent-moon-visibility-for-february-2026
  - Summary PDF (5 pages, city tables + notes): https://assets.publishing.service.gov.uk/media/699c463931713b50fd49c022/Crescent_Moon_Visibility_summary_for_February_2026.pdf
  - Daily maps: F2026Feb17.pdf, F2026Feb18.pdf, F2026Feb19.pdf (same assets host)
- Equivalent pages/PDFs for March 2026 and May 2026 on the same GOV.UK organisation page.
- Historical pattern confirmed back to at least 2015 (e.g. 2018 Ramadan/Eid guidance on GOV.UK).
- For Ramadan 1446 / 28 Feb 2025 00:45 UT conjunction: Direct 2025 GOV.UK landing page not in current listings. Strongest public attribution via Moon Sighting UK (https://www.moonsighting.org.uk/moon/visibility-maps-cat.html), which publishes HMNAO-derived RAG maps for 1446 AH and references Yallop methodology (NAO Technical Note 69).

Contact for archives/custom data: CustomerServices@UKHO.gov.uk

See the full research memo and two concrete pending example rows (from the February 2026 PDF) in the session artifacts and in examples.json below. All proposed rows carry explicit "pending-manual-verification" status.

This note was added after the Claude CLI auth blocker ("Not logged in · Please run /login") and a strict Grok web-assisted fallback that passed the Accuracy First gate.
