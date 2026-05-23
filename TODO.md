# TODO - Feature Roadmap

This document tracks planned enhancements and features for the Crescent Moon Visibility Maps Generator.

## High Priority Features

### 0. Apple Silicon GPU Support via FP32 + Double-Double Time
- **Task**: Add a single-precision OpenCL kernel path so the GPU renderer
  works on Apple Silicon (M1/M2/M3/M4), whose Metal-backed OpenCL has no
  hardware FP64. Today those machines fall back to the CPU renderer
  (~76 s for 3 maps on M4 Pro vs an expected ~3 s on GPU).
- **Approach**:
  - Author a sibling kernel (e.g. `gpu/visibility_kernel_fp32.cl`) that uses
    `float` for position/trig math but represents the per-pixel time as a
    *double-double* pair of floats (`t_hi`, `t_lo`).
  - Time arithmetic (sunset finding, Chebyshev `x` mapping, moon-age
    indicator) all goes through tiny DD helpers (`dd_add`, `dd_mul_d`,
    `dd_to_float`). This captures the precision-sensitive accumulation
    without paying the cost of full DD on every multiply.
  - In `gpu/gpu_render.c`, query `CL_DEVICE_DOUBLE_FP_CONFIG` (already done
    today for the friendly error message) and, when zero, transparently
    load the FP32 kernel instead of bailing out.
  - Chebyshev coefficients on the CPU side are computed in double, then
    truncated to float pairs (hi + (orig - hi)) before upload.
- **Expected outcome**:
  - Apple Silicon: ~3 s per 3 maps (≈25× faster than CPU on M4 Pro).
  - Per-pixel match with the CPU reference: 96–97 % (vs the current 97 %
    on the FP64 path — the DD time arithmetic preserves the boundary
    behaviour well; the remaining ULP-level boundary noise is unchanged).
  - x86/Linux/NVIDIA/AMD users see no change — the existing FP64 kernel
    stays the default whenever FP64 is supported.
- **Effort estimate**: ~3 days (kernel rewrite + DD helpers + dispatch
  logic + regression test on both Apple Silicon and an FP64 device).
- **Acceptance criteria**:
  - `make gpu` succeeds on macOS Apple Silicon.
  - `./crescent_maps -gpu -years 2027 -months 1` produces 3 maps on
    M-series Macs with > 95 % per-pixel agreement against the CPU output.
  - No regression in match rate (≥ 96.96 % per-pixel) on an FP64 device.

## Medium Priority Features

### 1. Additional Visibility Criteria
- **Task**: Implement alternative crescent visibility criteria
- **Options**:
  - Schaefer criterion (1988)
  - Bruin criterion
  - SAAO (South African Astronomical Observatory) criterion
  - Ilyas criterion
- **Rationale**: Different criteria may be preferred by different communities
- **Impact**: Broader applicability and validation opportunities

### 2. Atmospheric Extinction Modeling
- **Task**: Add atmospheric extinction variations
- **Details**:
  - Seasonal atmospheric density changes
  - Altitude-dependent atmospheric models
  - Geographic variation in atmospheric conditions
- **Impact**: Improved accuracy, especially at lower altitudes

### 3. Terrain Elevation Integration
- **Task**: Incorporate terrain elevation data
- **Implementation**:
  - Integrate SRTM or similar elevation datasets
  - Adjust horizon calculations for mountains/valleys
  - Account for elevated observation points
- **Impact**: More realistic visibility predictions for terrestrial observers

### 4. Observer Experience Factor
- **Task**: Add observer experience parameter
- **Details**:
  - Beginner vs expert observer adjustments
  - Visual acuity factors
  - Age-related visibility corrections
- **Impact**: Personalized predictions

## Low Priority / Future Enhancements

### 5. Web-Based Interface
- **Task**: Create web UI for interactive map generation
- **Features**:
  - Date picker
  - Location selector
  - Criteria selection
  - Real-time rendering
  - Download functionality
- **Technologies**: WebAssembly (compile C++ to WASM), JavaScript frontend
- **Impact**: Easier accessibility for non-technical users

### 6. Real-Time Visibility Predictions
- **Task**: Automatic calculation for upcoming new moons
- **Features**:
  - Automated scheduling
  - Email/SMS notifications
  - Location-based alerts
- **Impact**: Proactive user notifications

### 7. Historical Sighting Database
- **Task**: Validate predictions against historical sighting reports
- **Data Sources**:
  - ICOP (Islamic Crescents' Observation Project)
  - Historical records
  - Observatory reports
- **Impact**: Validation and calibration of models

### 8. Configurable Map Projections
- **Task**: Support multiple map projections
- **Options**:
  - Mercator (current)
  - Equirectangular
  - Robinson
  - Mollweide
- **Impact**: Better visualization for different use cases

### 9. Multi-Language Support
- **Task**: Internationalize output and annotations
- **Languages**: Arabic, English, French, Turkish, Urdu, Malay, etc.
- **Impact**: Global accessibility

## Research and Validation

### 10. Accuracy Analysis
- **Task**: Systematic comparison with observational data
- **Methodology**:
  - Compare with HMNAO predictions
  - Validate against sighting reports
  - Statistical analysis of differences
- **Output**: Research paper or technical report

### 11. Sensitivity Analysis
- **Task**: Analyze parameter sensitivity
- **Parameters**:
  - Atmospheric refraction model
  - Best time calculation (4/9 ratio)
  - Crescent width threshold
- **Impact**: Understanding of model robustness

---

## Completed

- ~~Latitude Capping~~ — Capped at ±60° in `visibility.cc` and OpenCL kernel render loops
- ~~Moon Age Display~~ — Moon age threaded through Go → Python pipeline
- ~~Before Conjunction Color Coding~~ — Categories `G` / `J` now render transparent so the base map shows through
- ~~GPU Acceleration~~ — OpenCL compute kernel in `gpu/visibility_kernel.cl` with `gpu/gpu_render.c` host
  - Cross-platform: macOS (Metal/OpenCL), NVIDIA (CUDA driver's OpenCL), AMD (ROCm), Intel GPU Compute Runtime
  - GPU binary auto-selected via `-gpu` flag in orchestrator
  - GPU blending via OpenCV OpenCL T-API in `gpu_blend.py`
- ~~Chebyshev Polynomial Ephemeris~~ — Replaced the 8 640-step dense ephemeris with degree-24 Chebyshev fits in `__constant` GPU memory. ~125 doubles total vs ~410 KB before.
- ~~Geocentric Vector Path for Yallop ARCV~~ — Fitted moon's 3-D EQD geocentric vector and derived RA/Dec on the GPU, matching CPU's `Astronomy_GeoVector` + EQD rotation. 100 % classification agreement.
- ~~GPU Kernel Optimization~~ — `native_sin`/`native_cos`, reduced coarse scan (200→32), bisection (20→12), additional build flags (`-cl-mad-enable`, etc.). ~4× kernel speedup at identical accuracy.
- ~~24h Date Offset Fix~~ — GPU search uses forward-only 1-day window from longitude-adjusted base (matches CPU's `Astronomy_SearchRiseSet(time, 1)`).
- ~~Diamond Markers~~ — Naked-eye (red) and telescope (blue) markers drawn as 22-pixel diamonds with a 3-pixel black outline for 4K visibility.
- ~~Vector Legend~~ — `gpu_blend.py` uses `draw.rectangle` / `draw.ellipse` instead of Unicode `■`/`●` glyphs (font-independent rendering).
- ~~Dynamic Day Selection~~ — Orchestrator emits maps only when the crescent is ≥ 12 h old at 18:00 UTC; new moon times preserved at full hour/minute/second precision.
- ~~WEBP Output~~ — Quality-98 WEBP at 60 % blend strength; ~2 MB per map vs ~9 MB PNG.
- ~~Cross-platform Makefile~~ — Auto-detects OpenCL headers and library paths (Debian/Fedora/Arch/ROCm/CUDA) and OpenMP support (`libomp` via Homebrew on macOS).
- ~~Caching and Memoization~~ — New moon dates cached in Go orchestrator (`newMoonCache`)
- ~~Refactoring~~ — CGO bindings extracted to `internal/astro`, C++ renderer moved to `cmd/visibility/`
- ~~Unit Testing~~ — Tests in `main_test.go` covering parseYears, astronomy, caching
- ~~Documentation~~ — Inline docs in `main.go`, `internal/astro/astro.go`, `gpu_blend.py`, `gpu/gpu_render.c`, `gpu/visibility_kernel.cl`

**Last Updated:** May 23, 2026

## How to Contribute

If you'd like to work on any of these items:

1. Check if there's an existing issue/PR for the feature
2. Create an issue describing your implementation plan
3. Fork the repository
4. Create a feature branch
5. Submit a pull request with tests and documentation

For questions or discussions, open an issue on GitHub.
