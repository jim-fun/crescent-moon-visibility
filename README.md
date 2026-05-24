# Crescent Moon Visibility Maps Generator

A high-performance application for generating accurate crescent moon visibility maps using astronomical calculations. This tool implements well-established visibility criteria (Yallop and Odeh) to predict when and where the new crescent moon will be visible to observers around the world.

## Architecture

The project is orchestrated entirely in **Go**, with a dual rendering path and a final GPU blending pass:

- **Go Orchestrator (`main.go`)** — Manages CLI flags, computes new-moon times via CGO into the Astronomy Engine, dynamically selects the right calendar days based on a moon-age threshold, and fans tasks out to parallel workers.
- **CPU Renderer (`cmd/visibility/visibility.cc`)** — OpenMP-parallel C++ pixel renderer (`visibility.out`). Reference implementation; produces the canonical output.
- **GPU Renderer (`gpu/gpu_render.c` + `gpu/visibility_kernel.cl` and `gpu/visibility_kernel_fp32.cl`)** — OpenCL host + kernels (`gpu_visibility.out`). Uses Chebyshev polynomial ephemeris fits in `__constant` memory.
  - Two kernels for maximum compatibility and accuracy:
    - `visibility_kernel.cl`: Full double-precision (FP64) path on devices that support it (NVIDIA, AMD, Intel, macOS Intel, etc.).
    - `visibility_kernel_fp32.cl`: FP32 + double-double (DD) compensated time path for devices without reliable FP64 (Apple Silicon M1/M2/M3/M4 via Metal-backed OpenCL, and other float-only OpenCL implementations).
  - Both paths deliver ~96.97–97 % per-pixel match vs the CPU reference while preserving Yallop classification boundaries for visual sighting predictions.
  - Platforms: macOS (Metal/OpenCL), Linux/NVIDIA (CUDA OpenCL), Linux/AMD (ROCm), Linux/Intel (Compute Runtime). See [docs/performance-accuracy.md](docs/performance-accuracy.md) for detailed benchmarks and accuracy data.
- **GPU Blending (`internal/blend`)** — Pure-Go compositing of the visibility overlay onto the NASA base map (60% alpha), legend rendering, and high-quality WEBP output. The legacy `gpu_blend.py` is no longer required for the default pipeline.

## Setup & Installation

### Prerequisites

1. **Go** 1.22+
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

### Day selection

For each new moon the orchestrator emits **3 maps**, walking forward day-by-day from the conjunction. It only includes days where the Moon reaches at least **0.2 % illumination** at the *latest sunset anywhere on Earth* for that calendar day (sampled at D+1 06:00 UTC to cover observers near the date line). 

This modern illumination-based rule (implemented in `main.go:201`) is more robust than a simple hour-age cutoff, captures record-young crescents under excellent conditions, and still excludes physically impossible same-day cases. The exact new-moon times are preserved at full sub-second precision from the Astronomy Engine. See the code and [PERFORMANCE_AND_ACCURACY.md](PERFORMANCE_AND_ACCURACY.md) for the scientific rationale.

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

MIT License. Copyright (c) 2023 @ebraminio and @hidp123.
