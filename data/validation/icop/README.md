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

This initial seed contains 8 sightings. Future increments may grow this to 30–50 entries.

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

All data originates from public ICOP reports (https://www.icoproject.org). Only a tiny curated subset is included here for reproducibility.

## License / Attribution

This curated subset is provided under the same spirit as the original ICOP data — for scientific and educational use in lunar crescent visibility research.

Do not use this dataset for commercial purposes without contacting the original observers or ICOP.