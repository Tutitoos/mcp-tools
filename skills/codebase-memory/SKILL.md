---
name: codebase-memory
description: >
  Persistent code knowledge graph and code search over indexed local repositories
  via the `mcp_tools_codebase_memory` MCP server. Use for ANY repo-wide task:
  search, grep, navigation, architecture, symbol trace, dependency graph, refactor
  risk, "where is X implemented", "who calls Y", "trace flow from A to B", "explain
  this repo". Triggers EN: "search", "grep", "find in repo", "where is", "who
  calls", "trace", "navigate", "explain this codebase", "what depends on", "refactor
  risk". Triggers ES: "busca", "encuentra", "grep en el repo", "dónde está",
  "quién llama a", "traza", "explica el repo", "navega", "de qué depende".
  ALWAYS prefer this MCP over native `Grep`/`Read`/`find`/`bash` for repo-scoped
  operations. Native `Read` is OK only when opening ONE specific file the user
  already named; native `Grep` is OK only inside a single known file with a fixed
  path. Any multi-file / repo-wide search MUST go through this MCP. Pass absolute
  paths (never `~`). Default indexing mode is `moderate`; use `full` only for
  architecture analysis or persistent team bootstrap.
---

# codebase-memory-mcp

## Purpose

Use `mcp_tools_codebase_memory` whenever the user asks about a local codebase, architecture, code navigation, dependencies, flows, symbols, refactors, implementation details, repository structure, or code search.

This MCP provides a persistent code knowledge graph and code search over indexed repositories.

## Fast path

For simple codebase-memory tasks, do not read this full skill file again unless the user explicitly asks.

Use `mcp_tools_codebase_memory` directly.

Fast workflows:

- List indexed projects: call `list_projects`.
- Get architecture: call `get_architecture` with the exact project name from `list_projects`.
- Search code: call `search_code`.
- Search graph: call `search_graph`.
- Get snippets: call `get_code_snippet`.
- Check indexing: call `index_status`.

Do not enter plan mode for simple read-only tasks.

Do not create local plan files for simple read-only tasks.

Do not ask follow-up questions after completing a simple read-only request.

If the project name is already known, use it directly and avoid calling `list_projects` again.

## Fast architecture mode

For requests like "analyze architecture", "show architecture", "explain the architecture", "analiza la arquitectura", or "dame la arquitectura":

1. Use only `get_architecture` first.
2. Do not call `get_code_snippet` unless the user asks for implementation details.
3. Do not call `get_graph_schema` unless debugging the graph model.
4. Do not call `trace_path` unless the user asks for a specific flow from A to B.
5. Do not fetch full source files for a high-level architecture answer.
6. Keep the answer compact: summary, main packages, hotspots, boundaries, and risks.

Only use snippets when the user asks for code-level details.

## Output limits

For normal architecture answers:

- Do not paste full source code.
- Do not paste raw JSON.
- Do not render large ASCII diagrams unless explicitly requested.
- Prefer short tables and concise bullets.
- Maximum default answer: around 600-900 words.
- If more detail is useful, summarize first and offer deeper sections.
- Do not duplicate the same explanation multiple times.
- Do not read large artifacts unless strictly necessary.

When using `get_code_snippet`, summarize the relevant lines instead of dumping the whole `source` field.

If the user asks for a quick answer, keep it under 300 words.

## Runtime

The MCP server name is:

```txt
mcp_tools_codebase_memory
```

The runtime is a host binary at `~/.local/bin/codebase-memory-mcp` (symlinked from `~/.local/share/codebase-memory-mcp/`), installed by `mcp-tools codebase-memory install`. No Docker container is involved. Data lives at `~/.local/share/codebase-memory-mcp/` (upstream default) — mcp-tools does not manage it manually.

## Transport

The MCP is configured as `stdio`.

Clients should call the configured MCP server named `mcp_tools_codebase_memory`.

Do not replace MCP tool calls with raw shell commands during normal code analysis unless the client fails to expose the requested MCP tool.

## Important client tool naming

Do not invent internal tool names like:

```txt
mcp__mcp_tools_codebase_memory_get_architecture
```

Use the MCP tools as exposed by the active client.

If direct MCP tool calling fails because the client does not expose a specific tool, the fallback is to inspect the upstream `codebase-memory-mcp --help` output — mcp-tools ships no bespoke CLI wrapper.

## Available tools

Common tools exposed by `mcp_tools_codebase_memory`:

- `list_projects`
- `index_repository`
- `index_status`
- `search_code`
- `search_graph`
- `query_graph`
- `trace_path`
- `get_code_snippet`
- `get_graph_schema`
- `get_architecture`
- `detect_changes`
- `delete_project`
- `manage_adr`
- `ingest_traces`

## Default workflow

When the user asks about a repository:

1. Call `list_projects` only if the project name is unknown.
2. Check whether the target repo is already indexed.
3. If it is not indexed, ask for or use the absolute repo path.
4. Call `index_repository`.
5. Verify with `index_status`.
6. Use `get_architecture`, `search_code`, `search_graph`, `query_graph`, `trace_path`, or `get_code_snippet` depending on the task.
7. Answer using findings from the MCP.

## Repo paths

Always use absolute repo paths.

Good:

```json
{
  "repo_path": "/home/tutitoos/Desktop/Kena/libraries/library-http"
}
```

Bad:

```json
{
  "repo_path": "~/Desktop/Kena/libraries/library-http"
}
```

Do not pass `~` as part of `repo_path`.

If the current user is not `tutitoos`, resolve the real home path first using `$HOME` or `pwd`, then pass the final absolute path.

## Indexing

Use `index_repository` when a repo is missing from `list_projects`, stale, or the user explicitly asks to index/reindex.

Typical arguments:

```json
{
  "repo_path": "/absolute/path/to/repo",
  "mode": "moderate",
  "persistence": false
}
```

## Indexing modes

Use:

- `fast` for quick symbol/code overview.
- `moderate` for normal work.
- `full` for the most complete graph, architecture analysis, semantic/similarity edges, or persistent team bootstrap.

Default mode:

```txt
moderate
```

Use `full` when the user explicitly asks for complete/deep indexing, architecture analysis, semantic graph quality, or persistent artifacts.

Do not degrade from `full` to `fast` without asking. If `full` is too heavy, prefer `moderate` and explain why.

## Persistence

Use:

```json
{
  "persistence": true
}
```

when the user wants a portable graph artifact written into the repository.

Expected artifact:

```txt
.codebase-memory/graph.db.zst
```

This is useful for team bootstrap and avoiding full re-indexing.

If persistence fails, report the error. Do not silently retry with `persistence: false`.

## Project names

Do not assume the final project name.

After indexing, call `list_projects` and use the project name returned by the MCP.

The project may be named after:

- the folder basename, for example `library-http`
- the package name, for example `@kena/http`
- the absolute path converted into a safe name, for example `home-tutitoos-Desktop-Kena-libraries-library-http`

Use the returned project name for later calls.

## Verification after indexing

After `index_repository`, verify in this order:

1. `list_projects`
2. `index_status`
3. `get_architecture`

If `persistence` was enabled and shell access is available, also verify:

```bash
ls -lh /absolute/path/to/repo/.codebase-memory/graph.db.zst
```

The file should exist and have size greater than zero.

## Search strategy

For "where is X implemented?":

1. `search_code`
2. `search_graph`
3. `get_code_snippet`

For "explain this architecture":

1. `get_architecture`
2. `query_graph` only if deeper graph relationships are needed.
3. `search_graph` only if specific symbols/packages need expansion.

For "trace flow from A to B":

1. `search_graph` for both symbols.
2. `trace_path`.
3. `get_code_snippet` for relevant nodes only if the user asks for implementation details.

For "what changed?":

1. `detect_changes`.
2. `search_code` if needed.

For "show me the relevant code":

1. `search_code`.
2. `get_code_snippet`.

## Error handling

If indexing returns an "already indexing" or similar error:

1. Do not launch another indexing job in parallel.
2. Call `index_status`.
3. Wait for or report the current status.

If `list_projects` is empty after indexing:

1. Retry `index_repository` once.
2. If it fails again, report the raw error.

If `get_architecture` says project not found:

1. Call `list_projects`.
2. Use the exact project name returned by the MCP.
3. Retry with that name.

If indexing fails because the repo is too large:

1. If current mode was `full`, retry with `moderate`.
2. Do not degrade to `fast` without user approval.

If an MCP tool is not exposed by the active client:

1. Do not keep retrying invented internal tool names.
2. Consult the upstream binary's `--help` for CLI equivalents.
3. Report clearly that CLI fallback was used.

## Do not do

Do not use relative repo paths.

Do not use `~` in MCP `repo_path`.

Do not re-index in parallel.

Do not pass `target_projects` unless using cross-repo intelligence mode.

Do not delete indexed projects unless the user explicitly asks.

Do not silently change `persistence: true` to `false`.

Do not read this entire skill file every time for simple tasks once these rules are already loaded.

Do not use `get_code_snippet` for general architecture summaries unless the user explicitly asks for source-level detail.

Do not call `get_graph_schema` for normal analysis.

Do not generate long ASCII architecture diagrams unless explicitly requested.

## Debug commands

Use these only for debugging the MCP runtime:

```bash
codebase-memory-mcp --version
codebase-memory-mcp --help
```

Verify install status via mcp-tools:

```bash
mcp-tools codebase-memory status
```

Reinstall (idempotent) if the binary is missing or stale:

```bash
mcp-tools codebase-memory install
```

## Example: index library-http

Use:

```json
{
  "repo_path": "/home/tutitoos/Desktop/Kena/libraries/library-http",
  "mode": "full",
  "persistence": true
}
```

Then verify:

1. `list_projects`
2. `index_status`
3. `get_architecture`

If persistence was enabled, also verify:

```bash
ls -lh /home/tutitoos/Desktop/Kena/libraries/library-http/.codebase-memory/graph.db.zst
```

## Example: fast architecture answer

For a compact architecture answer, use only `get_architecture` on the target project and produce:

- overall pattern
- main packages
- hotspots
- boundaries
- risks or refactor candidates

Do not fetch source snippets unless the user asks for code-level details.
