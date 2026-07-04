# mcp-custom

Colección de servidores MCP (Model Context Protocol) empaquetados en Docker, orquestados con `docker compose`. Cada servidor corre en un contenedor persistente que hace `sleep infinity`; los clientes MCP se conectan vía wrappers `docker exec -i` instalados en `~/.local/bin`. Diseñado para uso local con Claude Code / Claude Desktop, OpenCode y OMP.

## Servicios incluidos

| Servicio | Imagen | Contenedor | Propósito |
| --- | --- | --- | --- |
| `codebase_memory_mcp` | `mcp-custom/codebase-memory-mcp:latest` | `mcp-custom-codebase-memory-mcp` | Grafo de conocimiento y búsqueda de código sobre repos locales |
| `mem0` | `mcp-custom/mem0-mcp:latest` | `mcp-custom-mem0-mcp` | Memoria persistente self-hosted (mem0) con Ollama + Qdrant |
| `headroom` | `mcp-custom/headroom-mcp:latest` | `mcp-custom-headroom-mcp` | Compresión de texto/logs para reducir tokens |

## Requisitos

- Docker + `docker compose` v2.
- Linux (probado en kernel 7.0.12-1-pve, Debian/Ubuntu compatible).
- `~/.local/bin` en `$PATH` para los wrappers.
- Para `mem0`: instancia local de **Ollama** en `http://127.0.0.1:11434` y **Qdrant** en `http://127.0.0.1:6333` (por eso `network_mode: host`). Además el código fuente de mem0 clonado en `MEM0_SRC_PATH` (por defecto `$HOME/containers/mem0/mem0-src`) porque el wrapper hace `uvx --from /opt/mem0-src mem0-mcp-selfhosted`.
- Para `codebase_memory_mcp`: nada extra (el binario se instala dentro de la imagen desde `raw.githubusercontent.com/DeusData/codebase-memory-mcp`).
- Para `headroom`: nada extra (`pip install headroom-ai[mcp,proxy]` dentro de la imagen).

## Arranque rápido

```bash
git clone <repo-url> ~/mcp-custom
cd ~/mcp-custom

# 1. Genera .env desde el host (UID/GID/HOME) y crea ~/mcp-custom-data
./scripts/init-env.sh

# 2. (solo mem0) crea .env.mem0 con el bloque de la sección "mem0" más abajo
#    (no existe plantilla; .env.example no incluye las variables de mem0)

# 3. Construye las imágenes
./scripts/build.sh

# 4. Arranca los contenedores persistentes
./scripts/up.sh                                # solo codebase_memory_mcp
docker compose up -d mem0 headroom              # el resto, si los quieres
```

`up.sh` solo levanta `codebase_memory_mcp`; para `mem0` y `headroom` usa `docker compose up -d <servicio>` o deja que el wrapper los levante on-demand (los tres wrappers hacen `docker compose up -d` si el contenedor no existe).

## Instalar los wrappers

Los wrappers viven en `scripts/wrappers/` y deben quedar accesibles como `~/.local/bin/<nombre>`:

```bash
mkdir -p ~/.local/bin
ln -sf ~/mcp-custom/scripts/wrappers/codebase-memory-mcp-docker ~/.local/bin/
ln -sf ~/mcp-custom/scripts/wrappers/mem0-mcp-docker            ~/.local/bin/
ln -sf ~/mcp-custom/scripts/wrappers/headroom-mcp-docker        ~/.local/bin/
```

Cada wrapper carga `.env` (y `.env.mem0` si aplica), asegura los directorios de datos, arranca el contenedor si está parado y ejecuta `docker exec -i` contra el proceso final:

- `codebase-memory-mcp-docker` → `codebase-memory-mcp "$@"`
- `mem0-mcp-docker` → `uvx --quiet --from /opt/mem0-src mem0-mcp-selfhosted "$@"`
- `headroom-mcp-docker` → `headroom mcp serve "$@"`

## Configurar el cliente MCP

Reemplaza `<USUARIO>` por tu usuario del host (el mismo que ejecutó `init-env.sh`).

### Claude Desktop / Claude Code

Ubicación en Linux: `~/.config/claude/mcpServers.json`.

```json
{
  "mcpServers": {
    "codebase_memory_mcp": {
      "type": "stdio",
      "command": "/home/<USUARIO>/.local/bin/codebase-memory-mcp-docker",
      "args": ["--ui=false"],
      "env": {
        "HOME": "/home/<USUARIO>"
      }
    }
  }
}
```

### OpenCode

Ubicación: `~/.config/opencode/opencode.json`.

```json
{
  "mcp": {
    "codebase_memory_mcp": {
      "type": "local",
      "command": ["/home/<USUARIO>/.local/bin/codebase-memory-mcp-docker", "--ui=false"],
      "enabled": true
    }
  }
}
```

### OMP

```json
{
  "$schema": "https://raw.githubusercontent.com/can1357/oh-my-pi/main/packages/coding-agent/src/config/mcp-schema.json",
  "mcpServers": {
    "codebase_memory_mcp": {
      "command": "/home/<USUARIO>/.local/bin/codebase-memory-mcp-docker",
      "args": ["--ui=false"],
      "env": {
        "HOME": "/home/<USUARIO>"
      }
    }
  }
}
```

Los ejemplos solo declaran `codebase_memory_mcp`; para añadir `mem0` y `headroom` copia el mismo bloque cambiando `command` al wrapper correspondiente y el nombre a `mem0` / `headroom`.

## Configuración por servicio

### codebase_memory_mcp

- Sin variables propias en `.env`. Datos persistentes en `~/mcp-custom-data/codebase-memory-mcp/{cache,config}`.
- El contenedor monta `$HOST_HOME:$HOST_HOME` para poder indexar repos del host con rutas absolutas idénticas a las del host.
- `network_mode: none` — el binario funciona offline una vez construida la imagen.

### mem0

Requiere `.env.mem0` (no se autogenera; crear a mano). Variables actuales:

```
MEM0_PROVIDER=ollama
MEM0_LLM_MODEL=qwen2.5:7b
MEM0_EMBED_PROVIDER=ollama
MEM0_EMBED_MODEL=bge-m3
MEM0_OLLAMA_URL=http://127.0.0.1:11434/
MEM0_QDRANT_URL=http://127.0.0.1:6333/
MEM0_COLLECTION=mem0_tutitoos
MEM0_ENABLE_GRAPH=false
MEM0_HISTORY_DB_PATH=/data/history/history.db
```

| Variable | Descripción |
| --- | --- |
| `MEM0_PROVIDER` | Proveedor del LLM que usa mem0 para razonar sobre memorias. |
| `MEM0_LLM_MODEL` | Modelo LLM concreto servido por el proveedor. |
| `MEM0_EMBED_PROVIDER` | Proveedor del modelo de embeddings. |
| `MEM0_EMBED_MODEL` | Modelo de embeddings usado para indexar. |
| `MEM0_OLLAMA_URL` | Endpoint local de Ollama. |
| `MEM0_QDRANT_URL` | Endpoint local de Qdrant. |
| `MEM0_COLLECTION` | Nombre de la colección Qdrant donde se guardan las memorias. |
| `MEM0_ENABLE_GRAPH` | Activa/desactiva el grafo de relaciones. |
| `MEM0_HISTORY_DB_PATH` | Ruta dentro del contenedor de la BD SQLite de historial. |

- `MEM0_USER_ID` vive en `.env` (por defecto `$(whoami)`), no en `.env.mem0`.
- `MEM0_SRC_PATH` (en `.env`) debe apuntar a un clon local de `mem0-mcp-selfhosted`; el contenedor lo monta read-only en `/opt/mem0-src`.
- `network_mode: host` para poder llegar a Ollama/Qdrant en `127.0.0.1`.
- Datos persistentes en `~/mcp-custom-data/mem0/{history,uv-cache,config}`.

### headroom

- Sin variables propias. Datos persistentes en `~/mcp-custom-data/headroom/{cache,config,share}`.
- Se instala vía `pip install headroom-ai[mcp,proxy]`. El wrapper llama `headroom mcp serve`.
- Red por defecto (bridge) — no necesita host network.

## Scripts

| Script | Función |
| --- | --- |
| `scripts/init-env.sh` | Genera `.env` con `HOST_HOME/HOST_UID/HOST_GID/MCP_CUSTOM_ROOT/MCP_CUSTOM_DATA` y las variables de imagen; crea `~/mcp-custom-data/{codebase-memory-mcp,mem0,headroom}/*`. |
| `scripts/build.sh` | Ejecuta `docker compose build` (auto-invoca `init-env.sh` si falta `.env`). |
| `scripts/up.sh` | `docker compose up -d codebase_memory_mcp` únicamente. |
| `scripts/stop.sh` | `docker compose stop codebase_memory_mcp` únicamente. |
| `scripts/wrappers/*-docker` | Entrypoints `docker exec -i` para el cliente MCP. |

## Estructura del repo

```
mcp-custom/
├── compose.yaml
├── .env.example
├── .gitignore
├── AGENTS.md
├── CLAUDE.md
├── mcps/
│   ├── codebase-memory-mcp/Dockerfile
│   ├── mem0/Dockerfile
│   └── headroom/Dockerfile
├── scripts/
│   ├── init-env.sh
│   ├── build.sh
│   ├── up.sh
│   ├── stop.sh
│   └── wrappers/
│       ├── codebase-memory-mcp-docker
│       ├── mem0-mcp-docker
│       └── headroom-mcp-docker
├── configs/
│   └── examples/
│       ├── claude-mcpServers.json.example
│       ├── opencode.json.example
│       └── omp-mcp.json.example
└── skills/
    ├── codebase-memory-mcp/SKILL.md
    └── headroom-mcp/SKILL.md
```

## Datos persistentes

Toda la data vive fuera del repo, en `~/mcp-custom-data` (variable `MCP_CUSTOM_DATA` en `.env`). `.gitignore` excluye `mcp-custom-data/`, `data/`, `.cache/`, `*.db`, `*.sqlite*`, `*.zst`. Subdirectorios por servicio (los mismos que crea `init-env.sh`):

- `codebase-memory-mcp/cache`, `codebase-memory-mcp/config`
- `mem0/history`, `mem0/uv-cache`, `mem0/config`
- `headroom/cache`, `headroom/config`, `headroom/share`

## Skills

`skills/codebase-memory-mcp/SKILL.md` y `skills/headroom-mcp/SKILL.md` son cargados por el agente (Claude Code / OMP) según `AGENTS.md`. Documentan las reglas de uso de cada MCP (siempre usar el wrapper Docker, no llamar al binario del host, no bypasear el MCP con shell). Léelos si vas a añadir un servicio nuevo — sirven de plantilla.

## Añadir un nuevo servicio MCP

1. Crear `mcps/<nombre>/Dockerfile` (patrón: `python:3.12-slim` o `debian:bookworm-slim`, `ARG UID/GID`, `useradd -u $UID`, `HOME=/home/mcp`, `ENTRYPOINT ["sleep"] CMD ["infinity"]`).
2. Añadir servicio a `compose.yaml` copiando el bloque de `headroom` (usa bridge network) o `mem0` (usa host network).
3. Declarar `<NOMBRE>_IMAGE` en `.env.example` y `scripts/init-env.sh`.
4. Añadir `mkdir -p "$DATA_DIR/<nombre>/*"` en `scripts/init-env.sh`.
5. Crear wrapper en `scripts/wrappers/<nombre>-mcp-docker` (copiar `headroom-mcp-docker`, cambiar `SERVICE_NAME`, `CONTAINER_NAME`, y el comando final tras `docker exec -i`).
6. `chmod +x scripts/wrappers/<nombre>-mcp-docker`.
7. Symlink en `~/.local/bin/`.
8. Añadir bloque de config en el JSON del cliente MCP.

## Troubleshooting

- **Wrapper dice `missing .env`**: ejecuta `./scripts/init-env.sh`.
- **`mem0-mcp-docker` dice `MEM0_SRC_PATH does not exist`**: clona `mem0-mcp-selfhosted` en la ruta que apunta `MEM0_SRC_PATH` (por defecto `$HOME/containers/mem0/mem0-src`).
- **`mem0` no conecta con Ollama/Qdrant**: comprueba que Ollama escucha en `127.0.0.1:11434` y Qdrant en `127.0.0.1:6333` desde el host (el contenedor comparte la red del host).
- **Permisos en `~/mcp-custom-data`**: `init-env.sh` fija `HOST_UID`/`HOST_GID` para que el usuario `mcp` del contenedor coincida con el del host; si copias datos desde otro user, ajusta con `chown`.
- **Rebuild tras cambiar Dockerfile**: `./scripts/build.sh` (no necesita parar los contenedores; el próximo `docker exec` usará la imagen nueva solo tras recrear el contenedor, hazlo con `docker compose up -d --force-recreate <servicio>`).
