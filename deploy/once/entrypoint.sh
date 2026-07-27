#!/bin/sh
set -eu

export PGDATA="${PGDATA:-/storage/postgres}"
redis_data=/storage/redis

mkdir -p "$PGDATA" "$redis_data" /storage/whatsmeow /run/postgresql
chown -R postgres:postgres "$PGDATA" /run/postgresql
chown -R redis:redis "$redis_data"

if [ ! -s "$PGDATA/PG_VERSION" ]; then
  password_file="$(mktemp)"
  printf '%s\n' pai >"$password_file"
  chown postgres:postgres "$password_file"
  su-exec postgres initdb \
    --username=pai \
    --pwfile="$password_file" \
    --auth-host=scram-sha-256 \
    --auth-local=trust
  rm -f "$password_file"
fi

su-exec postgres pg_ctl -D "$PGDATA" -o "-h 127.0.0.1" -w start
if ! PGPASSWORD=pai psql -h 127.0.0.1 -U pai -d postgres -Atqc \
  "SELECT 1 FROM pg_database WHERE datname = 'pai'" | grep -q 1; then
  PGPASSWORD=pai createdb -h 127.0.0.1 -U pai pai
fi

su-exec redis redis-server \
  --bind 127.0.0.1 \
  --protected-mode yes \
  --dir "$redis_data" \
  --appendonly yes \
  --daemonize yes

export PAI_AUTH_SECRET="${PAI_AUTH_SECRET:-${SECRET_KEY_BASE:-change-me-in-production}}"
export PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL="${PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL:-admin@example.com}"
export PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD="${PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD:-demo-password}"
export LEARN_EMAIL_SMTP_ADDR="${LEARN_EMAIL_SMTP_ADDR:-${SMTP_ADDRESS:-}}"
export LEARN_EMAIL_SMTP_USERNAME="${LEARN_EMAIL_SMTP_USERNAME:-${SMTP_USERNAME:-}}"
export LEARN_EMAIL_SMTP_PASSWORD="${LEARN_EMAIL_SMTP_PASSWORD:-${SMTP_PASSWORD:-}}"
export LEARN_EMAIL_FROM_ADDRESS="${LEARN_EMAIL_FROM_ADDRESS:-${MAILER_FROM_ADDRESS:-}}"
export LEARN_WHATSAPP_MEOW_DB="${LEARN_WHATSAPP_MEOW_DB:-file:/storage/whatsmeow/whatsmeow.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)}"

goose -dir /app/migrations postgres "$LEARN_DATABASE_URL" up -allow-missing
if [ "${PAI_ONCE_SEED_DEMO:-false}" = "true" ]; then
  pai-seed
fi

shutdown() {
  nginx -s quit >/dev/null 2>&1 || true
  if [ -n "${server_pid:-}" ]; then
    kill "$server_pid" >/dev/null 2>&1 || true
  fi
  redis-cli -h 127.0.0.1 shutdown >/dev/null 2>&1 || true
  su-exec postgres pg_ctl -D "$PGDATA" -m fast -w stop >/dev/null 2>&1 || true
}
trap shutdown INT TERM EXIT

pai-server &
server_pid=$!

for _ in $(seq 1 60); do
  if curl -fsS http://127.0.0.1:8080/readyz >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$server_pid" >/dev/null 2>&1; then
    wait "$server_pid"
  fi
  sleep 1
done
curl -fsS http://127.0.0.1:8080/readyz >/dev/null

nginx -g "daemon off;" &
nginx_pid=$!

while kill -0 "$server_pid" >/dev/null 2>&1 && kill -0 "$nginx_pid" >/dev/null 2>&1; do
  sleep 1
done
exit 1
