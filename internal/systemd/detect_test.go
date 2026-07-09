package systemd

import "testing"

// withUnitExists swaps unitExists for the duration of the test and
// restores the original on cleanup.
func withUnitExists(t *testing.T, fn func(path string) bool) {
	t.Helper()
	orig := unitExists
	unitExists = fn
	t.Cleanup(func() { unitExists = orig })
}

// withSystemctlAvailable swaps systemctlAvailable for the duration of the
// test and restores the original on cleanup.
func withSystemctlAvailable(t *testing.T, available bool) {
	t.Helper()
	orig := systemctlAvailable
	systemctlAvailable = func() bool { return available }
	t.Cleanup(func() { systemctlAvailable = orig })
}

// TestDetectModePrefersSystemWhenSystemUnitExists is the regression guard
// for the bug reproduced live on a host where root has a working
// `systemctl --user` session (common over SSH via PAM) AND a system-mode
// unit already installed: DetectMode used to always prefer ModeUser in
// that case, found no user-mode unit file, and every caller (notably
// `mcp-tools web --restart`, run automatically by `make install`) treated
// the real, running system-mode daemon as "not installed" and silently
// skipped the restart — leaving the OLD binary running in production.
func TestDetectModePrefersSystemWhenSystemUnitExists(t *testing.T) {
	withSystemctlAvailable(t, true)
	systemPath, err := UnitPath(ModeSystem)
	if err != nil {
		t.Fatalf("UnitPath(ModeSystem): %v", err)
	}
	withUnitExists(t, func(path string) bool { return path == systemPath })

	mode, err := DetectMode("")
	if err != nil {
		t.Fatalf("DetectMode: %v", err)
	}
	if mode != ModeSystem {
		t.Fatalf("DetectMode() = %q, want %q (system unit is installed on disk)", mode, ModeSystem)
	}
}

// TestDetectModePrefersSystemOverUserWhenBothExist documents the
// deterministic tiebreak when both unit files somehow exist: ModeSystem
// wins.
func TestDetectModePrefersSystemOverUserWhenBothExist(t *testing.T) {
	withSystemctlAvailable(t, true)
	withUnitExists(t, func(path string) bool { return true })

	mode, err := DetectMode("")
	if err != nil {
		t.Fatalf("DetectMode: %v", err)
	}
	if mode != ModeSystem {
		t.Fatalf("DetectMode() = %q, want %q when both unit files exist", mode, ModeSystem)
	}
}

// TestDetectModeUsesUserWhenOnlyUserUnitExists confirms ModeUser is still
// picked correctly when that's the only unit actually installed (the
// common developer-host case this session's fix must not regress).
func TestDetectModeUsesUserWhenOnlyUserUnitExists(t *testing.T) {
	withSystemctlAvailable(t, true)
	userPath, err := UnitPath(ModeUser)
	if err != nil {
		t.Fatalf("UnitPath(ModeUser): %v", err)
	}
	withUnitExists(t, func(path string) bool { return path == userPath })

	mode, err := DetectMode("")
	if err != nil {
		t.Fatalf("DetectMode: %v", err)
	}
	if mode != ModeUser {
		t.Fatalf("DetectMode() = %q, want %q (only the user unit is installed on disk)", mode, ModeUser)
	}
}

// TestDetectModeOverrideWinsRegardlessOfDisk confirms an explicit
// --user/--system override always short-circuits, even if the on-disk
// unit state would suggest otherwise.
func TestDetectModeOverrideWinsRegardlessOfDisk(t *testing.T) {
	withUnitExists(t, func(path string) bool { return false })

	mode, err := DetectMode(ModeUser)
	if err != nil {
		t.Fatalf("DetectMode: %v", err)
	}
	if mode != ModeUser {
		t.Fatalf("DetectMode(override=ModeUser) = %q, want %q", mode, ModeUser)
	}
}

// TestDetectModeNoneWhenSystemctlUnavailable confirms DetectMode reports
// ModeNone — not a false ModeUser/ModeSystem from stale unit files — on a
// host without systemd at all (e.g. most CI containers), even if unit
// files somehow exist on disk.
func TestDetectModeNoneWhenSystemctlUnavailable(t *testing.T) {
	withSystemctlAvailable(t, false)
	withUnitExists(t, func(path string) bool { return true })

	mode, err := DetectMode("")
	if err != nil {
		t.Fatalf("DetectMode: %v", err)
	}
	if mode != ModeNone {
		t.Fatalf("DetectMode() = %q, want %q when systemctl is unavailable", mode, ModeNone)
	}
}
