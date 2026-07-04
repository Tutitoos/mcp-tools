#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="$HOME/mcp-tools-data"

cat > "$REPO_DIR/.env" <<EOF_ENV
HOST_HOME=$HOME
HOST_UID=$(id -u)
HOST_GID=$(id -g)

MCP_TOOLS_ROOT=$REPO_DIR
MCP_TOOLS_DATA=$DATA_DIR

MCP_TOOLS_CODEBASE_MEMORY_IMAGE=mcp-tools/codebase-memory:latest

MCP_TOOLS_MEM0_IMAGE=mcp-tools/mem0:latest
MCP_TOOLS_HEADROOM_IMAGE=mcp-tools/headroom:latest
MEM0_SRC_PATH=$HOME/containers/mem0/mem0-src
MEM0_USER_ID=$(whoami)
EOF_ENV

mkdir -p "$DATA_DIR/codebase-memory/cache"
mkdir -p "$DATA_DIR/codebase-memory/config"

mkdir -p "$DATA_DIR/mem0/history"
mkdir -p "$DATA_DIR/mem0/uv-cache"
mkdir -p "$DATA_DIR/mem0/config"

mkdir -p "$DATA_DIR/headroom/cache"
mkdir -p "$DATA_DIR/headroom/config"
mkdir -p "$DATA_DIR/headroom/share"

echo "OK: generado $REPO_DIR/.env"
echo "OK: data en $DATA_DIR"
