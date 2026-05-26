#!/bin/bash
# Runtime package builder for crescent-moon-visibility
#
# Assembles a *minimal, ready-to-run* distribution for a single platform from
# already-built binaries plus the runtime data the program needs. This is the
# single source of truth for the package layout; it is used both locally
# (`make package`) and by the Release CI workflow (.github/workflows/release.yml).
#
# Why this exists: a bare release binary does NOT run on its own. The Go
# orchestrator (crescent_maps) shells out to the CPU renderer and discovers it
# by a fixed set of names (see getRendererCandidates in main.go), and the
# blending step reads the NASA base map from the hardcoded relative path
# `data/map_nasa.png` (internal/blend/blend.go). So a usable download must ship:
#   - the orchestrator under its canonical name (crescent_maps[.exe])
#   - the CPU renderer under a name the orchestrator finds (bin/visibility[.exe|.out])
#   - data/map_nasa.png
#   - README.md, LICENSE, and a short QUICKSTART
#
# The GPU renderer is intentionally excluded (platform-specific OpenCL/driver
# deps); build it locally with `make gpu`. This mirrors the release policy.
#
# Resulting layout (extract and run from the top-level dir):
#   crescent-moon-visibility-<version>-<suffix>/
#   ├── crescent_maps[.exe]      # Go orchestrator (run this)
#   ├── bin/
#   │   └── visibility[.exe|.out]  # CPU reference renderer (auto-discovered)
#   ├── data/
#   │   └── map_nasa.png         # base map used by the blend step
#   ├── README.md
#   ├── LICENSE
#   └── QUICKSTART.txt
#
# Usage:
#   scripts/package.sh --version 0.5.3 --suffix linux-amd64 \
#       --orchestrator bin/crescent_maps --renderer bin/visibility.out
#
#   scripts/package.sh --version 0.5.3 --suffix windows-amd64 \
#       --orchestrator crescent_maps-0.5.3-windows-amd64.exe \
#       --renderer visibility-0.5.3-windows-amd64.exe
#
# Options:
#   --version V        Version string (required), e.g. 0.5.3
#   --suffix S         Platform suffix (required): linux-amd64 | windows-amd64 | darwin-arm64 ...
#                      A suffix containing "windows" produces a .zip with .exe names.
#   --orchestrator P   Path to the built crescent_maps binary (required)
#   --renderer P       Path to the built CPU visibility renderer (required)
#   --basemap P        Path to the NASA base map (default: data/map_nasa.png)
#   --readme P         Path to README   (default: README.md)
#   --license P        Path to LICENSE  (default: LICENSE)
#   --out DIR          Output directory for staging + archive (default: dist)
#
# Prints the path to the produced archive on stdout (last line).

set -euo pipefail

VERSION=""
SUFFIX=""
ORCHESTRATOR=""
RENDERER=""
BASEMAP="data/map_nasa.png"
README="README.md"
LICENSE="LICENSE"
OUT="dist"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)      VERSION="$2"; shift 2 ;;
    --suffix)       SUFFIX="$2"; shift 2 ;;
    --orchestrator) ORCHESTRATOR="$2"; shift 2 ;;
    --renderer)     RENDERER="$2"; shift 2 ;;
    --basemap)      BASEMAP="$2"; shift 2 ;;
    --readme)       README="$2"; shift 2 ;;
    --license)      LICENSE="$2"; shift 2 ;;
    --out)          OUT="$2"; shift 2 ;;
    -h|--help)      grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "package.sh: unknown option: $1" >&2; exit 2 ;;
  esac
done

die() { echo "package.sh: $*" >&2; exit 1; }

[[ -n "$VERSION" ]]      || die "missing --version"
[[ -n "$SUFFIX" ]]       || die "missing --suffix"
[[ -n "$ORCHESTRATOR" ]] || die "missing --orchestrator"
[[ -n "$RENDERER" ]]     || die "missing --renderer"
[[ -f "$ORCHESTRATOR" ]] || die "orchestrator not found: $ORCHESTRATOR"
[[ -f "$RENDERER" ]]     || die "renderer not found: $RENDERER"
[[ -f "$BASEMAP" ]]      || die "base map not found: $BASEMAP (the blend step requires it)"
[[ -f "$README" ]]       || die "readme not found: $README"
[[ -f "$LICENSE" ]]      || die "license not found: $LICENSE"

# Platform-specific naming. Windows ships .exe and a .zip; everything else
# ships an .out renderer and a .tar.gz.
case "$SUFFIX" in
  *windows*) IS_WINDOWS=1; ORCH_NAME="crescent_maps.exe"; REND_NAME="visibility.exe" ;;
  *)         IS_WINDOWS=0; ORCH_NAME="crescent_maps";     REND_NAME="visibility.out" ;;
esac

PKG="crescent-moon-visibility-${VERSION}-${SUFFIX}"
STAGE="${OUT}/${PKG}"

rm -rf "$STAGE"
mkdir -p "$STAGE/bin" "$STAGE/data"

# Orchestrator at the top level (this is what the user runs), renderer under
# bin/ where the orchestrator discovers it first, base map under data/.
cp "$ORCHESTRATOR" "$STAGE/${ORCH_NAME}"
cp "$RENDERER"     "$STAGE/bin/${REND_NAME}"
cp "$BASEMAP"      "$STAGE/data/map_nasa.png"
cp "$README"       "$STAGE/README.md"
cp "$LICENSE"      "$STAGE/LICENSE"
chmod +x "$STAGE/${ORCH_NAME}" "$STAGE/bin/${REND_NAME}" 2>/dev/null || true

# Short, platform-tailored quick-start so the download is self-explanatory.
if [[ "$IS_WINDOWS" -eq 1 ]]; then
  cat > "$STAGE/QUICKSTART.txt" <<EOF
Crescent Moon Visibility Maps ${VERSION} (${SUFFIX})

Self-contained CPU package. Extract anywhere and run from this folder so the
program can find bin\\visibility.exe and data\\map_nasa.png (both required).

  1. Open PowerShell in this extracted folder.
  2. Check it loads:        .\\crescent_maps.exe -version
  3. Generate a map:        .\\crescent_maps.exe -start 2027 -end 2027
                            (output_maps\\ will contain the .webp maps)

Notes
  - Keep the layout intact: crescent_maps.exe at the top, bin\\visibility.exe,
    and data\\map_nasa.png. Moving/renaming them will break map generation.
  - This package is CPU-only. The GPU renderer (-gpu) is not included; build it
    locally with 'make gpu' from source (needs OpenCL).
  - See README.md for full usage.
EOF
else
  cat > "$STAGE/QUICKSTART.txt" <<EOF
Crescent Moon Visibility Maps ${VERSION} (${SUFFIX})

Self-contained CPU package. Extract anywhere and run from this folder so the
program can find bin/visibility.out and data/map_nasa.png (both required).

  1. cd into this extracted folder.
  2. Check it loads:        ./crescent_maps -version
  3. Generate a map:        ./crescent_maps -start 2027 -end 2027
                            (output_maps/ will contain the .webp maps)

Notes
  - Keep the layout intact: crescent_maps at the top, bin/visibility.out, and
    data/map_nasa.png. Moving/renaming them will break map generation.
  - This package is CPU-only. The GPU renderer (-gpu) is not included; build it
    locally with 'make gpu' from source (needs OpenCL).
  - See README.md for full usage.
EOF
fi

# Archive. tar.gz for Unix-likes; zip for Windows (try common tools in order so
# this works on GitHub's Windows runners and typical dev machines alike).
ARCHIVE=""
if [[ "$IS_WINDOWS" -eq 1 ]]; then
  ARCHIVE="${OUT}/${PKG}.zip"
  rm -f "$ARCHIVE"
  if command -v zip >/dev/null 2>&1; then
    ( cd "$OUT" && zip -rq "${PKG}.zip" "$PKG" )
  elif command -v 7z >/dev/null 2>&1; then
    ( cd "$OUT" && 7z a -tzip -bso0 "${PKG}.zip" "$PKG" >/dev/null )
  elif command -v powershell >/dev/null 2>&1; then
    powershell -NoProfile -Command "Compress-Archive -Path '${STAGE}/*' -DestinationPath '${ARCHIVE}' -Force"
  elif command -v python3 >/dev/null 2>&1; then
    ( cd "$OUT" && python3 -m zipfile -c "${PKG}.zip" "$PKG" )
  else
    die "no zip tool found (need one of: zip, 7z, powershell, python3)"
  fi
else
  ARCHIVE="${OUT}/${PKG}.tar.gz"
  rm -f "$ARCHIVE"
  tar -czf "$ARCHIVE" -C "$OUT" "$PKG"
fi

echo "package.sh: built $ARCHIVE" >&2
echo "$ARCHIVE"
