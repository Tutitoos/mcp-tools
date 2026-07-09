package web

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Tutitoos/mcp-tools/webassets"
)

// SPAAssets is the embedded React Router v7 SPA bundle produced by
// `make build-web`. Always reference this var so the embed directive is
// not stripped by the compiler when the package isn't otherwise used.
var SPAAssets = webassets.WebAssets

// NewServer builds an http.Server whose handler is the chi router. The
// caller owns listener lifecycle (open + close) and calls srv.Serve(ln).
// Initialises the SSR engine; if node is missing or the bundle wasn't
// built, the server boots in SPA-only mode (InitSSR logs the reason and
// returns nil engine).
func NewServer() *http.Server {
	if err := InitSSR(SPAAssets); err != nil {
		fmt.Fprintf(os.Stderr, "ssr: disabled (%v); serving SPA fallback\n", err)
	}
	return &http.Server{
		Handler:           NewRouter(),
		ReadHeaderTimeout: 5 * 1_000_000_000, // 5s
	}
}

// Listen is a convenience that builds a server with sensible timeouts for
// long-lived SSE streams: ReadHeaderTimeout=5s, ReadTimeout=30s,
// WriteTimeout=0 (no write timeout — SSE handlers can stream forever).
func Listen() *http.Server {
	srv := NewServer()
	srv.WriteTimeout = 0
	return srv
}