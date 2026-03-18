---
title: "Challenge Invite Slice"
summary: "How the first /challenge invite-code slice works today, in Gherkin and in machine-level terms."
read_when:
  - You are changing /challenge invite-code flow
  - You are adding matchmaking, AI fallback, or challenge runtime and need the current baseline first
  - You want to understand the challenge slice without reading Go syntax line by line
---

# Challenge Invite Slice

This doc explains the first shipped `/challenge` slice.
It also sketches the next planned slice in Gherkin form.

It only covers invite-code creation and join.

The shipped code does not yet cover:

- human matchmaking
- AI fallback opponent
- frozen question snapshots
- challenge attempt runtime
- grading, settlement, XP, or review

## Current scope

Today the runtime supports two command shapes:

- `/challenge invite <topic>`
  Create a challenge and return a 6-character code.
- `/challenge <code>`
  Join an existing waiting challenge by code.

The current state machine is intentionally small:

- `waiting`
  Creator exists. No opponent yet. Code is joinable.
- `ready`
  Opponent joined. Code is no longer joinable.

## Gherkin

```gherkin
Feature: Invite-code challenge slice
  As a student
  I want to create or join a challenge by code
  So I can start a peer battle flow later

  Scenario: Create a challenge from an explicit topic
    Given challenge mode is enabled
    And I am not currently inside quiz mode
    And the topic "linear equations" resolves to an assessed curriculum topic
    When I send "/challenge invite linear equations"
    Then the engine creates a challenge
    And the challenge state is "waiting"
    And the match source is "invite_code"
    And the response includes a 6-character challenge code

  Scenario: Create a challenge from recent conversation context
    Given challenge mode is enabled
    And I am not currently inside quiz mode
    And my conversation already points at an assessed topic
    When I send "/challenge invite"
    Then the engine reuses the resolved topic from conversation context
    And the challenge state is "waiting"

  Scenario: Join a waiting challenge
    Given a challenge exists with code "ABC123"
    And that challenge is in state "waiting"
    And I am not the creator
    When I send "/challenge ABC123"
    Then the engine joins me to that challenge
    And the challenge state becomes "ready"
    And the response shows mini info about the creator, topic, and question count

  Scenario: Reject self-join
    Given I created a challenge with code "ABC123"
    When I send "/challenge ABC123"
    Then the engine rejects the join
    And the response says I cannot join my own challenge code

  Scenario: Reject an invalid or unavailable code
    Given no joinable challenge exists for code "ABC123"
    When I send "/challenge ABC123"
    Then the engine does not create or join anything
    And the response says the code is invalid or unavailable

  Scenario: Block challenge commands during quiz mode
    Given my conversation is currently owned by quiz mode
    When I send "/challenge invite linear equations"
    Then the challenge command is blocked
    And the response tells me to finish or cancel the quiz first

  Scenario: Bare /challenge is only a placeholder today
    Given challenge mode is enabled
    And I am not currently inside quiz mode
    When I send "/challenge"
    Then the engine does not enter matchmaking yet
    And the response shows the currently supported invite-code commands
```

## Planned next-slice Gherkin

This section is planned behavior, not current code.

```gherkin
Feature: Matchmaking and AI fallback challenge slice
  As a student
  I want /challenge to find me an opponent fast
  So I can play now without sharing a code manually

  Scenario: Start human matchmaking from a resolved topic
    Given challenge mode is enabled
    And I am not currently inside quiz mode
    And the engine can resolve a challenge topic from my command or conversation context
    When I send "/challenge"
    Then the engine creates or resumes one matchmaking ticket for me
    And the ticket state is "searching"
    And the response shows searching status plus cancel controls

  Scenario: Prompt topic selection when topic is not resolvable
    Given challenge mode is enabled
    And I am not currently inside quiz mode
    And no challenge topic can be resolved confidently
    When I send "/challenge"
    Then the engine does not silently guess a topic
    And the response prompts me to pick or clarify a topic

  Scenario: Pair two compatible human players
    Given student A has a searching ticket for topic "linear equations"
    And student B has a searching ticket for topic "linear equations"
    And both tickets belong to the same tenant
    When the matcher scans for the oldest compatible open ticket
    Then it marks both tickets as matched
    And it creates one challenge with match source "queue"
    And the new challenge state is "pending_acceptance"
    And both players receive an accept or start prompt

  Scenario: Human match becomes ready after both sides accept
    Given a queue-created challenge is in state "pending_acceptance"
    And both players accept before the acceptance deadline
    When the acceptance transaction completes
    Then the engine freezes one shared question snapshot
    And the challenge state becomes "ready"
    And both players receive the ready or start message

  Scenario: Human acceptance timeout requeues the willing player once
    Given a queue-created challenge is in state "pending_acceptance"
    And only one player accepts before the deadline
    When the acceptance window expires
    Then the non-accepting side is cancelled
    And the accepting side may be requeued once
    And the engine does not immediately convert that match into AI fallback

  Scenario: Search timeout creates an AI fallback opponent
    Given I have a matchmaking ticket in state "searching"
    And no human opponent is found before the wait timeout
    When the background sweeper processes my expired searching ticket
    Then it creates a challenge with match source "ai_fallback"
    And it skips the second acceptance step
    And it freezes the shared question snapshot immediately
    And the challenge state becomes "ready"
    And I receive ready or start info for the AI opponent

  Scenario: Cancel matchmaking before pairing
    Given I have a matchmaking ticket in state "searching"
    When I send "/challenge cancel"
    Then the engine closes my open searching ticket
    And I stop receiving matchmaking status for that ticket

  Scenario: Prevent duplicate search tickets per user
    Given I already have an open matchmaking ticket for the same tenant
    When I send "/challenge" again for the same topic
    Then the engine resumes the existing ticket instead of creating another one
    And the response shows the existing search status
```

## Planned next-slice state sketch

Planned next-slice states add queue and acceptance control before play:

```text
/challenge
  -> searching
  -> pending_acceptance (human pair only)
  -> ready

searching
  -> ready (AI fallback after wait timeout)
```

Planned rules:

- human-human queue pair uses `pending_acceptance`
- AI fallback skips `pending_acceptance`
- invite-code join still goes straight from `waiting` to `ready`

## Machine model

Think of this slice as three layers:

1. command router
2. challenge store
3. challenge state

### 1. Command router

The engine receives a chat message and routes it like this:

```text
ProcessMessage
  -> parse command
  -> /challenge
  -> handleChallengeCommand
     -> block if quiz currently owns the conversation
     -> if "invite", create challenge
     -> if "<code>", join challenge
     -> otherwise show help
```

This means the command handler is orchestration code.
It decides which lower-level store operation to call.
It does not itself hold challenge data.

### 2. Challenge store

The runtime uses the `ChallengeStore` interface:

```text
CreateInviteChallenge(...)
JoinChallenge(...)
GetChallenge(...)
```

That gives two interchangeable implementations:

- `MemoryChallengeStore`
  In-memory map. Good for tests and lightweight local runs.
- `PostgresChallengeStore`
  Persistent row storage in PostgreSQL. Used by the real server path.

This is the same idea as:

- TypeScript: interface + two classes
- Python: protocol/ABC + two implementations

The engine talks to the interface, not to one concrete store type.

### 3. Challenge state

The slice currently has one simple transition:

```text
create invite -> waiting
waiting + valid second player join -> ready
```

And three rejected transitions:

```text
waiting + creator joins own code -> ErrChallengeSelfJoin
ready + anyone tries to join again -> ErrChallengeNotJoinable
missing code -> ErrChallengeNotFound
```

## Memory model

`MemoryChallengeStore` is not a new product-level KV system.
It is just a tiny in-process map used as a fake store.

Conceptually:

```text
map[challengeCode] => *Challenge
```

Example:

```text
"ABC123" => {
  creator_id: "user1",
  opponent_id: "",
  topic_name: "Linear Equations",
  state: "waiting"
}
```

Why the mutex exists:

- Go maps are not safe for concurrent reads and writes
- the bot runtime may touch the same store from multiple goroutines
- `sync.RWMutex` is a readers/writer lock around that shared map

Mental model:

- `RLock`
  many readers allowed
- `Lock`
  one writer at a time

TypeScript analogy:

If multiple workers could mutate the same shared `Map`, you would need a lock around it.
Go makes that explicit.

## Postgres model

`PostgresChallengeStore` is the real persistence path.

On create:

1. resolve the external chat user id to the internal `users.id`
2. generate a code
3. insert a row into `challenges`
4. if the code collides with an existing unique code, retry

On join:

1. resolve the joiner's external chat user id to the internal `users.id`
2. start a transaction
3. `SELECT ... FOR UPDATE` the challenge row by `invite_code`
4. reject self-join or non-waiting rows
5. update `opponent_user_id`, `state`, and `ready_at`
6. commit

That `FOR UPDATE` lock is the important concurrency guard.
It prevents two joiners from successfully claiming the same waiting challenge at the same time.

## Why there is an "8 attempts" loop

Invite codes must be unique.

So create works like this:

```text
repeat up to N times:
  generate code
  try insert
  if unique collision:
    retry
  else:
    return success
```

`N` is currently `8`.

That is not business logic.
It is just a bounded retry loop so code generation cannot spin forever if a collision happens.

## Go syntax to TS/Python mapping

If Go syntax is the part slowing you down, use this mapping:

- `func (e *Engine) handleChallengeCommand(...)`
  Method on `Engine`.
  Roughly `class Engine { handleChallengeCommand(...) {} }`

- `(*Challenge, error)`
  Return two values: result plus error.
  Roughly `Promise<[challenge, err]>` or `tuple[Challenge | None, Error | None]`

- `if err != nil { ... }`
  Explicit error handling instead of exceptions

- `type ChallengeStore interface { ... }`
  Interface / protocol / abstract contract

- `sync.RWMutex`
  Readers/writer lock around shared mutable memory

- `*Challenge`
  Pointer to a challenge object
  Roughly "reference to an object" rather than a copied struct value

## What is still missing

This slice is only the invite-code doorway.

The plan still expects later work for:

- `/challenge` default find-opponent flow
- matchmaking queue
- AI fallback after queue timeout
- shared frozen question snapshot
- answer submission and grading
- result settlement and XP
- post-challenge review

So the right mental model is:

```text
current slice = lobby creation + lobby join
not yet = battle runtime
```
