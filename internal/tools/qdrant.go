package tools

import (
	"os"
	"os/exec"
	"strings"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

func qdrantTool() Tool {
	return Tool{
		Key:       "qdrant",
		Label:     "Qdrant vector store",
		Summary:   "Vector DB para mem0 (Docker, external volume mcp-qdrant-storage)",
		Deploy:    DeployDocker,
		DefaultOn: true,
		Install:   installQdrant,
		Upgrade:   installQdrant, // `up -d` pulls the pinned image; no `--pull` here to avoid surprise breakage.
		Uninstall: uninstallQdrant,
		Status:    statusQdrant,
	}
}

func installQdrant(dry bool, log func(string)) error {
	args := []string{
		"compose",
		"-f", "dockers/compose.yaml",
		"--env-file", ".env",
		"up", "-d", "mcp_tools_mem0_qdrant",
	}
	if dry {
		log("$ docker " + strings.Join(args, " "))
		return nil
	}
	cmd := exec.Command("docker", args...)
	cmd.Dir = config.RepoRoot()
	cmd.Env = os.Environ()
	return runCombined(cmd, "docker compose up qdrant")
}

func uninstallQdrant(dry bool, log func(string)) error {
	args := []string{
		"compose",
		"-f", "dockers/compose.yaml",
		"--env-file", ".env",
		"rm", "-sf", "mcp_tools_mem0_qdrant",
	}
	if dry {
		log("$ docker " + strings.Join(args, " "))
		return nil
	}
	cmd := exec.Command("docker", args...)
	cmd.Dir = config.RepoRoot()
	cmd.Env = os.Environ()
	return runCombined(cmd, "docker compose rm qdrant")
}

func statusQdrant() (StatusPayload, error) {
	p := StatusPayload{Extra: map[string]any{}}
	out, err := exec.Command("docker", "container", "inspect", "-f", "{{.State.Status}}", "mcp-tools-mem0-qdrant").Output()
	if err != nil {
		return p, nil
	}
	state := strings.TrimSpace(string(out))
	p.Installed = true
	p.Extra["state"] = state
	// image tag = version
	if img, err := exec.Command("docker", "container", "inspect", "-f", "{{.Config.Image}}", "mcp-tools-mem0-qdrant").Output(); err == nil {
		p.Version = strings.TrimSpace(string(img))
	}
	return p, nil
}
