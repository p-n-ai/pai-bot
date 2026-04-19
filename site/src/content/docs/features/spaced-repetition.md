---
title: "Spaced Repetition"
sidebar:
  order: 3
description: "SM-2 algorithm for optimal review scheduling and mastery tracking."
---

P&AI Bot uses the **SuperMemo 2 (SM-2)** algorithm to schedule reviews at optimal intervals, ensuring students retain what they learn.

## How Mastery Works

Each topic has a mastery score from 0 to 1. The score is updated after every study interaction using a blended formula: **70% historical mastery + 30% new session delta**.

| Score | Level | What It Means |
|-------|-------|---------------|
| 0 – 0.29 | Beginner | Just started or struggling |
| 0.30 – 0.59 | Developing | Making progress |
| 0.60 – 0.74 | Approaching | Almost there |
| 0.75 – 1.0 | Mastered | Ready to move on |

When mastery reaches **0.75**, the topic is considered mastered. This triggers:
- XP reward (50 XP)
- Milestone celebration message
- Prerequisite check to unlock dependent topics

## Review Scheduling

The SM-2 algorithm determines when to review each topic:

- **First review** — 1 day after learning
- **Second review** — 6 days after first review
- **Subsequent reviews** — Previous interval × ease factor

The **ease factor** (minimum 1.3) adjusts based on performance. Strong performance increases the interval; struggles bring it back down. Missed reviews reset the interval to 1 day without harsh penalties — the ease factor only decreases slightly.

## Topic Unlocking

Topics follow a prerequisite graph. When a student masters a topic, the system checks which dependent topics are now unlocked and notifies the student. This creates a natural learning progression through the curriculum.

## Adaptive Depth

The mastery score also controls how the bot explains things — see [Chat Tutoring](/features/chat-tutoring) for details on how beginner, developing, and proficient students get different explanation styles.
