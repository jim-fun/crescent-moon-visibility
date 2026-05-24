# Agentic Improvement Workflow

This document defines a structured, multi-agent workflow for continuously refining and improving the **Crescent Moon Visibility Maps Generator** project.

It is designed to be used with Grok's subagent system (`spawn_subagent`) in combination with the `crescent-moon-visibility-engineering` skill.

## Philosophy

We treat major changes, releases, and significant refactors as **agent-orchestrated processes** rather than solo human effort. The workflow enforces four distinct lenses:

1. **Improvement** — Creative, constructive enhancement
2. **Validation** — Empirical, test-driven verification (especially accuracy)
3. **Security Review** — Risk, supply-chain, and process hardening
4. **Judge** — Holistic, final decision-making with clear rationale

This mirrors professional engineering review boards while leveraging AI agents specialized via the project skill.

## Core Principles of the Project

All agents in this workflow **must** evaluate proposals and render decisions against the project's fundamental principles:

1. **Accuracy First** (Non-negotiable)
   - The project exists to produce reliable predictions suitable for real visual sighting.
   - Any change must preserve (or improve) the established accuracy bar (~96.97%+ exact per-pixel match with the CPU reference on the hardest cases).
   - Boundary ULP noise is tolerated; systematic bias or regression is not.

2. **Performance with Integrity**
   - We aggressively pursue speed (GPU acceleration, Chebyshev ephemeris, FP32+DD on Apple Silicon, etc.).
   - Performance gains are only valuable if they do not meaningfully degrade the accuracy guarantees.

3. **Verifiability & Reproducibility**
   - Changes must be testable and the results must be reproducible.
   - Release artifacts must be signed, checksummed, and clearly verifiable by end users.

4. **Minimalism & Portability**
   - We prefer fewer runtime dependencies (e.g., pure-Go blending over Python + OpenCV).
   - The tool should be buildable and usable across a wide range of platforms.

Every agent (Improvement, Validation, Security Review, and especially Judge) is expected to explicitly reference these principles in their reasoning.

## The Four-Stage Workflow

### Stage 1: Improvement Agent

**Goal**: Propose or implement concrete improvements, refactors, performance wins, documentation enhancements, or release process upgrades.

**Responsibilities**:
- Analyze current code, tests, workflows, and docs against the standards in the `crescent-moon-visibility-engineering` skill.
- Suggest or directly implement changes (with clear diffs or edit instructions).
- **Always evaluate proposals against the four Core Principles** defined above (Accuracy First is the highest priority).
- Consider performance (especially OpenCL/GPU kernels), usability, and maintainability.

**Reusable Prompt**:
See `scripts/agents/improvement-agent.md` for a ready-to-use, high-quality prompt template.

**Output expected**: List of proposed changes + rationale, or direct code edits (when the agent has write access).

### Stage 2: Validation Agent

**Goal**: Empirically validate that proposed changes meet the project's high standards for correctness and accuracy.

**Responsibilities**:
- Run relevant tests (`make test`, `make test-accuracy`, specific Go tests).
- Re-execute accuracy regression checks (especially `TestRendererAccuracy` for CPU vs GPU pixel match).
- Validate that changes uphold the **Accuracy First** and **Performance with Integrity** principles.
- Validate blending behavior, early-skip logic, legend correctness, and WEBP output.
- Check for behavioral drift between CPU reference and GPU paths.

**Key Artifacts to Validate**:
- `internal/blend/blend_test.go`
- `main_test.go` (especially `TestRendererAccuracy`)
- `docs/performance-accuracy.md` claims

**Reusable Prompt**:
See `scripts/agents/validation-agent.md` for a ready-to-use prompt template.

**Output expected**: Pass/fail with data, test logs, and specific recommendations for fixes if validation fails.

### Stage 3: Security Review Agent

**Goal**: Identify security, supply-chain, and process risks introduced by the changes.

**Responsibilities**:
- Review changes to release workflows, signing (Cosign), checksum generation, and artifact handling, with strong emphasis on **Verifiability & Reproducibility**.
- Check for introduction of new dependencies and assess their risk.
- Assess risks around the mixed-language nature (Go + C/C++ + OpenCL kernels).
- Review the `scripts/release.sh` and `Makefile` release targets for injection or tampering risks.
- Evaluate documentation of verification steps for end users.
- Flag any issues with pre-release vs stable release handling.

**Reusable Prompt**:
See `scripts/agents/security-review-agent.md` for a ready-to-use prompt template.

**Output expected**: Security findings (categorized by severity) + concrete mitigation recommendations.

### Stage 4: Judge Agent

**Goal**: Act as the final, authoritative gatekeeper. Deliver a clear, principle-driven verdict on whether proposed changes should be accepted, rejected, or accepted with conditions.

**Core Mandate**:
The Judge is the **guardian of the project's Core Principles**, especially **Accuracy First** and **Performance with Integrity**. No change may be considered complete without the Judge's explicit approval.

**Responsibilities**:
- Receive and synthesize the full outputs from the Improvement, Validation, and Security Review agents.
- Evaluate the proposal **principle-by-principle** against the four Core Principles defined above.
- Weigh trade-offs explicitly through the lens of Accuracy First (highest priority) and Performance with Integrity.
- Issue one of the following verdicts:
  - **Go** — Changes are ready to merge / proceed (minor suggestions optional).
  - **Go with Conditions** — Acceptable only after specific follow-up work is completed and re-validated.
  - **No-Go** — Rejected. Major issues must be resolved before re-entering the workflow.
- Provide clear, actionable rationale tied directly to the core principles.
- Request additional work from any previous agent if the input is insufficient.
- Ensure that approved changes are properly reflected in documentation (CHANGELOG, performance-accuracy.md, README, etc.).

**Judge Authority**:
The Judge has final say. Even if Improvement, Validation, and Security Review are positive, the Judge may still issue No-Go if the proposal violates Accuracy First or Performance with Integrity.

**Reusable Prompt**:
See `scripts/agents/judge-agent.md` for a ready-to-use prompt template.

**Required Output Format** (the Judge must follow this structure):

- **Verdict**: Go / Go with Conditions / No-Go
- **Core Principles Scorecard** (rate the proposal against each principle: Strong / Acceptable / Weak / Violated, with 1-2 sentence justification):
  - Accuracy First
  - Performance with Integrity
  - Verifiability & Reproducibility
  - Minimalism & Portability
- **Overall Rationale** (clear, concise explanation anchored to the principles)
- **Key Strengths**
- **Key Risks / Concerns**
- **Conditions or Required Follow-ups** (if any)
- **Recommendation for Next Steps**

---

## How to Kick Off Agents (Recommended)

The project provides a dedicated helper that makes the two most common workflows completely unambiguous:

- **Specific improvement** you want to make (feature, refactor, performance work, docs, etc.).
- **Review existing code / area** with the explicit goal of producing new, principle-aligned items for `TODO.md`.

### Primary Tool: `scripts/agentic-review.sh`

```bash
# See all options and examples
./scripts/agentic-review.sh --help
```

#### A. Kick Off for a Specific Improvement

Use this when you have a concrete change in mind (e.g. "add linux-arm64 to releases", "tune the FP32+DD kernel", "improve error messages in main.go").

```bash
./scripts/agentic-review.sh --improve \
  "Add linux-arm64 build to the release workflow while preserving Cosign signing and checksums"
```

What happens:
- The script creates a timestamped directory under `.agentic-review/`.
- It prints **exact, ready-to-paste blocks** for each of the four stages.
- Each block includes the full agent prompt + your task description + any necessary "CONTEXT: paste previous output here" instructions.
- It pre-populates `JUDGE_DECISION.md` from the official template.
- You copy the printed prompt for Improvement into a `spawn_subagent` call (or paste directly), save the full reply as `01-improvement.md`, then proceed to Validation with the next printed block (feeding prior output), and so on through Judge.

After the Judge returns a **Go** (or **Go with Conditions**), you have a complete, auditable record and a filled decision template.

#### B. Review Code to Add Items Directly to TODO.md

Use this when you want agents to scan an area and produce backlog items (exactly what the user asked for: "how I have them review the code to add to the TODO list").

```bash
./scripts/agentic-review.sh --review-todo \
  "GPU kernel accuracy risks on non-Apple OpenCL devices (FP32+DD path and Chebyshev handling)"
```

Special behavior in this mode:
- The Improvement agent receives **extra instructions** (from `scripts/agents/todo-review-instructions.txt`) telling it to emit every candidate using the **exact bullet format** already used in `TODO.md` (priority, rationale, ties to Core Principles, suggested validation).
- Validation and Security still run (they comment on test coverage and new risks).
- The **Judge** is explicitly asked to produce a final section titled:

  ```
  ## Curated TODO Items (ready for TODO.md)
  ```

  containing only the items the Judge endorses after principle-by-principle scoring (Accuracy First dominant). Each item includes a note such as "From agentic review of <area> on <date>".

- You copy the curated block straight into the appropriate section of `TODO.md`.

This is the cleanest, most repeatable way to keep the backlog fresh and aligned with the project's non-negotiable principles.

### Manual / Low-Level Control (still fully supported)

If you prefer to drive everything yourself without the helper script:

1. Write a clear one-sentence to one-paragraph task.
2. For a normal improvement, feed the four prompts in order (`cat scripts/agents/*.md`), always attaching prior stage outputs for stages 2–4.
3. For a TODO-generation review, prepend the content of `scripts/agents/todo-review-instructions.txt` to the Improvement prompt, and add the "produce a ## Curated TODO Items (ready for TODO.md)" instruction to the Judge prompt.
4. Always end by filling `JUDGE_DECISION.md` (copy from `scripts/agents/JUDGE_DECISION_TEMPLATE.md`).

The script simply automates the mechanical parts and guarantees the special TODO output format is used when needed.

**Quick one-liner reminders** (also shown by `make agentic-review`):

```bash
./scripts/agentic-review.sh --improve "Your specific improvement description here"
./scripts/agentic-review.sh --review-todo "Area of the codebase or docs to scan for new TODO items"
```

### Recommended Pattern for Recording Work

- Store all agent outputs for a review in a folder under `.agentic-review/`
- Commit the final `JUDGE_DECISION.md` (or a summary) when the work is merged
- Reference the review in commit messages or PR descriptions:
  > "Refactored X after agentic review (see .agentic-review/2026-05-24-.../04-judge.md)"

### Using Existing Specialized Agents

You can also map the four workflow roles to the agents we have already spawned in this session:

- Improvement → Release Engineering or GPU/OpenCL Performance Agent
- Validation → Testing & Accuracy Agent
- Security Review → Release Engineering Agent (strong on workflows)
- Judge → Any of the above + human oversight (recommended for important decisions)

---

## Quick Reference Commands

```bash
# Specific improvement (most common)
./scripts/agentic-review.sh --improve "Add linux-arm64 to release workflow with Cosign"

# Review code area and produce TODO items ready to paste
./scripts/agentic-review.sh --review-todo "Chebyshev coefficient handling and lag-time math in kernels"

# Safe push of main + tags to the public GitHub mirror (after Judge approval)
./scripts/push-to-github.sh --tags

# See all modes + examples
./scripts/agentic-review.sh --help

# Agent prompt templates
ls scripts/agents/

# Makefile reminder
make agentic-review
```

---

## Mapping Existing Agents to Workflow Roles

| Workflow Role       | Existing Specialized Agent(s)                  | When to Use |
|---------------------|------------------------------------------------|-------------|
| Improvement        | Release Engineering, GPU/OpenCL Performance, Documentation & Architecture | Most changes |
| Validation         | Testing & Accuracy Agent                       | Any change affecting output or accuracy |
| Security Review    | Release Engineering Agent (primary)            | Workflow, signing, dependency, or release changes |
| Judge              | Any senior agent + human oversight             | Final gate before merge to `main` |
| GitHub Migration   | `github-migration-agent.md` + `push-to-github.sh` | Safe, auditable promotion of `main` + tags to https://github.com/jim-fun/crescent-moon-visibility (run after Judge approval on release changes) |

---

## Success Criteria / Gates

A change passes the full workflow when:

- **Improvement** produces clear, well-reasoned enhancements aligned with the skill.
- **Validation** confirms no regression in the 96.97%+ pixel match target (or justified explanation).
- **Security Review** finds no high-severity issues, or all are explicitly accepted with mitigations.
- **Judge** issues a **Go** (or "Go with minor conditions") decision.

Changes that fail Validation or have unresolved high-severity security findings should not be merged to `main` without explicit human override (documented).

---

## Integration with Project Practices

This workflow is an extension of the engineering discipline captured in the `crescent-moon-visibility-engineering` skill. It is particularly recommended for:

- Changes to the release pipeline
- Modifications to accuracy-critical code (kernels, blending, Yallop math)
- Major architectural refactors
- Any change that will be part of a release

It complements (does not replace) human code review and the existing `make test` / `make test-accuracy` gates.

---

## Future Evolution

- Integrate with GitHub/Gitea PR checks (when agent invocation becomes more automated)
- Create specialized Judge personas per domain (e.g., "Accuracy Judge", "Release Judge")
- Possibly add non-interactive batch mode or GitHub Action that can invoke the four agents via the skill for fully automated reviews (longer term)

---

**Last Updated**: 2026-05-24 (agentic kickoff tooling + Judge template + clear TODO review path completed)  
**Maintained by**: The project maintainers + the specialized agents defined in the `crescent-moon-visibility-engineering` skill.