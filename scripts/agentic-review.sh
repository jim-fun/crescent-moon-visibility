#!/bin/bash
#
# Crescent Moon Visibility - Agentic Review Helper
#
# Provides two primary ways to run the 4-stage workflow (Improvement →
# Validation → Security Review → Judge) defined in AGENTIC_WORKFLOW.md:
#
#   1. Direct modes (recommended for clarity and speed):
#        --improve "description"     → kick off a specific targeted improvement
#        --review-todo "area"        → review code and produce items ready for TODO.md
#
#   2. Interactive wizard (default, no args) for guided step-by-step.
#
# The script never auto-spawns agents (you do that in your Grok session using
# spawn_subagent or by pasting prompts). It gives you exact copy-paste blocks,
# manages the .agentic-review/ working directory, pre-populates the Judge
# Decision Template, and ensures the special TODO-output format is used when
# reviewing code for the backlog.
#
# See AGENTIC_WORKFLOW.md "How to Kick Off Agents" for full usage examples.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
AGENTS_DIR="$SCRIPT_DIR/agents"
TEMPLATE="$AGENTS_DIR/JUDGE_DECISION_TEMPLATE.md"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/agentic-review.sh
      Interactive guided 4-stage workflow (prompts printed, you save outputs).

  ./scripts/agentic-review.sh --improve "Clear description of the improvement"
      Prints ready-to-paste spawn_subagent instructions for a specific
      improvement. Feeds prior stage outputs forward. Ends with Judge using
      the official decision template.

  ./scripts/agentic-review.sh --review-todo "Area of code / docs to review"
      Special mode for populating TODO.md. Improvement agent is instructed
      to emit candidates in the exact TODO.md format. Judge curates a clean
      block ready to paste directly into TODO.md.

  ./scripts/agentic-review.sh --help

Examples:
  ./scripts/agentic-review.sh --improve \
    "Add linux-arm64 build to the release workflow while preserving Cosign signing"

  ./scripts/agentic-review.sh --review-todo \
    "GPU kernel accuracy risks on non-Apple OpenCL devices (FP32+DD path)"
EOF
}

# --- Helpers -----------------------------------------------------------------

print_stage_header() {
  local n="$1" title="$2"
  echo
  echo "=================================================="
  echo "  STAGE $n: $title"
  echo "=================================================="
}

create_workdir() {
  local title="$1"
  local slug
  slug=$(echo "$title" | tr ' ' '-' | tr -cd '[:alnum:]-' | cut -c1-50)
  local ts
  ts=$(date +%Y%m%d-%H%M%S)
  local dir="$PROJECT_ROOT/.agentic-review/${ts}-${slug}"
  mkdir -p "$dir"
  echo "$dir"
}

prefill_judge_template() {
  local workdir="$1"
  local title="$2"
  local reviewer="${3:-$(whoami)}"
  cp "$TEMPLATE" "$workdir/JUDGE_DECISION.md"
  # Portable sed (macOS + Linux)
  if sed --version >/dev/null 2>&1; then
    sed -i -e "s|Change / Proposal Title:.*|Change / Proposal Title: $title|" \
           -e "s|Submitted by:.*|Submitted by: $reviewer|" \
           "$workdir/JUDGE_DECISION.md"
  else
    sed -i '' -e "s|Change / Proposal Title:.*|Change / Proposal Title: $title|" \
              -e "s|Submitted by:.*|Submitted by: $reviewer|" \
              "$workdir/JUDGE_DECISION.md"
  fi
  echo "$workdir/JUDGE_DECISION.md"
}

inject_task_into_prompt() {
  # Print the base prompt file, then append the user's task + workflow context
  local prompt_file="$1"
  local task="$2"
  local extra_instructions="${3:-}"
  cat "$prompt_file"
  echo
  echo "=================================================="
  echo "TASK / FOCUS FOR THIS RUN"
  echo "=================================================="
  echo "$task"
  if [[ -n "$extra_instructions" ]]; then
    echo
    echo "$extra_instructions"
  fi
}

# --- Mode: --improve (specific targeted improvement) ------------------------

do_improve() {
  local task="$1"
  local title="Improve: $(echo "$task" | cut -c1-60)"

  echo "Crescent Moon Visibility — Agentic Improvement (specific task)"
  echo "Task: $task"
  echo
  echo ">>> CORE PRINCIPLES REMINDER (MANDATORY FOR ALL AGENTS) <<<"
  echo "You must explicitly score the proposal on all four principles."
  echo "Accuracy First is non-negotiable. Weak/Violated ratings here usually result in No-Go."
  echo

  local workdir
  workdir=$(create_workdir "$title")
  echo "Working directory: $workdir"
  local judge_file
  judge_file=$(prefill_judge_template "$workdir" "$title")

  echo
  echo "Artifacts will be saved under: $workdir"
  echo "  01-improvement.md"
  echo "  02-validation.md"
  echo "  03-security.md"
  echo "  04-judge.md  (and JUDGE_DECISION.md template)"
  echo

  # Stage 1 - Improvement
  print_stage_header 1 "IMPROVEMENT"
  echo "Copy everything below the line and use it as the prompt for a"
  echo "spawn_subagent (or paste directly into Grok). Then save the full"
  echo "response as $workdir/01-improvement.md"
  echo
  echo "--------------------------------------------------------------------------------"
  inject_task_into_prompt "$AGENTS_DIR/improvement-agent.md" "$task" \
    "Focus tightly on the requested improvement. Evaluate explicitly against the four Core Principles (Accuracy First is non-negotiable)."
  echo "--------------------------------------------------------------------------------"
  echo
  echo "After saving 01-improvement.md, continue to stage 2 (Validation)."

  # Stage 2 - Validation (user will feed previous output)
  print_stage_header 2 "VALIDATION"
  echo "In the same Grok session (or a new one), feed the Improvement output + task."
  echo "Use this prompt block:"
  echo
  echo "--------------------------------------------------------------------------------"
  echo "CONTEXT: The previous agent produced the output in 01-improvement.md (paste it here)."
  echo
  inject_task_into_prompt "$AGENTS_DIR/validation-agent.md" "$task" \
    "Focus validation on Accuracy First and Performance with Integrity. Re-run or reference make test and make test-accuracy where relevant."
  echo "--------------------------------------------------------------------------------"
  echo
  echo "Save response as $workdir/02-validation.md"

  # Stage 3 - Security Review
  print_stage_header 3 "SECURITY REVIEW"
  echo "Feed the prior two outputs:"
  echo
  echo "--------------------------------------------------------------------------------"
  echo "CONTEXT: Improvement (01) + Validation (02) outputs pasted above."
  echo
  inject_task_into_prompt "$AGENTS_DIR/security-review-agent.md" "$task" \
    "Pay special attention to any release, signing, or mixed-language (Go+C+OpenCL) risks."
  echo "--------------------------------------------------------------------------------"
  echo
  echo "Save as $workdir/03-security.md"

  # Stage 4 - Judge
  print_stage_header 4 "JUDGE (final authority)"
  echo "Feed ALL three prior outputs + the original task to the Judge."
  echo "Use the official structured template (already copied to $judge_file)."
  echo
  echo "--------------------------------------------------------------------------------"
  echo "CONTEXT: Paste the full contents of 01-improvement.md, 02-validation.md, and"
  echo "03-security.md above this line."
  echo
  echo "You are the Judge. Use the exact output format required in AGENTIC_WORKFLOW.md"
  echo "and fill the pre-populated $judge_file (Core Principles Scorecard is mandatory)."
  echo
  cat "$AGENTS_DIR/judge-agent.md"
  echo "--------------------------------------------------------------------------------"
  echo
  echo "Save the Judge's full response as $workdir/04-judge.md"
  echo "Also complete the checklist in JUDGE_DECISION.md"
  echo
  echo "============================================================"
  echo "When the Judge returns Go (or Go with Conditions), the change is cleared"
  echo "for implementation / merge. Update TODO.md / CHANGELOG as needed."
  echo "All artifacts: $workdir"
}

# --- Mode: --review-todo (review code → produce TODO-ready items) -----------

do_review_todo() {
  local area="$1"
  local title="TODO Review: $(echo "$area" | cut -c1-55)"

  echo "Crescent Moon Visibility — Agentic Code Review for TODO Population"
  echo "Area under review: $area"
  echo
  echo ">>> CORE PRINCIPLES REMINDER (MANDATORY) <<<"
  echo "All proposed TODO items must be justified against the four principles,"
  echo "with Accuracy First as the dominant filter."
  echo
  echo "This mode configures the Improvement agent to emit findings in the"
  echo "exact format used by TODO.md, and instructs the Judge to curate a"
  echo "clean, prioritized block ready for direct insertion into TODO.md."
  echo

  local workdir
  workdir=$(create_workdir "$title")
  echo "Working directory: $workdir"
  local judge_file
  judge_file=$(prefill_judge_template "$workdir" "$title")

  echo "Artifacts: $workdir/{01-improvement,02-validation,03-security,04-judge}.md + JUDGE_DECISION.md"
  echo

  # Improvement – special TODO output instructions
  print_stage_header 1 "IMPROVEMENT (TODO generation mode)"
  echo "Copy the block below. The Improvement prompt contains extra instructions"
  echo "to output candidate items using the exact TODO.md bullet style."
  echo "Save full response as $workdir/01-improvement.md"
  echo
  echo "--------------------------------------------------------------------------------"
  echo "SPECIAL TODO POPULATION MODE (injected):"
  cat "$AGENTS_DIR/todo-review-instructions.txt"
  echo
  echo "The above instructions are prepended to the standard Improvement prompt."
  inject_task_into_prompt "$AGENTS_DIR/improvement-agent.md" "$area" \
    "$(cat "$AGENTS_DIR/todo-review-instructions.txt")"
  echo "--------------------------------------------------------------------------------"

  # Validation (still valuable even for pure review)
  print_stage_header 2 "VALIDATION"
  echo "Feed Improvement output. Ask it to comment on whether existing tests would"
  echo "have caught the issues surfaced, and what additional validation would be ideal."
  echo
  echo "--------------------------------------------------------------------------------"
  echo "CONTEXT: Paste 01-improvement.md here."
  echo
  inject_task_into_prompt "$AGENTS_DIR/validation-agent.md" "$area" \
    "Evaluate whether the current test suite (make test, make test-accuracy, blend tests) would catch the problems or opportunities identified. Suggest concrete new tests or accuracy regression dates if gaps exist."
  echo "--------------------------------------------------------------------------------"
  echo "Save as $workdir/02-validation.md"

  # Security (often relevant for workflow / release / kernel areas)
  print_stage_header 3 "SECURITY REVIEW"
  echo
  echo "--------------------------------------------------------------------------------"
  echo "CONTEXT: Paste 01 + 02 outputs."
  echo
  inject_task_into_prompt "$AGENTS_DIR/security-review-agent.md" "$area" \
    "Flag any items that would introduce supply-chain, signing, or reproducibility risks if implemented later."
  echo "--------------------------------------------------------------------------------"
  echo "Save as $workdir/03-security.md"

  # Judge – special TODO curation instructions
  print_stage_header 4 "JUDGE (curate final TODO list)"
  echo "Feed all three prior outputs. Instruct the Judge to produce a clean,"
  echo "prioritized block ready to paste straight into TODO.md."
  echo
  echo "--------------------------------------------------------------------------------"
  echo "CONTEXT: Paste the full 01, 02, and 03 outputs above."
  echo
  echo "You are the Judge. In addition to the normal verdict + scorecard, your most"
  echo "important output for this run is a section titled:"
  echo
  echo "  ## Curated TODO Items (ready for TODO.md)"
  echo
  echo "Under that heading, emit only the highest-value items (after applying your"
  echo "principle-by-principle filter, with Accuracy First dominant). Use the exact"
  echo "TODO.md bullet format. Include a short note for each: 'From agentic review"
  echo "of <area> on <date>'."
  echo "Only items the Judge explicitly endorses should appear in this block."
  echo
  cat "$AGENTS_DIR/judge-agent.md"
  echo "--------------------------------------------------------------------------------"
  echo
  echo "Save Judge response as $workdir/04-judge.md"
  echo "The 'Curated TODO Items' block from the Judge is what you copy into TODO.md."
  echo
  echo "============================================================"
  echo "All review artifacts are in: $workdir"
  echo "Use the Judge's curated list to keep TODO.md authoritative and principle-aligned."
}

# --- Main --------------------------------------------------------------------

main() {
  local mode="interactive"
  local arg=""

  if [[ $# -ge 1 ]]; then
    case "$1" in
      --improve)
        mode="improve"
        arg="${2:-}"
        if [[ -z "$arg" ]]; then
          echo "Error: --improve requires a description in quotes." >&2
          usage
          exit 1
        fi
        ;;
      --review-todo)
        mode="review-todo"
        arg="${2:-}"
        if [[ -z "$arg" ]]; then
          echo "Error: --review-todo requires an area description in quotes." >&2
          usage
          exit 1
        fi
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        echo "Unknown option: $1" >&2
        usage
        exit 1
        ;;
    esac
  fi

  case "$mode" in
    improve)
      do_improve "$arg"
      ;;
    review-todo)
      do_review_todo "$arg"
      ;;
    interactive)
      # Original interactive wizard (preserved for users who prefer prompts)
      echo "=================================================="
      echo "  Crescent Moon Visibility - Agentic Review"
      echo "=================================================="
      echo
      read -rp "What is the title of this improvement / review? " TITLE
      read -rp "Short description (one sentence): " DESCRIPTION
      read -rp "Your name / handle: " REVIEWER

      local workdir
      workdir=$(create_workdir "$TITLE")
      echo
      echo "Working directory: $workdir"
      echo
      prefill_judge_template "$workdir" "$TITLE" "$REVIEWER" >/dev/null

      echo "1. IMPROVEMENT AGENT"
      echo "-------------------"
      echo "Copy the following prompt and spawn the Improvement Agent:"
      echo
      cat "$AGENTS_DIR/improvement-agent.md"
      echo
      echo "After the Improvement Agent responds, save its full output to:"
      echo "   $workdir/01-improvement.md"
      echo
      read -rp "Press Enter when the Improvement output is saved..."

      echo
      echo "2. VALIDATION AGENT"
      echo "------------------"
      echo "Copy the following prompt and spawn the Validation Agent:"
      echo
      cat "$AGENTS_DIR/validation-agent.md"
      echo
      echo "After the Validation Agent responds, save its full output to:"
      echo "   $workdir/02-validation.md"
      echo
      read -rp "Press Enter when the Validation output is saved..."

      echo
      echo "3. SECURITY REVIEW AGENT"
      echo "------------------------"
      echo "Copy the following prompt and spawn the Security Review Agent:"
      echo
      cat "$AGENTS_DIR/security-review-agent.md"
      echo
      echo "After the Security Review Agent responds, save its full output to:"
      echo "   $workdir/03-security.md"
      echo
      read -rp "Press Enter when the Security output is saved..."

      echo
      echo "4. JUDGE AGENT"
      echo "-------------"
      echo "Copy the following prompt and spawn the Judge Agent."
      echo "Make sure to attach the contents of 01-, 02-, and 03- files."
      echo
      cat "$AGENTS_DIR/judge-agent.md"
      echo
      echo "After the Judge responds, save its full structured decision to:"
      echo "   $workdir/04-judge.md"
      echo
      read -rp "Press Enter when the Judge decision is saved..."

      echo
      echo "============================================================"
      echo "All four stages complete."
      echo
      echo "Next steps:"
      echo "  1. Review the Judge decision in $workdir/04-judge.md"
      echo "  2. If approved, copy the verdict into your PR / commit"
      echo "  3. If the Judge requested follow-ups, create TODO items"
      echo
      echo "You can find all artifacts in:"
      echo "   $workdir"
      echo
      echo "Done."
      ;;
  esac
}

main "$@"
