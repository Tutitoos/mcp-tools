# codebase-memory-mcp

## Purpose

Use `codebase_memory_mcp` whenever the user asks about a local codebase, architecture, code navigation, dependencies, flows, symbols, refactors, implementation details, or repository structure.

This MCP provides a persistent code knowledge graph and code search over indexed repositories.

The MCP must be used through Docker. Do not call the host binary directly.

## Runtime

The MCP server name is:

```txt
codebase_memory_mcp
```

The runtime wrapper is:

```bash
$HOME/.local/bin/codebase-memory-mcp-docker
```

The Docker project lives at:

```bash
$HOME/mcp-custom
```

Persistent data lives at:

```bash
$HOME/mcp-custom-data/codebase-memory-mcp
```

The wrapper uses a persistent Docker container and executes the MCP through:

```bash
docker exec -i mcp-custom-codebase-memory-mcp codebase-memory-mcp
```

Do not bypass this wrapper unless debugging the Docker setup.

## Transport

The MCP is configured as `stdio`.

Clients should call the configured MCP server named `codebase_memory_mcp`.

Do not replace MCP tool calls with raw shell commands during normal code analysis.

## Available tools

Common tools exposed by `codebase_memory_mcp`:

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

1. Call `list_projects`.
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

For “where is X implemented?”:

1. `search_code`
2. `search_graph`
3. `get_code_snippet`

For “explain this architecture”:

1. `get_architecture`
2. `query_graph`
3. `search_graph`

For “trace flow from A to B”:

1. `search_graph` for both symbols
2. `trace_path`
3. `get_code_snippet` for relevant nodes

For “what changed?”:

1. `detect_changes`
2. `search_code` if needed

For “show me the relevant code”:

1. `search_code`
2. `get_code_snippet`

## Error handling

If indexing returns an “already indexing” or similar error:

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

If indexing explodes because the repo is too large:

1. If current mode was `full`, retry with `moderate`.
2. Do not degrade to `fast` without user approval.

## Do not do

Do not call the host binary directly:

```bash
$HOME/.local/opt/codebase-memory-mcp
```

Do not bypass Docker.

Do not use relative repo paths.

Do not use `~` in MCP `repo_path`.

Do not re-index in parallel.

Do not pass `target_projects` unless using cross-repo intelligence mode.

Do not delete indexed projects unless the user explicitly asks.

Do not silently change `persistence: true` to `false`.

## Debug commands

Use these only for debugging the MCP runtime:

```bash
$HOME/.local/bin/codebase-memory-mcp-docker --version
$HOME/.local/bin/codebase-memory-mcp-docker --help
$HOME/.local/bin/codebase-memory-mcp-docker config list
```

Check persistent container:

```bash
docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}" | grep mcp-custom-codebase-memory-mcp
```

Start container manually:

```bash
cd $HOME/mcp-custom
docker compose up -d codebase_memory_mcp
```

Stop container manually:

```bash
cd $HOME/mcp-custom
docker compose stop codebase_memory_mcp
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
