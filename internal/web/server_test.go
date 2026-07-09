package web

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestServeOnLoopback boots the server on 127.0.0.1:0, hits /api/version
// (unauthenticated health probe) and / (SPA fallback), then shuts down.
// Equivalent to the integration check from step 5 of the plan.
func TestServeOnLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	srv := Listen()
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-errCh
	}()

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://" + addr + "/api/version")
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "version") {
		t.Errorf("body = %s, missing version", body)
	}

	resp, err = http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("get /: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(string(body), "<!doctype html>") {
		t.Errorf("body = %s, missing <!doctype html>", body)
	}
}

// TestServeAcceptsNonLoopback confirms a non-loopback RemoteAddr is no
// longer rejected at the IP layer. With no token file present, the
// request reaches /api/version and returns 200.
func TestServeAcceptsNonLoopback(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/version", nil)
	req.RemoteAddr = "8.8.8.8:1234"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("non-loopback status = %d, want 200", rec.Code)
	}
}
