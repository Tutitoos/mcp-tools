package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Instala los skills en Claude Code, OpenCode y OMP (symlinks)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunSkills(false)
	},
}

func init() { rootCmd.AddCommand(skillsCmd) }

var skillsNames = []string{"codebase-memory", "headroom"}

// RunSkills is the skills-subcommand behaviour; usable from the installer TUI.
func RunSkills(dry bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	src := filepath.Join(config.RepoRoot(), "skills")
	targets := []string{
		filepath.Join(home, ".claude/skills"),
		filepath.Join(home, ".config/opencode/skills"),
		filepath.Join(home, ".omp/agent/skills"),
	}
	// stale entries from the previous naming (before rename to mcp-tools- prefix)
	stale := []string{
		filepath.Join(home, ".claude/skills/codebase-memory-mcp"),
		filepath.Join(home, ".claude/skills/headroom-mcp"),
		filepath.Join(home, ".config/opencode/skills/codebase-memory-mcp"),
		filepath.Join(home, ".config/opencode/skills/headroom-mcp"),
		filepath.Join(home, ".omp/agent/skills/codebase-memory-mcp"),
		filepath.Join(home, ".omp/agent/skills/headroom-mcp"),
	}

	fmt.Println("== cleaning stale skill dirs ==")
	for _, s := range stale {
		if _, err := os.Lstat(s); errors.Is(err, os.ErrNotExist) {
			continue
		}
		fmt.Printf("  rm %s\n", s)
		if !dry {
			if err := os.RemoveAll(s); err != nil {
				return err
			}
		}
	}

	fmt.Println("== installing symlinks ==")
	for _, t := range targets {
		if !dry {
			if err := os.MkdirAll(t, 0o755); err != nil {
				return err
			}
		}
		for _, name := range skillsNames {
			dst := filepath.Join(t, name)
			srcPath := filepath.Join(src, name)
			if !dry {
				_ = os.Remove(dst) // unlink previous (idempotent)
				if err := os.Symlink(srcPath, dst); err != nil {
					return err
				}
			}
			fmt.Printf("  %s -> %s\n", dst, srcPath)
		}
	}

	if !dry {
		fmt.Println("== verify ==")
		for _, t := range targets {
			for _, name := range skillsNames {
				f := filepath.Join(t, name, "SKILL.md")
				if _, err := os.Stat(f); err != nil {
					return fmt.Errorf("FAIL %s: %w", f, err)
				}
				fmt.Printf("  OK %s\n", f)
			}
		}
	}

	fmt.Println("\nDone. Reload / restart your MCP client (Claude Code, OpenCode, OMP) to pick up the skills.")
	return nil
}
