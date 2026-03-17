---
name: test-bot
description: Agentic end-to-end testing of the bot via terminal-chat. Spins up docker compose, pipes test conversations, validates responses, and tears down. Use after implementing a feature to verify it works live.
argument-hint: [scenario]
allowed-tools: Bash, Read, Grep, Glob
---

# Agentic Bot Testing

Run end-to-end agentic tests against the live bot using terminal-chat with docker compose infrastructure.

## Setup

1. Start infrastructure (skip if already running):
```bash
docker compose up -d postgres dragonfly nats
```

2. Wait for postgres to be healthy:
```bash
until docker compose ps postgres --format '{{.Status}}' | grep -q healthy; do sleep 1; done
```

3. Export environment variables (always enable dev mode for testing):
```bash
export $(grep -v '^#' .env | grep -v '^$' | xargs) && export LEARN_DEV_MODE=true
```

## Test Execution

Run conversations through terminal-chat with `--memory` flag (in-memory state, no DB dependency for the store):

```bash
printf '<test_input>\n' | timeout 30 go run ./cmd/terminal-chat/ --memory 2>&1
```

Each test pipes one or more messages and checks the output for expected patterns.

### Important Lessons

- **Always start with `/clear` or `/dev-reset`** to ensure clean state when testing mastery/progress features
- **`--memory` flag** means each `go run` invocation starts fresh — mastery only accumulates within a single piped session
- **Mastery grading is async** — runs in a goroutine after AI responds. In piped sessions, there may not be enough time for the goroutine to complete before the next message is processed. For mastery/progress tests, include enough teaching turns (6+) in a single piped session
- **`/dev-reset`** (requires `LEARN_DEV_MODE=true`) fully clears mastery, XP, streaks, goals, and profile — use this to verify clean state
- **Unlock notifications are drained at the top of `ProcessMessage`** — they appear on the message *after* mastery crosses the threshold, on any message type (commands or chat)

## Scenario: $ARGUMENTS

If `$ARGUMENTS` is provided, test that specific feature/command. Otherwise, run the standard smoke test suite below.

## Standard Smoke Test Suite

Run these scenarios in order. For each one, pipe the input and verify the output contains expected strings.

### 1. Command: /learn (no args → usage)
```bash
printf '/learn\n' | timeout 15 go run ./cmd/terminal-chat/ --memory 2>&1
```
**Expect:** output contains `/learn` (usage hint)

### 2. Command: /learn with valid topic
```bash
printf '/learn persamaan linear\n' | timeout 15 go run ./cmd/terminal-chat/ --memory 2>&1
```
**Expect:** output contains topic name (e.g., "Persamaan Linear" or "Linear Equations")

### 3. Command: /learn with invalid topic
```bash
printf '/learn quantum physics\n' | timeout 15 go run ./cmd/terminal-chat/ --memory 2>&1
```
**Expect:** output contains "tidak dijumpai" or "not found"

### 4. Command: /progress
```bash
printf '/progress\n' | timeout 15 go run ./cmd/terminal-chat/ --memory 2>&1
```
**Expect:** output contains "Progress" or "XP", no error/panic

### 5. Command: /goal
```bash
printf '/goal kuasai algebra\n' | timeout 30 go run ./cmd/terminal-chat/ --memory 2>&1
```
**Expect:** output mentions goal or topic, no error/panic

### 6. Free-form teaching message
```bash
printf 'Ajar saya tentang persamaan linear\n' | timeout 30 go run ./cmd/terminal-chat/ --memory 2>&1
```
**Expect:** AI responds with teaching content (non-empty response after "P&AI>")

### 7. Unknown command
```bash
printf '/foobar\n' | timeout 15 go run ./cmd/terminal-chat/ --memory 2>&1
```
**Expect:** output contains "tidak diketahui" or "Unknown command"

### 8. Multi-turn teaching + mastery accumulation
```bash
printf '/learn persamaan linear\nPersamaan linear x + 3 = 7, tolak 3, x = 4\n2x + 4 = 10, tolak 4, 2x = 6, bahagi 2, x = 3\n3x - 9 = 0, tambah 9, 3x = 9, bahagi 3, x = 3\n5x + 10 = 25, tolak 10, 5x = 15, bahagi 5, x = 3\n4x - 8 = 12, tambah 8, 4x = 20, bahagi 4, x = 5\n/progress\n' | timeout 180 go run ./cmd/terminal-chat/ --memory 2>&1
```
**Expect:** `/progress` shows F1-06 with mastery > 0%, XP > 0, streak 1 day

### 9. /dev-reset hidden without dev mode
```bash
LEARN_DEV_MODE=false go run ./cmd/terminal-chat/ --memory <<< '/dev-reset' 2>&1
```
Note: This won't work with env prefix; use `export LEARN_DEV_MODE=false` before running.
**Expect:** output contains "tidak diketahui" or "Unknown command"

### 10. /dev-reset clears everything
```bash
printf '/learn persamaan linear\nPersamaan linear x+3=7, x=4\n2x+4=10, x=3\n3x-9=0, x=3\n5x+10=25, x=3\n4x-8=12, x=5\n/progress\n/dev-reset\n/progress\n' | timeout 180 go run ./cmd/terminal-chat/ --memory 2>&1
```
**Expect:** First `/progress` shows mastery/XP > 0. After `/dev-reset`, second `/progress` shows XP: 0 and no mastery.

## Validation

For each scenario:
1. Run the command
2. Check exit code (should be 0)
3. Check output contains expected string (case-insensitive)
4. Report PASS/FAIL with the actual output on failure

## Teardown

After all tests, stop infrastructure:
```bash
docker compose stop postgres dragonfly nats
```

## Report Format

```
Agentic Test Results
====================
 1. /learn (no args)           PASS
 2. /learn valid topic         PASS
 3. /learn invalid topic       PASS
 4. /progress                  PASS
 5. /goal                      PASS
 6. Teaching message           PASS
 7. Unknown command            PASS
 8. Multi-turn mastery         PASS
 9. /dev-reset hidden          PASS
10. /dev-reset clears all      PASS

Result: 10/10 passed
```

If any test fails, show the expected vs actual output.
