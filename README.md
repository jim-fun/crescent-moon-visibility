# Crescent Moon Visibility Maps Generator

A high-performance application for generating accurate crescent moon visibility maps using astronomical calculations. This tool implements well-established visibility criteria (Yallop and Odeh) to predict when and where the new crescent moon will be visible to observers around the world.

## Architecture

The project uses a carefully engineered **mixed-language architecture** for performance, portability, and long-term accuracy:

- **Go Orchestrator (`main.go` + `internal/astro`)**: The single entry point and coordination layer. Handles CLI, CGO bindings to the Astronomy Engine (thirdparty/astronomy.c) for precise new-moon and illumination calculations, dynamic day selection via illumination-fraction test (see below), task fan-out to parallel workers, renderer dispatch (CPU or GPU binary), and orchestration of the final blend. Version info injected via ldflags from the canonical `VERSION` file. New-moon caching avoids redundant CGO calls.

- **CPU Reference Renderer (`cmd/visibility/visibility.cc`)**: Pure double-precision C++ implementation using OpenMP. This is the *ground truth* for all accuracy claims. It implements the full Yallop (default) and Odeh visibility criteria, rise/set searches, geocentric vector handling for ARCV, best-time (4/9 lag) rule, moon-age line drawing, and first-visibility diamond markers (red/blue). Outputs raw RGBA .bin files. Any change to visibility math must be mirrored here first.

- **GPU Renderer (`gpu/gpu_render.c` host + kernels)**: High-performance OpenCL path (`gpu_visibility.out`).
  - **Chebyshev polynomial ephemeris** (`gpu/chebyshev.c/h` + degree-24 fits in `__constant` memory): Replaces a large dense table (~410 KB) with ~125 doubles of coefficients per quantity. CPU-side double-precision fit (via Astronomy Engine samples); GPU evaluates the polynomial per-pixel at the exact local sunset time. Delivers ~1e-12 rad accuracy, far below decision thresholds.
  - **Dual kernels for FP64 / FP32 compatibility**:
    - `visibility_kernel.cl`: Full `double` + native math on devices reporting `cl_khr_fp64`.
    - `visibility_kernel_fp32.cl`: `float` + `native_sin`/`native_cos` everywhere *except* a compensated double-double (`float2` hi+lo) accumulator *exclusively for the critical search time `t`* (coarse 32-step scan + 12-iter bisection, lag, t_best, G/J logic, moon-age lines). All other quantities (Chebyshev coeffs rounded to float, RA/Dec, alts, ARCV, W, Yallop Q) stay in float. This is the key design choice that unlocks Apple Silicon (Metal-backed OpenCL has no FP64) while preserving boundary fidelity.
  - Host (`gpu_render.c`) queries `CL_DEVICE_DOUBLE_FP_CONFIG` once and loads the appropriate kernel + coeff buffer sizes transparently. No user-visible difference on FP64 hardware.
  - Result: both paths achieve **~96.97–97 % exact per-pixel RGBA match** vs the CPU double reference (residuals are expected ULP noise at Yallop cubic thresholds). See the authoritative **[Performance and Accuracy](docs/performance-accuracy.md)** for methodology, M4 Pro numbers (~80 ms kernel), boundary analysis, and why only time needed DD (not Chebyshev coeffs).

- **Pure-Go Blending (`internal/blend`)**: Self-contained replacement (since 2026) for the legacy `scripts/legacy/gpu_blend.py`. Loads CPU .bin or GPU PNG overlays, early-skip on insufficient visibility (A–E pixels), 60 % alpha composite onto NASA base map (`data/map_nasa.png`), vector legend (colors + first-vis markers), high-quality WEBP (quality 98) output. Eliminates all Python runtime deps for the default pipeline. Moon-age value is threaded through but legend display remains a remaining polish item (see TODO.md).

**Why this design?** (documented in skill + perf doc)
- **Reference first**: CPU C++ double is the arbiter; GPU is a high-fidelity accelerator.
- **FP32+DD minimalism**: Full double everywhere on GPU was impossible on Apple Silicon; DD only on the accumulator that actually accumulates error in the horizon search is sufficient and cheap (~few dozen extra ops/pixel).
- **Chebyshev**: GPU-friendly (small constant mem, fast eval, no interp error of prior dense tables).
- **Pure Go blend**: Portability, zero external runtime for end-users, easier distribution and testing.
- **OpenCL + CGO**: Maximum cross-platform GPU reach (Metal, CUDA, ROCm, Compute Runtime) with one codebase; Astronomy Engine reused via C/C++.

The `crescent-moon-visibility-engineering` skill captures these patterns for similar scientific/mixed-language projects. All paths (CPU/GPU) produce overlays suitable for visual sighting predictions because classification + t_best agree with the reference at the 0.1° map resolution used for naked-eye (A/B red diamond) and telescope (C/D blue diamond) first-visibility markers.

See `Makefile` for build orchestration, `.github/workflows/release.yml` + `scripts/release.sh` for release engineering, and `docs/performance-accuracy.md` for the full accuracy story and numerical rationale.

## Setup & Installation

### Prerequisites

1. **Go** 1.22+ (the `go.mod` declares `go 1.25`; builds have been validated on recent Go releases)
2. **C/C++ Compiler** (GCC or Clang with OpenMP support; macOS needs `brew install libomp`)
3. **OpenCL SDK** (optional — only needed for the GPU renderer)

Python + OpenCV/Pillow are only required if you want to run the legacy `gpu_blend.py` (no longer part of the default pipeline).

### Installation

```bash
# Compile everything with make (no Python dependencies required for the core pipeline)
make              # builds CPU renderer, GPU renderer (if available), and Go binary

# Or build individual components:
make cpu          # visibility.out (CPU renderer)
make gpu          # gpu_visibility.out (GPU renderer)
make go           # crescent_maps (Go orchestrator)
```

### GPU dependency installation

| Platform | Headers | Library |
|----------|---------|---------|
| macOS    | built-in (Xcode CLI tools) | `-framework OpenCL` |
| Ubuntu/Debian | `sudo apt install opencl-headers` | `sudo apt install ocl-icd-opencl-dev` |
| Fedora   | `sudo dnf install opencl-headers` | `sudo dnf install ocl-icd-devel` |
| Arch     | `sudo pacman -S opencl-headers` | `sudo pacman -S ocl-icd` |
| AMD ROCm | `/opt/rocm/include/CL/cl.h` (auto-detected) | `/opt/rocm/lib/libOpenCL.so` |
| NVIDIA CUDA | `/usr/local/cuda/include/CL/cl.h` (auto-detected) | `-L/usr/local/cuda/lib64 -lOpenCL` |

If OpenCL is unavailable, the Makefile still builds CPU + Go components:

```bash
make cpu && make go
```

### Platform notes

- **Apple Silicon (M1/M2/M3/M4 and later):** The GPU renderer now works natively and at high speed. The binary auto-detects lack of FP64 support via `CL_DEVICE_DOUBLE_FP_CONFIG` and transparently loads the FP32 + double-double time kernel (`visibility_kernel_fp32.cl`). 
  - Achieves **96.97 % exact per-pixel RGBA match** vs the pure-C++ double reference on M4 Pro hardware (well within the 96–97 % band of the classic FP64 OpenCL path).
  - Typical kernel time: ~78–82 ms for a full 3600×2160 map.
  - The `-gpu` flag works out of the box. See the detailed [Performance and Accuracy](docs/performance-accuracy.md) document for methodology, boundary analysis, and comparison tables.
- **macOS OpenMP:** Apple Clang doesn't accept `-fopenmp` directly. Install Homebrew's libomp (`brew install libomp`) and `make` will auto-detect it. If libomp is missing, the CPU renderer still builds but runs single-threaded (you'll get a `make` warning).
- **Linux:** GCC ships with OpenMP support; no extra setup needed.
- **Cross-platform GPU note:** The FP32+DD technique is standard OpenCL C. It brings high-accuracy crescent mapping to any OpenCL device that previously lacked usable double precision. Traditional FP64 OpenCL devices continue to use the original double kernel with zero behavior change.

## Usage

```bash
# Generate all crescent maps for 2027 with the CPU renderer (default)
./crescent_maps -start 2027 -end 2027

# Generate using the GPU renderer (~4× faster)
./crescent_maps -start 2027 -end 2027 -gpu

# Year range with custom output dir and worker count
./crescent_maps -start 2024 -end 2025 -out maps_24_25 -workers 4 -gpu

# Specific years
./crescent_maps -years "2027,2028,2030" -out future_maps

# Specific months across a year range
./crescent_maps -years 2027 -months "3,9" -gpu     # March and September only

# Render overlays only, skip the blending pass (useful when OpenCV is unavailable)
./crescent_maps -years 2027 -noblend

# Show version information
./crescent_maps -version
```

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-years`   | Comma-separated list of years (e.g., `2027,2028`)             | —         |
| `-start`   | Start year (used if `-years` not set)                         | `2027`    |
| `-end`     | End year (inclusive)                                          | `2027`    |
| `-months`  | Comma-separated months (1-12) to limit which new moons render | all       |
| `-out`     | Output directory                                              | `output_maps` |
| `-workers` | Parallel worker count for the renderer                        | `4`       |
| `-gpu`     | Use GPU renderer instead of CPU                               | `false`   |
| `-noblend` | Skip GPU blending (writes raw overlays only)                  | `false`   |
| `-version` | Print version and build date, then exit                       | `false`   |

### Day selection

For each new moon the orchestrator emits **3 maps**, walking forward day-by-day from the conjunction. It only includes days where the Moon reaches at least **0.2 % illumination** at the *latest sunset anywhere on Earth* for that calendar day (sampled at D+1 06:00 UTC to cover observers near the date line). 

This modern illumination-based rule (implemented in `main.go:201`) is more robust than a simple hour-age cutoff, captures record-young crescents under excellent conditions, and still excludes physically impossible same-day cases. The exact new-moon times are preserved at full sub-second precision from the Astronomy Engine. See the code and [Performance and Accuracy](docs/performance-accuracy.md) for the scientific rationale.

## Output

Each generated map is a **3840 × 2160 (4K) WEBP** at quality 98, typically **1.8 – 2.5 MB**.

### Map color coding

The renderer emits these colors directly (`visibility.cc:223-242`); the legend drawn by the Go blending package (`internal/blend`) matches them exactly:

| Code | Color | RGB | Meaning |
|------|-------|-----|---------|
| **A** | cyan       | `#00CCCC` | Easily visible (naked eye) |
| **B** | darker cyan| `#00B3B3` | Visible, perfect conditions |
| **C** | yellow     | `#FFFF1A` | May need optical aid |
| **D** | yellow     | `#E6E600` | Will need optical aid |
| **E** | olive      | `#B3B300` | Telescope only |
| F / H / I | transparent | — | Not visible / moonset before sunset / no rise-set |
| G / J | transparent | — | Pre-conjunction |

Special markers drawn over the visibility zones:

| Marker | Meaning |
|--------|---------|
| ● red, black ring (`#FF0000`) | First naked-eye visibility (eastmost A/B pixel) |
| ● blue, black ring (`#0000FF`) | First telescope visibility (eastmost C/D pixel) |
| white pixels | Moon-age indicator lines (every 1/20 day) |

Overlay alpha is **60 %** so the underlying NASA base map (continents, oceans) remains visible.

For the most up-to-date performance numbers, accuracy methodology, and validation against visual sighting use-cases, see **[docs/performance-accuracy.md](docs/performance-accuracy.md)**.

## Scientific background

Visibility is computed via:

- **ARCL** — Arc of Light (angular elongation between sun and moon)
- **ARCV** — Arc of Vision (altitude difference between moon and sun at best time)
- **DAZ**  — Azimuth difference between sun and moon
- **W**    — Topocentric crescent width (arcmin)
- **Lag Time** — Time between sunset and moonset (evening) / moonrise and sunrise (morning)

Yallop classifies pixels into A–E (visible) or F (not) based on a cubic in `W` and the resulting `ARCV - threshold` value.

## Performance

| Component | Technique | Benefit |
|-----------|-----------|---------|
| Go orchestrator | Channel-based fan-out workers       | Parallel new-moon processing |
| CPU renderer    | OpenMP per-pixel loop               | Multi-core scaling on any host |
| GPU renderer (FP64 path) | OpenCL + Chebyshev + `native_*` + tightened loops | ~4× CPU at full accuracy on NVIDIA/AMD/etc. |
| GPU renderer (FP32+DD path) | Same + compensated double-double time only for search accumulator | Unlocks Apple Silicon & other float-only OpenCL devices at **96.97 %** pixel match |
| GPU blending    | OpenCV OpenCL T-API                  | GPU-side composite of overlay + base |
| New-moon cache  | In-memory deduplication              | Avoids redundant CGO calls |

**Key accuracy result** (measured on M4 Pro, 2025-01-30 Yallop map):
- FP32 + double-double OpenCL path: **96.97 %** exact RGBA pixel match vs pure double C++ reference.
- Residual differences are almost entirely ULP noise at Yallop decision boundaries — identical character to the classic FP64 OpenCL path on other hardware.

See the dedicated **[Performance and Accuracy](docs/performance-accuracy.md)** document for:
- Detailed benchmark tables (timings, match rates, device comparisons)
- Explanation of the double-double technique and why only time needs it
- Boundary analysis and validation against visual sighting requirements
- Historical numbers (NVIDIA GB10, etc.) and future validation plans

The GPU renderer (both kernels) matches the CPU's classification counts to a very high degree. Full-year runs that previously took ~95 s on CPU now complete in well under 10 s on modern Apple Silicon GPUs when using `-gpu`.

## Installation from GitHub Releases

Pre-built binaries (Go orchestrator + best-effort CPU renderer) are available on the [Releases](https://github.com/jim-fun/crescent-moon-visibility/releases) page. These are produced by the automated workflow (multi-platform, checksums + Cosign keyless signing).

**Recommended:** Download the `crescent_maps-*` orchestrator for your OS/arch (e.g. `linux-amd64`, `darwin-arm64`). The CPU renderer (`visibility-*`) is also attached for reference/validation use.

**GPU renderer note:** Not pre-built (OpenCL is highly platform-dependent). After installing the orchestrator, clone the repo and run `make gpu` (or full `make`) on your target machine for `-gpu` support. See [GPU dependency installation](#gpu-dependency-installation) and the detailed mixed-language [Architecture](#architecture) section above.

Example (Linux/macOS, adjust tag and suffix for your release):

```bash
curl -LO https://github.com/jim-fun/crescent-moon-visibility/releases/download/v0.2.1/crescent_maps-v0.2.1-linux-amd64
chmod +x crescent_maps-v0.2.1-linux-amd64
sudo mv crescent_maps-v0.2.1-linux-amd64 /usr/local/bin/crescent_maps
crescent_maps -version   # reports orchestrator + attempts to query bundled renderers
crescent_maps -help
```

See the expanded release notes on each GitHub Release page (includes architecture summary, verification steps, and links to CHANGELOG + Performance & Accuracy doc) for the authoritative post-release instructions. The workflow attaches LICENSE and README for offline use.

Full release engineering details (including pre-releases via `make release-rc`, Cosign verification, and `scripts/release.sh`): see the [Release Process](#release-process) section below.

## Release Process

### Preparing a Release

Use the convenient make targets:

```bash
make release-patch          # Patch release (e.g. 0.2.0 → 0.2.1)
make release-minor          # Minor release (0.2.0 → 0.3.0)
make release-major          # Major release

# Pre-releases
make release-rc             # Next release candidate (0.2.0 → 0.2.1-rc.1)
make release-beta           # Beta release
```

For full control (including custom pre-release versions):

```bash
./scripts/release.sh patch --rc
./scripts/release.sh minor --beta
./scripts/release.sh 0.3.0-rc.2
```

Then push the tag:

```bash
git push origin main --tags
```

### What the Release Workflow Does

The GitHub Actions workflow (`.github/workflows/release.yml`) will:

- Build the Go orchestrator (`crescent_maps`) for Linux amd64, macOS arm64, and macOS amd64
- Build the CPU reference renderer where possible
- Generate a combined `checksums.txt` for all release artifacts
- Sign `checksums.txt` using **Cosign (keyless)** via GitHub OIDC
- Create a GitHub Release (marked as pre-release for `rc`/`beta`/`alpha` tags)
- Attach binaries + checksums + signatures

### Verifying a Release

```bash
# Download checksums.txt and checksums.txt.sig + .pem from the release

cosign verify-blob checksums.txt \
  --signature checksums.txt.sig \
  --certificate checksums.txt.pem \
  --certificate-identity-regexp 'https://github.com/jim-fun/crescent-moon-visibility' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

See `.github/workflows/release.yml` for the full implementation.

## Testing & Validation

The project includes automated tests for both code correctness and output accuracy:

- `make test` — Runs the full Go test suite, including unit tests for the blending logic and early-skip visibility counting.
- `make test-accuracy` (or `RUN_ACCURACY_TEST=1 go test -run TestRendererAccuracy`) — Executes the CPU vs GPU renderer pixel-match regression test. This validates that the FP32+DD OpenCL path maintains ≥96% exact per-pixel agreement with the pure double-precision C++ reference (the key accuracy guarantee for visual sighting predictions).

See `internal/blend/blend_test.go` and `main_test.go` for the test implementation, and [docs/performance-accuracy.md](docs/performance-accuracy.md) for historical and current accuracy data.

## Credits

- **Original authors:** [@ebraminio](https://github.com/ebraminio), [@hidp123](https://github.com/hidp123)
- **Astronomy Engine:** Don Cross — [cosinekitty/astronomy](https://github.com/cosinekitty/astronomy/)
- **STB Image Write:** Sean Barrett
- **Architecture revamp:** Go orchestrator, Chebyshev GPU ephemeris, dual OpenCL kernels (FP64 + FP32+DD for Apple Silicon), OpenCL/OpenCV cross-platform pipeline (2026)

## License

MIT License. Copyright (c) 2022-present @ebraminio and @hidp123. (See LICENSE and COPYING for full text.)
