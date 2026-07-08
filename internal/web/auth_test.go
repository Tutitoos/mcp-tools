package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocalOnlyMiddleware covers the loopback enforcement. Unix-socket
// listeners (localOnlySkip=true) should pass through; TCP listeners
// should reject non-loopback.
func TestLocalOnlyMiddleware(t *testing.T) {
	handler := localOnly(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name       string
		skip       bool
		remote     string
		wantStatus int
	}{
		{"loopback v4", false, "127.0.0.1:1234", http.StatusOK},
		{"loopback v6", false, "[::1]:8080", http.StatusOK},
		{"public v4 rejected", false, "1.2.3.4:5678", http.StatusForbidden},
		{"private v4 rejected", false, "192.168.1.1:80", http.StatusForbidden},
		{"unix socket", false, "@mcp-tools-web", http.StatusOK},
		{"unix socket path", false, "/var/run/mcp.sock", http.StatusOK},
		{"skip enabled allows public", true, "1.2.3.4:99", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			SetLocalOnlySkip(tc.skip)
			defer SetLocalOnlySkip(false)
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tc.remote
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

// TestHandleAuthTokenMismatch covers the bearer-token path: when a token
// file exists, missing or wrong Authorization headers must yield 401.
func TestHandleAuthTokenMismatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	tokenPath := filepath.Join(home, ".mcp-tools-web.token")
	if err := os.WriteFile(tokenPath, []byte("supersecret-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	handler := handleAuth(true, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name       string
		auth       string
		wantStatus int
	}{
		{"no auth header", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic supersecret-token", http.StatusUnauthorized},
		{"wrong token", "Bearer wrong", http.StatusUnauthorized},
		{"correct token", "Bearer supersecret-token", http.StatusOK},
		{"correct token with extra whitespace", "Bearer    supersecret-token   ", http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

// TestHandleAuthDevMode confirms that with no token file, requests are
// allowed without an Authorization header.
func TestHandleAuthDevMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	handler := handleAuth(true, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("dev-mode status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestWriteJSON sets the content type to application/json.
func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusCreated, map[string]int{"x": 1})
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json…", ct)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"x":1`) {
		t.Errorf("body = %q, missing x:1", rec.Body.String())
	}
}

// TestRecovererMiddleware catches a panic and returns 500.
func TestRecovererMiddleware(t *testing.T) {
	handler := recoverer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}