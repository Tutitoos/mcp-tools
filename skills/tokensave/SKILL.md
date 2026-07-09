---
name: tokensave
description: >
  Ultra-fast tree-sitter code-graph MCP via the `tokensave` server. Use as
  the DEFAULT first tool for natural-language code exploration in a project
  where `tokensave init` has been run: `tokensave_context` returns the
  relevant symbols' verbatim source PLUS the call paths between them, in one
  call — replaces the grep+read loop. Triggers EN: "how does X work",
  "explore the code", "find the code that", "what does this project do",
  "explain this flow", "one-shot context", "code intelligence". Triggers ES:
  "cómo funciona", "explora el código", "encuentra el código que", "qué hace
  este proyecto", "explica este flujo", "contexto one-shot". Prefer tokensave
  over grep/find/Read for repo-scoped exploration in `tokensave init`'d
  projects. Prefer `serena` when the target is a NAMED symbol you want the
  compiler-accurate answer for. Prefer `codebase-memory` for cross-repo /
  architecture / repos NOT initialized in tokensave. NEVER call tokensave on
  a project without a `.tokensave/` index or global-DB registration — the
  server will refuse to start.
---

# tokensave

## Purpose

Use the `tokensave` MCP whenever the user asks a natural-language question about a project's code and that project has been indexed by `tokensave init`. `tokensave_context` returns the relevant symbols' verbatim source PLUS the call paths between them — in ONE call — so you can answer questions like "how does authentication work" or "where does upload land" without a manual grep+read loop.

Backed by tree-sitter (34 languages) with call/data-flow edges pre-computed at index time. Sub-millisecond queries; the whole index is cached under `.tokensave/` per project.

## Fast path

For simple tokensave tasks, do not read this full skill file again unless the user explicitly asks.

Use `tokensave` directly.

Fast workflows:

- Explore a topic: call `tokensave_context` with the user's own question as `task`.
- Find a specific symbol: call `tokensave_search` with the name.
- Find who calls X: call `tokensave_callers` with the qualified name.
- Find what X calls: call `tokensave_callees`.
- Impact of a change: call `tokensave_impact` with the symbol.
- Node details: call `tokensave_node` with the qualified name.
- Enumerate project files: call `tokensave_files`.
- Session-level scratch memory: `tokensave_todos` — do NOT use as a cross-session replacement for `mcp_tools_mem0`.

Do not enter plan mode for a single `tokensave_context` call.

## When to use vs when NOT to use

Use `tokensave` when:

- The user asks a natural-language question about the current project's code and `tokensave init` has been run there.
- You need one-shot context for a bug, flow, or feature area.
- You are about to grep+read a project — tokensave replaces that loop.
- The user says "explora", "encuentra el código", "cómo funciona X", "explain X flow".

Do NOT use `tokensave` when:

- The project has no `.tokensave/` directory and is not registered in the tokensave global DB. Server will refuse to start with `no TokenSave index found`.
- The question spans multiple repositories → prefer `mcp_tools_codebase_memory`.
- The target is a NAMED symbol you need compiler-accurate references or a rename for → prefer `mcp_tools_serena`.
- The task is opening a specific file the user already named → native `Read`.
- The user asks about architecture/community structure across the repo → prefer `mcp_tools_codebase_memory`'s `get_architecture`.

## `tokensave init` — the one prerequisite

If a project has not been initialised, tokensave `serve` fails at handshake. To index a project:

```bash
cd /absolute/path/to/project
tokensave init
```

Blast-radius notes for the user (init is aggressive):

- Creates `.tokensave/` in the project (adds it to `.gitignore` automatically).
- Registers the project in the global DB at `~/.tokensave/`.
- Re-runs its autodetect+install cycle on EVERY agent it can find (Claude, OpenCode, Codex, VS Code, Copilot, "pi" targeting pi.dev NOT OMP-Oh-My-Pi). If the user does not want those side effects, warn before running init the FIRST time on a fresh install.

Once initialised, keep it fresh with `tokensave sync` (incremental) or re-run `tokensave init` after large branch switches. Both are project-scoped and idempotent for already-wired agents.

## Runtime

The MCP server name is:

```txt
tokensave
```

Bare — NOT `mcp_tools_tokensave`. Tokensave `SelfRegisters` and uses its own naming convention; `mcp-tools mcp-config` deliberately skips it.

The runtime is `~/.cargo/bin/tokensave` (installed by `mcp-tools tokensave install`). No Docker. First install is 5–15 min because tokensave builds 30+ tree-sitter grammars from source.

Per-project index lives at `<project>/.tokensave/`.
Global project registry lives at `~/.tokensave/`.

## Transport

The MCP is configured as `stdio` with args:

```txt
tokensave serve
```

Optional flag: `--path <project-root>` to run against a specific project regardless of cwd. `--timings` annotates each `tools/call` response with pure execution time.

## Important client tool naming

Usa el nombre exacto que exponga tu cliente MCP activo — no lo adivines:
- Claude Code / OpenCode: nombre bare (`tokensave_context`, `tokensave_search`, …).
- OMP: namespaced pero SIN el prefijo `mcp_tools_` (`mcp__tokensave_context`, `mcp__tokensave_search`, …) — a diferencia de los demás MCP de mcp-tools, tokensave corre como server bare `tokensave`, no `mcp_tools_tokensave`.
- Si tu cliente aún no lo expone: `search_tool_bm25` con la capacidad como query lo activa.

If the client does not expose a specific tokensave tool, activate it via `search_tool_bm25` with a query like `tokensave context code exploration` — OMP's tool discovery layer will surface it.

## Available tools (subset)

- `tokensave_context` — natural-language query → relevant symbols + call paths + verbatim source.
- `tokensave_search` — find a symbol by name.
- `tokensave_callers` / `tokensave_callees` — call graph traversal from a qualified name.
- `tokensave_impact` — blast radius of changing a symbol.
- `tokensave_node` — full details of a single node.
- `tokensave_files` — enumerate files known to the index.
- `tokensave_affected` — files affected by a set of changed files.
- `tokensave_todos` — session-scoped scratchpad (NOT `mem0` replacement).
- CLI: `tokensave tool <name> --task "..."` runs any MCP tool from the shell.

## Default workflow

When the user asks an exploratory question about a project:

1. Verify the project has a `.tokensave/` (or is in `tokensave list`). If not, tell the user and offer to run `tokensave init`.
2. Call `tokensave_context` with the user's question as `task`.
3. Follow up with `tokensave_callers` / `tokensave_callees` if the answer needs the call path.
4. Only if `tokensave_context` misses details, escalate to `tokensave_search` for specific names, then `tokensave_node`.

When the user asks about impact of a change:

1. `tokensave_impact` on the target symbol's qualified name.
2. Summarise: number of callers, number of files affected, list top 5–10.

When the user asks "who calls X":

1. `tokensave_callers` on X's qualified name.
2. If the name is ambiguous, `tokensave_search` first to disambiguate.

## Paths

Always use absolute paths for `--path` flags and project references. `.tokensave/` is created inside the project root — do not move it, do not commit it.

## Output limits

- `tokensave_context` returns verbatim source. Do not paste it whole — extract the relevant symbol names + one-line summaries + file:line refs.
- Do not dump raw JSON.
- Bullet form: `file:line — kind — name — one-line summary`.
- If a query returns >20 nodes, summarise by module first and offer to drill down.

## Do not do

- Do not run tokensave against a project without an index — the server will refuse to start and Claude/OpenCode will mark it `not connected`.
- Do not use `tokensave_todos` as cross-session memory — it lives inside `.tokensave/` and dies with the project index. Use `mcp_tools_mem0` for that.
- Do not commit `.tokensave/` — `tokensave init` already appends it to `.gitignore`.
- Do not run `tokensave init` blindly on a project where the user cares about editor-config side effects — warn first.
- Do not use tokensave for cross-repo questions — one project per index.
- Do not read this full skill file for a single `tokensave_context` call — the fast path above is enough.

## Escalation if the MCP fails

1. `/mcp list` in the client to check status.
2. If tokensave shows `not connected`: the current project (client cwd) has no `.tokensave/`. Either run `tokensave init` there or pass `--path` to a project that IS initialised.
3. `/mcp reconnect tokensave` after init.
4. `tokensave doctor` — checks installation, configuration, and agent integration.
5. Missing binary: `mcp-tools tokensave install`.
6. Corrupt index: `tokensave wipe` and re-run `tokensave init`.
