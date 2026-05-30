#!/usr/bin/env bash
#
# Build an installable, versioned plugin zip into wordpress-plugin/dist/.
#
# Output: dist/crescent-visibility-<version>.zip
# (The folder INSIDE the zip stays "crescent-visibility/" — that is the install
#  slug / text domain and must not change between releases.)
#
# Enforces versioning:
#   - the "Version:" header and the CVI_VERSION constant must match
#   - refuses to overwrite an already-built zip for this version (pass --force)
#
# Usage:  bash wordpress-plugin/scripts/build-plugin.sh [--force]

set -euo pipefail

cd "$(dirname "$0")/.."   # -> wordpress-plugin/

PLUGIN_DIR="plugin"
SLUG="crescent-visibility"
DIST="$PWD/dist"
FORCE="${1:-}"

main_php="$PLUGIN_DIR/crescent-visibility.php"

header_ver=$(grep -m1 -E '^\s*\*\s*Version:' "$main_php" | sed -E 's/.*Version:[[:space:]]*//' | tr -d '[:space:]')
const_ver=$(grep -m1 "define('CVI_VERSION'" "$main_php" | sed -E "s/.*'CVI_VERSION'[^']*'([^']+)'.*/\1/")

echo "Header version : $header_ver"
echo "CVI_VERSION    : $const_ver"

if [[ "$header_ver" != "$const_ver" ]]; then
  echo "ERROR: 'Version:' header ($header_ver) does not match CVI_VERSION ($const_ver). Bump both." >&2
  exit 1
fi

mkdir -p "$DIST"
ZIP="$DIST/${SLUG}-${header_ver}.zip"

# Guard: don't silently re-ship the same version.
if [[ -f "$ZIP" && "$FORCE" != "--force" ]]; then
  echo "ERROR: $ZIP already exists. Bump the version, or pass --force." >&2
  exit 1
fi

# Lint everything before packaging.
find "$PLUGIN_DIR" -name '*.php' -exec php -l {} \; >/dev/null
command -v node >/dev/null && node --check "$PLUGIN_DIR/assets/js/interactive.js"

# Stage only the files the plugin actually loads. No data file is bundled —
# data is imported via Tools → Crescent Visibility.
STAGE=$(mktemp -d)
DEST="$STAGE/$SLUG"
mkdir -p "$DEST"/{includes,public,admin,assets/css,assets/js,assets/vendor,data,languages}

cp "$PLUGIN_DIR/crescent-visibility.php"          "$DEST/"
cp "$PLUGIN_DIR/uninstall.php"                     "$DEST/"
cp "$PLUGIN_DIR/readme.txt"                        "$DEST/" 2>/dev/null || true
cp "$PLUGIN_DIR/includes/interactive.php"         "$DEST/includes/"
cp "$PLUGIN_DIR/public/renderer.php"              "$DEST/public/"
cp "$PLUGIN_DIR/public/interactive-renderer.php"  "$DEST/public/"
cp "$PLUGIN_DIR/admin/admin-page.php"             "$DEST/admin/"
cp "$PLUGIN_DIR/admin/admin.css"                  "$DEST/admin/"
cp "$PLUGIN_DIR/assets/css/interactive.css"       "$DEST/assets/css/"
cp "$PLUGIN_DIR/assets/js/interactive.js"         "$DEST/assets/js/"
cp -R "$PLUGIN_DIR/assets/vendor/leaflet"         "$DEST/assets/vendor/"
# Bundled sample dataset (small; the full dataset is generated separately).
cp "$PLUGIN_DIR/data/sample.json"                 "$DEST/data/" 2>/dev/null || true
# Translation template, if present.
cp "$PLUGIN_DIR/languages/"*.pot                   "$DEST/languages/" 2>/dev/null || true

rm -f "$ZIP"
( cd "$STAGE" && zip -rq "$ZIP" "$SLUG" -x '*.DS_Store' )
rm -rf "$STAGE"

echo "Built dist/${SLUG}-${header_ver}.zip ($(du -h "$ZIP" | cut -f1), no bundled data)"
