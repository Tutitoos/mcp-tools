package cli

import (
	"runtime"
	"strings"
	"testing"
)

func TestTokensaveCapGuardMessageShape(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("guard only fires off-Linux; simulate via message-shape check")
	}
	err := runTokensaveCap(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "requires systemd") {
		t.Fatalf("expected systemd-guard error, got %v", err)
	}
}
