package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Instala los skills en Claude Code, OpenCode y OMP (symlinks)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunSkills(false, os.Stdout)
	},
}

func init() { rootCmd.AddCommand(skillsCmd) }

var skillsNames = []string{"codebase-memory", "mem0", "serena", "tokensave"}

// RunSkills is the skills-subcommand behaviour; usable from the installer TUI.
func RunSkills(dry bool, out io.Writer) error {
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
		filepath.Join(home, ".config/opencode/skills/codebase-memory-mcp"),
		filepath.Join(home, ".omp/agent/skills/codebase-memory-mcp"),
	}

	fmt.Fprintln(out, "── install skills (Claude Code, OpenCode, OMP)")

	// Silent stale cleanup — cuenta y menciona SOLO si hubo trabajo.
	stalesRemoved := 0
	for _, s := range stale {
		if _, err := os.Lstat(s); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if !dry {
			if err := os.RemoveAll(s); err != nil {
				return err
			}
		}
		stalesRemoved++
	}
	if stalesRemoved > 0 {
		fmt.Fprintf(out, "  · %d stale dir(s) removed\n", stalesRemoved)
	}

	// Silent symlink install; falla → mensaje concreto con la ruta que rompió.
	symlinksTotal := 0
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
					return fmt.Errorf("symlink %s: %w", dst, err)
				}
			}
			symlinksTotal++
		}
	}

	// Silent verify; falla → error con la ruta que no existe.
	if !dry {
		for _, t := range targets {
			for _, name := range skillsNames {
				f := filepath.Join(t, name, "SKILL.md")
				if _, err := os.Stat(f); err != nil {
					return fmt.Errorf("FAIL %s: %w", f, err)
				}
			}
		}
		fmt.Fprintf(out, "  OK %d symlinks verificados (%d skills × %d clients)\n", symlinksTotal, len(skillsNames), len(targets))
	} else {
		fmt.Fprintf(out, "  OK %d symlinks (dry — %d skills × %d clients)\n", symlinksTotal, len(skillsNames), len(targets))
	}
	return nil
}
