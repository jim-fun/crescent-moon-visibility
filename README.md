# Crescent Moon Visibility Maps Generator

A high-performance application for generating accurate crescent moon visibility maps using astronomical calculations. This tool implements well-established visibility criteria to predict when and where the new crescent moon will be visible to observers around the world.

## Architecture

This project is orchestrated entirely in **Golang**, with a dual rendering path:

- **Golang Orchestrator** — Manages execution, CLI flags, parallel jobs, and new moon computation via CGO.
- **CPU Renderer** — C++ OpenMP pixel rendering in `cmd/visibility/visibility.cc` (`visibility.out`).
- **GPU Renderer** — OpenCL compute via `gpu/gpu_render.c` + `gpu/visibility_kernel.cl` (`gpu_visibility.out`).
  - **macOS**: Metal/OpenCL
  - **Linux/NVIDIA**: CUDA via OpenCL
  - **Linux/AMD**: ROCm via OpenCL
  - **Linux/Intel**: Intel GPU Compute Runtime via OpenCL
- **GPU Blending** — OpenCV OpenCL T-API in `gpu_blend.py` composites generated maps onto the NASA base map.

## Setup & Installation

### Prerequisites

1. **Golang** (1.20+)
2. **C++ Compiler** (GCC or Clang with OpenMP support)
3. **Python 3.x**
4. **OpenCV for Python** (for GPU blending pipeline)
5. **OpenCL SDK** (for GPU renderer — optional)

### Installation

```bash
# 1. Install Python dependencies
pip install opencv-python-headless pillow numpy

# 2. Compile everything with make
make              # builds CPU renderer, GPU renderer (if headers found), and Go binary

# Build individual components:
make cpu          # compile visibility.out
make gpu          # compile gpu_visibility.out
make go           # compile crescent_maps
```

**GPU renderer build requirements by platform:**

| Platform | Header Location | OpenCL Library |
|----------|----------------|----------------|
| macOS | built-in | `-framework OpenCL` |
| Linux/NVIDIA | `/usr/include/CL/cl.h` | `-lOpenCL` (from `ocl-icd`) |
| Linux/AMD ROCm | `/opt/rocm/include/CL/cl.h` | `/opt/rocm/lib/libOpenCL.so` |
| Linux/NVIDIA CUDA | `/usr/local/cuda/include/CL/cl.h` | `-L/usr/local/cuda/lib64 -lOpenCL` |

### Build Troubleshooting

```bash
# If GPU renderer fails to build due to missing OpenCL headers/library:
# Ubuntu/Debian: sudo apt install ocl-icd-opencl-dev opencl-headers
# Fedora: sudo dnf install ocl-icd-devel opencl-headers
# Arch: sudo pacman -S ocl-icd opencl-headers
# macOS: OpenCL is built-in — ensure Xcode CLI tools are installed

# Build only CPU + Go if OpenCL headers are unavailable:
make cpu && make go
```

## Usage

Generate crescent maps using the `crescent_maps` binary:

```bash
# Generate maps for 2027 with default CPU renderer
./crescent_maps -start 2027 -end 2027

# Generate maps with GPU renderer
./crescent_maps -start 2027 -end 2027 -gpu

# Generate for a year range with custom output and workers
./crescent_maps -start 2024 -end 2025 -out maps_24_25 -workers 4 -gpu

# Run specific years
./crescent_maps -years "2027,2028,2030" -out future_maps
```

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-years` | Comma-separated list of years (e.g., `2027,2028`) | — |
| `-start` | Start year (if `-years` not set) | `2027` |
| `-end` | End year (inclusive) | `2027` |
| `-out` | Output directory for generated maps | `output_maps` |
| `-workers` | Number of parallel CPU workers for CPU renderer | `4` |
| `-gpu` | Use GPU renderer (`gpu_visibility.out`) instead of CPU | `false` |

## Features

- **Waxing (Evening) Crescent Visibility** — Predicts visibility after sunset
- **Waning (Morning) Crescent Visibility** — Predicts visibility before sunrise
- **High-Resolution Output** — Configurable resolution (4x for CPU, 10x for GPU)
- **Detailed Categorization** — Follows scientific Yallop and Odeh criteria

## Map Color Coding (Yallop Criteria)

- **A (Easily visible):** Yellow (#CCCC00)
- **B (Perfect conditions):** Gold (#B3B300)
- **C (May need aid):** Cyan (#FFFF1A)
- **D (Need aid):** Light cyan (#E6E600)
- **E (Telescope):** Dark cyan (#B3B300)

*Special Markers:*
- First naked-eye visibility: Red diamond
- First telescope visibility: Blue diamond
- Moonset before sunset: Black (no visibility)
- Before conjunction (pre-new moon): Semi-transparent dark gray (#404040)
- New moon age indicator: White line every 1/20 day

## Scientific Background

The program calculates crescent visibility using:
- **ARCL** (Arc of Light) — Elongation between sun and moon
- **ARCV** (Arc of Vision) — Altitude difference between sun and moon
- **DAZ** — Azimuth difference between sun and moon
- **W** — Crescent width
- **Lag Time** — Time between sunset/sunrise and moonset/moonrise

## Performance

| Component | Technique | Benefit |
|-----------|-----------|---------|
| Go orchestrator | Channel-based fan-out workers | Parallel new moon processing |
| CPU renderer | OpenMP parallel pixel rendering | Multi-core CPU utilization |
| GPU renderer | OpenCL pixel-parallel kernel | Offload computation to GPU |
| GPU blending | OpenCV OpenCL T-API | GPU-composited final maps |
| New moon cache | In-memory deduplication | Avoids redundant CGO calls |

## Credits

- **Original Authors:** @ebraminio, @hidp123
- **Astronomy Engine:** High-precision calculations by Don Cross ([cosinekitty/astronomy](https://github.com/cosinekitty/astronomy/))
- **STB Image Write:** Simple image writing by Sean Barrett
- **Architecture Revamp:** Parallel orchestration and cross-platform GPU via Golang / OpenCV OpenCL / OpenCL

## License

MIT License. Copyright (c) 2023 @ebraminio and @hidp123
