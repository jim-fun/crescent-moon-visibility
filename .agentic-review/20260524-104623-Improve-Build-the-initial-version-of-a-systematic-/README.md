Agentic Improvement Run - ICOP Validation Harness (MVP)

This directory contains the full 4-stage agentic workflow artifacts plus the initial implementation work.

Stages completed:
- 01-improvement.md  (detailed proposal + design)
- 02-validation.md   (Validation Agent review)
- 03-security.md     (Security Review - Low risk)
- 04-judge.md        (Judge: Go)

Implementation progress made during this run:
- Directory structure created
- Curated sightings seed data + README
- Data loading and validation logic
- Basic CLI
- Approximate Yallop evaluation (framework working, calculation is placeholder)
- Makefile target added

Next logical step: Replace the approximation with real Yallop math (either by calling the CPU renderer "point" mode or by sharing the exact calculation).

All changes are on the dev branch.
