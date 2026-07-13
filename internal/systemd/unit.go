package systemd

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"text/template"
)

// UnitConfig is the input to the systemd unit template.
type UnitConfig struct {
	// BinaryPath is the absolute path to the mcp-tools binary that
	// ExecStart invokes.
	BinaryPath string
	// Port is the listen port passed as --port.
	Port int
	// Bind is the listen address passed as --bind.
	Bind string
	// EnvFile is the path to the .env file loaded by EnvironmentFile.
	// Set to "-" prefix if missing (systemd tolerates a missing file).
	EnvFile string
	// User controls whether the unit lands in user or system mode.
	User bool
}

// unitTemplate is the deterministic mcp-tools-web.service template.
// ExecStart's binary path is double-quoted: systemd word-splits ExecStart,
// so an unquoted path with spaces ("/Users/Alice Smith/...") truncates to
// its first word (AUDIT-WEB-INSTALL INS-07). EnvironmentFile is NOT
// word-split by systemd, so it stays raw. RenderUnit validates both paths
// against quote/control characters, which would otherwise corrupt the
// unit syntax.
var unitTemplate = template.Must(template.New("mcp-tools-web.service").Parse(`[Unit]
Description=mcp-tools web admin panel
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
ExecStart="{{.BinaryPath}}" serve --port {{.Port}} --bind {{.Bind}}
EnvironmentFile=-{{.EnvFile}}
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy={{if .User}}default.target{{else}}multi-user.target{{end}}
`))

// RenderUnit returns the rendered unit text for cfg.
func RenderUnit(cfg UnitConfig) (string, error) {
	for name, p := range map[string]string{"BinaryPath": cfg.BinaryPath, "EnvFile": cfg.EnvFile} {
		if strings.ContainsAny(p, "\"\n") {
			return "", fmt.Errorf("systemd: %s %q contiene comillas o saltos de línea; no se puede escribir un unit válido", name, p)
		}
	}
	var b bytes.Buffer
	if err := unitTemplate.Execute(&b, cfg); err != nil {
		return "", fmt.Errorf("systemd: render unit: %w", err)
	}
	return b.String(), nil
}

// userHomeDir is split out so tests can stub it.
var userHomeDir = func() (string, error) {
	out, err := exec.Command("sh", "-c", "echo $HOME").Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(out)), nil
}
