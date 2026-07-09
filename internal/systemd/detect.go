package systemd

import (
	"fmt"
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

// DetectMode probes `systemctl --user status`. If the user bus is
// reachable, ModeUser is returned. Otherwise it falls back to probing
// `systemctl is-system-running`. If neither works, ModeNone.
//
// `override` (when non-empty) wins, regardless of probe results. Used by
// the CLI's --user / --system flags.
func DetectMode(override Mode) (Mode, error) {
	if override == ModeUser || override == ModeSystem {
		return override, nil
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return ModeNone, nil
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
