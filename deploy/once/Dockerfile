FROM golang:1.26-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/pai-server ./cmd/server \
    && CGO_ENABLED=0 go build -o /out/pai-seed ./cmd/seed

FROM node:24-alpine AS admin-builder
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@10.31.0 --activate
COPY admin-spa/package.json admin-spa/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY admin-spa/ .
RUN pnpm build

FROM admin-builder
ARG TARGETARCH
RUN apk add --no-cache ca-certificates curl nginx postgresql17 postgresql17-client redis su-exec tzdata \
    && arch="${TARGETARCH:-$(uname -m)}" \
    && case "$arch" in amd64|x86_64) goose_arch=x86_64 ;; arm64|aarch64) goose_arch=arm64 ;; *) exit 1 ;; esac \
    && curl -fsSLO "https://github.com/pressly/goose/releases/download/v3.26.0/goose_linux_${goose_arch}" \
    && curl -fsSLO https://github.com/pressly/goose/releases/download/v3.26.0/checksums.txt \
    && grep "goose_linux_${goose_arch}$" checksums.txt | sha256sum -c - \
    && install -m 0755 "goose_linux_${goose_arch}" /usr/local/bin/goose \
    && rm "goose_linux_${goose_arch}" checksums.txt \
    && mkdir -p /usr/share/nginx/html \
    && cp -R /app/dist/. /usr/share/nginx/html/ \
    && rm -rf /app /root/.cache /root/.local/share/pnpm

COPY --from=go-builder /out/pai-server /usr/local/bin/pai-server
COPY --from=go-builder /out/pai-seed /usr/local/bin/pai-seed
COPY migrations /app/migrations
COPY oss /app/oss
COPY deploy/once/nginx.conf /etc/nginx/http.d/default.conf
COPY deploy/once/entrypoint.sh /usr/local/bin/pai-once-entrypoint

RUN chmod +x /usr/local/bin/pai-once-entrypoint \
    && mkdir -p /storage \
    && chown -R postgres:postgres /storage

ENV LEARN_SERVER_HOST=127.0.0.1 \
    LEARN_SERVER_PORT=8080 \
    LEARN_DATABASE_URL=postgres://pai:pai@127.0.0.1:5432/pai?sslmode=disable \
    LEARN_CACHE_URL=redis://127.0.0.1:6379 \
    LEARN_CURRICULUM_PATH=/app/oss \
    LEARN_DEV_MODE=true \
    LEARN_LOG_FORMAT=text

EXPOSE 80
VOLUME ["/storage"]
HEALTHCHECK --interval=5s --timeout=3s --start-period=30s --retries=12 \
    CMD curl -fsS http://127.0.0.1/up || exit 1
ENTRYPOINT ["/usr/local/bin/pai-once-entrypoint"]
