#!/usr/bin/env bash
set -euo pipefail
REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RULES_SRC="$REPO_DIR/RULES.md"
MARKER_START="<!-- mcp-tools:start -->"
MARKER_END="<!-- mcp-tools:end -->"

if [ ! -f "$RULES_SRC" ]; then
  echo "ERROR: $RULES_SRC no existe" >&2
  exit 1
fi

# --- OMP: symlink como rule file ---
OMP_RULES="$HOME/.omp/rules"
mkdir -p "$OMP_RULES"
ln -snf "$RULES_SRC" "$OMP_RULES/mcp-tools.md"

# --- Claude Code: @import en ~/.claude/CLAUDE.md ---
CLAUDE_MD="$HOME/.claude/CLAUDE.md"
mkdir -p "$(dirname "$CLAUDE_MD")"
touch "$CLAUDE_MD"
IMPORT_LINE="@$RULES_SRC"
grep -Fxq "$IMPORT_LINE" "$CLAUDE_MD" || printf '%s\n' "$IMPORT_LINE" >> "$CLAUDE_MD"

# --- OpenCode: bloque marcado en ~/.config/opencode/AGENTS.md ---
OPENCODE_AGENTS="$HOME/.config/opencode/AGENTS.md"
mkdir -p "$(dirname "$OPENCODE_AGENTS")"
touch "$OPENCODE_AGENTS"
# borra bloque anterior si existe
sed -i "\|^${MARKER_START}\$|,\|^${MARKER_END}\$|d" "$OPENCODE_AGENTS"
# añade bloque nuevo con el CONTENIDO literal de RULES.md
{
  printf '%s\n' "$MARKER_START"
  cat "$RULES_SRC"
  printf '%s\n' "$MARKER_END"
} >> "$OPENCODE_AGENTS"

# verify
for f in "$OMP_RULES/mcp-tools.md" "$CLAUDE_MD" "$OPENCODE_AGENTS"; do
  [ -r "$f" ] && echo "OK $f" || { echo "FAIL $f" >&2; exit 1; }
done
echo "Done. Reload/restart your MCP client to pick up RULES."
