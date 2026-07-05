---
name: mem0
description: >
  Persistent cross-session memory via the `mcp_tools_mem0` MCP server. Use whenever
  the user asks to remember, recall, note, save, store, retrieve, or look up past
  facts, decisions, preferences, or context from prior sessions. Triggers EN:
  "remember", "recall", "note this", "save this", "we decided", "the user prefers",
  "prior session", "what did we discuss", "context from before". Triggers ES:
  "recuerda", "acuérdate", "guarda esto", "apunta esto", "hemos decidido",
  "el usuario prefiere", "sesión anterior", "qué habíamos hablado", "contexto de antes".
  ALWAYS call `search_memories` before `add_memory` to avoid duplicates. NEVER fall
  back to local notes files, bash `echo >> notes.md`, or agent scratchpad for these
  intents — persistent memory MUST go through this MCP. Native tools are only OK
  when the user's ask is scoped to the current session (single-turn scratch).
---

# mem0

## Purpose

Use `mcp_tools_mem0` whenever the user asks the agent to remember, recall, save, or look up any fact, decision, preference, or context that must survive the current session.

This MCP provides persistent cross-session memory: facts about the user, decisions taken, preferences, project context, prior conversations. Backed by Qdrant (vector search) and Ollama (embeddings) — both are Docker services managed by `mcp-tools up`.

## Fast path

For simple mem0 tasks, do not read this full skill file again unless the user explicitly asks.

Use `mcp_tools_mem0` directly.

Fast workflows:

- Recall / lookup past context: call `search_memories` with a semantic query.
- Save a new fact / preference / decision: call `search_memories` first to dedupe, then `add_memory`.
- List memories with filters: call `get_memories`.
- Get a specific memory by ID: call `get_memory`.
- Find relationships between entities: call `search_graph`.
- Modify an existing memory: call `update_memory`.
- List who/what has stored memories: call `list_entities`.

Do not enter plan mode for simple read-only recalls.

Do not create local plan files for simple recall requests.

Do not ask follow-up questions after completing a simple `search_memories` / `add_memory` request.

If the user's query cleanly matches an existing memory, prefer `search_memories` (single call) over `list_entities` + `get_memories` scans.

## When to use vs when NOT to use

Use `mcp_tools_mem0` when:

- The user says "recuerda", "acuérdate", "guarda esto", "apunta esto", "remember", "recall", "note this", "save this".
- The user asks "qué habíamos decidido", "what did we decide", "prefiero X" ("the user prefers X"), "en la sesión anterior".
- The user asks for context that clearly comes from a prior session.
- The user establishes a durable preference the agent should honour later (formatting, tone, tooling defaults, project conventions).

Do NOT use `mcp_tools_mem0` when:

- The user asks to read one specific file they already named → native `Read` is correct.
- The user asks to run a build / test / shell command → native `bash` is correct.
- The user asks to edit code → native `edit` / `write` is correct.
- The scratchpad is single-turn (the note is only meaningful inside this reply) → agent context is enough.
- The user asks about the codebase, architecture, symbols, or code search → that's `mcp_tools_codebase_memory`, not this MCP.

## Runtime

The MCP server name is:

```txt
mcp_tools_mem0
```

The runtime is a host binary `mem0-mcp-selfhosted` (installed by `uv tool install`), wrapped by `~/.local/bin/mem0-launcher`. The wrapper sources `~/mcp-tools/.env.mem0` on each invocation and execs the binary. No Docker container is involved for mem0 itself.

Dependencies (Docker services managed by `mcp-tools up`):

- `mcp-tools-ollama` on port `:11434` (embeddings + LLM).
- `mcp-tools-mem0-qdrant` on port `:6333` (vector store).

Persistent data lives under `~/mcp-tools-data/mem0/{history,uv-cache,config}` and the external Docker volume `mcp-qdrant-storage`.

## Transport

The MCP is configured as `stdio`.

Clients should call the configured MCP server named `mcp_tools_mem0`.

Do not replace MCP tool calls with raw shell commands during normal memory tasks unless the client fails to expose the requested MCP tool.

## Important client tool naming

Do not invent internal tool names like:

```txt
mcp__mcp_tools_mem0_search_memories
```

Use the MCP tools as exposed by the active client (Claude Code / OpenCode use the bare `<tool_name>`; OMP namespaces them as `mcp__mcp_tools_mem0_<tool>`).

If direct MCP tool calling fails because the client does not expose a specific tool, run the launcher directly to drop into a stdio MCP session and inspect its command surface with `--help` — mcp-tools ships no bespoke CLI subcommand.

## Available tools

Tools exposed by `mcp_tools_mem0`:

- `search_memories` — semantic search across stored memories; returns ranked matches.
- `add_memory` — store a new fact/preference/decision. Only after `search_memories` confirms no duplicate.
- `get_memories` — list memories with filters (entity, tag, date range).
- `get_memory` — fetch a single memory by ID.
- `search_graph` — find relationships between entities in the memory graph.
- `update_memory` — modify an existing memory (edit content, tags, or metadata).
- `list_entities` — list entities (users, projects) that have stored memories.

## Default workflow

When the user asks to recall:

1. Call `search_memories` with a semantic query built from the user's phrasing.
2. If matches exist → summarise the top hits and answer directly.
3. If nothing matches → say so; do NOT invent an answer, do NOT hallucinate a memory.

When the user asks to save (`recuerda`, `remember`, `guarda`):

1. Call `search_memories` with a query summarising the fact — cheap dedupe.
2. If an equivalent memory exists → `update_memory` on that ID (avoid duplicates).
3. If nothing matches → `add_memory` with a concise, self-contained content string.
4. Confirm with the user in one line: "Guardado" / "Saved" — do not read back the whole memory.

When the user asks to list:

1. Call `get_memories` with the tightest filter you can infer (entity, tag).
2. Return a compact summary, not the raw JSON.

When the user asks to delete a memory:

1. NEVER delete on the first mention. Ask for explicit confirmation.
2. Only after confirmation, use the appropriate deletion path (currently not exposed as a tool — treat as read-only unless the user provides an explicit override).

## Query hygiene

For `search_memories`:

- Use the user's own phrasing when possible; the embeddings match semantics, not keywords.
- Keep queries short and specific ("prefers static binaries", not "what does the user usually like when he compiles code with go or rust or c or c++").
- If the first query returns nothing, try one paraphrase before giving up.

For `add_memory` content:

- Self-contained sentences: "User prefers static binaries compiled with `-static -O2`." — not "he said he likes that thing".
- One fact per memory; if the user drops multiple facts, split into multiple `add_memory` calls.
- No PII beyond what the user themselves stated.

## Output limits

For memory answers:

- Do not paste raw JSON.
- Do not dump entire memory content when the user only asked for a summary.
- Prefer short bullets: one memory per line, format `<id> · <summary>`.
- If the user asks a yes/no ("¿tengo esto guardado?"), answer yes/no plus the matching memory in one sentence.

## Do not do

- Do not call `add_memory` without calling `search_memories` first — duplicates are silent and expensive to clean up.
- Do not delete memories without explicit user confirmation.
- Do not synthesise memory content the user did not state.
- Do not write local `notes.md` / `memories.txt` / scratchpad files to persist facts — that's exactly what this MCP replaces.
- Do not answer "de memoria" (from context) when the user asks about prior sessions or established preferences; call `search_memories` first.
- Do not read this full skill file for a simple `search_memories` or `add_memory` call — the fast path above is enough.
- Do not spawn the old `mcp-tools-mem0-docker` wrapper — it was removed. The correct entry point is `~/.local/bin/mem0-launcher`.

## Escalation if the MCP fails

1. `/mcp list` in the client to check status.
2. `/mcp reload` or `/mcp reconnect mcp_tools_mem0`.
3. If still failing: close the client fully and relaunch.
4. Ollama or Qdrant down: `mcp-tools up` restarts both services.
5. Logs: `mcp-tools logs mcp_tools_ollama --tail 50` and `mcp-tools logs mcp_tools_mem0_qdrant --tail 50`.
6. mem0 binary missing: `mcp-tools mem0 install` (idempotent).
