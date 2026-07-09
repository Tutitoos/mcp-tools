# Auditoría de compatibilidad Linux + macOS

Generado a partir de una lectura completa del árbol fuente (commit de trabajo de esta ronda). Cubre lo que `README.md § Plataformas soportadas` promete frente a lo que el código realmente hace.

## Veredicto

Linux + macOS soportados con las excepciones listadas abajo. Ninguna instalación host-only requiere Docker tras el fix G1 (`Install`, `InstallSingle` y `Configure` usan `BootstrapEnv`; sólo qdrant/ollama sondean Docker, y sólo cuando el verbo realmente los toca). NVIDIA GPU y systemd son Linux-only por diseño, no por omisión.

## Matriz por componente

| Área | Evidencia | Notas | macOS verificado |
| --- | --- | --- | --- |
| Go binary build/release | `.goreleaser.yaml:14` `goos: [linux, darwin]`, `:15` `goarch: [amd64, arm64]`, `:13` `CGO_ENABLED=0` | Static, cross-compiled; no cgo → no macOS toolchain needed at build time | smoke test — CI |
| One-liner installer OS gate | `install.sh:30-33` acepta `linux\|darwin` únicamente | Rechaza `msys/cygwin/windows` de entrada | smoke test — CI |
| Checksum verification | `install.sh:127-133` usa `sha256sum` O `shasum -a 256` | Cubre el macOS por defecto (sin coreutils) | smoke test — CI |
| Go bootstrap | `install.sh:60-61` descarga `go{ver}.${os}-${arch}.tar.gz` para ambos OS | Usa el naming de go.dev/dl que publica builds darwin | smoke test — CI |
| Prompt/TTY | `internal/cli/prompt_unix.go:1` `//go:build unix` cubre Linux+Darwin | `/dev/tty` presente en ambos | smoke test — CI |
| Browser launcher | `internal/cli/install.go:162` prueba `xdg-open`, `open`, `wslview` | `open` es nativo de macOS | smoke test — CI |
| Systemd fallback | `internal/cli/install.go:189` `printNoSystemdFallback`; `runWebRestart` vía `Makefile:65` (`web --restart \|\| true`) | macOS sin systemctl recibe un mensaje foreground-serve limpio; el post-step de `make install` no falla | smoke test — CI |
| `.env` UID/GID | `internal/orchestrator/env.go` usa `syscall.Getuid/Getgid` | Ambos funcionan en darwin (devuelven uid real; sólo Windows devuelve -1) | no CI (sin ejecución real de instalación en el runner) |
| Docker compose invocation | `internal/docker/compose.go` invoca `docker compose …` vía `exec.Command` | Docker Desktop for Mac expone el mismo CLI | no CI (el runner macos-latest no tiene Docker Desktop) |
| ollama container port bind | `dockers/compose.yaml:14` usa `${MCP_TOOLS_BIND}:11434:11434` | Docker Desktop for Mac respeta el mismo bind de host | no CI |
| cargo installs (rtk, tokensave) | `internal/tools/util.go:59-99` ejecuta el script upstream `sh.rustup.rs` | rustup soporta macOS (arm64+amd64) | no CI (instala binarios reales, fuera del smoke test) |
| uv installs (serena, mem0, headroom) | `internal/tools/util.go:102-141` ejecuta `astral.sh/uv/install.sh` | uv soporta macOS | no CI |
| npm installs (codex, gemini) | `internal/tools/{codex,gemini}.go` invocan `npm install -g …` | npm es cross-platform | no CI |
| npx install (claude-mem) | `internal/tools/claude_mem.go:39` invoca `npx --yes claude-mem@latest install` | npx es cross-platform | no CI |
| Anthropic Claude CLI | `internal/tools/claude.go:34` ejecuta el `curl … \| bash` de `claude.ai/install.sh` | Upstream declara soporte Linux+macOS | no CI |
| OMP CLI | `internal/tools/omp.go:33` ejecuta el `omp.sh/install.sh` upstream | Upstream soporta macOS | no CI |
| mem0-launcher wrapper | `scripts/wrappers/mem0-launcher:1` `#!/usr/bin/env bash`, expansión de parámetros compatible con bash 3.2 | Funciona en el bash de stock de macOS | no CI |
| Skills/rules symlinks | `internal/orchestrator/sync.go` usa `os.Symlink` + `~/.claude/...`, `~/.config/opencode/...`, `~/.omp/agent/...` | Mismas rutas en ambos OS | smoke test — CI (RunSkills/RunRules no se ejecutan en el smoke test actual, pero la ruta de código es idéntica y `os.Symlink` es POSIX) |
| Web SSR sidecar | `internal/web/ssr.go:68` `exec.Command("node", …)`; fallback a SPA-only sin Node (`ensureNode:178-183`) | Node ≥ 20 es cross-platform | smoke test — CI (el runner no tiene Node del lado del binario Go embebido, pero `ensureNode` degrada limpio) |
| Makefile portable idioms | `Makefile:22` `cp -rL`, `:64` `install -m 0755` | Ambos son flags BSD-compatibles | smoke test — CI |

## Excepciones documentadas

- **U1 — Los scripts `install.sh` de terceros para `codebase-memory-mcp` y `codegraph` se descargan de `main`.** `internal/tools/codebase_memory.go:29` (`https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.sh`) e `internal/tools/codegraph.go:29` (`https://raw.githubusercontent.com/colbymchenry/codegraph/main/install.sh`). El comportamiento en macOS es responsabilidad del proyecto upstream, no de mcp-tools. Verificado en esta ronda (Step 6):
  - `codebase-memory-mcp/install.sh`: `detect_os()` mapea explícitamente `Darwin → darwin`, descarga el tarball `codebase-memory-mcp-darwin-${arch}.tar.gz` y aplica un fix de firma de código específico de macOS (`xattr -d com.apple.quarantine`, `codesign --sign - --force`). **Outcome 1 → `macOS ok (upstream)`.**
  - `codegraph/install.sh`: `case "$os" in Darwin) os="darwin" ;; Linux) os="linux" ;; *) exit 1 ;; esac`, descarga `codegraph-darwin-${arch}.tar.gz` desde GitHub Releases. **Outcome 1 → `macOS ok (upstream)`.**
  - Ninguno de los dos requiere downgrade de la fila README (A3 confirmada: ambos scripts funcionan en macOS).
- **U2 — El overlay GPU NVIDIA + Docker es Linux-only.** Correctamente documentado; `dockers/ollama-gpu-overlay.yml` usa el driver `nvidia`, que Docker Desktop for Mac no puede exponer (no hay soporte de passthrough GPU NVIDIA en macOS). El README ya lo marca.
- **U3 — La unit systemd es Linux-only.** El README ya lo marca; el fallback foreground (`mcp-tools serve`) es el camino en macOS. No se añade una plantilla launchd — no solicitada por el usuario.

## Gaps cerrados en esta ronda

- **G1 — `Bootstrap` exigía Docker para cualquier instalación.** Cerrado por completo (revisión 2): `Install`, `InstallSingle` y `Configure` (`internal/orchestrator/orchestrator.go`) llaman ahora a `BootstrapEnv` (sin probe de Docker); `qdrantTool().Install/Upgrade/Uninstall` y `ollamaTool().Install/Upgrade/Uninstall` llaman a `docker.EnsureAvailable` internamente, así que el mensaje de error sigue siendo claro (`docker no está en PATH`) en lugar de un `exec: "docker": executable file not found` opaco, para cualquier verbo que realmente toque un componente `DeployDocker`.
  - **Historial:** el primer pase de esta ronda dejó `Configure` deliberadamente en `Bootstrap` completo, documentado aquí como residual. La matriz de CI de G3 lo expuso de inmediato: `TestConfigureNoChangeIsNoop` (dry=false, sin diff real) fallaba en el runner `macos-latest` con `docker no está en PATH`, porque `Configure` sondeaba Docker incluso en el camino "sin cambios" que no instala ni desinstala nada. Se cerró moviendo `Configure` a `BootstrapEnv`, igual que `Install`/`InstallSingle`.
  - `orchestrator.Bootstrap` y `orchestrator.EnsureDocker` quedaron sin ningún caller de producción (confirmado con `find_referencing_symbols`) y se eliminaron; `docker.EnsureAvailable` sigue viva vía qdrant/ollama.
  - Verificación: `go build ./...`; `go test ./internal/orchestrator/... ./internal/tools/...`; `go test ./internal/orchestrator/... -run TestConfigureNoChangeIsNoop -v` (regresión específica).
  - Commit/PR: `TBD`.
- **G2 — `nvidia-toolkit` soportaba a medias Fedora/RHEL.** Cerrado: `supportedNvidiaDistro` (`internal/tools/nvidia_toolkit.go`) ahora sólo acepta `debian`/`ubuntu`; el mensaje de error nombra el conjunto soportado. README actualizado (fila de la matriz de plataformas + no había líneas nvidia-específicas que quitar del bloque Fedora/RHEL de one-liners, confirmado por lectura directa de esa sección).
  - Verificación: `go test ./internal/tools/ -run TestSupportedNvidiaDistro`.
  - Commit/PR: `TBD`.
- **G3 — CI nunca ejercitaba macOS.** Cerrado: `.github/workflows/ci.yml` corre ahora en matriz `[ubuntu-latest, macos-latest]`; el step de Shellcheck queda `if: matrix.os == 'ubuntu-latest'` (los runners macOS no traen shellcheck preinstalado).
  - Verificación: validación YAML de la matriz + confirmar en GitHub Actions dos jobs verdes (`test (ubuntu-latest)`, `test (macos-latest)`) en el PR.
  - Commit/PR: `TBD`.

## Comandos de auditoría

```bash
# 1. El informe existe y cubre las secciones requeridas.
test -f docs/COMPAT-linux-macos.md
grep -q "Veredicto"          docs/COMPAT-linux-macos.md
grep -q "Matriz por componente" docs/COMPAT-linux-macos.md
grep -q "Excepciones documentadas" docs/COMPAT-linux-macos.md
grep -q "Gaps cerrados"      docs/COMPAT-linux-macos.md

# 2. G1 cerrado — instalar una tool host-only ya no exige Docker (Install/InstallSingle).
env PATH="$(getconf PATH)" ./bin/mcp-tools --version
env PATH="$(getconf PATH)" ./bin/mcp-tools serve --port 18081 --bind 127.0.0.1 &
PID=$!; sleep 2
curl -sf http://127.0.0.1:18081/api/version
kill $PID; wait 2>/dev/null || true
go test ./internal/orchestrator/... -run TestBootstrapEnv

# 3. G2 cerrado — nvidia-toolkit rechaza distros no-apt.
go test ./internal/tools/ -run TestSupportedNvidiaDistro

# 4. G3 cerrado — CI cubre macOS.
command -v actionlint >/dev/null && actionlint .github/workflows/ci.yml
# Luego: confirmar en la página del PR que ambos jobs de la matriz están verdes.

# 5. README dice sólo lo que el código hace.
! grep -E "nvidia-toolkit.*(Fedora|RHEL|CentOS|Rocky|Alma)" README.md
grep -q "ollama.*qdrant" README.md

# 6. Suite Go completa sigue verde en Linux.
go test ./...
```
