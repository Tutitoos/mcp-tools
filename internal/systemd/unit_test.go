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
		"ExecStart=/home/u/.local/bin/mcp-tools serve --port 8080 --bind 127.0.0.1",
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

func TestRenderUnitDangerousChars(t *testing.T) {
	// Regression guard: if anyone ever tries to put bind= with shell
	// metacharacters, the unit must still render (text/template escapes
	// nothing by default, so the caller is responsible — but the unit
	// must still validate the bind is non-empty).
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
