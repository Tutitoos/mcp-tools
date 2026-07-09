package systemd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// portRe / bindRe find the first "--port <int>" / "--bind <host>" tokens
// inside ExecStart. They're lenient — systemd may have split the args
// across multiple lines (we render them on a single line, so this is
// mainly a safety net for hand-edited units).
var (
	portRe = regexp.MustCompile(`--port\s+(\d+)`)
	bindRe = regexp.MustCompile(`--bind\s+(\S+)`)
)

// parsePortFromUnit extracts the port from a rendered unit body.
func parsePortFromUnit(body string) int {
	m := portRe.FindStringSubmatch(body)
	if len(m) < 2 {
		return 0
	}
	p, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return p
}

// parseBindFromUnit extracts the bind address.
func parseBindFromUnit(body string) string {
	m := bindRe.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// readFile is a thin wrapper so tests can stub it.
var readFile = func(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// scanLines yields each non-empty line of s, trimmed. Helper for future
// tests / future parsing needs.
func scanLines(s string) []string {
	var out []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// unitRenderedFmt is exposed for tests that want to assert the rendered
// shape directly.
const unitRenderedFmt = `[Unit]
Description=mcp-tools web admin panel
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s serve --port %d --bind %s
EnvironmentFile=-%s
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=%s
`

// RenderedForTests is a debug-only helper that returns the exact rendered
// unit text for the given inputs. Production code uses RenderUnit.
func RenderedForTests(binPath string, port int, bind, envFile string, userMode bool) string {
	target := "multi-user.target"
	if userMode {
		target = "default.target"
	}
	return fmt.Sprintf(unitRenderedFmt, binPath, port, bind, envFile, target)
}
