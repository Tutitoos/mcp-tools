package orchestrator

import (
	"strings"
	"testing"
)

// TestBootstrapEnv verifies the G1 fix's core contract: BootstrapEnv (now
// the sole prereq path for Install/InstallSingle/Configure) never probes
// for Docker, so a verb that never touches a DeployDocker tool (qdrant,
// ollama) succeeds on a host without Docker — including Configure's
// "no changes" no-op, which used to fail via the old Bootstrap→EnsureDocker
// call even though it performs no real work. Docker-deployed tools still
// get a clear error via their own Install/Upgrade/Uninstall closures
// (docker.EnsureAvailable in internal/tools/{qdrant,ollama}.go).
//
// Run with dry=true so RunEnv never touches the filesystem (see RunEnv's
// `if dry { ... continue/return without writing }` branches) — this test
// only inspects the logged lines.
func TestBootstrapEnv(t *testing.T) {
	var lines []string
	log := func(l string) { lines = append(lines, l) }

	if err := BootstrapEnv(true, log); err != nil {
		t.Fatalf("BootstrapEnv: %v", err)
	}

	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "docker") {
		t.Errorf("BootstrapEnv must never probe for Docker, got log:\n%s", joined)
	}
	if !strings.Contains(joined, "── env") {
		t.Errorf("expected RunEnv's \"── env\" marker in log, got:\n%s", joined)
	}
}

// TestBootstrapEnvWithoutHomeEnv is a regression guard for the
// mcp-tools-web systemd unit bug: in system mode (no explicit User=),
// systemd does NOT populate $HOME by default, so RunEnv's home resolution
// must not depend solely on os.UserHomeDir() (which fails with "$HOME is
// not defined" in that case) even though the process has a perfectly
// valid home directory (e.g. root's /root). See config.HomeDir.
func TestBootstrapEnvWithoutHomeEnv(t *testing.T) {
	t.Setenv("HOME", "")

	var lines []string
	log := func(l string) { lines = append(lines, l) }

	if err := BootstrapEnv(true, log); err != nil {
		t.Fatalf("BootstrapEnv with $HOME unset: %v", err)
	}
}
