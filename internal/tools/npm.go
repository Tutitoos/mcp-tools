package tools

import (
	"errors"
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

// exposeNpmGlobalBinary keeps nvm-owned global installs reachable from
// systemd and MCP clients, whose PATH deliberately does not include a
// version-specific ~/.nvm directory.
func exposeNpmGlobalBinary(name string) error {
	globalDir := npmGlobalDir()
	if globalDir == "" {
		return errors.New("no se pudo resolver el prefix global de npm")
	}
	src := filepath.Join(filepath.Dir(filepath.Dir(globalDir)), "bin", name)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("binario npm %s no encontrado: %w", src, err)
	}
	home, err := hostHome()
	if err != nil {
		return err
	}
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(binDir, name)
	if info, err := os.Lstat(dst); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("%s ya existe y no es un symlink; no se sobrescribe", dst)
		}
		if err := os.Remove(dst); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Symlink(src, dst)
}

func removeExposedNpmBinary(name string) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".local", "bin", name)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	return os.Remove(path)
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
