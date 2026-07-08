---
description: mcp-tools MCP routing and hard rules — which server to use for what, and how to pick between MCP and native tools.
alwaysApply: true
---

# mcp-tools — reglas de uso

Este fichero es cargado globalmente por Claude Code, OpenCode y OMP. Define qué MCP de `mcp-tools` usar para cada intención, cuándo NO recurrir a herramientas nativas, y las reglas duras compartidas.

## Known bugs — read first

## ⚠️ mem0: `search_memories` y `get_memories` rotos upstream (verificado 2026-07-06)

Las dos operaciones listadas abajo pasan `user_id` al top level; la lib mem0 nueva exige `filters={user_id: ...}`. NO uses estas operaciones; usa los workarounds indicados.

| Operación | Estado | Workaround |
| --- | --- | --- |
| `mcp_tools_mem0` → `search_memories(query)` | ❌ roto | `add_memory` con `event: ADD` para guardar; `get_memory` por UUID para recuperar |
| `mcp_tools_mem0` → `get_memories(user_id)` | ❌ roto | `list_entities` + UUIDs conocidos |
| `mcp_tools_mem0` → `add_memory(...)` | ✅ funciona | usar tal cual, aceptar riesgo de duplicado |
| `mcp_tools_mem0` → `get_memory(uuid)` | ✅ funciona | usar tal cual |
| `mcp_tools_mem0` → `list_entities()` | ✅ funciona | usar tal cual |

**Regla**: nunca borres memorias sin confirmación explícita del user.

## How to decide which tool to use (default: serena for code)

**Serena is the default for any code operation on a named symbol.** Use `mcp_tools_serena` (after `activate_project("/absolute/path")`) for: reading a function/method/struct body, listing who calls a function, finding its declaration, getting a file's symbol outline, renaming a symbol, replacing a symbol's body. It is LSP-accurate (~60% fewer tokens than `rtk grep`, ZERO false positives).

Decision tree (apply in order, first match wins):

0. **Code operation on a NAMED symbol** (function, class, method, struct, type, constant, field) — even "show me how X works", "muéstrame el cuerpo", "dónde se usa X" → **`mcp_tools_serena`**. Call `activate_project` first if you haven't. Default fallback to native `Read` for these is PROHIBITED (see hard rules below).
1. **Memoria persistente cross-session** (recordar, recuperar, guardar decisiones/preferencias/hechos, "qué habíamos hablado", "el usuario prefiere")
   → **`mcp_tools_mem0`**. Empieza SIEMPRE por `search_memories` (bug conocido — ver "Known bugs — read first" arriba). Nunca uses `echo >> notes.md`, scratchpad del agente, ni memoria de contexto para persistir.

2. **Pregunta natural-language sobre CÓMO funciona una zona del código** (proyecto con `tokensave init`) — "cómo funciona X", "explora el flujo Y", "encuentra el código que hace Z"
   → **`tokensave` (`tokensave_context`)**. Devuelve entry points + call paths verbatim en 1 call. Si el proyecto NO está init'd → cae al paso 3.

3. **Arquitectura, cross-repo, comunidades, ADR o repo no indexado por tokensave**
   → **`mcp_tools_codebase_memory`**. Único con `get_architecture`, community detection y `manage_adr`.

4. **Búsqueda de TEXTO LITERAL en fichero(s)** (no semántica; ej. una palabra concreta en comentarios, config, docs, `TODO`, un string exacto)
   → **`rtk grep`** (o `rtk find` si es por nombre de fichero, `rtk tree` para overview de directorio). Empíricamente: **60-77% menos tokens** que `grep`/`find` nativos con el mismo resultado textual.

5. **Lectura de un fichero muy grande** (log >500 líneas, JSON verboso, output docker)
   → **`rtk read`**. Ahorra 60-90% en volumen. **NO uses `rtk read` en código fuente pequeño** — ahí `rtk read` ≈ `cat` (0% savings).

6. **Ninguna de las anteriores**: leer UN fichero pequeño que el user ya nombró (raw config, doc, `.env`, log), editar código puntual, correr un test, `cd`/`ls` puntual
   → **herramientas nativas** (`Read`, `edit`, `bash`).

If the user's request matches multiple branches, apply them in the listed order and merge the results — do not skip serena just because another tool also fits.

### Serena-first quick reference

- "show me function X" / "muéstrame el cuerpo de X" / "cómo funciona X" → `find_symbol(name_path_pattern: "X", include_body: true)`
- "where is X used" / "quién llama a X" / "references of X" → `find_referencing_symbols(name_path_pattern: "X")`
- "where is X defined" / "declaración de X" → `find_declaration(name_path_pattern: "X")`
- "outline of file.go" / "symbols in file" → `get_symbols_overview(relative_path: "internal/.../file.go")`
- "rename X to Y" / "replace body of X" → `rename_symbol` / `replace_symbol_body`
- "list all classes / functions in this file" → `get_symbols_overview(relative_path: "...")`

If you have not yet called `activate_project` for the project, do it FIRST with the absolute path. After activation, all serena tools work for the rest of the session.

### Datos de decisión (benchmark empírico)

Query representativa: encontrar refs de `auth`, leer su body, listar `*.tsx` en `tasks-pilot`. Medido por bytes de output (≈ tokens/4) y latencia.

| Use case | Ganador (tokens) | Runner-up | Nota |
| --- | --- | --- | --- |
| Text literal (`TODO`, string exact) | **rtk grep** ~5t | native grep 0t | Fast, texto-nativo |
| Refs de símbolo (`auth`) | **serena** ~394t · LSP-accurate | rtk grep ~987t · con falsos positivos | tokensave `callers` FALLA en constantes |
| Body de un símbolo nombrado | **serena.find_symbol(include_body)** ~391t | rtk read del fichero ~1805t | 4.6× menos tokens |
| "Cómo funciona X" (pregunta open-ended) | **tokensave_context** ~1654t + call paths | serena para símbolo puntual | tokensave solo si proyecto init'd |
| Listar ficheros por patrón | **rtk find/tree** ~148t | native find ~641t | 77% ahorro |
| Leer fichero código pequeño | native Read ~1805t | rtk read = misma cifra | rtk NO ahorra aquí |
| Arquitectura / clusters | **codebase-memory get_architecture** | (único) | Sin equivalente en otros |

### Regla de desempate serena vs tokensave vs codebase-memory

| Dimensión | serena | tokensave | codebase-memory |
| --- | --- | --- | --- |
| Precisión | LSP compiler-grade | tree-sitter estructural | tree-sitter + BM25 + embeddings |
| Scope | 1 proyecto activado | 1 proyecto init'd | N repos indexados |
| Latencia típica | 70-100ms | 5-10ms | 5-15ms |
| Requisito | `activate_project` | `.tokensave/` presente | `index_repository` corrido |
| Fuerte en | símbolos nombrados, refs, renames, edits semánticos | preguntas open-ended, call paths verbatim | arquitectura, cross-repo, comunidades, ADR |
| Débil en | preguntas open-ended amplias | refs a constantes/data (miss) | edición precisa |

Regla: **si el target es un NOMBRE → serena. Si es una PREGUNTA → tokensave (si init'd) o codebase-memory (si no). Si es TEXTO LITERAL → rtk grep. Si es un GLOB de fichero → rtk find.**

## Regla dura de preferencia

Para toda tarea que caiga en (0), (1), (2), (3) o (4) del árbol:

- **PROHIBIDO** saltarse serena para "ver el cuerpo de una función", "encontrar refs de un símbolo nombrado", "obtener el outline de un fichero" o "editar un símbolo". Estos casos van al paso 0 → serena SIEMPRE. Si serena no responde, ver escalación al final; NO caer a `Read`/`Grep`/`rtk grep` por inercia.
- **PROHIBIDO** llamar a `Read` sobre un fichero `.go`/`.ts`/`.py`/`.rs`/`.java` cuyo objetivo es entender el código — usa serena. `Read` sobre código solo se permite cuando el user nombró el fichero por path (no por nombre de símbolo) y quiere ver el contenido raw.
- **PROHIBIDO** hacer fallback a `Grep`/`Read`/`find`/`bash grep`/scratchpad para búsqueda semántica de símbolos, refs, arquitectura o memoria persistente, salvo que el MCP correspondiente haya devuelto error explícito en la sesión actual.
- **PROHIBIDO** usar `rtk grep` para buscar refs de un símbolo — devuelve 60% más tokens que serena Y trae falsos positivos (matches en strings/comments). `rtk grep` es SOLO para texto literal (paso 4), no para código semántico (pasos 0-3).
- **PROHIBIDO** usar `rtk read` sobre código fuente pequeño (<300 líneas) — 0% savings. Usa native `Read` solo si el fichero no es LSP-indexable.
- **PROHIBIDO** sintetizar "notas" en ficheros locales para reemplazar mem0.
- **PROHIBIDO** intentar responder de memoria si el user pregunta por decisiones/preferencias previas sin haber llamado a `search_memories` primero (NOTA: `search_memories` está roto upstream, ver "Known bugs" — usar `get_memory` por UUID o `list_entities` como workaround).
- **PROHIBIDO** usar `serena.write_memory` o `tokensave_todos` como sustituto de `mcp_tools_mem0` — son scratchpads per-project que mueren con el índice.
- Si el MCP no está expuesto por el cliente activo → `search_tool_bm25` con la capacidad como query (activa tool discovery en OMP). CLI fallback solo tras eso.

## Routing: intención → MCP → tools

| Intención | Tool | Comandos/APIs principales |
| --- | --- | --- |
| **Code operation on a named symbol** (read body, refs, declaration, outline, rename, replace body) — DEFAULT para código | `mcp_tools_serena` | `activate_project` (primera vez), `find_symbol` (con `include_body: true` para cuerpo), `find_referencing_symbols`, `find_declaration`, `find_implementations`, `get_symbols_overview`, `replace_symbol_body`, `rename_symbol` |
| Exploración natural-language en proyecto `tokensave init`'d | `tokensave` | `tokensave_context`, `tokensave_search`, `tokensave_callers`, `tokensave_callees`, `tokensave_impact`, `tokensave_node` |
| Cross-repo, arquitectura, grafo global, ADR, comunidades | `mcp_tools_codebase_memory` | `list_projects`, `index_repository`, `index_status`, `search_code`, `search_graph`, `query_graph`, `trace_path`, `get_code_snippet`, `get_graph_schema`, `get_architecture`, `detect_changes`, `manage_adr`, `ingest_traces` |
| Memoria persistente cross-session: preferencias, decisiones, hechos del user | `mcp_tools_mem0` | `search_memories` (BUG: ver "Known bugs"), `add_memory`, `get_memories` (BUG), `get_memory` (UUID), `list_entities`, `update_memory` |
| Texto literal en fichero(s) (comments, config, strings, `TODO`) | `rtk grep` | `rtk grep <pattern> [path]` — ripgrep con output compacto |
| Ficheros por patrón / árbol de directorio | `rtk find` / `rtk tree` / `rtk ls` | 77% menos tokens que `find`/`tree`/`ls` nativos |
| Log largo, JSON verboso, output docker | `rtk read` / `rtk log` | 60-90% savings en volúmenes grandes |

## Nombres de tools en cada cliente

- **Claude Code / OpenCode**: `<tool_name>` directo (según cada MCP). Tokensave usa naming nativo bare (`tokensave_context`, no `mcp_tools_tokensave_context`).
- **OMP**: nombres namespaced, p.ej.:
  - `mcp__mcp_tools_codebase_memory_get_architecture`
  - `mcp__mcp_tools_mem0_search_memories`
  - `mcp__mcp_tools_serena_find_symbol`
  - `mcp__tokensave_context` (bare `tokensave` server, no `mcp_tools_` prefix)

Si en OMP no aparecen visibles, usa `search_tool_bm25` con la capacidad como query — activará el tool discovery. Queries ejemplo:
- codebase-memory arquitectura: `codebase memory architecture repository graph`
- codebase-memory search: `codebase memory search code symbols`
- serena: `serena find symbol activate project`
- tokensave: `tokensave context code exploration`
- mem0 search: `mem0 search memories persistent context`
- mem0 add: `mem0 add memory remember preference`

## Reglas duras (aplicables SIEMPRE)

1. **`codebase-memory` y `mem0` corren como binarios HOST**, no en Docker. Usa el nombre del wrapper (`codebase-memory-mcp`, `mem0-launcher`) — nunca invoques los binarios crudos con flags manuales para tareas normales.
2. **`serena` corre como binario HOST** (`~/.local/bin/serena`, instalado por `uv tool`). NUNCA llames a un símbolo por serena sin `activate_project` previa en la misma sesión.
3. **`tokensave` corre como binario HOST** (`~/.cargo/bin/tokensave`). NUNCA llames a `tokensave serve` en un cwd sin `.tokensave/` — el server falla al arrancar. Si el user pide exploración en un proyecto no indexado, avísale y ofrece correr `tokensave init` (side effect: reautodetecta agents; re-registra en `~/.pi/`, Codex, VS Code, Copilot).
4. **No bypasees el MCP con shell/docker exec** (`docker exec mcp-tools-...`, `python -c ...`) salvo debug explícito de la infra Docker.
5. **Nunca sintetices input** más grande para "probar" un MCP.
6. **Rutas absolutas** para `codebase_memory`, `serena.activate_project` y `tokensave --path` (nunca `~`, resuelve `$HOME` antes).
7. **Modo de indexado codebase-memory** por defecto `moderate`; solo `full` si el user pide arquitectura completa o grafo semántico persistente.
8. **`mem0` levanta ollama + qdrant automáticamente** vía `depends_on`. Si el wrapper falla con URL error, comprueba `docker compose ps` en `~/mcp-tools`.
9. **NUNCA borres una memoria de mem0** sin confirmación explícita del user.
10. **Scratchpads de MCP ≠ memoria persistente**: `serena.write_memory` (per-project `.serena/memories/`) y `tokensave_todos` (per-project `.tokensave/`) sirven SOLO como notas del proyecto activo. Cualquier fact que deba sobrevivir a la sesión va a `mcp_tools_mem0`.

## Escalación si un MCP no responde

0. **Serena first**: si serena no está conectado, ESTO es prioritario — el agente DEBE intentar activarlo antes de cualquier otra acción. Pasos:
   1. `mcp-tools serena install` (idempotente, instala `~/.local/bin/serena`).
   2. `which serena` para confirmar que está en PATH.
   3. `mcp-tools mcp-config` para re-registrar el MCP server en el cliente activo.
   4. Si el cliente sigue sin exponerlo: `search_tool_bm25` con query `serena find symbol activate project`.
   5. Si tras 3 intentos sigue sin responder, recurre a `tokensave_context` (paso 2) o `mcp_tools_codebase_memory` (paso 3) — NUNCA a `Read`/`Grep` para entender código.
1. **`/mcp list`** para ver el estado por cliente.
2. **`/mcp reload`** o **`/mcp reconnect <server>`**.
3. Si sigue fallando: **cierra completamente el cliente y relánzalo** (`/mcp reload` no purga entradas removidas de la config).
4. Container caído (mem0-qdrant, ollama): `docker compose -f ~/mcp-tools/dockers/compose.yaml --env-file ~/mcp-tools/.env up -d`.
5. Ver logs Docker: `docker logs mcp-tools-<name> --tail 50`.
6. `tokensave` marca "not connected" en Claude/OpenCode → el cwd no tiene `.tokensave/`. Corre `tokensave init` en un proyecto (una vez basta para que serve arranque desde cualquier cwd).
7. `serena` marca "not connected" tras los pasos 0.1–0.4 → `~/.local/bin/serena` no está en PATH o `activate_project` no se ha llamado. Llama `activate_project("/absolute/path")` antes de cualquier otra tool de serena.

## Skills específicos por MCP

Cada MCP tiene un skill con guía detallada. Léelos con `skill://<name>` o cárgalos automáticamente cuando la intención dispare su `description` frontmatter:

- `skill://codebase-memory` — indexado, modos, workflows por tipo de pregunta, CLI fallback.
- `skill://mem0` — search-first-then-add, filtros, workflows por intención, política de borrado.
- `skill://serena` — activate_project, LSP-accurate find/references/rename, cuándo NO usar.
- `skill://tokensave` — natural-language context queries, init prerequisite, blast radius del init.
