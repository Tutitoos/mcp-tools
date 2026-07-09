// Package webassets embeds the built SPA + SSR bundles produced by
// `make build-web`.
//
// `//go:embed` cannot follow symlinks and cannot traverse upward from the
// package directory, so the Makefile target `webassets/build` copies
// `web/build/` into `webassets/build/` (removing any prior copy first) so
// the embed directive picks up the latest SPA + SSR bundles:
//
//	build/client/index.html              -- SPA shell (also fallback)
//	build/client/assets/entry.client.js  -- hydrated client bundle
//	build/server/index.js                -- SSR bundle (Node ESM)
package webassets

import "embed"

//go:embed all:build
var WebAssets embed.FS
