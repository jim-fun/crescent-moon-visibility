# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.1] - 2026-05-24

### Added
- Hour markers on moon-age lines in the legend for improved readability.

### Changed
- Extensive legend visual overhaul driven by user feedback:
  - Two-column layout (A–E categories on left with color swatches; first-visibility diamonds on right).
  - Doubled font sizes using embedded Inter (44pt headers, 30pt labels).
  - Rounded corners (16px radius) with matching 1px dark border.
  - Reduced whitespace and tighter padding/spacing while preserving breathing room and aesthetics.
  - GitHub repo reference right-aligned inside the legend box.
  - Human-readable dates without leading zeros (e.g., "January 8, 2026").
- First-visibility diamonds now rendered exclusively in the pure-Go compositor (`internal/blend`). Removed C++ baking of red/blue diamonds and the associated post-processing strip logic for simpler, more consistent output between CPU and GPU renderers.
- Stronger and more robust diamond deduplication (relaxed color thresholds + explicit neighborhood clearing + post-blend safety pass) to guarantee exactly one clean outlined diamond per type on CPU maps.

### Fixed
- Duplicate first-visibility diamonds appearing on CPU-generated maps (multiple iterations of clearing logic, color-distance heuristics, and radius-based neighborhood wipes).
- Various legend positioning, sizing, and text formatting issues reported during iterative visual testing.

### Internal / Maintenance
- Final public main cleanup before release (removal of GITEA_HANDOFF.md, experimental outmaps/ test artifacts, AI tooling directories, build junk; hardened .gitignore).

## [0.4.0] - 2026-05-24

### Changed
- Version bump to 0.4.0
- Documentation and release process hygiene updates for the clean public GitHub mirror
- Improved contributor attribution in README
- Updated all version examples and references to 0.4.0

## [0.3.0] - 2026-05-24

### Added
- Pure Go blending engine (`internal/blend`) — removed Python dependency for normal usage
- Comprehensive accuracy test suite (`make test` and `make test-accuracy`)
- Proper versioning system with `VERSION` file and `-version` flag
- Release infrastructure (scripts + GitHub Actions workflow with checksums + cosign signing)
- Enhanced Architecture documentation in README with detailed mixed-language rationale (Go + C++ + dual OpenCL kernels + Chebyshev + FP32+DD)

### Changed
- Public GitHub repository restarted clean (https://github.com/jim-fun/crescent-moon-visibility) as the official public mirror. Internal maintainer tooling (agentic workflows, review artifacts) remains only on the private Gitea dev branch and the central agentic repository.
- Project reorganization (`bin/`, `data/`, `docs/`, `scripts/legacy/`)
- Improved Apple Silicon / Metal support via FP32 + double-double OpenCL kernel
- Documentation & Architecture refresh for clarity, accuracy guarantees, design choices (FP32+DD only on time accumulators, Chebyshev rationale), and long-term maintainability (cross-refs to `crescent-moon-visibility-engineering` skill)
- Fixed stale links and minor descriptive inaccuracies (day selection, license years, Go version notes)

### Fixed
- Incorrect cross-reference link to performance-accuracy document in day-selection section

## [0.2.1] - 2026-05-24

### Changed
- Documentation and architecture improvements (this release focuses on high-quality, maintainable docs aligned with the completed Python-removal and FP32+DD milestones).

## [0.2.0] - 2026-05-XX

### Added
- Full support for Apple Silicon via FP32 + double-double time kernel
- Pure-Go final blending step
- Automated release workflow with signing and checksums

[Unreleased]: https://github.com/jim-fun/crescent-moon-visibility/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/jim-fun/crescent-moon-visibility/releases/tag/v0.4.0
[0.3.0]: https://github.com/jim-fun/crescent-moon-visibility/releases/tag/v0.3.0
[0.2.1]: https://github.com/jim-fun/crescent-moon-visibility/releases/tag/v0.2.1
[0.2.0]: https://github.com/jim-fun/crescent-moon-visibility/releases/tag/v0.2.0