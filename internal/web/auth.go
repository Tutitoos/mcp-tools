package web

import (
	"bufio"
	"fmt"
	"log/slog"
	"net/http"
)

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

// writeJSON serialises v as JSON with the supplied status. Centralised so
// handlers don't accidentally set wrong content types or forget encoding.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := jsonEncoder(w)
	_ = enc.Encode(v)
	_ = enc.w.Flush()
}
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
	return encodeJSON(e.w, v)
}

// encodeJSON is the package-private encoding helper. We re-export from
// internal/web/json.go.
func encodeJSON(w *bufio.Writer, v any) error {
	return jsonEncode(w, v)
}

// suppress unused-import warning when callers don't pull fmt directly.
var _ = fmt.Sprintf