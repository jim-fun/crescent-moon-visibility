# Crescent Moon Visibility Maps Generator

A high-performance application for generating accurate crescent moon visibility maps using astronomical calculations. This tool implements well-established visibility criteria to predict when and where the new crescent moon will be visible to observers around the world.

## Architecture Update

This project has been redesigned to be orchestrated entirely in **Golang** to maximize parallel execution and performance, while using cross-platform GPU acceleration for image processing.

- **Golang Orchestrator:** Manages execution, CLI flags, and parallel jobs.
- **Native CGO:** `main.go` seamlessly bridges to `astronomy.c` using CGO to calculate new moon dates natively without Python wrappers.
- **Universal GPU Compositing (T-API):** Uses OpenCV's OpenCL Transparent API (`cv2.ocl`) via a lightweight Python script (`gpu_blend.py`) to run alpha blending and image resizing directly on **macOS (Metal/OpenCL), AMD GPUs (ROCm), and NVIDIA (CUDA)**.
- **C++ CPU OpenMP Rendering:** The core map processing (`visibility.out`) leverages C++ OpenMP for parallel pixel rendering.

## Setup & Installation

### Prerequisites

1. **Golang** (1.20+)
2. **C++ Compiler** (with OpenMP support)
3. **Python 3.x**
4. **OpenCV for Python** (required for the GPU composite pipeline)

### Installation

```bash
# 1. Install OpenCV Python (Headless recommended)
pip install opencv-python-headless pillow numpy

# 2. Compile the C++ visibility renderer
g++ -fopenmp -O3 -Wall -o visibility.out -fno-exceptions \
    -DPIXEL_PER_DEGREE_LON=10 -DPIXEL_PER_DEGREE_LAT=12 \
    -I. cmd/visibility/visibility.cc thirdparty/astronomy.c -lm

# 3. Build the Golang Orchestrator
go build -o crescent_maps .

# 4. Run tests
go test -v .
```

## Usage

Generate crescent maps quickly using the `crescent_maps` binary. The executable calculates the dates and triggers both CPU map generation and universal GPU blending.

```bash
# Generate maps for the year 2027 (default settings)
./crescent_maps -start 2027 -end 2027

# Generate maps for a range of years and output to a specific folder
./crescent_maps -start 2024 -end 2025 -out maps_24_25 -workers 4

# Run specific years using comma-separated values
./crescent_maps -years "2027,2028,2030" -out future_maps
```

### CLI Flags

- `-years`: Comma-separated list of years to process (e.g., `2027,2028`). Overrides `-start` and `-end`.
- `-start`: Start year to process (e.g., `2024`).
- `-end`: End year to process (inclusive, e.g., `2025`).
- `-out`: The output directory for the generated maps. Created automatically if it does not exist (default: `output_maps`).
- `-workers`: Number of parallel CPU workers to run for the `visibility.out` process (default: `4`).

## Features

- **Waxing (Evening) Crescent Visibility** - Predicts visibility after sunset
- **Waning (Morning) Crescent Visibility** - Predicts visibility before sunrise
- **High-Resolution Output** - Configurable resolution up to 4K.
- **Detailed Categorization** - Follows scientific **Yallop** and **Odeh** criteria for visibility thresholds.

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

## Scientific Background

The program calculates crescent visibility using several astronomical parameters:
- **ARCL** (Arc of Light) - The elongation between sun and moon
- **ARCV** (Arc of Vision) - The altitude difference between sun and moon
- **DAZ** - The azimuth difference between sun and moon
- **W** - The crescent width
- **Lag Time** - Time between sunset/sunrise and moonset/moonrise

## Credits

- **Original Authors:** @ebraminio, @hidp123
- **Astronomy Engine:** High-precision astronomical calculations by Don Cross ([cosinekitty/astronomy](https://github.com/cosinekitty/astronomy/))
- **STB Image Write:** Simple image writing library by Sean Barrett
- **Architecture Revamp:** Parallel orchestration and GPU support via Golang / OpenCV OpenCL.

## License

MIT License. Copyright (c) 2023 @ebraminio and @hidp123
