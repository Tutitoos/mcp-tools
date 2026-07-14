package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runNpmGlobal runs `npm <verb> -g <pkg>` with stdio wired to the caller.
//
// It makes the DeploySudo contract honest (hallazgo INS-04, auditoría web-install 2026-07-11): the
// old closures declared "requiere sudo" but exec'd plain `npm`, so on a
// root-owned prefix (/usr/local) they died mid-install with EACCES — and
// from the web panel there is no TTY to answer a sudo prompt anyway. This
// helper never elevates; instead it pre-checks that the global prefix is
// writable and, when it isn't, fails BEFORE npm runs, with the two real
// options spelled out. Root (euid 0) skips the probe.
func runNpmGlobal(verb, pkg string) error {
	if os.Geteuid() != 0 {
		if dir := npmGlobalDir(); dir != "" {
			if err := writableDir(dir); err != nil {
				return fmt.Errorf("npm global prefix %s no es escribible sin root: corre `sudo npm %s -g %s` en una terminal, o configura un prefix de usuario (`npm config set prefix ~/.npm-global` + añadirlo al PATH) y reintenta", dir, verb, pkg)
			}
		}
	}
	cmd := exec.Command("npm", verb, "-g", pkg)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm %s -g %s: %w", verb, pkg, err)
	}
	return nil
}

// npmGlobalDir resolves the directory npm actually writes packages into
// (<prefix>/lib/node_modules). Empty string = could not resolve (npm will
// surface its own error on run).
func npmGlobalDir() string {
	out, err := exec.Command("npm", "prefix", "-g").Output()
	if err != nil {
		return ""
	}
	prefix := strings.TrimSpace(string(out))
	if prefix == "" {
		return ""
	}
	return filepath.Join(prefix, "lib", "node_modules")
}

// writableDir probes dir for write access by creating and removing a temp
// file. Walks up to the nearest existing parent first (a user prefix like
// ~/.npm-global/lib/node_modules may not exist until the first install).
func writableDir(dir string) error {
	for !dirExists(dir) {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	f, err := os.CreateTemp(dir, ".mcp-tools-write-probe-*")
	if err != nil {
		return err
	}
	f.Close()
	return os.Remove(f.Name())
}

func dirExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}
