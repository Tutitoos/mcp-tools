package cli

// DefaultPort is the listen port used when no flag, env var, or
// persisted systemd unit specifies one. Picked to avoid collisions with
// the well-known dev-server port 8080 (which often hosts other tools).
//
// To change the default, update this single constant; install.go,
// serve.go, web.go, and open.go all reference it via the package-local
// symbol.
const DefaultPort = 8888