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

3. Export environment variables:
```bash
export $(grep -v '^#' .env | grep -v '^$' | xargs)
```

## Test Execution

Run conversations through terminal-chat with `--memory` flag (in-memory state, no DB dependency for the store):

```bash
printf '<test_input>\n' | timeout 30 go run ./cmd/terminal-chat/ --memory 2>&1
```

Each test pipes one or more messages and checks the output for expected patterns.

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
**Expect:** output is not empty, no error/panic

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
1. /learn (no args)         PASS
2. /learn valid topic       PASS
3. /learn invalid topic     PASS
4. /progress                PASS
5. /goal                    PASS
6. Teaching message         PASS
7. Unknown command          PASS

Result: 7/7 passed
```

If any test fails, show the expected vs actual output.
