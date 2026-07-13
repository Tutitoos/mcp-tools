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

This MCP provides persistent cross-session memory: facts about the user, decisions taken, preferences, project context, prior conversations. Backed by Qdrant (vector search) and Ollama (embeddings) — both are Docker services managed from the mcp-tools web panel (`/services`) or `docker compose -f ~/mcp-tools/dockers/compose.yaml --env-file ~/mcp-tools/.env up -d`.

## Known state (verifica antes de usar)

Esta sección es la fuente del detalle; `RULES.md` solo lleva la firma corta.

- **Bug upstream (verificado 2026-07-06, re-verificado 2026-07-13)**: `search_memories(query)` y `get_memories(user_id)` fallan con `ValueError: Top-level entity parameters frozenset({'user_id'}) are not supported in search(). Use filters={'user_id': '...'}` — la lib mem0 nueva exige `filters` y el MCP siempre inyecta `user_id` al top level. Pasar `filters={"user_id": ...}` como parámetro del tool NO lo evita: el MCP añade el top-level igualmente. Solo se arregla upstream (issues #10-#13 de `elvismdev/mem0-mcp-selfhosted`).
- **Estado degradado ocasional**: TODAS las ops (incluidas las fiables) devuelven `RuntimeError: Memory not initialized. Infrastructure may be unavailable.` Suele aparecer tras reiniciar qdrant/ollama (p.ej. `docker compose ... restart` o desde el panel `/services`) sin dar tiempo al init de mem0. Fix: reinicia el proceso — `pgrep -af mem0-mcp-selfhosted | awk '{print $1}' | xargs -r kill`, luego `/mcp reconnect mcp_tools_mem0`.
- **Fiables en estado sano**: `add_memory`, `get_memory(uuid)`, `list_entities`, `update_memory`, `mcp_search_graph`.
- **Rotas en cualquier estado**: `search_memories`, `get_memories` (el bug del filtro).
- **Destructivas**: `delete_memory`, `delete_entities`, `delete_all_memories` — NUNCA sin confirmación explícita del user.

## Fast path

For simple mem0 tasks, do not read this full skill file again unless the user explicitly asks.

Use `mcp_tools_mem0` directly.

Fast workflows:

- Recall / lookup past context: call `search_memories` with a semantic query. If it fails with the known `ValueError` (filters bug, see Known state) → fall back to `list_entities` + `get_memory(uuid)`; do NOT retry the same call.
- Save a new fact / preference / decision: call `search_memories` first to dedupe, then `add_memory`. If the dedupe call hits the known bug → proceed with `add_memory` directly (accept the duplicate risk; one failed attempt is enough).
- List memories with filters: call `get_memories`.
- Get a specific memory by ID: call `get_memory`.
- Find relationships between entities: call `mcp_search_graph`.
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

Dependencies (Docker services, started from the web panel `/services` or `docker compose ... up -d`):

- `mcp-tools-ollama` on port `:11434` (embeddings + LLM).
- `mcp-tools-mem0-qdrant` on port `:6333` (vector store).

Persistent data lives under `~/mcp-tools-data/mem0/{history,uv-cache,config}` and the external Docker volume `mcp-qdrant-storage`.

## Transport

The MCP is configured as `stdio`.

Clients should call the configured MCP server named `mcp_tools_mem0`.

Do not replace MCP tool calls with raw shell commands during normal memory tasks unless the client fails to expose the requested MCP tool.

## Important client tool naming

Usa el nombre exacto que exponga tu cliente MCP activo — no lo adivines:
- Claude Code / OpenCode: nombre bare (`search_memories`, `add_memory`, …).
- OMP: namespaced (`mcp__mcp_tools_mem_search_memories`, `mcp__mcp_tools_mem_add_memory`, …). Nota: OMP acorta `mem0` → `mem` en el prefijo.
- Si tu cliente aún no lo expone: `search_tool_bm25` con la capacidad como query lo activa.

If direct MCP tool calling fails because the client does not expose a specific tool, run the launcher directly to drop into a stdio MCP session and inspect its command surface with `--help` — mcp-tools ships no bespoke CLI subcommand.

## Available tools

Tools exposed by `mcp_tools_mem0` (11 total):

| Intención | Tool | Nota |
| --- | --- | --- |
| Buscar por texto natural | `search_memories(query)` | ❌ roto upstream — ver Known state |
| Listar todo para un usuario | `get_memories(user_id)` | ❌ roto upstream — usa `list_entities` + `get_memory` |
| Guardar un hecho nuevo | `add_memory(text, user_id)` | ✅ tras `search_memories` degradado, asume riesgo de duplicado |
| Recuperar por UUID | `get_memory(memory_id)` | ✅ funciona |
| Modificar un hecho existente | `update_memory(memory_id, text)` | ✅ funciona |
| Listar usuarios/agentes con memorias | `list_entities()` | ✅ funciona |
| Ver relaciones en el grafo | `mcp_search_graph(query)` | ✅ funciona |
| Ver relaciones de una entidad | `mcp_get_entity(name)` | ✅ funciona |
| **Borrar** una memoria | `delete_memory(memory_id)` | ⚠️ destructivo — sólo con confirmación explícita |
| **Borrar** todas las memorias de un scope | `delete_all_memories(user_id/agent_id/run_id)` | ⚠️ destructivo — MUY peligroso, sólo con confirmación explícita |
| **Borrar** entidad completa | `delete_entities(user_id/agent_id/run_id)` | ⚠️ destructivo — cascada, sólo con confirmación explícita |

## Default workflow

When the user asks to recall:

1. Call `search_memories` with a semantic query built from the user's phrasing. On the known `ValueError` → workaround path (`list_entities` + `get_memory`), no retries.
2. If matches exist → summarise the top hits and answer directly.
3. If nothing matches → say so; do NOT invent an answer, do NOT hallucinate a memory.

When the user asks to save (`recuerda`, `remember`, `guarda`):

1. Call `search_memories` with a query summarising the fact — cheap dedupe. On the known `ValueError` → skip dedupe, go straight to `add_memory` (one failed attempt is enough).
2. If an equivalent memory exists → `update_memory` on that ID (avoid duplicates).
3. If nothing matches → `add_memory` with a concise, self-contained content string.
4. Confirm with the user in one line: "Guardado" / "Saved" — do not read back the whole memory.

When the user asks to list:

1. Call `get_memories` with the tightest filter you can infer (entity, tag).
2. Return a compact summary, not the raw JSON.

When the user asks to delete a memory:

1. NEVER delete on the first mention. Ask for explicit confirmation.
2. Only after confirmation, use `delete_memory(memory_id)` (or `delete_all_memories` / `delete_entities` for a whole scope — doubly confirm those: they cascade).

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
4. Ollama or Qdrant down: restart from the web panel (`/services`) or `docker compose -f ~/mcp-tools/dockers/compose.yaml --env-file ~/mcp-tools/.env up -d`.
5. Logs: `docker compose -f ~/mcp-tools/dockers/compose.yaml --env-file ~/mcp-tools/.env logs --tail 50 mcp_tools_ollama` (same for `mcp_tools_mem0_qdrant`), or the panel's `/services` log viewer.
6. mem0 binary missing: reinstall from the web panel (`/tools` → mem0 → install, i.e. `POST /api/tools/mem0/install`).
