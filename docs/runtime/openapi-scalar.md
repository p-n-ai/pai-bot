---
title: "OpenAPI And Scalar Docs"
summary: "Current OpenAPI JSON and Scalar docs surface for pai-bot backend routes."
read_when:
  - You are changing backend HTTP routes or API response shapes.
  - You are updating internal/apidocs or the /openapi.json and /docs surfaces.
  - You need to verify API docs behavior locally.
---

# OpenAPI And Scalar Docs

The backend exposes API documentation from the Go server:

| Route | Purpose |
|---|---|
| `GET /openapi.json` | OpenAPI 3.1 JSON document. |
| `GET /docs` | Scalar HTML shell that loads `/openapi.json`. |

## Code ownership

| Path | Role |
|---|---|
| `internal/apidocs/document.go` | OpenAPI document model and JSON rendering. |
| `internal/apidocs/routes.go` | Route metadata. |
| `internal/apidocs/schema.go` | Schema helpers. |
| `cmd/server/main.go` | Mounts `/openapi.json` and `/docs`. |
| `cmd/server/main_test.go` | Verifies OpenAPI and Scalar routes. |

## Update rules

- When backend API routes change, update `internal/apidocs`.
- Keep docs generation in stdlib Go; do not add a docs framework unless needed.
- Verify with `go test ./cmd/server ./internal/apidocs`.
- If route docs lag code, trust `cmd/server/main.go` first and then fix `internal/apidocs`.
