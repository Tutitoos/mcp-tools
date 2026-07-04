---
description: mcp-tools MCP routing and hard rules — which server to use for what, and how.
alwaysApply: true
---

# mcp-tools — reglas de uso

Este fichero es cargado globalmente por Claude Code, OpenCode y OMP. Define
qué MCP de `mcp-tools` usar para cada intención y las reglas duras compartidas.

## Routing: intención → MCP

| Intención del usuario | MCP | Herramientas de entrada |
| --- | --- | --- |
| Navegar codebase, arquitectura, "dónde está X", grafo de código, refactors, trace de flujos | `mcp_tools_codebase_memory` | `get_architecture`, `search_code`, `search_graph`, `trace_path`, `get_code_snippet` |
| Recordar/recuperar memoria persistente entre sesiones (preferencias, decisiones, hechos) | `mcp_tools_mem0` | `add_memory`, `search_memories`, `get_memories`, `search_graph` |
| Comprimir texto/logs/JSON/output para reducir tokens; recuperar contenido comprimido por hash | `mcp_tools_headroom` | `compress`, `retrieve`, `stats` |

Si la intención es ambigua (p.ej. "compress and index this repo"), primero clarifica con el usuario antes de invocar varios MCPs.

## Nombres de herramientas en cada cliente

- **Claude Code / OpenCode**: `<tool_name>` directo (según cada MCP).
- **OMP**: nombres namespaced `mcp__<server>_<tool>`, p.ej.:
  - `mcp__mcp_tools_codebase_memory_get_architecture`
  - `mcp__mcp_tools_mem0_add_memory`
  - `mcp__mcp_tools_headroom_compress`

Si en OMP no aparecen visibles, usa `search_tool_bm25` con la capacidad ("headroom compress", "codebase memory architecture", "mem0 search memories") — activarán el tool discovery.

## Reglas duras (aplicables SIEMPRE)

1. **Usa siempre el wrapper Docker** (`~/.local/bin/mcp-tools-*-docker`). Nunca llames al binario del host directamente (`codebase-memory-mcp`, `headroom`, `mem0-mcp-selfhosted`) para tareas normales.
2. **No bypasees el MCP con shell/docker exec** (`docker exec mcp-tools-...`, `python -c ...`) salvo para debug explícito de la infra Docker.
3. **Nunca sintetices input** más grande para "probar" un MCP (aplica sobre todo a headroom).
4. **Rutas absolutas** para `codebase_memory_mcp` (no `~`, resuelve `$HOME` antes).
5. **Modo de indexado** por defecto `moderate`; solo usa `full` si el usuario pide arquitectura completa o grafo semántico persistente.
6. **`mem0` levanta ollama + qdrant automáticamente** vía `depends_on` (`mcp-tools-ollama` en `:11434` y `mcp-tools-mem0-qdrant` en `:6333`). Si el wrapper falla con URL error, comprueba `docker compose ps` en `~/mcp-tools`.

## Escalación si un MCP no responde

1. **`/mcp list`** para ver el estado por cliente.
2. **`/mcp reload`** o **`/mcp reconnect <server>`**.
3. Si sigue fallando: **cierra completamente el cliente y relánzalo** (`/mcp reload` no purga entradas removidas de la config).
4. Container caído: `docker compose -f ~/mcp-tools/compose.yaml up -d`.
5. Ver logs: `docker logs mcp-tools-<name> --tail 50`.

## Herramientas específicas de skills

Para cada MCP existe un skill con la guía detallada:
- `skill://codebase-memory` — modos de indexado, workflow de search, verificación
- `skill://headroom` — cuándo comprime bien, cuándo pasa passthrough, `hash` para retrieve

Léelos con `skill://<name>` en OMP o cárgalos automáticamente cuando la intención dispare su frontmatter (`description`).
