package web

import (
	"net"
	"net/http"
)

// serveOn is the canonical "serve on this listener" helper. Wrapping the
// stdlib http.Server.Serve avoids touching the http package's internal
// state machine and keeps callers in this package in control of listener
// lifecycle (the cli owns Close).
func serveOn(srv *http.Server, ln net.Listener) error {
	return srv.Serve(ln)
}