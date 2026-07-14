# Routing benchmarks

Datos empíricos que justifican la tabla de routing de `instructions/core.md` (distribuida como `RULES.md`). Movidos aquí desde `RULES.md` el 2026-07-13 para sacarlos del contexto always-on de los agentes; el routing en sí vive únicamente en el core.

Query representativa: encontrar refs de `auth`, leer su body, listar `*.tsx` en `tasks-pilot`. Medido por bytes de output (≈ tokens/4) y latencia.

## Datos de decisión

| Use case | Ganador (tokens) | Runner-up | Nota |
| --- | --- | --- | --- |
| Text literal (`TODO`, string exact) | **rtk grep** ~5t | native grep 0t | Fast, texto-nativo |
| Refs de símbolo (`auth`) | **serena** ~394t · LSP-accurate | rtk grep ~987t · con falsos positivos | tokensave `callers` FALLA en constantes |
| Body de un símbolo nombrado | **serena.find_symbol(include_body)** ~391t | rtk read del fichero ~1805t | 4.6× menos tokens |
| "Cómo funciona X" (pregunta open-ended) | **tokensave_context** ~1654t + call paths | serena para símbolo puntual | tokensave solo si proyecto init'd |
| Listar ficheros por patrón | **rtk find/tree** ~148t | native find ~641t | 77% ahorro |
| Leer fichero código pequeño | native Read ~1805t | rtk read = misma cifra | rtk NO ahorra aquí |
| Arquitectura / clusters | **codebase-memory get_architecture** | (único) | Sin equivalente en otros |

## Desempate serena vs tokensave vs codebase-memory

| Dimensión | serena | tokensave | codebase-memory |
| --- | --- | --- | --- |
| Precisión | LSP compiler-grade | tree-sitter estructural | tree-sitter + BM25 + embeddings |
| Scope | 1 proyecto activado | 1 proyecto init'd | N repos indexados |
| Latencia típica | 70-100ms | 5-10ms | 5-15ms |
| Requisito | `activate_project` | `.tokensave/` presente | `index_repository` corrido |
| Fuerte en | símbolos nombrados, refs, renames, edits semánticos | preguntas open-ended, call paths verbatim | arquitectura, cross-repo, comunidades, ADR |
| Débil en | preguntas open-ended amplias | refs a constantes/data (miss) | edición precisa |

Regla condensada (la que sí vive en el core): **NOMBRE → serena · PREGUNTA → tokensave (init'd) o codebase-memory · TEXTO LITERAL → rtk grep · GLOB → rtk find.**

## Ahorros rtk (referencia)

- `rtk grep`/`rtk find`/`rtk tree`: 60-77% menos tokens que los nativos con el mismo resultado.
- `rtk read`: 60-90% en logs/JSON/docker grandes; ~0% en código fuente pequeño (<300 líneas) — ahí usa `Read` nativo.
- serena vs `rtk grep` para refs: ~60% menos tokens y cero falsos positivos.
