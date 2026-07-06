# mcp-tools

Instalador declarativo de MCPs, herramientas host y servicios Docker para Claude Code, OpenCode y OMP. La selección de componentes vive en un multi-select TUI y se persiste en `~/mcp-tools-data/state.json` para que `install` / `update` / `configure` / `uninstall` operen sobre el mismo conjunto.

## Instalación

```bash
curl -fsSL https://raw.githubusercontent.com/Tutitoos/mcp-tools/main/install.sh | bash

git clone https://github.com/Tutitoos/mcp-tools ~/mcp-tools
cd ~/mcp-tools
mcp-tools install
```

`mcp-tools install` corre `env` (genera `.env` + `.env.mem0` + directorios de datos), abre un multi-select TUI con la lista de componentes disponibles y — al confirmar — instala cada tool en orden topológico. La selección se guarda en `state.json`; próximas ejecuciones sin flags reusan el mismo set. Añade `--dry` para preview, `--noselect` para reusar el state sin abrir el TUI, o `--reconfigure` para forzar el TUI.

Alternativas para bajar el binario:

- `MCP_TOOLS_VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/Tutitoos/mcp-tools/main/install.sh | bash` fija una versión concreta.
- `MCP_TOOLS_BIN=/usr/local/bin` instala en otro dir (requiere permisos).
- `go install github.com/Tutitoos/mcp-tools/cmd/mcp-tools@latest` desde source (Go 1.24+).

### Requisitos

- Docker + `docker compose` v2.
- `~/.local/bin` en `$PATH` (donde vive el binario y `mem0-launcher`).
- `git` en PATH (para `update --self`).
- Nvidia GPU + driver instalado si vas a marcar `nvidia-toolkit` (opcional). En hosts sin GPU la fila no aparece en el TUI.

## Plataformas soportadas

Soportado: Linux (x86_64 / aarch64) y macOS (Intel / Apple Silicon). Windows no está soportado — `install.sh` rechaza cualquier OS distinto de `linux`/`darwin`.

Diferencias entre plataformas:

| Tool | Linux | macOS |
| --- | --- | --- |
| codebase-memory, mem0, headroom, serena, tokensave (install/upgrade/status/uninstall) | ✓ | ✓ |
| claude-mem, codegraph, rtk | ✓ | ✓ |
| ollama, qdrant (Docker) | ✓ | ✓ (Docker Desktop) |
| `mcp-tools tokensave cap` / `uncap` | ✓ | ✗ — requiere `systemd-run`, devuelve error |
| `mcp-tools nvidia-toolkit install` | ✓ (Debian/Ubuntu/RHEL/Fedora/CentOS/Rocky/Alma) | ✗ — el row no aparece en el TUI y el CLI directo devuelve error |

## Componentes gestionados

Once componentes vienen preconfigurados en el registry:

| Componente | Deploy | Registrado por | Instalador |
| --- | --- | --- | --- |
| codebase-memory-mcp | Host | `mcp-config` | `mcp-tools install` (o `mcp-tools codebase-memory install`) |
| mem0-mcp-selfhosted | Host | `mcp-config` | `mcp-tools install` (requiere qdrant + ollama) |
| headroom | Host | `mcp-config` | `mcp-tools install` |
| rtk | Host (hook shell) | — (hook shell) | `mcp-tools install` |
| claude-mem | Host | Se auto-registra (Claude Code) | `mcp-tools install` (opt-in; Node ≥ 20) |
| codegraph | Host | Se auto-registra (8 IDEs) | `mcp-tools install` (opt-in) |
| serena | Host | `mcp-config` | `mcp-tools install` (o `mcp-tools serena install`; opt-in, uv tool Python 3.13) |
| tokensave | Host | Se auto-registra (Claude/OpenCode/OMP + agentes detectados) | `mcp-tools install` (opt-in; cargo install) |
| ollama | Docker (+ GPU opcional) | — (infra) | `mcp-tools install` |
| qdrant | Docker | — (infra) | `mcp-tools install` |
| nvidia-container-toolkit | Sudo | — (infra) | `mcp-tools install` (sólo si hay GPU) |

`ollama` y `qdrant` se exponen por defecto en todas las interfaces del host (`MCP_TOOLS_BIND=0.0.0.0`). Cambia el valor a `127.0.0.1` en `.env` para bindear sólo a loopback. Ninguno tiene autenticación por default — el user es responsable de firewall y segmentación.

Si el host tiene GPU NVIDIA **y** marcas `nvidia-toolkit` en el TUI, `mcp-tools up` incluye `dockers/ollama-gpu-overlay.yml` para pasarle la GPU al contenedor de ollama. En cualquier otro caso ollama corre en CPU.

## Uso desde tu cliente MCP

`mcp-tools install` registra los servers seleccionados en Claude Code, OpenCode y OMP. Reinicia el cliente para que aparezcan en `/mcp list`.

- **Verificar Claude Code**: `claude mcp list` — debe listar los servers como `✔ Connected`.
- **Otro cliente MCP** (Codex, Cursor, etc.): añade este bloque a la config del cliente (ajusta `<USUARIO>` y quita las entradas de tools no seleccionados):
  ```json
  {
    "mcp_tools_codebase_memory": {
      "type": "stdio",
      "command": "codebase-memory-mcp",
      "args": ["--ui=true"]
    },
    "mcp_tools_mem0": {
      "type": "stdio",
      "command": "/home/<USUARIO>/.local/bin/mem0-launcher"
    },
    "mcp_tools_headroom": {
      "type": "stdio",
      "command": "headroom",
      "args": ["mcp", "serve"]
    },
    "mcp_tools_serena": {
      "type": "stdio",
      "command": "serena",
      "args": ["start-mcp-server", "--context", "agent", "--project-from-cwd"]
    }
  }
  ```

`rtk`, `claude-mem`, `codegraph` y `tokensave` **no** son MCP registrables desde `mcp-config`: rtk es un hook shell, y `claude-mem`/`codegraph`/`tokensave` se auto-registran ellos mismos en los IDEs/clientes correspondientes.

## Configuración

- `.env` (root del repo, generado por `mcp-tools env`): `HOST_HOME`, `HOST_UID`, `HOST_GID`, `MCP_TOOLS_ROOT`, `MCP_TOOLS_DATA`, `MCP_TOOLS_BIND`, `MEM0_USER_ID`. 7 vars en total.
- `.env.mem0` (root del repo, autogenerado por `mcp-tools env` con defaults; se conserva si ya existe para respetar cambios de `mcp-tools select-model`).
- Datos persistentes: todo bajo `~/mcp-tools-data/{mem0,ollama}/` — por convención rígida. RTK, headroom, codebase-memory, claude-mem, codegraph viven en `~/.cargo/bin` o `~/.local/bin` / `~/.local/share`.

### Estado persistente

`~/mcp-tools-data/state.json` (schema v1):

```json
{
  "version": 1,
  "selected": ["qdrant", "ollama", "codebase-memory", "mem0", "headroom", "rtk"],
  "versions": {
    "codebase-memory": "codebase-memory-mcp 0.5.0",
    "mem0": "mem0-mcp-selfhosted 0.2.1"
  },
  "updated_at": "2026-07-05T20:00:00Z"
}
```

`selected` está topo-ordenado (deps primero). `versions` se actualiza tras cada `install` / `update`.

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
MEM0_HISTORY_DB_PATH=$HOME/mcp-tools-data/mem0/history/history.db
```

`MEM0_USER_ID` vive en `.env` no aquí. `MEM0_COLLECTION` se aísla por usuario para permitir varios devs en la misma qdrant. `mem0-launcher` sourcea `.env.mem0` en cada llamada, así que editarlo tiene efecto sin reinicios.

## Cambiar el modelo LLM de mem0

`mcp-tools select-model` es un TUI **single-select** que edita `.env.mem0` (LLM o embed) y hace `ollama pull` del tag elegido. `mcp-tools models` es un TUI **multi-select** que gestiona el catálogo de modelos Ollama (pull + rm) sin tocar `.env.mem0`. Verbos ortogonales.

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

Embeddings (`MEM0_EMBED_MODEL`), tag `embedding` en el catálogo:

| Tag Ollama | Notas |
| --- | --- |
| `bge-m3` | **Default**. Multilingüe (100+ idiomas), 1024 dims. |
| `mxbai-embed-large` | mixedbread.ai. Verificar dim con `ollama show`. |
| `snowflake-arctic-embed` | Familia Snowflake, varias variantes. |
| `nomic-embed-text` | Contexto largo. Verificar dim. |
| `all-minilm` | Mínimo (22m/33m params). Solo pruebas. |

Modelos qwen3/deepseek-r1 requieren `MEM0_OLLAMA_THINK=false` (default) para evitar colisión `<think>` + `format:"json"`.

## Comandos comunes

| Comando | Qué hace |
| --- | --- |
| `mcp-tools install [--dry] [--reconfigure] [--noselect]` | Multi-select + instala componentes. |
| `mcp-tools configure [--dry]` | Reabre el TUI y aplica el diff (uninstall dependents + install nuevos). |
| `mcp-tools update [--self] [--tools] [--dry]` | Actualiza mcp-tools (git pull + make install) y/o los componentes. |
| `mcp-tools uninstall <tool> [--dry] [--force]` | Quita un componente respetando reverse-deps. |
| `mcp-tools status [--table]` | Estado de todos los componentes (JSON por default). |
| `mcp-tools <tool> install/upgrade/status/uninstall` | Control per-tool granular. |
| `mcp-tools models [list/pull/rm]` | Multi-select TUI de modelos Ollama (o CLI no-interactiva). |
| `mcp-tools select-model` | Selector TUI de `MEM0_LLM_MODEL` / `MEM0_EMBED_MODEL`. |
| `mcp-tools up` / `stop` / `ps` / `logs <svc>` / `restart <svc>` | Docker lifecycle (ollama + qdrant). |
| `mcp-tools env [--force]` | (Re)genera `.env`. |
| `mcp-tools mcp-config` | Re-registra en Claude/OpenCode/OMP según el state actual. |
| `mcp-tools skills` / `rules` | Symlinks de skills y RULES a los 3 clientes. |
| `mcp-tools pull <tag>` | Alias corto de `models pull`. |
| `mcp-tools tokensave cap` / `uncap` | Envuelve/restaura el MCP `tokensave` en un cgroup con `MemoryMax=30G` en los clients MCP (idempotente; re-correr tras cada `tokensave install`/`upgrade`). |
| `mcp-tools tokens` / `tokens set <n>` | Lee/edita `compaction.thresholdTokens` de OMP (requiere `omp` en PATH). |

Para configuración avanzada por componente y la migración desde el pipeline viejo, ver [docs/ADVANCED.md](docs/ADVANCED.md).

## Estructura del repo

```
mcp-tools/
├── cmd/mcp-tools/          # entry point del binario Go
├── internal/
│   ├── cli/                # subcomandos cobra (install, configure, update, uninstall, status, models, per-tool, docker lifecycle)
│   ├── config/             # .env / paths
│   ├── docker/             # wrapper de docker compose (con overlays)
│   ├── mcp/                # registro en Claude/OpenCode/OMP
│   ├── state/              # $MCP_TOOLS_DATA/state.json
│   ├── tools/              # registry + Install/Upgrade/Uninstall/Status por tool
│   ├── tui/                # bubbletea (installer progress, toolselect, modelselect, selectmodel)
│   └── version/
├── dockers/
│   ├── compose.yaml
│   ├── qdrant-compose.yml
│   └── ollama-gpu-overlay.yml
├── scripts/wrappers/       # mem0-launcher
├── skills/                 # SKILL.md para clientes MCP
├── RULES.md                # Reglas globales para clientes MCP
├── docs/ADVANCED.md
├── go.mod · Makefile · .goreleaser.yaml
├── .env.example
└── README.md
```

## Troubleshooting

- **`missing .env`** en un wrapper → `mcp-tools env`.
- **`mcp_tools_mem0` no arranca** → `ls ~/.local/bin/mem0-launcher` y `mem0-mcp-selfhosted --version`; si falta uno, `mcp-tools mem0 install`.
- **`Failed to connect to url`** en el cliente MCP tras `/mcp list` → revisa configs residuales en `~/.claude/plugins/marketplaces/`, `~/.codex/config.toml`, o entradas viejas sin prefijo `mcp_tools_` en el cliente.
- **Ollama sin GPU aunque tengo `nvidia-toolkit` seleccionado** → verifica que `nvidia-smi -L` pasa, luego `mcp-tools restart mcp_tools_ollama`.
- **`state.json` corrupto** → bórralo y corre `mcp-tools install` (te abrirá el TUI de cero).
- **`compose up` falla con `MCP_TOOLS_BIND: variable is not set`** → corre `mcp-tools env` para regenerar `.env` con la key.
