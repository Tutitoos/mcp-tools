// Package orchestrator is the single source of truth for "what does
// install/configure/upgrade do". It exposes pure entry points (no TUI, no
// stdin/stdout) that take a context, an in-memory state, a dry-run flag,
// and a log callback, and return either the new state or an error.
//
// The CLI's `install`/`configure` verbs (and any future entry point —
// including the web admin panel) call into this package. Nothing in this
// package imports internal/cli to avoid an import cycle.
package orchestrator

import (
	"context"
	"fmt"

	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
)

// LogFn is the incremental output sink. Callers wire it to stdout, an SSE
// channel, a log file, or anything else. Each line should NOT include a
// trailing newline; the sink adds it.
type LogFn func(line string)

// Install runs the install workflow against the given (in-memory) selection.
// It is the entry point used by both the CLI `mcp-tools install` (after the
// port prompt + systemd unit) and the web panel's "Apply" action.
//
// Behaviour matches the legacy `internal/cli.runInstall`:
//  1. partition the selection into sudo / tui / interactive slices;
//  2. run inline (sudo, interactive) and Bubbletea (silent) in order;
//  3. register MCPs in the configured clients;
//  4. persist the new state BEFORE skills/rules run (H31: a failure here
//     must not leave the state desynced from the installation);
//  5. best-effort run skills + rules; non-fatal errors are returned alongside
//     the new state.
func Install(ctx context.Context, st state.State, dry bool, log LogFn) (state.State, error) {
	if err := ctx.Err(); err != nil {
		return st, err
	}
	if log == nil {
		log = func(string) {}
	}

	// 1. Bootstrap (docker + env) before any state change.
	if err := Bootstrap(dry, log); err != nil {
		return st, err
	}

	// 2. Persist the selection EARLY so a crash mid-install doesn't re-open
	// the multi-select the next time `install` runs.
	if !dry {
		early := state.State{Selected: st.Selected, Versions: st.Versions}
		if err := early.Save(); err != nil {
			return st, fmt.Errorf("save state (early): %w", err)
		}
	}

	// 3. Partition + run per-tool actions.
	if err := runAll(ctx, "install", st.Selected, dry, log); err != nil {
		return st, err
	}

	// 4. MCP registration with the fresh selection (state still in memory).
	if err := RunMcpConfig(dry, state.State{Selected: st.Selected}, writerFromLog(log)); err != nil {
		return st, fmt.Errorf("mcp-config: %w", err)
	}

	// 5. Build the new state BEFORE skills/rules; persist it. The new
	// versions are queried from each tool's Status().
	stNew := state.State{Selected: st.Selected, Versions: collectVersions(st.Selected)}
	if !dry {
		if err := stNew.Save(); err != nil {
			return stNew, fmt.Errorf("save state: %w", err)
		}
	}

	// 6. Skills + rules (idempotent). Errors are reported but do not block.
	if err := RunSkills(dry, writerFromLog(log)); err != nil {
		log("SKIP skills: " + err.Error())
	}
	if err := RunRules(dry, writerFromLog(log)); err != nil {
		log("SKIP rules: " + err.Error())
	}

	if dry {
		log("SKIP (dry) — no se toca state.json")
	} else {
		log(fmt.Sprintf("── install completo — %d tools · reinicia tu cliente MCP", len(st.Selected)))
	}
	return stNew, nil
}

// Configure is the diff-based add/remove workflow used by both the CLI
// `mcp-tools configure` and the web panel's "Configurar" tab. `next` is the
// target selection (post-edit); the diff against `prev.Selected` decides
// what gets installed/uninstalled.
func Configure(ctx context.Context, prev state.State, next []string, dry bool, log LogFn) (state.State, error) {
	if err := ctx.Err(); err != nil {
		return prev, err
	}
	if log == nil {
		log = func(string) {}
	}

	if err := Bootstrap(dry, log); err != nil {
		return prev, err
	}

	newSelected := append([]string(nil), next...)
	for _, k := range newSelected {
		if _, err := tools.Get(k); err != nil {
			return prev, fmt.Errorf("%q no está en el registry", k)
		}
	}

	toAdd, toRemove := diffKeys(newSelected, prev.Selected), diffKeys(prev.Selected, newSelected)
	if len(toAdd) == 0 && len(toRemove) == 0 {
		log("── sin cambios")
		return prev, nil
	}

	toRemoveSorted, err := TopoSort(toRemove)
	if err != nil {
		return prev, err
	}
	// Reverse-deps: dependents first.
	for i, j := 0, len(toRemoveSorted)-1; i < j; i, j = i+1, j-1 {
		toRemoveSorted[i], toRemoveSorted[j] = toRemoveSorted[j], toRemoveSorted[i]
	}

	toAddSorted, err := TopoSort(toAdd)
	if err != nil {
		return prev, err
	}

	// Uninstall (reverse-dep aware): sudo → tui → inter.
	if err := runAll(ctx, "uninstall", toRemoveSorted, dry, log); err != nil {
		return prev, err
	}
	// Install: sudo → tui → inter.
	if err := runAll(ctx, "install", toAddSorted, dry, log); err != nil {
		return prev, err
	}

	stNew := state.State{Selected: newSelected, Versions: prev.Versions}
	if err := RunMcpConfig(dry, stNew, writerFromLog(log)); err != nil {
		return prev, fmt.Errorf("mcp-config: %w", err)
	}

	// Skills + rules; non-fatal.
	if err := RunSkills(dry, writerFromLog(log)); err != nil {
		log("SKIP skills: " + err.Error())
	}
	if err := RunRules(dry, writerFromLog(log)); err != nil {
		log("SKIP rules: " + err.Error())
	}

	if dry {
		log("SKIP (dry) — no se toca state.json")
		return prev, nil
	}

	stNew.Versions = collectVersions(newSelected)
	if err := stNew.Save(); err != nil {
		return stNew, fmt.Errorf("save state: %w", err)
	}
	unchanged := len(newSelected) - len(toAdd)
	log(fmt.Sprintf("── configure completo — +%d añadidos, -%d eliminados, =%d sin cambios", len(toAdd), len(toRemove), unchanged))
	return stNew, nil
}

// Upgrade calls each tool's Upgrade closure in the same sudo/tui/inter
// partition Install uses. The CLI's `mcp-tools update --tools` (now removed
// from the CLI but kept for the orchestrator + web panel) and the per-row
// "upgrade" button in /api/tools/{key}/upgrade reach this.
func Upgrade(ctx context.Context, keys []string, dry bool, log LogFn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if log == nil {
		log = func(string) {}
	}
	return runAll(ctx, "upgrade", keys, dry, log)
}

// Uninstall removes one tool. `force=true` skips the reverse-dep check
// (mirrors `mcp-tools uninstall --force`). The CLI's per-tool uninstall
// and the web panel's row button reach this.
func Uninstall(ctx context.Context, key string, force bool, dry bool, log LogFn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if log == nil {
		log = func(string) {}
	}
	st, err := state.Load()
	if err != nil {
		return err
	}
	if !st.Has(key) {
		return fmt.Errorf("%q no está en el state; nada que desinstalar", key)
	}
	blocker := findBlocker(key, st.Selected)
	if blocker != "" && !force {
		return fmt.Errorf("no se puede desinstalar %s: %s lo requiere. Usa --force para saltarlo", key, blocker)
	}
	if err := runAll(ctx, "uninstall", []string{key}, dry, log); err != nil {
		return err
	}
	if dry {
		return nil
	}
	st.Selected = append([]string{}, st.Selected...)
	newSelected := st.Selected[:0]
	for _, k := range st.Selected {
		if k != key {
			newSelected = append(newSelected, k)
		}
	}
	st.Selected = newSelected
	delete(st.Versions, key)
	if err := st.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	log(fmt.Sprintf("OK %s desinstalado", key))
	if blocker != "" && force {
		log(fmt.Sprintf("WARN %s sigue seleccionado pero %s ha desaparecido — %s puede quedar roto", blocker, key, blocker))
	}
	return nil
}

// InstallSingle runs only one tool's Install closure. Used by the web
// panel's per-row install button when the user wants to add a single
// component without re-running the full selection diff.
func InstallSingle(ctx context.Context, key string, log LogFn) (state.State, error) {
	if err := ctx.Err(); err != nil {
		return state.State{}, err
	}
	if log == nil {
		log = func(string) {}
	}
	if err := Bootstrap(false, log); err != nil {
		return state.State{}, err
	}
	if err := runAll(ctx, "install", []string{key}, false, log); err != nil {
		return state.State{}, err
	}
	st, err := state.Load()
	if err != nil {
		return state.State{}, err
	}
	if !st.Has(key) {
		next := append([]string{}, st.Selected...)
		next = append(next, key)
		st.Selected = next
		if t, terr := tools.Get(key); terr == nil && t.Status != nil {
			if p, perr := t.Status(); perr == nil && p.Version != "" {
				if st.Versions == nil {
					st.Versions = map[string]string{}
				}
				st.Versions[key] = p.Version
			}
		}
		if err := st.Save(); err != nil {
			return st, fmt.Errorf("save state: %w", err)
		}
	}
	return st, nil
}

// UpgradeSingle runs only one tool's Upgrade closure. Used by the per-row
// upgrade button.
func UpgradeSingle(ctx context.Context, key string, log LogFn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if log == nil {
		log = func(string) {}
	}
	return runAll(ctx, "upgrade", []string{key}, false, log)
}

// UpdateSelf runs the git-based self-update via the CLI helper (the only
// way to do it without a Go-only implementation that duplicates the
// `git fetch / git pull / make install` orchestration).
//
// The legacy CLI exposes this via `mcp-tools update --self`; the web
// panel's "Settings → Update" action reuses it.
func UpdateSelf(ctx context.Context, dry bool, log LogFn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if log == nil {
		log = func(string) {}
	}
	return RunSelfUpdate(dry, log)
}

// RefreshMcpConfig re-runs the MCP registration against the current state.
// Mirrors `mcp-tools mcp-config` (now removed from the CLI but reachable
// via /api/mcp-config/sync).
func RefreshMcpConfig(ctx context.Context, dry bool, log LogFn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	st, err := state.Load()
	if err != nil {
		return err
	}
	return RunMcpConfig(dry, st, writerFromLog(log))
}

// InstallSkills re-creates the symlinks under the supported clients.
// Mirrors `mcp-tools skills` (now removed from the CLI but reachable via
// /api/skills/sync).
func InstallSkills(ctx context.Context, dry bool, log LogFn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return RunSkills(dry, writerFromLog(log))
}

// InstallRules re-installs the RULES.md blocks in the supported clients.
// Mirrors `mcp-tools rules` (now removed from the CLI but reachable via
// /api/rules/sync).
func InstallRules(ctx context.Context, dry bool, log LogFn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return RunRules(dry, writerFromLog(log))
}

// RegenerateEnv (re)writes .env / .env.mem0. `force=true` overwrites an
// existing file (matches `mcp-tools env --force`). The web panel's
// /api/env and /api/env-mem0 endpoints accept arbitrary key updates via
// `config.UpdateEnv` directly; this helper is the full-rewrite variant.
func RegenerateEnv(ctx context.Context, force bool, dry bool, log LogFn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return RunEnv(dry, force, writerFromLog(log))
}

// runAll dispatches a verb (install/upgrade/uninstall) across the keys,
// respecting the sudo / tui / interactive partition from internal/cli.
func runAll(ctx context.Context, verb string, keys []string, dry bool, log LogFn) error {
	sudoKeys, tuiKeys, interKeys := PartitionByStdio(keys)
	if err := runInlineTools(verb, sudoKeys, dry, log); err != nil {
		return err
	}
	if err := runToolSteps(ctx, verb, tuiKeys, dry, log); err != nil {
		return err
	}
	if err := runInlineTools(verb, interKeys, dry, log); err != nil {
		return err
	}
	return nil
}
