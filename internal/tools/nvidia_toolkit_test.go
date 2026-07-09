package tools

import "testing"

// TestSupportedNvidiaDistro verifies G2: nvidia-toolkit only claims support
// for distros installNvidiaToolkit actually has an apt-based code path for
// (Debian/Ubuntu). RHEL-family distros used to be accepted here and then
// abort mid-install on "apt-get: command not found" — this locks the
// narrowed surface in place.
func TestSupportedNvidiaDistro(t *testing.T) {
	tests := map[string]bool{
		"ubuntu":    true,
		"debian":    true,
		"fedora":    false,
		"rhel":      false,
		"centos":    false,
		"rocky":     false,
		"almalinux": false,
		"":          false,
	}
	for id, want := range tests {
		if got := supportedNvidiaDistro(id); got != want {
			t.Errorf("supportedNvidiaDistro(%q) = %v, want %v", id, got, want)
		}
	}
}
