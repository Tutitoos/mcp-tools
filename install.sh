#!/usr/bin/env bash
set -euo pipefail
REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALLER="$REPO_DIR/scripts/installer"

if ! command -v bun >/dev/null 2>&1; then
  cat >&2 <<'MSG'
ERROR: bun no está instalado.
Instálalo: curl -fsSL https://bun.sh/install | bash
Luego relanza: ./install.sh
MSG
  exit 1
fi

if [ ! -d "$INSTALLER/node_modules" ]; then
  echo "==> bootstrap installer deps (bun install)"
  (cd "$INSTALLER" && bun install --silent)
fi

exec bun "$INSTALLER/index.tsx" "$@"
