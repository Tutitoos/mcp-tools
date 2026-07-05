# Advanced

Documentación densa para configurar más allá del `mcp-tools install` por defecto. El README raíz cubre el flujo estándar; este fichero es la referencia para: gestionar cada componente por separado, migrar desde el pipeline viejo, y añadir componentes nuevos al registry.

## Tool registry

`internal/tools/registry.go` es el punto único de verdad. Cada `Tool` declara:

- `Key` (kebab-case), `Label`, `Summary`.
- `Deploy` (`Host`, `Docker`, `Sudo`).
- `Deps []string` (keys que deben instalarse antes).
- `DefaultOn` (marcado por default en el TUI).
- `SelfRegisters` (skippea `mcp-config` — el tool se auto-registra en sus IDEs).
- Closures `Install`, `Upgrade`, `Uninstall`, `Status`.

Para añadir un componente nuevo:

1. Crear `internal/tools/<nombre>.go` con `func <nombre>Tool() Tool { … }`.
2. Añadir la llamada en `Registry()` (`internal/tools/registry.go`).
3. Si expone un MCP stdio, añadirle un `case` en `internal/mcp/servers.go` `Servers(state.State)`.
4. Si expone verbos propios (`mcp-tools <nombre> install/upgrade/…`), crear `internal/cli/<nombre>.go` copiando `internal/cli/rtk.go` (5 líneas).

Correr `mcp-tools install --reconfigure` para verlo aparecer en el TUI.

## Configuración por componente

### codebase-memory (host)

- Binario en `~/.local/share/codebase-memory-mcp/codebase-memory-mcp`, symlink en `~/.local/bin/codebase-memory-mcp`.
- Instalación vía script upstream de DeusData (`curl … | bash -s -- --standard --skip-config --dir …`).
- Registrado en clientes como MCP `mcp_tools_codebase_memory`, `command=codebase-memory-mcp`, `args=["--ui=false"]`.
- Al pasar a host desaparece la sandbox de red (Docker `network_mode: none`). Upstream promete "100% local, no telemetry". Si no confías, desmárcalo en el TUI.

### mem0 (host)

- Binario en `~/.local/bin/mem0-mcp-selfhosted` (uv tool install). Wrapper en `~/.local/bin/mem0-launcher` (script bash que sourcea `.env.mem0` y execs).
- Requiere `qdrant` y `ollama` seleccionados (declarado en `Deps`; el TUI auto-marca ambos si marcas mem0).
- Registrado en clientes como MCP `mcp_tools_mem0`, `command=/home/<USUARIO>/.local/bin/mem0-launcher`.
- Variables en `.env.mem0`:

| Variable | Descripción |
| --- | --- |
| `MEM0_PROVIDER` | Proveedor del LLM que usa mem0 para razonar sobre memorias. |
| `MEM0_LLM_MODEL` | Modelo LLM concreto servido por el proveedor. |
| `MEM0_EMBED_PROVIDER` | Proveedor del modelo de embeddings. |
| `MEM0_EMBED_MODEL` | Modelo de embeddings usado para indexar. |
| `MEM0_OLLAMA_URL` | Endpoint local de Ollama (default `http://127.0.0.1:11434/`). |
| `MEM0_QDRANT_URL` | Endpoint local de Qdrant (default `http://127.0.0.1:6333/`). |
| `MEM0_COLLECTION` | Nombre de la colección Qdrant donde se guardan las memorias. |
| `MEM0_ENABLE_GRAPH` | Activa/desactiva el grafo de relaciones. |
| `MEM0_HISTORY_DB_PATH` | Ruta dentro del wrapper de la BD SQLite de historial. |

- `MEM0_USER_ID` vive en `.env` (por defecto `$(whoami)`), no en `.env.mem0`.
- Datos persistentes en `~/mcp-tools-data/mem0/{history,uv-cache,config}`.

### headroom (host)

Headroom expone un MCP stdio (`headroom mcp serve`) y un proxy HTTP opcional en `127.0.0.1:8787`.

- **Instalación**: `mcp-tools headroom install`. Corre `uv tool install "headroom-ai[mcp,proxy]"` con fallback automático a `headroom-ai[mcp]` si la build de `mitmproxy` falla.
- **Upgrade**: `mcp-tools headroom upgrade` (`uv tool upgrade headroom-ai`).
- **Status**: `mcp-tools headroom status` — JSON con versión, path y clientes MCP.
- **Activar el proxy de compresión** (opt-in): `headroom proxy` sirve en `127.0.0.1:8787`; export `ANTHROPIC_BASE_URL=http://127.0.0.1:8787`.
- **Durable Claude hooks** (opt-in): `headroom init claude` — persiste la compresión sin exportar `ANTHROPIC_BASE_URL`.
- **Uninstall**: `mcp-tools uninstall headroom` (usa `uv tool uninstall`).

### rtk (host)

RTK es un hook shell (no MCP): rewrite de comandos con 60–90 % de ahorro de tokens en dev ops.

- **Instalación**: `mcp-tools rtk install`. Corre rustup unattended (`--no-modify-path`) si `cargo` no está en PATH, luego `cargo install --git https://github.com/makoMakoGo/rtk.git --branch feat/omp-extension-rewrite --locked rtk` y `rtk init --agent omp --auto-patch` (+ `--agent claude` si el CLI `claude` está en PATH). Escribe `~/.omp/extensions/rtk.ts` y patchea `~/.claude/settings.json`.
- **Upgrade**: `mcp-tools rtk upgrade` (`cargo install --force` + re-init).
- **Uninstall**: `mcp-tools uninstall rtk` — hace `cargo uninstall rtk` y borra `~/.omp/extensions/rtk.ts`. La entrada `rtk hook claude` en `~/.claude/settings.json` hay que quitarla a mano (el CLI `claude` no expone `hooks remove`).

### claude-mem (host)

Plugin de Claude Code. Opt-in en el TUI. Requiere Node ≥ 20 (verifica con `node --version`).

- **Instalación**: `mcp-tools claude-mem install` corre `npx --yes claude-mem@latest install`. El plugin se auto-registra en Claude Code.
- **Uninstall**: `mcp-tools claude-mem uninstall` corre `npx --yes claude-mem@latest uninstall`.
- Con `nvm`: el CLI aterriza en `~/.nvm/versions/node/vXX.Y.Z/bin`. Si cambias de versión de Node el status puede aparecer como no-instalado; re-instala en la nueva versión.

### codegraph (host)

MCP con auto-registro en 8 IDEs (Claude Code, Cursor, Windsurf, etc.). Opt-in en el TUI.

- **Instalación**: `mcp-tools codegraph install` corre el bundle self-contained (`curl -fsSL … | sh`) y luego `codegraph install --yes` para auto-registrarse.
- **Uninstall**: `mcp-tools uninstall codegraph` corre `codegraph uninstall --yes`.

### ollama (Docker + GPU opcional)

- Imagen `ollama/ollama:latest`. Puerto expuesto en `${MCP_TOOLS_BIND}:11434`.
- Modelos en `${MCP_TOOLS_DATA}/ollama` (bind mount → `/root/.ollama`).
- Post-install: el tool descarga `MEM0_LLM_MODEL` y `MEM0_EMBED_MODEL` declarados en `.env.mem0` (idempotente).
- GPU passthrough: NO viene por default. Se activa cuando (a) `nvidia-smi -L` pasa **y** (b) `nvidia-toolkit` está en `state.Selected`. En ese caso `mcp-tools up` incluye `dockers/ollama-gpu-overlay.yml` que añade `deploy.resources.reservations.devices` con `driver: nvidia`. La lógica vive en `internal/tools/compose.go OllamaComposeFiles(state.State)` — sync point compartido por `Ollama.Install` y `internal/cli/up.go`.
- Si tu compose es < 1.28 y no soporta el bloque `deploy`, cambia el overlay al equivalente `gpus: all` (compatible con versiones más antiguas).

### qdrant (Docker)

- Imagen `qdrant/qdrant:v1.12.0` (pin explícito). Puerto expuesto en `${MCP_TOOLS_BIND}:6333`.
- Datos en el volumen docker externo `mcp-qdrant-storage` (declarado `external: true` en `dockers/qdrant-compose.yml`). Se conserva al hacer `docker compose down` sin `-v`.
- Healthcheck sobre TCP 6333 cada 10 s; usado por `depends_on` de otros servicios.

### nvidia-container-toolkit (Sudo)

Requerido para GPU passthrough a ollama. Solo aparece en el TUI si `nvidia-smi -L` detecta una GPU.

- **Instalación**: `mcp-tools nvidia-toolkit install`. Corre con stdio heredado (el sudo prompt es interactivo). Vía apt: importa la clave upstream de libnvidia-container, añade el repo, `apt-get install nvidia-container-toolkit`, `nvidia-ctk runtime configure --runtime=docker`, `systemctl restart docker`.
- **Distros soportadas**: Debian, Ubuntu, RHEL, Fedora, CentOS, Rocky, AlmaLinux. Si `/etc/os-release` `ID=` no está en esa lista, el install falla explícitamente.
- **Upgrade**: no expuesto (se hace vía `apt-get upgrade nvidia-container-toolkit` a mano).
- **Uninstall**: `mcp-tools uninstall nvidia-toolkit` corre `nvidia-ctk runtime configure --runtime=docker --unset` + `apt-get purge -y nvidia-container-toolkit` + `systemctl restart docker`.
- **CI / no-TTY**: el prompt sudo requiere shell interactivo; en CI el step falla con "corre en shell interactiva". El resto del install sigue por diseño (cada tool independiente).

## Estado persistente

`~/mcp-tools-data/state.json` (schema v1). Se escribe atómicamente (tempfile + rename).

```json
{
  "version": 1,
  "selected": ["qdrant", "ollama", "codebase-memory", "mem0", "headroom", "rtk"],
  "versions": {
    "codebase-memory": "codebase-memory-mcp 0.5.0",
    "mem0": "mem0-mcp-selfhosted 0.2.1",
    "headroom": "headroom, version 0.28.0",
    "rtk": "rtk 0.40.0"
  },
  "updated_at": "2026-07-05T20:00:00Z"
}
```

- `selected` está topo-ordenado (deps primero) — `TopoSort` en `internal/tools/registry.go`.
- `versions` se actualiza tras cada `install`/`update`; se lee de `Tool.Status().Version`.
- Si el fichero está corrupto (JSON malformado) `mcp-tools` no puede correr `mcp-config`; borra el fichero y re-instala.

`--force` en `mcp-tools uninstall <tool>` bypassa el reverse-dep check pero **no** persiste el estado "broken": solo imprime un WARN. Si desinstalas ollama con `--force` mientras mem0 está seleccionado, mem0 seguirá en `state.selected` pero se romperá al arrancar. Vuelve a instalar ollama o quita mem0 del state.

## Migración desde el pipeline viejo

Si vienes de la revisión con contenedores para codebase-memory + mem0:

```bash
docker rm -f mcp-tools-codebase-memory mcp-tools-mem0 2>/dev/null || true
docker image rm mcp-tools/codebase-memory:latest mcp-tools/mem0:latest 2>/dev/null || true
rm -f ~/.local/bin/mcp-tools-codebase-memory-docker ~/.local/bin/mcp-tools-mem0-docker
mcp-tools env --force        # regenera .env sin las variables viejas
mcp-tools install             # TUI aparece; elige componentes; state.json se crea
```

El volumen `mcp-qdrant-storage` se preserva (declarado `external: true`).

Si vienes de la versión previa con prefijo `mcp-custom` (nombres de contenedor y directorio de datos distintos), el flujo es:

1. Parar contenedores viejos: `docker compose -p mcp-custom down`.
2. Mover datos: `mv ~/mcp-custom-data ~/mcp-tools-data` (respeta las convenciones actuales).
3. `mcp-tools env` → `mcp-tools install`.

## Migración desde `mcp-infra` (qdrant + ollama)

Si tenías qdrant/ollama corriendo bajo el proyecto compose `mcp-infra`, ahora son propiedad de este repo:

1. Parar el stack viejo sin borrar volúmenes: `docker compose -p mcp-infra -f /path/al/docker-compose.yml down`.
2. Verificar que el volumen `mcp-qdrant-storage` sigue existiendo (`docker volume ls | grep mcp-qdrant-storage`).
3. Copiar los modelos de Ollama al path convencional: `docker run --rm -v /path/a/ollama-data:/src:ro -v ~/mcp-tools-data/ollama:/dst alpine sh -c 'cp -a /src/. /dst/'`.
4. `mcp-tools up` — adopta el volumen y la carpeta de modelos sin re-descargar nada.
5. Los nombres de contenedor cambian: cualquier script externo que use `docker exec mem0-qdrant …` o `docker exec mcp-ollama …` hay que actualizarlo a `mcp-tools-mem0-qdrant` / `mcp-tools-ollama`.

## Seguridad

Cinco instaladores upstream usan `curl … | sh`: rustup (RTK), uv (headroom + mem0), install.sh de codebase-memory, install.sh de codegraph, y el apt repo de nvidia-container-toolkit. Audita los scripts si no confías en la fuente antes de correr `mcp-tools install`. Los tools instalados aquí son opt-in per-registry: puedes marcar sólo lo que quieras en el TUI multi-select.

Con `MCP_TOOLS_BIND=0.0.0.0` (default), qdrant y ollama son alcanzables desde toda la LAN. Ninguno tiene auth por default. Setea `MCP_TOOLS_BIND=127.0.0.1` en `.env` antes de `mcp-tools install` para bindear sólo a loopback.
