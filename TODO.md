# TODO - Feature Roadmap

This document tracks planned enhancements and features for the Crescent Moon Visibility Maps Generator.

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

- ~~Latitude Capping~~ — Capped at ±60° in `visibility.cc` render loop
- ~~Moon Age Display~~ — Moon age shown in legend via Go → Python pipeline
- ~~Before Conjunction Color Coding~~ — Category 'G' and 'J' now render as semi-transparent dark gray
- ~~GPU Acceleration~~ — OpenCV OpenCL T-API in `gpu_blend.py` (macOS/AMD/NVIDIA)
- ~~Caching and Memoization~~ — New moon dates cached in Go orchestrator (`newMoonCache`)
- ~~Refactoring~~ — CGO bindings extracted to `internal/astro`, C++ renderer moved to `cmd/visibility/`
- ~~Unit Testing~~ — 8 tests in `main_test.go` covering parseYears, astronomy, caching
- ~~Documentation~~ — Inline docs added to `main.go`, `internal/astro/astro.go`, `gpu_blend.py`

**Last Updated:** May 17, 2026

## How to Contribute

If you'd like to work on any of these items:

1. Check if there's an existing issue/PR for the feature
2. Create an issue describing your implementation plan
3. Fork the repository
4. Create a feature branch
5. Submit a pull request with tests and documentation

For questions or discussions, open an issue on GitHub.
