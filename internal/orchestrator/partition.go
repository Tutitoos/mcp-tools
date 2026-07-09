package orchestrator

import (
	"context"
	"fmt"

	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
)

// PartitionByStdio splits keys into three ordered slices in the intended
// execution order:
//
//   - sudo:  Deploy == DeploySudo — run FIRST, inline, so the sudo password
//     prompt appears immediately and the sudo timestamp stays fresh through
//     the tool's remaining sudo calls.
//   - tui:   neither sudo nor Interactive — wrapped by the Bubbletea
//     progress runner in the middle, giving the user fast per-step feedback.
//   - inter: Interactive == true AND Deploy != DeploySudo — run LAST, inline,
//     AFTER Bubbletea closes, so the user only attends upstream prompts once
//     every silent tool has finished. Can walk away in the middle.
//
// Unknown key (tools.Get error) → tui bucket (safe default: no stdio inherit).
// A tool that is both DeploySudo AND Interactive → sudo bucket (sudo trumps;
// keeps the password prompt upfront and lets the tool's own TUI run right
// after in the same call). No such tool exists today; rule is proactive.
// Order within each slice preserves the caller's input order.
func PartitionByStdio(keys []string) (sudo, tui, inter []string) {
	for _, k := range keys {
		t, err := tools.Get(k)
		if err != nil {
			tui = append(tui, k)
			continue
		}
		switch {
		case t.Deploy == tools.DeploySudo:
			sudo = append(sudo, k)
		case t.Interactive:
			inter = append(inter, k)
		default:
			tui = append(tui, t.Key)
		}
	}
	return sudo, tui, inter
}

// runInlineTools runs each tool closure with inherited stdio so upstream
// prompts (sudo password, interactive installers) are visible and usable.
// Runs OUTSIDE any Bubbletea TUI.
func runInlineTools(verb string, keys []string, dry bool, log LogFn) error {
	if len(keys) == 0 {
		return nil
	}
	for _, k := range keys {
		t, err := tools.Get(k)
		if err != nil {
			return err
		}
		fn := pickToolFn(t, verb)
		if fn == nil {
			return fmt.Errorf("%s: verb %q no expuesto", k, verb)
		}
		hint := "interactivo — puede requerir input"
		if t.Deploy == tools.DeploySudo {
			hint = "sudo — puede pedir contraseña"
		}
		log(fmt.Sprintf("── %s %s (%s)", verb, t.Label, hint))
		if err := fn(dry, log); err != nil {
			return fmt.Errorf("%s %s: %w", verb, k, err)
		}
	}
	return nil
}

// runToolSteps wraps each tool closure in a Bubbletea progress runner.
// This is the orchestrator's ONLY remaining TUI dependency. The web panel
// doesn't use this path (it streams log lines via the logbus + SSE); the
// CLI install/configure verbs do, when a TTY is attached. When the
// orchestrator is invoked from a non-TTY context (CI, web panel), the
// caller should pass an empty keys slice for the tui bucket — or fall
// back to runInlineTools-style plain execution.
//
// In the current refactor, the legacy Bubbletea-based step runner is
// re-exported via a callback set by the CLI adapter (see SetStepRunner).
// When no runner is registered, runToolSteps degrades to inline
// execution so the orchestrator stays usable from non-TTY callers.
func runToolSteps(_ context.Context, verb string, keys []string, dry bool, log LogFn) error {
	if len(keys) == 0 {
		return nil
	}
	if stepRunner == nil {
		return runInlineTools(verb, keys, dry, log)
	}
	return stepRunner(verb, keys, dry, log)
}

// StepRunner is the legacy Bubbletea-wrapped per-tool execution. The CLI
// adapter registers a runner in its init() so the orchestrator can stay
// TUI-free. See internal/cli/install.go for the implementation.
type StepRunner func(verb string, keys []string, dry bool, log LogFn) error

var stepRunner StepRunner

// SetStepRunner registers the Bubbletea-backed step runner. Idempotent;
// replacing a runner mid-flight is undefined.
func SetStepRunner(fn StepRunner) {
	stepRunner = fn
}

func pickToolFn(t tools.Tool, verb string) func(bool, func(string)) error {
	switch verb {
	case "install":
		return t.Install
	case "upgrade":
		return t.Upgrade
	case "uninstall":
		return t.Uninstall
	}
	return nil
}

// diffKeys returns items in `a` that are not in `b`, preserving `a` order.
func diffKeys(a, b []string) []string {
	inB := make(map[string]bool, len(b))
	for _, k := range b {
		inB[k] = true
	}
	var out []string
	for _, k := range a {
		if !inB[k] {
			out = append(out, k)
		}
	}
	return out
}

// TopoSort is a thin wrapper over tools.TopoSort so callers don't need
// to import internal/tools directly. It exists to make the orchestrator
// surface self-contained.
func TopoSort(keys []string) ([]string, error) {
	return tools.TopoSort(keys)
}

func collectVersions(keys []string) map[string]string {
	out := map[string]string{}
	for _, key := range keys {
		t, err := tools.Get(key)
		if err != nil || t.Status == nil {
			continue
		}
		s, err := t.Status()
		if err != nil {
			continue
		}
		if s.Version != "" {
			out[key] = s.Version
		}
	}
	return out
}

// findBlocker walks the selected tools and returns the first one whose
// Deps contains `key`. Used by Uninstall to enforce the reverse-dep rule
// outside the CLI.
func findBlocker(key string, selected []string) string {
	for _, other := range selected {
		if other == key {
			continue
		}
		ot, err := tools.Get(other)
		if err != nil {
			continue
		}
		for _, dep := range ot.Deps {
			if dep == key {
				return other
			}
		}
	}
	return ""
}

// unused import silencer for `state` — kept for clarity in this file's
// neighbours that do use it (orchestrator.go).
var _ = state.SchemaVersion
