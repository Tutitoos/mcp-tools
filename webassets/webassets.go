// Package webassets embeds the built SPA bundle produced by `make build-web`.
//
// `//go:embed` cannot follow symlinks and cannot traverse upward from the
// package directory, so the Makefile target `webassets/build` copies
// `web/build/` into `webassets/build/` (removing any prior copy first) so
// the embed directive picks up the latest SPA bundle.
package webassets

import "embed"

//go:embed all:build/client
var WebAssets embed.FS