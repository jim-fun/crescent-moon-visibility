# Improvement Agent Output

**Date**: 2026-05-24
**Task**: Build the initial version of a systematic external validation harness against the ICOP sighting database, starting with a small curated dataset and comparison script. Focus on Accuracy First and Verifiability.

## Summary of Proposed Improvement

I propose creating a minimal, reproducible **ICOP External Validation Harness** as the first concrete step toward closing the largest remaining gap in our Accuracy First claim.

This initial version will include:
- A small, curated, version-controlled dataset of real ICOP sightings (starting with ~25–40 high-quality entries).
- A simple Go-based comparison tool (`cmd/validate-icop`).
- Basic data structures and loader in `internal/validation/icop`.
- A Makefile target to run the harness.
- Clear documentation on data provenance and selection criteria.

The goal for this increment is **not** a full-featured database importer, but a working, trustworthy foundation that can be extended later.

## Detailed Design

### 1. Directory Structure

```
data/validation/icop/
    sightings.json          # Curated subset of real sightings
    README.md               # Provenance, selection rules, format spec

internal/validation/
    icop/
        icop.go             # Data models + loader + basic comparison logic

cmd/validate-icop/
    main.go                 # CLI entrypoint

Makefile                    # Add `validate-icop` target
```

### 2. Data Format (Initial)

Simple, human-readable JSON:

```json
[
  {
    "id": "ICOP-2025-03-29-042",
    "date": "2025-03-29",
    "latitude": 31.7683,
    "longitude": 35.2137,
    "instrument": "naked_eye",
    "reported_result": "seen",
    "notes": "Clear skies, experienced observer, ~10 min after sunset"
  }
]
```

`instrument` values: `naked_eye`, `binoculars`, `telescope`

`reported_result`: `seen`, `not_seen`

### 3. Tool Behavior (MVP)

```bash
make validate-icop
# or
go run ./cmd/validate-icop --dataset data/validation/icop/sightings.json --method yallop
```

Output example:
```
ICOP Validation Harness (Yallop)
Dataset: 32 sightings
Overall match rate: 78.1% (25/32)

By category:
  A/B (naked eye expected):  91% match
  C/D (aided expected):      65% match

Mismatches: 7
  - 3 false negatives (predicted F but reported seen)
  - 4 false positives (predicted A/B but not seen)
```

### 4. Core Principles Evaluation

- **Accuracy First** (Highest priority): This work directly attacks the biggest weakness in our current accuracy story. We have strong internal consistency but almost no external grounding. This is the correct next step.
- **Verifiability & Reproducibility**: Excellent. A small curated dataset + deterministic Go code is highly reproducible.
- **Performance with Integrity**: Neutral. This is post-processing only. No risk to the renderer fidelity.
- **Minimalism & Portability**: Very good. Pure Go + JSON data files. No new runtime dependencies.

### Risks & Trade-offs

- Data quality risk: ICOP data varies in reliability. Mitigation: Strict curation rules (documented in README) + small initial set.
- Scope creep: Easy to want to support every field ICOP collects. We will deliberately start minimal.

### Concrete Implementation Plan (This Increment)

1. Create directory structure.
2. Add `data/validation/icop/sightings.json` with 25–40 carefully selected entries (I will start with a seed set).
3. Write `internal/validation/icop/icop.go`:
   - `Sighting` struct
   - `LoadSightings(path string)` function
   - Basic `Compare` logic (run renderer at location/time → map to Yallop category → compare to reported)
4. Write `cmd/validate-icop/main.go` as a simple CLI.
5. Add `validate-icop` target to Makefile.
6. Add `data/validation/icop/README.md` with strict selection criteria.

This gives us a working, commit-worthy foundation.

### Suggested Next Validation Steps (for Validation Agent)

- Ensure the new code does not affect existing `make test` or `make test-accuracy`.
- Verify that the harness can be run without the GPU renderer (falls back gracefully to CPU).
- Confirm that adding this does not introduce any new external dependencies.

I am ready to begin actual implementation once this proposal is accepted by the later stages.