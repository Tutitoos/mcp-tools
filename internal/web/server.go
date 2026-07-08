package web

import (

	"net/http"

	"github.com/Tutitoos/mcp-tools/webassets"
)

// SPAAssets is the embedded React Router v7 SPA bundle produced by
// `make build-web`. Always reference this var so the embed directive is
// not stripped by the compiler when the package isn't otherwise used.
var SPAAssets = webassets.WebAssets

// NewServer builds an http.Server whose handler is the chi router. The
// caller owns listener lifecycle (open + close) and calls srv.Serve(ln).
func NewServer() *http.Server {
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