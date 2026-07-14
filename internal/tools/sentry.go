package tools

const sentryMCPRemoteVersion = "0.1.38"

func sentryTool() Tool {
	return Tool{
		Key:       "sentry",
		Label:     "Sentry MCP",
		Summary:   "Sentry SaaS remoto con OAuth vía mcp-remote (host, npm)",
		Deploy:    DeployHost,
		DefaultOn: false,
		Install:   installSentryMCP,
		Upgrade:   installSentryMCP,
		Uninstall: uninstallSentryMCP,
		Status:    statusSentryMCP,
	}
}

func installSentryMCP(dry bool, log func(string)) error {
	if err := ensureNodeMin(18); err != nil {
		return err
	}
	pkg := "mcp-remote@" + sentryMCPRemoteVersion
	if dry {
		log("$ npm install -g " + pkg)
		return nil
	}
	if err := runNpmGlobal("install", pkg); err != nil {
		return err
	}
	return exposeNpmGlobalBinary("mcp-remote")
}

func uninstallSentryMCP(dry bool, log func(string)) error {
	if dry {
		log("$ npm uninstall -g mcp-remote")
		return nil
	}
	if which("mcp-remote") == "" {
		_ = removeExposedNpmBinary("mcp-remote")
		log("mcp-remote no está instalado - nada que desinstalar")
		return nil
	}
	if err := runNpmGlobal("uninstall", "mcp-remote"); err != nil {
		return err
	}
	return removeExposedNpmBinary("mcp-remote")
}

func statusSentryMCP() (StatusPayload, error) {
	p := StatusPayload{}
	if bin := which("mcp-remote"); bin != "" {
		p.Installed = true
		p.Binary = bin
		p.Version = versionOf(bin, "--version")
		if p.Version == "" {
			p.Version = "mcp-remote " + sentryMCPRemoteVersion
		}
	}
	return p, nil
}
