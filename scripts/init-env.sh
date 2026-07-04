#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="$HOME/mcp-custom-data"

cat > "$REPO_DIR/.env" <<EOF
HOST_HOME=$HOME
HOST_UID=$(id -u)
HOST_GID=$(id -g)

MCP_CUSTOM_ROOT=$REPO_DIR
MCP_CUSTOM_DATA=$DATA_DIR

CODEBASE_MEMORY_MCP_IMAGE=mcp-custom/codebase-memory-mcp:latest
EOF

mkdir -p "$DATA_DIR/codebase-memory-mcp/cache"
mkdir -p "$DATA_DIR/codebase-memory-mcp/config"

echo "OK: generado $REPO_DIR/.env"
echo "OK: data en $DATA_DIR"
