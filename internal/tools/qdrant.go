package tools

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/docker"
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
	volArgs := []string{"volume", "create", "mcp-qdrant-storage"}
	if dry {
		log("$ docker " + strings.Join(volArgs, " "))
	} else {
		volCmd := exec.Command("docker", volArgs...)
		volCmd.Dir = config.RepoRoot()
		volCmd.Env = os.Environ()
		if err := runCombined(volCmd, "docker volume create qdrant-storage"); err != nil {
			return err
		}
	}

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
	timeout := 5 * time.Second
	if out, err := docker.RunCmdWithTimeout(timeout, "container", "inspect", "-f", "{{.State.Status}}", "mcp-tools-mem0-qdrant").Output(); err == nil {
		p.Installed = true
		p.Extra["state"] = strings.TrimSpace(string(out))
	}
	if img, err := docker.RunCmdWithTimeout(timeout, "container", "inspect", "-f", "{{.Config.Image}}", "mcp-tools-mem0-qdrant").Output(); err == nil {
		p.Version = strings.TrimSpace(string(img))
	}
	// image_drift: local image digest != running container's image digest
	if imgDigest, err := docker.RunCmdWithTimeout(timeout, "images", "inspect", "mcp-tools-mem0-qdrant", "-f", "{{.Id}}").Output(); err == nil {
		p.Extra["image_digest"] = strings.TrimSpace(string(imgDigest))
		if ctrDigest, err := docker.RunCmdWithTimeout(timeout, "container", "inspect", "-f", "{{.Image}}", "mcp-tools-mem0-qdrant").Output(); err == nil {
			p.Extra["container_digest"] = strings.TrimSpace(string(ctrDigest))
			if p.Extra["image_digest"] != p.Extra["container_digest"] {
				p.Extra["image_drift"] = true
			}
		}
	} else {
		p.Extra["image_missing"] = true
	}
	return p, nil
}
