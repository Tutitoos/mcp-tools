package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestPluginPkg(t *testing.T, root, name, content string) {
	t.Helper()
	dir := filepath.Join(root, "plugins", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeTestLockfile(t *testing.T, home, content string) {
	t.Helper()
	dir := filepath.Join(home, ".omp", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "omp-plugins.lock.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestAPIPluginsEndpointEmpty confirms GET /api/plugins returns 200 with a
// JSON `[]` (not `null`) for a workspace with no plugins dir.
func TestAPIPluginsEndpointEmpty(t *testing.T) {
	t.Setenv("MCP_TOOLS_ROOT", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/plugins", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("body = %q, want %q", got, "[]")
	}
}

// TestAPIPluginsEndpointOneUnlinked confirms a workspace plugin with no
// lockfile entry is reported linked=false, enabled=false.
func TestAPIPluginsEndpointOneUnlinked(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MCP_TOOLS_ROOT", root)
	t.Setenv("HOME", t.TempDir())
	writeTestPluginPkg(t, root, "foo", `{"name":"foo","version":"1.0.0"}`)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/plugins", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var rows []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1; body=%s", len(rows), rec.Body.String())
	}
	row := rows[0]
	if row["name"] != "foo" {
		t.Errorf("name = %v, want foo", row["name"])
	}
	if row["linked"] != false || row["enabled"] != false {
		t.Errorf("linked=%v enabled=%v, want both false", row["linked"], row["enabled"])
	}
	wantPath := filepath.Join(root, "plugins", "foo")
	if row["path"] != wantPath {
		t.Errorf("path = %v, want %v", row["path"], wantPath)
	}
}

// TestAPIPluginsEndpointLinkedEnabled confirms a plugin present (and
// enabled) in the lockfile is reported linked=true, enabled=true.
func TestAPIPluginsEndpointLinkedEnabled(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("MCP_TOOLS_ROOT", root)
	t.Setenv("HOME", home)
	writeTestPluginPkg(t, root, "foo", `{"name":"foo"}`)
	writeTestLockfile(t, home, `{"plugins":{"foo":{"enabled":true}}}`)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/plugins", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var rows []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1; body=%s", len(rows), rec.Body.String())
	}
	if rows[0]["linked"] != true || rows[0]["enabled"] != true {
		t.Errorf("linked=%v enabled=%v, want both true", rows[0]["linked"], rows[0]["enabled"])
	}
}

// TestAPIPluginActionNotFound confirms POST /api/plugins/{name}/link 404s
// when the workspace scan has no matching plugin.
func TestAPIPluginActionNotFound(t *testing.T) {
	t.Setenv("MCP_TOOLS_ROOT", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/plugins/nope/link", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPIPluginActionNoOmp confirms a 503 (with a message mentioning omp)
// when the `omp` binary isn't in PATH, for an otherwise-valid plugin.
func TestAPIPluginActionNoOmp(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MCP_TOOLS_ROOT", root)
	t.Setenv("HOME", t.TempDir())
	writeTestPluginPkg(t, root, "foo", `{"name":"foo"}`)
	t.Setenv("PATH", "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/plugins/foo/link", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "omp") {
		t.Errorf("body = %q, want mention of omp", rec.Body.String())
	}
}
