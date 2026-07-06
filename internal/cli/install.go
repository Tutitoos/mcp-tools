package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
	"github.com/Tutitoos/mcp-tools/internal/tui/installer"
	"github.com/Tutitoos/mcp-tools/internal/tui/toolselect"
)

var (
	installDry         bool
	installNoSelect    bool
	installReconfigure bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Multi-select + instala componentes. Reusa el state existente con --noselect.",
	Long: "Abre un TUI multi-select con los componentes disponibles (nvidia-toolkit solo si hay GPU). " +
		"Guarda la selección en $MCP_TOOLS_DATA/state.json. `configure` reabre el TUI.",
	RunE: runInstall,
}

func init() {
	installCmd.Flags().BoolVar(&installDry, "dry", false, "no ejecuta comandos; solo muestra qué haría")
	installCmd.Flags().BoolVar(&installNoSelect, "noselect", false, "usa el state existente sin abrir TUI; error si falta")
	installCmd.Flags().BoolVar(&installReconfigure, "reconfigure", false, "fuerza el TUI aunque haya state (alias de 'configure')")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	stOld, err := state.Load()
	if err != nil {
		return err
	}

	// 1. Bootstrap (prereq + env) fuera del TUI para no ocultar prompts sudo.
	logf := func(s string) { fmt.Fprintln(os.Stdout, s) }
	if err := runBootstrap(installDry, logf); err != nil {
		return err
	}

	// 2. Decidir selección: state existente, TUI multi-select, o error.
	selected, err := resolveSelection(stOld)
	if err != nil {
		return err
	}

	// 3a. Persiste la selección YA (antes del loop de install/mcp-config/skills)
	// para que un crash a mitad de instalación no re-abra el TUI la próxima vez.
	if !installDry {
		stEarly := state.State{Selected: selected, Versions: stOld.Versions}
		if err := stEarly.Save(); err != nil {
			return fmt.Errorf("save state (early): %w", err)
		}
	}

	// 3. Partición en 3: sudo primero (prompt upfront, timestamp fresco),
	//    silent en el TUI Bubbletea (feedback rápido), interactive al final
	//    (user solo atiende TUIs upstream cuando lo silencioso terminó).
	sudoKeys, tuiKeys, interKeys := partitionByStdio(selected)
	if err := runInlineTools("install", sudoKeys, installDry, logf); err != nil {
		return err
	}
	if err := runToolSteps("install", tuiKeys, installDry); err != nil {
		return err
	}
	if err := runInlineTools("install", interKeys, installDry, logf); err != nil {
		return err
	}

	// 4. Registro MCP con la selección fresca (state aún no persistida).
	if err := RunMcpConfig(installDry, state.State{Selected: selected}, os.Stdout); err != nil {
		return fmt.Errorf("mcp-config: %w", err)
	}

	// 5. Skills + rules (idempotentes; sirven a los 3 clientes).
	var buf bytes.Buffer
	if err := RunSkills(installDry, &buf); err != nil {
		return fmt.Errorf("skills: %w", err)
	}
	if err := RunRules(installDry, &buf); err != nil {
		return fmt.Errorf("rules: %w", err)
	}
	if s := strings.TrimRight(buf.String(), "\n"); s != "" {
		fmt.Fprintln(os.Stdout, s)
	}

	// 6. Persiste state al final con versiones frescas (skip en dry).
	if installDry {
		fmt.Fprintln(os.Stdout, "SKIP (dry) — no toca state.json")
		return nil
	}
	stNew := state.State{Selected: selected, Versions: collectVersions(selected)}
	if err := stNew.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	fmt.Fprintf(os.Stdout, "── install completo — %d tools · reinicia tu cliente MCP\n", len(selected))
	return nil
}

// runToolSteps wraps each tool.<verb> in a Bubbletea progress runner.
func runToolSteps(verb string, keys []string, dry bool) error {
	if len(keys) == 0 {
		return nil
	}
	steps := make([]installer.Step, 0, len(keys))
	for _, k := range keys {
		key := k
		t, err := tools.Get(key)
		if err != nil {
			return err
		}
		fn := pickToolFn(t, verb)
		if fn == nil {
			return fmt.Errorf("%s: verb %q no expuesto", key, verb)
		}
		steps = append(steps, installer.Step{
			Key:   key,
			Label: fmt.Sprintf("%s %s", verb, t.Label),
			Phase: capitalize(verb),
			Run:   fn,
		})
	}
	if !isTerminal(os.Stdout) {
		return runToolStepsPlain(steps, dry)
	}
	model := installer.New(steps, []string{capitalize(verb)}, dry)
	p := tea.NewProgram(model)
	res, err := p.Run()
	if err != nil {
		return err
	}
	m, ok := res.(installer.Model)
	if !ok {
		return errors.New("installer: modelo inesperado")
	}
	if code := m.ExitCode(); code != 0 {
		var msgs []string
		for k, v := range m.Errors() {
			msgs = append(msgs, k+": "+v)
		}
		return fmt.Errorf("%s: %s", verb, strings.Join(msgs, "; "))
	}
	return nil
}

// runToolStepsPlain executes each step sequentially, printing captured lines
// straight to stdout. Used when we don't have a TTY (CI, tests, piping).
func runToolStepsPlain(steps []installer.Step, dry bool) error {
	var errs []error
	for _, s := range steps {
		fmt.Fprintf(os.Stdout, "── %s\n", s.Label)
		log := func(line string) { fmt.Fprintln(os.Stdout, "  "+line) }
		if err := s.Run(dry, log); err != nil {
			fmt.Fprintf(os.Stdout, "  FAIL %v\n", err)
			errs = append(errs, fmt.Errorf("%s: %w", s.Key, err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// isTerminal reports whether f is attached to a real TTY.
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
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

// resolveSelection picks the tool set based on flags + persisted state.
func resolveSelection(st state.State) ([]string, error) {
	switch {
	case installNoSelect && installReconfigure:
		return nil, errors.New("--noselect y --reconfigure son mutuamente excluyentes")
	case installNoSelect:
		if len(st.Selected) == 0 {
			return nil, errors.New("state ausente y --noselect fijado — corre 'mcp-tools install' sin flag para elegir")
		}
		return st.Selected, nil
	case installReconfigure || len(st.Selected) == 0:
		avail := availableTools()
		pre := preChecked(st, avail)
		model, err := runToolSelect(avail, pre)
		if err != nil {
			return nil, err
		}
		if !model.Confirmed() {
			return nil, errors.New("cancelado por el user")
		}
		return model.Selected(), nil
	default:
		return st.Selected, nil
	}
}

// availableTools returns the Registry filtered for the current host: nvidia-toolkit
// only when the host actually has an NVIDIA GPU.
func availableTools() []tools.Tool {
	reg := tools.Registry()
	out := make([]tools.Tool, 0, len(reg))
	for _, t := range reg {
		if t.Key == "nvidia-toolkit" && !t.DefaultOn {
			continue
		}
		out = append(out, t)
	}
	return out
}

// preChecked seeds the multi-select from state + DefaultOn + already-installed.
func preChecked(st state.State, avail []tools.Tool) map[string]bool {
	pre := map[string]bool{}
	if len(st.Selected) > 0 {
		for _, k := range st.Selected {
			pre[k] = true
		}
		return pre
	}
	for _, t := range avail {
		if t.DefaultOn {
			pre[t.Key] = true
			continue
		}
		if t.Status != nil {
			if s, err := t.Status(); err == nil && s.Installed {
				pre[t.Key] = true
			}
		}
	}
	return pre
}

func runToolSelect(available []tools.Tool, pre map[string]bool) (toolselect.Model, error) {
	p := tea.NewProgram(toolselect.New(available, pre))
	res, err := p.Run()
	if err != nil {
		return toolselect.Model{}, err
	}
	m, ok := res.(toolselect.Model)
	if !ok {
		return toolselect.Model{}, errors.New("toolselect: modelo inesperado")
	}
	return m, nil
}

// collectVersions runs each tool's Status() and stores the version string.
func collectVersions(selected []string) map[string]string {
	out := map[string]string{}
	for _, key := range selected {
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

// runBootstrap is the shared prereq + env step every top-level verb needs to
// run before touching state or Docker.
func runBootstrap(dry bool, log func(string)) error {
	if err := ensureDocker(dry, log); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := RunEnv(dry, false, &buf); err != nil {
		return fmt.Errorf("env: %w\n%s", err, strings.TrimSpace(buf.String()))
	}
	if s := strings.TrimSpace(buf.String()); s != "" {
		log(s)
	}
	return nil
}

func ensureDocker(dry bool, log func(string)) error {
	if dry {
		log("$ command -v docker")
		log("$ docker compose version")
		return nil
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return errors.New("docker no está en PATH")
	}
	return exec.Command("docker", "compose", "version").Run()
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// partitionByStdio splits keys into three ordered slices in the intended
// execution order:
//   - sudo:  Deploy == DeploySudo — run FIRST, inline, before the Bubbletea
//     TUI, so the sudo password prompt appears immediately and the sudo
//     timestamp stays fresh through the tool's remaining sudo calls.
//   - tui:   neither sudo nor Interactive — wrapped by the Bubbletea progress
//     runner in the middle, giving the user fast per-step feedback.
//   - inter: Interactive == true AND Deploy != DeploySudo — run LAST, inline,
//     AFTER Bubbletea closes, so the user only attends upstream prompts once
//     every silent tool has finished. Can walk away in the middle.
//
// Unknown key (tools.Get error) → tui bucket (safe default: no stdio inherit).
// A tool that is both DeploySudo AND Interactive → sudo bucket (sudo trumps;
// keeps the password prompt upfront and lets the tool's own TUI run right
// after in the same call). No such tool exists today; rule is proactive.
// Order within each slice preserves the caller's input order.
func partitionByStdio(keys []string) (sudo, tui, inter []string) {
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
			tui = append(tui, k)
		}
	}
	return sudo, tui, inter
}

// runInlineTools runs each tool closure with inherited stdio so upstream
// prompts (sudo password, interactive installers) are visible and usable.
// Runs OUTSIDE any Bubbletea TUI.
func runInlineTools(verb string, keys []string, dry bool, log func(string)) error {
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
