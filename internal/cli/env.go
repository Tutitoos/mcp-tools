package cli

import (
	"fmt"
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
		return RunEnv(false, envForce)
	},
}

func init() {
	envCmd.Flags().BoolVar(&envForce, "force", false, "sobrescribe .env si ya existe")
	rootCmd.AddCommand(envCmd)
}

// RunEnv is the env-subcommand behaviour, reusable from the installer TUI.
// If dry is true, no filesystem changes happen; only the intended actions are printed.
func RunEnv(dry, force bool) error {
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
		"HOST_HOME":                       home,
		"HOST_UID":                        fmt.Sprintf("%d", syscall.Getuid()),
		"HOST_GID":                        fmt.Sprintf("%d", syscall.Getgid()),
		"MCP_TOOLS_ROOT":                  repoDir,
		"MCP_TOOLS_DATA":                  dataDir,
		"MCP_TOOLS_CODEBASE_MEMORY_IMAGE": "mcp-tools/codebase-memory:latest",
		"MCP_TOOLS_MEM0_IMAGE":            "mcp-tools/mem0:latest",
		"MCP_TOOLS_HEADROOM_IMAGE":        "mcp-tools/headroom:latest",
		"MEM0_SRC_PATH":                   filepath.Join(dataDir, "mem0/src"),
		"MEM0_USER_ID":                    u.Username,
	}

	// .env write (idempotent)
	if _, err := os.Stat(envPath); err == nil && !force {
		if dry {
			fmt.Printf("OK: %s ya existe, se conserva (dry)\n", envPath)
		} else {
			fmt.Printf("OK: %s ya existe, se conserva\n", envPath)
		}
	} else {
		if dry {
			fmt.Printf("OK: escribiría %s con 10 variables (dry)\n", envPath)
		} else {
			if err := config.WriteEnv(envPath, contents); err != nil {
				return fmt.Errorf("escribir .env: %w", err)
			}
			fmt.Printf("OK: generado %s\n", envPath)
		}
	}

	// mkdirs (always; -p is idempotent)
	dirs := []string{
		filepath.Join(dataDir, "codebase-memory/cache"),
		filepath.Join(dataDir, "codebase-memory/config"),
		filepath.Join(dataDir, "mem0/history"),
		filepath.Join(dataDir, "mem0/uv-cache"),
		filepath.Join(dataDir, "mem0/config"),
		filepath.Join(dataDir, "headroom/cache"),
		filepath.Join(dataDir, "headroom/config"),
		filepath.Join(dataDir, "headroom/share"),
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
		fmt.Printf("OK: data en %s (dry — no se crean directorios)\n", dataDir)
	} else {
		fmt.Printf("OK: data en %s\n", dataDir)
	}
	return nil
}
