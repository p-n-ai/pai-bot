#!/bin/bash
# deploy-remote.sh — Runs ON the server via SSH.
# Expects env vars: ECR_TOKEN, GHCR_TOKEN, GHCR_USER, POSTGRES_IMAGE, REGISTRY,
# TAG, APP_DIGEST, ADMIN_DIGEST.
# Expects DEPLOY_DIR env var or defaults to /opt/pai-bot.
# No AWS CLI required — only Docker + docker compose.
set -eEuo pipefail

cd "${DEPLOY_DIR:-/opt/pai-bot}"

BACKUP_TMP=""
ROLLBACK_ARMED=false

cleanup() {
  if [ -n "$BACKUP_TMP" ]; then
    rm -f "$BACKUP_TMP"
  fi
  docker logout "$REGISTRY" >/dev/null 2>&1 || true
  docker logout ghcr.io >/dev/null 2>&1 || true
}

rollback_release() {
  trap - ERR
  set +e
  echo "--- Rolling back application images and environment ---"
  if [ -z "${PREV_APP_ID:-}" ] || [ -z "${PREV_ADMIN_ID:-}" ]; then
    echo "Rollback unavailable: no complete previous application image pair"
    return
  fi

  local rollback_ok=true
  if [ -f .env.rollback ]; then
    restore_tmp=$(mktemp .env.restore.XXXXXX)
    cp .env.rollback "$restore_tmp" || rollback_ok=false
    chmod 600 "$restore_tmp" || rollback_ok=false
    mv "$restore_tmp" .env || rollback_ok=false
    if [ "$rollback_ok" = "true" ]; then
      echo "Restored previous environment"
    fi
  fi
  docker tag "$PREV_APP_ID" pai-bot:latest || rollback_ok=false
  docker tag "$PREV_ADMIN_ID" pai-admin:latest || rollback_ok=false
  docker compose -f docker-compose.yml -f docker-compose.prod.yml \
    up -d --force-recreate app admin || rollback_ok=false
  if [ "$rollback_ok" = "true" ]; then
    echo "Rollback restored app $PREV_APP_ID and admin $PREV_ADMIN_ID"
  else
    echo "ERROR: rollback did not fully restore the previous release"
  fi
}

fail_release() {
  trap - ERR
  echo "ERROR: $1"
  rollback_release
  docker compose -f docker-compose.yml -f docker-compose.prod.yml \
    logs --tail=50 app admin
  exit 1
}

unexpected_failure() {
  local status=$?
  if [ "$BASH_SUBSHELL" -gt 0 ]; then
    return "$status"
  fi
  trap - ERR
  echo "ERROR: deployment command failed with status $status"
  if [ "$ROLLBACK_ARMED" = "true" ]; then
    rollback_release
  fi
  exit "$status"
}

trap cleanup EXIT
trap unexpected_failure ERR

echo "--- Disabling host nginx if present ---"
sudo systemctl stop nginx 2>/dev/null || true
sudo systemctl disable nginx 2>/dev/null || true

echo "--- ECR login ---"
echo "$ECR_TOKEN" | docker login --username AWS --password-stdin "$REGISTRY"

echo "--- Recording previous images for rollback ---"
PREV_APP_ID=$(docker inspect --format='{{.Image}}' "$(docker compose -f docker-compose.yml -f docker-compose.prod.yml ps -q app 2>/dev/null)" 2>/dev/null || echo "")
PREV_ADMIN_ID=$(docker inspect --format='{{.Image}}' "$(docker compose -f docker-compose.yml -f docker-compose.prod.yml ps -q admin 2>/dev/null)" 2>/dev/null || echo "")
echo "Previous app image ID: ${PREV_APP_ID:-none}"
echo "Previous admin image ID: ${PREV_ADMIN_ID:-none}"

echo "--- Reclaiming unused Docker storage ---"
docker image prune -af
docker builder prune -af

echo "--- Pulling candidate images ---"
docker pull "$REGISTRY/pai-bot/app@$APP_DIGEST"
docker pull "$REGISTRY/pai-bot/admin@$ADMIN_DIGEST"
EXPECTED_APP_ID=$(docker image inspect --format '{{.Id}}' "$REGISTRY/pai-bot/app@$APP_DIGEST")
EXPECTED_ADMIN_ID=$(docker image inspect --format '{{.Id}}' "$REGISTRY/pai-bot/admin@$ADMIN_DIGEST")
ROLLBACK_ARMED=true
docker tag "$EXPECTED_APP_ID" pai-bot:latest
docker tag "$EXPECTED_ADMIN_ID" pai-admin:latest

echo "--- Validating production secrets ---"
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm config-check

echo "--- Ensuring infra services ---"
printf '%s' "$GHCR_TOKEN" | docker login --username "$GHCR_USER" --password-stdin ghcr.io
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull postgres dragonfly
docker logout ghcr.io >/dev/null 2>&1 || true
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d postgres dragonfly
sleep 3

echo "--- Running migrations ---"
DB_URL=$(grep LEARN_DATABASE_URL .env | cut -d= -f2-)
echo "--- Creating pre-migration database backup ---"
umask 077
BACKUP_DIR="${DEPLOY_DIR:-/opt/pai-bot}/backups"
mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"
BACKUP_TMP=$(mktemp "$BACKUP_DIR/.pai-backup.XXXXXX")
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T postgres \
  pg_dump "$DB_URL" --format=custom > "$BACKUP_TMP"
test -s "$BACKUP_TMP"
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T postgres \
  pg_restore --list < "$BACKUP_TMP" > /dev/null
BACKUP_PATH="$BACKUP_DIR/pai-$(date -u +%Y%m%dT%H%M%SZ)-${TAG:0:12}.dump"
mv "$BACKUP_TMP" "$BACKUP_PATH"
BACKUP_TMP=""
echo "Database backup: $BACKUP_PATH"

echo "--- Checking provider-qualified user identities ---"
if ! USERS_TABLE_EXISTS=$(docker compose -f docker-compose.yml -f docker-compose.prod.yml \
  exec -T postgres psql "$DB_URL" -Atqc \
  "SELECT to_regclass('public.users') IS NOT NULL"); then
  fail_release "could not inspect the production schema"
fi
if [ "$USERS_TABLE_EXISTS" = "t" ]; then
  docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T postgres \
    psql "$DB_URL" -f - < scripts/preflight-conversation-identities.sql
elif [ "$USERS_TABLE_EXISTS" = "f" ]; then
  echo "Fresh database: identity preflight will be enforced by the migration itself"
else
  fail_release "unexpected users-table probe result: $USERS_TABLE_EXISTS"
fi
docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile tools run --rm goose \
  go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 \
  -dir /app/migrations postgres "$DB_URL" up -allow-missing

echo "--- Rolling out ---"
# --remove-orphans drops the pre-existing nats container; its volume needs a one-time `docker volume rm pai-bot_nats-data`
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --remove-orphans

echo "--- Health check: app container ---"
APP_CONTAINER=$(docker compose -f docker-compose.yml -f docker-compose.prod.yml ps -q app)
APP_HEALTH=""
for i in $(seq 1 30); do
  APP_HEALTH=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$APP_CONTAINER" 2>/dev/null || echo "missing")
  if [ "$APP_HEALTH" = "healthy" ]; then
    echo "App healthy after attempt $i"
    break
  fi
  echo "Attempt $i/30: $APP_HEALTH"
  sleep 2
done

if [ "$APP_HEALTH" != "healthy" ]; then
  fail_release "app did not become healthy"
fi

RUNNING_APP_ID=$(docker inspect --format '{{.Image}}' "$APP_CONTAINER")
if [ "$RUNNING_APP_ID" != "$EXPECTED_APP_ID" ]; then
  fail_release "app image $RUNNING_APP_ID does not match candidate $EXPECTED_APP_ID"
fi

COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

echo "--- Health check: app endpoint ---"
if ! $COMPOSE exec -T app curl -sf http://localhost:8080/healthz > /dev/null; then
  fail_release "app endpoint health check failed"
fi

echo "--- Health check: application and AI provider status ---"
STATUS_EXPECTED='{"status":"ok","components":[{"id":"application","status":"operational"},{"id":"ai_provider","status":"operational"}]}'
STATUS_RESPONSE=$($COMPOSE exec -T app curl -fsS --max-time 30 http://localhost:8080/health/status) || STATUS_RESPONSE=""
if [ "$STATUS_RESPONSE" != "$STATUS_EXPECTED" ]; then
  fail_release "AI response health check failed"
fi

echo "--- Health check: Caddy ingress ---"
if curl -sf --max-time 10 http://localhost/healthz > /dev/null 2>&1; then
  echo "Caddy route OK"
else
  echo "WARNING: Caddy route check failed (may be expected with HTTPS-only domain)"
fi

echo "--- Health check: admin container ---"
ADMIN_CONTAINER=$(docker compose -f docker-compose.yml -f docker-compose.prod.yml ps -q admin 2>/dev/null || echo "")
if [ -z "$ADMIN_CONTAINER" ]; then
  fail_release "admin container is missing"
fi
ADMIN_STATUS=$(docker inspect --format '{{.State.Status}}' "$ADMIN_CONTAINER" 2>/dev/null || echo "unknown")
if [ "$ADMIN_STATUS" != "running" ]; then
  fail_release "admin container status is $ADMIN_STATUS"
fi
RUNNING_ADMIN_ID=$(docker inspect --format '{{.Image}}' "$ADMIN_CONTAINER")
if [ "$RUNNING_ADMIN_ID" != "$EXPECTED_ADMIN_ID" ]; then
  fail_release "admin image $RUNNING_ADMIN_ID does not match candidate $EXPECTED_ADMIN_ID"
fi
if ! docker compose -f docker-compose.yml -f docker-compose.prod.yml \
  exec -T admin wget -qO- http://localhost:3000/ > /dev/null; then
  fail_release "admin HTTP check failed"
fi
echo "Admin container is running the candidate image"

echo "--- Smoke test: bot commands ---"
SMOKE_PASS=0
SMOKE_FAIL=0

smoke() {
  local name="$1" input="$2" expect="$3"
  output=$($COMPOSE exec -T -e LEARN_DEV_MODE=true app \
    sh -c "printf '$input\n' | timeout 30 /pai-terminal-chat --memory" 2>&1 || true)
  if echo "$output" | grep -qiE "$expect"; then
    echo "  PASS: $name"
    SMOKE_PASS=$((SMOKE_PASS + 1))
  else
    echo "  FAIL: $name (expected: $expect)"
    echo "    got: $(echo "$output" | grep "P&AI>" | head -2)"
    SMOKE_FAIL=$((SMOKE_FAIL + 1))
  fi
}

smoke "/learn usage" "/learn" "/learn"
smoke "/progress" "/progress" "Progress|XP"
smoke "/help" "/help" "available commands|arahan yang tersedia|可用的指令"
smoke "unknown cmd" "/foobar" "diketahui|Unknown"

echo "  Smoke: $SMOKE_PASS passed, $SMOKE_FAIL failed"
if [ "$SMOKE_FAIL" -gt 0 ]; then
  fail_release "$SMOKE_FAIL bot smoke test(s) failed"
fi

echo "--- Recording successfully deployed PostgreSQL aliases ---"
case "$POSTGRES_IMAGE" in
  ghcr.io/p-n-ai/pai-postgres@sha256:*) ;;
  *) echo "Invalid PostgreSQL deployment image: $POSTGRES_IMAGE" >&2; exit 1 ;;
esac
postgres_repository=${POSTGRES_IMAGE%@*}
postgres_release_image=$(
  docker compose -f docker-compose.yml -f docker-compose.prod.yml \
    config --variables |
    awk '$1 == "POSTGRES_IMAGE" { print $3; exit }'
)
case "$postgres_release_image" in
  "$postgres_repository":*) ;;
  *) echo "Invalid PostgreSQL release image: $postgres_release_image" >&2; exit 1 ;;
esac

printf '%s' "$GHCR_TOKEN" | docker login --username "$GHCR_USER" --password-stdin ghcr.io
postgres_image_id=$(docker image inspect --format '{{.Id}}' "$POSTGRES_IMAGE")
for alias_image in \
  "$postgres_repository:deployed" \
  "$postgres_release_image"
do
  docker tag "$postgres_image_id" "$alias_image"
  docker push "$alias_image"
done
docker logout ghcr.io >/dev/null 2>&1 || true

ROLLBACK_ARMED=false
trap - ERR
docker image prune -f || echo "WARNING: post-deploy image cleanup failed"
$COMPOSE ps || echo "WARNING: post-deploy status report failed"
echo ""
echo "Deploy successful (image: $TAG)"
