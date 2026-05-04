---
title: "Tutor Behavior Contract"
summary: "Runtime contract for enforcing the pai-bot tutor personality with deterministic guards, runtime behavior mechanisms, and smoke-test harness coverage."
read_when:
  - You are changing tutor personality, answer pacing, prompt privacy, or algebra scope behavior.
  - You are deciding whether a tutor behavior belongs in prompt wording, runtime guards, unit tests, or the AI quality harness.
---

# Tutor Behavior Contract

This slice moves tutor behavior out of prompt wording where code can own it.
The AI quality harness remains a smoke test; deterministic guards and unit tests are the stronger proof.

## Behavior-drilling notes

Runtime mechanisms worth using:

| Mechanism | Runtime pattern | Pai-bot adaptation |
|---|---|---|
| Runtime owns the route | Session state and dispatch decide the path before the model runs. | Quiz, prompt-privacy, and scope gates run before normal tutor AI. |
| Trust labels, not trust vibes | External content is wrapped with explicit untrusted boundaries and provenance. | Tutor context packets stay trust-labeled; learner/model text is quoted data. |
| Control tokens are parsed and stripped | Inline directive tags are machine-readable, then removed from display/history surfaces. | Tutor review tokens and hidden-prompt leakage are sanitized after model output. |
| Suspicious text is detectable | Injection-like patterns are matched by code and covered by unit tests. | Hidden/system-prompt requests are refused deterministically before AI. |
| Transcript hygiene is provider/runtime policy | Bad transcript shapes are repaired in code, not explained to the model. | Answer dumping and scope leakage are blocked by output guards where detectable. |
| Harness is a smell detector | Fixtures check broad outcomes, not exact phrasing. | Conversation harness covers naturalness and regressions; unit tests cover contracts. |

The useful idea is layered control: route, provenance, sanitization, deterministic tests, then model prompt.

## Tutor contract

| Student situation | Runtime contract | Hard proof |
|---|---|---|
| asks for first step or hint only | one guiding move/question, no final answer | output guard removes detectable final answer |
| asks for setup only | define variables/equation only, stop before solving | output guard blocks solved value |
| asks to check only | verify briefly, first mistake or confirmation, no full restart | prompt policy + harness smoke test |
| asks hidden/system prompt | refuse briefly and redirect to math | pre-AI deterministic guard |
| confused or frustrated | one tiny explanation plus one tiny check question | prompt policy + label stripper |
| asks for practice | one question only, no answer/solution | output guard removes detectable answer |
| outside Form 1-3 Algebra lane | deterministic redirect to nearest algebra prerequisite | pre-AI deterministic guard |

## Guard layering

1. Conversation state gates: onboarding, language selection, rating, goal, quiz.
2. Tutor deterministic gates: hidden-prompt extraction and algebra-scope redirect.
3. Prompt compiler: trust-labeled context and tutor policy.
4. Model output post-processing: instruction leak suppression, answer-dump suppression where detectable, short-reply label stripping.
5. Regression checks: unit tests for deterministic guards; conversation harness for live model smoke.
