# mcp-tools agent instructions

This repository contains custom Dockerized MCP servers and operational skills.

Before working with local code repositories, read:

- `skills/codebase-memory/SKILL.md`

Use `mcp_tools_codebase_memory` for codebase navigation, indexing, architecture analysis, code search, symbol tracing, and repository understanding.

Important rules:

- Always use the Docker wrapper.
- Do not call the host `codebase-memory-mcp` binary directly.
- The active wrapper is `$HOME/.local/bin/mcp-tools-codebase-memory-docker`.
- The MCP runtime uses Docker exec into the persistent container `mcp-tools-codebase-memory`.
- Persistent data lives under `$HOME/mcp-tools-data/codebase-memory`.

## Headroom MCP usage

For Headroom tasks, always use MCP tools first:

- `headroom_compress`
- `headroom_retrieve`
- `headroom_stats`

Do not inspect the CLI, Docker container, or Python package for normal usage. Do not create larger synthetic examples unless explicitly requested. Use the exact content provided by the user.
