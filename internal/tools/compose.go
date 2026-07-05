package tools

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/state"
)

// OllamaComposeFiles is the shared authority on which compose files describe
// mcp_tools_ollama: base compose.yaml alone, or base + GPU overlay when the
// host has an NVIDIA GPU and the nvidia-toolkit tool is in the selected set.
//
// Returned paths are relative to config.RepoRoot() so callers pass them as
// `-f` to `docker compose` with cwd = RepoRoot().
func OllamaComposeFiles(st state.State) []string {
	base := []string{"dockers/compose.yaml"}
	if !st.Has("nvidia-toolkit") || !hasNvidiaGPU() {
		return base
	}
	overlay := filepath.Join(config.RepoRoot(), "dockers/ollama-gpu-overlay.yml")
	if _, err := os.Stat(overlay); err != nil {
		return base
	}
	return append(base, "dockers/ollama-gpu-overlay.yml")
}

// hasNvidiaGPU reports whether `nvidia-smi -L` exits 0 with non-empty output.
func hasNvidiaGPU() bool {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return false
	}
	out, err := exec.Command("nvidia-smi", "-L").Output()
	if err != nil {
		return false
	}
	return len(out) > 0
}
