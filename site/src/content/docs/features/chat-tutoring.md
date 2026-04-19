---
title: "Chat Tutoring"
sidebar:
  order: 1
description: "Structured problem-solving with curriculum-aware explanations."
---

The core of P&AI Bot is a chat-based tutoring experience that follows structured pedagogical patterns rather than free-form conversation.

## How It Works

When a student asks a math question, the bot follows a five-stage teaching flow:

1. **Understand** — Restate the problem to confirm understanding
2. **Plan** — Guide the student to think about approach before solving
3. **Solve** — Walk through the solution step-by-step
4. **Verify** — Check the answer and catch common mistakes
5. **Connect** — Link the concept to related topics in the curriculum

This dual-loop problem-solving pattern is inspired by [DeepTutor](https://github.com/HKUDS/DeepTutor)'s multi-agent reasoning architecture, adapted for chat-based K-12 tutoring.

## Curriculum Citations

Every explanation references the curriculum source path. For example:

> "This is from **KSSM Form 1 > Algebra > Linear Equations**"

This helps students connect what they're learning to their actual school syllabus and exam topics.

## Adaptive Explanation Depth

The bot adjusts its teaching style based on the student's mastery level:

| Mastery Level | Style |
|--------------|-------|
| **Beginner** (< 30%) | Simple language, more examples, smaller steps |
| **Developing** (30–60%) | Standard explanations, gradual formal notation |
| **Proficient** (> 60%) | Concise, focus on edge cases and cross-topic connections |

## Topic Detection

When a student mentions a math concept, the bot automatically detects the relevant curriculum topic and loads the corresponding teaching notes into context. This means explanations are grounded in the actual syllabus content, not generic AI knowledge.

## Conversation Continuity

The bot maintains conversation context through rolling compaction — older messages are summarized while recent exchanges stay in full. This means students can have long study sessions without the bot forgetting what they were discussing.
