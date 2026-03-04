#!/usr/bin/env bash
set -euo pipefail

SERVER="${DEPLOY_HOST:-}"
USER="${DEPLOY_USER:-ubuntu}"
APP_DIR="${DEPLOY_DIR:-/opt/pai-bot}"
SSH_KEY="${DEPLOY_KEY:-}"

if [[ -z "$SERVER" ]]; then
    echo "Error: DEPLOY_HOST is not set"
    echo ""
    echo "Usage:"
    echo "  DEPLOY_HOST=<ip> ./scripts/deploy.sh"
    echo ""
    echo "Options (env vars):"
    echo "  DEPLOY_HOST  — Server IP or hostname (required)"
    echo "  DEPLOY_USER  — SSH user (default: ubuntu)"
    echo "  DEPLOY_DIR   — App directory on server (default: /opt/pai-bot)"
    echo "  DEPLOY_KEY   — Path to SSH private key (optional)"
    exit 1
fi

SSH_OPTS="-o StrictHostKeyChecking=accept-new"
if [[ -n "$SSH_KEY" ]]; then
    SSH_OPTS="$SSH_OPTS -i $SSH_KEY"
fi

echo "=== Deploying P&AI Bot to $USER@$SERVER:$APP_DIR ==="

# shellcheck disable=SC2087
ssh $SSH_OPTS "$USER@$SERVER" << REMOTE
set -euo pipefail
cd $APP_DIR

echo "--- Pulling latest code ---"
git pull origin main

echo "--- Building app ---"
docker compose -f docker-compose.yml -f docker-compose.prod.yml build app

echo "--- Restarting services ---"
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

echo "--- Waiting for app to start ---"
sleep 5

echo "--- Container status ---"
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
REMOTE

echo ""
echo "--- Health check ---"
HEALTH=$(curl -sf --max-time 10 "http://$SERVER/healthz" 2>&1) && {
    echo "OK: $HEALTH"
} || {
    echo "WARNING: Health check failed (app may still be starting)"
    echo "  Try: curl http://$SERVER/healthz"
}

echo ""
echo "=== Deploy complete ==="
