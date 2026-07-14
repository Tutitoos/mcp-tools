package tools

import (
	"fmt"
	"os"
	"os/exec"
)

const redisMCPVersion = "0.5.0"

func redisTool() Tool {
	return Tool{
		Key:       "redis",
		Label:     "Redis MCP Server",
		Summary:   "Datos, búsqueda y streams de Redis por MCP (host, uv)",
		Deploy:    DeployHost,
		DefaultOn: false,
		Install:   installRedisMCP,
		Upgrade:   installRedisMCP,
		Uninstall: uninstallRedisMCP,
		Status:    statusRedisMCP,
	}
}

func installRedisMCP(dry bool, log func(string)) error {
	home, err := hostHome()
	if err != nil {
		return err
	}
	if err := ensureUV(dry, log, home); err != nil {
		return err
	}
	pkg := "redis-mcp-server==" + redisMCPVersion
	if dry {
		log("$ uv tool install --force --python 3.14 " + pkg)
		return nil
	}
	cmd := exec.Command(resolveUV(home), "tool", "install", "--force", "--python", "3.14", pkg)
	cmd.Env = withLocalBinPath(os.Environ(), home)
	if err := runCombined(cmd, "uv tool install redis-mcp-server"); err != nil {
		return err
	}
	return nil
}

func uninstallRedisMCP(dry bool, log func(string)) error {
	if dry {
		log("$ uv tool uninstall redis-mcp-server")
		return nil
	}
	if which("redis-mcp-server") == "" {
		log("redis-mcp-server no está instalado - nada que desinstalar")
		return nil
	}
	home, err := hostHome()
	if err != nil {
		return err
	}
	cmd := exec.Command(resolveUV(home), "tool", "uninstall", "redis-mcp-server")
	cmd.Env = withLocalBinPath(os.Environ(), home)
	return runCombined(cmd, "uv tool uninstall redis-mcp-server")
}

func statusRedisMCP() (StatusPayload, error) {
	p := StatusPayload{}
	if bin := which("redis-mcp-server"); bin != "" {
		p.Installed = true
		p.Binary = bin
		p.Version = versionOf(bin, "--version")
	}
	if p.Installed && p.Version == "" {
		p.Version = fmt.Sprintf("redis-mcp-server %s", redisMCPVersion)
	}
	return p, nil
}
