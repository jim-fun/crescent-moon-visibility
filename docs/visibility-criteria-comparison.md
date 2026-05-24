# Visibility Criteria Comparison

This document provides a detailed comparison of the lunar crescent visibility calculation methods implemented or referenced by the Crescent Moon Visibility Maps Generator.

## 1. Overview

Different visibility criteria have been developed over the past century to predict when and where the young crescent moon will be visible after conjunction. They differ in:

- The physical parameters they emphasize (elongation, width, lag time, extinction, observer physiology, etc.)
- Empirical calibration datasets
- Complexity vs. usability

The project currently provides full, high-fidelity implementations of the two most widely referenced modern empirical criteria:

- **Yallop (1997)** — Default method
- **Odeh (2004)** — Alternative method (selectable via `--method odeh` or equivalent)

## 2. Yallop (1997) Criterion

**Reference**: B.D. Yallop, "A Method for Predicting the First Sighting of the New Crescent Moon", HMNAO Technical Note No. 69 (1997, revised 1998).

### Core Formula

Yallop uses a topocentric crescent width `W'` (in arcminutes) and derives a "q" value from a cubic in `W'`:

**q = ARCV − (11.8371 − 6.3226W' + 0.7319W'² − 0.1018W'³)**

Where:
- `ARCV` = Arc of Vision (altitude difference between moon and sun at the best time)
- `W'` = Topocentric crescent width (accounting for parallax)

Classification uses `q` and a secondary parameter (often related to elongation or other factors) to assign categories A–F (plus G/I/J special cases).

### Lag Rule
Uses the well-known **4/9 rule**: the "best time" for sighting is approximately 4/9 of the way from sunset to moonset (evening) or moonrise to sunrise (morning).

### Project Implementation
- Full implementation in `cmd/visibility/visibility.cc` (CPU reference) and both OpenCL kernels.
- "Point" query mode supports `yallop` (default).
- First-visibility diamonds and moon-age lines are generated using Yallop logic.

## 3. Odeh (2004) Criterion

**Reference**: M.S. Odeh, "New Criterion for Lunar Crescent Visibility", *The Observatory*, Vol. 124 (2004).

### Core Differences from Yallop

- Uses a **different cubic** for the visibility threshold.
- Emphasizes **topocentric elongation (ARCL)** more directly in some formulations.
- Simpler category system in practice: A / C / E / F (with different threshold angles, e.g., roughly 5.65°, 2.00°, −0.96° in some presentations).

Odeh's approach is popular in many modern astronomical apps and some regional moon-sighting communities because it is relatively straightforward to implement while still being calibrated on real observations.

### Project Implementation
- Fully supported as an alternative path in the same renderers.
- Selected via command-line or API flag (`--method odeh`).
- Reuses the same search, lag, and rendering infrastructure for consistency.

## 4. Side-by-Side Comparison

| Aspect                    | Yallop (1997)                          | Odeh (2004)                          | Notes |
|---------------------------|----------------------------------------|--------------------------------------|-------|
| Primary parameter         | Topocentric width `W'` + ARCV         | Different cubic; strong use of topocentric ARCL | Both empirical |
| Cubic / threshold         | 11.8371 − 6.3226W' + ...              | Different published coefficients    | See source papers for exact numbers |
| Categories                | A–F (+ G/I/J specials)                | A/C/E/F                             | Yallop has more granular "naked eye" gradations |
| Lag rule                  | 4/9 of sunset→moonset interval        | Similar best-time concept           | Project uses consistent 4/9 for both |
| Calibration dataset       | 295 historical sightings              | Separate modern dataset             | Both respected in literature |
| Project support           | Default (`yallop`)                    | Alternative (`odeh`)                | Both achieve same high internal fidelity |
| Typical use               | HMNAO / scientific / detailed maps    | Simpler apps, some Islamic calendars | — |

## 5. Other Criteria and Approaches (Considered)

While not fully implemented as first-class modes, the project is aware of and documents the following for context and future work:

- **Schaefer (1988–)**: Highly physical model incorporating atmospheric extinction, contrast, and human visual acuity. Very complex; not suitable for fast per-pixel rendering.
- **Maunder / Indian (early 20th century)**: Simpler quadratic models that influenced Yallop.
- **Regional / community methods**: Many groups (Islamic, Karaite, etc.) use slight variations or local horizon adjustments.

## 6. Observational Data Sources for Validation

Real-world sighting reports are essential for external validation of any criterion. The project plans to integrate data from multiple sources:

### Primary Recommended Sources

- **ICOP (Islamic Crescents Observation Project)**: Largest curated database of positive and negative sightings. Already identified as the top priority for the external validation harness (see TODO.md).
- **HMNAO / UKHO predictions**: Official UK predictions using (a version of) Yallop. Planned baseline comparison.
- **moonsighting.com**: Publishes extensive historical visibility maps and criteria discussions (Islamic context). Useful for cross-checking predicted vs. reported visibility on many dates.
- **Northwest Moonwatch Initiative (nwmi.org)**: Regional North American predictions and reports.
- **Paleo Times** (paleotimes.org/blog/): Observational new moon / crescent reports with Karaite/ancient calendar focus.
- **Truth of Yahweh** (truthofyahweh.org/moon.htm): Detailed moon observation records and methodology.
- **Date Tree** (patreon.com/c/datetree/posts): Regular posting of moon sighting data and predictions.

### Additional Sites Worth Monitoring

- al-habib.info (HilalMap tool) — Interactive visibility mapping.
- Various Karaite and Messianic communities (nehemiaswall.com, abluethread.com, etc.) — Often publish actual naked-eye reports with location and instrument details.
- IMCCE (France) and other national almanac offices — Scientific lunar data.

## 7. Future Work & Recommendations

- Implement a "criteria comparison" mode or diagnostic output that can emit side-by-side Yallop vs Odeh results for a given location/date.
- Build the ICOP ingestion + scoring harness (highest priority).
- Add support for ingesting sighting reports from the community sites listed above.
- Consider a lightweight "Schaefer-lite" or extinction-adjusted mode for advanced users.

This document will be updated as more comparison data and validation results become available.

---

**Related Documents**
- [Performance and Accuracy](performance-accuracy.md)
- [Yallop Criteria and External Validation](yallop-criteria-and-external-validation.md)
- [TODO.md](../TODO.md) (High Priority: ICOP and HMNAO work)
