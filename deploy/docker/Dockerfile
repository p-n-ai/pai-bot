# Stage 1: Build Go binary
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /pai-server ./cmd/server
RUN CGO_ENABLED=0 go build -o /pai-terminal-chat ./cmd/terminal-chat
RUN CGO_ENABLED=0 go build -o /pai-terminal-nudge ./cmd/terminal-nudge
RUN CGO_ENABLED=0 go build -o /pai-seed ./cmd/seed

# Stage 2: Final image (~25MB)
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl \
    && curl -fsSL https://github.com/pressly/goose/releases/download/v3.24.3/goose_linux_x86_64 \
       -o /usr/local/bin/goose && chmod +x /usr/local/bin/goose
COPY --from=builder /pai-server /pai-server
COPY --from=builder /pai-terminal-chat /pai-terminal-chat
COPY --from=builder /pai-terminal-nudge /pai-terminal-nudge
COPY --from=builder /pai-seed /pai-seed
COPY --from=builder /app/oss /oss
COPY --from=builder /app/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/pai-server"]
