#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

action="${1:-dev}"
once_version="${ONCE_VERSION:-v0.1.10}"
once_namespace="${ONCE_NAMESPACE:-pai-dev}"
once_host="${ONCE_HOST:-localhost}"
registry_name="${ONCE_REGISTRY_CONTAINER:-pai-once-registry}"
image="${ONCE_IMAGE:-127.0.0.1:5000/pai-bot:dev}"

install_once() {
  local os arch asset tmp expected
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
  case "$os" in
    linux|darwin) ;;
    *) echo "ONCE supports Linux and macOS; found $os" >&2; exit 1 ;;
  esac

  asset="once-${os}-${arch}"
  tmp="$(mktemp -d)"
  curl -fsSL "https://github.com/basecamp/once/releases/download/${once_version}/${asset}" -o "$tmp/once"
  curl -fsSL "https://github.com/basecamp/once/releases/download/${once_version}/checksums.txt" -o "$tmp/checksums.txt"
  expected="$(awk -v asset="$asset" '$2 == asset { print $1 }' "$tmp/checksums.txt")"
  if [ -z "$expected" ]; then
    echo "checksum for $asset was not found" >&2
    exit 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s  %s\n' "$expected" "$tmp/once" | sha256sum -c -
  else
    [ "$(shasum -a 256 "$tmp/once" | awk '{print $1}')" = "$expected" ]
  fi
  mkdir -p "$HOME/.local/bin"
  install -m 0755 "$tmp/once" "$HOME/.local/bin/once"
  rm -rf "$tmp"
  export PATH="$HOME/.local/bin:$PATH"
}

command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }
docker info >/dev/null 2>&1 || { echo "the Docker daemon is not running" >&2; exit 1; }
command -v once >/dev/null 2>&1 || install_once
once_bin="$(command -v once)"

case "$action" in
  stop)
    exec "$once_bin" --namespace "$once_namespace" stop "$once_host"
    ;;
  remove)
    exec "$once_bin" --namespace "$once_namespace" remove "$once_host"
    ;;
  dev) ;;
  *)
    echo "usage: $0 [dev|stop|remove]" >&2
    exit 2
    ;;
esac

if [ "${ONCE_SKIP_BUILD:-false}" != "true" ]; then
  if ! docker inspect "$registry_name" >/dev/null 2>&1; then
    docker run -d \
      --name "$registry_name" \
      --restart unless-stopped \
      -p 127.0.0.1:5000:5000 \
      registry:2 >/dev/null
  elif [ "$(docker inspect -f '{{.State.Running}}' "$registry_name")" != "true" ]; then
    docker start "$registry_name" >/dev/null
  fi

  docker build -f deploy/once/Dockerfile -t "$image" .
  docker push "$image"
fi

once_args=(
  --namespace "$once_namespace"
  --host "$once_host"
  --disable-tls
  --auto-update=false
  --env "PAI_ONCE_SEED_DEMO=${PAI_ONCE_SEED_DEMO:-true}"
)

if [ -f .env ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%$'\r'}"
    case "$line" in
      ""|\#*) continue ;;
    esac
    key="${line%%=*}"
    value="${line#*=}"
    if [[ "$value" == \"*\" && "$value" == *\" ]] || [[ "$value" == \'*\' && "$value" == *\' ]]; then
      value="${value:1:${#value}-2}"
    fi
    case "$key" in
      LEARN_SERVER_HOST|LEARN_SERVER_PORT|LEARN_DATABASE_URL|LEARN_CACHE_URL|LEARN_CURRICULUM_PATH)
        continue
        ;;
      PAI_AUTH_SECRET)
        [ "$value" = "change-me-in-production" ] && continue
        ;;
      LEARN_*|PAI_AUTH_*|PAI_CONFIG_ENCRYPTION_KEY) ;;
      *) continue ;;
    esac
    [ -n "$value" ] && once_args+=(--env "$key=$value")
  done <.env
fi

if [ -n "${ONCE_CPUS:-}" ]; then
  once_args+=(--cpus "$ONCE_CPUS")
fi
if [ -n "${ONCE_MEMORY_MB:-}" ]; then
  once_args+=(--memory "$ONCE_MEMORY_MB")
fi

if "$once_bin" --namespace "$once_namespace" list | grep -Fq "$once_host"; then
  "$once_bin" --namespace "$once_namespace" update "$once_host" --image "$image" "${once_args[@]:2}"
else
  "$once_bin" --namespace "$once_namespace" deploy "$image" "${once_args[@]:2}"
fi

printf 'P&AI is running at http://%s\n' "$once_host"
