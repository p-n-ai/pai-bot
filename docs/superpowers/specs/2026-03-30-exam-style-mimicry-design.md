# Exam-Style Question Mimicry Design

**Task:** `P-W2D7-4`
**Date:** 2026-03-30
**Status:** Approved

## Goal

When a student exhausts static questions during a quiz, transparently generate more questions in UASA/SPM exam style using AI, so the quiz continues seamlessly up to 10 questions per session.

## Trigger

Quiz engine detects `currentIndex >= len(staticQuestions)` and total questions served < 10.

## Flow

1. Quiz starts with static questions from `assessments.yaml` (filtered by intensity/difficulty)
2. Student answers through static pool normally
3. When static questions are exhausted and total < 10:
   a. Pick 2-3 random exemplar questions from the same topic, filtered by quiz intensity first. If <2 match the intensity, use any available and specify target difficulty in the prompt.
   b. Load the topic's teaching notes for curriculum context
   c. Call `CompleteJSON` (cheapest model, `ai.TaskGrading`) with exam-mimicry prompt
   d. Request 3 questions as structured JSON matching `QuizQuestion` schema
   e. Parse response, append to session question list
   f. Continue quiz — student sees no difference
4. Quiz ends when: student answers question 10 OR clicks [Stop] OR all generated questions are served

## Question Cap

- Maximum 10 questions per quiz session (static + generated combined)
- If static pool has >= 10 questions for the intensity, no generation needed
- Generation requests only enough to reach 10: if 7 static served, generate 3

## Prompt Template

```
You are a KSSM Mathematics exam question writer for Malaysian secondary students.

Generate {n} new questions for:
- Topic: {topic_name} ({topic_id})
- Syllabus: {syllabus_id}
- Difficulty: {intensity}

Curriculum context:
{teaching_notes_excerpt}

Use these real exam questions as style and format references:
{exemplar_questions_json}

Requirements:
- Match the style, format, and difficulty of the examples
- Each question must have: text, answer (type + value + working), difficulty, hints (2 levels), and distractors (for multiple_choice)
- Use Bahasa Melayu or English matching the exemplar language
- Include LaTeX math notation where appropriate (e.g., $2x + 3$)
- Do not duplicate any of the example questions

Return a JSON array of questions.
```

## Structured Output Schema

The `CompleteJSON` call expects this JSON schema:

```json
[
  {
    "text": "Solve: $3x + 5 = 20$",
    "difficulty": "medium",
    "answer_type": "exact",
    "answer": "5",
    "working": "3x + 5 = 20\n3x = 15\nx = 5",
    "hints": [
      {"level": 1, "text": "Start by subtracting 5 from both sides."},
      {"level": 2, "text": "After subtracting: 3x = 15. Now divide by 3."}
    ],
    "distractors": []
  }
]
```

For `multiple_choice` type:
```json
{
  "text": "Simplify: $(2x + 3) - (x - 1)$\n\nA) $x + 4$\nB) $x + 2$\nC) $3x + 2$\nD) $3x + 4$",
  "difficulty": "medium",
  "answer_type": "multiple_choice",
  "answer": "A",
  "working": "= 2x + 3 - x + 1 = x + 4",
  "hints": [
    {"level": 1, "text": "Be careful with the negative sign when removing brackets."},
    {"level": 2, "text": "-(x - 1) becomes -x + 1, not -x - 1."}
  ],
  "distractors": [
    {"value": "B", "feedback": "You may have subtracted 1 instead of adding it."},
    {"value": "C", "feedback": "You forgot to subtract x from 2x."},
    {"value": "D", "feedback": "Check both the x terms and constant terms."}
  ]
}
```

## Exemplar Selection

1. Get all static questions for the topic
2. Filter by current quiz intensity (difficulty match)
3. If >= 2 match: pick 2-3 at random
4. If < 2 match: use all available questions (any difficulty) and specify target difficulty explicitly in the prompt
5. Serialize selected exemplars as JSON for the prompt

## Error Handling

- `CompleteJSON` fails (network, provider down): end quiz with current score summary, log warning
- AI returns invalid JSON: end quiz gracefully, log warning
- AI returns fewer questions than requested: use what we got, generate more on next exhaustion if still < 10
- AI returns duplicate of existing question: serve it anyway (deterministic dedup is complex, not worth the effort for v1)

## Files

| File | Change |
|------|--------|
| `internal/agent/quiz_generate.go` | New: generation logic, prompt template, exemplar selection, response parsing |
| `internal/agent/quiz_generate_test.go` | New: unit tests for generation |
| `internal/agent/quiz_runtime.go` | Modify: detect exhaustion, call generator, append to session |
| `internal/agent/quiz.go` | Modify: add method to append questions to session, expose max question cap |

## Testing

- Unit: exemplar selection filters by difficulty correctly
- Unit: exemplar selection falls back when <2 match
- Unit: prompt template renders correctly with exemplars + teaching notes
- Unit: generated questions parse into QuizQuestion structs
- Unit: quiz session respects 10-question cap
- Unit: graceful end on generation failure
- Integration: terminal-chat quiz goes past static questions into AI-generated ones
