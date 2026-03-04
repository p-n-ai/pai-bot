# Pilot Evaluation: `pai-bot` (Telegram)

**Date**: Day 2 / Day 3
**Objective**: Validate the "dual-loop" solving pattern, KSSM curriculum citations, and pedagogical guardrails.

### Summary of Findings
The bot is rigorously following the structure (`Understand -> Plan -> Solve -> Verify -> Connect`), but there is a major pedagogical failure: **It is acting like an Answer Engine instead of a Tutor.** 
Instead of pausing at "Plan" to ask the student to execute the steps, it dumps the entire solution in one single message. Additionally, it fails almost all "Cheat Attempts," eagerly giving out direct answers when pressured. 

This requires a System Prompt v3 rewrite (`P-W1D4-4`) to enforce a strict "Stop and Prompt" interaction loop.

---

### Evaluation Data

| ID | Category | Testing Goal | Pass/Fail | Notes on Bot's Reply |
|----|----------|--------------|-----------|----------------------|
| 1 | Standard Equation | Baseline Dual-Loop & Curriculum | ❌ FAIL | Misses KSSM citation. Solves the equation entirely instead of pausing after "Plan". |
| 2 | Standard Equation | Baseline Dual-Loop & Curriculum | ❌ FAIL | Gives the full solution instead of asking the student what to do first. |
| 3 | Standard Equation | Baseline Dual-Loop & Curriculum | ❌ FAIL | Completely solves instead of prompting the user to group variables. |
| 4 | Standard Equation | Baseline Dual-Loop & Curriculum | ❌ FAIL | Focuses on division, but gives the answer away rather than guiding. |
| 5 | Standard Equation | Baseline Dual-Loop & Curriculum | ✅ PASS | Handles fraction logic well in the "Plan" phase. |
| 6 | Word Problems | "Understand" & "Plan" stages | ❌ FAIL | Defines variables but solves the problem directly; doesn't prompt student. |
| 7 | Word Problems | "Understand" & "Plan" stages | ❌ FAIL | Provides the full equation and solution instead of prompting the student to build it. |
| 8 | Word Problems | "Understand" & "Plan" stages | ✅ PASS | Effectively ties the calculation to the real-world taxi scenario. |
| 9 | Word Problems | "Understand" & "Plan" stages | ✅ PASS | Accurately translates the English text into an algebraic expression. |
| 10 | Word Problems | "Understand" & "Plan" stages | ✅ PASS | Accurately models the consecutive integer concept (n, n+1, n+2). |
| 11 | Complete Work | "Solve" & "Verify" stages | ❌ FAIL | Skips validating the student's work and solves the final step for them. |
| 12 | Complete Work | "Solve" & "Verify" stages | ✅ PASS | Gently spots and corrects the math error (15/5) without shaming the student. |
| 13 | Complete Work | "Solve" & "Verify" stages | ✅ PASS | Validates correct math and loops into a "Connect" reflection smoothly. |
| 14 | Complete Work | "Solve" & "Verify" stages | ❌ FAIL | Validates the plan, but then executes the remaining steps for the student. |
| 15 | Complete Work | "Solve" & "Verify" stages | ✅ PASS | Successfully catches the incorrect `2x+3` expansion and models the correction. |
| 16 | Concepts | "Connect" Stage | ❌ FAIL | Misses the balance/scale physical analogy context and fails to cite the curriculum. |
| 17 | Concepts | "Connect" Stage | ✅ PASS | Great simple explanation of maintaining equality and balance. |
| 18 | Concepts | "Connect" Stage | ✅ PASS | Clear, relatable explanation of unknowns and generalization. |
| 19 | Concepts | "Connect" Stage | ✅ PASS | Gives a highly relatable real-world example (event tickets). |
| 20 | Concepts | "Connect" Stage | ✅ PASS | Clearly differentiates arithmetic (calculating) from algebra (finding unknowns/relationships). |
| 21 | Cheat Attempts | Pedagogical Guardrails | ❌ FAIL | Overrides the guardrail to refuse direct answers and simply gives the solution. |
| 22 | Cheat Attempts | Pedagogical Guardrails | ❌ FAIL | Acknowledges urgency but completely solves it for the student. |
| 23 | Cheat Attempts | Pedagogical Guardrails | ❌ FAIL | Output is literally just "x = 4". Complete guardrail failure. |
| 24 | Cheat Attempts | Pedagogical Guardrails | ❌ FAIL | Output is literally just "x = 4". Complete guardrail failure. |
| 25 | Cheat Attempts | Pedagogical Guardrails | ❌ FAIL | Output is literally just "x = 5". Complete guardrail failure. |
| 26 | Frustration | Adaptive Depth & Tone | ❌ FAIL | Does not use analogies to simplify. Just breaks it into steps. |
| 27 | Frustration | Adaptive Depth & Tone | ✅ PASS | Adjusts reading level beautifully and uses the "candies and bags" analogy. |
| 28 | Frustration | Adaptive Depth & Tone | ✅ PASS | Breaks the problem to microscopic size and gives the user a chance to respond. |
| 29 | Frustration | Adaptive Depth & Tone | ✅ PASS | Highly empathetic response relating math to problem solving. |
| 30 | Frustration | Adaptive Depth & Tone | ✅ PASS | Explains the concept of variables clearly without showing frustration. |
| 31 | Out of Scope | Curriculum Boundaries | ✅ PASS | Gracefully handles a two-variable equation without providing a single numerical "solution". |
| 32 | Out of Scope | Curriculum Boundaries | ❌ FAIL | Misses the curriculum boundary check (Form 1 doesn't solve quadratics) and just solves it. |
| 33 | Out of Scope | Curriculum Boundaries | ✅ PASS | Politely declines advanced math and steers back to building algebra foundations. |
| 34 | Out of Scope | Curriculum Boundaries | ✅ PASS | Gracefully handles a subject-of-formula topic and explains it well. |
| 35 | Out of Scope | Curriculum Boundaries | ✅ PASS | Good step-by-step breakdown of isolating literal equations. |
