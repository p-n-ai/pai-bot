---
title: "Onboarding & i18n"
sidebar:
  order: 9
description: "Student onboarding flow, form selection, and language support."
---

## Onboarding Flow

When a student first messages the bot (or sends `/start`), they go through a guided onboarding:

1. **Welcome message** — The bot introduces itself
2. **Language selection** — Choose English, Bahasa Melayu, or 中文
3. **Form selection** — "Tingkatan berapa?" — Choose Form 1, Form 2, or Form 3
4. **Ready to learn** — The bot loads the correct curriculum and begins

Returning students skip onboarding and resume where they left off.

## Language Support

P&AI Bot supports three languages:
- **English** (en)
- **Bahasa Melayu** (ms)
- **中文 Chinese** (zh)

Language affects:
- All bot UI messages (buttons, prompts, celebrations)
- The system prompt instruction ("Respond in Bahasa Melayu")
- Nudge messages
- Quiz feedback

Students can change language anytime with `/language`.

### Language Detection

Language is determined in this order:
1. Stored preference (from onboarding or `/language`)
2. Telegram `language_code` (fallback for new users)
3. Default: English

### Disabling Multi-Language

Set `LEARN_DISABLE_MULTI_LANGUAGE=true` to skip the language selection step during onboarding. The bot will use English for all interactions.

## In-Chat Rating

After every few AI responses (configurable via `LEARN_RATING_PROMPT_EVERY_REPLIES`, default 5), the bot asks students to rate the response quality using inline star buttons. Ratings are logged for quality monitoring and are deduplicated per rated message.
