#!/bin/bash
# deploy-remote.sh — Runs ON the EC2 instance via SSM or SSH.
# Expects env vars: REGISTRY, TAG, REGION
set -euo pipefail

DEPLOY_DIR="/opt/pai-bot"
cd "$DEPLOY_DIR"

echo "--- Pulling secrets from AWS Secrets Manager ---"
SECRETS=$(aws secretsmanager get-secret-value \
  --secret-id pai-bot/production-env \
  --region "$REGION" \
  --query SecretString --output text)

echo "--- Writing .env ---"
PG_PASS=$(echo "$SECRETS" | jq -r .POSTGRES_PASSWORD)
printf '%s\n' \
  "LEARN_SERVER_PORT=8080" \
  "LEARN_DATABASE_URL=postgres://pai:${PG_PASS}@postgres:5432/pai?sslmode=disable" \
  "LEARN_DATABASE_MAX_CONNS=25" \
  "LEARN_CACHE_URL=redis://dragonfly:6379" \
  "LEARN_NATS_URL=nats://nats:4222" \
  "LEARN_TELEGRAM_BOT_TOKEN=$(echo "$SECRETS" | jq -r .LEARN_TELEGRAM_BOT_TOKEN)" \
  "PAI_AUTH_SECRET=$(echo "$SECRETS" | jq -r .PAI_AUTH_SECRET)" \
  "PAI_AUTH_GOOGLE_CLIENT_ID=$(echo "$SECRETS" | jq -r '.PAI_AUTH_GOOGLE_CLIENT_ID // empty')" \
  "PAI_AUTH_GOOGLE_CLIENT_SECRET=$(echo "$SECRETS" | jq -r '.PAI_AUTH_GOOGLE_CLIENT_SECRET // empty')" \
  "PAI_AUTH_GOOGLE_ALLOWED_DOMAIN=$(echo "$SECRETS" | jq -r '.PAI_AUTH_GOOGLE_ALLOWED_DOMAIN // empty')" \
  "LEARN_AI_OPENAI_API_KEY=$(echo "$SECRETS" | jq -r .LEARN_AI_OPENAI_API_KEY)" \
  "LEARN_AI_ANTHROPIC_API_KEY=$(echo "$SECRETS" | jq -r '.LEARN_AI_ANTHROPIC_API_KEY // empty')" \
  "LEARN_AI_DEEPSEEK_API_KEY=$(echo "$SECRETS" | jq -r '.LEARN_AI_DEEPSEEK_API_KEY // empty')" \
  "LEARN_AI_GOOGLE_API_KEY=$(echo "$SECRETS" | jq -r '.LEARN_AI_GOOGLE_API_KEY // empty')" \
  "LEARN_AI_OPENROUTER_API_KEY=$(echo "$SECRETS" | jq -r '.LEARN_AI_OPENROUTER_API_KEY // empty')" \
  "LEARN_AI_OLLAMA_ENABLED=false" \
  "LEARN_TENANT_MODE=single" \
  "LEARN_CURRICULUM_PATH=./oss" \
  "LEARN_DISABLE_MULTI_LANGUAGE=false" \
  "LEARN_RATING_PROMPT_EVERY_REPLIES=5" \
  "LEARN_AI_PERSONALIZED_NUDGES_ENABLED=true" \
  "LEARN_EMAIL_SMTP_ADDR=$(echo "$SECRETS" | jq -r '.LEARN_EMAIL_SMTP_ADDR // empty')" \
  "LEARN_EMAIL_SMTP_USERNAME=$(echo "$SECRETS" | jq -r '.LEARN_EMAIL_SMTP_USERNAME // empty')" \
  "LEARN_EMAIL_SMTP_PASSWORD=$(echo "$SECRETS" | jq -r '.LEARN_EMAIL_SMTP_PASSWORD // empty')" \
  "LEARN_EMAIL_FROM_ADDRESS=$(echo "$SECRETS" | jq -r '.LEARN_EMAIL_FROM_ADDRESS // empty')" \
  "LEARN_EMAIL_FROM_NAME=$(echo "$SECRETS" | jq -r '.LEARN_EMAIL_FROM_NAME // empty')" \
  "LEARN_LOG_LEVEL=info" \
  "LEARN_LOG_FORMAT=json" \
  "POSTGRES_PASSWORD=${PG_PASS}" \
  > .env

DOMAIN_VAL=$(echo "$SECRETS" | jq -r '.DOMAIN // empty')
if [ -n "$DOMAIN_VAL" ] && [ "$DOMAIN_VAL" != "localhost" ]; then
  echo "DOMAIN=${DOMAIN_VAL}" >> .env
  cp deploy/caddy/Caddyfile.https deploy/caddy/Caddyfile
else
  cp deploy/caddy/Caddyfile.http deploy/caddy/Caddyfile
fi

echo "--- Disabling host nginx if present ---"
sudo systemctl stop nginx 2>/dev/null || true
sudo systemctl disable nginx 2>/dev/null || true

echo "--- Pulling images from ECR ---"
aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$REGISTRY"
docker pull "$REGISTRY/pai-bot/app:$TAG"
docker pull "$REGISTRY/pai-bot/admin:$TAG"
docker tag "$REGISTRY/pai-bot/app:$TAG" pai-bot:latest
docker tag "$REGISTRY/pai-bot/admin:$TAG" pai-admin:latest

echo "--- Ensuring infra services ---"
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d postgres dragonfly nats
sleep 3

echo "--- Running migrations ---"
docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile tools run --rm goose \
  go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 \
  -dir /app/migrations postgres \
  "postgres://pai:${PG_PASS}@postgres:5432/pai?sslmode=disable" \
  up -allow-missing

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
