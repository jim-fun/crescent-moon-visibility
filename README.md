# Crescent Moon Visibility Maps Generator

A high-performance application for generating accurate crescent moon visibility maps using astronomical calculations. This tool implements well-established visibility criteria (Yallop and Odeh) to predict when and where the new crescent moon will be visible to observers around the world.

## Architecture

The project is orchestrated entirely in **Go**, with a dual rendering path and a final GPU blending pass:

- **Go Orchestrator (`main.go`)** — Manages CLI flags, computes new-moon times via CGO into the Astronomy Engine, dynamically selects the right calendar days based on a moon-age threshold, and fans tasks out to parallel workers.
- **CPU Renderer (`cmd/visibility/visibility.cc`)** — OpenMP-parallel C++ pixel renderer (`visibility.out`). Reference implementation; produces the canonical output.
- **GPU Renderer (`gpu/gpu_render.c` + `gpu/visibility_kernel.cl`)** — OpenCL host + kernel (`gpu_visibility.out`). Uses Chebyshev polynomial ephemeris fits in `__constant` memory for ~4× speedup over the CPU at 100 % classification agreement.
  - **macOS**: Metal/OpenCL
  - **Linux/NVIDIA**: CUDA driver's OpenCL
  - **Linux/AMD**: ROCm OpenCL
  - **Linux/Intel**: Intel Compute Runtime OpenCL
- **GPU Blending (`gpu_blend.py`)** — Composites the per-pixel visibility overlay onto the NASA base map via OpenCV's OpenCL Transparent API (T-API), then renders a legend and writes WEBP output.

## Setup & Installation

### Prerequisites

1. **Go** 1.20+
2. **C/C++ Compiler** (GCC or Clang with OpenMP support; macOS needs `brew install libomp`)
3. **Python** 3.8+
4. **OpenCV for Python** (used by the blending stage)
5. **OpenCL SDK** (optional — only needed for the GPU renderer)

### Installation

```bash
# 1. Install Python dependencies
pip install opencv-python-headless pillow numpy

# 2. Compile everything with make
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

- **Apple Silicon (M1/M2/M3/M4):** the kernel uses double-precision math, which Metal-backed OpenCL on Apple Silicon does **not** expose. The GPU binary detects this at startup and emits a clear error — use the CPU renderer (the default) on those machines.
- **macOS OpenMP:** Apple Clang doesn't accept `-fopenmp` directly. Install Homebrew's libomp (`brew install libomp`) and `make` will auto-detect it. If libomp is missing, the CPU renderer still builds but runs single-threaded (you'll get a `make` warning).
- **Linux:** GCC ships with OpenMP support; no extra setup needed.

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

For each new moon the orchestrator emits **3 maps**, walking forward day-by-day from the conjunction and including only days where the crescent is at least **12 h old at 18:00 UTC** (defined in `main.go`). This skips physically impossible same-day cases while including record-young crescents when the geometry permits.

## Output

Each generated map is a **3840 × 2160 (4K) WEBP** at quality 98, typically **1.8 – 2.5 MB**.

### Map color coding

The renderer emits these colors directly (`visibility.cc:223-242`); the legend in `gpu_blend.py` matches them exactly:

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
| GPU renderer    | OpenCL kernel + Chebyshev ephemeris in `__constant` memory + `native_sin`/`native_cos` + reduced search-loop counts | ~4× the CPU at full accuracy |
| GPU blending    | OpenCV OpenCL T-API                  | GPU-side composite of overlay + base |
| New-moon cache  | In-memory deduplication              | Avoids redundant CGO calls |

The GPU renderer matches the CPU's classification counts to **100.0 %** (per-pixel exact match ≈ 97 %; the residual 3 % is boundary pixels at the Yallop value thresholds where ULP-level differences flip the discrete class). Full year of 39 maps renders in ~23 s on an NVIDIA GB10 vs ~95 s on CPU.

## Credits

- **Original authors:** [@ebraminio](https://github.com/ebraminio), [@hidp123](https://github.com/hidp123)
- **Astronomy Engine:** Don Cross — [cosinekitty/astronomy](https://github.com/cosinekitty/astronomy/)
- **STB Image Write:** Sean Barrett
- **Architecture revamp:** Go orchestrator, Chebyshev GPU ephemeris, OpenCL/OpenCV cross-platform pipeline

## License

MIT License. Copyright (c) 2023 @ebraminio and @hidp123.
