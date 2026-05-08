---
name: github-deep-review
description: Deep review GitHub issues, pull requests, bug reports, and maintainer threads for pai-bot. Use when asked to review, dig in, find root cause, judge the best fix, check whether an issue is still real, or decide whether a refactor is worth doing.
argument-hint: [issue-or-pr-ref]
allowed-tools: Bash, Read, Grep, Glob
---

# GitHub Deep Review

Review with evidence first. Goal: understand the bug class, trace the real code path, judge the fix quality, and say what remains unproven.

## Start

Use `gh`, not browser URLs.

```bash
gh issue view <n> --json number,title,state,author,body,comments,labels,updatedAt,url
gh pr view <n> --json number,title,state,author,body,comments,reviews,files,commits,statusCheckRollup,mergeStateStatus,headRefName,headRepositoryOwner,url
gh pr diff <n> --patch
```

Before deciding, read local rules and repo context:

```bash
git status --short --branch
docs-list
```

Open only relevant docs from `docs-list`. For PRs, also inspect touched files, adjacent owners, and tests. If behavior depends on a dependency or platform contract, check the current upstream docs/source before assuming.

## Review Contract

Always answer:

- Ref: issue/PR number and affected surface.
- Bug: what behavior is broken or being changed.
- Cause: root cause with code path, or exact missing evidence.
- Fix: whether the proposed/current fix is the best available shape.
- Refactor: whether a larger refactor improves correctness or only adds risk.
- Proof: tests, CI, live repro, docs/source, or current branch behavior.
- Risk: remaining uncertainty or coverage gap.

## Code Reading Depth

Follow the real path:

- entrypoint -> validation/parsing -> routing/dispatch -> owner module -> shared helper -> persistence/network/runtime boundary
- config/docs -> runtime usage -> doctor/migration/fix path
- channel/provider owner -> shared core only when the bug crosses owners
- tests around touched surface plus adjacent regression tests

Prefer current code and executable proof over issue comments. Treat stale comments, old CI, and old release behavior as hints until rechecked.

## Fix Bar

Good fixes:

- live at the ownership boundary where the bug belongs
- preserve public behavior unless retiring it is the point
- add the smallest meaningful regression test
- avoid hidden migrations, semantic sentinels, broad special cases, or provider IDs in generic core
- update docs/changelog when user-visible behavior changes
- fail clearly and repair through established doctor/migration paths when they exist

Call out symptom-level fixes. Recommend a refactor only when it makes the invariant clearer or reduces the bug class without widening risk too much.

## PR Review Shape

Lead with findings. Each finding needs file/line/symbol reference and a concrete failure mode. Avoid vague "consider" comments.

If no blocking issue:

- say no blocking correctness issues found
- list strongest proof checked
- name residual risks/test gaps
- answer whether the design is the best available shape

Do not approve, comment, close, merge, push, or land unless explicitly asked.

## Issue Review Shape

1. Reconstruct reporter scenario and affected version/surface.
2. Check whether current branch already fixes it.
3. Reproduce or create minimal proof when feasible.
4. Identify cause and best fix when clear.
5. If solved already, only comment/close when asked; include proof and canonical commit/PR if known.

If reproduction is blocked, say exactly what is missing.

## Output Template

```text
Ref: #123 / PR #456
Surface: <runtime/CLI/provider/channel/docs>

Bug: <one or two sentences>
Cause: <code path + confidence>
Best fix: <what should change and why>
Refactor: <yes/no, specific shape>
Proof: <tests/live/CI/source/dependency docs>
Risk: <remaining uncertainty>
```

Keep it concise. Do not skip cause/fix/refactor/proof.
