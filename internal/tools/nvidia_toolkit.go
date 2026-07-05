package tools

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func nvidiaToolkitTool() Tool {
	return Tool{
		Key:       "nvidia-toolkit",
		Label:     "NVIDIA Container Toolkit",
		Summary:   "GPU passthrough para Docker (requiere sudo, sólo si hay GPU NVIDIA)",
		Deploy:    DeploySudo,
		DefaultOn: hasNvidiaGPU(),
		Install:   installNvidiaToolkit,
		Upgrade: func(dry bool, log func(string)) error {
			// Upstream has no meaningful "upgrade" verb; documented in ADVANCED.md.
			// Re-run install to pick up new apt-repo state.
			return installNvidiaToolkit(dry, log)
		},
		Uninstall: uninstallNvidiaToolkit,
		Status:    statusNvidiaToolkit,
	}
}

func installNvidiaToolkit(dry bool, log func(string)) error {
	distroID, err := readDistroID()
	if err != nil {
		return err
	}
	if !supportedNvidiaDistro(distroID) {
		return fmt.Errorf("distro %q no soportada para nvidia-container-toolkit", distroID)
	}
	// The apt path (Debian/Ubuntu). RHEL/Fedora/Rocky/AlmaLinux would use dnf; we
	// only branch when we know the concrete host — no untested paths.
	steps := [][]string{
		// Import upstream signing key.
		{"sh", "-c", "curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --yes --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg"},
		{"sh", "-c", `curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list`},
		{"bash", "-c", `set -o pipefail; sudo apt-get update 2>&1 | { grep -v 'configured multiple times' || true; }`},
		{"bash", "-c", `set -o pipefail; sudo apt-get install -y nvidia-container-toolkit 2>&1 | { grep -v 'configured multiple times' || true; }`},
		{"sudo", "nvidia-ctk", "runtime", "configure", "--runtime=docker"},
		{"sudo", "systemctl", "restart", "docker"},
	}
	if dry {
		for _, s := range steps {
			log("$ " + strings.Join(s, " "))
		}
		return nil
	}
	for _, s := range steps {
		cmd := exec.Command(s[0], s[1:]...)
		// Inherit stdio for sudo password prompts and long-running apt output.
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", strings.Join(s, " "), err)
		}
	}
	return nil
}

func uninstallNvidiaToolkit(dry bool, log func(string)) error {
	distroID, err := readDistroID()
	if err != nil {
		return err
	}
	if !supportedNvidiaDistro(distroID) {
		return fmt.Errorf("distro %q no soportada para nvidia-container-toolkit", distroID)
	}
	steps := [][]string{
		{"sudo", "nvidia-ctk", "runtime", "configure", "--runtime=docker", "--unset"},
		{"sudo", "apt-get", "purge", "-y", "nvidia-container-toolkit"},
		{"sudo", "systemctl", "restart", "docker"},
	}
	if dry {
		for _, s := range steps {
			log("$ " + strings.Join(s, " "))
		}
		return nil
	}
	for _, s := range steps {
		cmd := exec.Command(s[0], s[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// nvidia-ctk unset may fail cleanly if the runtime isn't configured;
			// don't abort on that — apt-get purge is the load-bearing step.
			if strings.Contains(strings.Join(s, " "), "nvidia-ctk") {
				log(fmt.Sprintf("WARN %s: %v (continuando)", strings.Join(s, " "), err))
				continue
			}
			return fmt.Errorf("%s: %w", strings.Join(s, " "), err)
		}
	}
	return nil
}

func statusNvidiaToolkit() (StatusPayload, error) {
	p := StatusPayload{Extra: map[string]any{}}
	if bin := which("nvidia-ctk"); bin != "" {
		p.Installed = true
		p.Binary = bin
		if v := versionOf(bin, "--version"); v != "" {
			p.Version = v
		}
	}
	p.Extra["gpu_detected"] = hasNvidiaGPU()
	return p, nil
}

func readDistroID() (string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("/etc/os-release: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ID=") {
			continue
		}
		id := strings.TrimPrefix(line, "ID=")
		id = strings.Trim(id, `"'`)
		return strings.ToLower(id), nil
	}
	return "", errors.New("ID= no encontrado en /etc/os-release")
}

func supportedNvidiaDistro(id string) bool {
	switch id {
	case "debian", "ubuntu", "rhel", "fedora", "centos", "rocky", "almalinux":
		return true
	}
	return false
}

// captureCombined is a small helper used from other tool files.
func captureCombined(cmd *exec.Cmd) (string, error) {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return strings.TrimSpace(buf.String()), err
}
