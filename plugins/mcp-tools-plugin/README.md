# mcp-tools-plugin

Extensiones OMP para el ecosistema `mcp-tools`: fuerza el enrutamiento de trabajo de inteligencia de código hacia los servidores MCP conectados (serena/tokensave/codebase-memory/mem0) en vez de herramientas nativas, y ofrece un recordatorio de mantenimiento tras turnos que modifican código.

## Qué hace cada guard/extensión

- **`serena-symbol`** — bloquea `grep`/`ast_grep`/`glob`/`bash(grep|rg|ag|find …)` cuando el patrón parece un nombre de símbolo (sin espacios, sin comodines), y redirige a `find_symbol`/`find_referencing_symbols` de `mcp_tools_serena`. Ejemplo: `grep(pattern: "someIdentifier")` con serena conectado se bloquea; `grep(pattern: "hello world")` pasa porque tiene un espacio.
- **`codebase-memory-cross-repo`** — bloquea `glob`/`bash(grep|rg|ag|find …)` cuando el path apunta a un repositorio git *distinto* del actual, y redirige a `search_code`/`search_graph`/`trace_path` de `mcp_tools_codebase_memory`. Ejemplo: desde `~/mcp-tools`, `glob(path: "~/other-project/**")` se bloquea si `~/other-project` es otro repo git.
- **`tokensave-explore`** — bloquea `task(agent: "explore")` cuando tokensave está conectado, forzando el uso directo de `tokensave_context`. Ejemplo: pedir "explora el código de auth" dispara un subagente explore, que queda bloqueado con instrucciones de usar `tokensave_context` en su lugar.
- **`mem0-search-first`** — bloquea `add_memory` hasta que se haya llamado `search_memories`/`get_memories` al menos una vez en la sesión actual, para no duplicar memorias ya guardadas. El estado es por sesión y se limpia en `session_shutdown`/`session_switch`/`session_branch`.
- **`post-task-maintenance-offer`** — tras un turno que editó/escribió archivos (o hizo una mutación serena), inyecta un mensaje oculto que obliga al modelo a preguntar si se desea correr `tokensave sync`, actualizar el índice de `codebase-memory`, o guardar una memoria durable en `mem0`. Solo pregunta una vez por mutación (debounce).

## Instalación

Desarrollo local (symlink):

```bash
omp plugin link ~/mcp-tools/plugins/mcp-tools-plugin
```

Remoto (una vez publicado en GitHub):

```bash
omp plugin install github:Luqueee/mcp-tools-plugin
```

## Configuración

Cada guard y el nudge de mantenimiento se pueden desactivar sin tocar código, vía JSON de dos capas (`user <- project`, el proyecto gana). Ejemplo completo con todos los valores por defecto — `~/.omp/agent/mcp-tools-plugin.config.json`:

```json
{
  "guards": {
    "serena-symbol": { "enabled": true },
    "codebase-memory-cross-repo": { "enabled": true },
    "tokensave-explore": { "enabled": true },
    "mem0-search-first": { "enabled": true }
  },
  "postTaskMaintenance": { "enabled": true }
}
```

Un `<cwd>/.omp/mcp-tools-plugin.config.json` con la misma forma sobreescribe, por clave, la capa de usuario para ese proyecto solamente. Claves desconocidas se ignoran; un `enabled` que no sea booleano cae al valor por defecto de esa clave. No hay recarga en caliente — los ajustes se leen una vez al cargar la extensión.

## Desinstalación

```bash
omp plugin uninstall mcp-tools-plugin
```

Esto solo quita el symlink/entrada de lockfile bajo `~/.omp/plugins/`; el código fuente en `~/mcp-tools/plugins/mcp-tools-plugin/` queda intacto.

## Desarrollo

```bash
bun install
bun test
bunx tsc --noEmit
```
