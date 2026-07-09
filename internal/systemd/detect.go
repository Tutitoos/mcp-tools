package systemd

import (
	"fmt"
	"os"
	"os/exec"
)

// Mode reports whether systemd is available and, if so, in which flavour:
// user vs system.
type Mode string

const (
	// ModeUser runs the unit in the user's systemd manager (no root
	// required; works on most developer hosts).
	ModeUser Mode = "user"
	// ModeSystem runs the unit under /etc/systemd/system. Requires root.
	ModeSystem Mode = "system"
	// ModeNone prints a "systemd not available" hint and a nohup fallback.
	ModeNone Mode = "none"
)

// unitExists is a swappable indirection so tests can fake filesystem state
// without touching the real disk.
var unitExists = func(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// systemctlAvailable is a swappable indirection (mirrors unitExists) so
// tests can simulate a host with/without systemctl on PATH without
// depending on the actual test-runner environment — CI containers
// commonly run without systemd (and therefore without a systemctl
// binary) entirely, which made the unmocked exec.LookPath("systemctl")
// check short-circuit DetectMode to ModeNone before the unitExists fakes
// below it ever ran, regardless of what they returned.
var systemctlAvailable = func() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

// DetectMode first checks whether a unit file is ALREADY installed on
// disk, preferring ModeSystem then ModeUser — an existing install is
// authoritative over "which systemd manager happens to be reachable
// right now". Without this, a host where the invoking user has BOTH a
// working `systemctl --user` session (common over SSH via PAM, even for
// root) AND a system-mode unit already installed would silently prefer
// ModeUser, find no user-mode unit file, and treat the real running
// daemon as "not installed" — e.g. `mcp-tools web --restart` (run by
// `make install`) silently no-ops, leaving the OLD binary running.
//
// If neither unit file exists yet (fresh host, first `mcp-tools
// install`), falls back to probing `systemctl --user status` then
// `systemctl is-system-running`, preferring user mode (no root
// required) as the friendlier default.
//
// `override` (when non-empty) wins, regardless of probe results. Used by
// the CLI's --user / --system flags.
func DetectMode(override Mode) (Mode, error) {
	if override == ModeUser || override == ModeSystem {
		return override, nil
	}
	if !systemctlAvailable() {
		return ModeNone, nil
	}
	if systemPath, err := UnitPath(ModeSystem); err == nil && unitExists(systemPath) {
		return ModeSystem, nil
	}
	if userPath, err := UnitPath(ModeUser); err == nil && unitExists(userPath) {
		return ModeUser, nil
	}
	if err := exec.Command("systemctl", "--user", "status").Run(); err == nil {
		return ModeUser, nil
	}
	if err := exec.Command("systemctl", "is-system-running").Run(); err == nil {
		return ModeSystem, nil
	}
	return ModeNone, nil
}

// UnitPath returns the absolute path of the unit file for a given mode.
// User: ~/.config/systemd/user/mcp-tools-web.service
// System: /etc/systemd/system/mcp-tools-web.service
func UnitPath(mode Mode) (string, error) {
	switch mode {
	case ModeUser:
		home, err := userHomeDir()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s/.config/systemd/user/mcp-tools-web.service", home), nil
	case ModeSystem:
		return "/etc/systemd/system/mcp-tools-web.service", nil
	}
	return "", fmt.Errorf("systemd: no unit path for mode %q", mode)
}

// SystemctlPrefix returns the right `systemctl` invocation prefix for
// the given mode ("--user" for user mode, "" for system/none).
func SystemctlPrefix(mode Mode) []string {
	if mode == ModeUser {
		return []string{"--user"}
	}
	return nil
}
