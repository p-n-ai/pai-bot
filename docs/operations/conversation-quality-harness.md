# Conversation quality harness

`cmd/conversation-harness` runs scripted learner conversations through the real
tutor engine. It separates deterministic conversation mechanics from
model-variable response checks:

- `wait` starts after earlier work finishes.
- `queue` preserves unfinished work and runs the message in arrival order.
- `interrupt` cancels the in-flight reply, suppresses queued stale replies, and
  runs the replacement message.
- `character` applies stable `first_name`, `username`, and `language` fields to
  every inbound message in one conversation.
- Conversation checks apply to every delivered reply or the full transcript.
  Turn checks apply only to that reply, which is useful for terse follow-ups.

The default fixture is
`internal/agent/testdata/ai_quality_conversations.yaml`. Its `queue`,
`interruptions`, and `short-replies` tags are the natural-conversation suite.

## Run it

Run the black-box pass/fail verifier:

```bash
just conversation-harness-verify
```

It builds the real CLI, checks queue, interruption, and terse-character cases,
requires an overlong reply to fail its turn-scoped limit, and requires a
misspelled fixture field to fail during loading. It does not need credentials or
network access.

Run the deterministic coordinator and command tests:

```bash
go test ./internal/conversationharness ./cmd/conversation-harness
```

Exercise one dynamic with a mock response:

```bash
go run ./cmd/conversation-harness \
  --tag interruptions \
  --mock-response "You can read 4x as four copies of x." \
  --show-responses
```

Run the naturalness cases against the configured provider:

```bash
go run ./cmd/conversation-harness --tag naturalness --show-responses
```

Use `--jsonl` for automation and `--dump-requests <path> --request-only` to
inspect model inputs without scoring. Request dumps are created with mode
`0600` because prompts can contain learner context.

JSON results include aggregate counts plus one safe structural outcome per turn:

```json
{
  "id": "Q13",
  "passed": true,
  "turns": 2,
  "delivered": 1,
  "interrupted": 1,
  "outcomes": [
    {"turn": 1, "delivery": "wait", "status": "interrupted", "duration_ms": 0},
    {"turn": 2, "delivery": "interrupt", "status": "delivered", "duration_ms": 0}
  ]
}
```

The JSON outcome intentionally omits learner text and model responses. Use
`--show-responses` for human review or the mode-`0600` request dump when model
input inspection is required.

## Fixture shape

```yaml
version: 2
provider: openai
characters:
  - id: aina
    first_name: Aina
    username: aina
    language: en
conversations:
  - id: NAT01
    title: Interruption replaces a stale answer
    tags: [interruptions, naturalness]
    character: aina
    turns:
      - user: "Explain every step."
        expect_status: interrupted
      - user: "Wait—just explain what 4x means."
        delivery: interrupt
        expect_status: delivered
        checks:
          max_response_chars: 320
          forbid_section_labels_on_turn: [2]
    checks:
      require_non_empty_replies: true
      forbid_fallback_message: true
```

`delivery` defaults to `wait`; `expect_status` defaults to `delivered`.
Supported statuses are `delivered`, `interrupted`, and `failed`. An optional
`after` value accepts a Go duration such as `25ms`, but zero-delay events are
preferable for deterministic queue and interruption cases.

Fixture decoding is strict. Unknown keys, unsupported delivery/status values,
duplicate IDs, invalid turn references, and unsupported versions fail before
the tutor engine or provider is constructed. Version 1 fixtures remain
supported when they use fields understood by this command.

The coordinator never exposes a canceled response, even if a provider returns
fallback text after cancellation. It also waits for canceled processor work
before returning, so harness runs do not leak background turns.

## Boundary

These delivery modes belong to the QA harness. They let a fixture state the
interaction contract it wants to evaluate, but they do not change channel
runtime behavior. Production ingress currently serializes and queues messages
per destination; a passing interruption case does not prove that a live channel
cancels an in-flight reply.
