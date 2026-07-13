// Package docker centralises `docker compose` / `docker exec` invocations.
// Every command runs with cwd = config.RepoRoot(), reads the repo's compose file and .env.
package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

// EnsureAvailable checks the host has `docker` in PATH and that the
// compose plugin is callable. Lives here (not internal/orchestrator) so
// both internal/orchestrator (host-wide Bootstrap for Docker-touching
// verbs) and internal/tools (qdrant/ollama Install/Upgrade/Uninstall
// closures) can call it without an import cycle — both already depend on
// internal/docker, neither may depend on the other.
func EnsureAvailable(dry bool, log func(string)) error {
	if dry {
		log("$ command -v docker")
		log("$ docker compose version")
		return nil
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker no está en PATH")
	}
	return exec.Command("docker", "compose", "version").Run()
}

// Compose builds an exec.Cmd for `docker compose -f dockers/compose.yaml --env-file .env <args...>`.
// stdout/stderr are wired to the caller's terminal unless the caller overrides them.
func Compose(args ...string) *exec.Cmd {
	return ComposeWithFiles([]string{"dockers/compose.yaml"}, args...)
}

// composeArgs builds the `compose -f <files...> --env-file .env <args...>`
// argument list shared by ComposeWithFiles and ComposeCmdContext.
func composeArgs(files []string, args []string) []string {
	full := make([]string, 0, len(args)+2*len(files)+3)
	full = append(full, "compose")
	for _, f := range files {
		full = append(full, "-f", f)
	}
	full = append(full, "--env-file", ".env")
	full = append(full, args...)
	return full
}

// ComposeWithFiles is like Compose but lets the caller specify the compose
// files (relative to RepoRoot). Used by callers that need overlays (e.g.
// dockers/ollama-gpu-overlay.yml).
func ComposeWithFiles(files []string, args ...string) *exec.Cmd {
	cmd := exec.Command("docker", composeArgs(files, args)...)
	cmd.Dir = config.RepoRoot()
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// ComposeCmdContext is like ComposeWithFiles but binds the command to ctx —
// canceling ctx kills the child process (exec.CommandContext's default
// Cancel is Process.Kill). Unlike ComposeWithFiles, stdout/stderr are left
// nil for the caller to attach (e.g. via StdoutPipe/StderrPipe for
// streaming) instead of being wired to the terminal.
func ComposeCmdContext(ctx context.Context, files []string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "docker", composeArgs(files, args)...)
	cmd.Dir = config.RepoRoot()
	cmd.Env = os.Environ()
	return cmd
}

// Run executes `docker compose ... <args>` and returns any error.
func Run(args ...string) error { return Compose(args...).Run() }

// RunWithFiles runs `docker compose -f <files...> --env-file .env <args>`.
func RunWithFiles(files []string, args ...string) error {
	return ComposeWithFiles(files, args...).Run()
}

// Output captures stdout of `docker compose ... <args>`, bounded by a 10s
// deadline. Its only caller is status polling (listComposeServices): without
// the deadline a hung Docker daemon pins one goroutine + child process per
// poll, indefinitely.
func Output(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return ComposeCmdContext(ctx, []string{"dockers/compose.yaml"}, args...).Output()
}

// Exec builds `docker exec <container> <cmd...>`. stdio wired to terminal.
func Exec(container string, cmdAndArgs ...string) *exec.Cmd {
	args := append([]string{"exec", container}, cmdAndArgs...)
	cmd := exec.Command("docker", args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// RunCmdWithTimeout builds `docker <args...>` (host-level, no compose / exec)
// wrapped in a context with the given timeout. Use this from status()
// functions so a hung daemon doesn't hang the whole CLI.
func RunCmdWithTimeout(d time.Duration, args ...string) *exec.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = os.Environ()
	// Note: we intentionally do NOT call cancel here. ctx will be released
	// when cmd finishes (or the deadline is reached) — that's sufficient for
	// short-lived status calls, and avoids the goroutine leak path of a
	// manually-managed cancel.
	_ = cancel
	return cmd
}
