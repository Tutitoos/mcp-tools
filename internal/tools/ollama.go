package tools

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/state"
)

func ollamaTool() Tool {
	return Tool{
		Key:       "ollama",
		Label:     "Ollama LLM + embed",
		Summary:   "LLM+embed inference; auto-usa GPU cuando nvidia-toolkit está seleccionado",
		Deploy:    DeployDocker,
		DefaultOn: true,
		Install: func(dry bool, log func(string)) error {
			return installOllama(dry, log, loadStateOrEmpty())
		},
		Upgrade: func(dry bool, log func(string)) error {
			return installOllama(dry, log, loadStateOrEmpty())
		},
		Uninstall: uninstallOllama,
		Status:    statusOllama,
	}
}

// loadStateOrEmpty reads state.json and returns the zero state on any error —
// enough for OllamaComposeFiles to decide GPU-overlay resolution.
func loadStateOrEmpty() state.State {
	s, _ := state.Load()
	return s
}

func installOllama(dry bool, log func(string), st state.State) error {
	files := OllamaComposeFiles(st)
	args := []string{"compose"}
	for _, f := range files {
		args = append(args, "-f", f)
	}
	args = append(args, "--env-file", ".env", "up", "-d", "mcp_tools_ollama")

	if dry {
		log("$ docker " + strings.Join(args, " "))
		log("$ (post) pull MEM0_LLM_MODEL + MEM0_EMBED_MODEL en mcp-tools-ollama")
		return nil
	}
	cmd := exec.Command("docker", args...)
	cmd.Dir = config.RepoRoot()
	cmd.Env = os.Environ()
	if err := runCombined(cmd, "docker compose up ollama"); err != nil {
		return err
	}
	// Post-install: pull LLM + embed models declared in .env.mem0.
	return pullMem0Models(log)
}

func uninstallOllama(dry bool, log func(string)) error {
	args := []string{
		"compose",
		"-f", "dockers/compose.yaml",
		"--env-file", ".env",
		"rm", "-sf", "mcp_tools_ollama",
	}
	if dry {
		log("$ docker " + strings.Join(args, " "))
		return nil
	}
	cmd := exec.Command("docker", args...)
	cmd.Dir = config.RepoRoot()
	cmd.Env = os.Environ()
	return runCombined(cmd, "docker compose rm ollama")
}

func statusOllama() (StatusPayload, error) {
	p := StatusPayload{Extra: map[string]any{}}
	out, err := exec.Command("docker", "container", "inspect", "-f", "{{.State.Status}}", "mcp-tools-ollama").Output()
	if err != nil {
		return p, nil
	}
	p.Installed = true
	p.Extra["state"] = strings.TrimSpace(string(out))
	if v, err := exec.Command("docker", "exec", "mcp-tools-ollama", "ollama", "--version").Output(); err == nil {
		p.Version = firstLine(string(v))
	}
	p.Extra["gpu_overlay"] = len(OllamaComposeFiles(loadStateOrEmpty())) > 1
	return p, nil
}

func pullMem0Models(log func(string)) error {
	envMem0 := config.EnvMem0File()
	if _, err := os.Stat(envMem0); errors.Is(err, os.ErrNotExist) {
		log("SKIP pull: .env.mem0 aún no existe")
		return nil
	}
	env, err := config.LoadEnv(envMem0)
	if err != nil {
		return fmt.Errorf(".env.mem0: %w", err)
	}
	var models []string
	if m := env["MEM0_LLM_MODEL"]; m != "" {
		models = append(models, m)
	}
	if m := env["MEM0_EMBED_MODEL"]; m != "" {
		models = append(models, m)
	}
	if len(models) == 0 {
		log("SKIP pull: ni MEM0_LLM_MODEL ni MEM0_EMBED_MODEL en .env.mem0")
		return nil
	}
	var listOut []byte
	for range 10 {
		listOut, err = exec.Command("docker", "exec", "mcp-tools-ollama", "ollama", "list").Output()
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("ollama no responde tras 20s: %w", err)
	}
	for _, m := range models {
		if strings.Contains(string(listOut), m+" ") || strings.Contains(string(listOut), m+"\t") {
			continue
		}
		log("· pull " + m)
		cmd := exec.Command("docker", "exec", "mcp-tools-ollama", "ollama", "pull", m)
		if err := runCombined(cmd, "ollama pull "+m); err != nil {
			return err
		}
	}
	return nil
}
