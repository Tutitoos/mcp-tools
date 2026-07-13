---
description: mcp-tools MCP routing and hard rules — which server to use for what, and how to pick between MCP and native tools.
alwaysApply: true
---

# mcp-tools — reglas de uso

Cargado globalmente por Claude Code, OpenCode y OMP. **Única fuente de la tabla de routing**: los skills (`skill://<name>`) documentan cada herramienta en detalle pero NO redefinen el routing. Benchmarks que justifican estas reglas: `docs/ROUTING-BENCHMARKS.md` en el repo mcp-tools.

## Routing (first match wins)

| # | Intención | Herramienta |
| --- | --- | --- |
| 0 | Operación sobre un SÍMBOLO nombrado — cuerpo, refs, declaración, outline de fichero, rename, edit semántico. Incluye "cómo funciona X" / "muéstrame el cuerpo" / "dónde se usa X" | `mcp_tools_serena` (SIEMPRE tras `activate_project("/abs/path")`, una vez por sesión) |
| 1 | Memoria persistente cross-session: recordar, guardar, recuperar decisiones/preferencias/hechos | `mcp_tools_mem0` (ver Known bugs abajo) |
| 2 | Pregunta natural-language sobre cómo funciona una zona del código, en proyecto con `tokensave init` | `tokensave` (`tokensave_context`: entry points + call paths en 1 call). Sin índice → fila 3 |
| 3 | Arquitectura, cross-repo, comunidades, ADR, o repo no indexado por tokensave | `mcp_tools_codebase_memory` (único con `get_architecture` y `manage_adr`) |
| 4 | Texto LITERAL en ficheros: strings exactos, comments, config, `TODO` | `rtk grep` (ficheros por patrón → `rtk find`; árbol → `rtk tree`) |
| 5 | Fichero muy grande: log >500 líneas, JSON verboso, output docker | `rtk read` (NO en código fuente pequeño — 0% de ahorro ahí) |
| 6 | Resto: UN fichero pequeño ya nombrado por path, edit puntual, correr un test, shell | nativas (`Read`, `edit`, `bash`) |

Desempate: el target es un **NOMBRE** → serena · una **PREGUNTA** → tokensave (init'd) o codebase-memory · **TEXTO LITERAL** → rtk grep · un **GLOB** → rtk find. Si aplican varias filas, en orden y fusiona resultados — no te saltes serena porque otra también encaje.

## Prohibiciones

- PROHIBIDO `Read`/`Grep`/`rtk grep` para entender código o buscar refs de un símbolo nombrado — eso es la fila 0 (serena). `Read` sobre código solo cuando el user nombró el fichero por path y quiere el contenido raw.
- PROHIBIDO `rtk grep` para refs de símbolo: trae falsos positivos (strings/comments) y más tokens que serena. Solo texto literal (fila 4).
- PROHIBIDO el fallback a herramientas nativas para símbolos, arquitectura o memoria persistente salvo que el MCP haya devuelto error explícito en ESTA sesión (ver Escalación).
- PROHIBIDO persistir hechos en notas locales (`notes.md`, scratchpad) — eso es mem0. `serena.write_memory` y `tokensave_todos` son scratchpads per-proyecto que mueren con el índice, no memoria.
- PROHIBIDO responder de memoria sobre decisiones/preferencias previas sin consultar mem0 primero (con su workaround si el bug aplica).
- MCP no visible en el cliente → `search_tool_bm25` con la capacidad como query (activa el discovery en OMP). Fallback a CLI solo después.

## Reglas duras

1. Los MCP corren como binarios HOST, no Docker: `codebase-memory-mcp` y `mem0-launcher` y `serena` en `~/.local/bin`, `tokensave` en `~/.cargo/bin`. Usa los wrappers; nunca los binarios crudos con flags manuales.
2. NUNCA llames a una tool de serena sin `activate_project` previa en la misma sesión.
3. NUNCA `tokensave serve` en un cwd sin `.tokensave/` — falla al arrancar. Proyecto sin indexar → avisa y ofrece `tokensave init` (side effect: re-registra su MCP en todos los agentes que encuentra, menos OMP; ver `skill://tokensave` §Blast radius).
4. No bypasees un MCP con shell/docker exec salvo debug explícito de la infra Docker.
5. Nunca sintetices input más grande para "probar" un MCP.
6. Rutas absolutas para codebase-memory, `serena.activate_project` y tokensave — nunca `~`, resuelve `$HOME` antes.
7. Indexado de codebase-memory: `moderate` por defecto; `full` solo si piden arquitectura completa o grafo semántico persistente.
8. `mem0` NO levanta ollama+qdrant por sí mismo: `mem0-launcher` solo exporta `.env.mem0` y ejecuta el binario. Los servicios Docker se levantan desde el panel web (`/services`) o `docker compose -f ~/mcp-tools/dockers/compose.yaml --env-file ~/mcp-tools/.env up -d`.
9. NUNCA borres una memoria de mem0 sin confirmación explícita del user.

## Known bugs — mem0 (firma corta)

`search_memories(query)` y `get_memories(user_id)` fallan upstream con `ValueError: Top-level entity parameters frozenset({'user_id'}) are not supported in search()` — pasar `filters={"user_id": ...}` NO lo evita (el MCP inyecta el top-level igual; re-verificado 2026-07-13). Workaround: `list_entities` + `get_memory(uuid)`; guardar → `add_memory` directo asumiendo riesgo de duplicado. Estado degradado `Memory not initialized` en TODAS las ops → reinicio del proceso. Detalle completo y comandos: `skill://mem0` §Known state.

## Nombres de tools por cliente

- **Claude Code / OpenCode**: nombre directo por MCP; tokensave usa naming bare (`tokensave_context`, sin prefijo `mcp_tools_`).
- **OMP**: namespaced — `mcp__mcp_tools_serena_find_symbol`, `mcp__mcp_tools_codebase_memory_get_architecture`, `mcp__tokensave_context`; OMP recorta `mem0`→`mem`: `mcp__mcp_tools_mem_search_memories`. Si no aparecen, `search_tool_bm25` con la capacidad como query (p.ej. `serena find symbol activate project`, `mem0 add memory remember preference`).

## Escalación si un MCP no responde

1. **Serena primero** — prioridad sobre cualquier otra acción: instala si falta (`uv tool install -p 3.13 serena-agent`, o panel `/tools` → serena → install), `which serena`, re-registra los MCP en clientes (panel `/settings` → "Re-run mcp-config" = `POST /api/mcp-config/sync`; el CLI ya NO tiene `mcp-config`), `search_tool_bm25`. Tras 3 intentos → tokensave o codebase-memory; NUNCA `Read`/`Grep` para entender código.
2. `/mcp list` → `/mcp reload` o `/mcp reconnect <server>`; si persiste, cierra el cliente del todo y relánzalo.
3. Contenedores caídos (`mcp_tools_mem0_qdrant`, `mcp_tools_ollama`): `docker compose -f ~/mcp-tools/dockers/compose.yaml --env-file ~/mcp-tools/.env up -d`. Logs: mismo compose con `logs --tail 50 <service-key>`, o panel `/services` (el CLI ya NO tiene `logs`). Nombre de contenedor real para `docker exec`: `mcp-tools-mem0-qdrant`, `mcp-tools-ollama`.
4. Escalación específica de cada herramienta: sección "Escalation" de su skill.

## Skills

Guía detallada por MCP — se cargan por trigger o con `skill://<name>`: `skill://codebase-memory` (indexado, modos, workflows), `skill://mem0` (search-first-then-add, known bugs, borrado), `skill://serena` (activate_project, find/references/rename), `skill://tokensave` (context queries, init, blast radius).
