package tools

import (
	"bytes"
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

// Pinned to a reviewed commit of DeusData/codebase-memory-mcp `main`
// (2026-07-13) + SHA-256 of the script at that commit. fetchVerified
// refuses to execute anything else. Bump = review the new script, then
// update BOTH constants. Closes AUDIT-2026-07-11 F7 / REVIEW-rd2 H26.
const (
	codebaseMemoryInstallURL    = "https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/2469ecc3a7a2f80debe296e1f17a1efcfdb9450c/install.sh"
	codebaseMemoryInstallSHA256 = "90ef82a3da3336ddc2c3851ad56822067b161856f24cd88cbd405fe423af6a66"
)

func installCodebaseMemory(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	installDir := filepath.Join(home, ".local", "share", "codebase-memory-mcp")
	binDir := filepath.Join(home, ".local", "bin")
	dst := filepath.Join(binDir, "codebase-memory-mcp")

	if dry {
		log(fmt.Sprintf("$ curl -fsSL %s | bash -s -- --ui --skip-config --dir %s  # sha256-verified", codebaseMemoryInstallURL, installDir))
		log(fmt.Sprintf("$ ln -snf %s/codebase-memory-mcp %s", installDir, dst))
		return nil
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}

	script, err := fetchVerified(codebaseMemoryInstallURL, codebaseMemoryInstallSHA256)
	if err != nil {
		return fmt.Errorf("codebase-memory install.sh: %w", err)
	}
	bash := exec.Command("bash", "-s", "--", "--ui", "--skip-config", "--dir", installDir)
	bash.Env = append(os.Environ(), "HOME="+home)
	bash.Stdin = bytes.NewReader(script)
	var out bytes.Buffer
	bash.Stdout = &out
	bash.Stderr = &out
	if err := bash.Run(); err != nil {
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
