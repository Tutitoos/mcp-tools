package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Install writes the unit file, runs `daemon-reload`, then `enable --now`.
// The returned error (if any) chains the underlying command output for
// easier diagnosis.
//
// In ModeSystem, writing /etc/systemd/system requires root. Callers
// should pre-check sudo availability; Install itself does NOT elevate.
func Install(mode Mode, port int, bind, binaryPath, envFile string) error {
	if mode == ModeNone {
		return fmt.Errorf("systemd: no installable mode")
	}
	unitPath, err := UnitPath(mode)
	if err != nil {
		return err
	}
	cfg := UnitConfig{
		BinaryPath: binaryPath,
		Port:       port,
		Bind:       bind,
		EnvFile:    envFile,
		User:       mode == ModeUser,
	}
	rendered, err := RenderUnit(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return fmt.Errorf("systemd: mkdir %s: %w", filepath.Dir(unitPath), err)
	}
	if mode == ModeUser {
		if err := os.WriteFile(unitPath, []byte(rendered), 0o644); err != nil {
			return fmt.Errorf("systemd: write %s: %w", unitPath, err)
		}
	} else {
		// System unit: caller is expected to have sudo. If not, write
		// fails fast and the CLI prints the no-systemd fallback.
		if err := os.WriteFile(unitPath, []byte(rendered), 0o644); err != nil {
			return fmt.Errorf("systemd: write %s (¿sudo?): %w", unitPath, err)
		}
	}
	prefix := SystemctlPrefix(mode)
	if err := run(prefix, "daemon-reload"); err != nil {
		return fmt.Errorf("systemd: daemon-reload: %w", err)
	}
	if err := run(prefix, "enable", "--now", "mcp-tools-web.service"); err != nil {
		return fmt.Errorf("systemd: enable --now: %w", err)
	}
	return nil
}

// Stop / Restart / Status are thin wrappers used by the CLI's `stop`,
// `restart`, and `status-web` subcommands.
func Stop(mode Mode) error { return run(SystemctlPrefix(mode), "stop", "mcp-tools-web.service") }
func Restart(mode Mode) error {
	return run(SystemctlPrefix(mode), "restart", "mcp-tools-web.service")
}

// Status runs `systemctl is-active mcp-tools-web.service` and returns
// the output + exit error. Used by `status-web`.
func Status(mode Mode) (string, error) {
	prefix := SystemctlPrefix(mode)
	out, err := exec.Command("systemctl", append(prefix, "is-active", "mcp-tools-web.service")...).CombinedOutput()
	return string(out), err
}

// run is a small wrapper that runs systemctl with the given args and
// captures stderr for the error message.
func run(prefix []string, args ...string) error {
	full := append([]string{}, prefix...)
	full = append(full, args...)
	cmd := exec.Command("systemctl", full...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %w\n%s", full, err, string(out))
	}
	return nil
}