#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [ ! -f "$REPO_DIR/.env" ]; then
  "$REPO_DIR/scripts/init-env.sh"
fi

cd "$REPO_DIR"

docker compose up -d codebase_memory_mcp
