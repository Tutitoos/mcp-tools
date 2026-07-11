# Auditoría web, instalación y compatibilidad — `mcp-tools`

- **Fecha:** 2026-07-11
- **Árbol auditado:** `/home/tutitoos/mcp-tools`
- **HEAD:** `65dbcbdba3b971b2beb1d00b7be82273bde171ce` (`docs(audit): round-3 read-only bug audit at e2df05a`)
- **Objetivo:** auditar la aplicación web, el ciclo install/update/uninstall y la compatibilidad real Linux/macOS.
- **Política:** auditoría read-only. No se modificaron código, lockfiles, workflows ni documentación existente. El único archivo nuevo de la auditoría es este informe.

## Resumen ejecutivo

| ID | Severidad | Área | Estado |
|---|---:|---|---|
| WEB-03 | **Critical** | Panel HTTP sin autenticación/origin gate, escucha en `0.0.0.0` y expone `.env`/`.env.mem0` | **Confirmado por código y reproducción runtime** |
| INS-01 | **High** | `go.mod` requiere Go 1.25.0; installer/CI/release fijan Go 1.24.4 | **Confirmado por manifest mismatch y reproducción con toolchain limpia** |
| INS-03 | **High** | Autodetección puede elegir system mode y escribir `/etc` sin `sudo` ni fallback user | **Confirmado por código y fixture runtime** |
| INS-04 | **High** | Tools npm declaradas `DeploySudo` no ejecutan `sudo` | **Confirmado por código y fixture npm** |
| WEB-01 | **Medium** | `useJobStream` conserva estado/eventos del job anterior al cambiar `jobId` | **Confirmado por code-trace; runtime de React no reproducido** |
| WEB-02 | **Medium** | Sync de Settings muestra éxito por `202 Accepted`, pero no muestra fallos del job asíncrono | **Confirmado por wiring; reproducción HTTP aislada no ejecutada** |
| INS-05 | **Medium** | `make install` ignora el fallo de `web --restart` | **Confirmado por Makefile y reproducción con stub** |
| INS-07 | **Medium** | Unit systemd no escapa `ExecStart`/`EnvironmentFile` con espacios | **Confirmado por template y `systemd-analyze verify`** |
| INS-08 | **Medium** | Drift entre `.env.example` (`127.0.0.1`) y defaults runtime (`0.0.0.0`) | **Confirmado por código y archivos actuales** |
| INS-09 | **Medium** | Self-update web documentado, pero `/api/update/self` y botón Settings no existen | **Confirmado por wiring y `POST` runtime 404** |
| INS-02 | **Medium** | macOS cae a instrucciones `serve`/`nohup`; no instala supervisor launchd | **Confirmado por código; cross-build Darwin OK; runtime macOS no disponible** |
| INS-06 | **Low** | Trap temprano de `install.sh` merece endurecimiento; repro de `unbound variable` no concluyente | **No reproducido; `unverified — confirm first`** |

Se mantienen fuera de este informe como contexto histórico, no como nuevos hallazgos: exposición y filtración de estado ya descritas en `docs/AUDIT-2026-07-11.md`, problemas de `ReadTimeout`, timeout Docker, carrera Load→Save y pinning de terceros. WEB-03 se revalida aquí porque la solicitud exige comprobar la superficie web actual.

## 1. Hallazgos confirmados en la web actual

### WEB-03 — Panel abierto en LAN y filtración de configuración

- **Severidad:** Critical.
- **Confianza/estado:** Alta; confirmado por código y reproducción runtime.
- **Archivos/símbolos:**
  - `internal/cli/constants.go:12-17`, `DefaultBind = "0.0.0.0"`.
  - `internal/web/router.go:52-100`, rutas mutantes sin auth, CSRF u origin gate.
  - `internal/web/api.go:78-99`, `handleStatus` devuelve `env` y `env_mem0` completos.
  - `dockers/compose.yaml:13-14` y `dockers/qdrant-compose.yml:6-7`, publican puertos usando `MCP_TOOLS_BIND`.
- **Comportamiento:** sin `--bind`, `serve` escucha en todas las interfaces. El router documenta explícitamente que la API queda abierta. `GET /api/status` devuelve contenido de los archivos de entorno, incluyendo rutas locales, usuario y bind.
- **Reproducción mínima, aislada y no destructiva:**

  ```text
  go build -o /tmp/mcp-tools-audit/mcp-tools ./cmd/mcp-tools
  /tmp/mcp-tools-audit/mcp-tools serve --port 18890
  mcp-tools web listening on 0.0.0.0:18890

  ss -lnt | grep 18890
  LISTEN ... *:18890 ...

  curl -s http://127.0.0.1:18890/api/status
  keys: ['compose_services', 'docker_running', 'env', 'env_mem0', 'state']
  env.MCP_TOOLS_BIND = 0.0.0.0
  env.MEM0_USER_ID = tutitoos
  env_mem0.MEM0_HISTORY_DB_PATH = /home/tutitoos/mcp-tools-data/mem0/history/history.db
  ```

  La petición se realizó sin token, cookie, `Origin` especial ni header de autenticación. No se ejecutó ningún `POST` mutante.
- **Impacto:** cualquier host con acceso a la interfaz puede leer configuración y llamar a instalaciones, upgrades, uninstall, cambios de `.env`, servicios Docker, modelos, plugins y sincronizadores. El panel no verifica `Origin`, `Sec-Fetch-Site` ni un bearer token.
- **Alcance:** afecta al default `serve`/`install` y a cualquier despliegue donde el firewall no compense explícitamente la decisión. El README reconoce este contrato, por lo que es una exposición de seguridad deliberada en código, no una simple contradicción documental.
- **Recomendación:** cambiar el default del panel a loopback y exigir opt-in para LAN; redactar `env`/`env_mem0` en `/api/status`; introducir autenticación o al menos un gate explícito para mutaciones; usar publish loopback para Compose por defecto. Mantener una opción documentada para administración LAN autenticada.

### WEB-01 — Cambio rápido A→B conserva estado y puede aplicar eventos viejos

- **Severidad:** Medium.
- **Confianza/estado:** Media-alta; confirmado por code-trace actual, sin runtime React determinista.
- **Archivos/símbolos:** `web/app/lib/sse.ts:24-174` (`useJobStream`), `web/app/routes/jobs.tsx:123-131`; también consumidores en `models.tsx`, `tools.tsx` y `plugins.tsx`.
- **Comportamiento:** al entrar en el efecto dependiente de `jobId`, sólo se reinician `pending.current` y `flushScheduled.current` (`sse.ts:42-43`). No se ejecuta `setState({lines: [], done: false, ok: false, error: null, open: false})`. El cleanup aborta el reader, pero las continuaciones async de handshake, `catch`, flush y finalización pueden llamar a `setState` después del cleanup.
- **Secuencia mínima:**
  1. montar job A;
  2. resolver handshake de A o dejar un `catch` pendiente;
  3. seleccionar B antes de finalizar A;
  4. cleanup A aborta el reader;
  5. se monta B, pero el estado React sigue siendo el de A;
  6. un `setState` tardío de A puede dejar `done`, `error`, `open` o líneas `a-*` en el visor de B.
- **Resultado esperado para B:** `lines=[]`, `done=false`, `error=null`, `open=false` antes del primer evento B y ausencia de líneas `a-*` posteriores. El código actual no garantiza ese contrato.
- **Impacto:** visor de Jobs/diálogos de tools puede mostrar que B terminó, falló o contiene logs del job anterior. No se observó pérdida de datos del backend; es un bug de estado de UI.
- **Límite de evidencia:** `web/package.json` no contiene Vitest/Jest/Testing Library ni script de test React. `pnpm run typecheck` pasa, pero no prueba el ciclo de efectos. No se añadieron dependencias ni se alteró el lockfile para crear un runner.
- **Recomendación:** resetear el estado al inicio de cada efecto con `jobId`; además, proteger cada actualización async con una generación/cancel flag del efecto, no sólo el bucle de lectura.

### WEB-02 — Settings confirma enqueue, no resultado del sync

- **Severidad:** Medium.
- **Confianza/estado:** Media; confirmado por wiring estático; reproducción HTTP de un fallo de job no ejecutada.
- **Archivo/símbolo:** `web/app/routes/settings.tsx:98-104`, `syncMut`; `internal/web/api.go:351-368`, `handleSync`.
- **Comportamiento:** `syncMut` sólo define `mutationFn` y `onSuccess`. Los tres botones se deshabilitan mientras el POST está pendiente, pero no hay `onError` local. El endpoint siempre encola el trabajo y devuelve `202` con `job_id`; el error real ocurre más tarde dentro de `safeGo` y sólo queda en el job/SSE.
- **Reproducción mínima contractual:** un fixture de `/api/skills/sync` que devuelva HTTP 500 sí activaría el error de React Query, pero el código no registra `onError`; un job que devuelva error después de `202` tampoco puede activar el `onError` del POST. `EnvTable.submit` en las líneas 58-70 sí tiene `try/catch` y `toast.error`, mostrando una diferencia interna de UX.
- **Impacto:** el toast `toast.success("sync ...")` comunica éxito aunque sólo se haya creado el job. Si la ejecución falla, el usuario debe abrir Jobs y descubrir el fallo; Settings no lo muestra localmente.
- **Alcance:** no es un error de serialización ni de HTTP wrapper. `api` y `ApiError` ya convierten errores HTTP y se reutilizan correctamente.
- **Recomendación:** mantener el toast como “encolado”, no “éxito”; enlazar el `job_id` al visor o abrir un diálogo SSE; mostrar el error final del job en Settings. Añadir `onError` para fallos inmediatos del POST.

### WEB-04 / INS-09 — Self-update web prometido pero inalcanzable

- **Severidad:** Medium.
- **Confianza/estado:** Alta; confirmado por wiring y reproducción runtime 404.
- **Archivos/símbolos:**
  - `internal/orchestrator/selfupdate.go:14-17`, comentario afirma `/api/update/self`.
  - `internal/orchestrator/orchestrator.go:289-303`, `UpdateSelf` existe.
  - `internal/cli/update.go:15-31`, la ruta CLI `mcp-tools update` sí llama a `orchestrator.UpdateSelf`.
  - `internal/web/router.go:52-100`, no registra `/api/update/self`.
  - `web/app/routes/settings.tsx:91-193` y `web/app/routes.tsx:26-68`, no hay botón ni ruta UI de Update.
- **Reproducción:**

  ```text
  curl -i -X POST http://127.0.0.1:18890/api/update/self
  HTTP/1.1 404 Not Found
  ```

  El log sólo registra la petición; no se ejecutaron `git`, `pull`, `make` ni `install`.
- **Impacto:** usuarios y mantenedores pueden creer que Settings expone self-update cuando sólo existe el comando CLI. El endpoint documentado no hace nada y devuelve 404.
- **Recomendación:** decidir una sola opción: eliminar/corregir los comentarios que prometen wiring web, o implementar una ruta explícita con confirmación, autenticación, locking y tratamiento especial del proceso que se actualiza a sí mismo. No exponer un self-update remoto sin autorización fuerte.

## 2. Hallazgos confirmados en instalación y compatibilidad

### INS-01 — Toolchain incompatible entre `go.mod`, installer y CI

- **Severidad:** High.
- **Confianza/estado:** Alta; manifest mismatch y prueba con Go 1.24.4 aislado.
- **Archivos:** `go.mod:3` (`go 1.25.0`), `install.sh:12` (`REQUIRED_GO_VERSION="1.24.4"`), `.github/workflows/ci.yml:18-20`, `.github/workflows/release.yml:26-28`.
- **Reproducción:** se descargó únicamente el tarball oficial a `/tmp/go124-audit`.

  ```text
  /tmp/go124-audit/go/bin/go version
  go version go1.24.4 linux/amd64

  GOTOOLCHAIN=local /tmp/go124-audit/go/bin/go test ./...
  go: go.mod requires go >= 1.25.0 (running go 1.24.4; GOTOOLCHAIN=local)

  GOTOOLCHAIN=auto /tmp/go124-audit/go/bin/go test ./...
  ... paquetes OK ...
  ```

- **Comportamiento:** con `GOTOOLCHAIN=local`, el toolchain que instala/usa el proyecto rechaza el módulo. Con `auto`, Go 1.24 puede seleccionar/descargar un toolchain posterior, pero eso introduce una descarga implícita y no es una garantía reproducible del installer/CI.
- **Fuente primaria:** la documentación oficial de Go Toolchains indica que la línea `go` es el mínimo obligatorio y que `GOTOOLCHAIN=local` rechaza módulos que requieren una versión posterior; `auto` puede seleccionar/descargar un toolchain más nuevo.
- **Impacto:** `install.sh` anuncia/instala Go 1.24.4 para un módulo cuyo mínimo actual es 1.25.0. La build sólo funciona si el entorno permite auto-descarga o ya contiene Go 1.25.
- **Recomendación:** fijar installer y workflows a Go 1.25.x, o bajar `go.mod` si el proyecto realmente debe soportar 1.24. No dejar la compatibilidad dependiendo de `GOTOOLCHAIN=auto` sin documentarlo y verificarlo.

### INS-02 — macOS instala binario, pero no servicio supervisor

- **Severidad:** Medium.
- **Confianza/estado:** Alta para el comportamiento del código; runtime macOS no disponible.
- **Archivos:** `internal/cli/install.go:80-89,190-204`, `internal/systemd/detect.go:59-78`.
- **Comportamiento:** cuando `DetectMode` devuelve `ModeNone`, `runInstall` llama a `printNoSystemdFallback`, imprime comandos `serve` y `nohup` y devuelve `nil`. No se crea un plist launchd, no se habilita un supervisor y no se deja un listener activo por esa misma operación.
- **Compatibilidad verificada:**
  - `GOOS=darwin GOARCH=amd64 go build -o /tmp/mcp-tools-audit/mcp-tools-darwin-amd64 ./cmd/mcp-tools` → OK; archivo Mach-O x86_64.
  - `GOOS=darwin GOARCH=arm64 go build -o /tmp/mcp-tools-audit/mcp-tools-darwin-arm64 ./cmd/mcp-tools` → OK; archivo Mach-O arm64.
  - No se ejecutó un binario Darwin sobre Linux.
- **Fuente primaria:** Apple recomienda `launchd` para daemons/agentes y requiere un plist con `Label`, `ProgramArguments` y, según el caso, `KeepAlive`/sockets. El repositorio no genera ninguno.
- **Impacto:** `mcp-tools install` puede terminar correctamente en macOS y dejar sólo instrucciones manuales. La capa “release binary” es compatible; la capa “service supervisor” es fallback manual, no instalación completa.
- **Recomendación:** documentar explícitamente “foreground/manual” como resultado exitoso en macOS, o implementar un LaunchAgent/LaunchDaemon. No afirmar que `install` deja el panel activo bajo supervisión en macOS.

### INS-03 — Autodetección system mode sin elevación

- **Severidad:** High.
- **Confianza/estado:** Alta; fixture runtime aislado.
- **Archivos:** `internal/systemd/detect.go:59-78`, `internal/cli/install.go:73-89`, `internal/systemd/install.go:10-56`.
- **Reproducción:** `systemctl` falso en `/tmp/install-audit/fake-bin/systemctl` hizo fallar `systemctl --user status` y devolver éxito para `systemctl is-system-running`. Ejecutado como usuario no root:

  ```text
  PATH=/tmp/install-audit/fake-bin:$PATH \
  HOME=/tmp/install-audit/install-mock/home \
  /tmp/mcp-tools-audit/mcp-tools install --port 18892 --no-open --mode auto

  EXIT=1
  error: systemd install: systemd: write /etc/systemd/system/mcp-tools-web.service (¿sudo?):
  open /etc/systemd/system/mcp-tools-web.service: permission denied
  ```

- **Comportamiento:** `DetectMode` selecciona `ModeSystem`; `systemd.Install` escribe directamente en `/etc/systemd/system` y el propio comentario dice que no eleva. No se invoca `sudo` ni se reintenta `ModeUser`. `runInstall` también descarta el error de `DetectMode` con `mode, _ := ...`.
- **Impacto:** un host Linux con systemd de sistema accesible pero sin permisos root falla aunque una instalación user pudiera ser viable. El mensaje menciona `sudo`, pero el programa no lo ejecuta.
- **Recomendación:** preferir explícitamente `ModeUser` cuando el usuario no es root; si se selecciona `ModeSystem`, comprobar privilegios y emitir una instrucción accionable o usar un mecanismo de elevación interactivo. No descartar el error de detección.

### INS-04 — `DeploySudo` no se traduce a ejecución con `sudo`

- **Severidad:** High para instalaciones desde panel/CLI no root.
- **Confianza/estado:** Alta; código y fixture npm.
- **Archivos:** `internal/tools/codex.go:13-45,48-82`, `internal/tools/gemini.go:12-44,47-79`, `internal/orchestrator/partition.go:89-115`.
- **Comportamiento:** Codex y Gemini tienen `Deploy: DeploySudo`, pero sus closures usan directamente `exec.Command("npm", "install", "-g", package)` y `exec.Command("npm", "uninstall", "-g", package)`. `runInlineTools` sólo cambia el texto de ayuda a “sudo — puede pedir contraseña”; no antepone `sudo`.
- **Reproducción con npm falso:**

  ```text
  npm called: install -g @openai/codex
  EACCES: npm cannot write to /usr/local/lib/node_modules without sudo
  exit code: 1

  grep sudo internal/tools/codex.go internal/tools/gemini.go
  no sudo invocation in either file
  ```

- **Impacto:** con npm global en `/usr/local` no escribible, el install falla por permisos. Desde el panel no hay TTY para resolver una contraseña de `sudo`, y desde el CLI el bucket `DeploySudo` no altera el argv real.
- **Alcance:** no se afirma que falle en instalaciones npm con prefix de usuario (`NVM`, `npm config set prefix`, etc.). El bug afecta el contrato declarado `DeploySudo` y los hosts donde el prefix requiere root.
- **Recomendación:** elegir una política única: usar prefix de usuario compatible con panel, o ejecutar una helper de elevación controlada desde CLI interactiva. No declarar `DeploySudo` si sólo se imprime un hint.

### INS-05 — `make install` oculta el fallo de restart

- **Severidad:** Medium.
- **Confianza/estado:** Alta; `Makefile:63-65` y stub reproducible.
- **Código:**

  ```make
  install: build
  	install -m 0755 bin/$(BINARY) $${MCP_TOOLS_BIN:-$$HOME/.local/bin}/$(BINARY)
  	@$${MCP_TOOLS_BIN:-$$HOME/.local/bin}/$(BINARY) web --restart || true
  ```

- **Reproducción:** un binario falso que devuelve 1 para `web --restart` produjo:

  ```text
  direct invocation: exit code 1
  web --restart || true: exit code of full expression: 0
  ```

- **Impacto:** `make install` puede salir verde aunque el daemon siga ejecutando el binario anterior o no se haya reiniciado. Build/deploy y restart no son observables como estados separados.
- **Recomendación:** eliminar `|| true` o convertirlo en warning explícito con resultado final no ambiguo; después verificar que el servicio activo usa el nuevo binario/version.

### INS-07 — Unit systemd rompe paths con espacios

- **Severidad:** Medium.
- **Confianza/estado:** Alta; template y `systemd-analyze verify`.
- **Archivos:** `internal/systemd/unit.go:26-52`.
- **Código relevante:**

  ```ini
  ExecStart={{.BinaryPath}} serve --port {{.Port}} --bind {{.Bind}}
  EnvironmentFile=-{{.EnvFile}}
  ```

- **Fixture:** `UnitConfig{BinaryPath: "/Users/Alice Smith/.local/bin/mcp-tools", EnvFile: "/Users/Alice Smith/mcp-tools/.env"}` renderizó:

  ```ini
  ExecStart=/Users/Alice Smith/.local/bin/mcp-tools serve --port 8888 --bind 0.0.0.0
  EnvironmentFile=-/Users/Alice Smith/mcp-tools/.env
  ```

- **Reproducción:**

  ```text
  systemd-analyze verify /tmp/mcp-tools-audit/unit-with-spaces.service
  unit-with-spaces.service: Command /Users/Alice is not executable: No such file or directory
  ```

- **Impacto:** la unidad no arranca con home/repo/binario o env path que contenga espacios. La documentación de systemd trata `ExecStart` como una línea de comandos tokenizada; el render actual no escapa estos campos.
- **Recomendación:** implementar quoting/escaping según sintaxis de systemd para rutas y argumentos, validar bind y paths antes de escribir la unidad, y mantener un test de `systemd-analyze verify` para espacios y caracteres especiales.

### INS-08 — Defaults de bind contradictorios

- **Severidad:** Medium.
- **Confianza/estado:** Alta; comparación directa del árbol actual.
- **Evidencia:**
  - `.env.example:7`: `MCP_TOOLS_BIND=127.0.0.1`.
  - `internal/orchestrator/env.go:40`: `"MCP_TOOLS_BIND": "0.0.0.0"`.
  - `internal/cli/constants.go:17`: `DefaultBind = "0.0.0.0"`.
  - `internal/cli/serve.go:35`: flag `--bind` usa `DefaultBind`.
  - `.env:7` actual: `MCP_TOOLS_BIND=0.0.0.0`.
  - `README.md:143,149` documenta loopback para Compose mediante `.env.example`, pero también panel abierto por defecto.
- **Impacto:** una persona que copie `.env.example` obtiene un comportamiento distinto del que produce `RunEnv` o `serve` por defecto. Como Compose publica `${MCP_TOOLS_BIND}:11434` y `${MCP_TOOLS_BIND}:6333`, el drift también cambia la exposición de Ollama/Qdrant.
- **Recomendación:** definir una fuente única de verdad. Si el producto mantiene LAN-open por decisión, actualizar `.env.example`; si la intención de seguridad es loopback, cambiar `DefaultBind`, `RunEnv` y la documentación conjuntamente. No presentar los dos defaults como equivalentes.

## 3. Candidato no reproducido

### INS-06 — Trap temprano de `install.sh`

- **Severidad:** Low/Medium potencial.
- **Confianza/estado:** `unverified — confirm first`; no se eleva a bug confirmado.
- **Archivo:** `install.sh:44-77,98-99`.
- **Hipótesis del plan:** `ensure_go_local` instala un trap que referencia `$tmp` antes de que el script principal asigne `tmp`, por lo que un fallo temprano de `curl`/`tar` podría producir un error secundario bajo `set -u`.
- **Verificación ejecutada:** copia de `install.sh` y HOME bajo `/tmp`, `bash -u`, fixtures de `curl`/`tar`, sin tocar `$HOME` real. El fake de `curl` no llegó a producir una reproducción confiable del fallo; las corridas terminaron usando/descargando el tarball real de Go. No se afirma que el bug exista.
- **Resultado:** no reproducido. La secuencia de traps merece una revisión manual y un test shell determinista, pero no debe reportarse como `unbound variable` confirmado.
- **Recomendación:** usar variables inicializadas antes del primer trap y un único cleanup con defaults seguros (`${tmp:-}`, `${go_tmp:-}`), pero confirmar primero con una fixture de `curl` que registre argv y falle específicamente antes de cualquier download real.

## 4. APIs y superficies revisadas sin bug nuevo confirmado

### Cliente `api`/`apiStream`

`web/app/lib/api.ts:17-56` fue revisado:

- serializa objetos JSON y conserva `FormData`;
- siempre establece `Accept`;
- lee respuestas vacías como string vacío y JSON válido cuando existe;
- convierte errores HTTP en `ApiError(status, message, body)`;
- permite `AbortSignal` para SSE;
- las rutas dinámicas de tools, plugins, servicios y logs usan `encodeURIComponent` en los consumidores observados.

No se encontró un bug nuevo confirmado en esta capa. La API sigue sin autenticación, pero eso está cubierto por WEB-03, no debe duplicarse como hallazgo de wrapper.

### Mutaciones de tools/plugins/models/services/configure

Los consumidores observados deshabilitan el botón durante el POST y muestran `onError` para errores inmediatos. Invalidan las queries principales:

- tools: `tools` y `status` en `tools.tsx:142-145`;
- configure: `tools` y `status` en `configure.tsx:50-57`;
- services: `services` y `status` en `services.tsx:99-105`;
- models: `models`/`status` según la mutación en `models.tsx:108-156`;
- plugins: `plugins` en finalización de job y errores locales en `plugins.tsx:131-147`.

Riesgo residual: el refresco tras `202` representa enqueue, no finalización. WEB-02 documenta el caso específico de sync en Settings; no se ha convertido esa misma propiedad en varios IDs duplicados.

### SSR/SPA y cierre del sidecar

Revisados `internal/web/router.go:118-200`, `internal/web/ssr.go:36-175`, `web/app/entry.server.tsx:32-155`, `web/app/entry.client.tsx:30-50` y `web/scripts/check-bundle.mjs`.

- rutas `/api/*` se separan del fallback SPA;
- assets con extensión se sirven desde el embed;
- errores o no-match SSR caen a `index.html` SPA;
- `InitSSR` deshabilita SSR si falta Node/bundle y sirve fallback;
- el sidecar escucha en `127.0.0.1:0`, exige handshake `READY`, tiene timeout de render y `ShutdownSSR` mata/reapea el proceso.

No se confirmó un bug SSR nuevo. `node scripts/check-bundle.mjs` empezó a recorrer rutas y viewport, pero no terminó dentro de 30/60 s; no se utilizará ese timeout como evidencia de fallo funcional.

## 5. Matriz de compatibilidad Linux/macOS

| Capa | Linux | macOS | Estado/evidencia |
|---|---|---|---|
| Release binary amd64 | compatible | compatible | GoReleaser declara ambos GOOS/GOARCH; asset v0.1.8 presente. |
| Release binary arm64 | compatible | compatible | assets `linux_arm64` y `darwin_arm64` presentes. |
| `install.sh` OS/arch | compatible | compatible | acepta sólo `linux`/`darwin`, mapea amd64/arm64, verifica sha256sum o shasum. |
| Asset release vigente | compatible | compatible | v0.1.8: cuatro tarballs + `checksums.txt`; tarballs contienen `mcp-tools` y checksum verificado para los assets descargados. |
| Source build | compatible con Go 1.25 | compatible con Go 1.25 | mismatch INS-01 si sólo se usa Go 1.24.4 local. |
| `make install` build/deploy | compatible con caveat | compatible con caveat | `PNPM`/`GO` dependen del entorno; restart puede ocultar fallo por INS-05. |
| Panel foreground | compatible | compatible | `serve` es Go/HTTP y cross-build Darwin OK. |
| Supervisor systemd | compatible con user/root | Linux-only | macOS no tiene systemd; el código sólo imprime fallback manual. |
| Supervisor launchd | no aplica | **fallback manual** | no hay plist ni `launchctl`; INS-02. |
| Node/SSR | compatible con Node >=20; SPA fallback sin Node | compatible con Node >=20; SPA fallback sin Node | `exec.Command("node")`, fallback implementado. |
| Docker Compose | compatible con Docker Engine/Compose v2 | compatible con Docker Desktop | Docker Desktop Mac exige instalación/aceptación y requisitos propios; no se verificó runtime Docker macOS. |
| Ollama/Qdrant | compatible con Docker; exposición depende bind | compatible con Docker Desktop; sin NVIDIA passthrough | `MCP_TOOLS_BIND` determina publish; default drift INS-08. |
| NVIDIA toolkit | Linux-only, Debian/Ubuntu según README/código | Linux-only | Docker Desktop Mac no ofrece el camino NVIDIA del overlay. Limitación deliberada de plataforma. |
| npm Codex/Gemini | compatible si npm prefix escribible/root | compatible si npm prefix escribible/user | `DeploySudo` no eleva; INS-04. |
| cargo/uv tools | compatible con prerequisitos | compatible con prerequisitos | rustup/uv son cross-platform; toolchain C, permisos y red siguen siendo requisitos. |
| Browser opener | `xdg-open`/manual | `open`/manual | código reconoce ambos. |
| Registro MCP/skills/rules | compatible con rutas POSIX | compatible con rutas POSIX | no se ejecutaron instalaciones reales de terceros. |
| API bind/auth | **bug si se deja default LAN-open** | **bug si se deja default LAN-open** | WEB-03 es independiente del OS. |

### Clasificación final por capa

- **Linux release binary:** compatible; los cuatro assets vigentes existen y los bins son entregables GoReleaser.
- **macOS release binary:** compatible por cross-build AMD64/arm64; no se ejecutó runtime Darwin.
- **Linux `mcp-tools install`:** compatible en user systemd; con autodetección system mode y usuario sin root falla por INS-03.
- **macOS `mcp-tools install`:** binario compatible, pero sólo foreground/manual sin launchd por INS-02.
- **Web foreground:** compatible en ambos OS con Node opcional y SPA fallback; el bind por defecto y la ausencia de auth siguen siendo WEB-03.
- **Docker/Ollama/Qdrant:** compatible con Docker Engine/Desktop si están instalados; no es compatible con NVIDIA GPU passthrough de Docker Desktop Mac.
- **Tools npm:** compatible sólo cuando npm prefix y privilegios ya están satisfechos; no hay traducción efectiva de `DeploySudo`.

## 6. Verificación ejecutada

Todos los comandos se ejecutaron desde `/home/tutitoos/mcp-tools`, salvo que se indique otro `cwd`. Fixtures y descargas se limitaron a `/tmp`.

| Comando/acción | Resultado |
|---|---|
| `git status --short` inicial | 3 archivos modificados por el usuario: `internal/cli/install.go`, `internal/cli/web.go`, `internal/cli/web_test.go`; sin staged/untracked. |
| `git rev-parse HEAD` | `65dbcbdba3b971b2beb1d00b7be82273bde171ce`. |
| `go test ./...` | Todos los paquetes OK. |
| `go vet ./...` | Sin diagnósticos. |
| `cd web && pnpm run typecheck` | `tsc --noEmit`, exit 0. |
| `cd web && pnpm audit --prod` | `No known vulnerabilities found`; no delta git en package/lockfile. |
| `bash -n install.sh scripts/wrappers/mem0-launcher` | Ambos OK. |
| `shellcheck ...` | No ejecutado: `shellcheck` no está instalado localmente; CI sí contiene un step Ubuntu para instalarlo. |
| build Linux de fixture | `/tmp/mcp-tools-audit/mcp-tools` OK. |
| WEB-03 server + `ss` + `curl /api/status` | `0.0.0.0:18890`, endpoint 200 sin auth y env expuesto. |
| `POST /api/update/self` | HTTP 404, sin `git`/`make` ejecutados. |
| Go 1.24.4 aislado + `GOTOOLCHAIN=local` | Rechaza `go.mod` por requerir 1.25.0. |
| Go 1.24.4 + `GOTOOLCHAIN=auto` | Suite pasa mediante selección/descarga de toolchain posterior; no se considera solución reproducible. |
| `GOOS=darwin GOARCH=amd64/arm64 go build` | Ambos builds OK; Mach-O amd64/arm64. |
| INS-03 fake `systemctl` | Selecciona system mode y falla al escribir `/etc`; no invoca sudo; exit 1. |
| INS-04 fake npm | Observa `install -g @openai/codex` sin `sudo`, devuelve EACCES. |
| INS-05 fake binary | `web --restart` exit 1; `... || true` exit 0. |
| INS-07 RenderUnit fixture + `systemd-analyze verify` | Rechaza `/Users/Alice` como comando; path con espacios no escapado. |
| INS-08 grep de defaults | `.env.example=127.0.0.1`, `RunEnv/DefaultBind/.env=0.0.0.0`. |
| `node scripts/check-bundle.mjs` | Recorrió rutas iniciales, pero excedió 30/60 s; no se reporta como bug funcional. |
| INS-06 fixture shell | No reproducción confiable; se conserva como `unverified — confirm first`. |
| `git status --short` final | Idéntico al baseline más este informe nuevo: sólo los 3 archivos del usuario y `docs/AUDIT-WEB-INSTALL-2026-07-11.md`. |

## 7. Cambios laterales y límites

- No se ejecutaron `git pull`, `make install`, `npm install`, `pnpm install`, migraciones ni endpoints mutantes contra el servidor real.
- El `pnpm audit --prod` ajustó el contenido local de `node_modules` para modo production, pero no modificó `web/package.json`, `web/pnpm-lock.yaml` ni el estado git.
- Se crearon y eliminaron fixtures bajo `/tmp`; el test temporal de `internal/systemd` se eliminó inmediatamente y la verificación final no muestra archivo extra en el repositorio.
- No se ejecutó runtime nativo macOS; la evidencia Darwin se limita a cross-build y lectura de APIs/documentación oficiales.
- No se atribuyen como bugs las limitaciones deliberadas de NVIDIA-on-macOS, la ausencia de systemd en macOS o el requisito de Docker Desktop, siempre que se documenten como requisito/fallback.
- No se trata la ausencia de `/api/update/self` como una vulnerabilidad por sí sola; es un contrato/documentación roto y evita que el self-update web se ejecute.

## 8. Fuentes primarias consultadas

- Go toolchain: <https://go.dev/doc/toolchain> — `go` es el mínimo requerido; `GOTOOLCHAIN=local` rechaza módulos que requieren una versión más nueva; `auto` puede seleccionar/descargar otro toolchain.
- Release API: <https://api.github.com/repos/Tutitoos/mcp-tools/releases/latest> — tag observado `v0.1.8`.
- Release assets: <https://github.com/Tutitoos/mcp-tools/releases/tag/v0.1.8> — `checksums.txt` y `linux|darwin × amd64|arm64`.
- Docker Desktop Mac: <https://docs.docker.com/desktop/setup/install/mac-install/> — requisitos, Docker Desktop para Apple Silicon/Intel y configuración privilegiada.
- Docker Compose: <https://docs.docker.com/compose/> — referencia primaria del CLI Compose usado por el proyecto.
- systemd service: <https://www.freedesktop.org/software/systemd/man/latest/systemd.service.html> — sintaxis de unidades, `ExecStart`, `Type`, `EnvironmentFile` y validación de comandos.
- Apple launchd: <https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html> — launchd recomendado para daemons/agentes; plist y `ProgramArguments`/`KeepAlive`.

## Prioridad de corrección

1. **WEB-03:** reducir exposición por defecto y redactar el endpoint de status antes de seguir ampliando mutaciones web.
2. **INS-01:** alinear Go 1.25.0 en `install.sh`, CI y release, o bajar el mínimo del módulo conscientemente.
3. **INS-03/INS-04:** hacer explícitos los privilegios y el modo user/system; no dejar hints que no se traducen en ejecución.
4. **WEB-01:** reset y cancelación por generación en `useJobStream`.
5. **INS-07/INS-05:** corregir escape systemd y no ocultar restart fallido.
6. **WEB-02/INS-09:** distinguir enqueue de éxito final y corregir la promesa de self-update web.
7. **INS-08:** unificar la fuente de verdad del bind.
8. **INS-06:** añadir un test shell determinista sólo después de confirmar el comportamiento real del trap.
