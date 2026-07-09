// Package webassets embeds the built SPA + SSR bundles produced by
// `make build-web`. The path is relative to repo root because Vite
// writes both outputs to `web/build/` (configured in web/vite.config.ts):
//
//	web/build/client/index.html            -- SPA shell (also used as fallback)
//	web/build/client/assets/entry.client.js -- hydrated client bundle
//	web/build/server/index.js              -- SSR bundle (Node ESM)
//
// During step 1 the directory may be empty (only present when the SPA
// has been built at least once); the embed directive tolerates a missing
// tree at compile time as long as the *containing* directory exists at
// build. The Makefile target creates a minimal placeholder when the
// build is skipped.
package webassets

import "embed"

//go:embed all:web/build
var WebAssets embed.FS
