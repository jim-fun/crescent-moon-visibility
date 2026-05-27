# ICOP Curated Validation Dataset

This directory contains a small, carefully curated subset of real sighting reports from the Islamic Crescents' Observation Project (ICOP).

## Purpose

This dataset exists to support the external validation of the Crescent Moon Visibility Maps Generator's Yallop and Odeh implementations.

**Important**: This is **not** a full dump of the ICOP database. It is a deliberately small, high-quality, version-controlled collection used for regression testing and accuracy analysis.

## Selection Criteria (Strict)

Entries in this dataset must meet **all** of the following:

1. **Clear provenance** — Direct report to ICOP with verifiable observer details.
2. **Good observing conditions** — Observer explicitly states skies were clear or mostly clear.
3. **Instrument clearly recorded** — naked_eye, binoculars, or telescope.
4. **Definite outcome** — Either clearly "seen" or clearly "not seen". Ambiguous reports are excluded.
5. **Geographic diversity** — Preference for different latitudes and longitudes.
6. **Recent enough** — Focus on 2020 onward where possible (better modern reporting standards).

## Current Size

This PR 2 foundation set contains **12 real, high-quality, conjunction-aligned ICOP records** (Feb 28 + Mar 1 2025 Ramadan 1446 lunation, new moon 2025-02-28 00:45 UT).

All entries drawn from public reports on https://astronomycenter.net/icop/ram46.html?l=en with explicit observer, location, sky conditions, instrument, and outcome. Mix: 9 naked_eye + 3 aided (binoculars/telescope); 9 "seen" + 3 "not_seen"; young marginal (10–24 h) + easier (36–42 h) cases.

Future increments (PR 3/4) may grow this to 30–50 entries across multiple lunations.

## Format

See `sightings.json` for the schema.

## Usage

Run the validation harness:

```bash
make validate-icop
```

Or directly:

```bash
go run ./cmd/validate-icop
```

## Provenance

All data originates from public ICOP reports at https://astronomycenter.net/icop/ram46.html?l=en (Ramadan 1446, conjunction 2025-02-28 00:45 UT). Only a tiny curated subset meeting the strict criteria is included here for reproducibility and regression.

## PR 2 Validation Harness Results (on this dataset)

Run via `make validate-icop` (exact CPU renderer "point" mode + InstrumentAwareMatch):

- **100.0% match rate** (12/12)
- naked_eye: 100% (9/9)
- aided: 100% (3/3)
- Mean moon age (exact, renderer best-time): 24.4 h

One marginal naked-eye "seen" on predicted B (Algeria south) correctly accepted. Three difficult young aided sightings on D correctly accepted. Non-sightings on F correctly accepted.

This demonstrates the hardened harness + real aligned data now produces trustworthy external evidence (Accuracy First). See docs/yallop-criteria-and-external-validation.md for full analysis (PR 4).

## License / Attribution

This curated subset is provided under the same spirit as the original ICOP data — for scientific and educational use in lunar crescent visibility research.

Do not use this dataset for commercial purposes without contacting the original observers or ICOP.