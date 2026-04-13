#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if ! command -v just >/dev/null 2>&1; then
  echo "just is required to stop local dev"
  exit 1
fi

exec just stop-local
