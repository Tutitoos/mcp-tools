package cli

import (
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
		wantOK  bool
	}{
		{"no flags", false, false, 0, false, true},
		{"enable alone", true, false, 0, false, true},
		{"disable alone", false, true, 0, false, true},
		{"set-port alone", false, false, 9090, false, true},
		{"status alone", false, false, 0, true, true},
		{"enable+disable", true, true, 0, false, false},
		{"enable+set-port", true, false, 9090, false, false},
		{"disable+set-port", false, true, 9090, false, false},
		{"status+enable", true, false, 0, true, false},
		{"status+disable", false, true, 0, true, false},
		{"status+set-port", false, false, 9090, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oldEnable, oldDisable, oldPort, oldStatus := webEnable, webDisable, webSetPort, webShowStatus
			webEnable = tc.enable
			webDisable = tc.disable
			webSetPort = tc.port
			webShowStatus = tc.status
			defer func() {
				webEnable = oldEnable
				webDisable = oldDisable
				webSetPort = oldPort
				webShowStatus = oldStatus
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

// keep systemd import alive; the test exercises types this package uses.
var _ = systemd.ModeUser