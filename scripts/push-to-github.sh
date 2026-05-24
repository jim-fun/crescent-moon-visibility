#!/bin/bash
#
# Crescent Moon Visibility — Safe Push / Migrate to GitHub Helper
#
# Target: https://github.com/jim-fun/crescent-moon-visibility
#
# This script is intentionally conservative. It never force-pushes main
# without explicit human confirmation and always recommends running the
# full agentic workflow (Improvement → Validation → Security → Judge)
# for any non-trivial change before pushing.
#
# Usage:
#   ./scripts/push-to-github.sh            # interactive, safe defaults
#   ./scripts/push-to-github.sh --check    # only run preflight checks
#   ./scripts/push-to-github.sh --tags     # also push tags after main
#
# The companion agent prompt (scripts/agents/github-migration-agent.md)
# tells a specialized subagent exactly how to use and extend this helper.
#
# IMPORTANT: AI / Agentic tooling exclusion for public GitHub
# ----------------------------------------------------------------
# The following files and directories are INTERNAL ONLY (for the
# project maintainer + Grok agents). They are deliberately excluded
# when pushing to the public GitHub mirror at
# https://github.com/jim-fun/crescent-moon-visibility
#
# Excluded from public GitHub:
#   - AGENTIC_WORKFLOW.md
#   - GITHUB_MIGRATION.md
#   - scripts/agentic-review.sh
#   - scripts/agents/   (all prompt templates and the Judge template)
#
# These exist only in the internal Gitea repository and your local
# clone to support the repeatable agentic improvement process.
# The public GitHub repo contains only the core project.

set -euo pipefail

GITHUB_REMOTE_URL="https://github.com/jim-fun/crescent-moon-visibility.git"
GITHUB_REMOTE_NAME="github"
TARGET_BRANCH="main"

# === Internal-only AI / Agent tooling (never ship to public GitHub) ===
AI_ONLY_PATHS=(
    "AGENTIC_WORKFLOW.md"
    "GITHUB_MIGRATION.md"
    "scripts/agentic-review.sh"
    "scripts/agents"
)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "=================================================="
echo "  Crescent Moon Visibility — GitHub Push Helper"
echo "  Target: $GITHUB_REMOTE_URL"
echo "=================================================="
echo

# --- Preflight checks (always run) ------------------------------------------

echo "[1/6] Preflight checks"

CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
echo "  Current branch: $CURRENT_BRANCH"

if [[ "$CURRENT_BRANCH" != "$TARGET_BRANCH" && "$CURRENT_BRANCH" != "dev" ]]; then
  echo "  WARNING: You are not on 'main' or 'dev'."
  read -rp "  Continue anyway? (y/N) " ans
  if [[ ! "$ans" =~ ^[Yy] ]]; then
    echo "Aborted."
    exit 1
  fi
fi

echo "  Running 'make' (build) ..."
make -s || { echo "Build failed — fix before pushing."; exit 1; }

echo "  Running 'make test' ..."
make test -s || { echo "Tests failed — fix before pushing."; exit 1; }

if command -v ./bin/crescent_maps >/dev/null 2>&1; then
  echo "  Version check:"
  ./bin/crescent_maps -version || true
fi

echo "  Git status (should be clean for a normal release push):"
git status --short

echo
read -rp "Preflight passed. Continue with remote setup and push? (y/N) " ans
if [[ ! "$ans" =~ ^[Yy] ]]; then
  echo "Aborted before remote operations."
  exit 0
fi

# --- Remote management -------------------------------------------------------

echo
echo "[2/6] Ensuring GitHub remote exists"

if git remote get-url "$GITHUB_REMOTE_NAME" >/dev/null 2>&1; then
  echo "  Remote '$GITHUB_REMOTE_NAME' already configured:"
  git remote get-url "$GITHUB_REMOTE_NAME"
else
  echo "  Adding remote '$GITHUB_REMOTE_NAME' → $GITHUB_REMOTE_URL"
  git remote add "$GITHUB_REMOTE_NAME" "$GITHUB_REMOTE_URL"
fi

echo
echo "  Fetching from GitHub (to check for divergence)..."
git fetch "$GITHUB_REMOTE_NAME" --tags || true

# --- AI tooling exclusion for public GitHub ---------------------------------
echo
echo "[2.5/6] Preparing public GitHub tree (excluding internal AI tooling)"

# Create a temporary sanitized branch for the public push
PUBLIC_PREP_BRANCH="github-public-prep-$(date +%s)"
git branch -D "$PUBLIC_PREP_BRANCH" 2>/dev/null || true
git checkout -b "$PUBLIC_PREP_BRANCH"

echo "  Removing internal-only AI/agent files from the tree for GitHub..."
for path in "${AI_ONLY_PATHS[@]}"; do
    if [ -e "$path" ]; then
        echo "    - Excluding: $path"
        git rm -r --cached --ignore-unmatch "$path" >/dev/null 2>&1 || true
    fi
done

# Commit the removal if anything changed
if ! git diff --cached --quiet; then
    git commit -m "chore: remove internal AI/agent tooling for public GitHub mirror

These files (AGENTIC_WORKFLOW.md, GITHUB_MIGRATION.md,
scripts/agentic-review.sh, scripts/agents/) are for the project
maintainer and the agentic improvement workflow only.
They are deliberately omitted from the public repository."
    echo "  ✓ Created sanitized commit for public push."
else
    echo "  (No AI tooling changes to remove in this state.)"
fi

# Return to original branch
git checkout "$CURRENT_BRANCH" 2>/dev/null || git checkout - || true

echo "  Will push the sanitized branch to GitHub as 'main'."

# --- Push main ---------------------------------------------------------------

echo
echo "[3/6] Pushing $TARGET_BRANCH to GitHub"

if [[ "$CURRENT_BRANCH" != "$TARGET_BRANCH" ]]; then
  echo "  You are on '$CURRENT_BRANCH'. The script will push the local '$TARGET_BRANCH' ref."
  read -rp "  Switch to $TARGET_BRANCH first? (recommended) (Y/n) " ans
  if [[ ! "$ans" =~ ^[Nn] ]]; then
    git checkout "$TARGET_BRANCH"
  fi
fi

echo "  About to execute (public-safe push):"
echo "    git push $GITHUB_REMOTE_NAME $PUBLIC_PREP_BRANCH:$TARGET_BRANCH"
echo
read -rp "  Proceed with sanitized public push to GitHub 'main'? (y/N) " ans
if [[ "$ans" =~ ^[Yy] ]]; then
  git push "$GITHUB_REMOTE_NAME" "$PUBLIC_PREP_BRANCH:$TARGET_BRANCH"
  echo "  ✓ Public 'main' updated on GitHub (AI tooling excluded)."
else
  echo "  Skipped public push."
fi

# Clean up the temporary prep branch
git branch -D "$PUBLIC_PREP_BRANCH" 2>/dev/null || true

# --- Optional tag push -------------------------------------------------------

if [[ "${1:-}" == "--tags" || "${2:-}" == "--tags" ]]; then
  echo
  echo "[4/6] Pushing tags to GitHub"
  echo "  About to execute:"
  echo "    git push $GITHUB_REMOTE_NAME --tags"
  echo "  (This is safe for annotated tags; use --force-with-lease only if you know what you are doing.)"
  read -rp "  Push all tags? (y/N) " ans
  if [[ "$ans" =~ ^[Yy] ]]; then
    git push "$GITHUB_REMOTE_NAME" --tags
    echo "  ✓ Tags pushed."
  else
    echo "  Skipped tag push."
  fi
else
  echo
  echo "[4/6] Tag push skipped (run with --tags to include them)."
fi

# --- Post-push verification --------------------------------------------------

echo
echo "[5/6] Post-push verification (manual steps recommended)"

echo "  1. Visit https://github.com/jim-fun/crescent-moon-visibility/actions"
echo "     and confirm the release / build workflow ran successfully."
echo "  2. If you pushed a new version tag, verify the GitHub Release page."
echo "  3. If you maintain a Gitea mirror, decide whether to push there too."
echo "  4. Update GITEA_HANDOFF.md or any migration notes if the state changed."

# --- Final reminder ----------------------------------------------------------

echo
echo "[6/6] Agentic & Documentation Reminder"
echo
echo "Before any future non-trivial push, strongly consider:"
echo "  ./scripts/agentic-review.sh --improve \"<description of what changed>\""
echo "  (This runs the full Improvement → Validation (accuracy) → Security → Judge process.)"
echo
echo "Also consider running the accuracy regression on a representative date:"
echo "  make test-accuracy   # (requires built renderers + RUN_ACCURACY_TEST=1)"
echo
echo "All done. GitHub remote state should now be up to date with your local $TARGET_BRANCH."
echo "=================================================="