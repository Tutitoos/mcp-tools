package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func rtkTool() Tool {
	return Tool{
		Key:       "rtk",
		Label:     "RTK",
		Summary:   "Hook shell para OMP + Claude Code (60–90% ahorro de tokens)",
		Deploy:    DeployHost,
		DefaultOn: true,
		Install: func(dry bool, log func(string)) error {
			return installRTK(dry, log, false)
		},
		Upgrade: func(dry bool, log func(string)) error {
			return installRTK(dry, log, true)
		},
		Uninstall: uninstallRTK,
		Status:    statusRTK,
	}
}

// TODO(security): pin to a stable upstream tag. The branch below can be
// force-pushed. See docs/REVIEW.md (H5) for guidance.
const (
	rtkGitURL    = "https://github.com/makoMakoGo/rtk.git"
	rtkGitBranch = "feat/omp-extension-rewrite"
)

func installRTK(dry bool, log func(string), force bool) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if err := ensureCargo(dry, log, home); err != nil {
		return err
	}
	args := []string{"install", "--git", rtkGitURL, "--branch", rtkGitBranch, "--locked"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, "rtk")
	if dry {
		log("$ cargo " + strings.Join(args, " "))
	} else {
		cmd := exec.Command(cargoBin(home), args...)
		cmd.Env = withCargoPath(os.Environ(), home)
		if err := runCombined(cmd, "cargo install rtk"); err != nil {
			return err
		}
	}
	agents := []string{"omp"}
	if _, err := exec.LookPath("claude"); err == nil {
		agents = append(agents, "claude")
	}
	for _, agent := range agents {
		if dry {
			log(fmt.Sprintf("$ rtk init --agent %s --auto-patch", agent))
			continue
		}
		cmd := exec.Command(rtkBin(home), "init", "--agent", agent, "--auto-patch")
		cmd.Env = withCargoPath(os.Environ(), home)
		cmd.Env = append(cmd.Env, "HOME="+home)
		if err := runCombined(cmd, "rtk init --agent "+agent); err != nil {
			return err
		}
	}
	return nil
}

func uninstallRTK(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	ompHook := filepath.Join(home, ".omp/extensions/rtk.ts")
	if dry {
		log("$ cargo uninstall rtk")
		log("$ rm -f " + ompHook)
		log("NOTE: edita ~/.claude/settings.json a mano para quitar 'rtk hook claude' de PreToolUse")
		return nil
	}
	if bin, err := exec.LookPath("cargo"); err == nil {
		_ = exec.Command(bin, "uninstall", "rtk").Run()
	} else if _, err := os.Stat(cargoBin(home)); err == nil {
		_ = exec.Command(cargoBin(home), "uninstall", "rtk").Run()
	}
	_ = os.Remove(ompHook)
	log("NOTE: edita ~/.claude/settings.json a mano para quitar 'rtk hook claude' de PreToolUse")
	return nil
}

func statusRTK() (StatusPayload, error) {
	home, err := hostHome()
	if err != nil {
		return StatusPayload{}, err
	}
	p := StatusPayload{Extra: map[string]any{}}
	hooked := []string{}
	if bin := lookRTKBinary(home); bin != "" {
		p.Installed = true
		p.Binary = bin
		if v := versionOf(bin, "--version"); v != "" {
			p.Version = v
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".omp/extensions/rtk.ts")); err == nil {
		hooked = append(hooked, "omp")
	}
	if data, err := os.ReadFile(filepath.Join(home, ".claude/settings.json")); err == nil {
		var settings map[string]any
		if err := json.Unmarshal(data, &settings); err == nil && containsRTKClaudeHook(settings) {
			hooked = append(hooked, "claude")
		}
	}
	p.Extra["hooked_agents"] = hooked
	return p, nil
}

func lookRTKBinary(home string) string {
	if p, err := exec.LookPath("rtk"); err == nil && !strings.Contains(p, ".local/bin/rtk") {
		return p
	}
	if _, err := os.Stat(rtkBin(home)); err == nil {
		return rtkBin(home)
	}
	if p, err := exec.LookPath("rtk"); err == nil {
		return p
	}
	return ""
}
