# mcp-custom agent instructions

This repository contains custom Dockerized MCP servers and operational skills.

Before working with local code repositories, read:

- `skills/codebase-memory-mcp/SKILL.md`

Use `codebase_memory_mcp` for codebase navigation, indexing, architecture analysis, code search, symbol tracing, and repository understanding.

Important rules:

- Always use the Docker wrapper.
- Do not call the host `codebase-memory-mcp` binary directly.
- The active wrapper is `$HOME/.local/bin/codebase-memory-mcp-docker`.
- The MCP runtime uses Docker exec into the persistent container `mcp-custom-codebase-memory-mcp`.
- Persistent data lives under `$HOME/mcp-custom-data/codebase-memory-mcp`.

## Headroom MCP hard rule

When the user asks to use Headroom, compress text, compress logs, reduce context, save tokens, retrieve compressed content, or inspect Headroom stats, use the MCP tools directly:

- `headroom_compress`
- `headroom_retrieve`
- `headroom_stats`

Do not use shell commands, Docker commands, the host `headroom` binary, Python imports, package internals, or synthetic expanded test cases for normal Headroom tasks.

Forbidden unless explicitly debugging MCP setup:
- `which headroom`
- `headroom --help`
- `docker exec ... headroom`
- `docker exec ... python`
- `python -c "from headroom import ..."`
- reading Headroom source files

If the MCP tools are missing, ask the user to run `/mcp list` and `/mcp test headroom`. Do not invent a CLI fallback.

For compression requests, pass the exact user-provided content to `headroom_compress`. Do not expand, replicate, or modify the input unless the user asks.

## Headroom OMP tool names

In OMP, the Headroom MCP server exposes tools with namespaced callable names:

- `mcp__headroom_headroom_compress`
- `mcp__headroom_headroom_retrieve`
- `mcp__headroom_headroom_stats`

Do not look for bare `headroom_compress` in the model tool inventory. `/mcp test headroom` shows bare server tool names, but OMP injects callable tools as `mcp__<server>_<tool>`.

For Headroom compression requests, call `mcp__headroom_headroom_compress` with the exact user-provided content.
