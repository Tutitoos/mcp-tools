package docker

import (
	"path/filepath"
	"testing"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

// TestComposeWithFilesNoVerbDuplication reproduces B2: the web layer used
// to build args as `[..., verb]` and then, for any verb other than
// "restart", append `verb, name` AGAIN — producing `docker compose ... up
// up mem0-qdrant` (invalid) for `up`/`stop`. ComposeWithFiles must place
// the verb exactly once, followed by the service name.
func TestComposeWithFilesNoVerbDuplication(t *testing.T) {
	for _, verb := range []string{"up", "stop", "restart"} {
		cmd := ComposeWithFiles([]string{"dockers/compose.yaml"}, verb, "mem0-qdrant")
		args := cmd.Args[1:] // Args[0] is the resolved "docker" binary path
		want := []string{"compose", "-f", "dockers/compose.yaml", "--env-file", ".env", verb, "mem0-qdrant"}
		if len(args) != len(want) {
			t.Fatalf("verb %q: args = %v, want %v", verb, args, want)
		}
		for i := range want {
			if args[i] != want[i] {
				t.Errorf("verb %q: args[%d] = %q, want %q (full: %v)", verb, i, args[i], want[i], args)
			}
		}
		n := 0
		for _, a := range args {
			if a == verb {
				n++
			}
		}
		if n != 1 {
			t.Errorf("verb %q appears %d times in args %v, want exactly 1", verb, n, args)
		}
	}
}

// TestComposeWithFilesRunsFromRepoRoot reproduces B7: the web layer ran
// compose via a raw exec.Command with a RELATIVE compose file path,
// inheriting the process's cwd. Under systemd (no WorkingDirectory=) that
// cwd is "/", so `dockers/compose.yaml` never resolved. ComposeWithFiles
// must pin cmd.Dir to RepoRoot() regardless of the caller's cwd.
func TestComposeWithFilesRunsFromRepoRoot(t *testing.T) {
	t.Chdir(t.TempDir()) // simulate a caller started from an unrelated (or "/") cwd
	cmd := ComposeWithFiles([]string{"dockers/compose.yaml"}, "up", "mem0-qdrant")
	want := config.RepoRoot()
	if cmd.Dir != want {
		t.Errorf("cmd.Dir = %q, want %q (RepoRoot) — resolving dockers/compose.yaml would fail from the caller's actual cwd", cmd.Dir, want)
	}
	if !filepath.IsAbs(cmd.Dir) {
		t.Errorf("cmd.Dir = %q is not absolute", cmd.Dir)
	}
}
