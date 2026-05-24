# GitHub Migration Notes

**Target**: https://github.com/jim-fun/crescent-moon-visibility

This repository is developed primarily on Gitea (or local) with `main` as the stable branch. The GitHub repository is the public mirror.

## Recommended Process (Always)

1. All non-trivial changes that will land on `main` should have gone through the full 4-stage **agentic workflow** (see `AGENTIC_WORKFLOW.md`):
   - `./scripts/agentic-review.sh --improve "..."` or `--review-todo "..."` as appropriate.
   - Judge issues **Go** (or Go with Conditions) and a filled `JUDGE_DECISION.md`.

2. Run the safe helper:
   ```bash
   ./scripts/push-to-github.sh --tags
   ```
   The script performs:
   - Build + `make test`
   - Remote setup (`github` remote pointing at the target)
   - Conservative push of `main`
   - Optional tag push
   - Post-push verification checklist (GitHub Actions, etc.)

3. After pushing:
   - Verify the GitHub Actions run for the release workflow.
   - If a new version was tagged, draft or update the GitHub Release from CHANGELOG.md + the Judge decision.
   - Update `GITEA_HANDOFF.md` (or create a dated migration note) if the external state changed.

## Specialized Agent

For complex or high-stakes migrations, spawn the dedicated agent:

Use the prompt in `scripts/agents/github-migration-agent.md` (it references this file, the skill, the helper script, and enforces Accuracy First + Verifiability).

## Safety Rules

- Never force-push `main` to GitHub without explicit human sign-off and written rationale.
- Always run at least `make` + `make test` before pushing `main`.
- For release tags, also consider `make test-accuracy` when renderers are available.
- The public GitHub `main` must remain as clean and reproducible as the internal one.

**Last updated**: 2026-05 (together with the reusable GitHub migration agent and helper script).