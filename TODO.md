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

- [x] **High** - Build systematic external validation harness against ICOP sighting database (PR 2 of roadmap-execution-plan-f01edaab)
  **Status (2026-05)**: Foundation complete. Hardened `cmd/validate-icop` with `InstrumentAwareMatch` (naked A/B only; aided A-E), exact renderer moon-age alignment (point mode now emits age=), robust parsing/JSON summary, per-instrument breakdowns. Replaced mocks with 12 real, conjunction-aligned, provenance-rich ICOP records (Ramadan 1446, 2025-02-28 00:45 UT conjunction from https://astronomycenter.net/icop/ram46.html?l=en).
  **Result**: 100.0% match rate (12/12) on the curated set using exact CPU reference. Ages 10–42 h sensible. All instrument rules applied correctly. `make validate-icop --report=json` functional. Decoupled harness from astro CGO for portability.
  **Next (PR 3/4)**: Expand to 30–50 records across lunations + HMNAO baseline + publish quantitative report in yallop doc + regression hook.
  Ties to Core Principles: Directly strengthens Accuracy First (non-negotiable) and Verifiability & Reproducibility. This was the single largest gap called out in the May 2026 agentic review.
  From agentic review of TODO prioritization on 2026-05-24 + roadmap PR 2.

- [ ] **High** - Add HMNAO / UKHO lunar crescent visibility predictions as a comparison baseline (PR 3)
  **Status**: PR 3 initialization complete (`data/validation/hmnao/` + README skeleton + example placeholder + cross-refs). Full curated excerpts (10–20 lunations), comparison harness extension, and quantitative deltas targeted for the body of PR 3.
  Rationale: HMNAO publishes official predictions using (a version of) the Yallop method. Comparing our maps against their published tables for the same dates provides an independent implementation check and increases credibility.
  Ties to Core Principles: Strong Verifiability & Reproducibility + Accuracy First. Helps prove our Chebyshev + rise/set + Yallop logic produces equivalent results to the official source.
  Suggested validation: Manually or semi-automatically compare 10–20 dates against published HMNAO visibility tables. Document any systematic differences in first-visibility longitude or category boundaries.
  From agentic review of TODO prioritization on 2026-05-24. (PR 3 start per roadmap-execution-plan-f01edaab)

- [x] **High** - Harden validation match logic and align sighting records (delivered in PR 2)
  **Status**: Complete (roadmap PR 2). `InstrumentAwareMatch` implemented and wired (naked A/B only; aided A-E). 12 real conjunction-aligned ICOP records replaced mocks. Harness now 100% on exact renderer (see CHANGELOG and data/validation/icop/README.md). The original 0% problem is resolved.
  Ties to Core Principles: Directly improves Verifiability & Reproducibility + Accuracy First.

## Medium Priority (Supporting Accuracy + Performance)

- [ ] **Medium** - Implement at least one additional modern visibility criterion (start with a recent published method)
  Rationale: Having only Yallop + Odeh limits users who prefer other calibrated methods. Adding a third increases the tool's scientific utility.
  Ties to Core Principles: Supports Accuracy First by allowing direct side-by-side comparison. Improves Verifiability.
  Suggested validation: New `TestRendererAccuracy` variant that also exercises the new criterion. Ensure CPU/GPU match remains ≥96% for the new path.
  From agentic review of TODO prioritization on 2026-05-24.

- [x] **Medium** - Create golden sighting test dataset and regression harness (PR 8 + PR4 foundation)
  **Status (2026-05)**: Initial golden file `data/validation/golden/validate-icop.json` committed (exact 100% Summary from the hardened ICOP 12-record Ramadan 1446 run using CPU reference renderer). `validate-icop-ci` target already supports `ICOP_GOLDEN=...` for strict comparison. Native `--update-golden` support + guarded Makefile targets (`validate-icop-golden-update`, `validate-icop-golden-check`) added in PR4 hardening. 
  Rationale and ties to Core Principles unchanged (Verifiability & Reproducibility + Accuracy First protection).
  See `docs/roadmap-implementation-pr-body-draft.md` (Ollama-generated + Grok-corrected outline) and the companion `docs/roadmap-implementation-pr-creation-checklist.md` (ready-to-execute steps + suggested final PR title/summary) for consolidated PR preparation. Next: make golden-check a mandatory pre-merge gate in CI, capture additional HMNAO golden baselines once PR3 curation advances.
  From agentic review of TODO prioritization on 2026-05-24 + PR8/PR4 work on roadmap-execution-plan-f01edaab.

- [ ] **Medium** - Dynamic grid dimensions (metadata headers) for binary files
  Rationale: The Go compositor (`internal/blend/blend.go`) currently detects grid resolutions by probing a few hardcoded dimension arrays `{{3600, 2160}, {3840, 2160}, {1440, 720}}`. If the C++ binaries change their resolution macros (e.g. `PIXEL_PER_DEGREE`), the compositor fails to load the binary files.
  Ties to Core Principles: Improves Code Robustness, Modularity & Portability. By prefixing the raw binary grid output files with 8 bytes of metadata (4-byte width, 4-byte height), we eliminate fragile hardcoded dimension lookups.

- [x] **Medium** - Add Windows x64 (CPU-only) release artifacts (phased) — **Phases 1-3 + GPU local-build Phase 4 complete** (PR 5/6 of roadmap)
  **Phased approach**:
  - **Phase 1**: On-push CI build workflow (`.github/workflows/build.yml`, formerly `windows-cpu.yml`) — complete. Now a Linux + Windows matrix (MinGW + CGO + renderer + orchestrator + `go test` + verification; OpenMP fallback on Windows).
  - **Phase 2**: Integrate into main release workflow (`release.yml`) — complete (matrix + MinGW setup + static linking attempts + cross-platform tests + artifact globs + Windows-aware binary discovery in main.go).
  - **Phase 3**: Polish documentation + local build instructions (MinGW/choco) and update README/Makefile — **completed** (PR 5: dedicated "Building from source on Windows" subsection + anchor repair + public documentation-maintenance.md process). Light Makefile comments added. See docs/documentation-maintenance.md.
  - **Phase 4** (optional, later): Attempt OpenCL GPU renderer support on Windows. **Status update (PR 6)**: Best-effort guarded Makefile support + comprehensive local-build documentation delivered. Strict "local only, never released" policy now explicit everywhere. Real-hardware verification on Windows + discrete GPU noted as follow-up (pragmatic Unix proxy used during development per roadmap). See README "Building the GPU renderer on Windows" and docs/documentation-maintenance.md.
  Rationale: Windows is a major desktop platform. CPU-only first approach delivers immediate value. Full GPU support deferred.
  Ties to Core Principles: Modularity & Portability, Verifiability & Reproducibility.
  Status (post full agentic review + remediation 2026-05-25): Core functionality + release integration + runtime portability in orchestrator complete. Re-Judge gave "Go with Conditions" — conditions addressed. Standalone CI + release support ready for v0.5.2.

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

**Last Updated:** May 2026 (PR 5: Documentation & Architecture Agent process + Windows CPU Phase 3 completed per roadmap-execution-plan-f01edaab.md. External validation work remains High Priority.)

## How to Contribute

If you'd like to work on any of these items:

1. Check if there's an existing issue/PR for the feature
2. Create an issue describing your implementation plan
3. Fork the repository
4. Create a feature branch
5. Submit a pull request with tests and documentation

For questions or discussions, open an issue on GitHub.
