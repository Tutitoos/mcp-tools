package tools

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallDockerMCPArchive(t *testing.T) {
	var bundle bytes.Buffer
	gz := gzip.NewWriter(&bundle)
	tw := tar.NewWriter(gz)
	payload := []byte("docker-mcp-binary")
	if err := tw.WriteHeader(&tar.Header{Name: "docker-mcp", Mode: 0o755, Size: int64(len(payload))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "cli-plugins", "docker-mcp")
	if err := installDockerMCPArchive(bundle.Bytes(), path); err != nil {
		t.Fatalf("installDockerMCPArchive: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("installed payload = %q, want %q", got, payload)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("installed plugin is not executable: %v", info.Mode())
	}
}
