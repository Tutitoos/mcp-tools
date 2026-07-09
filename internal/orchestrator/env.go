package orchestrator

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

// RunEnv is the orchestrator's port of the legacy cli.RunEnv. It generates
// / refreshes the per-host `.env` and `.env.mem0` and ensures the data
// directories exist. Idempotent unless `force` is true.
func RunEnv(dry, force bool, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	repoDir := config.RepoRoot()
	dataDir := config.DataDir()
	envPath := filepath.Join(repoDir, ".env")

	u, err := user.Current()
	if err != nil {
		return err
	}

	contents := map[string]string{
		"HOST_HOME":      home,
		"HOST_UID":       fmt.Sprintf("%d", syscall.Getuid()),
		"HOST_GID":       fmt.Sprintf("%d", syscall.Getgid()),
		"MCP_TOOLS_ROOT": repoDir,
		"MCP_TOOLS_DATA": dataDir,
		"MCP_TOOLS_BIND": "0.0.0.0",
		"MEM0_USER_ID":   u.Username,
	}
	fmt.Fprintln(out, "── env")

	if _, err := os.Stat(envPath); err == nil && !force {
		fmt.Fprintf(out, "  OK %s ya existe, se conserva%s\n", envPath, dryPrefix(dry))
	} else {
		if dry {
			fmt.Fprintf(out, "  OK escribiría %s con 7 variables (dry)\n", envPath)
		} else {
			if err := config.WriteEnv(envPath, contents); err != nil {
				return fmt.Errorf("escribir .env: %w", err)
			}
			if err := os.Chmod(envPath, 0o600); err != nil {
				return fmt.Errorf("chmod .env: %w", err)
			}
			fmt.Fprintf(out, "  OK generado %s\n", envPath)
		}
	}

	mem0EnvPath := filepath.Join(repoDir, ".env.mem0")
	if _, err := os.Stat(mem0EnvPath); err == nil && !force {
		fmt.Fprintf(out, "  OK %s ya existe, se conserva%s\n", mem0EnvPath, dryPrefix(dry))
	} else {
		mem0EnvBody := fmt.Sprintf(`MEM0_PROVIDER=ollama
MEM0_LLM_MODEL=qwen2.5:7b

MEM0_EMBED_PROVIDER=ollama
MEM0_EMBED_MODEL=bge-m3
MEM0_OLLAMA_URL=http://127.0.0.1:11434/

MEM0_QDRANT_URL=http://127.0.0.1:6333/
MEM0_COLLECTION=mem0_%s
MEM0_ENABLE_GRAPH=false

MEM0_HISTORY_DB_PATH=%s
MEM0_OLLAMA_THINK=false
`, u.Username, filepath.Join(dataDir, "mem0/history/history.db"))
		if dry {
			fmt.Fprintf(out, "  OK escribiría %s con defaults (dry)\n", mem0EnvPath)
		} else {
			if err := os.WriteFile(mem0EnvPath, []byte(mem0EnvBody), 0o600); err != nil {
				return fmt.Errorf("escribir .env.mem0: %w", err)
			}
			if err := os.Chmod(mem0EnvPath, 0o600); err != nil {
				return fmt.Errorf("chmod .env.mem0: %w", err)
			}
			fmt.Fprintf(out, "  OK generado %s\n", mem0EnvPath)
		}
	}

	dirs := []string{
		filepath.Join(dataDir, "mem0/history"),
		filepath.Join(dataDir, "mem0/uv-cache"),
		filepath.Join(dataDir, "mem0/config"),
		filepath.Join(dataDir, "ollama"),
	}
	for _, d := range dirs {
		if dry {
			continue
		}
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	if dry {
		fmt.Fprintf(out, "  OK data en %s (dry — no se crean directorios)\n", dataDir)
	} else {
		fmt.Fprintf(out, "  OK data en %s\n", dataDir)
	}
	return nil
}

// BootstrapEnv is the prereq step every orchestrator verb runs before
// touching state: it (re)generates the .env files via RunEnv. It never
// probes for Docker — callers that actually deploy a Docker-based tool
// (qdrant, ollama Install/Upgrade/Uninstall closures) call
// docker.EnsureAvailable themselves, so a verb that never touches a
// DeployDocker tool never requires Docker on the host.
func BootstrapEnv(dry bool, log LogFn) error {
	out := writerFromLog(log)
	return RunEnv(dry, false, out)
}

func dryPrefix(dry bool) string {
	if dry {
		return " (dry)"
	}
	return ""
}

// writerFromLog turns an incremental LogFn into an io.Writer by writing
// line-by-line. Used by legacy helpers that take an io.Writer.
func writerFromLog(log LogFn) io.Writer {
	if log == nil {
		return io.Discard
	}
	return logWriter{log: log}
}

type logWriter struct{ log LogFn }

func (w logWriter) Write(p []byte) (int, error) {
	w.log(string(p))
	return len(p), nil
}
