// Package webassets embeds the built SPA bundle produced by `make build-web`.
// The path is relative to repo root because the React Router build writes to
// `web/build/client/` (configured in web/vite.config.ts).
//
// During step 1 the directory may be empty (only present when the SPA has
// been built at least once); the embed directive tolerates a missing tree
// at compile time as long as the *containing* directory exists at build.
// The Makefile target creates a minimal placeholder when the build is
// skipped.
package webassets

import "embed"

//go:embed all:web/build/client
var WebAssets embed.FS