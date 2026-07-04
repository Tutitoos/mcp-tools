package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/config"

	"github.com/Tutitoos/mcp-tools/internal/tui/installer"
)

var installDry bool

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Instalador TUI end-to-end (10 pasos)",
	Long:  "Ejecuta la secuencia completa: prereq → env → mem0-src → build → wrappers → skills → rules → mcp-config → up → smoke. Idempotente. --dry captura los comandos sin ejecutarlos.",
	RunE: func(cmd *cobra.Command, args []string) error {
		steps := buildSteps()
		p := tea.NewProgram(installer.New(steps, installDry))
		result, err := p.Run()
		if err != nil {
			return err
		}
		if m, ok := result.(installer.Model); ok {
			if code := m.ExitCode(); code != 0 {
				os.Exit(code)
			}
		}
		return nil
	},
}

func init() {
	installCmd.Flags().BoolVar(&installDry, "dry", false, "no ejecuta comandos; solo muestra qué haría")
	rootCmd.AddCommand(installCmd)
}

func buildSteps() []installer.Step {
	return []installer.Step{
		{Key: "prereq", Label: "Comprobar prerequisitos (docker + docker compose)", Phase: "Preparación", Run: stepPrereq},
		{Key: "env", Label: "Generar .env desde el host", Phase: "Preparación", Run: stepEnv},
		{Key: "mem0-src", Label: "Verificar clon de mem0-mcp-selfhosted", Phase: "Preparación", Run: stepMem0Src},
		{Key: "build", Label: "docker compose build (puede tardar)", Phase: "Build", Run: stepBuild},
		{Key: "wrappers", Label: "Wrappers en ~/.local/bin/", Phase: "Instalación", Run: stepWrappers},
		{Key: "skills", Label: "Skills globales", Phase: "Instalación", Run: stepSkills},
		{Key: "rules", Label: "RULES.md globales", Phase: "Instalación", Run: stepRules},
		{Key: "mcp-config", Label: "Registrar MCPs en Claude Code / OpenCode / OMP", Phase: "Instalación", Run: stepMcpConfig},
		{Key: "up", Label: "Arrancar contenedores (docker compose up -d)", Phase: "Arranque", Run: stepUp},
		{Key: "smoke", Label: "Smoke test MCP handshake", Phase: "Arranque", Run: stepSmoke},
	}
}

func stepPrereq(dry bool, log func(string)) error {
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

func stepEnv(dry bool, log func(string)) error {
	if dry {
		log("$ mcp-tools env")
		return nil
	}
	return RunEnv(false, false)
}

func stepMem0Src(dry bool, log func(string)) error {
	return installer.CheckMem0Src(dry, log)
}

func stepBuild(dry bool, log func(string)) error {
	if dry {
		log("$ docker compose -f dockers/compose.yaml --env-file .env build")
		return nil
	}
	cmd := exec.Command("docker", "compose", "-f", "dockers/compose.yaml", "--env-file", ".env", "build")
	cmd.Dir = config.RepoRoot()
	cmd.Env = os.Environ()
	return cmd.Run()
}

func stepWrappers(dry bool, log func(string)) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	binDir := filepath.Join(home, ".local/bin")
	if dry {
		log(fmt.Sprintf("$ mkdir -p %s", binDir))
		for _, w := range []string{"codebase-memory", "mem0", "headroom"} {
			src := filepath.Join(config.RepoRoot(), "scripts/wrappers/mcp-tools-"+w+"-docker")
			log(fmt.Sprintf(`$ ln -snf %s %s/`, src, binDir))
		}
		return nil
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	for _, w := range []string{"codebase-memory", "mem0", "headroom"} {
		src := filepath.Join(config.RepoRoot(), "scripts/wrappers/mcp-tools-"+w+"-docker")
		dst := filepath.Join(binDir, "mcp-tools-"+w+"-docker")
		_ = os.Remove(dst)
		if err := os.Symlink(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func stepSkills(dry bool, log func(string)) error {
	if dry {
		log("$ mcp-tools skills")
		return nil
	}
	return RunSkills(false)
}

func stepRules(dry bool, log func(string)) error {
	if dry {
		log("$ mcp-tools rules")
		return nil
	}
	return RunRules(false)
}

func stepMcpConfig(dry bool, log func(string)) error {
	if dry {
		log("$ mcp-tools mcp-config")
		return nil
	}
	return RunMcpConfig(false)
}

func stepUp(dry bool, log func(string)) error {
	if dry {
		log("$ docker compose -f dockers/compose.yaml --env-file .env up -d")
		return nil
	}
	cmd := exec.Command("docker", "compose", "-f", "dockers/compose.yaml", "--env-file", ".env", "up", "-d")
	cmd.Dir = config.RepoRoot()
	cmd.Env = os.Environ()
	return cmd.Run()
}

func stepSmoke(dry bool, log func(string)) error {
	home, _ := os.UserHomeDir()
	cbm := filepath.Join(home, ".local/bin/mcp-tools-codebase-memory-docker")
	hr := filepath.Join(home, ".local/bin/mcp-tools-headroom-docker")
	m0 := filepath.Join(home, ".local/bin/mcp-tools-mem0-docker")

	if dry {
		log(fmt.Sprintf("$ %s --version", cbm))
		log(fmt.Sprintf("$ timeout 5 %s --help >/dev/null 2>&1 || true", hr))
		log(fmt.Sprintf(`$ echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"installer","version":"1"}}}' | timeout 15 %s | grep -q '"serverInfo"'`, m0))
		return nil
	}

	if err := exec.Command(cbm, "--version").Run(); err != nil {
		return fmt.Errorf("codebase-memory smoke: %w", err)
	}
	// headroom: best-effort with timeout
	hrCmd := exec.Command("timeout", "5", hr, "--help")
	hrCmd.Stdout, hrCmd.Stderr = nil, nil
	_ = hrCmd.Run()

	// mem0: JSON-RPC initialize and grep serverInfo
	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"installer","version":"1"}}}`
	m0Cmd := exec.Command("timeout", "15", m0)
	m0Cmd.Stdin = strings.NewReader(init)
	m0Cmd.Env = append(os.Environ(), "MCP_TOOLS_ROOT="+config.RepoRoot())
	var out bytes.Buffer
	m0Cmd.Stdout = &out
	m0Cmd.Stderr = nil
	if err := m0Cmd.Run(); err != nil {
		return fmt.Errorf("mem0 handshake: %w", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"serverInfo"`)) {
		return fmt.Errorf("mem0 handshake: no serverInfo in output")
	}
	return nil
}

// time import kept because bubbletea may need it later; suppress unused warnings.
var _ = time.Time{}
