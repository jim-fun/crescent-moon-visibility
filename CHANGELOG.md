# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.3] - 2026-05-25

### Fixed
- **Windows CPU build** (`error: too many decimal points in number`): the version
  macro was passed as `-DVERSION_STR="x.y.z"`, but the shell (PowerShell, and bash)
  strips the quotes, so `g++` received the bare pp-number and rejected it. Removing
  the quotes then exposed a second issue: PowerShell splits the bare `x.y.z` token,
  feeding `.y.z` to the linker. Fixed by stringizing the macro in
  `cmd/visibility/visibility.cc` (so an unquoted value compiles) while keeping the
  quotes in the workflows (so PowerShell passes a single token). The same fix is
  applied to the Windows leg of `release.yml`.
- `Makefile`: the `VERSION` definition is now placed above `CPU_CFLAGS`/`GPU_CFLAGS`
  (which use `:=` immediate expansion), so local `make` builds embed a non-empty
  version string. Quote-escaping of the define removed (now handled in C++).

### Changed
- On-push CI build is now a **Linux + Windows matrix** (`.github/workflows/build.yml`,
  renamed from `windows-cpu.yml`): every push to `main`/`dev` and PRs to `main` build
  the CPU renderer + Go orchestrator on both platforms and run `go test`.

## [0.5.2] - 2026-05-25

### Added
- Windows x64 (CPU-only) release artifacts now produced by the automated workflow.
  - Dedicated CI workflow (`.github/workflows/windows-cpu.yml`) for ongoing validation on push/PR.
  - Full integration into the tagged release process (`.github/workflows/release.yml`).
  - Pre-built `crescent_maps-*-windows-amd64.exe` + CPU renderer included in releases (GPU remains Linux/macOS only via local `make`).
- Cross-platform binary discovery in the Go orchestrator (`main.go`) and tests (`main_test.go`) using `runtime.GOOS` + suffixed candidates.

### Changed
- Windows release builds now include MinGW toolchain setup, static linking attempts for the renderer, and `go test` execution.
- Documentation (README, release notes body) updated with Windows usage examples and "CPU only" caveats.
- `TODO.md` updated with detailed phased plan (now in active implementation).

## [0.5.1] - 2026-05-25

### Added
- `.github/SECURITY.md` (GitHub security policy and vulnerability reporting guidelines).
- CodeQL static analysis workflow (`.github/workflows/codeql.yml`) for Go and C++ code, running on pushes to main/dev, pull requests, and weekly schedule.

### Fixed
- Release workflow (`release.yml`): Fixed artifact download so built binaries, checksums, and signatures are correctly attached to GitHub Releases. Added missing `checkout` step and corrected `download-artifact` configuration (removed `path: artifacts` + `merge-multiple: true`).
- Made release file globs explicit for the current Linux-amd64-only build matrix.

## [0.5.0] - 2026-05-25

### Changed
- **Release process & CI modernization**:
  - Release workflow restricted to **Linux amd64 only**. macOS (Intel + Apple Silicon) builds are now fully user-driven via `make` for optimal OpenCL/Metal compatibility.
  - Fixed invalid GitHub Actions expression syntax (`${{ github.ref_name#v }}`) that was causing workflow validation failures. Added a proper `Compute version without 'v' prefix` step.
  - Improved release notes body with clearer Linux quick-start and explicit macOS build instructions.
- Overall release engineering now produces cleaner, more maintainable artifacts focused on the primary supported platform (Linux amd64) while keeping the project fully buildable everywhere via the Makefile.

### Maintenance
- Updated documentation (README, workflow comments) to reflect the new Linux-first release policy.
- Version bump and release parity preparation for both Gitea and GitHub.

## [0.4.2] - 2026-05-25

### Changed
- Release workflow now targets **Linux amd64 only**. macOS builds (both Intel and Apple Silicon) have been removed from automated CI. Users on macOS are expected to build locally with `make` for the best OpenCL/Metal experience on their hardware. The GitHub Actions release process continues to provide signed Linux amd64 artifacts (Go orchestrator + best-effort CPU renderer) with checksums and Cosign keyless signatures.

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