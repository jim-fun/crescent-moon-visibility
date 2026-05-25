# TODO - Feature Roadmap

This document tracks planned enhancements and features for the Crescent Moon Visibility Maps Generator.

## Completed High-Priority Work (Historical)

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

## High Priority (Accuracy First + Verifiability)

These items are the current focus because they most directly strengthen the project's core claim of high-accuracy, trustworthy visibility predictions.

- [ ] **High** - Build systematic external validation harness against ICOP sighting database
  Rationale: We have a high-quality computational implementation of the Yallop (1997) q-test and Odeh (2004) criteria, plus a detailed comparison document (`docs/yallop-criteria-and-external-validation.md`). However, we lack quantitative evidence of how well the current predictions match real-world naked-eye and telescopic observations. This is the single largest gap in the "Accuracy First" claim.
  Ties to Core Principles: Directly strengthens Accuracy First (non-negotiable) and Verifiability & Reproducibility. Without this, all internal 96.97% numbers are only self-consistency, not external truth.
  Suggested validation: Curated set of 50–100 ICOP positive/negative sightings from 2015–2025. Automated comparison script that ingests sighting reports (lat/lon, date, instrument, success/failure) and runs the renderer at those locations/times. Report precision/recall per category (A/B vs C/D/E) and confusion matrix. Add as `make validate-icop` or similar.
  From agentic review of TODO prioritization on 2026-05-24.

- [ ] **High** - Add HMNAO / UKHO lunar crescent visibility predictions as a comparison baseline
  Rationale: HMNAO publishes official predictions using (a version of) the Yallop method. Comparing our maps against their published tables for the same dates provides an independent implementation check and increases credibility.
  Ties to Core Principles: Strong Verifiability & Reproducibility + Accuracy First. Helps prove our Chebyshev + rise/set + Yallop logic produces equivalent results to the official source.
  Suggested validation: Manually or semi-automatically compare 10–20 dates against published HMNAO visibility tables. Document any systematic differences in first-visibility longitude or category boundaries.
  From agentic review of TODO prioritization on 2026-05-24.

- [ ] **High** - Harden validation match logic and align sighting records
  Rationale: In `cmd/validate-icop/main.go`, category `C` ("May need optical aid") and `D` ("Will need optical aid") are ignored by the validation match algorithm. Additionally, the dates/locations in the current `sightings.json` represent test/mock sightings that do not align with the actual astronomical conjunction times (producing a 0% match rate).
  Ties to Core Principles: Directly improves Verifiability & Reproducibility + Accuracy First. Ensuring the test harness uses real-world observational database records and correctly handles instrument type for marginal categories (`C`, `D`, `E`) is essential to generate true quantitative validation results.

## Medium Priority (Supporting Accuracy + Performance)

- [ ] **Medium** - Implement at least one additional modern visibility criterion (start with a recent published method)
  Rationale: Having only Yallop + Odeh limits users who prefer other calibrated methods. Adding a third increases the tool's scientific utility.
  Ties to Core Principles: Supports Accuracy First by allowing direct side-by-side comparison. Improves Verifiability.
  Suggested validation: New `TestRendererAccuracy` variant that also exercises the new criterion. Ensure CPU/GPU match remains ≥96% for the new path.
  From agentic review of TODO prioritization on 2026-05-24.

- [ ] **Medium** - Create golden sighting test dataset and regression harness
  Rationale: As we add new criteria or modeling improvements, we need reproducible test cases that protect the accuracy bar.
  Ties to Core Principles: Excellent for Verifiability & Reproducibility and long-term protection of Accuracy First.
  Suggested validation: 20–30 high-quality ICOP or published sightings stored as JSON/CSV. Simple Go test or script that runs the renderer and asserts category or first-visibility time is within tolerance.
  From agentic review of TODO prioritization on 2026-05-24.

- [ ] **Medium** - Dynamic grid dimensions (metadata headers) for binary files
  Rationale: The Go compositor (`internal/blend/blend.go`) currently detects grid resolutions by probing a few hardcoded dimension arrays `{{3600, 2160}, {3840, 2160}, {1440, 720}}`. If the C++ binaries change their resolution macros (e.g. `PIXEL_PER_DEGREE`), the compositor fails to load the binary files.
  Ties to Core Principles: Improves Code Robustness, Modularity & Portability. By prefixing the raw binary grid output files with 8 bytes of metadata (4-byte width, 4-byte height), we eliminate fragile hardcoded dimension lookups.

- [ ] **Medium** - Add Windows x64 (CPU-only) release artifacts (phased)
  **Phased approach**:
  - **Phase 1 (Current)**: Create initial Windows CPU-only CI workflow. Build `crescent_maps.exe` + CPU renderer (`visibility.exe`) on `windows-latest` using MinGW-w64 + CGO. No GPU support yet.
  - **Phase 2**: Integrate Windows binaries into the main release workflow so `v*` tags produce Windows artifacts alongside Linux.
  - **Phase 3**: Document Windows build instructions (MinGW or MSVC) and update README / Makefile.
  - **Phase 4** (optional, later): Attempt OpenCL GPU renderer support on Windows (significantly harder due to driver variability and CI limitations).
  Rationale: Windows is a major desktop platform. Currently no pre-built binaries are provided. Starting with CPU-only keeps scope reasonable and delivers immediate value. Full GPU support can be deferred.
  Ties to Core Principles: Modularity & Portability, Verifiability & Reproducibility.
  Suggested first implementation: New workflow `.github/workflows/windows-cpu.yml` (or extend release.yml) that uses `choco install mingw` + `CGO_ENABLED=1 go build` + MinGW g++ for the C++ renderer.
  From discussion on adding Windows x64 releases (May 2026).

## Future / Stretch Goals

The following items were deprioritized during the May 2026 agentic review because they score significantly lower on Accuracy First and Verifiability compared to external validation work. They remain desirable long-term enhancements.

- Web-Based Interface (WASM + JS frontend)
- Real-Time Visibility Predictions + notifications
- Terrain Elevation Integration (SRTM + horizon adjustments)
- Atmospheric Extinction Modeling
- Observer Experience / acuity factors
- Configurable Map Projections
- Multi-Language Support

These may be revisited after the High Priority validation work is substantially complete.

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

- ~~Public GitHub Repository Restart & Documentation Polish (2026)~~ — Fresh clean launch of the official public mirror at https://github.com/jim-fun/crescent-moon-visibility. Internal AI/agentic tooling, .agentic-review/ artifacts, and maintainer-only documents (AGENTIC_WORKFLOW.md, scripts/agents/, etc.) deliberately excluded from public main. README badges, version examples, agentic references, and CHANGELOG updated for the launch. Proper annotated v0.3.0 tag created on clean public history.
- ~~WEBP Output~~ — Quality-98 WEBP at 60 % blend strength; ~2 MB per map vs ~9 MB PNG.
- ~~Cross-platform Makefile~~ — Auto-detects OpenCL headers and library paths (Debian/Fedora/Arch/ROCm/CUDA) and OpenMP support (`libomp` via Homebrew on macOS).
- ~~Caching and Memoization~~ — New moon dates cached in Go orchestrator (`newMoonCache`)
- ~~Refactoring~~ — CGO bindings extracted to `internal/astro`, C++ renderer moved to `cmd/visibility/`
- ~~Unit Testing~~ — Tests in `main_test.go` covering parseYears, astronomy, caching
- ~~Documentation~~ — Extensive inline docs + dedicated `docs/performance-accuracy.md` covering the FP32+DD technique, measured results, and visual-sighting validation (2026 update)

**Last Updated:** May 2026 (Major agentic workflow prioritization: Elevated external validation (ICOP + HMNAO) to High Priority. Deprioritized older feature work in favor of strengthening Accuracy First and Verifiability.)

## How to Contribute

If you'd like to work on any of these items:

1. Check if there's an existing issue/PR for the feature
2. Create an issue describing your implementation plan
3. Fork the repository
4. Create a feature branch
5. Submit a pull request with tests and documentation

For questions or discussions, open an issue on GitHub.
