package tools

import "fmt"

const mongodbMCPVersion = "1.14.0"

func mongodbTool() Tool {
	return Tool{
		Key:       "mongodb",
		Label:     "MongoDB MCP Server",
		Summary:   "MongoDB y Atlas por MCP (host, npm; read-only por defecto)",
		Deploy:    DeployHost,
		DefaultOn: false,
		Install:   installMongoDBMCP,
		Upgrade:   installMongoDBMCP,
		Uninstall: uninstallMongoDBMCP,
		Status:    statusMongoDBMCP,
	}
}

func installMongoDBMCP(dry bool, log func(string)) error {
	if err := ensureNodeMin(22); err != nil {
		return err
	}
	pkg := "mongodb-mcp-server@" + mongodbMCPVersion
	if dry {
		log("$ npm install -g " + pkg)
		return nil
	}
	if err := runNpmGlobal("install", pkg); err != nil {
		return err
	}
	return exposeNpmGlobalBinary("mongodb-mcp-server")
}

func uninstallMongoDBMCP(dry bool, log func(string)) error {
	if dry {
		log("$ npm uninstall -g mongodb-mcp-server")
		return nil
	}
	if which("mongodb-mcp-server") == "" {
		_ = removeExposedNpmBinary("mongodb-mcp-server")
		log("mongodb-mcp-server no está instalado - nada que desinstalar")
		return nil
	}
	if err := runNpmGlobal("uninstall", "mongodb-mcp-server"); err != nil {
		return err
	}
	return removeExposedNpmBinary("mongodb-mcp-server")
}

func statusMongoDBMCP() (StatusPayload, error) {
	p := StatusPayload{}
	if bin := which("mongodb-mcp-server"); bin != "" {
		p.Installed = true
		p.Binary = bin
		p.Version = versionOf(bin, "--version")
	}
	if p.Installed && p.Version == "" {
		p.Version = fmt.Sprintf("mongodb-mcp-server %s", mongodbMCPVersion)
	}
	return p, nil
}
