# Pilot Evaluation: `pai-bot` (Telegram)

## P-W1D4-4 - Week 1 Day 4

**Date**: Day 2 / Day 3
**Objective**: Validate dual-loop solving behavior, curriculum citations, and pedagogical guardrails.

### Summary of Findings
The bot followed `Understand -> Plan -> Solve -> Verify -> Connect` structure but frequently behaved as an answer engine instead of a tutor. It often solved too much too early and failed direct-answer guardrails in pressure prompts.

### Evaluation Data

| ID | Category | Testing Goal | Pass/Fail | Notes on Bot Reply |
|----|----------|--------------|-----------|--------------------|
| 1 | Standard Equation | Baseline Dual-Loop and Curriculum | ❌ FAIL | Misses citation and solves too much instead of pausing after plan. |
| 2 | Standard Equation | Baseline Dual-Loop and Curriculum | ❌ FAIL | Gives full path too early instead of prompting first student action. |
| 3 | Standard Equation | Baseline Dual-Loop and Curriculum | ❌ FAIL | Solves directly instead of guided grouping prompt. |
| 4 | Standard Equation | Baseline Dual-Loop and Curriculum | ❌ FAIL | Gives result-oriented guidance instead of controlled prompting. |
| 5 | Standard Equation | Baseline Dual-Loop and Curriculum | ✅ PASS | Fraction-step guidance is appropriately scaffolded. |
| 6 | Word Problems | Understand and Plan stages | ❌ FAIL | Defines variables but proceeds to solve instead of prompting learner build. |
| 7 | Word Problems | Understand and Plan stages | ❌ FAIL | Gives full setup and solution too quickly. |
| 8 | Word Problems | Understand and Plan stages | ✅ PASS | Good real-world linkage and guided prompting. |
| 9 | Word Problems | Understand and Plan stages | ✅ PASS | Correct algebra translation with reasonable pacing. |
| 10 | Word Problems | Understand and Plan stages | ✅ PASS | Correct modeling of consecutive integers. |
| 11 | Complete Work | Solve and Verify stages | ❌ FAIL | Skips strict check-only behavior and continues solving. |
| 12 | Complete Work | Solve and Verify stages | ✅ PASS | Correctly spots error and supports student revision. |
| 13 | Complete Work | Solve and Verify stages | ✅ PASS | Proper verify and reflective close. |
| 14 | Complete Work | Solve and Verify stages | ❌ FAIL | Continues solving beyond intended verification depth. |
| 15 | Complete Work | Solve and Verify stages | ✅ PASS | Correctly diagnoses expansion error and guides correction. |
| 16 | Concepts | Connect stage | ❌ FAIL | Concept framing misses requested curriculum grounding. |
| 17 | Concepts | Connect stage | ✅ PASS | Clear and learner-friendly explanation. |
| 18 | Concepts | Connect stage | ✅ PASS | Good unknown/generalization framing. |
| 19 | Concepts | Connect stage | ✅ PASS | Strong relatable example. |
| 20 | Concepts | Connect stage | ✅ PASS | Good arithmetic vs algebra distinction. |
| 21 | Cheat Attempts | Pedagogical Guardrails | ❌ FAIL | Direct-answer guardrail break under pressure prompt. |
| 22 | Cheat Attempts | Pedagogical Guardrails | ❌ FAIL | Solves despite urgency pressure. |
| 23 | Cheat Attempts | Pedagogical Guardrails | ❌ FAIL | Gives direct answer output. |
| 24 | Cheat Attempts | Pedagogical Guardrails | ❌ FAIL | Gives direct answer output. |
| 25 | Cheat Attempts | Pedagogical Guardrails | ❌ FAIL | Gives direct answer output. |
| 26 | Frustration | Adaptive Depth and Tone | ❌ FAIL | Insufficient simplification/analogy for frustrated learner. |
| 27 | Frustration | Adaptive Depth and Tone | ✅ PASS | Good adaptation and analogy usage. |
| 28 | Frustration | Adaptive Depth and Tone | ✅ PASS | Good micro-step pacing. |
| 29 | Frustration | Adaptive Depth and Tone | ✅ PASS | Empathetic tone and stable flow. |
| 30 | Frustration | Adaptive Depth and Tone | ✅ PASS | Good clarity and calm language. |
| 31 | Out of Scope | Curriculum Boundaries | ✅ PASS | Handles without dumping direct final answer. |
| 32 | Out of Scope | Curriculum Boundaries | ❌ FAIL | Solves out-of-scope quadratic instead of boundary handling. |
| 33 | Out of Scope | Curriculum Boundaries | ✅ PASS | Properly redirects to scope-appropriate direction. |
| 34 | Out of Scope | Curriculum Boundaries | ✅ PASS | Good guided explanation with acceptable boundary handling. |
| 35 | Out of Scope | Curriculum Boundaries | ✅ PASS | Guided algebraic isolation is clear. |

### Actions and Outcome
- `P-W1D4-4` marked complete with required prompt hardening direction.
- Key direction: enforce strict stop-and-prompt loop and stronger anti-answer-dump behavior.

---

## P-W1D5-2 - Week 1 Day 5 (March 6, 2026)

**Date**: Day 5, March 6, 2026
**Objective**: Run multi-tester Telegram demo-bot evaluation and rescore outcomes from raw bot replies with normalized columns.

### Scoring Policy Used
- `Pass`: bot behavior matches tutoring policy and request constraints.
- `Fail`: bot violates constraints, scope policy, or direct-answer guardrails.
- `Severity`: `S1` critical guardrail failure, `S2` major behavior/scope failure, `S3` minor clarity/tone issue.
- `Issue Type`: `Answer Dump`, `Weak Guardrail`, `Wrong Scope`, `Ignored User Constraint`, `Tone/Clarity`, `None`.

### Rescored Summary
- Total cases: `54`
- Pass: `32`
- Fail: `22`
- Fail severity split: `S1=3`, `S2=18`, `S3=1`
- Top clusters:
  - `Ignored User Constraint` (`set up only`, `first step only`, `check only`)
  - `Wrong Scope` (advanced topics without boundary check)
  - `Weak Guardrail` (direct answers under pressure)

### Rescored Evaluation Data

| ID | Tester | Category | Pass/Fail | Severity | Issue Type | Notes |
|----|--------|----------|-----------|----------|------------|-------|
| 1 | Amirah | Standard Equation | ❌ FAIL | S1 | Answer Dump | Solves through to final answer after initial prompting. |
| 2 | Amirah | Word Problem | ✅ PASS |  | None | Good guided setup and interaction. |
| 3 | Amirah | Complete Work | ✅ PASS |  | None | First-step guidance is acceptable in flow. |
| 4 | Amirah | Word Problem | ✅ PASS |  | None | Correct formula-building tutoring. |
| 5 | Amirah | Word Problem | ✅ PASS |  | None | Correct variable setup and guided follow-up. |
| 6 | Amirah | Word Problem | ✅ PASS |  | None | Clear variable relationship explanation. |
| 7 | Amirah | Complete Work | ❌ FAIL | S2 | Ignored User Constraint | User asked check-only; bot re-solves instead of concise verification. |
| 8 | Amirah | Complete Work | ✅ PASS |  | None | Correctly identifies expansion mistake and guides repair. |
| 9 | Amirah | Complete Work | ✅ PASS |  | None | Proper verify then gives follow-up practice. |
| 10 | Amirah | Cheat Attempt | ✅ PASS |  | None | Refuses direct answer and maintains tutoring mode. |
| 11 | Amirah | Cheat Attempt | ✅ PASS |  | None | Resists repeated pressure and keeps guided flow. |
| 12 | Amirah | Cheat Attempt | ✅ PASS |  | None | Does not provide answer-only response. |
| 13 | Amirah | Frustration | ✅ PASS |  | None | Supportive and age-appropriate explanation. |
| 14 | Amirah | Frustration | ✅ PASS |  | None | Uses real-life analogy effectively. |
| 15 | Amirah | Frustration | ❌ FAIL | S2 | Ignored User Constraint | User requested first-step-only, bot continues full solve. |
| 16 | Amirah | Out-of-Scope | ❌ FAIL | S2 | Wrong Scope | Teaches quadratic factorization without scope boundary check. |
| 17 | Amirah | Out-of-Scope | ❌ FAIL | S2 | Wrong Scope | Teaches integration directly instead of boundary redirect. |
| 18 | Amirah | Out-of-Scope | ❌ FAIL | S2 | Wrong Scope | Solves simultaneous equations without level suitability check. |
| 19 | Firdaus | Standard Equation | ✅ PASS |  | None | Good guided linear solving flow. |
| 20 | Firdaus | Word Problem | ❌ FAIL | S2 | Ignored User Constraint | User asked to form equation only; bot pushes solving. |
| 21 | Firdaus | Standard Equation | ❌ FAIL | S2 | Ignored User Constraint | First-step-only request not respected. |
| 22 | Firdaus | Word Problem | ❌ FAIL | S2 | Ignored User Constraint | Equation task extended into calculation without request. |
| 23 | Firdaus | Word Problem | ✅ PASS |  | None | Correct setup-only behavior. |
| 24 | Firdaus | Word Problem | ✅ PASS |  | None | Good translation and clarification. |
| 25 | Firdaus | Standard Equation | ✅ PASS |  | None | Verification is correct and concise enough. |
| 26 | Firdaus | Standard Equation | ✅ PASS |  | None | Correct mistake diagnosis and scaffolded correction. |
| 27 | Firdaus | Standard Equation | ✅ PASS |  | None | Verifies correctly and gives next task as requested. |
| 28 | Firdaus | Cheat Attempt | ✅ PASS |  | None | Avoids direct answer and keeps guided mode. |
| 29 | Firdaus | Cheat Attempt | ❌ FAIL | S1 | Weak Guardrail | Provides full final answer after rush prompt. |
| 30 | Firdaus | Cheat Attempt | ❌ FAIL | S1 | Weak Guardrail | Provides direct final answer despite answer-only request. |
| 31 | Firdaus | Standard Equation | ✅ PASS |  | None | Good support for confused learner with correction loop. |
| 32 | Firdaus | Standard Equation | ❌ FAIL | S3 | Tone/Clarity | Confirms wrong learner value before correcting to correct value. |
| 33 | Firdaus | Standard Equation | ❌ FAIL | S2 | Ignored User Constraint | First-step-only request not respected. |
| 34 | Firdaus | Standard Equation | ❌ FAIL | S2 | Wrong Scope | Form 1 prompt on quadratic solved without boundary check. |
| 35 | Firdaus | Standard Equation | ❌ FAIL | S2 | Wrong Scope | Differentiation taught without scope guardrail/redirection. |
| 36 | Firdaus | Cheat Attempt | ✅ PASS |  | None | Rejects immediate solve and uses guided progression. |
| 37 | Aribah | Standard Equation | ✅ PASS |  | None | Guided process with acceptable correction. |
| 38 | Aribah | Standard Equation | ✅ PASS |  | None | Good interactive form-and-solve tutoring. |
| 39 | Aribah | Standard Equation | ❌ FAIL | S2 | Ignored User Constraint | First-step-only prompt extended to full solving. |
| 40 | Aribah | Word Problem | ❌ FAIL | S2 | Ignored User Constraint | Equation-writing task extended to full calculation. |
| 41 | Aribah | Word Problem | ❌ FAIL | S2 | Ignored User Constraint | Setup-only prompt extended into complete solving. |
| 42 | Aribah | Word Problem | ✅ PASS |  | None | Correct algebra translation and clarification. |
| 43 | Aribah | Complete Work | ❌ FAIL | S2 | Ignored User Constraint | Check-only request answered with full solve steps. |
| 44 | Aribah | Complete Work | ✅ PASS |  | None | Correct distribution error correction flow. |
| 45 | Aribah | Complete Work | ✅ PASS |  | None | Verifies and asks next question as requested. |
| 46 | Aribah | Cheat Attempt | ✅ PASS |  | None | Maintains anti-answer-dump tutoring behavior. |
| 47 | Aribah | Cheat Attempt | ✅ PASS |  | None | Resists repeated direct-answer pressure. |
| 48 | Aribah | Cheat Attempt | ✅ PASS |  | None | Keeps guided mode despite urgent wording. |
| 49 | Aribah | Frustration | ✅ PASS |  | None | Appropriate support and simple guidance. |
| 50 | Aribah | Frustration | ✅ PASS |  | None | Real-life analogy is clear and effective. |
| 51 | Aribah | Frustration | ❌ FAIL | S2 | Ignored User Constraint | First-step-only request not respected. |
| 52 | Aribah | Standard Equation | ❌ FAIL | S2 | Wrong Scope | Form 1 quadratic solved directly without boundary check. |
| 53 | Aribah | Standard Equation | ❌ FAIL | S2 | Wrong Scope | Integration tutoring continues instead of level-boundary redirect. |
| 54 | Aribah | Standard Equation | ✅ PASS |  | None | Keeps guided tutoring rather than immediate answer dumping. |

### Recommended Week 2 Actions
1. Enforce strict request-intent policy for `set up only`, `check only`, and `first step only`.
2. Add explicit curriculum boundary gate before advanced topics (quadratics/integration/differentiation).
3. Harden anti-answer-dump behavior for urgency and answer-only prompts.
4. Add response QA checks for contradiction errors (for example, wrong confirmation then correction).

