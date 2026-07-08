package web

import (
	"bufio"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

// localOnly is middleware that rejects non-loopback / non-unix connections
// with HTTP 403. The systemd unit (and any developer) binds the server to
// 127.0.0.1 by default; the loopback check is the last line of defence
// against accidentally exposing the API.
//
// Unix-socket listeners set localOnlySkip to true so the middleware is a
// no-op for socket-bound servers. The flag is stored as an atomic int32
// on the package for tests that swap listeners at runtime.
var localOnlySkip atomic.Bool

// SetLocalOnlySkip toggles the loopback check. Pass true for unix-socket
// listeners; false for TCP.
func SetLocalOnlySkip(skip bool) { localOnlySkip.Store(skip) }

func localOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if localOnlySkip.Load() {
			next.ServeHTTP(w, r)
			return
		}
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// unix sockets: RemoteAddr is "@" or "/var/run/..."; treat as ok.
			if strings.HasPrefix(r.RemoteAddr, "@") || strings.HasPrefix(r.RemoteAddr, "/") {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "forbidden: bad remote addr", http.StatusForbidden)
			return
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			slog.Warn("web: rejecting non-loopback request", "remote", r.RemoteAddr, "path", r.URL.Path)
			http.Error(w, "forbidden: loopback only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// recoverer wraps panic-protected handlers in the style of chi's middleware
// but without depending on the chi-specific logger.
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("web: panic recovered", "err", rec, "path", r.URL.Path)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// requestLogger emits a single structured log line per request.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("web: request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// tokenPath is the on-disk location of the bearer token. `mcp-tools install`
// generates the token and writes the file with 0o600. A missing file means
// "dev mode": no Authorization header is required.
func tokenPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mcp-tools-web.token"), nil
}

// readToken returns the configured bearer token, or "" if no token file is
// present (dev mode). The token is trimmed of whitespace to tolerate shells
// that append a trailing newline.
func readToken() (string, error) {
	p, err := tokenPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// handleAuth wraps a handler with bearer-token verification when a token
// file exists. `required` is currently informational — the presence of a
// token file is what flips the check on/off.
func handleAuth(_ bool, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := readToken()
		if err != nil {
			http.Error(w, "internal error: token read", http.StatusInternalServerError)
			return
		}
		if token == "" {
			// No token configured → dev mode, no auth required.
			h(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="mcp-tools"`)
			http.Error(w, "unauthorized: missing bearer token", http.StatusUnauthorized)
			return
		}
		got := strings.TrimPrefix(auth, prefix)
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "unauthorized: token mismatch", http.StatusUnauthorized)
			return
		}
		h(w, r)
	}
}

// writeJSON serialises v as JSON with the supplied status. Centralised so
// handlers don't accidentally set wrong content types or forget encoding.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := jsonEncoder(w)
	_ = enc.Encode(v)
}

// writeError responds with a {"error":"..."} JSON payload.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// jsonEncoder is a tiny indirection so tests can swap it for a buffer if
// needed. Returns a fresh encoder each call.
func jsonEncoder(w http.ResponseWriter) *encoder {
	return &encoder{w: bufio.NewWriter(w)}
}

type encoder struct{ w *bufio.Writer }

func (e *encoder) Encode(v any) error {
	defer e.w.Flush()
	// Cheap JSON encoder to avoid pulling encoding/json into hot paths
	// when callers want streaming; the underlying writer is still buffered.
	return encodeJSON(e.w, v)
}

// encodeJSON is the package-private encoding helper. We re-export from
// internal/web/json.go.
func encodeJSON(w *bufio.Writer, v any) error {
	return jsonEncode(w, v)
}

// suppress unused-import warning when callers don't pull bufio directly.
var _ = fmt.Sprintf