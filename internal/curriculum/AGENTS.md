# CURRICULUM LOADER

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Loads OSS curriculum YAML, normalizes topic data, and evaluates prerequisites.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| YAML schema/types | `types.go`, `doc.go` |
| Loader behavior | `loader.go`, `loader_test.go` |
| Topic unlock prerequisites | `prerequisites.go`, `prerequisites_test.go` |
| Content mirror | `oss/` |
| Agent consumers | `internal/agent/context_loader.go`, `internal/agent/topic_unlock.go` |

## CONVENTIONS

- Schema changes require fixture/test updates and consumer checks.
- Loader errors identify source path/topic for content fixes.
- KSSM names stay stable; prompts cite these paths.
- Use fixtures that look like real KSSM algebra topics.

## ANTI-PATTERNS

- No ad-hoc curriculum structs in agent code.
- No silent skipping of malformed assessments.
- Do not assume `oss/` content paths are writable.

## NOTES

- Prerequisite logic feeds topic unlock and quiz routing.
