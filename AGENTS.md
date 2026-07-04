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

If the MCP tools are missing, ask the user to run `/mcp list` and `/mcp test mcp_tools_headroom`. Do not invent a CLI fallback.

For compression requests, pass the exact user-provided content to `headroom_compress`. Do not expand, replicate, or modify the input unless the user asks.

## Headroom OMP tool names

In OMP, the Headroom MCP server exposes tools with namespaced callable names:

- `mcp__mcp_tools_headroom_compress`
- `mcp__mcp_tools_headroom_retrieve`
- `mcp__mcp_tools_headroom_stats`

Do not look for bare `headroom_compress` in the model tool inventory. `/mcp test mcp_tools_headroom` shows bare server tool names, but OMP injects callable tools as `mcp__<server>_<tool>`.

For Headroom compression requests, call `mcp__mcp_tools_headroom_compress` with the exact user-provided content.

## OMP MCP tool discovery rule

OMP may expose MCP tools behind `search_tool_bm25` instead of loading every MCP tool directly.

When the user asks for a Headroom task and the callable Headroom tools are not directly visible, first use tool discovery with a query like:

- `headroom compress text logs reduce tokens`
- `headroom retrieve compressed content hash`
- `headroom stats compression savings`

After discovery activates the Headroom tools, call the matching Headroom MCP tool.

Do not claim Headroom is unavailable just because the MCP tool is not initially visible. First try tool discovery.

## OMP MCP discovery workflow

OMP v16.3.5 may expose MCP servers as discoverable tools instead of loading every MCP tool directly into the initial tool inventory.

When a user asks for a task handled by an MCP server and the corresponding `mcp__...` tool is not initially visible, do not say the tool is unavailable.

First use `search_tool_bm25` with a query describing the needed capability.

Examples:
- Headroom compression: `headroom compress text logs reduce tokens`
- Headroom retrieve: `headroom retrieve compressed content hash`
- Headroom stats: `headroom stats compression savings`
- codebase-memory architecture: `codebase memory architecture repository graph`
- codebase-memory search: `codebase memory search code symbols`
- mem0 memory search: `mem0 search memories persistent context`
- mem0 add memory: `mem0 add memory remember preference`

After discovery activates the matching MCP tool, call the activated tool.

Do not fall back to bash, Docker, Python, or host binaries for normal MCP tasks unless explicitly debugging MCP setup.
