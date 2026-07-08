package cli

// DefaultPort is the listen port used when no flag, env var, or
// persisted systemd unit specifies one. Picked to avoid collisions with
// the well-known dev-server port 8080 (which often hosts other tools).
//
// To change the default, update this single constant; install.go,
// serve.go, web.go, and open.go all reference it via the package-local
// symbol.
const DefaultPort = 8888

// DefaultBind is the listen address used when no flag or env var
// specifies one. 0.0.0.0 (all interfaces) is the project default so the
// panel is reachable from other devices on the LAN. The bearer token
// (~/.mcp-tools-web.token) is the security gate — every request must
// include `Authorization: Bearer <token>` once the token file exists.
//
// Use 127.0.0.1 explicitly via `--bind 127.0.0.1` (install) or
// `--bind 127.0.0.1` (serve) to opt back into loopback-only mode.
const DefaultBind = "0.0.0.0"