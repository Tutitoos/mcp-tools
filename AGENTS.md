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
