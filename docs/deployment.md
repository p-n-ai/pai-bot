# Deployment

P&AI Bot supports two deployment models today: local development and single-server production (Docker Compose). Kubernetes (Helm) is planned.

## Local Development

```bash
just go       # Start backend: infra + migrations + Go server
just next     # Start backend + Next.js admin panel
just stop     # Stop everything
```

See [setup.md](setup.md) for prerequisites and first-time setup.

## Single-Server Production (Docker Compose)

Recommended for small deployments (up to ~500 students). Uses `docker-compose.yml` with `docker-compose.prod.yml` override.

### Prerequisites

- Linux server (e.g., AWS t3.medium, 2 vCPU / 4 GB RAM)
- Docker + Docker Compose v2
- Domain name pointing to the server (for HTTPS via Caddy)

### Setup

1. Clone the repo on the server:

```bash
git clone https://github.com/p-n-ai/pai-bot.git /opt/pai-bot
cd /opt/pai-bot
```

2. Configure environment:

```bash
cp .env.example .env
# Edit .env — set production values:
#   LEARN_TELEGRAM_BOT_TOKEN=<your-token>
#   LEARN_AI_OPENAI_API_KEY=<your-key>
#   PAI_AUTH_SECRET=<random-32-char-string>
#   LEARN_DEV_MODE=false
```

3. Build and start:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

4. Verify:

```bash
curl http://localhost:8080/healthz    # → {"status":"ok"}
docker compose logs -f app            # Watch logs
```

### Services

| Service | Image | Port | Resource Limit |
|---------|-------|------|----------------|
| **app** | `pai-bot:latest` | 8080 (internal) | 512 MB |
| **admin** | `pai-admin:latest` | 3000 (internal) | 256 MB |
| **caddy** | `caddy:2-alpine` | 80, 443 | 128 MB |
| **postgres** | `postgres:17-alpine` | 5432 (internal) | 512 MB |
| **dragonfly** | `docker.dragonflydb.io/dragonflydb/dragonfly` | 6379 (internal) | 300 MB |
| **nats** | `nats:2.10-alpine` | 4222 (internal) | 128 MB |

### Reverse Proxy (Caddy)

Caddy handles TLS termination and routing:

```
/api/*     → app:8080
/healthz   → app:8080
/*         → admin:3000
```

For HTTPS, use `deploy/caddy/Caddyfile.https` which reads the `DOMAIN` env var. The default `Caddyfile` listens on port 80 via `CADDY_SITE_ADDRESS`:

```env
# For HTTPS (use Caddyfile.https):
DOMAIN=learn.yourschool.edu.my

# For HTTP only (default Caddyfile):
CADDY_SITE_ADDRESS=:80
```

### Automated Deployment

The `scripts/deploy-remote.sh` script handles remote deployment:

```bash
# On the server
./scripts/deploy-remote.sh
```

It performs:
1. Pull latest images from container registry
2. Start infrastructure (Postgres, Dragonfly, NATS)
3. Run database migrations
4. Start application with health checks
5. Smoke tests (health endpoint, bot commands)
6. Automatic rollback on failure

### Updating

```bash
cd /opt/pai-bot
git pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

After updating, run migrations explicitly before restarting the app:

```bash
just migrate
# or via Docker:
docker compose run --rm app goose -dir /migrations postgres "$LEARN_DATABASE_URL" up
```

## Docker Images

### Application (`deploy/docker/Dockerfile`)

Multi-stage build:
- **Builder:** Go 1.22 compiles the server binary
- **Runtime:** Alpine 3.20 (~25 MB final image)
- Includes: `pai-server`, `pai-terminal-chat`, `pai-terminal-nudge`, `pai-seed`, migrations, OSS curriculum

### Admin Panel (`deploy/docker/Dockerfile.admin`)

Multi-stage build:
- **Builder:** Node.js with pnpm, builds Next.js app
- **Runtime:** Node.js Alpine, serves on `:3000`

## Kubernetes (Planned)

A Helm chart for Kubernetes deployment is planned (`P-W5D23-2` in the development timeline) but not yet available. The target structure is `deploy/helm/pai/` with Deployment, StatefulSet, ConfigMap, Secret, Service, and Ingress resources.

### Health Probes

The Go server exposes:
- `/healthz` — Liveness probe (always returns 200 if the process is running)
- `/readyz` — Readiness probe (currently returns 200 unconditionally; dependency checks planned)

Graceful shutdown on `SIGTERM`.

## Monitoring

### Logs

Structured JSON logs via `slog`:

```bash
just logs                    # Tail app logs (docker compose logs -f app)
docker compose logs -f app   # App logs only
```

### Analytics

```bash
just analytics    # Quick metrics: DAU, messages/session, AI latency, tokens by model
```

The admin panel at `/dashboard/ai-usage` shows:
- Tenant token budget and usage
- Daily token trend
- Per-student average tokens

### API

`GET /api/admin/analytics/report` returns comprehensive metrics (42-day DAU, retention, nudge response, AI usage) in a single payload. Requires admin authentication.

## Backup

### Database

```bash
# Dump
docker compose exec postgres pg_dump -U pai pai > backup.sql

# Restore
docker compose exec -T postgres psql -U pai pai < backup.sql
```

### Volumes

Docker Compose uses named volumes for persistence:
- `postgres-data` — Database files
- `dragonfly-data` — Cache data

Back up the Docker volumes or use PostgreSQL replication for high availability.
