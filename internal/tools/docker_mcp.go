package tools

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const dockerMCPVersion = "v0.43.1"

var dockerMCPReleases = map[string]struct {
	asset  string
	sha256 string
}{
	"darwin-amd64": {"docker-mcp-darwin-amd64.tar.gz", "ad728238767096f8aeccf6c7ffa5a51f20dd45aebed770a1d1c13ea0b2fe6265"},
	"darwin-arm64": {"docker-mcp-darwin-arm64.tar.gz", "6f0e4d9c3c7f75f93e5ab86a5e2b5e0b4d02c9e44dc3a3cd5a253e61c0ad62b9"},
	"linux-amd64":  {"docker-mcp-linux-amd64.tar.gz", "92434f3afb995bf0e923a3ad84fe4e80f09818312cc8d0bb6c9aeb079d3a32a1"},
	"linux-arm64":  {"docker-mcp-linux-arm64.tar.gz", "698161c844a48a10eacb405304f327acc8a66f5054cd19bfbb51611bea94951e"},
}

func dockerMCPTool() Tool {
	return Tool{
		Key:       "docker-mcp-toolkit",
		Label:     "Docker MCP Toolkit",
		Summary:   "Catálogo y gateway MCP oficial de Docker (CLI plugin)",
		Deploy:    DeployHost,
		DefaultOn: false,
		Install:   installDockerMCP,
		Upgrade:   installDockerMCP,
		Uninstall: uninstallDockerMCP,
		Status:    statusDockerMCP,
	}
}

func dockerMCPPluginPath(home string) string {
	return filepath.Join(home, ".docker", "cli-plugins", "docker-mcp")
}

func installDockerMCP(dry bool, log func(string)) error {
	if which("docker") == "" {
		return errors.New("docker no está en PATH; instala Docker Engine o Docker Desktop primero")
	}
	release, ok := dockerMCPReleases[runtime.GOOS+"-"+runtime.GOARCH]
	if !ok {
		return fmt.Errorf("Docker MCP Gateway no publica binario para %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	url := fmt.Sprintf("https://github.com/docker/mcp-gateway/releases/download/%s/%s", dockerMCPVersion, release.asset)
	home, err := hostHome()
	if err != nil {
		return err
	}
	if dry {
		log("$ download " + url + " (sha256 " + release.sha256 + ")")
		log("$ docker mcp feature enable profiles")
		log("$ docker mcp feature enable dynamic-tools")
		return nil
	}
	bundle, err := fetchVerifiedLimit(url, release.sha256, 64<<20)
	if err != nil {
		return fmt.Errorf("docker-mcp release: %w", err)
	}
	if err := installDockerMCPArchive(bundle, dockerMCPPluginPath(home)); err != nil {
		return err
	}
	for _, feature := range []string{"profiles", "dynamic-tools"} {
		cmd := exec.Command("docker", "mcp", "feature", "enable", feature)
		cmd.Env = append(os.Environ(), "DOCKER_MCP_IN_CONTAINER=1")
		if err := runCombined(cmd, "docker mcp feature enable "+feature); err != nil {
			return err
		}
	}
	return nil
}

func installDockerMCPArchive(bundle []byte, path string) error {
	gz, err := gzip.NewReader(bytes.NewReader(bundle))
	if err != nil {
		return fmt.Errorf("docker-mcp archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return errors.New("docker-mcp archive: binario ausente")
		}
		if err != nil {
			return fmt.Errorf("docker-mcp archive: %w", err)
		}
		if header.Name != "docker-mcp" || !header.FileInfo().Mode().IsRegular() {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		tmp, err := os.CreateTemp(filepath.Dir(path), ".docker-mcp-*")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		if _, err := io.Copy(tmp, tr); err != nil {
			tmp.Close()
			return err
		}
		if err := tmp.Close(); err != nil {
			return err
		}
		if err := os.Chmod(tmp.Name(), 0o755); err != nil {
			return err
		}
		return os.Rename(tmp.Name(), path)
	}
}

func uninstallDockerMCP(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	path := dockerMCPPluginPath(home)
	if dry {
		log("$ rm -f " + path)
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func statusDockerMCP() (StatusPayload, error) {
	home, err := hostHome()
	if err != nil {
		return StatusPayload{}, err
	}
	path := dockerMCPPluginPath(home)
	p := StatusPayload{Binary: path}
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		p.Installed = true
		if docker := which("docker"); docker != "" {
			p.Version = versionOf(docker, "mcp", "version")
		}
	}
	return p, nil
}
