# mcp-tools

Panel de administración web auto-hospedado para tu stack MCP (Claude Code, OpenCode, OMP). Instala, actualiza y desinstala servidores MCP, gestiona modelos Ollama y servicios Docker, y sincroniza skills/rules/mcp-config — todo desde `http://<host>:8888`, sin TUI ni SSH al día a día. La selección de componentes se persiste en `~/mcp-tools-data/state.json`.

## Instalación

```bash
# 1. Instala el binario desde la última release (v0.1.8 al 2026-07-08).
curl -fsSL https://raw.githubusercontent.com/Tutitoos/mcp-tools/main/install.sh | bash

# 2. El binario vive en ~/.local/bin por defecto. Si no está en tu $PATH,
#    añádelo antes del siguiente paso (el instalador avisa con un "warn"
#    si tu PATH no lo incluye):
export PATH="$HOME/.local/bin:$PATH"

# 3. Clona el repo e instala el panel como servicio systemd.
git clone https://github.com/Tutitoos/mcp-tools ~/mcp-tools
cd ~/mcp-tools
mcp-tools install
```

`mcp-tools install` escribe el unit file `mcp-tools-web.service` (systemd `--user` o `--system`, autodetectado), lo habilita, lo arranca y abre `http://127.0.0.1:8888/` en tu navegador. Ahí eliges qué instalar — no hay nada más que correr por CLI para dar de alta un servidor MCP.

Alternativas y overrides del paso 1:

- `MCP_TOOLS_VERSION=v0.1.7 curl -fsSL .../install.sh | bash` fija una versión concreta.
- `MCP_TOOLS_BIN=/usr/local/bin curl -fsSL .../install.sh | bash` instala en otro dir (requiere permisos).
- `go install github.com/Tutitoos/mcp-tools/cmd/mcp-tools@v0.1.8` desde source (Go 1.24+).
- `go install github.com/Tutitoos/mcp-tools/cmd/mcp-tools@latest` desde source, última release.

Si `mcp-tools install` falla con `bash: mcp-tools: command not found` justo
después del paso 1, tu shell no tiene `~/.local/bin` en `$PATH`. El paso 2
lo arregla para la sesión actual; para hacerlo permanente añade la línea
`export` a tu `~/.bashrc` / `~/.zshrc`. El instalador imprime en pantalla
el comando exacto sugerido y, desde v0.1.8, el "Siguiente paso" usa el
path absoluto (`$BIN_DIR/mcp-tools install`) para ser copy-paste safe.

### Requisitos del host

Para que las instalaciones lanzadas desde el panel (`/tools`) completen sin error,
el host debe tener disponibles antes de correr el instalador:

| Componente | Por qué | Quién lo requiere |
| --- | --- | --- |
| **Docker** + `docker compose` v2 | Orquestar `mcp_tools_ollama`, `mcp_tools_mem0_qdrant`. Requerido sólo si vas a instalar `ollama` o `qdrant` (los únicos componentes `DeployDocker`); las demás tools se instalan sin Docker. | `ollama`, `qdrant` |
| **curl** + **git** + **tar** + **sha256sum** | Descargar tarballs (install.sh), `codebase-memory-mcp` install script, rustup, etc. | install.sh, `codebase-memory` |
| **Toolchain C** + `pkg-config` + `libssl-dev` + `libsqlite3-dev` | `cargo install` compila `ring` (TLS, depende de openssl via pkg-config) y `rusqlite` (SQLite con FTS5). Sin `cc` el build falla con `error: linker 'cc' not found`. | `rtk`, `tokensave` |
| **Node ≥ 20** | `claude-mem` corre vía `npx --yes claude-mem@latest`; el panel también usa `node` para renderizar SSR (`internal/web/ssr.go`) si está disponible (opcional, cae a SPA-only sin él). | `claude-mem`, SSR del panel |
| **sudo** + acceso al package manager | Instala `nvidia-container-toolkit` (apt/dnf, llave GPG, systemctl). | `nvidia-toolkit` (opt-in) |
| **Nvidia GPU + driver propietario** | Pasar GPU al container de ollama. | `nvidia-toolkit` (opt-in) |

`mcp-tools` auto-instala lo que puede: `cargo` (rustup vía `curl \| sh`) y `uv`
(script oficial) se traen en background si faltan. Lo demás (Docker,
toolchain C, Node, sudo) son requisitos del host que el installer no
gestiona — asegúrate de tenerlos antes de instalar componentes desde el panel.

#### One-liners por distro

Debian / Ubuntu:
```bash
sudo apt-get update && sudo apt-get install -y \
  build-essential pkg-config libssl-dev libsqlite3-dev \
  curl git sudo ca-certificates

# Node ≥ 20 (sólo si vas a usar claude-mem o SSR):
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs
# Alternativa portable: nvm (https://github.com/nvm-sh/nvm)

# Docker (sólo si vas a usar ollama + qdrant):
# https://docs.docker.com/engine/install/ — post-install: sudo usermod -aG docker $USER
```

Fedora / RHEL / Rocky / Alma:
```bash
sudo dnf install -y \
  gcc make pkgconf-pkg-config openssl-devel sqlite-devel \
  curl git sudo

# Node ≥ 20:
# Fedora Modules: sudo dnf module install -y nodejs:20
# Alternativa: NodeSource (https://github.com/nodesource/distributions) o nvm

# Docker:
# sudo dnf install -y docker docker-compose-plugin
# sudo systemctl enable --now docker
```

macOS (Intel / Apple Silicon):
```bash
xcode-select --install   # provee cc, git, sha256sum, make, pkg-config (via CLT)
brew install curl sqlite openssl pkg-config node

# Docker — Docker Desktop for Mac
```

## Plataformas soportadas

Soportado:

| Tool | Linux | macOS |
| --- | --- | --- |
| codebase-memory, mem0, headroom, serena, tokensave, MongoDB MCP, Redis MCP, Docker MCP Toolkit, Sentry MCP (install/upgrade/status/uninstall desde `/tools`)¹ | ✓ | ✓ |
| claude-mem, rtk | ✓ | ✓ |
| ollama, qdrant (Docker) | ✓ | ✓ (Docker Desktop) |
| Clientes MCP (claude, codex, gemini) — instalador oficial de cada CLI | ✓ | ✓ |
| `mcp-tools` (systemd `--user`/`--system`) | ✓ (requiere systemd) | ✗ — usa `mcp-tools serve` en foreground como fallback |
| `nvidia-toolkit` (instalación) | ✓ (Debian/Ubuntu) | ✗ — el job falla con error (no hay NVIDIA en macOS) |

¹ instalables sin Docker desde `v0.1.9` (fix G1).

macOS sin systemd: `mcp-tools install` detecta la ausencia de systemd y cae a `mcp-tools serve --port <n> --bind <addr>` en foreground (ver `printNoSystemdFallback` en `internal/cli/install.go`), imprimiendo el comando para correrlo vía tu propio supervisor (launchd, tmux, etc.).

## Componentes gestionados

El registry (`internal/tools/registry.go`) tiene 18 entradas: 14 servidores/servicios MCP y 4 clientes CLI opt-in.

Servidores y servicios:

| Componente | Deploy | Registrado por | Instalador |
| --- | --- | --- | --- |
| codebase-memory-mcp | Host | `mcp-config` | `/tools` (panel) |
| mem0-mcp-selfhosted | Host | `mcp-config` | `/tools` (requiere qdrant + ollama) |
| headroom | Host | `mcp-config` | `/tools` |
| rtk | Host (hook shell) | — (hook shell) | `/tools` |
| claude-mem | Host | Se auto-registra (Claude Code) | `/tools` (opt-in; Node ≥ 20) |
| serena | Host | `mcp-config` | `/tools` (opt-in, uv tool Python 3.13) |
| tokensave | Host | Se auto-registra (Claude/OpenCode/OMP + agentes detectados) | `/tools` (opt-in; cargo install) |
| MongoDB MCP Server | Host (npm) | `mcp-config` | `/tools` (conexión directa o Atlas API) |
| Redis MCP Server | Host (uv) | `mcp-config` | `/tools` |
| Docker MCP Toolkit | Host (plugin Docker CLI) | `mcp-config` | `/tools` (requiere Docker) |
| Sentry MCP | Remote SaaS vía `mcp-remote` | `mcp-config` | `/tools` (OAuth en el primer uso) |
| ollama | Docker (+ GPU opcional) | — (infra) | `/tools` |
| qdrant | Docker | — (infra) | `/tools` |
| nvidia-container-toolkit | Sudo | — (infra) | `/tools` (sólo si hay GPU) |

Clientes MCP (binarios standalone; sólo instalan el CLI, no registran servers — eso lo hace `mcp-config sync`):

| Componente | Instalador oficial |
| --- | --- |
| claude (Claude Code CLI) | `curl -fsSL https://claude.ai/install.sh \| bash` |
| codex | instalador oficial de OpenAI |
| gemini | instalador oficial de Google |
| omp (oh-my-pi) | `npm i -g @tutitoos/oh-my-pi` |

`ollama` y `qdrant` sólo escuchan en `MCP_TOOLS_BIND` (`127.0.0.1`, loopback-only por default — tanto en `.env.example` como en el `.env` autogenerado). Cambia `MCP_TOOLS_BIND` desde `/settings` (o editando `.env`) a `0.0.0.0` si necesitas exponerlos a la LAN; ninguno de los dos tiene autenticación, así que hazlo sólo si confías en la red. El bind del panel web usa el mismo default (`internal/config/bind.go`, ver "Panel web" abajo).

Si el host tiene GPU NVIDIA **y** seleccionas `nvidia-toolkit` desde `/tools`, la instalación de `ollama` incluye `dockers/ollama-gpu-overlay.yml` para pasarle la GPU al contenedor (`internal/tools/compose.go OllamaComposeFiles`). En cualquier otro caso ollama corre en CPU.

## Panel web

`mcp-tools install` (o `mcp-tools serve` en foreground) levanta un panel React Router v7 con SSR opcional, que consume `/api/*`. **La API no tiene autenticación** — por eso el default es loopback-only (`internal/config/bind.go DefaultBind = "127.0.0.1"`) y las mutaciones cross-site desde un navegador se rechazan (gate de `Origin`/`Sec-Fetch-Site`). Para administrarlo desde otro dispositivo de tu LAN, opt-in explícito: `mcp-tools install --bind 0.0.0.0` (o `--bind` en `serve`), idealmente detrás de un firewall.

Rutas (`web/app/routes.tsx`):

| Ruta | Qué hace |
| --- | --- |
| `/` | Dashboard — conteo de tools instaladas, servicios activos, estado general. |
| `/tools` | Instala/actualiza/desinstala cada componente del registry; log de cada acción vía SSE + link a `/jobs?q=<tool>`. |
| `/configure` | Aplica un diff de selección múltiple (uninstall de dependientes + install de los nuevos) en un solo job. |
| `/models` | Multi-select de modelos Ollama (`pull` / `rm`, tag libre) **y** swap del `MEM0_LLM_MODEL` / `MEM0_EMBED_MODEL` activo (con pull opcional) en la misma vista. |
| `/services` | `docker compose ps` + up/stop/restart por servicio + logs en vivo (`GET /api/logs/{service}`, SSE). |
| `/plugins` | Plugins del workspace OMP (`plugins/`): link/unlink/enable/disable, con link a `/jobs?q=<plugin>`. |
| `/jobs` | Historial de jobs (install/upgrade/uninstall/plugin/etc.) — filtro `?q=`, streaming de log vía SSE, cancelar job en curso. In-memory, TTL 5 min (`MCP_TOOLS_JOB_TTL`). |
| `/logs` | Logs de servicios Docker en vivo. |
| `/settings` | Edita `.env` y `.env.mem0` directamente; botones "Sync skills", "Sync rules", "Re-run mcp-config". |

**Mobile layout**: abre el hamburger arriba a la izquierda para cambiar de sección en el teléfono. El header colapsa el wordmark por debajo del breakpoint `sm`; el nav se vuelve un Sheet drawer en `<md`.

## Comandos CLI (`mcp-tools`)

La CLI es deliberadamente delgada — sólo gestiona el ciclo de vida del *servicio del panel*. Todo lo demás (tools, modelos, plugins, servicios, skills/rules/mcp-config) se hace desde el panel web.

| Comando | Qué hace |
| --- | --- |
| `mcp-tools install [--port] [--bind] [--mode user\|system\|auto] [--no-open]` | Escribe el unit file `mcp-tools-web.service`, lo habilita e inicia. Abre el navegador al terminar. |
| `mcp-tools web [--enable\|--disable\|--set-port <n>\|--status\|--restart] [--mode]` | Gestión general del servicio. Sin flags: abre el navegador. |
| `mcp-tools open web [--mode]` | Alias de `mcp-tools web` (sin flags). |
| `mcp-tools serve [--port] [--bind] [--unix-socket]` | Arranca la API + el panel en foreground (lo que corre el unit systemd; también útil en hosts sin systemd). |
| `mcp-tools stop [--mode]` | Alias de `mcp-tools web --disable`. |
| `mcp-tools restart [--mode]` | Reinicia el servicio systemd. |
| `mcp-tools status-web [--mode]` | Alias de `mcp-tools web --status`. |
| `mcp-tools update [--self]` | Self-update: `git pull` + `make install` (recompila y reinstala el binario). El upgrade de tools se hace desde `/tools`. |
| `mcp-tools --version` / `-v` | Versión + commit + fecha de build. |

## Uso desde tu cliente MCP

El panel registra los servers seleccionados en Claude Code, OpenCode y OMP automáticamente al instalarlos desde `/tools` (o al re-aplicar la selección desde `/configure`); el botón "Re-run mcp-config" en `/settings` fuerza una re-sincronización manual (`POST /api/mcp-config/sync`, equivalente al viejo `mcp-tools mcp-config`). Reinicia el cliente para que aparezcan en `/mcp list`.

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

`rtk`, `claude-mem` y `tokensave` **no** son MCP registrables desde `mcp-config`: rtk es un hook shell, y `claude-mem`/`tokensave` se auto-registran ellos mismos en los IDEs/clientes correspondientes.

## Configuración

- `.env` (root del repo): 16 variables: host/runtime (`HOST_HOME`, `HOST_UID`, `HOST_GID`, `MCP_TOOLS_ROOT`, `MCP_TOOLS_DATA`, `MCP_TOOLS_BIND`), identidad mem0 (`MEM0_USER_ID`), credenciales MongoDB (`MDB_MCP_CONNECTION_STRING`, `MDB_MCP_API_CLIENT_ID`, `MDB_MCP_API_CLIENT_SECRET`) y conexión Redis (`REDIS_HOST`, `REDIS_PORT`, `REDIS_DB`, `REDIS_USERNAME`, `REDIS_PWD`, `REDIS_SSL`). Se genera/actualiza automáticamente en cada `install` o acción del panel (`internal/orchestrator.RunEnv`, corre dentro de `BootstrapEnv()`); edítalo desde `/settings`.
- `.env.mem0` (root del repo, autogenerado con defaults; se conserva si ya existe para respetar el modelo elegido desde `/models`). Editable también desde `/settings`.
- Datos persistentes: todo bajo `~/mcp-tools-data/{mem0,ollama}/` — por convención rígida. RTK, headroom, codebase-memory, claude-mem viven en `~/.cargo/bin` o `~/.local/bin` / `~/.local/share`.

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

`selected` está topo-ordenado (deps primero). `versions` se actualiza tras cada install/upgrade lanzado desde `/tools` o `/configure`.

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
MEM0_HISTORY_DB_PATH=/home/USER/mcp-tools-data/mem0/history/history.db
MEM0_OLLAMA_THINK=false
```

`MEM0_USER_ID` vive en `.env` no aquí. `MEM0_COLLECTION` se aísla por usuario para permitir varios devs en la misma qdrant. `MEM0_OLLAMA_THINK=false` evita que modelos qwen3/deepseek-r1 devuelvan bloques `<think>` que rompen el `format:"json"` que mem0 exige. `mem0-launcher` sourcea `.env.mem0` en cada llamada, así que editarlo tiene efecto sin reinicios.

## Cambiar el modelo LLM de mem0

Desde `/models`: cada fila de modelo tiene una acción para asignarlo como LLM o embed activo (`POST /api/select-model`, con `pull` automático si el tag todavía no está descargado); el mismo panel también gestiona el catálogo completo de Ollama (`pull` / `rm` por tag libre) sin tocar `.env.mem0` a menos que uses la acción de swap.

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

Para configuración avanzada por componente, ver [docs/ADVANCED.md](docs/ADVANCED.md).

## Estructura del repo

```
mcp-tools/
├── cmd/mcp-tools/          # entry point del binario Go
├── internal/
│   ├── cli/                # subcomandos cobra (install, web, serve, open, stop, restart, update, status-web)
│   ├── web/                # router HTTP, SSR, job bus (SSE), handlers /api/*
│   ├── orchestrator/       # Configure/BootstrapEnv — diffing de selección, RunEnv/RunSkills/RunRules/RunMcpConfig
│   ├── plugins/            # descubrimiento + lockfile de plugins del workspace (backs /api/plugins)
│   ├── config/             # .env / paths / RepoRoot / DataDir
│   ├── docker/             # wrapper de docker compose (con overlays GPU)
│   ├── mcp/                # registro en Claude/OpenCode/OMP (claude.go, codex.go, ...)
│   ├── state/              # $MCP_TOOLS_DATA/state.json
│   ├── systemd/            # generación/control del unit file, detección user vs system
│   ├── tools/               # registry + Install/Upgrade/Uninstall/Status por tool
│   └── version/
├── web/                     # SPA + SSR del panel (React Router v7)
│   ├── app/                 # routes/ (9 rutas), components/ui/, lib/ (api.ts, sse.ts)
│   └── scripts/check-bundle.mjs   # smoke test Playwright (layout + nav + no runtime errors)
├── webassets/               # copia de web/build embebida vía go:embed
├── webassets.go
├── plugins/mcp-tools-plugin/ # plugin OMP propio (guards + nudges), bun + su propio CI
├── dockers/
│   ├── compose.yaml
│   ├── qdrant-compose.yml
│   └── ollama-gpu-overlay.yml
├── scripts/wrappers/        # mem0-launcher
├── skills/                  # SKILL.md para clientes MCP
├── RULES.md                 # Reglas globales para clientes MCP
├── docs/ADVANCED.md
├── go.mod · Makefile · .goreleaser.yaml
├── .env.example · .env.mem0.example
└── README.md
```

## Troubleshooting

- **`missing .env`** en un wrapper → se regenera solo en la siguiente acción del panel; si necesitas forzarlo, edita cualquier campo en `/settings` y guarda.
- **`mcp_tools_mem0` no arranca** → `ls ~/.local/bin/mem0-launcher` y `mem0-mcp-selfhosted --version`; si falta uno, reinstala `mem0` desde `/tools`.
- **`compose up` falla con `MCP_TOOLS_BIND: variable is not set`** → falta `.env`; corre cualquier acción desde el panel (regenera `.env` automáticamente) o créalo a mano copiando `.env.example`.
- **`error: linker 'cc' not found` al instalar `rtk` o `tokensave`** → falta el toolchain C. Debian/Ubuntu: `sudo apt-get install -y build-essential pkg-config libssl-dev`. Fedora/RHEL: `sudo dnf install -y gcc make openssl-devel`. macOS: `xcode-select --install`.
