# Stage 1: Build Go binary
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /pai-server ./cmd/server

# Stage 2: Final image (~25MB)
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /pai-server /pai-server
EXPOSE 8080
ENTRYPOINT ["/pai-server"]
