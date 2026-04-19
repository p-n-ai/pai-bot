---
title: "What is P&AI Bot?"
sidebar:
  order: 1
description: "An open-source AI learning agent that teaches students through chat."
---

P&AI Bot is an open-source, self-hostable AI learning agent that teaches students through chat. It works on Telegram, WhatsApp, and any website via an embeddable widget.

Unlike generic AI chatbots, P&AI Bot doesn't wait for students to ask questions — it **initiates study sessions**, tracks mastery with spaced repetition, and keeps students motivated through battles, streaks, leaderboards, and goals.

## Core Philosophy

**Content is commodity. Motivation is the moat.**

Every AI can explain quadratic equations. P&AI Bot is the one that texts you at 3pm to review them, celebrates your 7-day streak, and lets you battle your classmate on the same questions.

## Who Is It For?

- **Students** — Get a patient, always-available tutor on the messaging app you already use
- **Teachers** — See which students struggle, nudge them, and track class progress from a dashboard
- **Schools** — Self-host on your own servers with full data ownership. No student data leaves your network
- **Developers** — Extend with new curricula, AI providers, or chat channels. Apache 2.0 licensed

## Current Curriculum

The first curriculum target is **KSSM Matematik** (Malaysian national syllabus) covering Form 1, Form 2, and Form 3, with Algebra topics as the primary validation target. The curriculum system is designed to support any structured syllabus — community contributions welcome.

## Key Capabilities

| Feature | Description |
|---------|-------------|
| **Chat Tutoring** | Structured problem-solving with curriculum citations |
| **Quiz Engine** | Static + AI-generated questions with exam-style mimicry |
| **Spaced Repetition** | SM-2 algorithm for optimal review scheduling |
| **Motivation Engine** | Streaks, XP, goals, milestones, leaderboards |
| **Peer Challenges** | 5-question battles with classmates or AI |
| **Proactive Nudges** | Scheduled review reminders respecting quiet hours |
| **Multi-Channel** | Telegram, WhatsApp, embeddable web widget |
| **Admin Dashboard** | Teacher, parent, and school admin views |
| **6 AI Providers** | OpenAI, Anthropic, Gemini, DeepSeek, OpenRouter, Ollama |
| **Self-Hostable** | Docker Compose or Kubernetes. Your data, your servers |

## Architecture

P&AI Bot is a **modular monolith** — a single Go binary with clean domain boundaries. See the [Architecture guide](/guides/architecture) for details.

## Getting Started

Ready to try it? Head to the [Setup guide](/getting-started/setup) to get running in under 10 minutes.
