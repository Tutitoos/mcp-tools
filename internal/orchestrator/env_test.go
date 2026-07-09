package orchestrator

import (
	"strings"
	"testing"
)

// TestBootstrapEnv verifies the G1 fix's core contract: BootstrapEnv (the
// prereq path Install/InstallSingle now use) never probes for Docker, so a
// host-only tool install succeeds on a host without Docker. Bootstrap (still
// used by Configure/Upgrade/Uninstall, which may touch Docker-deployed
// tools) keeps the Docker probe. Run with dry=true so RunEnv never touches
// the filesystem (see RunEnv's `if dry { ... continue/return without
// writing }` branches) — this test only inspects the logged lines.
func TestBootstrapEnv(t *testing.T) {
	tests := []struct {
		name       string
		run        func(log LogFn) error
		wantDocker bool // must the log mention a docker probe?
	}{
		{
			name:       "BootstrapEnv skips the Docker probe",
			run:        func(log LogFn) error { return BootstrapEnv(true, log) },
			wantDocker: false,
		},
		{
			name:       "Bootstrap still probes Docker",
			run:        func(log LogFn) error { return Bootstrap(true, log) },
			wantDocker: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lines []string
			log := func(l string) { lines = append(lines, l) }

			if err := tt.run(log); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}

			joined := strings.Join(lines, "\n")
			hasDocker := strings.Contains(joined, "docker")
			if hasDocker != tt.wantDocker {
				t.Errorf("docker probe present = %v, want %v. Log:\n%s", hasDocker, tt.wantDocker, joined)
			}
			// Both paths must still (re)generate the env files via RunEnv.
			if !strings.Contains(joined, "── env") {
				t.Errorf("expected RunEnv's \"── env\" marker in log, got:\n%s", joined)
			}
		})
	}
}
