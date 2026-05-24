# TODO - Feature Roadmap

This document tracks planned enhancements and features for the Crescent Moon Visibility Maps Generator.

## High Priority Features

### 0. Apple Silicon GPU Support via FP32 + Double-Double Time
- **Status**: **Completed** (implemented May 2026)
- **Result**: The GPU renderer now works on Apple Silicon. A sibling kernel
  `gpu/visibility_kernel_fp32.cl` uses `float` + `native_*` math with a
  compensated double-double (float2) accumulator only for the per-pixel
  search time `t`. The host in `gpu/gpu_render.c` detects the lack of FP64
  and loads the FP32+DD kernel transparently.
- **Accuracy achieved**: 96.97 % exact per-pixel RGBA match vs the CPU
  reference on an M4 Pro (well inside the 96–97 % band of the original
  FP64 OpenCL path). Boundary ULP noise remains the only source of the
  residual ~3 % mismatches.
- **Performance**: ~82 ms kernel for a full 3600×2160 map on M4 Pro
  (hundreds of times faster than the single-threaded or lightly-threaded
  CPU path on the same machine).
- **Compatibility**: x86/Linux/NVIDIA/AMD devices continue to use the
  original FP64 kernel with no behavior change.

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
- ~~Moon Age Display~~ — Moon age (mid-day UTC) computed and passed through Go orchestrator to the pure-Go blending package (parameter received but not yet rendered in legend)
- ~~Before Conjunction Color Coding~~ — Categories `G` / `J` now render transparent so the base map shows through
- ~~GPU Acceleration~~ — OpenCL compute kernels (`visibility_kernel.cl` FP64 + `visibility_kernel_fp32.cl` FP32+DD) with `gpu/gpu_render.c` host
  - Cross-platform: macOS (Metal/OpenCL including Apple Silicon via automatic FP32+DD path), NVIDIA (CUDA OpenCL), AMD (ROCm), Intel Compute Runtime
  - Automatic kernel selection in `gpu_render.c` based on `CL_DEVICE_DOUBLE_FP_CONFIG`
  - 96.97 % per-pixel match on Apple Silicon M4 Pro (matches fidelity of FP64 path on other GPUs)
  - GPU binary auto-selected via `-gpu` flag in orchestrator
  - GPU blending now implemented in pure Go (`internal/blend`). Legacy `gpu_blend.py` (OpenCV T-API) retained only for advanced users.
- ~~Apple Silicon / FP32+DD OpenCL Support (2026)~~ — Full GPU acceleration now available on M1/M2/M3/M4+ via automatic selection of `visibility_kernel_fp32.cl`. 96.97 % pixel match, ~80 ms kernel time on M4 Pro, identical Yallop classification fidelity to FP64 path. See `docs/performance-accuracy.md`.
- ~~Python Removal + Automated Accuracy Testing (2026)~~ — Last significant Python component (`gpu_blend.py`) replaced by pure-Go implementation in `internal/blend`. Added `internal/blend/blend_test.go` + `TestRendererAccuracy` regression test (enforces ≥96% CPU/GPU match). `make test` and `make test-accuracy` targets added.
- ~~Chebyshev Polynomial Ephemeris~~ — Replaced the 8 640-step dense ephemeris with degree-24 Chebyshev fits in `__constant` GPU memory. ~125 doubles total vs ~410 KB before.
- ~~Geocentric Vector Path for Yallop ARCV~~ — Fitted moon's 3-D EQD geocentric vector and derived RA/Dec on the GPU, matching CPU's `Astronomy_GeoVector` + EQD rotation. 100 % classification agreement.
- ~~GPU Kernel Optimization~~ — `native_sin`/`native_cos`, reduced coarse scan (200→32), bisection (20→12), additional build flags (`-cl-mad-enable`, etc.). ~4× kernel speedup at identical accuracy.
- ~~24h Date Offset Fix~~ — GPU search uses forward-only 1-day window from longitude-adjusted base (matches CPU's `Astronomy_SearchRiseSet(time, 1)`).
- ~~Diamond Markers~~ — Naked-eye (red) and telescope (blue) markers drawn as 22-pixel diamonds with a 3-pixel black outline for 4K visibility.
- ~~Vector Legend~~ — Legend (colors + first-visibility markers) drawn with vector primitives in the pure-Go blending package (no font glyph dependency).
- ~~Dynamic Day Selection~~ — Orchestrator emits maps only for days reaching ≥ 0.2 % illumination at latest sunset on Earth (D+1 06:00 UTC sample); modern illumination-threshold rule with full sub-second new-moon precision.
- ~~WEBP Output~~ — Quality-98 WEBP at 60 % blend strength; ~2 MB per map vs ~9 MB PNG.
- ~~Cross-platform Makefile~~ — Auto-detects OpenCL headers and library paths (Debian/Fedora/Arch/ROCm/CUDA) and OpenMP support (`libomp` via Homebrew on macOS).
- ~~Caching and Memoization~~ — New moon dates cached in Go orchestrator (`newMoonCache`)
- ~~Refactoring~~ — CGO bindings extracted to `internal/astro`, C++ renderer moved to `cmd/visibility/`
- ~~Unit Testing~~ — Tests in `main_test.go` covering parseYears, astronomy, caching
- ~~Documentation~~ — Extensive inline docs + dedicated `docs/performance-accuracy.md` covering the FP32+DD technique, measured results, and visual-sighting validation (2026 update)

**Last Updated:** May 2026 (Python blending fully removed + Go accuracy test suite + documentation refresh)

## How to Contribute

If you'd like to work on any of these items:

1. Check if there's an existing issue/PR for the feature
2. Create an issue describing your implementation plan
3. Fork the repository
4. Create a feature branch
5. Submit a pull request with tests and documentation

For questions or discussions, open an issue on GitHub.
