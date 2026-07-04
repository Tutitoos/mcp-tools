// Package docker centralises `docker compose` / `docker exec` invocations.
// Every command runs with cwd = config.RepoRoot(), reads the repo's compose file and .env.
package docker

import (
	"os"
	"os/exec"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

// Compose builds an exec.Cmd for `docker compose -f dockers/compose.yaml --env-file .env <args...>`.
// stdout/stderr are wired to the caller's terminal unless the caller overrides them.
func Compose(args ...string) *exec.Cmd {
	full := append([]string{
		"compose",
		"-f", "dockers/compose.yaml",
		"--env-file", ".env",
	}, args...)
	cmd := exec.Command("docker", full...)
	cmd.Dir = config.RepoRoot()
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// Run executes `docker compose ... <args>` and returns any error.
func Run(args ...string) error { return Compose(args...).Run() }

// Output captures stdout of `docker compose ... <args>`.
func Output(args ...string) ([]byte, error) {
	cmd := Compose(args...)
	cmd.Stdout = nil // let Output capture
	return cmd.Output()
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
