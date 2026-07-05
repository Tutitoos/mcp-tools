package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// hostHome returns $HOME or an error if empty (defensive against systemd units
// with an empty environment).
func hostHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("HOME vacío — establece $HOME antes de correr install")
	}
	return home, nil
}

func cargoBin(home string) string { return filepath.Join(home, ".cargo", "bin", "cargo") }
func rtkBin(home string) string   { return filepath.Join(home, ".cargo", "bin", "rtk") }
func uvBin(home string) string    { return filepath.Join(home, ".local", "bin", "uv") }

// withCargoPath appends ~/.cargo/bin to PATH so a freshly-installed cargo is
// resolvable within the same process.
func withCargoPath(env []string, home string) []string {
	dir := filepath.Join(home, ".cargo", "bin")
	return prependPath(env, dir)
}

// withLocalBinPath prepends ~/.local/bin to PATH so freshly-installed uv +
// tool launchers are resolvable within the same process.
func withLocalBinPath(env []string, home string) []string {
	return prependPath(env, filepath.Join(home, ".local", "bin"))
}

func prependPath(env []string, dir string) []string {
	for i, kv := range env {
		if !strings.HasPrefix(kv, "PATH=") {
			continue
		}
		if strings.Contains(kv, dir) {
			return env
		}
		env[i] = "PATH=" + dir + string(os.PathListSeparator) + strings.TrimPrefix(kv, "PATH=")
		return env
	}
	return append(env, "PATH="+dir)
}

// ensureCargo installs rustup+cargo unattended if cargo is not reachable.
func ensureCargo(dry bool, log func(string), home string) error {
	if _, err := exec.LookPath("cargo"); err == nil {
		return nil
	}
	if _, err := os.Stat(cargoBin(home)); err == nil {
		return nil
	}
	if dry {
		log("$ curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable --profile minimal --no-modify-path")
		return nil
	}
	if _, err := exec.LookPath("curl"); err != nil {
		return errors.New("cargo/rustup ausente y curl no está en PATH; instala rustup manualmente")
	}
	script := exec.Command("curl", "--proto", "=https", "--tlsv1.2", "-sSf", "https://sh.rustup.rs")
	shell := exec.Command("sh", "-s", "--", "-y", "--default-toolchain", "stable", "--profile", "minimal", "--no-modify-path")
	shell.Env = append(os.Environ(),
		"HOME="+home,
		"CARGO_HOME="+filepath.Join(home, ".cargo"),
		"RUSTUP_HOME="+filepath.Join(home, ".rustup"),
	)
	pipe, err := script.StdoutPipe()
	if err != nil {
		return err
	}
	shell.Stdin = pipe
	var out bytes.Buffer
	shell.Stdout = &out
	shell.Stderr = &out
	if err := shell.Start(); err != nil {
		return fmt.Errorf("rustup shell start: %w", err)
	}
	if err := script.Run(); err != nil {
		return fmt.Errorf("rustup curl: %w", err)
	}
	if err := shell.Wait(); err != nil {
		return fmt.Errorf("rustup install: %w\n%s", err, strings.TrimSpace(out.String()))
	}
	log("OK rustup+cargo instalado en ~/.cargo/bin")
	return nil
}

// ensureUV installs uv unattended into ~/.local/bin if not reachable.
func ensureUV(dry bool, log func(string), home string) error {
	if _, err := exec.LookPath("uv"); err == nil {
		return nil
	}
	if _, err := os.Stat(uvBin(home)); err == nil {
		return nil
	}
	if dry {
		log("$ curl -LsSf https://astral.sh/uv/install.sh | sh")
		return nil
	}
	if _, err := exec.LookPath("curl"); err != nil {
		return errors.New("uv ausente y curl no está en PATH; instala uv manualmente")
	}
	script := exec.Command("curl", "-LsSf", "https://astral.sh/uv/install.sh")
	shell := exec.Command("sh")
	shell.Env = append(os.Environ(),
		"HOME="+home,
		"UV_INSTALL_DIR="+filepath.Join(home, ".local", "bin"),
	)
	pipe, err := script.StdoutPipe()
	if err != nil {
		return err
	}
	shell.Stdin = pipe
	var out bytes.Buffer
	shell.Stdout = &out
	shell.Stderr = &out
	if err := shell.Start(); err != nil {
		return fmt.Errorf("uv shell start: %w", err)
	}
	if err := script.Run(); err != nil {
		return fmt.Errorf("uv curl: %w", err)
	}
	if err := shell.Wait(); err != nil {
		return fmt.Errorf("uv install: %w\n%s", err, strings.TrimSpace(out.String()))
	}
	log("OK uv instalado en ~/.local/bin")
	return nil
}

// hasKeyIn walks a JSON tree at `path` and returns true iff every key is
// present (values under maps). Missing file → false, no error.
func hasKeyIn(path string, keys []string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return false
	}
	cur := root
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		v, ok := m[k]
		if !ok {
			return false
		}
		cur = v
	}
	return true
}

// containsRTKClaudeHook walks the Claude settings.json shape looking for a
// PreToolUse hook whose command contains "rtk hook claude".
func containsRTKClaudeHook(settings map[string]any) bool {
	hooks, _ := settings["hooks"].(map[string]any)
	pre, _ := hooks["PreToolUse"].([]any)
	for _, entry := range pre {
		e, _ := entry.(map[string]any)
		inner, _ := e["hooks"].([]any)
		for _, h := range inner {
			hm, _ := h.(map[string]any)
			cmd, _ := hm["command"].(string)
			if strings.Contains(cmd, "rtk hook claude") {
				return true
			}
		}
	}
	return false
}

// runCombined runs cmd with combined stdout+stderr and wraps error messages
// with the trailing output for easier diagnosis.
func runCombined(cmd *exec.Cmd, tag string) error {
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w\n%s", tag, err, strings.TrimSpace(out.String()))
	}
	return nil
}

// firstLine returns the first non-empty line of s, trimmed.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if trim != "" {
			return trim
		}
	}
	return ""
}

// versionOf tries `bin --version` and returns the first line trimmed.
func versionOf(bin string, args ...string) string {
	if bin == "" {
		return ""
	}
	if len(args) == 0 {
		args = []string{"--version"}
	}
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		return ""
	}
	return firstLine(string(out))
}

// which resolves a binary in PATH; returns "" if missing.
func which(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}
