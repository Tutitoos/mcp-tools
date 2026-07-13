package systemd

import (
	"strings"
	"testing"
)

func TestRenderUnitUser(t *testing.T) {
	out, err := RenderUnit(UnitConfig{
		BinaryPath: "/home/u/.local/bin/mcp-tools",
		Port:       8080,
		Bind:       "127.0.0.1",
		EnvFile:    "/home/u/mcp-tools/.env",
		User:       true,
	})
	if err != nil {
		t.Fatalf("RenderUnit: %v", err)
	}
	must := []string{
		"[Unit]",
		"Description=mcp-tools web admin panel",
		"After=network-online.target docker.service",
		"[Service]",
		"Type=simple",
		`ExecStart="/home/u/.local/bin/mcp-tools" serve --port 8080 --bind 127.0.0.1`,
		"EnvironmentFile=-/home/u/mcp-tools/.env",
		"Restart=on-failure",
		"RestartSec=5",
		"[Install]",
		"WantedBy=default.target",
	}
	for _, s := range must {
		if !strings.Contains(out, s) {
			t.Errorf("rendered unit missing %q\n---\n%s", s, out)
		}
	}
}

func TestRenderUnitSystem(t *testing.T) {
	out, err := RenderUnit(UnitConfig{
		BinaryPath: "/usr/local/bin/mcp-tools",
		Port:       9000,
		Bind:       "0.0.0.0",
		EnvFile:    "/etc/mcp-tools.env",
		User:       false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "WantedBy=multi-user.target") {
		t.Errorf("system mode unit should target multi-user.target\n%s", out)
	}
	if !strings.Contains(out, "--port 9000 --bind 0.0.0.0") {
		t.Errorf("port/bind not threaded through\n%s", out)
	}
}

// TestRenderUnitSpacesAndQuotes pins the INS-07 contract: paths with spaces
// must survive systemd's ExecStart word-splitting (via quoting), and paths
// that would break the quoting/unit syntax must be rejected, not written.
func TestRenderUnitSpacesAndQuotes(t *testing.T) {
	out, err := RenderUnit(UnitConfig{
		BinaryPath: "/Users/Alice Smith/.local/bin/mcp-tools",
		Port:       8888,
		Bind:       "127.0.0.1",
		EnvFile:    "/Users/Alice Smith/mcp-tools/.env",
		User:       true,
	})
	if err != nil {
		t.Fatalf("RenderUnit: %v", err)
	}
	if !strings.Contains(out, `ExecStart="/Users/Alice Smith/.local/bin/mcp-tools" serve`) {
		t.Errorf("binary path with spaces must be quoted\n%s", out)
	}
	// EnvironmentFile is not word-split by systemd; raw spaces are valid.
	if !strings.Contains(out, "EnvironmentFile=-/Users/Alice Smith/mcp-tools/.env") {
		t.Errorf("env file path mangled\n%s", out)
	}

	for _, bad := range []UnitConfig{
		{BinaryPath: `/tmp/evil"quote/mcp-tools`, Port: 1, Bind: "127.0.0.1", EnvFile: "/tmp/.env"},
		{BinaryPath: "/tmp/mcp-tools", Port: 1, Bind: "127.0.0.1", EnvFile: "/tmp/evil\ninjected/.env"},
	} {
		if _, err := RenderUnit(bad); err == nil {
			t.Errorf("RenderUnit(%q, %q) should reject quote/newline", bad.BinaryPath, bad.EnvFile)
		}
	}
}

func TestRenderUnitEmptyBind(t *testing.T) {
	// text/template escapes nothing by default; the unit must still render
	// with an empty bind (validation happens at the caller).
	_, err := RenderUnit(UnitConfig{
		BinaryPath: "/bin/mcp-tools",
		Port:       1,
		Bind:       "",
		EnvFile:    "/etc/mcp-tools.env",
		User:       false,
	})
	if err != nil {
		t.Errorf("empty bind should still render, got %v", err)
	}
}
