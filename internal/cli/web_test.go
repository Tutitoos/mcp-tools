package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tutitoos/mcp-tools/internal/systemd"
)

// TestWebFlagCombinations confirms the mutually-exclusive flag checks.
func TestWebFlagCombinations(t *testing.T) {
	cases := []struct {
		name    string
		enable  bool
		disable bool
		port    int
		status  bool
		noOpen  bool
		wantOK  bool
	}{
		{"no flags", false, false, 0, false, false, true},
		{"enable alone", true, false, 0, false, false, true},
		{"disable alone", false, true, 0, false, false, true},
		{"set-port alone", false, false, 9090, false, false, true},
		{"status alone", false, false, 0, true, false, true},
		{"enable+disable", true, true, 0, false, false, false},
		{"enable+set-port", true, false, 9090, false, false, false},
		{"disable+set-port", false, true, 9090, false, false, false},
		{"status+enable", true, false, 0, true, false, false},
		{"status+disable", false, true, 0, true, false, false},
		{"status+set-port", false, false, 9090, true, false, false},
		{"no-open alone", false, false, 0, false, true, true},
		{"no-open+enable", true, false, 0, false, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oldEnable, oldDisable, oldPort, oldStatus, oldNoOpen := webEnable, webDisable, webSetPort, webShowStatus, webNoOpen
			webEnable = tc.enable
			webDisable = tc.disable
			webSetPort = tc.port
			webShowStatus = tc.status
			webNoOpen = tc.noOpen
			defer func() {
				webEnable = oldEnable
				webDisable = oldDisable
				webSetPort = oldPort
				webShowStatus = oldStatus
				webNoOpen = oldNoOpen
			}()
			err := validateWebFlags()
			ok := err == nil
			if ok != tc.wantOK {
				t.Errorf("validateWebFlags() ok=%v, want %v (err=%v)", ok, tc.wantOK, err)
			}
		})
	}
}

// TestValidatePort confirms the port range check.
func TestValidatePort(t *testing.T) {
	for _, p := range []int{1, 80, 8080, 65535} {
		if err := validatePort(p); err != nil {
			t.Errorf("port %d should be valid, got %v", p, err)
		}
	}
	for _, p := range []int{0, -1, 65536, 100000} {
		if err := validatePort(p); err == nil {
			t.Errorf("port %d should be invalid", p)
		}
	}
}

// TestWebURLBuild confirms the loopback URL is built with proper IPv6
// brackets via net.JoinHostPort (no fmt.Sprintf %s:%d).
func TestWebURLBuild(t *testing.T) {
	cases := []struct {
		bind string
		port int
		want string
	}{
		{"127.0.0.1", 8080, "http://127.0.0.1:8080/"},
		{"::1", 8080, "http://[::1]:8080/"},
		{"0.0.0.0", 80, "http://0.0.0.0:80/"},
	}
	for _, tc := range cases {
		t.Run(tc.bind, func(t *testing.T) {
			got := webURL(tc.bind, tc.port)
			if got != tc.want {
				t.Errorf("webURL(%s, %d) = %q, want %q", tc.bind, tc.port, got, tc.want)
			}
		})
	}
}

// TestHasBrowserLauncher confirms the PATH probe used to short-circuit
// the browser attempt on headless hosts.
func TestHasBrowserLauncher(t *testing.T) {
	dir := t.TempDir()
	fakeXdgOpen := filepath.Join(dir, "xdg-open")
	if err := os.WriteFile(fakeXdgOpen, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake xdg-open: %v", err)
	}
	t.Setenv("PATH", dir)
	if !hasBrowserLauncher() {
		t.Error("hasBrowserLauncher() = false, want true with xdg-open on PATH")
	}

	t.Setenv("PATH", t.TempDir())
	if hasBrowserLauncher() {
		t.Error("hasBrowserLauncher() = true, want false with no launcher on PATH")
	}
}

// TestRunWebOpenHeadlessNoError confirms runWebOpen degrades to an
// informational, exit-0 print (never an error) when there is no
// browser launcher on PATH.
func TestRunWebOpenHeadlessNoError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	oldNoOpen := webNoOpen
	webNoOpen = false
	defer func() { webNoOpen = oldNoOpen }()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWebOpen(systemd.ModeNone)

	w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	out := string(data)

	if err != nil {
		t.Fatalf("runWebOpen() error = %v, want nil", err)
	}
	if !strings.Contains(out, "url:") {
		t.Errorf("stdout missing %q, got %q", "url:", out)
	}
	if !strings.Contains(out, "no instalado") && !strings.Contains(out, "foreground") {
		t.Errorf("stdout missing service hint, got %q", out)
	}
}

// keep systemd import alive; the test exercises types this package uses.
var _ = systemd.ModeUser
