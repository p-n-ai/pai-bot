---
title: "Bot Commands"
sidebar:
  order: 8
description: "All available slash commands for the bot."
---

P&AI Bot supports the following slash commands. These are available in Telegram's command autocomplete menu and work in all chat channels.

## Learning Commands

| Command | Description |
|---------|-------------|
| `/start` | Begin a new learning session. Creates your account, asks your form level (Form 1/2/3), and preferred language |
| `/learn [topic]` | Set your current topic and start a teaching session. Example: `/learn linear equations` |
| `/progress` | View your learning progress — mastery bars per topic, XP, streak, active goals, and next review date |
| `/clear` | Reset the current conversation context and start fresh |

## Language & Settings

| Command | Description |
|---------|-------------|
| `/language` | Change your preferred language. Choose from English, Bahasa Melayu, or 中文 (Chinese) |

## Motivation & Social

| Command | Description |
|---------|-------------|
| `/goal` | Set a learning goal using natural language. Example: `/goal master linear equations by Friday` |
| `/goal clear` | Remove all active goals |
| `/challenge` | Start a peer challenge — searches for an opponent or resumes an active challenge |
| `/challenge invite [topic]` | Create a challenge with a shareable 6-character invite code |
| `/challenge [code]` | Join a challenge using an invite code |
| `/challenge cancel` | Cancel an active challenge search |
| `/leaderboard` | View the weekly leaderboard for your study group (top 10 by mastery gain) |

## Groups

| Command | Description |
|---------|-------------|
| `/create_group [name]` | Create a new study group |
| `/join [code]` | Join an existing study group using its code |

## Other

| Command | Description |
|---------|-------------|
| `/help` | List all available commands |

## Dev Commands

These are only available when `LEARN_FEATURES_DEV_MODE=true`:

| Command | Description |
|---------|-------------|
| `/dev_reset` | Full reset: mastery, XP, streaks, and goals |
| `/dev_boost` | Boost current topic mastery (default 85%) |
| `/dev_close_group` | Toggle a group between open and closed |

## Natural Language

Many features work without commands. Students can just type naturally:

- "Quiz me on algebra" → starts a quiz
- "I want to learn about fractions" → sets the topic
- "Challenge me" → starts matchmaking
- "What's my progress?" → shows progress
