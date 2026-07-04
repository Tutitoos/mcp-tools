# mcp-tools

Stack de MCP servers self-hosted en Docker para Claude Code, OpenCode y OMP.
Incluye memoria persistente (mem0), grafo de código (codebase-memory) y compresión de contexto (headroom), más Ollama y Qdrant compartidos.

## Instalación

```bash
curl -fsSL https://raw.githubusercontent.com/Tutitoos/mcp-tools/main/install.sh | bash

git clone https://github.com/Tutitoos/mcp-tools ~/mcp-tools
cd ~/mcp-tools
mcp-tools install
```

El primer comando descarga el binario `mcp-tools` a `~/.local/bin/` (detecta OS/arch, resuelve la latest release). El tercero lanza el TUI 10-pasos: prereq → `.env` → verificación de fuente mem0 → build → wrappers → skills → RULES → registro MCP en Claude/OpenCode/OMP → arranque de contenedores → smoke test.

Idempotente. Añade `--dry` a `mcp-tools install` para preview sin ejecutar nada. Alternativas para el binario:

- `MCP_TOOLS_VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/Tutitoos/mcp-tools/main/install.sh | bash` fija una versión concreta.
- `MCP_TOOLS_BIN=/usr/local/bin` instala en otro dir (requiere permisos).
- `go install github.com/Tutitoos/mcp-tools/cmd/mcp-tools@latest` desde source (requiere Go 1.22+).

### Requisitos

- Docker + `docker compose` v2.
- `~/.local/bin` en `$PATH` (donde vive el binario y los wrappers).
- Para `mcp_tools_mem0`: clon de `elvismdev/mem0-mcp-selfhosted` en `~/mcp-tools-data/mem0/src`. El paso `mem0-src` del installer falla con el `git clone` exacto si falta.

## Servicios

| Servicio | Puerto | Propósito |
| --- | --- | --- |
| `mcp_tools_codebase_memory` | — | Grafo de código y búsqueda semántica sobre repos locales. |
| `mcp_tools_mem0` | — | Memoria persistente self-hosted (mem0-mcp-selfhosted). |
| `mcp_tools_ollama` | `127.0.0.1:11434` | Ollama compartido (LLM + embeddings) para mem0 y futuros MCPs. |
| `mcp_tools_mem0_qdrant` | `127.0.0.1:6333` | Vector store de mem0 (qdrant v1.12.0). |
| `mcp_tools_headroom` | — | Compresión de texto/logs para reducir tokens. |

Los 3 primeros exponen tools MCP vía wrappers stdio en `~/.local/bin`; ollama y qdrant son infra compartida (HTTP en loopback).

## Uso desde tu cliente MCP

El installer registra los servers automáticamente en Claude Code, OpenCode y OMP. Reinicia el cliente y aparecerán en `/mcp list`.

- **Verificar Claude Code**: `claude mcp list` — debe listar los 3 como `✔ Connected`.
- **Otro cliente MCP** (Codex, Cursor, etc.): añade este bloque a la config del cliente (ajusta `<USUARIO>`):
  ```json
  {
    "mcp_tools_codebase_memory": {
      "type": "stdio",
      "command": "/home/<USUARIO>/.local/bin/mcp-tools-codebase-memory-docker",
      "args": ["--ui=false"]
    },
    "mcp_tools_mem0": {
      "type": "stdio",
      "command": "/home/<USUARIO>/.local/bin/mcp-tools-mem0-docker"
    },
    "mcp_tools_headroom": {
      "type": "stdio",
      "command": "/home/<USUARIO>/.local/bin/mcp-tools-headroom-docker"
    }
  }
  ```

## Configuración

- `.env` (root del repo, generado por `mcp-tools env`): `HOST_HOME`, `HOST_UID`, `HOST_GID`, `MCP_TOOLS_ROOT`, `MCP_TOOLS_DATA`, `MEM0_SRC_PATH`, `MEM0_USER_ID`, tags de imagen.
- `.env.mem0` (root del repo, NO se autogenera): configuración de mem0. Copiar del bloque de abajo.
- Datos persistentes: todo bajo `~/mcp-tools-data/{codebase-memory,mem0,headroom,ollama}/` — por convención rígida.

### `.env.mem0`

```
MEM0_PROVIDER=ollama
MEM0_LLM_MODEL=qwen2.5:7b
MEM0_EMBED_PROVIDER=ollama
MEM0_EMBED_MODEL=bge-m3
MEM0_OLLAMA_URL=http://127.0.0.1:11434/
MEM0_QDRANT_URL=http://127.0.0.1:6333/
MEM0_COLLECTION=mem0_<username>
MEM0_ENABLE_GRAPH=false
MEM0_HISTORY_DB_PATH=/data/history/history.db
```

`MEM0_USER_ID` vive en `.env` no aquí. `MEM0_COLLECTION` se aísla por usuario para permitir varios devs en la misma qdrant.

## Cambiar el modelo LLM de mem0

mem0 usa el LLM para extraer memorias (function-calling). El LLM DEBE tener tag `tools` en https://ollama.com/library. `.env.mem0` trae `qwen2.5:7b` por defecto.

| Tag Ollama | Params | Notas |
| --- | --- | --- |
| `qwen2.5:7b` | 7B | **Default `.env.mem0`**. Multilingüe, tool calling maduro. |
| `qwen3:8b` | 8B | Generación siguiente de qwen; mejor calidad a coste similar. |
| `mistral-nemo:12b` | 12B | Mistral+NVIDIA, contexto 128k, multilingüe. |
| `llama3.1:8b` | 8B | Meta, menos multilingüe que qwen. |
| `mistral:7b` | 7B | Function calling desde v0.3. |
| `qwen3:4b` | 4B | Compacto dentro de qwen3. |
| `qwen2.5:3b` | 3B | Ligero dentro de qwen2.5. |
| `llama3.2:3b` | 3B | Meta ligero. |
| `granite3.1-moe:3b` | 3B (MoE) | IBM mixture-of-experts, punchea por encima. |
| `smollm2:1.7b` | 1.7B | Mínimo viable, solo para probar. |

Embeddings (`MEM0_EMBED_MODEL`), tag `embedding` en el catálogo. La dimensión debe coincidir con `MEM0_EMBED_DIMS` y con la colección qdrant existente:

| Tag Ollama | Notas |
| --- | --- |
| `bge-m3` | **Default**. Multilingüe (100+ idiomas), 1024 dims. |
| `mxbai-embed-large` | mixedbread.ai. Verificar dim con `ollama show`. |
| `snowflake-arctic-embed` | Familia Snowflake, varias variantes por tamaño. |
| `nomic-embed-text` | Contexto largo. Verificar dim antes de fijar `MEM0_EMBED_DIMS`. |
| `all-minilm` | Mínimo (22m/33m params). Solo pruebas. |

Cambio interactivo con selector (recomendado):

```bash
mcp-tools select-model
```

Elige LLM o Embed → selecciona modelo → confirma. Hace `ollama pull`, edita `.env.mem0` (incluye `MEM0_OLLAMA_THINK=false` automáticamente si eliges qwen3/deepseek-r1), y recrea `mcp-tools-mem0`.

Cambio manual:

1. `mcp-tools pull <tag>`.
2. Editar `.env.mem0`: `MEM0_LLM_MODEL=<tag>` (o `MEM0_EMBED_MODEL=<tag>`).
3. Si cambia dimensión de embeddings: cambiar `MEM0_COLLECTION` a nombre nuevo, o `curl -X DELETE http://127.0.0.1:6333/collections/<nombre>`.
4. `mcp-tools restart mcp_tools_mem0`.

Modelos qwen3/deepseek-r1 requieren `MEM0_OLLAMA_THINK=false` (default) para evitar colisión `<think>` + `format:"json"`.

## Comandos comunes

| Comando | Qué hace |
| --- | --- |
| `mcp-tools install [--dry]` | Instalador TUI end-to-end. Idempotente. |
| `mcp-tools up` | Levanta los 5 contenedores. |
| `mcp-tools stop` | Para los 5 contenedores (mantiene volúmenes). |
| `mcp-tools build` | Reconstruye las imágenes locales tras editar Dockerfiles. |
| `mcp-tools env` | (Re)genera `.env` si no existe. Idempotente. |
| `mcp-tools select-model` | Selector TUI para cambiar `MEM0_LLM_MODEL` / `MEM0_EMBED_MODEL`, hacer `ollama pull` y recrear mem0. |
| `mcp-tools mcp-config` | Re-registra los servers en Claude/OpenCode/OMP. |
| `mcp-tools skills` | Symlinks de skills a los 3 clientes. |
| `mcp-tools rules` | Instala `RULES.md` en los 3 clientes. |
| `mcp-tools ps` | Estado de los 5 contenedores. |
| `mcp-tools logs <svc> [--follow]` | Muestra logs de un servicio. |
| `mcp-tools restart <svc>` | Recrea un servicio releyendo `.env` / `.env.mem0`. |
| `mcp-tools pull <tag>` | Descarga un modelo Ollama. |

Para configuración avanzada por servicio, migraciones desde el layout previo o cómo añadir un servicio nuevo, ver [docs/ADVANCED.md](docs/ADVANCED.md).

## Estructura del repo

```
mcp-tools/
├── cmd/mcp-tools/         # entry point del binario Go
├── internal/
│   ├── cli/               # subcomandos cobra
│   ├── config/            # .env / paths
│   ├── docker/            # wrapper de docker compose
│   ├── mcp/               # registro en Claude/OpenCode/OMP
│   ├── tui/               # bubbletea TUIs (installer + select-model)
│   └── version/
├── dockers/
│   ├── compose.yaml
│   └── mem0-compose.yml
├── mcps/
│   ├── codebase-memory/Dockerfile
│   ├── mem0/Dockerfile
│   └── headroom/Dockerfile
├── scripts/wrappers/      # mcp-tools-*-docker (los invoca el cliente MCP)
├── skills/                # SKILL.md para clientes MCP
├── RULES.md               # Reglas globales para clientes MCP
├── docs/ADVANCED.md
├── go.mod · Makefile · .goreleaser.yaml
├── .env.example
└── README.md
```

## Troubleshooting

- **`missing .env`** en un wrapper → `mcp-tools env`.
- **`MEM0_SRC_PATH does not exist`** → `git clone https://github.com/elvismdev/mem0-mcp-selfhosted ~/mcp-tools-data/mem0/src`.
- **`Failed to connect to url`** en el cliente MCP tras `/mcp list` → revisa configs residuales en `~/.claude/plugins/marketplaces/`, `~/.codex/config.toml`, o entradas viejas sin prefijo `mcp_tools_` en el cliente.
- **`mcp_tools_mem0` no conecta con Ollama/Qdrant** → `mcp-tools ps` para ver estado; qdrant debe estar `Up (healthy)`.
- **Rebuild tras cambiar Dockerfile** → `mcp-tools build && mcp-tools restart <servicio>`.
