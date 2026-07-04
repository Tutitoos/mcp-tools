# mcp-tools

Colección de servidores MCP (Model Context Protocol) empaquetados en Docker, orquestados con `docker compose`. Cada servidor corre en un contenedor persistente que hace `sleep infinity`; los clientes MCP se conectan vía wrappers `docker exec -i` instalados en `~/.local/bin`. Diseñado para uso local con Claude Code / Claude Desktop, OpenCode y OMP.

## Servicios incluidos

| Servicio (compose / MCP key) | Imagen | Contenedor | Propósito |
| --- | --- | --- | --- |
| `mcp_tools_codebase_memory` | `mcp-tools/codebase-memory:latest` | `mcp-tools-codebase-memory` | Grafo de conocimiento y búsqueda de código sobre repos locales |
| `mcp_tools_mem0` | `mcp-tools/mem0:latest` | `mcp-tools-mem0` | Memoria persistente self-hosted (mem0) con Ollama + Qdrant |
| `mcp_tools_headroom` | `mcp-tools/headroom:latest` | `mcp-tools-headroom` | Compresión de texto/logs para reducir tokens |

## Requisitos

- Docker + `docker compose` v2.
- Linux (probado en kernel 7.0.12-1-pve, Debian/Ubuntu compatible).
- `~/.local/bin` en `$PATH` para los wrappers.
- Para `mcp_tools_mem0`: instancia local de **Ollama** en `http://127.0.0.1:11434` y **Qdrant** en `http://127.0.0.1:6333` (por eso `network_mode: host`). Además el código fuente de mem0 clonado en `MEM0_SRC_PATH` (por defecto `$MCP_TOOLS_DATA/mem0/src`, i.e. `~/mcp-tools-data/mem0/src`) porque el wrapper hace `uvx --from /opt/mem0-src mem0-mcp-selfhosted`. Repo upstream: `https://github.com/elvismdev/mem0-mcp-selfhosted`.
- Para `mcp_tools_codebase_memory`: nada extra (el binario se instala dentro de la imagen desde `raw.githubusercontent.com/DeusData/codebase-memory-mcp`).
- Para `mcp_tools_headroom`: nada extra (`pip install headroom-ai[mcp,proxy]` dentro de la imagen).

## Arranque rápido

```bash
git clone <repo-url> ~/mcp-tools
cd ~/mcp-tools

# 1. Genera .env desde el host (UID/GID/HOME) y crea ~/mcp-tools-data
./scripts/init-env.sh

# 2. (solo mem0) crea .env.mem0 con el bloque de la sección "mcp_tools_mem0" más abajo
#    (no existe plantilla; .env.example no incluye las variables de mem0)

# 3. Construye las imágenes
./scripts/build.sh

# 4. Arranca los contenedores persistentes
./scripts/up.sh                                            # levanta los 3 MCPs
```

`up.sh` levanta los 3 MCPs; los wrappers también arrancan on-demand el contenedor que necesiten si está parado.

## Instalar los wrappers

Los wrappers viven en `scripts/wrappers/` y deben quedar accesibles como `~/.local/bin/<nombre>`:

```bash
mkdir -p ~/.local/bin
ln -sf ~/mcp-tools/scripts/wrappers/mcp-tools-codebase-memory-docker ~/.local/bin/
ln -sf ~/mcp-tools/scripts/wrappers/mcp-tools-mem0-docker            ~/.local/bin/
ln -sf ~/mcp-tools/scripts/wrappers/mcp-tools-headroom-docker        ~/.local/bin/
```

Cada wrapper carga `.env` (y `.env.mem0` si aplica), asegura los directorios de datos, arranca el contenedor si está parado y ejecuta `docker exec -i` contra el proceso final:

- `mcp-tools-codebase-memory-docker` → `codebase-memory-mcp "$@"` (binario upstream dentro del contenedor)
- `mcp-tools-mem0-docker` → `uvx --quiet --from /opt/mem0-src mem0-mcp-selfhosted "$@"`
- `mcp-tools-headroom-docker` → `headroom mcp serve "$@"`

## Configurar el cliente MCP

Reemplaza `<USUARIO>` por tu usuario del host (el mismo que ejecutó `init-env.sh`).

### Claude Desktop / Claude Code

Ubicación en Linux: `~/.config/claude/mcpServers.json`.

```json
{
  "mcpServers": {
    "mcp_tools_codebase_memory": {
      "type": "stdio",
      "command": "/home/<USUARIO>/.local/bin/mcp-tools-codebase-memory-docker",
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
    "mcp_tools_codebase_memory": {
      "type": "local",
      "command": ["/home/<USUARIO>/.local/bin/mcp-tools-codebase-memory-docker", "--ui=false"],
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
    "mcp_tools_codebase_memory": {
      "command": "/home/<USUARIO>/.local/bin/mcp-tools-codebase-memory-docker",
      "args": ["--ui=false"],
      "env": {
        "HOME": "/home/<USUARIO>"
      }
    }
  }
}
```

Los ejemplos solo declaran `mcp_tools_codebase_memory`; para añadir `mcp_tools_mem0` y `mcp_tools_headroom` copia el mismo bloque cambiando `command` al wrapper correspondiente y el nombre a la clave respectiva.

## Configuración por servicio

### mcp_tools_codebase_memory

- Sin variables propias en `.env`. Datos persistentes en `~/mcp-tools-data/codebase-memory/{cache,config}`.
- El contenedor monta `$HOST_HOME:$HOST_HOME` para poder indexar repos del host con rutas absolutas idénticas a las del host.
- `network_mode: none` — el binario funciona offline una vez construida la imagen.

### mcp_tools_mem0

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
- Datos persistentes en `~/mcp-tools-data/mem0/{history,uv-cache,config}`.

### mcp_tools_headroom

- Sin variables propias. Datos persistentes en `~/mcp-tools-data/headroom/{cache,config,share}`.
- Se instala vía `pip install headroom-ai[mcp,proxy]`. El wrapper llama `headroom mcp serve`.
- Red por defecto (bridge) — no necesita host network.

## Scripts

| Script | Función |
| --- | --- |
| `scripts/init-env.sh` | Genera `.env` con `HOST_HOME/HOST_UID/HOST_GID/MCP_TOOLS_ROOT/MCP_TOOLS_DATA` y las variables de imagen; crea `~/mcp-tools-data/{codebase-memory,mem0,headroom}/*`. |
| `scripts/build.sh` | Ejecuta `docker compose build` (auto-invoca `init-env.sh` si falta `.env`). |
| `scripts/up.sh` | `docker compose up -d` — arranca todos los servicios de `compose.yaml`. |
| `scripts/stop.sh` | `docker compose stop` — para todos los servicios de `compose.yaml`. |
| `scripts/wrappers/*-docker` | Entrypoints `docker exec -i` para el cliente MCP. |

## Estructura del repo

```
mcp-tools/
├── compose.yaml
├── .env.example
├── .gitignore
├── AGENTS.md
├── CLAUDE.md
├── mcps/
│   ├── codebase-memory/Dockerfile
│   ├── mem0/Dockerfile
│   └── headroom/Dockerfile
├── scripts/
│   ├── init-env.sh
│   ├── build.sh
│   ├── up.sh
│   ├── stop.sh
│   └── wrappers/
│       ├── mcp-tools-codebase-memory-docker
│       ├── mcp-tools-mem0-docker
│       └── mcp-tools-headroom-docker
├── configs/
│   └── examples/
│       ├── claude-mcpServers.json.example
│       ├── opencode.json.example
│       └── omp-mcp.json.example
└── skills/
    ├── codebase-memory/SKILL.md
    └── headroom/SKILL.md
```

## Datos persistentes

Toda la data vive fuera del repo, en `~/mcp-tools-data` (variable `MCP_TOOLS_DATA` en `.env`). `.gitignore` excluye `mcp-tools-data/`, `data/`, `.cache/`, `*.db`, `*.sqlite*`, `*.zst`. Subdirectorios por servicio (los mismos que crea `init-env.sh`):

- `codebase-memory/cache`, `codebase-memory/config`
- `mem0/history`, `mem0/uv-cache`, `mem0/config`
- `headroom/cache`, `headroom/config`, `headroom/share`

## Skills

`skills/codebase-memory/SKILL.md` y `skills/headroom/SKILL.md` son cargados por el agente (Claude Code / OMP) según `AGENTS.md`. Documentan las reglas de uso de cada MCP (siempre usar el wrapper Docker, no llamar al binario del host, no bypasear el MCP con shell). Léelos si vas a añadir un servicio nuevo — sirven de plantilla.

## Añadir un nuevo servicio MCP

1. Crear `mcps/<nombre>/Dockerfile` (patrón: `python:3.12-slim` o `debian:bookworm-slim`, `ARG UID/GID`, `useradd -u $UID`, `HOME=/home/mcp`, `ENTRYPOINT ["sleep"] CMD ["infinity"]`).
2. Añadir servicio a `compose.yaml` como `mcp_tools_<nombre>` copiando el bloque de `mcp_tools_headroom` (bridge) o `mcp_tools_mem0` (host network). `container_name: mcp-tools-<nombre>`.
3. Declarar `MCP_TOOLS_<NOMBRE>_IMAGE` en `.env.example` y `scripts/init-env.sh`.
4. Añadir `mkdir -p "$DATA_DIR/<nombre>/*"` en `scripts/init-env.sh`.
5. Crear wrapper en `scripts/wrappers/mcp-tools-<nombre>-docker` (copiar `mcp-tools-headroom-docker`, cambiar `SERVICE_NAME`, `CONTAINER_NAME`, y el comando final tras `docker exec -i`).
6. `chmod +x scripts/wrappers/mcp-tools-<nombre>-docker`.
7. Symlink en `~/.local/bin/`.
8. Añadir bloque de config en el JSON del cliente MCP con clave `mcp_tools_<nombre>`.

## Migración desde `mcp-custom`

Si vienes de la versión previa con prefijo `mcp-custom`, tras hacer merge de este branch **no basta con git pull**: hay estado fuera del repo que debes mover a mano. Todos los pasos son one-shot en el host.

1. **Parar contenedores viejos y borrarlos** (los nombres cambiaron, no se pueden reusar):
   ```bash
   docker compose -p mcp-custom down
   docker rm -f mcp-custom-codebase-memory-mcp mcp-custom-mem0-mcp mcp-custom-headroom-mcp 2>/dev/null || true
   ```
2. **Mover el directorio de datos**:
   ```bash
   mv ~/mcp-custom-data ~/mcp-tools-data
   mv ~/mcp-tools-data/codebase-memory-mcp ~/mcp-tools-data/codebase-memory
   ```
3. **(Opcional) renombrar el directorio del repo** — si lo mueves, ajusta cualquier symlink que apunte dentro:
   ```bash
   mv ~/mcp-custom ~/mcp-tools
   ```
4. **Regenerar `.env`** (usa los nuevos nombres de variables e imágenes):
   ```bash
   cd ~/mcp-tools   # o ~/mcp-custom si no renombraste
   ./scripts/init-env.sh
   ```
5. **Reconstruir imágenes** con los tags nuevos:
   ```bash
   ./scripts/build.sh
   # (opcional) borra las imágenes viejas
   docker image rm mcp-custom/codebase-memory-mcp:latest mcp-custom/mem0-mcp:latest mcp-custom/headroom-mcp:latest 2>/dev/null || true
   ```
6. **Rehacer los symlinks en `~/.local/bin/`**:
   ```bash
   rm -f ~/.local/bin/codebase-memory-mcp-docker ~/.local/bin/mem0-mcp-docker ~/.local/bin/headroom-mcp-docker
   ln -sf ~/mcp-tools/scripts/wrappers/mcp-tools-codebase-memory-docker ~/.local/bin/
   ln -sf ~/mcp-tools/scripts/wrappers/mcp-tools-mem0-docker            ~/.local/bin/
   ln -sf ~/mcp-tools/scripts/wrappers/mcp-tools-headroom-docker        ~/.local/bin/
   ```
7. **Actualizar el JSON del cliente MCP** (fuera del repo, no lo toca este branch):
   - Cambiar la clave del servidor (p. ej. `codebase_memory_mcp` → `mcp_tools_codebase_memory`).
   - Cambiar `command` al nuevo wrapper (`mcp-tools-codebase-memory-docker`).
   - Repetir para `mem0` y `headroom` si los tenías declarados.
   - Reiniciar el cliente MCP para que recargue la config.
8. **Verificar**: `docker compose config >/dev/null` (sin errores), `mcp-tools-codebase-memory-docker --help`, y en el cliente MCP comprueba que las herramientas ahora aparecen bajo el prefijo `mcp__mcp_tools_<servicio>_*`.

Las tools del cliente cambian de namespace (`mcp__headroom_compress` → `mcp__mcp_tools_headroom_compress`, etc.). Si tienes cualquier documento o skill externa que las referencie por nombre, actualízalo.

## Troubleshooting

- **Wrapper dice `missing .env`**: ejecuta `./scripts/init-env.sh`.
- **`mcp-tools-mem0-docker` dice `MEM0_SRC_PATH does not exist`**: clona `elvismdev/mem0-mcp-selfhosted` en `~/mcp-tools-data/mem0/src` (o donde apunte `MEM0_SRC_PATH`).
- **`mcp_tools_mem0` no conecta con Ollama/Qdrant**: comprueba que Ollama escucha en `127.0.0.1:11434` y Qdrant en `127.0.0.1:6333` desde el host (el contenedor comparte la red del host).
- **Permisos en `~/mcp-tools-data`**: `init-env.sh` fija `HOST_UID`/`HOST_GID` para que el usuario `mcp` del contenedor coincida con el del host; si copias datos desde otro user, ajusta con `chown`.
- **Rebuild tras cambiar Dockerfile**: `./scripts/build.sh` (no necesita parar los contenedores; el próximo `docker exec` usará la imagen nueva solo tras recrear el contenedor, hazlo con `docker compose up -d --force-recreate <servicio>`).
