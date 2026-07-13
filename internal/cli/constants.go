package cli

import "github.com/Tutitoos/mcp-tools/internal/config"

// DefaultPort is the listen port used when no flag, env var, or
// persisted systemd unit specifies one. Picked to avoid collisions with
// the well-known dev-server port 8080 (which often hosts other tools).
//
// To change the default, update this single constant; install.go,
// serve.go, web.go, and open.go all reference it via the package-local
// symbol.
const DefaultPort = 8888

// DefaultBind mirrors config.DefaultBind (loopback). LAN exposure is
// opt-in via `--bind 0.0.0.0` or MCP_TOOLS_BIND — see internal/config/bind.go
// for the rationale.
const DefaultBind = config.DefaultBind
