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

## Kubernetes (Helm)

A Helm chart is available at `deploy/helm/pai/`. It deploys the full stack: Go app, Next.js admin, PostgreSQL, Dragonfly, NATS, with database migrations, health probes, and ingress routing.

### Prerequisites

- Kubernetes cluster (any: EKS, GKE, AKS, k3s, minikube, k3d)
- `helm` v3.12+
- `kubectl` configured for your cluster
- Container images pushed to a registry (or imported locally for k3d/minikube)

### Quick Start

```bash
# 1. Build and push images (or import into local cluster)
docker build -f deploy/docker/Dockerfile -t ghcr.io/p-n-ai/pai-bot:latest .
docker build -f deploy/docker/Dockerfile.admin -t ghcr.io/p-n-ai/pai-admin:latest .

# 2. Install with your values
helm install pai deploy/helm/pai \
  --set secrets.telegramBotToken=YOUR_TOKEN \
  --set secrets.ai.openaiApiKey=YOUR_KEY \
  --set secrets.authSecret=$(openssl rand -hex 16) \
  --set ingress.enabled=true \
  --set ingress.host=learn.yourschool.edu.my \
  --set ingress.className=nginx

# 3. Check status
kubectl get pods
helm status pai
```

### Local Testing with k3d

```bash
# Create a local cluster
k3d cluster create pai-local --port "9090:80@loadbalancer"

# Build and import images
docker build -f deploy/docker/Dockerfile -t pai-bot:local .
docker build -f deploy/docker/Dockerfile.admin -t pai-admin:local .
k3d image import pai-bot:local pai-admin:local -c pai-local

# Install in dev mode (no Telegram/AI keys needed)
helm install pai-local deploy/helm/pai \
  --set app.image.repository=pai-bot \
  --set app.image.tag=local \
  --set app.image.pullPolicy=Never \
  --set admin.image.repository=pai-admin \
  --set admin.image.tag=local \
  --set admin.image.pullPolicy=Never \
  --set config.devMode=true \
  --set ingress.enabled=true \
  --set ingress.host=localhost \
  --set ingress.className=traefik

# Visit http://localhost:9090
# Login: platform-admin@example.com / demo-password

# Cleanup
k3d cluster delete pai-local
```

### What Gets Deployed

| Resource | Type | Purpose |
|----------|------|---------|
| `pai-app` | Deployment | Go backend (port 8080) |
| `pai-admin` | Deployment | Next.js admin panel (port 3000) |
| `pai-postgres` | StatefulSet + PVC | PostgreSQL 17 database |
| `pai-dragonfly` | StatefulSet + PVC | Dragonfly cache (Redis-compatible) |
| `pai-nats` | Deployment | NATS with JetStream |
| `pai` | ConfigMap | Non-secret environment variables |
| `pai` | Secret | API keys, auth secret, DB password |
| `pai` | Ingress | Routes `/api/*` to app, `/` to admin |

Database migrations run automatically as init containers on the app pod before the server starts.

### Configuration

Override values via `--set` flags or a custom values file:

```bash
helm install pai deploy/helm/pai -f my-values.yaml
```

Key values in `values.yaml`:

| Value | Default | Description |
|-------|---------|-------------|
| `config.tenantMode` | `single` | `single` or `multi` tenant |
| `config.devMode` | `false` | Skip Telegram/AI requirements |
| `secrets.authSecret` | `change-me-in-production` | JWT signing secret |
| `secrets.telegramBotToken` | `""` | Telegram bot token |
| `secrets.ai.openaiApiKey` | `""` | OpenAI API key |
| `postgres.enabled` | `true` | Use built-in PostgreSQL (set `false` for external DB) |
| `dragonfly.enabled` | `true` | Use built-in Dragonfly cache |
| `nats.enabled` | `true` | Use built-in NATS |
| `admin.enabled` | `true` | Deploy admin panel |
| `ingress.enabled` | `false` | Create ingress resource |
| `ingress.host` | `pai.example.com` | Ingress hostname |

For external databases, disable the built-in StatefulSet and set the connection URL in config:

```bash
helm install pai deploy/helm/pai \
  --set postgres.enabled=false \
  --set app.env.LEARN_DATABASE_URL=postgres://user:pass@your-rds:5432/pai
```

### Upgrading

```bash
helm upgrade pai deploy/helm/pai --set ...
```

Migrations run automatically on upgrade via init containers.

### Uninstalling

```bash
helm uninstall pai
# PVCs are retained — delete manually if you want to wipe data:
kubectl delete pvc -l app.kubernetes.io/instance=pai
```

### Health Probes

The Go server exposes:
- `/healthz` — Liveness probe (always returns 200 if the process is running)
- `/readyz` — Readiness probe (returns 200 when the server is ready to accept traffic)

Graceful shutdown on `SIGTERM` with a 15-second termination grace period.

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
