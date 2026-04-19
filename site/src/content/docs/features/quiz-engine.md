---
title: "Quiz Engine"
sidebar:
  order: 2
description: "Static and AI-generated quizzes with exam-style formatting."
---

The quiz engine tests student understanding with a mix of pre-authored and AI-generated questions.

## Starting a Quiz

Students can start a quiz naturally — no `/quiz` command required. Just say something like:

- "Quiz me on linear equations"
- "I want to practice algebra"
- "Test me"

The bot detects the intent, loads questions for the relevant topic, and begins.

## Question Sources

### Static Questions (OSS Pool)
Questions are loaded from the Open School Syllabus `assessments.yaml` files. Each question includes:
- Difficulty level (easy, medium, hard)
- Learning objectives and TP level (1–6)
- Progressive hints (multi-level)
- Distractors with targeted feedback
- KBAT flags for higher-order thinking questions
- Exam provenance tags (UASA/SPM format)

### Dynamic Questions (AI-Generated)
When the static pool is exhausted (fewer than 5 remaining for a topic), the AI generates additional questions using `CompleteJSON` on the cheapest available model. Generated questions follow exam-style mimicry — using 2–3 real UASA/SPM exemplar questions as style references.

## Grading

Grading is deterministic and one-shot per question:

- **Multiple choice** — Exact match required
- **Free text** — Token normalization and structured math expression parsing
- **Numeric ranges** — Tolerance-based matching

On correct answers, students advance immediately. On incorrect answers, the bot provides:
1. Distractor-matched feedback (explains why the wrong answer is wrong)
2. The first available hint
3. Option to try again or move on

## Quiz Flow

Quizzes support up to 10 questions per session. The bot handles side conversations gracefully — if a student asks an off-topic question mid-quiz, the bot pauses the quiz, answers the question, then resumes. It doesn't grade off-topic messages as wrong answers.

At the end, students see a summary with their percentage score, and quiz performance feeds into the mastery tracking system.
