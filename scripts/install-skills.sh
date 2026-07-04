#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC="$REPO_DIR/skills"

SKILLS=(codebase-memory headroom)

TARGETS=(
  "$HOME/.claude/skills"          # Claude Code + OpenCode external + OMP compat
  "$HOME/.config/opencode/skills" # OpenCode global
  "$HOME/.omp/agent/skills"       # OMP user-level
)

# stale entries from the previous naming (before rename to mcp-tools- prefix)
STALE=(
  "$HOME/.claude/skills/codebase-memory-mcp"
  "$HOME/.claude/skills/headroom-mcp"
  "$HOME/.config/opencode/skills/codebase-memory-mcp"
  "$HOME/.config/opencode/skills/headroom-mcp"
  "$HOME/.omp/agent/skills/codebase-memory-mcp"
  "$HOME/.omp/agent/skills/headroom-mcp"
)

echo "== cleaning stale skill dirs =="
for s in "${STALE[@]}"; do
  if [ -L "$s" ] || [ -d "$s" ]; then
    echo "  rm $s"
    rm -rf "$s"
  fi
done

echo "== installing symlinks =="
for t in "${TARGETS[@]}"; do
  mkdir -p "$t"
  for name in "${SKILLS[@]}"; do
    ln -snf "$SRC/$name" "$t/$name"
    echo "  $t/$name -> $SRC/$name"
  done
done

echo "== verify =="
for t in "${TARGETS[@]}"; do
  for name in "${SKILLS[@]}"; do
    f="$t/$name/SKILL.md"
    if [ -r "$f" ]; then
      echo "  OK $f"
    else
      echo "  FAIL $f" >&2
      exit 1
    fi
  done
done

echo
echo "Done. Reload / restart your MCP client (Claude Code, OpenCode, OMP) to pick up the skills."
