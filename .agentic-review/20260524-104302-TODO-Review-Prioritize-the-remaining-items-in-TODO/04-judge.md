# Judge Decision

**Date**: 2026-05-24
**Change / Proposal Title**: Prioritize remaining TODO items with strong emphasis on external accuracy validation
**Submitted by**: Agentic Workflow (Improvement + Validation + Security Review)

---

## Verdict

**Final Verdict**:
- [x] **Go** – Approved for merge / implementation (with the curated list below)
- [ ] **Go with Conditions**
- [ ] **No-Go**

---

## Core Principles Scorecard

| Principle                    | Rating     | 1-Sentence Justification |
|-----------------------------|------------|--------------------------|
| **Accuracy First**          | Strong     | The proposal correctly elevates external validation work (ICOP + HMNAO comparison) as the highest priority. This is the most direct way to strengthen the project's central claim. |
| **Performance with Integrity** | Acceptable | No changes proposed that would risk the existing 96.97% fidelity bar or performance characteristics. |
| **Verifiability & Reproducibility** | Strong | The top items (systematic sighting comparison harness + golden dataset) directly improve the project's ability to demonstrate truth against independent observations. |
| **Minimalism & Portability** | Acceptable | The validation work can be implemented with low new dependency footprint. |

**Overall Principle Alignment**:  
Strong. The Improvement Agent correctly identified that the largest remaining gap is not in adding more features, but in proving that the current high-quality implementation matches reality. This prioritization is disciplined and aligned with the project's stated identity.

---

## Summary of Agent Inputs

- **Improvement Agent**: Produced a ruthless re-prioritization. Elevated external validation to High. Recommended deprioritizing older feature work (terrain, web UI, etc.) in favor of proving current accuracy claims.
- **Validation Agent**: Confirmed that existing internal tests (`TestRendererAccuracy`) do not address external truth. Strongly supported the new validation harness and golden dataset ideas.
- **Security Review Agent**: Low risk overall. Minor note on data provenance for ICOP sightings (easily mitigable by curating a versioned subset).

---

## Overall Rationale

The project has invested heavily in internal numerical fidelity (dual renderers, 96.97% pixel match, FP32+DD on Apple Silicon) and professional engineering practices. We now have a detailed Yallop comparison document. The missing piece for "Accuracy First" to be credible externally is systematic comparison against real observations.

The proposed focus on ICOP/HMNAO validation work, golden datasets, and (secondarily) additional criteria is the correct next phase. It protects and strengthens the hardest-won technical achievement rather than diffusing effort across many lower-impact features.

---

## Key Strengths

- Clear, principle-driven re-prioritization.
- Strong emphasis on Verifiability through external data.
- Practical, implementable suggestions (harness, golden dataset).
- Good recognition that many older TODO items should be deprioritized.

---

## Key Risks / Concerns

- External validation work is data-intensive and may surface uncomfortable results (systematic biases). This is actually desirable for scientific integrity.
- Scope creep risk on the validation harness — the Judge recommends starting small (curated 30–50 high-quality sightings) rather than trying to ingest the entire ICOP database at once.

---

## Decision & Conditions

**Verdict**: Go

**Conditions**:
1. The top item ("Build systematic external validation harness against ICOP") should be scoped to a minimal viable harness + curated dataset first (not a full live integration).
2. Any new criterion added later must also meet the existing ≥96% CPU/GPU pixel match standard (or have a clear justification and updated test gate).

---

## Required Follow-up Actions

| Action | Owner | Agent to Re-engage | Due |
|--------|-------|--------------------|-----|
| Update TODO.md with the Curated list below | Human / Documentation agent | — | Immediate |
| Start ICOP validation harness as a feature branch | — | Improvement Agent | After this review |
| Run full agentic workflow on the first implementation increment of the harness | — | Full 4-stage | When prototype exists |

---

## Curated TODO Items (ready for TODO.md)

### High Priority (Accuracy First + Verifiability)

- [ ] **High** - Build systematic external validation harness against ICOP sighting database
  Rationale: We have a high-quality computational implementation of the Yallop (1997) q-test and Odeh (2004) criteria, plus a detailed comparison document. However, we lack quantitative evidence of how well the current predictions match real-world naked-eye and telescopic observations. This is the single largest gap in the "Accuracy First" claim.
  Ties to Core Principles: Directly strengthens Accuracy First (non-negotiable) and Verifiability & Reproducibility. Without this, all internal 96.97% numbers are only self-consistency, not external truth.
  Suggested validation: Curated set of 50–100 ICOP positive/negative sightings from 2015–2025. Automated comparison script that ingests sighting reports (lat/lon, date, instrument, success/failure) and runs the renderer at those locations/times. Report precision/recall per category (A/B vs C/D/E) and confusion matrix. Add as `make validate-icop` or similar.
  From agentic review of TODO prioritization on 2026-05-24.

- [ ] **High** - Add HMNAO / UKHO lunar crescent visibility predictions as a comparison baseline
  Rationale: HMNAO publishes official predictions using (a version of) the Yallop method. Comparing our maps against their published tables for the same dates provides an independent implementation check and increases credibility.
  Ties to Core Principles: Strong Verifiability & Reproducibility + Accuracy First. Helps prove our Chebyshev + rise/set + Yallop logic produces equivalent results to the official source.
  Suggested validation: Manually or semi-automatically compare 10–20 dates against published HMNAO visibility tables. Document any systematic differences in first-visibility longitude or category boundaries.
  From agentic review of TODO prioritization on 2026-05-24.

### Medium Priority (supporting Accuracy + Performance)

- [ ] **Medium** - Implement at least one additional modern visibility criterion (start with a recent published method)
  Rationale: Having only Yallop + Odeh limits users who prefer other calibrated methods. Adding a third increases the tool's scientific utility.
  Ties to Core Principles: Supports Accuracy First by allowing direct side-by-side comparison. Improves Verifiability.
  Suggested validation: New `TestRendererAccuracy` variant that also exercises the new criterion. Ensure CPU/GPU match remains ≥96% for the new path.
  From agentic review of TODO prioritization on 2026-05-24.

- [ ] **Medium** - Create golden sighting test dataset and regression harness
  Rationale: As we add new criteria or modeling improvements, we need reproducible test cases that protect the accuracy bar.
  Ties to Core Principles: Excellent for Verifiability & Reproducibility and long-term protection of Accuracy First.
  Suggested validation: 20–30 high-quality ICOP or published sightings stored as JSON/CSV. Simple Go test or script that runs the renderer and asserts category or first-visibility time is within tolerance.
  From agentic review of TODO prioritization on 2026-05-24.

### Deprioritized / Move to Stretch Goals

(The older medium/low items such as terrain elevation modeling, atmospheric extinction, web-based UI, real-time notifications, configurable projections, and multi-language support are moved to a new "Future / Stretch Goals" section. They remain valuable long-term but score significantly lower on Accuracy First and Verifiability compared to external validation work at this stage of the project.)

---

## Next Steps Recommendation

1. Human: Paste the "Curated TODO Items" section above into `TODO.md` (replacing or augmenting the current Medium/Low/Research sections as appropriate).
2. Create a new feature branch for the ICOP validation harness (start small).
3. When the first increment exists, run the full 4-stage agentic workflow (`./scripts/agentic-review.sh --improve "Initial ICOP sighting comparison harness..."`).

---

## Judge Confidence

- [x] High
- [ ] Medium
- [ ] Low

**Judge Notes**: This was a high-quality, disciplined review. The Improvement Agent avoided the common trap of just adding more features and correctly focused on closing the external validation gap. The other agents provided useful calibration. I endorse the direction without reservation.

---

**Signed**: Judge Agent (via Grok 4.3)  
**Date**: 2026-05-24