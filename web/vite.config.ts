import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import tsconfigPaths from "vite-tsconfig-paths";

// Library-mode SPA build (no @react-router/dev, no SSR/prerender).
// Vite's standard `index.html` entry produces:
//   build/client/index.html              -- the SPA shell
//   build/client/assets/<chunk>.{js,css} -- hashed bundles
// which the Go binary embeds via webassets/ //go:embed.
export default defineConfig({
  plugins: [react(), tailwindcss(), tsconfigPaths()],
  build: {
    outDir: "build/client",
    emptyOutDir: true,
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
});