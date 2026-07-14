package config

import "os"

// DefaultBind is the listen address used when nothing else specifies one
// (CLI flag, existing systemd unit, or .env). Loopback by default: the
// panel's API is unauthenticated and mutating, and the same value is
// interpolated into the docker compose port publishes (ollama 11434,
// qdrant 6333), so an all-interfaces default exposed three unauthenticated
// surfaces to the LAN at once (AUDIT-2026-07-11 F1 / WEB-03).
//
// LAN exposure is a deliberate opt-in: `mcp-tools serve --bind 0.0.0.0`,
// `mcp-tools install --bind 0.0.0.0`, or MCP_TOOLS_BIND=0.0.0.0 in .env.
// Existing installs are untouched — the systemd unit and the generated
// .env keep whatever bind they already have.
const DefaultBind = "127.0.0.1"

// BindFromEnv returns the bind address configured via the environment:
// the MCP_TOOLS_BIND process variable wins, then the repo .env file.
// Empty string when neither sets one. This function is what makes the
// documented contract ("MCP_TOOLS_BIND=0.0.0.0 in .env") actually
// reach `install`, `serve`, and the `web` URL helpers.
func BindFromEnv() string {
	if v := os.Getenv("MCP_TOOLS_BIND"); v != "" {
		return v
	}
	if env, err := LoadEnv(EnvFile()); err == nil && env["MCP_TOOLS_BIND"] != "" {
		return env["MCP_TOOLS_BIND"]
	}
	return ""
}
