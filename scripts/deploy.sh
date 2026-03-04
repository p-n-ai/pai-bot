#!/usr/bin/env bash
set -euo pipefail

SERVER="${DEPLOY_HOST:-}"
USER="${DEPLOY_USER:-ubuntu}"
APP_DIR="${DEPLOY_DIR:-/opt/pai-bot}"

if [[ -z "$SERVER" ]]; then
    echo "Error: DEPLOY_HOST is not set"
    echo "Usage: DEPLOY_HOST=your-server ./scripts/deploy.sh"
    exit 1
fi

echo "=== Deploying P&AI Bot to $USER@$SERVER:$APP_DIR ==="

ssh "$USER@$SERVER" << REMOTE
set -euo pipefail
cd $APP_DIR
echo "--- Pulling latest code ---"
git pull origin main
echo "--- Building app ---"
docker compose build app
echo "--- Restarting app ---"
docker compose up -d app
echo "=== Deploy complete ==="
docker compose logs --tail=20 app
REMOTE
