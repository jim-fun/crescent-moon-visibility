# Crescent Moon Visibility Maps Generator

A high-performance C++ application for generating accurate crescent moon visibility maps using astronomical calculations. This tool implements well-established visibility criteria to predict when and where the new crescent moon will be visible to observers around the world.

## Overview

This project generates crescent moon visibility maps based on two internationally recognized scientific criteria:

- [Yallop criteria](https://astro.ukho.gov.uk/download/NAOTN69.pdf) - Developed by B.D. Yallop at HM Nautical Almanac Office
- [Odeh criteria](https://www.astronomycenter.net/pdf/2006_cri.pdf) - Developed by Mohammad Odeh at the Astronomy Center

### Key Features

- **Waxing (Evening) Crescent Visibility** - Predicts visibility after sunset
- **Waning (Morning) Crescent Visibility** - Predicts visibility before sunrise
- **Special Markers** - Moonset before sunset shown in red (similarly moonrise after sunrise)
- **High-Resolution Output** - Configurable resolution up to 4K (16 pixels per degree)
- **Parallel Processing** - OpenMP support for faster rendering
- **Detailed Table Output** - Generate TSV data with astronomical parameters
- **First Visibility Markers** - Diamond markers showing earliest visibility points

## Scientific Background

The program calculates crescent visibility using several astronomical parameters:

- **ARCL** (Arc of Light) - The elongation between sun and moon
- **ARCV** (Arc of Vision) - The altitude difference between sun and moon
- **DAZ** - The azimuth difference between sun and moon
- **W** - The crescent width
- **Lag Time** - Time between sunset/sunrise and moonset/moonrise
- **Moon Age** - Time elapsed since new moon
- **Best Time** - Optimal time for observation

### Visibility Categories

#### Yallop Criteria
- **A** - Easily visible to the naked eye
- **B** - Visible under perfect atmospheric conditions
- **C** - May need optical aid to find crescent
- **D** - Will need optical aid to find crescent
- **E** - Not visible with telescope
- **F** - Below any realistic detection limit

#### Odeh Criteria
- **A** - Visible by naked eye
- **C** - Visible by optical aid
- **E** - Visible only by optical aid
- **F** - Not visible

# Output Examples

## Wed, 29 June 2022 (29 ZH 1443H) Evening Crescent

### Yallop

![2022-06-29 (E) Yallop](https://user-images.githubusercontent.com/833473/193407627-e8895f15-7d6f-46c2-9c7f-a770131ad387.png)

For comparison
![2022-06-29 (E) Yallop (HMNAO)](https://user-images.githubusercontent.com/84683703/191850568-3f661abb-74f2-4720-b256-1404d69757cc.jpg)

### Odeh

![2022-06-29 (E) Odeh](https://user-images.githubusercontent.com/833473/193407716-07674584-06c5-47eb-944b-5a6a8ba182bb.png)
  
For comparison
![image](https://user-images.githubusercontent.com/84683703/191850739-bd009136-5e8d-4d0f-ba1d-aac2ace6a564.png)

# Installation and Build Instructions

## Prerequisites

The project requires the following dependencies:

- **C++ Compiler** with C++11 support and OpenMP
  - Linux: GCC (gcc-c++ or g++ package)
  - macOS: LLVM from Homebrew
  - Windows: MinGW-w64 via MSYS2
- **ImageMagick** - For image composition and annotation
- **Python 3** (optional) - For new moon date calculation utilities

## Platform-Specific Setup

### Linux

```bash
# Install dependencies (Ubuntu/Debian)
sudo apt-get install build-essential imagemagick

# Install dependencies (Fedora/RHEL)
sudo dnf install gcc-c++ ImageMagick

# Build and run
./run.sh 2026-01-18
```

### macOS

```bash
# Install dependencies
brew install llvm imagemagick

# Build and run (the script automatically detects macOS)
./run.sh 2026-01-18
```

### Windows

```bash
# Install MSYS2 from https://www.msys2.org/
# Then in MSYS2 terminal:
pacman -S mingw-w64-x86_64-gcc

# Build
./build-mingw.sh
```

## Configuration Options

### Resolution Settings

The resolution can be adjusted by setting the `RESOLUTION` environment variable:

```bash
# HD resolution (4 pixels per degree) - Fast
RESOLUTION=4 ./run.sh 2026-01-18

# Full HD resolution (8 pixels per degree) - Balanced
RESOLUTION=8 ./run.sh 2026-01-18

# 4K resolution (16 pixels per degree) - High quality (default)
RESOLUTION=16 ./run.sh 2026-01-18
```

### Prevent Auto-Opening

By default, the generated image opens automatically. To disable:

```bash
NOOPEN=1 ./run.sh 2026-01-18
```

## Usage

### Generating Maps

The compiled binary supports multiple modes:

#### 1. Map Generation Mode

```bash
./visibility.out YYYY-MM-DD map <evening|morning> <yallop|odeh> output.png
```

Examples:
```bash
# Evening crescent using Yallop criteria
./visibility.out 2026-01-18 map evening yallop evening_yallop.png

# Morning crescent using Odeh criteria
./visibility.out 2026-01-18 map morning odeh morning_odeh.png
```

#### 2. Table Generation Mode

Generate detailed astronomical data in TSV format:

```bash
./visibility.out YYYY-MM-DD table LAT,LON,ALT DAYS > output.tsv
```

Example:
```bash
# Generate 100 days of data for Mecca
./visibility.out 2026-01-18 table 21.4225,39.8262,0 100 > mecca_data.tsv
```

Parameters:
- `LAT` - Latitude in decimal degrees (positive for North)
- `LON` - Longitude in decimal degrees (positive for East)
- `ALT` - Altitude in meters above sea level
- `DAYS` - Number of consecutive days to calculate

#### 3. Table (Ignore Best Time) Mode

```bash
./visibility.out YYYY-MM-DD table-ignore-besttime LAT,LON,ALT DAYS > output.tsv
```

This mode uses sunset/sunrise time instead of the calculated "best time" for observations.

### Finding New Moon Dates

Use the Python utility to find new moon dates for any year:

```bash
# Get new moons for current year
python3 get_new_moons.py

# Get new moons for specific year
python3 get_new_moons.py 2026
```

## Output Format

### Map Output

Generated PNG images include:
- **Color-coded visibility regions** based on criteria
- **White contour lines** showing moon age isolines
- **Red diamond** - First location visible by naked eye
- **Blue diamond** - First location visible with optical aid
- **Map overlay** - Geographic boundaries and coastlines
- **Annotation** - Date, criteria type, and visibility type

### Table Output

TSV file with columns including:
- Date and location parameters
- Sun and moon rise/set times
- New moon times (previous and next)
- Moon age calculations
- Lag time
- Best observation time
- Visibility classifications (Yallop and Odeh)
- Q and V values
- Detailed astronomical parameters (azimuth, altitude, RA, Dec, etc.)

## Project Structure

```
crescent-moon-visibility/
├── visibility.cc           # Main C++ source file with calculation logic
├── run.sh                  # Build and execution script for Linux/macOS
├── build-mingw.sh         # Build script for Windows (MinGW)
├── get_new_moons.py       # Python utility to calculate new moon dates
├── map.png                # Base world map for overlay
├── thirdparty/            # Third-party libraries
│   ├── astronomy.c        # Astronomy Engine C implementation
│   ├── astronomy.h        # Astronomy Engine header
│   └── stb_image_write.h  # STB image writing library
└── README.md              # This file
```

## Technical Details

### Algorithm Overview

The visibility calculation follows these steps:

1. **Time Adjustment** - Adjusts base time by longitude to account for timezone
2. **Sun/Moon Events** - Calculates sunset/sunrise and moonset/moonrise times
3. **Lag Time Calculation** - Determines time difference between sun and moon events
4. **Best Time Selection** - Computes optimal observation time (4/9 of lag time after sunset)
5. **New Moon Search** - Finds previous and next new moon times
6. **Moon Age Calculation** - Computes elapsed time since new moon
7. **Astronomical Parameters** - Calculates:
   - Moon and sun positions (azimuth, altitude, RA, Dec)
   - Semi-diameter and lunar parallax
   - Topocentric corrections
   - Arc of light (ARCL) - geocentric for Yallop, topocentric for Odeh
   - Arc of vision (ARCV)
   - Crescent width (W)
8. **Visibility Classification** - Applies Yallop or Odeh formulas to determine visibility category

### Performance Optimizations

- **OpenMP Parallelization** - Multi-threaded rendering for faster map generation
- **Template-based Design** - Compile-time optimization for evening/morning and Yallop/Odeh modes
- **Efficient Memory Layout** - Direct pixel buffer manipulation
- **Optimized Resolution** - Configurable pixels-per-degree for speed/quality trade-off

### Color Coding

Maps use the following color scheme (ABGR format):

**Yallop Criteria:**
- Category A (Easily visible): Yellow (#CCCC00)
- Category B (Perfect conditions): Gold (#B3B300)
- Category C (May need aid): Cyan (#FFFF1A)
- Category D (Need aid): Light cyan (#E6E600)
- Category E (Telescope): Dark cyan (#B3B300)

**Odeh Criteria:**
- Category A (Naked eye): Yellow (#CCCC00)
- Category C (Optical aid): Cyan (#FFFF1A)
- Category E (Optical aid only): Dark cyan (#B3B300)

**Special Markers:**
- Moon age contours: White
- First naked-eye visibility: Red diamond
- First telescope visibility: Blue diamond
- Moonset before sunset: Black (no visibility)

## Use Cases

This tool is valuable for:

- **Religious Calendar Determination** - Islamic calendar month beginnings
- **Astronomical Research** - Crescent visibility studies
- **Educational Purposes** - Teaching lunar phases and astronomical calculations
- **Photography Planning** - Planning crescent moon photography sessions
- **Historical Analysis** - Retroactive calculation of historical lunar sightings

## Accuracy and Limitations

### Accuracy
- Uses high-precision astronomical algorithms from Astronomy Engine
- Accounts for:
  - Topocentric parallax corrections
  - Atmospheric refraction effects
  - Earth's oblateness
  - Lunar libration
  - Geocentric vs topocentric coordinates (method-dependent)

### Limitations
- Does not account for local weather conditions
- Does not model atmospheric extinction variations
- Assumes sea-level horizon (mountain/valley effects not included)
- Does not include observer experience factors
- Fixed atmospheric model (no seasonal variations)

## Contributing

Contributions are welcome. Areas for improvement include:

- Additional visibility criteria (e.g., Schaefer, Bruin, SAAO)
- Atmospheric extinction modeling
- Terrain elevation data integration
- Observer experience factors
- Web-based interface
- Real-time visibility predictions
- Historical sighting database validation

## References

### Scientific Papers
1. Yallop, B.D. et al. (1998) "A Method for Predicting the First Sighting of the New Crescent Moon", NAO Technical Note No. 69
2. Odeh, M.S. (2006) "New Criterion for Lunar Crescent Visibility", Experimental Astronomy, 18, 39-64
3. Schaefer, B.E. (1988) "Visibility of the Lunar Crescent", Quarterly Journal of the Royal Astronomical Society, 29, 511-523

### Related Resources
- [International Astronomical Center](https://www.astronomycenter.net/) - Lunar crescent visibility
- [HM Nautical Almanac Office](https://astro.ukho.gov.uk/) - Astronomical data and publications
- [Islamic Crescents' Observation Project (ICOP)](http://www.icoproject.org/) - Crescent sighting reports

# Credits

## Authors
- @ebraminio
- @hidp123

## Dependencies
- [Astronomy Engine](https://github.com/cosinekitty/astronomy/) - High-precision astronomical calculations by Don Cross
- [STB Image Write](https://github.com/nothings/stb) - Simple image writing library by Sean Barrett
- [ImageMagick](https://imagemagick.org/) - Image manipulation and composition
- [PyEphem](https://rhodesmill.org/pyephem/) - Python astronomical calculations (optional utility)

# License

MIT License

Copyright (c) 2023 @ebraminio and @hidp123

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

---

**Last Updated:** December 28, 2025
