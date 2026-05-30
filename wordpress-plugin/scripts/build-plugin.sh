#!/usr/bin/env bash
#
# Build the installable crescent-visibility.zip from the current plugin files.
#
# Enforces that the plugin is versioned on every package update:
#   - the "Version:" header and the CVI_VERSION constant must match
#   - refuses to overwrite a zip that already embeds this exact version
#     (pass --force to override) so you remember to bump it
#
# Usage:  bash wordpress-plugin/scripts/build-plugin.sh [--force]

set -euo pipefail

cd "$(dirname "$0")/.."   # -> wordpress-plugin/

PLUGIN_DIR="plugin"
ZIP="$PWD/crescent-visibility.zip"
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

# Guard: don't silently re-ship the same version.
if [[ -f "$ZIP" && "$FORCE" != "--force" ]]; then
  if unzip -p "$ZIP" crescent-visibility/crescent-visibility.php 2>/dev/null | grep -q "Version:.*$header_ver"; then
    echo "ERROR: $ZIP already contains version $header_ver. Bump the version, or pass --force." >&2
    exit 1
  fi
fi

# Lint everything before packaging.
find "$PLUGIN_DIR" -name '*.php' -not -name 'renderer-improved.php' -not -name 'renderer-with-atmosphere.php' \
  -exec php -l {} \; >/dev/null
command -v node >/dev/null && node --check "$PLUGIN_DIR/assets/js/interactive.js"

# Stage only the files the plugin actually loads (exclude exploratory renderers).
# Note: no data file is bundled — data is imported separately via
# Tools → Crescent Visibility (the JSON is parsed into the DB on upload).
STAGE=$(mktemp -d)
DEST="$STAGE/crescent-visibility"
mkdir -p "$DEST"/{includes,public,admin,assets/css,assets/js}

cp "$PLUGIN_DIR/crescent-visibility.php"          "$DEST/"
cp "$PLUGIN_DIR/uninstall.php"                     "$DEST/"
cp "$PLUGIN_DIR/includes/interactive.php"         "$DEST/includes/"
cp "$PLUGIN_DIR/public/renderer.php"              "$DEST/public/"
cp "$PLUGIN_DIR/public/interactive-renderer.php"  "$DEST/public/"
cp "$PLUGIN_DIR/admin/admin-page.php"             "$DEST/admin/"
cp "$PLUGIN_DIR/admin/admin.css"                  "$DEST/admin/"
cp "$PLUGIN_DIR/assets/css/interactive.css"       "$DEST/assets/css/"
cp "$PLUGIN_DIR/assets/js/interactive.js"         "$DEST/assets/js/"

rm -f "$ZIP"
( cd "$STAGE" && zip -rq "$ZIP" crescent-visibility -x '*.DS_Store' )
rm -rf "$STAGE"

echo "Built $ZIP (v$header_ver, no bundled data, $(du -h "$ZIP" | cut -f1))"
