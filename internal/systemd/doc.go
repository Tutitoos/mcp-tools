// Package systemd owns the unit template + writer for the
// `mcp-tools-web.service` systemd unit. The CLI's `install`, `stop`,
// `restart`, and `status-web` subcommands are the only callers; the web
// panel never invokes this package directly.
package systemd