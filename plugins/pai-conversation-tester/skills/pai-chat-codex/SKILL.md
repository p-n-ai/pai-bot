---
name: pai-chat-codex
description: Run and evaluate live, multi-turn PaiBot conversations through the local `just chat-codex` CLI and PaiBot's managed Codex application provider. Use when the user asks Codex to talk to PaiBot, smoke-test tutor tone or prompt flow, probe short replies and context continuity, collect a real terminal transcript, or reproduce a conversational issue without PostgreSQL.
---

# PaiBot Conversation Tester

Converse with PaiBot as a learner through its real tutor engine. Treat the transcript as qualitative evidence, not deterministic proof.

## Establish the boundary

1. Resolve the PaiBot checkout with `git rev-parse --show-toplevel`; read its nearest `AGENTS.md`.
2. Confirm `just --dry-run chat-codex` contains `--memory`, `--provider codex`, and `--interactive`. Stop if it can open PostgreSQL or allow provider fallback.
3. Treat an explicit request to use this skill as authorization for one bounded live evaluation of up to eight learner messages. Ask before exceeding that scope.
4. Never print environment variables, credentials, the isolated Codex home contents, or device codes in the final report.
5. Do not use browser automation. If the CLI requests device authorization, show the user the safe verification URL and ask them to complete it; continue after the CLI reports connection.
6. Test only unless the user also asks for a fix. Do not edit, commit, push, or open a PR from a test request.

## Plan the conversation

Choose a plausible learner and a concrete learning goal. Adapt each message to PaiBot's previous reply instead of replaying a fixed script.

Within four to eight messages, cover the relevant subset:

- A natural opener with enough context to teach.
- A terse or ambiguous follow-up such as `why?`, `2`, or `wait`.
- A reference to earlier context without repeating it.
- A correction or change of mind.
- A request to shorten, reframe, or switch language.
- One repair attempt after a weak answer to distinguish recoverable dialogue from a persistent failure.

Do not tell PaiBot it is under evaluation unless the requested scenario requires that behavior.

## Run the live PTY

Launch from the repository root:

```text
exec_command(
  cmd="just chat-codex",
  tty=true,
  yield_time_ms=1000
)
```

Keep the returned session ID. Then:

1. Wait for `Codex provider ready.`, `Codex connected.`, or the device-login instructions.
2. Wait for `Interactive chat ready.` and `You> `.
3. Send `/status\n`. Require `provider=codex` and `memory=true`; retain the displayed character and prompt hash for the report.
4. Send exactly one learner message with `write_stdin(chars="<message>\n")`.
5. Wait for `P&AI> ` and the next `You> `. Poll with empty input when needed; do not send duplicate messages.
6. Read the response, update the learner's next message, and continue.
7. Send `/exit\n` when the evaluation is complete.
8. If shutdown hangs, send Ctrl-C once and verify the process exits.

When the requested evaluation covers delivery mechanics, send an additional learner message while a reply is in flight and require `[queued #N]`, or send `/interrupt <replacement>` and require `[interrupted]` before the replacement reply. Do not use these controls during an ordinary tone evaluation.

## Reload a candidate

`just chat-codex` watches `.codex/chat-codex-candidate/candidate.yaml`. Keep prompt and characters in that single file so one save is one coherent candidate. A change prints `Candidate changed; type /reload to apply.`

- `/reload` validates the entire candidate and starts a fresh memory session.
- `/new` starts a fresh memory session while preserving the selected character.
- `/character <id>` selects a configured character and starts fresh.
- `/status` reports only safe structural state, including `sha256:<hash>`.

Edit `candidate.yaml` only when the user asks to tune or compare a prompt or character. Never place credentials, private learner data, or raw production prompts there.

Keep user-facing status updates under one minute apart while a turn is still running.

## Judge the conversation

Evaluate observable behavior:

- **Continuity:** understands pronouns, terse replies, and earlier facts.
- **Brevity fit:** response length matches the learner's request and message size.
- **Teaching:** guides rather than dumping an answer or worksheet.
- **Repair:** responds naturally to correction or confusion.
- **Voice:** conversational, age-appropriate, and free of canned headings.
- **Language:** follows the learner's language and requested changes.
- **Flow:** no duplicate replies, unexplained fallback, hang, or lost turn.

Classify a finding by its likely owner:

- `prompt/content`: tone, verbosity, teaching decision, context interpretation.
- `flow/runtime`: ordering, cancellation, duplication, state loss, error handling.
- `uncertain`: one model-variable result without a reproducible pattern.

Do not convert a subjective dislike into a defect without quoting the triggering learner message and summarizing the observed consequence.

## Report

Lead with the result. Include:

- Provider path used and whether the session stayed memory-only.
- Active prompt hash and character ID.
- Number and shape of turns tested.
- What felt natural.
- Failures with the learner message, PaiBot behavior, and likely owner.
- Whether a second attempt reproduced or repaired the problem.
- The smallest next test or fix.

Keep device authorization details, credentials, and raw model-facing prompts out of the report. Provide a full transcript only when the user explicitly asks for it.
