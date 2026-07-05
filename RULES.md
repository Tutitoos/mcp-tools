---
description: mcp-tools MCP routing and hard rules — which server to use for what, and how to pick between MCP and native tools.
alwaysApply: true
---

# mcp-tools — reglas de uso

Este fichero es cargado globalmente por Claude Code, OpenCode y OMP. Define qué MCP de `mcp-tools` usar para cada intención, cuándo NO recurrir a herramientas nativas, y las reglas duras compartidas.

## Cómo decidir qué herramienta usar

Antes de tirar de `Grep`/`Read`/`find`/`bash`/scratchpad, aplica este árbol en orden:

1. **¿La petición implica memoria persistente cross-session** (recordar, recuperar, guardar decisiones/preferencias/hechos, "qué habíamos hablado", "el usuario prefiere")?
   → **`mcp_tools_mem0`**. Empieza SIEMPRE por `search_memories`. Nunca uses `echo >> notes.md`, scratchpad del agente, ni memoria de contexto para persistir.

2. **¿La petición implica navegar/entender un repo** (búsqueda por el codebase, "dónde está X", arquitectura, quién llama a Y, trazar un flujo, refactor risk, "explica el repo")?
   → **`mcp_tools_codebase_memory`**. `list_projects` → indexar si falta → herramienta según intención. Nunca uses `Grep`/`Read`/`find`/`bash grep` para búsqueda multi-fichero o preguntas de arquitectura.

3. **Ninguna de las dos**: leer UN fichero que el user ya nombró, editar código, correr un test, cd/ls puntual → **herramientas nativas** (`Read`, `edit`, `bash`).

Si la petición mezcla ambas (p.ej. "busca en el repo dónde se usa X y recuérdame lo que decidimos"), llama primero al MCP correspondiente a cada parte. No preguntes al user; ejecuta.

## Regla dura de preferencia

Para toda tarea que caiga en (1) o (2) del árbol:

- **PROHIBIDO** hacer fallback a `Grep`/`Read`/`find`/`bash grep`/scratchpad para búsqueda repo-wide o memoria persistente, salvo que el MCP haya devuelto error explícito en la sesión actual.
- **PROHIBIDO** sintetizar "notas" en ficheros locales para reemplazar mem0.
- **PROHIBIDO** intentar responder de memoria si el user pregunta por decisiones/preferencias previas sin haber llamado a `search_memories` primero.
- Si el MCP no está expuesto por el cliente activo → **CLI fallback** vía el wrapper Docker (`$HOME/.local/bin/mcp-tools-<name>-docker cli <tool> '<json>'`). Solo entonces se admite shell.

## Routing: intención → MCP → tools

| Intención | MCP | Tools disponibles |
| --- | --- | --- |
| Navegar codebase, arquitectura, "dónde está X", grafo de código, refactors, trace de flujos, búsqueda repo-wide, indexar/reindexar | `mcp_tools_codebase_memory` | `list_projects`, `index_repository`, `index_status`, `search_code`, `search_graph`, `query_graph`, `trace_path`, `get_code_snippet`, `get_graph_schema`, `get_architecture`, `detect_changes`, `delete_project`, `manage_adr`, `ingest_traces` |
| Memoria persistente cross-session: preferencias, decisiones, hechos del user, contexto de sesiones anteriores | `mcp_tools_mem0` | `search_memories`, `add_memory`, `get_memories`, `search_graph`, `get_memory`, `update_memory`, `list_entities` |

## Nombres de tools en cada cliente

- **Claude Code / OpenCode**: `<tool_name>` directo (según cada MCP).
- **OMP**: nombres namespaced `mcp__<server>_<tool>`, p.ej.:
  - `mcp__mcp_tools_codebase_memory_get_architecture`
  - `mcp__mcp_tools_mem0_search_memories`

Si en OMP no aparecen visibles, usa `search_tool_bm25` con la capacidad como query — activará el tool discovery. Queries ejemplo:
- codebase-memory arquitectura: `codebase memory architecture repository graph`
- codebase-memory search: `codebase memory search code symbols`
- mem0 search: `mem0 search memories persistent context`
- mem0 add: `mem0 add memory remember preference`

## Reglas duras (aplicables SIEMPRE)

1. **Usa siempre el wrapper Docker** (`~/.local/bin/mcp-tools-*-docker`). Nunca llames al binario del host directamente (`codebase-memory-mcp`, `mem0-mcp-selfhosted`) para tareas normales.
2. **No bypasees el MCP con shell/docker exec** (`docker exec mcp-tools-...`, `python -c ...`) salvo debug explícito de la infra Docker.
3. **Nunca sintetices input** más grande para "probar" un MCP.
4. **Rutas absolutas** para `codebase_memory_mcp` (no `~`, resuelve `$HOME` antes).
5. **Modo de indexado** por defecto `moderate`; solo usa `full` si el user pide arquitectura completa o grafo semántico persistente.
6. **`mem0` levanta ollama + qdrant automáticamente** vía `depends_on`. Si el wrapper falla con URL error, comprueba `docker compose ps` en `~/mcp-tools`.
7. **NUNCA borres una memoria de mem0** sin confirmación explícita del user.

## Escalación si un MCP no responde

1. **`/mcp list`** para ver el estado por cliente.
2. **`/mcp reload`** o **`/mcp reconnect <server>`**.
3. Si sigue fallando: **cierra completamente el cliente y relánzalo** (`/mcp reload` no purga entradas removidas de la config).
4. Container caído: `docker compose -f ~/mcp-tools/dockers/compose.yaml --env-file ~/mcp-tools/.env up -d`.
5. Ver logs: `docker logs mcp-tools-<name> --tail 50`.

## Skills específicos por MCP

Cada MCP tiene un skill con guía detallada. Léelos con `skill://<name>` o cárgalos automáticamente cuando la intención dispare su `description` frontmatter:

- `skill://codebase-memory` — indexado, modos, workflows por tipo de pregunta, CLI fallback.
- `skill://mem0` — search-first-then-add, filtros, workflows por intención, política de borrado.
