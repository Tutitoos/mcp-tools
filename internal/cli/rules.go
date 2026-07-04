package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Instala RULES.md en Claude Code, OpenCode y OMP",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunRules(false)
	},
}

func init() { rootCmd.AddCommand(rulesCmd) }

const (
	markerStart = "<!-- mcp-tools:start -->"
	markerEnd   = "<!-- mcp-tools:end -->"
)

// RunRules ports scripts/install-rules.sh. dry=true suppresses filesystem changes.
func RunRules(dry bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	rulesSrc := filepath.Join(config.RepoRoot(), "RULES.md")
	if _, err := os.Stat(rulesSrc); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("RULES.md no existe en %s", rulesSrc)
		}
		return err
	}

	// --- OMP: symlink como rule file ---
	ompRules := filepath.Join(home, ".omp/rules")
	ompTarget := filepath.Join(ompRules, "mcp-tools.md")
	if !dry {
		if err := os.MkdirAll(ompRules, 0o755); err != nil {
			return err
		}
		_ = os.Remove(ompTarget)
		if err := os.Symlink(rulesSrc, ompTarget); err != nil {
			return err
		}
	}

	// --- Claude Code: @import en ~/.claude/CLAUDE.md ---
	claudeMd := filepath.Join(home, ".claude/CLAUDE.md")
	importLine := "@" + rulesSrc
	if !dry {
		if err := os.MkdirAll(filepath.Dir(claudeMd), 0o755); err != nil {
			return err
		}
		content := ""
		if b, err := os.ReadFile(claudeMd); err == nil {
			content = string(b)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if !lineExists(content, importLine) {
			if content != "" && !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += importLine + "\n"
			if err := os.WriteFile(claudeMd, []byte(content), 0o644); err != nil {
				return err
			}
		}
	}

	// --- OpenCode: bloque marcado en ~/.config/opencode/AGENTS.md ---
	opencodeAgents := filepath.Join(home, ".config/opencode/AGENTS.md")
	if !dry {
		if err := os.MkdirAll(filepath.Dir(opencodeAgents), 0o755); err != nil {
			return err
		}
		content := ""
		if b, err := os.ReadFile(opencodeAgents); err == nil {
			content = string(b)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		// Strip any existing marker block
		blockRE := regexp.MustCompile(`(?ms)^` + regexp.QuoteMeta(markerStart) + `.*?^` + regexp.QuoteMeta(markerEnd) + `\n?`)
		content = blockRE.ReplaceAllString(content, "")
		rulesBody, err := os.ReadFile(rulesSrc)
		if err != nil {
			return err
		}
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += markerStart + "\n" + string(rulesBody)
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += markerEnd + "\n"
		if err := os.WriteFile(opencodeAgents, []byte(content), 0o644); err != nil {
			return err
		}
	}

	if !dry {
		for _, f := range []string{ompTarget, claudeMd, opencodeAgents} {
			if _, err := os.Stat(f); err != nil {
				return fmt.Errorf("FAIL %s: %w", f, err)
			}
			fmt.Printf("OK %s\n", f)
		}
	}
	fmt.Println("Done. Reload/restart your MCP client to pick up RULES.")
	return nil
}

func lineExists(content, want string) bool {
	for _, line := range strings.Split(content, "\n") {
		if line == want {
			return true
		}
	}
	return false
}
