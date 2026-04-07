#!/bin/bash
# deploy-remote.sh — Runs ON the server via SSH.
# Expects env vars: ECR_TOKEN, REGISTRY, TAG
# No AWS CLI required on the server — only Docker + docker compose.
set -euo pipefail

DEPLOY_DIR="/opt/pai-bot"
cd "$DEPLOY_DIR"

echo "--- Disabling host nginx if present ---"
sudo systemctl stop nginx 2>/dev/null || true
sudo systemctl disable nginx 2>/dev/null || true

echo "--- ECR login ---"
echo "$ECR_TOKEN" | docker login --username AWS --password-stdin "$REGISTRY"

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

echo "--- Health check ---"
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
  echo "ERROR: app not healthy"
  docker compose -f docker-compose.yml -f docker-compose.prod.yml logs --tail=50 app
  exit 1
fi

docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T app curl -sf http://localhost:8080/healthz
echo ""
echo "Deploy successful (image: $TAG)"
docker image prune -f
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
