---
name: serena
description: >
  Language-server-backed semantic code MCP via `mcp_tools_serena`. Use for
  precise LSP-grade operations on a SINGLE project: find symbol, find
  declaration, find references, rename, replace symbol body, list symbols
  in a file, hierarchical outline. Requires calling `activate_project` first
  with an absolute path. Triggers EN: "find references", "who uses this
  symbol", "rename symbol", "list symbols in file", "symbol at", "declaration
  of", "replace symbol body", "LSP", "semantic edit". Triggers ES: "quién usa
  este símbolo", "encuentra referencias", "renombra", "símbolos en el archivo",
  "declaración de", "reemplaza el cuerpo del símbolo", "operación LSP",
  "edición semántica". Prefer serena over grep-based tools when the target
  is a NAMED symbol you want the compiler-accurate answer for. Prefer
  `tokensave` for natural-language exploration; prefer `codebase-memory` for
  cross-repo / architecture. NEVER call serena on a project you have not
  activated in this session.
---

# serena

## Purpose

Use `mcp_tools_serena` for precise, LSP-accurate operations on a single project: symbol resolution, references, declarations, symbol-scoped edits, and language-aware outlines.

Backed by Solid-LSP running per-language language servers (TypeScript, Python, Rust, Go, Java, C#, Ruby, PHP, and more). Because it is an LSP, its answers match what your compiler/IDE sees — not what a regex matches.

## Fast path

For simple serena tasks, do not read this full skill file again unless the user explicitly asks.

Use `mcp_tools_serena` directly.

Fast workflows:

- Locate a symbol by name: call `find_symbol` with `name_path_pattern` and an optional `relative_path`.
- Find where a symbol is used: call `find_referencing_symbols`.
- Find the declaration: call `find_declaration`.
- List LSP symbols of a file: call `get_symbols_overview`.
- Read a symbol's body: call `find_symbol` with `include_body: true` (or `include_info` for signature only).
- Semantic edit: call `replace_symbol_body` / `insert_after_symbol` / `insert_before_symbol`.

Do not enter plan mode for a single read-only symbol lookup.

## When to use vs when NOT to use

Use `mcp_tools_serena` when:

- The user names a specific symbol (function, class, method, constant, type) and asks about it.
- The user asks for compiler-accurate references, declarations, or a rename.
- The user asks to edit a symbol's body without re-typing the file around it.
- The user asks for an outline / symbol overview of a specific file.
- The project is single-repo, indexable by an LSP, and the user has already told the agent which project.

Do NOT use `mcp_tools_serena` when:

- The question spans multiple repositories → prefer `mcp_tools_codebase_memory`.
- The user asks a natural-language exploratory question ("how does auth work here") → prefer `tokensave` if the project is `tokensave init`'d, else `mcp_tools_codebase_memory`.
- No project has been activated in this session and the user has not named one.
- The target file is not part of an LSP-indexable language.
- The task is opening a specific file the user already named → native `Read`.

## Activate the project FIRST

Every serena session starts with:

```txt
activate_project(project: "/absolute/path/to/project")
```

Use the absolute path. Never `~`.

Serena persists a `.serena/` dir at the project root with cached memories; that is expected — it is the project's onboarding state.

If a project has never been activated, `activate_project` will also create it — this is fine, keep going.

## Runtime

The MCP server name is:

```txt
mcp_tools_serena
```

The runtime is `~/.local/bin/serena` (installed by `mcp-tools serena install` via `uv tool install -p 3.13 serena-agent`). No Docker.

Serena downloads per-language LSP servers on demand under `~/.serena/language_servers/`. First call for a new language may take 10–60 s; subsequent calls are cached.

## Transport

The MCP is configured as `stdio` with args:

```txt
serena start-mcp-server --context agent --project-from-cwd
```

`--context agent` = autonomous-agent scenario (keeps `activate_project` available).
`--project-from-cwd` = default project falls back to the client's cwd.

## Important client tool naming

Do not invent internal tool names like:

```txt
mcp__mcp_tools_serena_find_symbol
```

Use the MCP tools as exposed by the active client (Claude Code / OpenCode use the bare `<tool_name>`; OMP namespaces them as `mcp__mcp_tools_serena_<tool>`).

If the client does not expose a specific serena tool, activate it via `search_tool_bm25` with a query like `serena find symbol` — OMP's tool discovery layer will surface it.

## Available tools

Common tools exposed by `mcp_tools_serena` (subset):

- `activate_project`
- `get_symbols_overview`
- `find_symbol`
- `find_declaration`
- `find_implementations`
- `find_referencing_symbols`
- `replace_symbol_body`
- `insert_after_symbol`
- `insert_before_symbol`
- `list_dir`
- `read_file` (LSP-aware chunking)
- `write_memory` / `read_memory` (per-project notes; NOT a substitute for `mcp_tools_mem0`)
- `search_for_pattern` (regex fallback inside the activated project)

## Default workflow

When the user asks about a specific symbol:

1. Ensure the project is activated. If not, `activate_project` with the absolute path.
2. Call `find_symbol` with a `name_path_pattern` matching the user's phrasing.
3. If they asked "who uses X" / "where is X called": call `find_referencing_symbols`.
4. If they asked "where is X defined": call `find_declaration`.
5. If they asked for the source: use `include_body: true` on `find_symbol`.

When the user asks to edit a symbol's body:

1. `find_symbol` with `include_body: true` to confirm the current source.
2. `replace_symbol_body` with the new source.
3. Do NOT re-write surrounding context — the LSP boundaries do that for you.

When the user asks for a file overview:

1. `get_symbols_overview` with `relative_path` set to the file.
2. Return a compact tree: `Class > method`, `Interface > field` — not the raw JSON.

## Paths

Always use absolute paths for `project`. `relative_path` arguments are relative to the activated project root — never absolute.

Bad:

```json
{ "project": "~/mcp-tools" }
```

Good:

```json
{ "project": "/home/tutitoos/mcp-tools" }
```

## Output limits

- Do not paste the full raw JSON that serena returns; extract the key fields (name, kind, file, line range).
- Do not dump symbol bodies unless the user asked for the code.
- Bullet form: `file:line — kind — name — one-line summary`.
- If `find_referencing_symbols` returns many hits (>30), summarise per-file counts first and ask if the user wants the full list.

## Do not do

- Do not call any serena tool before `activate_project` in a fresh session.
- Do not run serena against a non-LSP language and expect precision — fall back to `tokensave` or `codebase-memory`.
- Do not use serena's `write_memory` as a substitute for `mcp_tools_mem0`. Serena memories are per-project scratchpads and do NOT survive across projects.
- Do not use serena for cross-repo questions — one LSP session = one project.
- Do not synthesise symbol locations from context — always call the MCP.
- Do not read this full skill file for a single `find_symbol` call — the fast path above is enough.

## Escalation if the MCP fails

1. `/mcp list` in the client to check status.
2. `/mcp reload` or `/mcp reconnect mcp_tools_serena`.
3. If still failing: close the client fully and relaunch. LSP servers can leak on abrupt kills.
4. Missing LSP for a language: `~/.serena/language_servers/` gets populated on the first `activate_project` for that language — first call for a new language may take ~30 s.
5. Serena binary missing: `mcp-tools serena install` (idempotent).
6. Serena config file: `~/.serena/serena_config.yml`.
