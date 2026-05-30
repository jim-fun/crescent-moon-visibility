#!/bin/bash
#
# feed-v3-v4-to-claude.sh
#
# Convenience script to send the production prompt + v3 and v4 exploratory drafts
# to Claude Code CLI for the merged production implementation of the
# "Visibility for My Location" interactive experience.
#
# IMPORTANT:
#   This script must be run from a terminal where you have an active
#   authenticated Claude Code session (`claude login`).
#
#   The Grok tool environment cannot use your local login.
#
# Recommended usage:
#   1. In your normal terminal: cd to the repo root
#   2. claude login   (if not already logged in)
#   3. bash wordpress-plugin/scripts/feed-v3-v4-to-claude.sh
#
# This will launch Claude Code in non-interactive print mode with the full context.

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

PROMPT_FILE="wordpress-plugin/prompts/claude-production-app-parity-interactive.md"
V3_FILE="wordpress-plugin/exploratory-drafts/qwen3-app-parity-interactive-v3.php"
V4_FILE="wordpress-plugin/exploratory-drafts/qwen3-ajax-defaults-dynamic-loading-v4.php"

if [ ! -f "$PROMPT_FILE" ]; then
  echo "ERROR: Production prompt not found at $PROMPT_FILE"
  exit 1
fi

echo "=== Feeding App Parity work to Claude ==="
echo "Prompt : $PROMPT_FILE"
echo "v3     : $V3_FILE"
echo "v4     : $V4_FILE"
echo

# Check if claude is available
if ! command -v claude &> /dev/null; then
  echo "Claude CLI not found in PATH."
  echo "Make sure ~/.local/bin/claude (or equivalent) is in your PATH."
  exit 1
fi

# The most reliable way: use --print with a wrapper prompt that tells Claude
# to treat the referenced files as the primary source material.

CLAUDE_PROMPT=$(cat << 'EOF'
You are to follow the instructions in the file below with extreme precision.

Read and internalize the entire content of:
- wordpress-plugin/prompts/claude-production-app-parity-interactive.md

This prompt contains the full role, constraints, goals, and required deliverables for producing the production-grade WordPress plugin implementation of the interactive "Visibility for My Location" experience.

You are also given access (via the workspace context) to the two key exploratory source files that you must synthesize:

1. wordpress-plugin/exploratory-drafts/qwen3-app-parity-interactive-v3.php
2. wordpress-plugin/exploratory-drafts/qwen3-ajax-defaults-dynamic-loading-v4.php

Your task is to produce a clean, merged, production-ready set of files for the plugin that makes the interactive experience as close as possible to the original Go web app /point tool, while respecting all the minimal-footprint and precomputed-data constraints.

Output the full code for the recommended files (main plugin updates, renderer, JS, AJAX handlers, etc.).

Begin now.
EOF
)

echo "Launching Claude Code in non-interactive mode..."
echo "This may take some time as Claude reads the files and generates the implementation."
echo

# Note: This command must be run in an environment where you are logged into Claude Code.
# The command below is the correct invocation.
echo "If this fails with 'Not logged in', run 'claude login' first in your terminal."
echo ""
echo "When Claude finishes, paste its output into the folder:"
echo "  wordpress-plugin/production-output/"
echo "See production-output/INSTRUCTIONS.md for the recommended process."

# Execute
claude -p "$CLAUDE_PROMPT" \
  --add-dir wordpress-plugin/exploratory-drafts \
  --add-dir wordpress-plugin/prompts \
  --add-dir wordpress-plugin/plugin \
  --bare \
  --print

echo
echo "=== Claude run complete ==="
echo "Review the output above. You can also continue the conversation in Claude Code if needed (run 'claude' interactively in this directory)."