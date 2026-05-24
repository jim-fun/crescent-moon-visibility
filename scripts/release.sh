#!/bin/bash
# Release helper for crescent-moon-visibility
#
# Usage examples:
#   ./scripts/release.sh patch           # 0.2.0 → 0.2.1
#   ./scripts/release.sh minor           # 0.2.0 → 0.3.0
#   ./scripts/release.sh major           # 0.2.0 → 1.0.0
#   ./scripts/release.sh patch --rc      # 0.2.0 → 0.2.1-rc.1   (or increments existing rc)
#   ./scripts/release.sh patch --beta    # creates beta pre-release
#   ./scripts/release.sh 0.2.1-rc.3      # explicit version
#
#   make release-patch
#   make release-minor
#   make release-rc

set -euo pipefail

# --- Helper functions ---

bump_version() {
    local version=$1
    local type=$2

    # Remove 'v' prefix if present
    version=${version#v}

    IFS='.' read -r major minor patch <<< "$version"

    case "$type" in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            echo "Unknown bump type: $type" >&2
            exit 1
            ;;
    esac

    echo "${major}.${minor}.${patch}"
}

parse_prerelease() {
    local current=$1
    local kind=$2   # rc, beta, alpha

    # Remove 'v' if present
    current=${current#v}

    # Check if current version already has this prerelease type
    if [[ "$current" =~ ^([0-9]+\.[0-9]+\.[0-9]+)-(${kind})\.([0-9]+)$ ]]; then
        local base="${BASH_REMATCH[1]}"
        local num="${BASH_REMATCH[3]}"
        echo "${base}-${kind}.$((num + 1))"
    else
        # Start new prerelease series from the base version
        local base
        base=$(echo "$current" | cut -d- -f1)
        echo "${base}-${kind}.1"
    fi
}

# --- Main logic ---

CMD=${1:-}
PRE_RELEASE=""

# Parse flags like --rc, --beta, --alpha
shift || true
while [[ $# -gt 0 ]]; do
    case "$1" in
        --rc)     PRE_RELEASE="rc";;
        --beta)   PRE_RELEASE="beta";;
        --alpha)  PRE_RELEASE="alpha";;
        *) echo "Unknown option: $1"; exit 1;;
    esac
    shift
done

CURRENT_VERSION=$(cat VERSION 2>/dev/null || echo "0.0.0")

if [[ "$CMD" =~ ^(patch|minor|major)$ ]]; then
    NEW_VERSION=$(bump_version "$CURRENT_VERSION" "$CMD")

    if [[ -n "$PRE_RELEASE" ]]; then
        NEW_VERSION=$(parse_prerelease "$NEW_VERSION" "$PRE_RELEASE")
    fi

    echo "Bumping $CMD: $CURRENT_VERSION → $NEW_VERSION"
elif [[ "$CMD" =~ ^[0-9]+\.[0-9]+\.[0-9]+ ]]; then
    # Explicit version provided
    NEW_VERSION=${CMD#v}
    echo "Using explicit version: $NEW_VERSION"
else
    echo "Usage:"
    echo "  $0 patch [--rc|--beta|--alpha]"
    echo "  $0 minor [--rc|--beta|--alpha]"
    echo "  $0 major [--rc|--beta|--alpha]"
    echo "  $0 0.2.1-rc.2"
    exit 1
fi

# Update VERSION file (without leading v)
echo "$NEW_VERSION" > VERSION

TAG="v${NEW_VERSION}"

git add VERSION
git commit -m "chore(release): prepare ${TAG}" || true

git tag -a "$TAG" -m "Release ${TAG}"

echo ""
echo "✓ Prepared release ${TAG}"
echo "  VERSION file updated to ${NEW_VERSION}"
echo ""
echo "Next steps:"
echo "  git push origin main --tags"
echo ""
echo "This will trigger the GitHub Actions release workflow."