ARG CODEX_CLI_VERSION=0.145.0

# Package the native Codex binary without carrying Node into the runtime image.
FROM node:22-alpine AS codex-cli
ARG CODEX_CLI_VERSION
RUN npm install --global --omit=dev --no-audit --no-fund "@openai/codex@${CODEX_CLI_VERSION}" \
    && codex --version \
    && codex_path="$(find /usr/local/lib/node_modules/@openai/codex -type f -path '*/bin/codex' | head -n 1)" \
    && test -n "${codex_path}" \
    && cp "${codex_path}" /codex \
    && chmod 0755 /codex

# Build the Go binaries.
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /pai-server ./cmd/server \
    && CGO_ENABLED=0 go build -o /pai-terminal-chat ./cmd/terminal-chat \
    && CGO_ENABLED=0 go build -o /pai-terminal-nudge ./cmd/terminal-nudge \
    && CGO_ENABLED=0 go build -o /pai-seed ./cmd/seed \
    && CGO_ENABLED=0 go build -o /pai-validate-production-secrets ./cmd/validate-production-secrets

# Stage 2: Final image (~25MB)
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl \
    && curl -fsSL https://github.com/pressly/goose/releases/download/v3.24.3/goose_linux_x86_64 \
       -o /usr/local/bin/goose && chmod +x /usr/local/bin/goose
COPY --from=builder /pai-server /pai-server
COPY --from=builder /pai-terminal-chat /pai-terminal-chat
COPY --from=builder /pai-terminal-nudge /pai-terminal-nudge
COPY --from=builder /pai-seed /pai-seed
COPY --from=builder /pai-validate-production-secrets /pai-validate-production-secrets
COPY oss /oss
COPY skills /skills
COPY migrations /migrations
COPY --from=codex-cli /codex /usr/local/bin/codex
RUN codex --version
EXPOSE 8080
ENTRYPOINT ["/pai-server"]
