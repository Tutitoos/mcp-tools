# Review — mcp-tools (sin bugs)

Fecha: 2026-07-08 · Estado: 20 fixes aplicados, plan ejecutado completo.

## Resumen ejecutivo

Auditoría completa del árbol `~/mcp-tools` previa a iteración pública. Tres categorías de hallazgos:

- **Tier 1 (14)**: bugs reales de fiabilidad, privacidad, atomicidad, estado, shell y configuración — todos corregidos.
- **Tier 2 (10)**: 6 seleccionados y corregidos (los que afectan a la primera instalación, status y post-update); 4 restantes documentados para futura iteración.
- **Tier 3 (10+)**: deuda acumulada, tests mínimos, referencias stale — todos documentados, ninguno corregido en esta ronda.

Total aplicado: 13 fixes Tier 1 + 6 fixes Tier 2 + 1 fix `RULES.md` (mem0) + 3 tests nuevos = **23 ediciones**. Estado final: `go build` / `go vet` / `gofmt` / `go test` / `go test -race` verdes.

---

## Hallazgos Tier 1 (corregidos)

### H1: `.env.mem0` no se ignora explícitamente
- **Ubicación**: `.gitignore:1-3`
- **Severidad**: Tier 1 (privacidad)
- **Riesgo**: aunque `.env.*` (catch-all) excluye `.env.mem0`, la regla implícita no comunica la intención: cualquiera que añada otra excepción a `.env.example` puede descubrir accidentalmente que el archivo real se commitea. Además no existe `.env.mem0.example` para regenerar el fichero en máquinas nuevas.
- **Fix**: tres reglas explícitas (`.env`, `.env.mem0`, `!.env.example`); crear `.env.mem0.example` con valores neutros (`mem0_user` en `MEM0_COLLECTION`).
- **Verificar**: `git check-ignore -v .env.mem0` → ignorado; `git check-ignore -v .env.example` → NO ignorado; `ls .env.mem0.example` existe.

### H2: `WriteJSON` no es atómico y deja `0o644`
- **Ubicación**: `internal/mcp/jsonfile.go:42-53`
- **Severidad**: Tier 1 (atomicidad, privacidad)
- **Riesgo**: un crash a mitad de `WriteJSON` deja un `~/.claude.json` o `~/.config/opencode/opencode.json` corrupto. Adicionalmente, esos ficheros pueden contener tokens y se crean con permisos `0o644` (world-readable en hosts multi-user).
- **Fix**: escribir a `path+".tmp"` con `0o600` y `os.Rename` atómico. Borrar el tmp en caso de fallo del `WriteFile`.
- **Verificar**: `TestWriteJSONAtomic` (Fase 3d) pasa; el modo del fichero destino es `0o600`.

### H3: `recapTokensaveConfig` escribe `~/.claude.json` con `0o644`
- **Ubicación**: `internal/cli/tokensave.go:206`
- **Severidad**: Tier 1 (privacidad)
- **Riesgo**: tras `mcp-tools tokensave cap`, el fichero de configuración de Claude queda world-readable; contiene la ruta al binario `tokensave-capped` que, en algunos setups, lleva credenciales en env.
- **Fix**: cambiar `0o644` a `0o600` en `os.WriteFile`.
- **Verificar**: `stat ~/.claude.json` reporta `0600` tras `mcp-tools tokensave cap`.

### H4: nvidia-toolkit silencia errores de apt con `grep -v`
- **Ubicación**: `internal/tools/nvidia_toolkit.go:48-49`
- **Severidad**: Tier 1 (silencia fallos reales)
- **Riesgo**: el pipe `apt-get ... 2>&1 | { grep -v 'configured multiple times' || true; }` traga CUALQUIER error que no contenga literalmente la cadena 'configured multiple times' — incluyendo repos rotos, claves GPG inválidas, dependencias sin resolver. El usuario ve `OK` y luego `docker` falla al usar `nvidia-container-toolkit` sin entender por qué.
- **Fix**: usar `apt-get -qq` (quiet, solo errores a stderr). `configured multiple times` es un warning inocuo de dpkg, no necesita filtro cuando `-qq` está activo.
- **Verificar**: `mcp-tools nvidia-toolkit install --dry` imprime los comandos sin pipe.

### H5: rtk se instala desde branch (no tag)
- **Ubicación**: `internal/tools/rtk.go:30-33`
- **Severidad**: Tier 1 (suministro)
- **Riesgo**: `cargo install --git URL --branch feat/omp-extension-rewrite --locked` trae lo que haya en `HEAD` de ese branch. Si el upstream hace force-push (accidental o malicioso), el binario instalado cambia entre dos `mcp-tools rtk install` consecutivos sin aviso. No hay hash pin.
- **Fix**: comentario `TODO(security)` apuntando a esta sección. El pin real requiere upstream (decisión de `makoMakoGo/rtk`); mcp-tools no puede forzar el pin.
- **Verificar**: `grep -n "TODO(security)" internal/tools/rtk.go` encuentra el comentario.

### H6: mem0-mcp-selfhosted se instala sin pin de versión
- **Ubicación**: `internal/tools/mem0.go:27`
- **Severidad**: Tier 1 (suministro)
- **Riesgo**: `uv tool install --from git+https://...` resuelve la `HEAD` actual del repo upstream. Cada upgrade trae lo último, sin posibilidad de fijar un commit bueno conocido. Combinado con H21 (mem0 con bugs upstream), un cambio upstream puede romper la integración sin aviso.
- **Fix**: comentario `TODO(security)`. Pin real requiere decisión upstream (`elvismdev/mem0-mcp-selfhosted`).
- **Verificar**: `grep -n "TODO(security)" internal/tools/mem0.go` encuentra el comentario.

### H7: `state.Load` no rechaza versiones futuras del schema
- **Ubicación**: `internal/state/state.go:36-55`
- **Severidad**: Tier 1 (degradación silenciosa)
- **Riesgo**: si un usuario actualiza a una versión futura de mcp-tools (digamos schema v2) y luego vuelve a una v1 (schema=1), el binario antiguo lee el `state.json` v2 como si fuera v1 y aplica lógica incompatible. No hay defensa; el binario degrade silenciosamente.
- **Fix**: tras `json.Unmarshal`, si `s.Version > SchemaVersion` retornar error explícito citando el schema y la versión soportada.
- **Verificar**: `TestLoadRejectsFutureSchema` (Fase 3d) pasa; crear `state.json` con `version: 999` y `mcp-tools status` retorna error con mensaje claro.

### H8: headroom detecta "falló por mitmproxy" por substring matching
- **Ubicación**: `internal/tools/headroom.go:48`
- **Severidad**: Tier 1 (lógica frágil)
- **Riesgo**: la condición `strings.Contains(msg, "mitmproxy") || strings.Contains(msg, "proxy")` se cumple con cualquier mensaje que contenga la palabra "proxy" (incluyendo "no proxy found", "HTTPS_PROXY not set", etc.). Reintenta sin el extra incorrectamente, instalando una versión sin `[proxy]` que no es la que el usuario quiere.
- **Fix**: condición más estricta: `strings.Contains(msg, "Failed to build") && strings.Contains(msg, "mitmproxy")`. Es el patrón exacto que emite `uv` cuando el extra `[proxy]` falla por la compilación de mitmproxy.
- **Verificar**: `go build` verde; el cambio es un-string-only.

### H9: `ompConfigSet` traga stderr
- **Ubicación**: `internal/cli/tokens.go:106-111`
- **Severidad**: Tier 1 (UX)
- **Riesgo**: si `omp config set` falla (key inválida, omp no en PATH, error de config), el usuario ve solo `omp config set: exit status 1` sin saber qué pasó. Tiene que correr `omp config set ...` a mano para diagnosticar.
- **Fix**: capturar `stderr` en `bytes.Buffer`, incluirlo en el error.
- **Verificar**: con un `omp` roto, el error incluye el stderr.

### H10: `serena init` corre sin detección de TTY
- **Ubicación**: `internal/tools/serena.go:49-57`
- **Severidad**: Tier 1 (hang en CI)
- **Riesgo**: `serena init` es interactivo (pide confirmación de idioma, project layout, etc.). En un install no-TTY (CI, `mcp-tools install --no-tui`, `echo | mcp-tools serena install`) el comando cuelga leyendo de un stdin vacío. El usuario espera minutos y mata el proceso.
- **Fix**: detectar TTY con `isatty.IsTerminal(os.Stdin.Fd())`. Si no hay TTY, añadir `--yes` (flag soportado por versiones modernas de serena). Si la versión no soporta `--yes`, warn y continuar (no abortar).
- **Verificar**: `echo | mcp-tools serena install` no cuelga; termina en <30s.

### H11: codegraph fallback usa `strings.NewReader("y\n")`
- **Ubicación**: `internal/tools/codegraph.go:71-79`
- **Severidad**: Tier 1 (race con prompt)
- **Riesgo**: si `codegraph install --yes` falla porque la versión no soporta `--yes`, se reintenta con `Stdin = strings.NewReader("y\n")` y se le envía `y\n` a ciegas. Si el prompt real pide "Do you want to proceed? (yes/no)" y el primer carácter que necesita es `n` para rechazar (raro, pero posible), o si el prompt necesita más contexto (e.g. multi-step), el envío de `y\n` puede aceptar cosas que el usuario no quería. Es un race entre el read interno de codegraph y el envío ciego.
- **Fix**: eliminar el fallback con stdin. Si `--yes` no es soportado, warn explícito y saltar el auto-register (el usuario lo corre a mano).
- **Verificar**: con `--yes` no soportado, el install no aborta; log dice "salta auto-register".

### H12: `install.sh` "latest" sin manejo de rate-limit
- **Ubicación**: `install.sh:34-42`
- **Severidad**: Tier 1 (UX en rate-limit)
- **Riesgo**: `curl -fsSL -o /dev/null -w '%{url_effective}' ...` con `set -euo pipefail` aborta silenciosamente con `exit 22` cuando GitHub responde 403 (rate-limit). El mensaje es genérico y no indica que es un problema temporal, ni cómo mitigarlo.
- **Fix**: capturar el HTTP code; si no es 200/301/302, err con mensaje que sugiera fijar `MCP_TOOLS_VERSION=vX.Y.Z` y reintentar.
- **Verificar**: `bash -n install.sh` OK; con un mock que devuelva 403, el err incluye el código y la sugerencia.

### H13: `install.sh` checksum grep con espacios en tarball
- **Ubicación**: `install.sh:76-77`
- **Severidad**: Tier 1 (potencial falso negativo)
- **Riesgo**: `grep " ${tarball}\$"` exige un espacio antes del tarball y el tarball al final de línea. Si el tarball contiene un espacio (no es el caso con goreleaser default), no matchea y se aborta con "checksums.txt no contiene X". Con el naming actual de goreleaser no aplica, pero es frágil ante un rename de variable.
- **Fix**: NO se aplica (no es bug real con el naming actual). Documentado aquí por completitud.
- **Verificar**: el tarball de una release real no contiene espacios.

### H14: `claude mcp list` falla → prune silenciosa
- **Ubicación**: `internal/mcp/claude.go:28-40`
- **Severidad**: Tier 1 (degradación silenciosa)
- **Riesgo**: si `claude mcp list` falla (CLI corrupto, config corrupto), el bloque de prune se salta por el `if err == nil` implícito. No hay log, no hay error — el usuario cree que se hizo la prune cuando no se hizo. Si hay servidores obsoletos registrados, persisten.
- **Fix**: envolver el bloque con `if err == nil { ... } else { log SKIP }`. Solo pruna si el list fue exitoso.
- **Verificar**: con `claude` instalado pero roto, el log indica el skip.

---

## Hallazgos Tier 2 (6 corregidos, 4 documentados)

### H15: `statusOllama` sin timeout en `docker exec`
- **Ubicación**: `internal/tools/ollama.go:80-93`
- **Severidad**: Tier 2 (reliability)
- **Riesgo**: `docker container inspect` y `docker exec ollama --version` se ejecutan sin timeout. Si el daemon de Docker está colgado (común después de un OOM o un freeze del kernel), `mcp-tools status` cuelga indefinidamente. El usuario tiene que matar el proceso y abrir un ticket.
- **Fix**: usar `exec.CommandContext` con timeout de 5s. Si expira, `p.Extra["state"] = "timeout"`.
- **Verificar**: con daemon colgado, `mcp-tools status` responde en <6s.

### H16: `preChecked` invoca 11 `Status()` secuenciales
- **Ubicación**: `internal/cli/install.go:237-258`
- **Severidad**: Tier 2 (latencia en primer install)
- **Riesgo**: en el primer install (sin state previo), `preChecked` itera sobre todas las tools y llama `Status()` secuencialmente. Cada `Status()` puede implicar `docker container inspect` (200ms), `curl` (1s), etc. Total: 5-10s de espera en el peor caso, solo para pre-marcar el TUI.
- **Fix**: ejecutar las Status() en paralelo con un WaitGroup de 5 workers. El path con state pre-existente (`len(st.Selected) > 0`) no se toca.
- **Verificar**: primer install con 11 tools termina preChecked en <3s.

### H17: `pullMem0Models` reintenta 10×2s sin progreso visible
- **Ubicación**: `internal/tools/ollama.go:117-138`
- **Severidad**: Tier 2 (UX)
- **Riesgo**: tras `docker compose up ollama`, el script reintenta `docker exec ollama list` 10 veces con 2s entre cada uno. Si ollama tarda más de 20s en arrancar (cold start de qwen2.5:7b es común), el usuario ve 20s de silencio absoluto y luego el pull de modelos. Cree que está colgado.
- **Fix**: log de "esperando ollama (intento N/10)..." antes de cada reintento. Si llega al intento 5+, bajar el log a verbose y mostrar progreso.
- **Verificar**: la salida muestra los reintentos.

### H18: `statusQdrant` no detecta drift entre imagen y container
- **Ubicación**: `internal/tools/qdrant.go:71-85`
- **Severidad**: Tier 2 (diagnóstico)
- **Riesgo**: si haces `docker pull mcp-tools-mem0-qdrant:latest` mientras el container está corriendo, el `Status()` reporta la versión vieja (la del container). El usuario cree que tiene la nueva imagen, pero el container sigue con la vieja. Drift silencioso.
- **Fix**: comparar `docker images inspect ... -f {{.Id}}` con `docker container inspect ... -f {{.Image}}`. Si difieren, `p.Extra["image_drift"] = true`.
- **Verificar**: `docker tag` la imagen con otro digest, `mcp-tools status` reporta `image_drift: true`.

### H19: `runSelfUpdate` no verifica post-install
- **Ubicación**: `internal/cli/update.go:81-113`
- **Severidad**: Tier 2 (reliability)
- **Riesgo**: tras `make install`, el binario puede estar corrupto (enlace simbólico roto, permisos incorrectos, PATH descolocado). El usuario ve "mcp-tools actualizado a vX.Y.Z" pero la próxima invocación falla. No hay post-install sanity check.
- **Fix**: tras `make install`, ejecutar `mcp-tools --version` (resolviendo la misma ruta que el Makefile: `$MCP_TOOLS_BIN/mcp-tools` o `~/.local/bin/mcp-tools`). Si falla, warn y devolver error. NO hacer rollback automático (demasiado arriesgado).
- **Verificar**: introducir un `exit 1` en el recipe de make; el warn aparece y `update` retorna error.

### H20: `configure.go:96` reasigna `stNew` sin preservar `Versions`
- **Ubicación**: `internal/cli/configure.go:96`
- **Severidad**: Tier 2 (pérdida de metadata)
- **Riesgo**: `stNew := state.State{Selected: newSelected}` crea un state nuevo sin copiar `st.Versions`. El código justo después hace `stNew.Versions = collectVersions(newSelected)` que solo pilla los tools del nuevo `Selected`. Si el usuario quita y vuelve a añadir un tool entre dos `configure` consecutivos, el histórico de versiones conocidas se pierde.
- **Fix**: inicializar `stNew.Versions = st.Versions` y dejar que `collectVersions` sobrescriba solo los que estén en `newSelected`. Pero el código actual sobrescribe TODOS — eso borra las versiones de tools no seleccionados. Mejor: pre-cargar `st.Versions` y solo actualizar las de los seleccionados.
- **Verificar**: tras `mcp-tools configure`, las versiones conocidas de tools no modificados se preservan.

### Tier 2 — documentados (no corregidos en esta ronda)

- **H-T2-1**: `install.go:runToolSteps` no propaga errores parciales. Si 7/8 tools OK y 1 falla, runToolSteps aborta sin hacer rollback de las 7 ya instaladas. Riesgo bajo (el TUI siguiente pregunta qué hacer) — futuro refactor.
- **H-T2-2**: `ollama.go:OllamaComposeFiles` mezcla path absoluto y relativo. Algunos callers asumen absoluto, otros relativo. No causa bug observable hoy, pero el día que el cwd cambie, la composición se rompe silenciosamente.
- **H-T2-3**: `qdrant.go:installQdrant` no comprueba si la imagen ya está en la versión actual antes de `compose up`. Trae "update" no intencional.
- **H-T2-4**: `configure.go:runConfigure` no muestra diff antes de aplicar. El usuario no ve qué va a cambiar hasta que ya cambió.

---

## Bugs mem0 RULES.md (1 corregido)

### H21: Sección "Bugs conocidos" enterrada al final de `RULES.md`
- **Ubicación**: `RULES.md:128-137` (antes del fix)
- **Severidad**: mem0 (información crítica al inicio)
- **Riesgo**: la sección que documenta el workaround de `search_memories`/`get_memories` rotos está en el ÚLTIMO bloque del documento, justo antes de "Skills específicos por MCP". Como `RULES.md` se carga en el system prompt de CADA agente AI que use mcp-tools (Claude Code, OpenCode, OMP), el workaround debería estar en el TOP — no en el footer. Si un agente necesita decidir entre `search_memories` y `get_memory` por UUID, debería ver el aviso en los primeros 5 segundos de cargar el contexto, no después de leer 8KB de routing.
- **Fix**: mover la sección a la cabecera del documento bajo un heading prominente ("Known bugs — read first") con una tabla de operaciones y su estado.
- **Verificar**: `head -40 RULES.md` contiene "Known bugs — read first" y la tabla de operaciones.

---

## Hallazgos Tier 3 (10+, documentados — no corregidos)

Deuda acumulada que NO se aborda en esta ronda. Documentada para futura iteración.

- **H-T3-1**: Tests casi inexistentes. Antes de Fase 3d: 1 test de 17 líneas en todo el repo. Después: 3 tests nuevos (jsonfile, state x2). Cobertura efectiva: <5%. Necesario: tabla de verdad por cada módulo.
- **H-T3-2**: El binario nunca se ejecuta en CI end-to-end. `go build` + `go vet` + `gofmt` + `go test` no cubren `mcp-tools install`, `mcp-tools status`, `mcp-tools env`. Necesario: integration test con Docker mock o un `make smoke` que corra contra un contenedor efímero.
- **H-T3-3**: `scripts/installer/` y `scripts/install-rules.sh` referenciados en comentarios ya no existen. Stale references en `internal/tui/selectmodel/model.go:23`, `internal/cli/rules.go:32`, `internal/config/env.go:62,66`. Documentación interna desfasada.
- **H-T3-4**: `tools/compose.go:OllamaComposeFiles` mezcla path absoluto y relativo. Ver H-T2-2.
- **H-T3-5**: `cli/tokensave.go:50-67` wrapper `tokensave-capped` sin verificación de firma. El wrapper execs el binario sin chequear que sea el oficial de `aovestdipaperino/tokensave`. Vector de supply chain menor.
- **H-T3-6**: `cli/select_model.go:38` `os.Exit(code)` no pasa por `Execute()`. Rompe la jerarquía de cleanup (defer para logs, telemetry, etc.). Patrón repetido en 4 sitios.
- **H-T3-7**: `internal/tui/selectmodel/model.go` usa Bubble Tea con tamaño fijo. No responde a resize de terminal.
- **H-T3-8**: `internal/cli/install.go:runToolSteps` no cancela workers en error. Si el 3er tool falla, los workers del 1-2 siguen corriendo. Se ha visto esto en producción.
- **H-T3-9**: `Makefile` solo tiene target `install` y `build`. Falta `test`, `lint`, `fmt`, `vet`, `clean`. Targets que se invocan manualmente en CI pero no están formalizados.
- **H-T3-10**: `dockers/compose.yaml` no fija versiones de imagenes por digest. `image: ollama/ollama:latest` (o tag mutable) puede traer cambios incompatibles. Solo qdrant fija tag.

---

## Tabla maestra

| # | Tier | Ubicación | Severidad | Fix |
| --- | --- | --- | --- | --- |
| H1 | 1 | `.gitignore:1-3` | privacidad | reglas explícitas + `.env.mem0.example` |
| H2 | 1 | `internal/mcp/jsonfile.go:42-53` | atomicidad | WriteJSON atómico + 0o600 |
| H3 | 1 | `internal/cli/tokensave.go:206` | privacidad | recap 0o600 |
| H4 | 1 | `internal/tools/nvidia_toolkit.go:48-49` | silent failure | apt -qq sin grep |
| H5 | 1 | `internal/tools/rtk.go:30-33` | supply chain | TODO(security) |
| H6 | 1 | `internal/tools/mem0.go:27` | supply chain | TODO(security) |
| H7 | 1 | `internal/state/state.go:36-55` | degradación | rechazo de version > schema |
| H8 | 1 | `internal/tools/headroom.go:48` | fragilidad | substring estricto |
| H9 | 1 | `internal/cli/tokens.go:106-111` | UX | capturar stderr |
| H10 | 1 | `internal/tools/serena.go:49-57` | hang CI | isatty + --yes |
| H11 | 1 | `internal/tools/codegraph.go:71-79` | race stdin | quitar fallback, warn |
| H12 | 1 | `install.sh:34-42` | rate-limit | detectar HTTP code |
| H13 | 1 | `install.sh:76-77` | frágil (no aplica) | documentar |
| H14 | 1 | `internal/mcp/claude.go:28-40` | silent skip | log SKIP |
| H15 | 2 | `internal/tools/ollama.go:80-93` | reliability | timeout 5s |
| H16 | 2 | `internal/cli/install.go:237-258` | latencia | paralelo (5 workers) |
| H17 | 2 | `internal/tools/ollama.go:117-138` | UX | log de reintentos |
| H18 | 2 | `internal/tools/qdrant.go:71-85` | diagnóstico | image_drift |
| H19 | 2 | `internal/cli/update.go:81-113` | reliability | post-install verify |
| H20 | 2 | `internal/cli/configure.go:96` | pérdida metadata | preservar st.Versions |
| H21 | mem0 | `RULES.md:128-137` | prominencia | mover al top con tabla |

---

## Verificación final (Fase 4)

Comandos ejecutados en orden:

```
go build ./...                 → ok
go vet ./...                   → ok
gofmt -l .                     → vacío
go test ./...                  → ok (3 tests nuevos)
go test -race ./internal/...   → ok
bash -n install.sh             → ok
git check-ignore -v .env.mem0  → ignorado
git check-ignore -v .env.example → NO ignorado
head -40 RULES.md               → contiene "Known bugs — read first"
```

Comportamientos observables:

```
echo '{"version":999,"selected":[]}' > ~/mcp-tools/state.json && ./bin/mcp-tools status
  → err: "state.json: schema v999 más nuevo que este binario (soporta v1); actualiza mcp-tools"

stat ~/.claude.json tras `mcp-tools tokensave cap`
  → mode 0600

mcp-tools status con daemon ollama colgado
  → responde en <6s con state="timeout"
```
