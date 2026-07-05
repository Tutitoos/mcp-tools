package cli

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

var envForce bool

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "(Re)genera .env si no existe y crea los directorios de datos",
	Long:  "Idempotente: no toca un .env existente salvo con --force. Crea siempre los directorios bajo $MCP_TOOLS_DATA.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunEnv(false, envForce, os.Stdout)
	},
}

func init() {
	envCmd.Flags().BoolVar(&envForce, "force", false, "sobrescribe .env si ya existe")
	rootCmd.AddCommand(envCmd)
}

// RunEnv is the env-subcommand behaviour, reusable from the installer TUI.
// If dry is true, no filesystem changes happen; only the intended actions are printed.
func RunEnv(dry, force bool, out io.Writer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	repoDir := config.RepoRoot()
	dataDir := filepath.Join(home, "mcp-tools-data")
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

	// .env write (idempotent)
	if _, err := os.Stat(envPath); err == nil && !force {
		if dry {
			fmt.Fprintf(out, "  OK %s ya existe, se conserva (dry)\n", envPath)
		} else {
			fmt.Fprintf(out, "  OK %s ya existe, se conserva\n", envPath)
		}
	} else {
		if dry {
			fmt.Fprintf(out, "  OK escribiría %s con 7 variables (dry)\n", envPath)
		} else {
			if err := config.WriteEnv(envPath, contents); err != nil {
				return fmt.Errorf("escribir .env: %w", err)
			}
			fmt.Fprintf(out, "  OK generado %s\n", envPath)
		}
	}

	// .env.mem0 write (idempotent; user may have edited it via `mcp-tools select-model`)
	mem0EnvPath := filepath.Join(repoDir, ".env.mem0")
	if _, err := os.Stat(mem0EnvPath); err == nil && !force {
		if dry {
			fmt.Fprintf(out, "  OK %s ya existe, se conserva (dry)\n", mem0EnvPath)
		} else {
			fmt.Fprintf(out, "  OK %s ya existe, se conserva\n", mem0EnvPath)
		}
	} else {
		mem0EnvBody := fmt.Sprintf(`MEM0_PROVIDER=ollama
MEM0_LLM_MODEL=qwen2.5:7b

MEM0_EMBED_PROVIDER=ollama
MEM0_EMBED_MODEL=bge-m3
MEM0_OLLAMA_URL=http://127.0.0.1:11434/

MEM0_QDRANT_URL=http://127.0.0.1:6333/
MEM0_COLLECTION=mem0_%s
MEM0_ENABLE_GRAPH=false

MEM0_HISTORY_DB_PATH=$HOME/mcp-tools-data/mem0/history/history.db
MEM0_OLLAMA_THINK=false
`, u.Username)
		if dry {
			fmt.Fprintf(out, "  OK escribiría %s con defaults (dry)\n", mem0EnvPath)
		} else {
			if err := os.WriteFile(mem0EnvPath, []byte(mem0EnvBody), 0o644); err != nil {
				return fmt.Errorf("escribir .env.mem0: %w", err)
			}
			fmt.Fprintf(out, "  OK generado %s\n", mem0EnvPath)
		}
	}

	// mkdirs (always; -p is idempotent)
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
