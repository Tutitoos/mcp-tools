# Advanced

Documentación densa para configurar más allá del `./install.sh` por defecto. El README raíz cubre el flujo estándar; este fichero es la referencia para: personalizar servicios, migrar desde el layout previo y añadir un servicio nuevo.

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
- qdrant se declara en `dockers/mem0-compose.yml` (top-level `include:` desde `dockers/compose.yaml`); ollama es el servicio compartido `mcp_tools_ollama` en `dockers/compose.yaml`. `depends_on` bloquea el arranque de mem0 hasta que qdrant esté `service_healthy`.
- Datos persistentes en `~/mcp-tools-data/mem0/{history,uv-cache,config}`.

### mcp_tools_mem0_qdrant

- Imagen `qdrant/qdrant:v1.12.0` (pin explícito). Puerto expuesto en `127.0.0.1:6333`.
- Datos en el volumen docker externo `mcp-qdrant-storage` (declarado `external: true` en `dockers/mem0-compose.yml`). Adoptado del stack previo `mcp-infra`; se conserva al hacer `docker compose down` sin `-v`.
- Healthcheck sobre TCP 6333 cada 10 s; usado por `depends_on` de `mcp_tools_mem0`.

### mcp_tools_ollama

- Imagen `ollama/ollama:latest`. Puerto expuesto en `127.0.0.1:11434`.
- Modelos en `${MCP_TOOLS_DATA}/ollama` (bind mount → `/root/.ollama`). Path fijo por convención — mcp-tools siempre usa `~/mcp-tools-data/` para toda la data persistente.
- Sin GPU passthrough — la imagen corre en CPU. Habilitar GPU requiere instalar `nvidia-container-toolkit` en el host y añadir `deploy.resources.reservations.devices` al servicio; fuera de alcance.
- Servicio compartido: cualquier MCP que necesite un LLM local puede consumirlo apuntando a `http://127.0.0.1:11434/` (o `http://mcp-tools-ollama:11434/` si se une a una red bridge del compose).

### mcp_tools_headroom

- Sin variables propias. Datos persistentes en `~/mcp-tools-data/headroom/{cache,config,share}`.
- Se instala vía `pip install headroom-ai[mcp,proxy]`. El wrapper llama `headroom mcp serve`.
- Red por defecto (bridge) — no necesita host network.

## Añadir un nuevo servicio MCP

1. Crear `mcps/<nombre>/Dockerfile` (patrón: `python:3.12-slim` o `debian:bookworm-slim`, `ARG UID/GID`, `useradd -u $UID`, `HOME=/home/mcp`, `ENTRYPOINT ["sleep"] CMD ["infinity"]`).
2. Añadir servicio a `dockers/compose.yaml` como `mcp_tools_<nombre>` copiando el bloque de `mcp_tools_headroom` (bridge) o `mcp_tools_mem0` (host network). `container_name: mcp-tools-<nombre>`.
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
8. **Verificar**: `docker compose -f dockers/compose.yaml --env-file .env config >/dev/null` (sin errores), `mcp-tools-codebase-memory-docker --help`, y en el cliente MCP comprueba que las herramientas ahora aparecen bajo el prefijo `mcp__mcp_tools_<servicio>_*`.

Las tools del cliente cambian de namespace (`mcp__headroom_compress` → `mcp__mcp_tools_headroom_compress`, etc.). Si tienes cualquier documento o skill externa que las referencie por nombre, actualízalo.

## Migración desde `mcp-infra` (qdrant + ollama)

Si tenías qdrant/ollama corriendo bajo el proyecto compose `mcp-infra` (`/home/tutitoos/containers/mcp-infra/docker-compose.yml`), ahora son propiedad de este repo (qdrant en `dockers/mem0-compose.yml`, ollama como servicio compartido en `dockers/compose.yaml`):

1. Parar el stack viejo sin borrar volúmenes:
   ```bash
   docker compose -p mcp-infra -f /home/tutitoos/containers/mcp-infra/docker-compose.yml down
   ```
2. Verificar que el volumen `mcp-qdrant-storage` sigue existiendo (`docker volume ls | grep mcp-qdrant-storage`). Copiar los modelos de Ollama al path convencional: `docker run --rm -v /home/tutitoos/containers/ollama-data:/src:ro -v ~/mcp-tools-data/ollama:/dst alpine sh -c 'cp -a /src/. /dst/'` (usa `alpine` como helper root porque el bind viejo suele ser root-owned).
3. `./scripts/up.sh` — el nuevo stack adopta el volumen (declarado `external: true`) y la carpeta de modelos sin re-descargar nada.
4. Los nombres de contenedor cambian: `mem0-qdrant` → `mcp-tools-mem0-qdrant`, `mcp-ollama` → `mcp-tools-ollama`. Cualquier script externo que use `docker exec mem0-qdrant …` o `docker exec mcp-ollama …` hay que actualizarlo.
