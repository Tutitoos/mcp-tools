#!/usr/bin/env node
// Generates the SPA shell index.html for embedding into the Go binary.
//
// React Router v7 SPA mode (`ssr: false`) emits JS bundles + a CSS file but
// no top-level index.html — it expects you to serve via @react-router/serve
// or supply the HTML yourself. We produce a minimal shell that loads every
// emitted JS chunk (so the route registration in app/routes.ts can hydrate)
// and the single root-*.css asset.
//
// Usage: node scripts/postbuild.mjs

import { readdirSync, readFileSync, writeFileSync, existsSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const CLIENT = join(__dirname, "..", "build", "client");
const ASSETS = join(CLIENT, "assets");

if (!existsSync(ASSETS)) {
  console.error(`postbuild: ${ASSETS} not found — did you run 'react-router build'?`);
  process.exit(1);
}

const files = readdirSync(ASSETS);
const jsFiles = files.filter((f) => f.endsWith(".js")).sort();
const cssFiles = files.filter((f) => f.endsWith(".css")).sort();

const scriptTags = jsFiles.map((f) => `<script type="module" src="/assets/${f}"></script>`).join("\n    ");
const styleTags = cssFiles.map((f) => `<link rel="stylesheet" href="/assets/${f}" />`).join("\n    ");

const html = `<!doctype html>
<html lang="es">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>mcp-tools · admin panel</title>
    <meta name="description" content="Web admin panel auto-hospedado para el stack MCP" />
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500;600&display=swap" />
    ${styleTags}
  </head>
  <body>
    <div id="root"></div>
    ${scriptTags}
  </body>
</html>
`;

const out = join(CLIENT, "index.html");
writeFileSync(out, html, "utf-8");
console.log(`postbuild: wrote ${out} (${jsFiles.length} js, ${cssFiles.length} css)`);