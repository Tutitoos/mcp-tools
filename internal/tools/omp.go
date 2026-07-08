package tools

import (
	"os"
	"os/exec"
)

// ompTool is the OMP (oh-my-pi) CLI by can1357. It's a CLIENT that consumes
// MCP servers registered in ~/.omp/agent/mcp.json (see mcp.ConfigureOMP). The
// upstream distribution is the script at https://omp.sh/install.sh, which
// writes the binary to ~/.local/bin/omp → Deploy = DeployHost.
func ompTool() Tool {
	return Tool{
		Key:       "omp",
		Label:     "OMP CLI (oh-my-pi)",
		Summary:   "Cliente MCP can1357/oh-my-pi (curl https://omp.sh/install.sh | bash; ~/.local/bin)",
		Deploy:    DeployHost,
		DefaultOn: false,
		Install:   installOMP,
		Upgrade:   upgradeOMP,
		Uninstall: uninstallOMP,
		Status:    statusOMP,
	}
}

const ompInstallURL = "https://omp.sh/install.sh"

func installOMP(dry bool, log func(string)) error {
	if dry {
		log("$ curl -fsSL " + ompInstallURL + " | bash")
		return nil
	}
	cmd := exec.Command("bash", "-c", "curl -fsSL "+ompInstallURL+" | bash")
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func upgradeOMP(dry bool, log func(string)) error {
	// The official installer is idempotent: re-running it picks the latest
	// stable release and replaces the existing binary in place.
	return installOMP(dry, log)
}

func uninstallOMP(dry bool, log func(string)) error {
	bin, err := exec.LookPath("omp")
	if err != nil {
		log("  omp no está en PATH — nada que desinstalar")
		return nil
	}
	if dry {
		log("$ rm -f " + bin)
		return nil
	}
	if err := os.Remove(bin); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func statusOMP() (StatusPayload, error) {
	p := StatusPayload{}
	if bin := which("omp"); bin != "" {
		p.Installed = true
		p.Binary = bin
		if v := versionOf(bin, "--version"); v != "" {
			p.Version = v
		}
	}
	return p, nil
}
