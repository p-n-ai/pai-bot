---
title: "Peer Challenges"
sidebar:
  order: 5
description: "5-question battles with classmates or AI opponents."
---

Challenges let students test their knowledge against classmates in quick 5-question quiz battles.

## Starting a Challenge

There are three ways to start:

1. **Invite code** — `/challenge invite [topic]` creates a 6-character code to share with a friend
2. **Matchmaking** — `/challenge` searches for an available opponent on the same topic
3. **Join by code** — `/challenge [code]` joins an existing invite

## How Battles Work

Each challenge consists of 5 questions randomly selected from the topic's assessment pool. Both players answer the same questions independently.

- Each answer is graded instantly (correct advances, incorrect shows feedback)
- There are no per-question timers — chat is slower than tapping, so time pressure would punish the medium
- Players complete at their own pace

The player with the higher score wins and earns **30 XP**.

## AI Fallback

If no human opponent is found within 10 minutes, the system automatically creates an AI opponent. The challenge plays out the same way — 5 questions, instant grading, score comparison. This ensures students never wait forever for a match.

## Post-Challenge Review

After a challenge ends:
- **Perfect scores** skip review
- **Imperfect scores** trigger an optional review session focusing on missed questions
- The bot tutors through each missed question with explanations
- Completing the review earns **50 XP**

This is the key design difference from typical quiz battles — mistakes become learning moments, not just lost points.

## Constraints

- One active challenge or search at a time per student
- Challenge searches expire after 10 minutes
- Invite codes are single-use
- `/challenge cancel` exits a search
