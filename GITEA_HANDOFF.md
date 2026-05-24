# Crescent Moon Visibility – Gitea Handoff Notes

**Date:** May 2026  
**Current Branches (local):**
- `main` — Stable, release-ready code
- `dev` — Active development (you are here)

**Remote:** Gitea at `ssh://git@freenas.jim.intelpc.com:30009/IntelPC/crescent-moon-visibility.git`

---

## Project State Summary

This repository contains a mature, production-oriented implementation of the **Crescent Moon Visibility Maps Generator**.

### Major Achievements (Recent Work)

1. **Full Apple Silicon / Metal Support**
   - New `gpu/visibility_kernel_fp32.cl` using `float` + compensated double-double time arithmetic.
   - Achieves **96.97%** exact per-pixel match vs the pure-double CPU reference on M4 Pro.
   - Automatic kernel selection in `gpu/gpu_render.c`.

2. **Complete Removal of Python Runtime Dependency**
   - The final Python component (`gpu_blend.py` + OpenCV/PIL) was replaced by a pure-Go implementation in `internal/blend/`.
   - Legacy script moved to `scripts/legacy/`.

3. **Professional Release Engineering System**
   - Canonical `VERSION` file.
   - Semantic version bumping + pre-release support (`--rc`, `--beta`).
   - `make release-patch`, `make release-minor`, `make release-rc`, etc.
   - GitHub Actions workflow with:
     - Multi-platform native builds
     - Combined `checksums.txt`
     - Cosign keyless signing (OIDC)
   - `scripts/release.sh` helper.

4. **Strong Accuracy & Testing Culture**
   - `TestRendererAccuracy` regression test (enforces ≥96% CPU/GPU match).
   - Extensive unit tests in `internal/blend/blend_test.go`.
   - `make test` and `make test-accuracy` targets.

5. **Clean Project Organization**
   - `bin/` for build outputs
   - `data/` for assets
   - `docs/`, `scripts/`, proper separation of legacy code

---

## How to Submit to Gitea

### Recommended Steps

1. **On your local machine (on `dev` branch):**
   ```bash
   git push origin main
   git push origin dev
   ```

2. **Clean up the old remote branch (optional but recommended):**
   ```bash
   git push origin --delete golang
   ```

3. **On Gitea:**
   - Set the default branch to `main` (if not already).
   - Consider protecting `main` and requiring PRs into `dev` → `main`.

---

## Key Files & Their Purpose

| File | Purpose |
|------|---------|
| `VERSION` | Single source of truth for version |
| `scripts/release.sh` | Local release preparation + version bumping |
| `.github/workflows/release.yml` | Automated builds, checksums, Cosign signing |
| `Makefile` | Central build + `make release-*` targets |
| `internal/blend/` | Pure Go blending engine (no Python) |
| `gpu/visibility_kernel_fp32.cl` | Apple Silicon / FP32+DD kernel |
| `CHANGELOG.md` | Release history (Keep a Changelog format) |
| `GITEA_HANDOFF.md` | This document |

---

## Specialized Agents Available

Four background agents have been spawned in this Grok session to help continue the work:

- **Release Engineering Agent** (`019e5a57-8b01-7f41-91e1-18c39dcbf133`)
- **Testing & Accuracy Agent** (`019e5a57-977c-7ea3-8f93-ddc5c88cf1e7`)
- **Documentation & Architecture Agent** (`019e5a57-a4a3-73e2-a1c9-8c21d43424d4`)
- **GPU/OpenCL Performance Agent** (`019e5a58-ffa5-75a1-ad31-b01dfd4e2028`)

You can continue conversations with them using their task IDs.

---

## Recommended Next Work (Prioritized)

### High Priority
- Expand release workflow to `linux-arm64` and Windows (where feasible)
- Improve Cosign / GPG signing robustness in CI
- Add source tarball generation to releases
- Flesh out `CHANGELOG.md` with actual historical entries

### Medium Priority
- GPU/OpenCL performance tuning (vectorized DD, subgroups, better occupancy)
- More dates and edge cases in `TestRendererAccuracy`
- Better error handling and user messaging around missing renderers
- SBOM generation for releases

### Documentation
- Keep `docs/performance-accuracy.md` and the skill up to date
- Consider adding architecture diagrams

---

## Quick Commands

```bash
# Normal development
make
make test

# Prepare a release
make release-patch
make release-rc

# After pushing tags, the GitHub Action (or equivalent on Gitea) will run

# View current version
./bin/crescent_maps -version
```

---

## Notes for Gitea Migration

- The repository currently has two remotes in history (origin points to Gitea).
- The `.github/` folder can stay even if you primarily use Gitea (you can adapt the workflow or create a Gitea Actions equivalent later).
- Consider mirroring to GitHub later for visibility if desired.

---

**This handoff document was generated on 2026-05-24 as part of preparing the project for external submission.**

The foundation is solid. Future work can focus on polish, broader platform support, and performance while maintaining the strong accuracy guarantees already in place.