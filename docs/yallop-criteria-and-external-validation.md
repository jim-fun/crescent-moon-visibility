# Yallop Criteria Implementation and External Validation

**Crescent Moon Visibility Maps Generator — Rigorous Comparison to Published Methods and Real-World Data**

This document provides the authoritative analysis of how faithfully the project implements the classic Yallop (1997) lunar crescent visibility criterion, its relationship to the supported Odeh alternative, and the current state of external validation against historical and observational datasets.

**Core Project Principle**: Accuracy First (non-negotiable). All visibility math changes must preserve or improve fidelity to the calibrated empirical model while remaining suitable for real visual sighting predictions.

---

## Executive Summary

- The project implements a **high-fidelity computational realization** of B.D. Yallop’s 1997 method (HM Nautical Almanac Office Technical Note No. 69), including the exact published cubic coefficients, topocentric crescent width `W'`, 4/9 best-time (lag) rule, geocentric ARCV handling for Yallop, pre-conjunction G/I/J categories, and the six q-based visibility classes A–F.
- An alternative **Odeh (2004)** criterion is also fully supported (`--method odeh` / `yallop|odeh` argument) with its own published constants.
- Internal numerical fidelity between the pure-double C++ CPU reference (`cmd/visibility/visibility.cc`) and both GPU kernels is **~96.97% exact per-pixel RGBA** on the hardest cases (M4 Pro FP32+DD path); residuals are almost entirely expected ULP noise at decision boundaries.
- The implementation matches the original Yallop paper coefficients and formulas to machine precision (within floating-point representation).
- **External validation status**: The computational model is faithful to the published Yallop calibration dataset (295 sightings, 1859–1996). Systematic statistical comparison against modern independent sighting corpora (ICOP, HMNAO predictions, etc.) is identified as future work in the project roadmap (see TODO.md).
- The resulting global maps are therefore suitable for visual sighting planning, record-young crescent analysis, and educational use, with the important caveat that they do not incorporate local weather, light pollution, or observer-specific physiological factors.

---

## 1. The Yallop (1997) Method — Original Formulation

Yallop’s single-parameter `q`-test (NAO Technical Note No. 69, 1997) is an empirical adaptation of the long-used “Indian method” (itself derived from Schoch 1930 and adopted by HMNAO in 1996), calibrated against 295 verified first sightings spanning 1859–1996.

### 1.1 Core Variables (as defined by Yallop)

- **ARCL** — Geocentric elongation (angular separation Sun–Moon at Earth center).
- **ARCV** — Arc of Vision. For the Yallop path the code uses a **geocentric-derived ARCV** (see implementation details below).
- **DAZ** — Difference in azimuth (Sun azimuth − Moon azimuth).
- **W (or W')** — Width of the crescent in arcminutes. Topocentric version `W'` includes lunar parallax correction:
  ```
  SD  = 0.27245 * π (semi-diameter in degrees, geocentric)
  SD' = SD * (1 + sin(h) * sin(π))   // topocentric adjustment
  W'  = SD' * (1 − cos(ARCL))
  ```
  This matches Yallop §3.8–3.10 exactly.

### 1.2 The q-Test and Cubic Threshold (Exact Match)

Yallop gives the visibility criterion as a cubic in `W'` (equation 3.6 in the paper, the form adopted from the Indian method):

```
ARCV > 11.8371 − 6.3226 W + 0.7319 W² − 0.1018 W³
```

The project’s single test parameter is:

```
value = q = (ARCV − (11.8371 − 6.3226*W + 0.7319*W² − 0.1018*W³)) / 10
```

**Exact coefficients** from Yallop 1997 (p. 3 of the PDF) are hard-coded in three places:

- `cmd/visibility/visibility.cc:142`
- `gpu/visibility_kernel.cl:320`
- `gpu/visibility_kernel_fp32.cl:368`

The six visibility classes (Table 5 / text in Yallop) are also reproduced verbatim:

| Class | q range          | Meaning (Yallop)                     | Project color |
|-------|------------------|--------------------------------------|---------------|
| A     | > +0.216        | Easily visible to the unaided eye    | Cyan          |
| B     | > −0.014        | Visible under perfect conditions     | Darker cyan   |
| C     | > −0.160        | May need optical aid                 | Light cyan    |
| D     | > −0.232        | Will need optical aid                | Bright cyan   |
| E     | > −0.293        | Visible only with telescope          | Dark cyan     |
| F     | ≤ −0.293        | Not visible (below Danjon limit)     | Transparent   |

These thresholds come directly from calibration on the 295-record database.

### 1.3 Best Time of Observation (4/9 Lag Rule)

Yallop adopts a simple practical rule derived from Bruin (1977) graphs:

```
t_best = sunset + lag * (4/9)   (evening; sign-reversed for morning)
```

This is implemented identically in the CPU reference and both kernels. The lag is the time from sunset to moonset (positive when Moon sets after Sun).

### 1.4 Pre-Conjunction and Special Categories (G/I/J)

The code adds transparent “impossible” categories before the new moon (G = before conjunction at sunset, I = Moon sets before sunset, J = mixed) that are consistent with Yallop’s physical intent even if not part of the original six-class table.

Moon-age lines (white) are drawn every ~72 minutes of lunar age for visual context.

---

## 2. Project Implementation Fidelity

### 2.1 CPU Reference (Ground Truth)

`cmd/visibility/visibility.cc` (template `calculate<evening, yallop>`, lines 44–192) contains the authoritative logic:

- Rise/set searches via Astronomy Engine.
- Lag + 4/9 best-time (lines 60–69).
- Geocentric vector path for Yallop ARCV (lines 116–125) using `Astronomy_GeoVector` + EQD rotation + `Astronomy_Horizon`.
- Topocentric `W'` with parallax (lines 103–106).
- Exact cubic + q thresholds + A–F classification (lines 140–155).
- First-visibility diamond tracking (red/blue) and moon-age line rasterization (lines 197+).

### 2.2 GPU Paths (Performance with Integrity)

Both kernels (`visibility_kernel.cl` full double and `visibility_kernel_fp32.cl` FP32+DD) replicate the identical formulas, constants, 4/9 rule, G/I/J logic, and moon-age line test.

The only intentional difference in the FP32+DD kernel is compensated double-double arithmetic **exclusively for the search time accumulator** `t` / `t_best` / `lag_time`. All visibility math (ARCV, W, cubic, q) remains in native float. This design decision is documented in the kernel header and preserves boundary fidelity.

Result: **96.97% exact per-pixel match** vs the CPU double reference on M4 Pro (see `docs/performance-accuracy.md` and `main_test.go:TestRendererAccuracy`).

### 2.3 Odeh (2004) Support

When `is_yallop == 0` the code switches to Odeh’s formulation:

- Topocentric elongation for ARCL.
- Different cubic constants (7.1651 − 6.3226W + …).
- Four categories (A / C / E / F) with thresholds 5.65°, 2.00°, −0.96°.
- Implemented in the same three files with identical structure.

This gives users a well-known independent modern criterion alongside the Yallop baseline.

---

## 3. Comparison to Other Historical and Modern Criteria

| Criterion     | Year | Key Formula                          | Categories     | Project Support | Notes |
|---------------|------|--------------------------------------|----------------|-----------------|-------|
| Maunder       | 1911 | Quadratic in DAZ                     | Visible / not  | No (historical reference only) | Simple early model |
| Indian (Schoch) | ~1930 / 1966 | Quadratic in DAZ                     | —              | No | Basis for Yallop |
| Bruin         | 1977 | Cubic in W (different coeffs)        | Curves         | No | Yallop’s source for 4/9 rule |
| **Yallop (Indian adopted)** | **1997** | **Cubic in W' (exact coeffs 3.6)** | **A–F via q** | **Full (default)** | Calibrated on 295 sightings; HMNAO standard |
| Odeh          | 2004 | Different cubic + topocentric ARCL   | A/C/E/F        | **Full (alternative)** | Simpler, widely used in modern apps |
| Schaefer (theoretical) | 1988– | Extinction, physiology, contrast     | Probabilistic  | No | Most physically complete but complex |
| Ilyas / others | various | Variants of lag + elongation         | —              | No | Many regional adaptations |

Yallop’s cubic (3.6) is numerically very close to the Indian quadratic in the critical 8°–12° elongation zone but behaves better at higher latitudes (Yallop explicitly notes this advantage over pure Bruin).

---

## 4. External / Observational Validation Status

### 4.1 Internal (CPU ↔ GPU)

Automated, quantitative, and excellent (≥96.97% exact RGBA on production maps). This guards numerical stability of the Yallop math across hardware.

### 4.2 Historical Calibration Dataset (Yallop’s 295 records)

Because the project uses the **exact published coefficients** derived from those 295 sightings, any map generated with the Yallop path is, by construction, applying the same empirical test Yallop calibrated.

### 4.3 Modern Independent Datasets (Current Gap)

- **ICOP (Islamic Crescents Observation Project)**: Largest public collection of verified positive and negative naked-eye and telescopic sightings since the late 1990s. Many peer-reviewed papers use ICOP + Yallop/Odeh/Schaefer models for statistical validation.
- **HMNAO / UK Hydrographic Office** predictions and almanac tables.
- National astronomical society archives and individual observer logs (e.g., from the 2020s great conjunction era and regular new-moon watches).

**Current project status**: No automated or systematic comparison against these corpora has yet been performed and committed. This is explicitly called out as future work in `TODO.md`, `docs/performance-accuracy.md`, and the agentic workflow backlog.

When such a study is undertaken it should:
- Use the exact same Yallop and Odeh paths the renderers expose.
- Report success rates, false-positive/negative rates, and boundary cases.
- Be added to the automated test suite where possible (or as a documented research notebook).
- Feed back into possible coefficient refinements only under the strict “Accuracy First” discipline (any change requires full 4-stage agentic review + Judge sign-off).

---

## 5. Strengths of the Current Implementation

- **Mathematical fidelity** — Coefficients, formulas, and decision logic are byte-for-byte matches to the 1997 source.
- **Dual independent verification** — CPU reference + two GPU kernels with measured high pixel agreement.
- **Performance without compromise** — Chebyshev ephemeris + selective double-double time on Apple Silicon delivers speed while staying inside the ULP tolerance of the original model.
- **Global, high-resolution, reproducible maps** — 0.25° or better resolution, identical output format across platforms.
- **Transparency** — Full source, tests (`make test-accuracy`), and this document.

---

## 6. Limitations and Caveats (Honest Assessment)

- The q-test is an **empirical fit**, not a first-principles atmospheric model. It does not explicitly vary with temperature, humidity, aerosol load, or observer acuity (unlike full Schaefer-style calculations).
- Local conditions (clouds, light pollution, altitude, horizon profile) are outside scope.
- First-visibility “diamonds” (red = naked-eye A/B, blue = telescope C/D) and numeric moon-age in the legend are only partially implemented (CPU only for diamonds today).
- Systematic external validation against post-1996 ICOP/HMNAO data is still pending.
- The model can (and does) predict visibility on dates and at longitudes where real-world observers may fail due to the factors above.

These limitations are **by design** — the tool produces the pure geometric/empirical prediction that can then be compared with actual sightings.

---

## 7. References & Sources

- Yallop, B.D. (1997). *A Method for Predicting the First Sighting of the New Crescent Moon*. HM Nautical Almanac Office Technical Note No. 69. (PDF: astronomycenter.net or HMNAO archives). Equation 3.6 and Table 5 are the direct sources of the cubic and q thresholds.
- Odeh, M.S. (2004). “New Criterion for Lunar Crescent Visibility.” *Observatory* 124.
- Bruin, F. (1977). “The First Visibility of the Lunar Crescent.” *Vistas in Astronomy* 21.
- Indian Astronomical Ephemeris (various editions) — source of the quadratic that Yallop refined.
- Astronomy Engine (Don Cross) — rise/set search, elongation, vector routines used by the renderers.
- Project source: `cmd/visibility/visibility.cc`, `gpu/visibility_kernel*.cl`, `docs/performance-accuracy.md`, `main_test.go`.
- ICOP — https://www.icoproject.org (sighting database).
- HMNAO / UKHO lunar crescent predictions.

---

## 8. Recommended Next Steps (Tied to Agentic Workflow)

1. Systematic ICOP / HMNAO comparison study (High priority in current TODO).
2. Port remaining diamond + legend features to the GPU path (Validation + Judge review required).
3. Consider adding a “Yallop vs Odeh vs Schaefer-lite” mode or diagnostic table output for researchers.
4. Any future coefficient tweak or new criterion must go through the full Improvement → Validation (accuracy regression) → Security → **Judge** process with explicit Accuracy First scorecard.

Use the agentic tooling:

```bash
./scripts/agentic-review.sh --improve "Perform systematic comparison of Yallop maps against ICOP sighting records for 2020–2025 and publish results"
./scripts/agentic-review.sh --review-todo "External validation gaps vs ICOP/HMNAO in the Yallop implementation"
```

---

**Document status**: May 2026 — created as part of the project’s ongoing commitment to Accuracy First and transparent external validation.

**Maintained by**: Project maintainers + the crescent-moon-visibility-engineering skill and agentic workflow (especially Validation, Documentation & Architecture, and Judge agents).

---

*This document complements `docs/performance-accuracy.md` (internal renderer fidelity) and should be updated whenever new external validation data or implementation changes are accepted by the Judge.*