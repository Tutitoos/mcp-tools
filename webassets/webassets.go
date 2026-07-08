// Package webassets embeds the built SPA bundle produced by `make build-web`.
//
// `//go:embed` cannot traverse upward from the package directory, so the
// Makefile target `build-web` creates a symlink `webassets/build` that
// points at `../web/build` before the Go embed directive is compiled.
// During CI this happens in the same shell pipeline that runs `go build`.
package webassets

import "embed"

//go:embed all:build/client
var WebAssets embed.FS