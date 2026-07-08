// scripts/check-bundle.mjs - drive the built SPA through Chromium
// and exercise every route. The API is open (no auth), so we just
// verify each route renders real content with no runtime errors.
//
// Run after `vite build`.

import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import process from "node:process";

const ROOT = resolve(process.cwd(), "..");
const URL_BASE = "http://127.0.0.1:18897";

function discoverChunk() {
  const html = readFileSync(resolve(process.cwd(), "build/client/index.html"), "utf8");
  const m = html.match(/src="\/assets\/(index-[^"]+\.js)"/);
  if (!m) throw new Error("entry chunk not found in build/client/index.html");
  return m[1];
}

function startServer() {
  const proc = spawn(resolve(ROOT, "bin/mcp-tools"), [
    "serve", "--bind", "127.0.0.1", "--port", "18897",
  ], { stdio: ["ignore", "pipe", "pipe"] });
  return proc;
}

async function waitForServer(timeoutMs = 8000) {
  const t0 = Date.now();
  while (Date.now() - t0 < timeoutMs) {
    try {
      const r = await fetch(URL_BASE + "/api/version");
      if (r.ok) return;
    } catch {}
    await new Promise((r) => setTimeout(r, 200));
  }
  throw new Error("server did not respond within " + timeoutMs + "ms");
}

const ROUTES = [
  "/", "/tools", "/configure", "/models",
  "/services", "/logs", "/settings",
];

const server = startServer();
const errors = [];
let browser;

async function snapshot(page, route) {
  await page.goto(URL_BASE + route, { waitUntil: "networkidle" });
  await page.waitForTimeout(800);
  const result = await page.evaluate(() => {
    const root = document.getElementById("root");
    const text = (root?.textContent ?? "").slice(0, 100).replace(/\s+/g, " ").trim();
    return { rootLen: root?.innerHTML.length ?? 0, text };
  });
  console.log(`  ${route.padEnd(12)} root=${String(result.rootLen).padStart(5)}  "${result.text}"`);
  if (result.rootLen === 0) errors.push(`${route}: root empty`);
}

try {
  await waitForServer();
  browser = await chromium.launch({ headless: true });
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  page.on("pageerror", (err) => errors.push("pageerror: " + err.message));
  page.on("console", (msg) => {
    if (msg.type() === "error") {
      const t = msg.text();
      // Skip 401 noise (none expected anymore, but tolerate).
      if (!t.includes("401")) errors.push("console.error: " + t.slice(0, 200));
    }
  });
  console.log("chunk:", discoverChunk());
  for (const route of ROUTES) await snapshot(page, route);
} finally {
  if (browser) await browser.close();
  server.kill("SIGTERM");
}

console.log("\n--- runtime check ---");
console.log("errors:", errors.length);
errors.slice(0, 20).forEach((e) => console.log("  ", e));
if (errors.length > 0) {
  console.error("\nFAIL:", errors.length, "issue(s)");
  process.exit(1);
}
console.log("\nPASS");
