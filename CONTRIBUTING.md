# Contributing to Crescent Moon Visibility Maps Generator

Thank you for your interest in contributing!

This project is licensed under the **MIT License**. By contributing, you agree that your contributions will be licensed under the same terms.

## Code of Conduct

Be respectful, inclusive, and constructive. We welcome contributions from people with a wide range of backgrounds and experience levels.

## How to Contribute

1. **Check existing issues** — Look for open issues or discussions before starting new work.
2. **Fork & branch** — Create a feature branch from `main` (or the current development branch).
3. **Make focused changes** — One logical change per pull request when possible.
4. **Update documentation** — If you change behavior, update README.md, TODO.md, or the relevant docs.
5. **Test your changes** — At minimum, ensure `make` succeeds and the generated maps look correct for a test date.
6. **Open a Pull Request** — Describe what you changed and why. Link any related issues.

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/crescent-moon-visibility.git
cd crescent-moon-visibility

# Build everything
make

# Common targets
make cpu          # CPU renderer only
make gpu          # GPU renderer (if OpenCL available)
make go           # Go orchestrator
make clean        # Remove build artifacts from bin/
```

### Prerequisites

- Go 1.22+
- C/C++ compiler with OpenMP (or `libomp` on macOS via Homebrew)
- Python 3 + OpenCV/Pillow/numpy (only needed if you want to run the legacy `gpu_blend.py` — no longer required for normal use)
- OpenCL headers/libraries (optional but recommended for GPU path)

The core pipeline is now pure Go + CGO (the final Python component was removed in 2026).

## Project Structure (After Recent Organization)

- `cmd/` — Standalone binaries (CPU renderer)
- `internal/` — Private packages (CGO bindings, future blend logic)
- `gpu/` — OpenCL host + kernels (FP64 and FP32+DD paths)
- `data/` — Static assets (`map_nasa.png`)
- `bin/` — Compiled outputs (populated by `make`)
- `docs/` — Detailed technical documentation

## Releases

Releases are automated via GitHub Actions on `v*` tags. See the [Release Process](#release-process) section in the README.

## Adding New Features or Fixing Bugs

- Visibility criteria, new map projections, better accuracy, performance improvements, and documentation are all welcome.
- When adding new visibility math, try to keep the CPU reference (`cmd/visibility/visibility.cc`) and the GPU kernels in sync.
- For changes that affect output images, consider adding a note in `docs/performance-accuracy.md`.

## Questions?

Open an issue with the `question` label or start a discussion.

We look forward to your contributions!