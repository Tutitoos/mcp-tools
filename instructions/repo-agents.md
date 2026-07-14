# mcp-tools agent instructions

This repository contains an installer + registry for MCP servers and operational skills.

**Tool routing (which MCP for which intent) is defined ONCE in `RULES.md`** — installed globally in every supported client. It is not duplicated here; if it is not loaded in your session, read `RULES.md` at the repo root first.

## Serena-first for this repo

`mcp_tools_serena` is the default for ANY operation on a named symbol (LSP-accurate, no string/comment false positives). First code-operation call of every session: `activate_project("/home/tutitoos/mcp-tools")`. Then:

- body of X / "cómo funciona X" → `find_symbol(name_path_pattern: "X", include_body: true)`
- who uses X / "quién llama a X" → `find_referencing_symbols`; declaration → `find_declaration`
- file outline → `get_symbols_overview(relative_path: "internal/.../file.go")`
- semantic edit / rename → `replace_symbol_body` / `rename_symbol`

Native `Read` is only for raw config, docs, `.env`, logs, JSON dumps, and non-LSP languages. Never to "see how function X works" — about to `Read` a `.go`/`.ts`/`.py`/`.rs`/`.java` file for that? Stop and use serena. If serena errors or shows "not connected" → escalation list in `RULES.md`.

## Per-MCP skills

Read as needed before repo work: `skills/serena/SKILL.md`, `skills/tokensave/SKILL.md`, `skills/codebase-memory/SKILL.md`, `skills/mem0/SKILL.md`. Note: the upstream mem0 `search_memories`/`get_memories` bug is patched by mcp-tools post-install (`internal/tools/mem0_patch.go`); details in `skill://mem0` §Known state.

## Repo facts

- MCP servers run as host binaries (not Docker): `~/.local/bin/codebase-memory-mcp`, `~/.local/bin/mem0-launcher` (sources `.env.mem0`), `~/.local/bin/serena` (install: web panel `/tools` → serena, or `uv tool install -p 3.13 serena-agent`), `~/.cargo/bin/tokensave`.
- Old `mcp-tools-*-docker` wrappers were removed. If a client still references them, re-run mcp-config from the web panel (`/settings` → "Re-run mcp-config" = `POST /api/mcp-config/sync`). The `mcp-tools` CLI has no `mcp-config` subcommand.
- Persistent data: `$HOME/mcp-tools-data/` (subdirs `mem0`, `ollama`, plus `state.json`). Per-project serena state: `<project>/.serena/`.

## OMP tool discovery

OMP may expose MCP servers as discoverable tools instead of preloading them. If an `mcp__...` tool is not visible, do NOT declare it unavailable: call `search_tool_bm25` with the capability as query (e.g. `serena find symbol activate project`, `tokensave context code exploration`, `mem0 add memory remember preference`), then call the activated tool. Do not fall back to bash, Docker, Python, or host binaries for normal MCP tasks unless explicitly debugging MCP setup.
