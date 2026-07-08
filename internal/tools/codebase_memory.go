package tools

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func codebaseMemoryTool() Tool {
	return Tool{
		Key:       "codebase-memory",
		Label:     "codebase-memory-mcp",
		Summary:   "Grafo de código + búsqueda semántica (host binary)",
		Deploy:    DeployHost,
		DefaultOn: true,
		Install:   installCodebaseMemory,
		Upgrade:   installCodebaseMemory, // upstream install.sh is idempotent
		Uninstall: uninstallCodebaseMemory,
		Status:    statusCodebaseMemory,
	}
}

// TODO(security): pin to a known-good commit. The script is fetched from
// `main` (not a tagged release) and executed locally. See docs/REVIEW-rd2.md (H26).
const codebaseMemoryInstallURL = "https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.sh"

func installCodebaseMemory(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	installDir := filepath.Join(home, ".local", "share", "codebase-memory-mcp")
	binDir := filepath.Join(home, ".local", "bin")
	dst := filepath.Join(binDir, "codebase-memory-mcp")

	if dry {
		log(fmt.Sprintf("$ curl -fsSL %s | bash -s -- --standard --skip-config --dir %s", codebaseMemoryInstallURL, installDir))
		log(fmt.Sprintf("$ ln -snf %s/codebase-memory-mcp %s", installDir, dst))
		return nil
	}
	if _, err := exec.LookPath("curl"); err != nil {
		return errors.New("curl no está en PATH; instala curl antes de codebase-memory-mcp")
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}

	curl := exec.Command("curl", "-fsSL", codebaseMemoryInstallURL)
	bash := exec.Command("bash", "-s", "--", "--ui", "--skip-config", "--dir", installDir)
	bash.Env = append(os.Environ(), "HOME="+home)
	pipe, err := curl.StdoutPipe()
	if err != nil {
		return err
	}
	bash.Stdin = pipe
	var out bytes.Buffer
	bash.Stdout = &out
	bash.Stderr = &out
	if err := bash.Start(); err != nil {
		return fmt.Errorf("codebase-memory bash start: %w", err)
	}
	if err := curl.Run(); err != nil {
		return fmt.Errorf("codebase-memory curl: %w", err)
	}
	if err := bash.Wait(); err != nil {
		return fmt.Errorf("codebase-memory install.sh: %w\n%s", err, strings.TrimSpace(out.String()))
	}

	// Resolve the actual install path: prefer our --dir; fall back to parsing
	// "Installed to <path>" from install.sh output.
	binSrc := filepath.Join(installDir, "codebase-memory-mcp")
	if _, err := os.Stat(binSrc); err != nil {
		if p := parseInstalledTo(out.String()); p != "" {
			binSrc = filepath.Join(p, "codebase-memory-mcp")
		}
	}
	if _, err := os.Stat(binSrc); err != nil {
		return fmt.Errorf("codebase-memory-mcp binary no encontrado tras install.sh: %w", err)
	}
	_ = os.Remove(dst)
	if err := os.Symlink(binSrc, dst); err != nil {
		return fmt.Errorf("symlink %s -> %s: %w", dst, binSrc, err)
	}
	return nil
}

func uninstallCodebaseMemory(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	installDir := filepath.Join(home, ".local", "share", "codebase-memory-mcp")
	dst := filepath.Join(home, ".local", "bin", "codebase-memory-mcp")

	if dry {
		log("$ codebase-memory-mcp uninstall  # si existe")
		log("$ rm -f " + dst)
		log("$ rm -rf " + installDir)
		return nil
	}
	if bin, err := exec.LookPath("codebase-memory-mcp"); err == nil {
		// Best-effort; some builds expose no `uninstall` verb.
		_ = exec.Command(bin, "uninstall").Run()
	}
	_ = os.Remove(dst)
	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("rm -rf %s: %w", installDir, err)
	}
	return nil
}

func statusCodebaseMemory() (StatusPayload, error) {
	p := StatusPayload{}
	bin := which("codebase-memory-mcp")
	if bin == "" {
		return p, nil
	}
	p.Installed = true
	p.Binary = bin
	if v := versionOf(bin, "--version"); v != "" {
		p.Version = v
	}
	return p, nil
}

// parseInstalledTo scans installer output for `Installed to <path>` (upstream
// convention).
func parseInstalledTo(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Installed to ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Installed to "))
		}
	}
	return ""
}
