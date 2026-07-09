package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandleVersion confirms /api/version returns 200 with the build
// metadata keys populated.
func TestHandleVersion(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/version", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	handleVersion(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, k := range []string{"version", "commit", "date"} {
		if _, ok := body[k]; !ok {
			t.Errorf("missing key %q in /api/version response", k)
		}
	}
}

// TestAPIToolsEndpoint hits /api/tools and asserts the response contains
// the canonical claude, ollama, and qdrant keys.
func TestAPIToolsEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/tools", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var rows []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	wantKeys := map[string]bool{"claude": false, "ollama": false, "qdrant": false}
	for _, row := range rows {
		if k, ok := row["key"].(string); ok {
			if _, expected := wantKeys[k]; expected {
				wantKeys[k] = true
			}
		}
	}
	for k, found := range wantKeys {
		if !found {
			t.Errorf("missing %q in /api/tools response", k)
		}
	}
}

// TestAPIStatusEndpoint confirms /api/status returns a JSON envelope with
// the expected keys (even when state.json is empty and docker is missing).
func TestAPIStatusEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/status", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, k := range []string{"state", "env", "env_mem0", "compose_services", "docker_running"} {
		if _, ok := body[k]; !ok {
			t.Errorf("missing key %q in /api/status response", k)
		}
	}
}

// TestRouterAcceptsNonLoopback confirms the router doesn't filter by
// source IP. The bind address (0.0.0.0 vs 127.0.0.1) controls reach,
// the router itself just serves requests.
func TestRouterAcceptsNonLoopback(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/version", nil)
	req.RemoteAddr = "8.8.8.8:80"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("non-loopback status = %d, want 200", rec.Code)
	}
}

// TestSPAFallbackReturnsIndex confirms that a request for an unknown route
// (e.g. /dashboard) returns the embedded index.html with the SPA shell.
func TestSPAFallbackReturnsIndex(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some-spa-route", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Errorf("body missing <!doctype html>; got %q", rec.Body.String())
	}
}

// TestAPINotFound confirms that unknown /api/* routes get 404 (not the
// SPA fallback).
func TestAPINotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/does-not-exist", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// TestAPILogsStreamRejectsBadService confirms /api/logs/{service} rejects
// service keys with shell metacharacters before ever invoking docker.
func TestAPILogsStreamRejectsBadService(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/logs/foo;rm", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid service name") {
		t.Errorf("body = %q, want it to contain %q", rec.Body.String(), "invalid service name")
	}
}
