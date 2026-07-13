# mcp-tools agent instructions

This repository contains an installer + registry for MCP servers and operational skills.

## Serena-first code exploration (READ THIS FIRST)

**`mcp_tools_serena` is the default for any code operation on a named symbol** (function, class, method, struct, type, constant, field). Use it for: reading a function body, listing who calls it, finding its declaration, getting a file's symbol outline, editing a symbol's body. It is LSP-accurate, returns ~60% fewer tokens than `rtk grep`, and ZERO false positives (no string/comment noise).

Workflow (EVERY session):
1. `activate_project("/absolute/path")` once per project. For this repo: `activate_project("/home/tutitoos/mcp-tools")`.
2. Then use `find_symbol` / `find_referencing_symbols` / `find_declaration` / `get_symbols_overview` / `find_symbol(include_body: true)` as needed.

Default tool mapping (overrides "use native Read" instinct):
- "show me function X" / "cómo funciona X" / "muéstrame el cuerpo" → `find_symbol(name_path_pattern: "X", include_body: true)`
- "where is X used" / "quién llama a X" / "references of X" → `find_referencing_symbols(name_path_pattern: "X")`
- "where is X defined" / "declaración de X" → `find_declaration(name_path_pattern: "X")`
- "outline of file.go" / "symbols in file" → `get_symbols_overview(relative_path: "internal/.../file.go")`
- "rename X to Y" / "replace body of X" → `replace_symbol_body` / `rename_symbol` (LSP-accurate)

Native `Read` is only for: raw config, docs, `.env`, logs, JSON dumps, and files in non-LSP languages. **Never** use native `Read` to "see how function X works" — that is serena's job.

## Other MCP servers

Before working with local code repositories, read:

- `skills/codebase-memory/SKILL.md`
- `skills/serena/SKILL.md`
- `skills/tokensave/SKILL.md`
- `skills/mem0/SKILL.md`

Use `mcp_tools_serena` for symbol-level code operations (DEFAULT — see block above).

Use `tokensave` (`tokensave_context`) for natural-language exploration in a `tokensave init`'d project ("how does X work" open-ended).

Use `mcp_tools_codebase_memory` for cross-repo architecture, ADR, community detection, dependency graphs.

Use `mcp_tools_mem0` for persistent cross-session memory (facts, preferences, decisions). Always call `search_memories` before `add_memory` to avoid duplicates. NOTE: `search_memories` and `get_memories` are BROKEN upstream (lib mem0 API change) — see RULES.md "Known bugs" before relying on them.

Important rules:

- The MCP servers run as host binaries (not Docker). `codebase-memory-mcp` lives at `~/.local/bin/codebase-memory-mcp`; `mem0-mcp-selfhosted` runs behind the `~/.local/bin/mem0-launcher` wrapper (sources `.env.mem0`); `serena` lives at `~/.local/bin/serena` (installed from the mcp-tools web panel, `/tools` → serena → install, or `uv tool install -p 3.13 serena-agent`).
- Do not spawn old `mcp-tools-*-docker` wrappers — they were removed. If a client still references them, re-run mcp-config from the web panel (`/settings` → "Re-run mcp-config", i.e. `POST /api/mcp-config/sync`). The `mcp-tools` CLI no longer has a `mcp-config` subcommand.
- Persistent data lives under `$HOME/mcp-tools-data/` (subdirs per MCP: `mem0`, `ollama`, plus `state.json`). Per-project serena state lives at `<project>/.serena/`.
- NEVER fall back to native `Grep`/`Read`/`find`/`bash grep` for repo-wide code search — use serena (named symbol), tokensave (open question), or codebase-memory (cross-repo).
- NEVER use `rtk grep` to find references of a named symbol — it matches strings/comments, not symbol identity. Use serena.
- NEVER write local `notes.md`/scratchpad files to persist facts across sessions — that's what `mcp_tools_mem0` is for.

## OMP MCP discovery workflow

OMP v16.3.5 may expose MCP servers as discoverable tools instead of loading every MCP tool directly into the initial tool inventory.

When a user asks for a task handled by an MCP server and the corresponding `mcp__...` tool is not initially visible, do not say the tool is unavailable.

First use `search_tool_bm25` with a query describing the needed capability.

Examples (serena is a top priority — the user wants it used heavily):
- serena find symbol: `serena find symbol activate project`
- serena find references: `serena find references symbol callers`
- serena get symbols overview: `serena get symbols overview file outline`
- serena replace symbol body: `serena replace symbol body semantic edit`
- codebase-memory architecture: `codebase memory architecture repository graph`
- codebase-memory search: `codebase memory search code symbols`
- tokensave context: `tokensave context code exploration`
- mem0 memory search: `mem0 search memories persistent context`
- mem0 add memory: `mem0 add memory remember preference`

After discovery activates the matching MCP tool, call the activated tool.

Do not fall back to bash, Docker, Python, or host binaries for normal MCP tasks unless explicitly debugging MCP setup.
