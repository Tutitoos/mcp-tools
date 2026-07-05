# mcp-tools agent instructions

This repository contains an installer + registry for MCP servers and operational skills.

Before working with local code repositories, read:

- `skills/codebase-memory/SKILL.md`
- `skills/mem0/SKILL.md`

Use `mcp_tools_codebase_memory` for codebase navigation, indexing, architecture analysis, code search, symbol tracing, and repository understanding.

Use `mcp_tools_mem0` for persistent cross-session memory (facts, preferences, decisions). Always call `search_memories` before `add_memory` to avoid duplicates.

Important rules:

- The MCP servers run as host binaries (not Docker). `codebase-memory-mcp` lives at `~/.local/bin/codebase-memory-mcp`; `mem0-mcp-selfhosted` runs behind the `~/.local/bin/mem0-launcher` wrapper (sourcea `.env.mem0`).
- Do not spawn old `mcp-tools-*-docker` wrappers — they were removed. If a client still references them, run `mcp-tools mcp-config` to re-register cleanly.
- Persistent data lives under `$HOME/mcp-tools-data/` (subdirs per MCP: `mem0`, `ollama`, plus `state.json`).
- NEVER fall back to native `Grep`/`Read`/`find`/`bash grep` for repo-wide search — that's what `mcp_tools_codebase_memory` is for.
- NEVER write local `notes.md`/scratchpad files to persist facts across sessions — that's what `mcp_tools_mem0` is for.

## OMP MCP discovery workflow

OMP v16.3.5 may expose MCP servers as discoverable tools instead of loading every MCP tool directly into the initial tool inventory.

When a user asks for a task handled by an MCP server and the corresponding `mcp__...` tool is not initially visible, do not say the tool is unavailable.

First use `search_tool_bm25` with a query describing the needed capability.

Examples:
- codebase-memory architecture: `codebase memory architecture repository graph`
- codebase-memory search: `codebase memory search code symbols`
- mem0 memory search: `mem0 search memories persistent context`
- mem0 add memory: `mem0 add memory remember preference`

After discovery activates the matching MCP tool, call the activated tool.

Do not fall back to bash, Docker, Python, or host binaries for normal MCP tasks unless explicitly debugging MCP setup.
