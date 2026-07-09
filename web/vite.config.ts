import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import tsconfigPaths from "vite-tsconfig-paths";

// Library-mode SPA build (no @react-router/dev, no SSR/prerender).
// Vite's standard `index.html` entry produces:
//   build/client/index.html              -- the SPA shell
//   build/client/assets/entry.client-<hash>.js
//                                       -- the entry chunk (content-hashed
//                                          name; Vite rewrites the
//                                          <script src=...> in
//                                          index.html at build time)
// When invoked with `--ssr` (via the Makefile), the same config produces
//   build/server/index.js                -- a self-contained Node ESM module
//                                           that exports a default async
//                                           function (url) => Promise<string>
// which the Go binary embeds via webassets/ //go:embed and invokes per
// request through internal/web/ssr.go.
export default defineConfig(({ isSsrBuild }) => ({
  plugins: [react(), tailwindcss(), tsconfigPaths()],
  build: isSsrBuild
    ? {
        // SSR build: a single Node ESM module that exports a default
        // async function (url: string) => Promise<string> returning
        // fully-rendered HTML for the given path. The Go server invokes
        // it per request via `node build/server/index.js <url>`.
        outDir: "build/server",
        emptyOutDir: true,
        ssr: true,
        // Inline every dep so the SSR bundle is self-contained.
        // `noExternal: true` forces Vite/Rollup to bundle react,
        // react-dom, react-router, etc. into the single output file;
        // the runtime needs no node_modules on disk.
        rollupOptions: {
          input: "app/entry.server.tsx",
          output: { entryFileNames: "index.js", format: "esm" },
        },
        target: "node20",
        minify: false, // keep stack traces readable
      }
    : {
      // Client build. The entry chunk gets a content hash so every
      // release produces a new URL — the browser is forced to refetch
      // instead of reusing a stale cache entry. Vite rewrites the
      // <script src=...> in the built index.html automatically; the SSR
      // template loader (which reads build/client/index.html once at
      // startup) sees the hashed name with no extra work.
      outDir: "build/client",
      emptyOutDir: true,
      rollupOptions: {
        output: { entryFileNames: "assets/entry.client-[hash].js" },
      },
    },
  ssr: {
    noExternal: true,
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8888",
        changeOrigin: true,
      },
    },
  },
}));