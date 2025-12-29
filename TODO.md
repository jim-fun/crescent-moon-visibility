# TODO - Feature Roadmap

This document tracks planned enhancements and features for the Crescent Moon Visibility Maps Generator.

## High Priority Features

### 1. Latitude Capping
- **Task**: Cap latitude at -60° & +60°
- **Rationale**: Extreme polar regions have unusual sunset/sunrise behavior that can cause edge cases
- **Implementation**: Add latitude bounds checking in the render function
- **Impact**: Improved stability and more reliable results

### 2. Moon Age Display
- **Task**: Add moon age information to map visualization
- **Rationale**: Users want to know the elapsed time since new moon
- **Implementation**:
  - Display moon age contour lines (already partially implemented with white lines)
  - Add moon age value to image annotation
  - Include in legend or map metadata
- **Impact**: Better understanding of lunar phase timing

### 3. Before Conjunction Color Coding
- **Task**: Implement distinct color for regions before astronomical conjunction
- **Rationale**: Distinguish between pre-conjunction (old crescent from previous lunation) and post-conjunction (new crescent)
- **Implementation**:
  - Add color category for pre-conjunction regions
  - Update rendering logic to detect this state
  - Document the new color in README
- **Status**: Partially implemented (category 'G' exists but rendered as black)
- **Impact**: More accurate astronomical representation

## Medium Priority Features

### 4. Additional Visibility Criteria
- **Task**: Implement alternative crescent visibility criteria
- **Options**:
  - Schaefer criterion (1988)
  - Bruin criterion
  - SAAO (South African Astronomical Observatory) criterion
  - Ilyas criterion
- **Rationale**: Different criteria may be preferred by different communities
- **Impact**: Broader applicability and validation opportunities

### 5. Atmospheric Extinction Modeling
- **Task**: Add atmospheric extinction variations
- **Details**:
  - Seasonal atmospheric density changes
  - Altitude-dependent atmospheric models
  - Geographic variation in atmospheric conditions
- **Impact**: Improved accuracy, especially at lower altitudes

### 6. Terrain Elevation Integration
- **Task**: Incorporate terrain elevation data
- **Implementation**:
  - Integrate SRTM or similar elevation datasets
  - Adjust horizon calculations for mountains/valleys
  - Account for elevated observation points
- **Impact**: More realistic visibility predictions for terrestrial observers

### 7. Observer Experience Factor
- **Task**: Add observer experience parameter
- **Details**:
  - Beginner vs expert observer adjustments
  - Visual acuity factors
  - Age-related visibility corrections
- **Impact**: Personalized predictions

## Low Priority / Future Enhancements

### 8. Web-Based Interface
- **Task**: Create web UI for interactive map generation
- **Features**:
  - Date picker
  - Location selector
  - Criteria selection
  - Real-time rendering
  - Download functionality
- **Technologies**: WebAssembly (compile C++ to WASM), JavaScript frontend
- **Impact**: Easier accessibility for non-technical users

### 9. Real-Time Visibility Predictions
- **Task**: Automatic calculation for upcoming new moons
- **Features**:
  - Automated scheduling
  - Email/SMS notifications
  - Location-based alerts
- **Impact**: Proactive user notifications

### 10. Historical Sighting Database
- **Task**: Validate predictions against historical sighting reports
- **Data Sources**:
  - ICOP (Islamic Crescents' Observation Project)
  - Historical records
  - Observatory reports
- **Impact**: Validation and calibration of models

### 11. Configurable Map Projections
- **Task**: Support multiple map projections
- **Options**:
  - Mercator (current)
  - Equirectangular
  - Robinson
  - Mollweide
- **Impact**: Better visualization for different use cases

### 12. Multi-Language Support
- **Task**: Internationalize output and annotations
- **Languages**: Arabic, English, French, Turkish, Urdu, Malay, etc.
- **Impact**: Global accessibility

## Code Quality Improvements

### 13. Unit Testing
- **Task**: Add comprehensive test suite
- **Coverage**:
  - Astronomical calculation accuracy
  - Edge cases (polar regions, date boundaries)
  - Regression tests
- **Framework**: GoogleTest or similar

### 14. Documentation
- **Task**: Add inline code documentation
- **Details**:
  - Doxygen-style comments
  - Algorithm explanations
  - Parameter descriptions
- **Impact**: Better code maintainability

### 15. Refactoring
- **Task**: Code organization improvements
- **Areas**:
  - Separate astronomical calculations into library
  - Modularize rendering logic
  - Extract constants to configuration file
- **Impact**: Easier maintenance and extension

## Performance Optimizations

### 16. GPU Acceleration
- **Task**: Implement CUDA/OpenCL rendering
- **Expected Gain**: 10-100x speedup for high-resolution maps
- **Impact**: Real-time rendering of 4K+ resolution maps

### 17. Caching and Memoization
- **Task**: Cache repeated calculations
- **Targets**:
  - New moon times
  - Sun/moon ephemeris data
- **Impact**: Faster batch processing

## Research and Validation

### 18. Accuracy Analysis
- **Task**: Systematic comparison with observational data
- **Methodology**:
  - Compare with HMNAO predictions
  - Validate against sighting reports
  - Statistical analysis of differences
- **Output**: Research paper or technical report

### 19. Sensitivity Analysis
- **Task**: Analyze parameter sensitivity
- **Parameters**:
  - Atmospheric refraction model
  - Best time calculation (4/9 ratio)
  - Crescent width threshold
- **Impact**: Understanding of model robustness

---

**Last Updated:** December 28, 2025

## How to Contribute

If you'd like to work on any of these items:

1. Check if there's an existing issue/PR for the feature
2. Create an issue describing your implementation plan
3. Fork the repository
4. Create a feature branch
5. Submit a pull request with tests and documentation

For questions or discussions, open an issue on GitHub.
