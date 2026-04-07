#!/bin/bash
# deploy-remote.sh — Runs ON the server via SSH.
# Expects env vars: ECR_TOKEN, REGISTRY, TAG
# Expects DEPLOY_DIR env var or defaults to /opt/pai-bot.
# No AWS CLI required — only Docker + docker compose.
set -euo pipefail

cd "${DEPLOY_DIR:-/opt/pai-bot}"

echo "--- Disabling host nginx if present ---"
sudo systemctl stop nginx 2>/dev/null || true
sudo systemctl disable nginx 2>/dev/null || true

echo "--- ECR login ---"
echo "$ECR_TOKEN" | docker login --username AWS --password-stdin "$REGISTRY"

echo "--- Recording previous image for rollback ---"
PREV_APP=$(docker inspect --format='{{.Config.Image}}' "$(docker compose -f docker-compose.yml -f docker-compose.prod.yml ps -q app 2>/dev/null)" 2>/dev/null || echo "")
PREV_ADMIN=$(docker inspect --format='{{.Config.Image}}' "$(docker compose -f docker-compose.yml -f docker-compose.prod.yml ps -q admin 2>/dev/null)" 2>/dev/null || echo "")
echo "Previous app: ${PREV_APP:-none}"
echo "Previous admin: ${PREV_ADMIN:-none}"

echo "--- Pulling images ---"
docker pull "$REGISTRY/pai-bot/app:$TAG"
docker pull "$REGISTRY/pai-bot/admin:$TAG"
docker tag "$REGISTRY/pai-bot/app:$TAG" pai-bot:latest
docker tag "$REGISTRY/pai-bot/admin:$TAG" pai-admin:latest

echo "--- Ensuring infra services ---"
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d postgres dragonfly nats
sleep 3

echo "--- Running migrations ---"
DB_URL=$(grep LEARN_DATABASE_URL .env | cut -d= -f2-)
docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile tools run --rm goose \
  go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 \
  -dir /app/migrations postgres "$DB_URL" up -allow-missing

echo "--- Rolling out ---"
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

echo "--- Health check: app container ---"
APP_CONTAINER=$(docker compose -f docker-compose.yml -f docker-compose.prod.yml ps -q app)
APP_HEALTH=""
for i in $(seq 1 30); do
  APP_HEALTH=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$APP_CONTAINER")
  if [ "$APP_HEALTH" = "healthy" ]; then
    echo "App healthy after attempt $i"
    break
  fi
  echo "Attempt $i/30: $APP_HEALTH"
  sleep 2
done

if [ "$APP_HEALTH" != "healthy" ]; then
  echo "ERROR: app not healthy — rolling back"
  if [ -n "$PREV_APP" ]; then
    docker tag "$PREV_APP" pai-bot:latest
    docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d app
    echo "Rolled back app to $PREV_APP"
  fi
  docker compose -f docker-compose.yml -f docker-compose.prod.yml logs --tail=50 app
  exit 1
fi

echo "--- Health check: app endpoint ---"
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T app curl -sf http://localhost:8080/healthz > /dev/null

echo "--- Health check: Caddy ingress ---"
if curl -sf --max-time 10 http://localhost/healthz > /dev/null 2>&1; then
  echo "Caddy route OK"
else
  echo "WARNING: Caddy route check failed (may be expected with HTTPS-only domain)"
fi

echo "--- Health check: admin container ---"
ADMIN_CONTAINER=$(docker compose -f docker-compose.yml -f docker-compose.prod.yml ps -q admin 2>/dev/null || echo "")
if [ -n "$ADMIN_CONTAINER" ]; then
  ADMIN_STATUS=$(docker inspect --format '{{.State.Status}}' "$ADMIN_CONTAINER" 2>/dev/null || echo "unknown")
  if [ "$ADMIN_STATUS" = "running" ]; then
    echo "Admin container running"
  else
    echo "WARNING: Admin container status: $ADMIN_STATUS"
  fi
fi

echo "--- Smoke test: bot commands ---"
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"
SMOKE_PASS=0
SMOKE_FAIL=0

smoke() {
  local name="$1" input="$2" expect="$3"
  output=$($COMPOSE exec -T -e LEARN_DEV_MODE=true app \
    sh -c "printf '$input\n' | timeout 30 /pai-terminal-chat --memory" 2>&1)
  if echo "$output" | grep -qiE "$expect"; then
    echo "  PASS: $name"
    SMOKE_PASS=$((SMOKE_PASS + 1))
  else
    echo "  FAIL: $name (expected: $expect)"
    echo "    got: $(echo "$output" | grep "P&AI>" | head -2)"
    SMOKE_FAIL=$((SMOKE_FAIL + 1))
  fi
}

smoke "/learn usage"         "/learn"                    "/learn"
smoke "/progress"            "/progress"                 "Progress|XP"
smoke "/create_group"        "/create_group Test Deploy" "Test Deploy"
smoke "unknown cmd"          "/foobar"                   "diketahui|Unknown"

echo "  Smoke: $SMOKE_PASS passed, $SMOKE_FAIL failed"
if [ "$SMOKE_FAIL" -gt 0 ]; then
  echo "WARNING: $SMOKE_FAIL smoke test(s) failed — deploy succeeded but bot may have issues"
fi

echo ""
echo "Deploy successful (image: $TAG)"
docker image prune -f
$COMPOSE ps
