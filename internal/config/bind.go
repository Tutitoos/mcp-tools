package config

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
