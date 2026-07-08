// scripts/check-bundle.mjs - drive the built SPA through Chromium
// and exercise every route both pre-auth (401s expected) and
// post-auth (token from ~/.mcp-tools-web.token injected into
// localStorage). Catches runtime errors `pnpm typecheck` misses.
//
// Run after `vite build`. Forces SPA rebuild before launching.

import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { readFileSync, existsSync, rmSync } from "node:fs";
import { resolve } from "node:path";
import os from "node:os";
import process from "node:process";

const ROOT = resolve(process.cwd(), "..");
const URL_BASE = "http://127.0.0.1:18897";
const TOKEN_FILE = resolve(os.homedir(), ".mcp-tools-web.token");

// Discover the current entry chunk name from the built index.html so
// the script doesn't need editing every time Vite re-hashes.
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
  "/services", "/logs", "/settings", "/setup",
];

const server = startServer();
const errors = [];
let token = null;
try {
  token = readFileSync(TOKEN_FILE, "utf8").trim();
  console.log("token:", token.length, "chars from", TOKEN_FILE);
} catch {
  console.log("no token file at", TOKEN_FILE);
}

const CHUNK = discoverChunk();
console.log("chunk:", CHUNK);

let browser;
async function snapshot(page, route, label) {
  await page.goto(URL_BASE + route, { waitUntil: "networkidle" });
  await page.waitForTimeout(800);
  const result = await page.evaluate(() => {
    const root = document.getElementById("root");
    const errEl = document.querySelector("[role=alert], .error-boundary");
    return {
      rootLen: root?.innerHTML.length ?? 0,
      hasError: !!errEl,
      text: (root?.textContent ?? "").slice(0, 80).replace(/\s+/g, " ").trim(),
    };
  });
  const tag = `${route.padEnd(12)} root=${String(result.rootLen).padStart(5)}`;
  console.log(`  ${tag}  ${label}: hasError=${result.hasError}`);
  if (result.rootLen === 0) errors.push(`${label} ${route}: root empty`);
  if (result.hasError && label === "post") errors.push(`${label} ${route}: ErrorBoundary shown`);
  console.log(`    "${result.text}"`);
}

try {
  await waitForServer();
  browser = await chromium.launch({ headless: true });

  console.log("\n=== pre-auth ===");
  {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    let preAuth401 = 0;
    page.on("response", (r) => {
      if (r.url().includes("/api/") && r.status() === 401) preAuth401++;
    });
    for (const route of ROUTES) await snapshot(page, route, "pre");
    console.log(`  pre-auth 401s (expected): ${preAuth401}`);
    await ctx.close();
  }

  if (token) {
    console.log("\n=== post-auth ===");
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.addInitScript((t) => {
      window.localStorage.setItem("mcp-tools-token", t);
    }, token);
    page.on("pageerror", (err) => errors.push("pageerror: " + err.message));
    page.on("console", (msg) => {
      if (msg.type() === "error" && msg.text().includes("401")) {
        errors.push("post-auth 401: " + msg.text());
      }
    });
    for (const route of ROUTES) await snapshot(page, route, "post");
    await ctx.close();
  }
} finally {
  if (browser) await browser.close();
  server.kill("SIGTERM");
}

console.log("\n--- runtime check ---");
console.log("errors:", errors.length);
errors.slice(0, 30).forEach((e) => console.log("  ", e));
if (errors.length > 0) {
  console.error("\nFAIL:", errors.length, "issue(s)");
  process.exit(1);
}
console.log("\nPASS");
