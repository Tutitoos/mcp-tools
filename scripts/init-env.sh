#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="$HOME/mcp-custom-data"

cat > "$REPO_DIR/.env" <<EOF_ENV
HOST_HOME=$HOME
HOST_UID=$(id -u)
HOST_GID=$(id -g)

MCP_CUSTOM_ROOT=$REPO_DIR
MCP_CUSTOM_DATA=$DATA_DIR

CODEBASE_MEMORY_MCP_IMAGE=mcp-custom/codebase-memory-mcp:latest

MEM0_MCP_IMAGE=mcp-custom/mem0-mcp:latest
HEADROOM_IMAGE=mcp-custom/headroom-mcp:latest
MEM0_SRC_PATH=$HOME/containers/mem0/mem0-src
MEM0_USER_ID=$(whoami)
EOF_ENV

mkdir -p "$DATA_DIR/codebase-memory-mcp/cache"
mkdir -p "$DATA_DIR/codebase-memory-mcp/config"

mkdir -p "$DATA_DIR/mem0/history"
mkdir -p "$DATA_DIR/mem0/uv-cache"
mkdir -p "$DATA_DIR/mem0/config"

mkdir -p "$DATA_DIR/headroom/cache"
mkdir -p "$DATA_DIR/headroom/config"
mkdir -p "$DATA_DIR/headroom/share"

echo "OK: generado $REPO_DIR/.env"
echo "OK: data en $DATA_DIR"
