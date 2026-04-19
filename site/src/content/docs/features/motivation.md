---
title: "Motivation Engine"
sidebar:
  order: 4
description: "Streaks, XP, goals, milestones, and leaderboards."
---

P&AI Bot's motivation engine keeps students engaged through game mechanics that reward consistent learning.

## Streaks

Streaks track consecutive days of engagement. Missing a day resets the streak to zero. Milestone celebrations trigger at:

**3, 7, 14, 30, 60, and 100 days**

Each milestone comes with a localized celebration message and bonus XP.

## XP System

Experience points are earned across all activities:

| Activity | XP |
|----------|-----|
| Study session exchange | 10 |
| Quiz correct answer | 20 |
| Topic mastered | 50 |
| Streak milestone | 100 |
| Challenge won | 30 |
| Post-challenge review completed | 50 |

## Goals

Students can set learning goals using the `/goal` command or natural language:

- "I want to master linear equations by next week"
- "My goal is to finish all Form 1 Algebra"

The bot uses AI to parse vague goals into concrete targets with mastery thresholds (default 75%). Goals auto-update as the student progresses and show in the `/progress` view.

Multiple active goals are supported. Goals can be cleared with `/goal clear`.

## Milestones

Celebratory messages trigger automatically when students:
- Master a topic
- Hit XP thresholds
- Complete a subject area

Milestones are A/B tested — Group A sees milestone celebrations, Group B doesn't, allowing measurement of their impact on retention.

## Leaderboards

Weekly leaderboards rank students by mastery gain within their study group. The `/leaderboard` command shows the top 10. Rankings are membership-gated — students only see peers in their own group, preventing cross-tenant data leakage.

A Monday morning recap is sent automatically to all group members summarizing the week's leaderboard.

## Study Groups

Students can form or join study groups:
- `/create_group [name]` — Create a new study group
- `/join [code]` — Join an existing group with a 6-character code

Groups power the leaderboard and weekly recap features. Teachers can also create class groups from the admin panel.
