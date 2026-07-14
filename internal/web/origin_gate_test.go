package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCrossOriginGate covers the CSRF surface from F1/WEB-03 (auditoría 2026-07-11):
// a malicious page in the user's browser could fire POSTs at the panel
// (loopback or LAN) without reading the response. The gate must reject
// browser-marked cross-site mutations while keeping curl/CLI (no Origin)
// and the same-origin SPA working, and must never block reads (SSE streams).
func TestCrossOriginGate(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := crossOriginGate(ok)

	cases := []struct {
		name    string
		method  string
		headers map[string]string
		want    int
	}{
		{"curl POST sin Origin", http.MethodPost, nil, http.StatusNoContent},
		{"SPA same-origin", http.MethodPost, map[string]string{"Origin": "http://example.com:8888"}, http.StatusNoContent},
		{"same-origin case-insensitive", http.MethodPost, map[string]string{"Origin": "http://EXAMPLE.com:8888"}, http.StatusNoContent},
		{"cross-origin Origin", http.MethodPost, map[string]string{"Origin": "http://evil.test"}, http.StatusForbidden},
		{"Sec-Fetch-Site cross-site", http.MethodPost, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"opaque origin null", http.MethodPost, map[string]string{"Origin": "null"}, http.StatusForbidden},
		{"origin no parseable", http.MethodPost, map[string]string{"Origin": "::not-a-url"}, http.StatusForbidden},
		{"GET cross-site pasa (read-only + SSE)", http.MethodGet, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusNoContent},
		{"same-site (subdominio) pasa por Origin match rule", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-site", "Origin": "http://example.com:8888"}, http.StatusNoContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://example.com:8888/api/env", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Errorf("%s: got %d, want %d", tc.name, rec.Code, tc.want)
			}
		})
	}
}

// TestRedactEnv pins the /api/status redaction contract: secret-shaped keys
// keep their name but lose their value; everything else passes through.
func TestRedactEnv(t *testing.T) {
	got := redactEnv(map[string]string{
		"MEM0_USER_ID":              "tutitoos",
		"SOME_API_KEY":              "abc123",
		"AUTH_TOKEN":                "t0k3n",
		"DB_PASSWORD":               "hunter2",
		"CLIENT_SECRET":             "sssh",
		"REDIS_PWD":                 "redis-secret",
		"MDB_MCP_CONNECTION_STRING": "mongodb://user:password@host/db",
		"EMPTY_KEY":                 "",
		"MCP_TOOLS_BIND":            "127.0.0.1",
	})
	for _, k := range []string{"SOME_API_KEY", "AUTH_TOKEN", "DB_PASSWORD", "CLIENT_SECRET", "REDIS_PWD", "MDB_MCP_CONNECTION_STRING"} {
		if got[k] != "••••••••" {
			t.Errorf("%s = %q, want redacted", k, got[k])
		}
	}
	if got["MEM0_USER_ID"] != "tutitoos" || got["MCP_TOOLS_BIND"] != "127.0.0.1" {
		t.Errorf("non-secret values must pass through: %v", got)
	}
	if got["EMPTY_KEY"] != "" {
		t.Errorf("empty values stay empty (nothing to hide): %q", got["EMPTY_KEY"])
	}
}

func TestUpdateEnvHandlerPreservesMaskedSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("MDB_MCP_API_CLIENT_SECRET=real-secret\nREDIS_HOST=old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/env", strings.NewReader(`{"values":{"MDB_MCP_API_CLIENT_SECRET":"••••••••","REDIS_HOST":"new"}}`))
	rec := httptest.NewRecorder()
	updateEnvHandler(path, rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("updateEnvHandler status = %d, body = %s", rec.Code, rec.Body.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "MDB_MCP_API_CLIENT_SECRET=real-secret") || !strings.Contains(got, "REDIS_HOST=new") {
		t.Fatalf("masked secret was overwritten or regular value not updated:\n%s", got)
	}
}
