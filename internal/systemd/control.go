package systemd

import (
	"fmt"
	"os/exec"
	"strings"
)

// Enable enables + starts the unit (idempotent — re-running `enable`
// on an already-enabled unit is a no-op).
func Enable(mode Mode) error {
	prefix := SystemctlPrefix(mode)
	if err := run(prefix, "enable", "--now", "mcp-tools-web.service"); err != nil {
		return err
	}
	return nil
}

// Disable stops + disables the unit. Equivalent to
// `systemctl disable --now mcp-tools-web.service`.
func Disable(mode Mode) error {
	prefix := SystemctlPrefix(mode)
	if err := run(prefix, "disable", "--now", "mcp-tools-web.service"); err != nil {
		return err
	}
	return nil
}

// IsActive returns true when the unit is currently `active` (or
// `activating` — systemd is still spinning it up).
func IsActive(mode Mode) bool {
	prefix := SystemctlPrefix(mode)
	out, err := exec.Command("systemctl", append(prefix, "is-active", "mcp-tools-web.service")...).CombinedOutput()
	if err != nil {
		return false
	}
	state := strings.TrimSpace(string(out))
	return state == "active" || state == "activating"
}

// IsEnabled returns true when the unit is enabled (will start on boot).
func IsEnabled(mode Mode) bool {
	prefix := SystemctlPrefix(mode)
	out, err := exec.Command("systemctl", append(prefix, "is-enabled", "mcp-tools-web.service")...).CombinedOutput()
	if err != nil {
		return false
	}
	state := strings.TrimSpace(string(out))
	return state == "enabled" || state == "enabled-runtime" || state == "static"
}

// ActiveState returns the raw `is-active` output (active / inactive /
// failed / activating / unknown). Useful for `mcp-tools web --status`.
func ActiveState(mode Mode) string {
	prefix := SystemctlPrefix(mode)
	out, _ := exec.Command("systemctl", append(prefix, "is-active", "mcp-tools-web.service")...).CombinedOutput()
	return strings.TrimSpace(string(out))
}

// EnabledState returns the raw `is-enabled` output.
func EnabledState(mode Mode) string {
	prefix := SystemctlPrefix(mode)
	out, _ := exec.Command("systemctl", append(prefix, "is-enabled", "mcp-tools-web.service")...).CombinedOutput()
	return strings.TrimSpace(string(out))
}

// JournalTail returns the last `n` journal lines for the unit.
func JournalTail(mode Mode, n int) (string, error) {
	prefix := SystemctlPrefix(mode)
	args := append(prefix, "log", "--no-pager", "-n", fmt.Sprintf("%d", n), "-u", "mcp-tools-web.service")
	out, err := exec.Command("journalctl", args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// CurrentPort parses the existing unit file (if any) and returns the
// `--port` value baked into ExecStart. Returns 0 when the unit file is
// missing or unparseable.
func CurrentPort(mode Mode) int {
	unitPath, err := UnitPath(mode)
	if err != nil {
		return 0
	}
	data, err := readFile(unitPath)
	if err != nil {
		return 0
	}
	return parsePortFromUnit(string(data))
}

// CurrentBind returns the `--bind` value from the existing unit, or
// "0.0.0.0" (all interfaces — the project default) when the unit file
// is missing or unparseable.
func CurrentBind(mode Mode) string {
	unitPath, err := UnitPath(mode)
	if err != nil {
		return "0.0.0.0"
	}
	data, err := readFile(unitPath)
	if err != nil {
		return "0.0.0.0"
	}
	return parseBindFromUnit(string(data))
}
