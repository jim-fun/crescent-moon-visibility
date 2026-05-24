# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Pure Go blending engine (`internal/blend`) — removed Python dependency for normal usage
- Comprehensive accuracy test suite (`make test` and `make test-accuracy`)
- Proper versioning system with `VERSION` file and `-version` flag
- Release infrastructure (scripts + GitHub Actions workflow with checksums + cosign signing)

### Changed
- Project reorganization (`bin/`, `data/`, `docs/`, `scripts/legacy/`)
- Improved Apple Silicon / Metal support via FP32 + double-double OpenCL kernel

## [0.2.0] - 2026-05-XX

### Added
- Full support for Apple Silicon via FP32 + double-double time kernel
- Pure-Go final blending step
- Automated release workflow with signing and checksums

[Unreleased]: https://github.com/jim-fun/crescent-moon-visibility/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/jim-fun/crescent-moon-visibility/releases/tag/v0.2.0