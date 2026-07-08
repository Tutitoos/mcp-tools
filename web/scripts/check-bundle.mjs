// scripts/check-bundle.mjs - drive the built SPA through Chromium
// and exercise every route both pre-auth (must redirect to /setup,
// no console spam) and post-auth (real content, 200s).
//
// Run after `vite build`. Catches runtime errors `pnpm typecheck`
// misses: broken module imports, hooks violations, null derefs,
// auth-gate regressions.

import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import os from "node:os";
import process from "node:process";

const ROOT = resolve(process.cwd(), "..");
const URL_BASE = "http://127.0.0.1:18897";
const TOKEN_FILE = resolve(os.homedir(), ".mcp-tools-web.token");

// Discover the current entry chunk name from the built index.html so
// the script doesn\'t need editing every time Vite re-hashes.
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
const console401 = [];
let token = null;
try {
  token = readFileSync(TOKEN_FILE, "utf8").trim();
  console.log("token:", token.length, "chars from", TOKEN_FILE);
} catch {
  console.log("no token file at", TOKEN_FILE);
}

let browser;
async function checkRoute(page, route, label) {
  const before = console401.length;
  await page.goto(URL_BASE + route, { waitUntil: "networkidle" });
  await page.waitForTimeout(800);
  // After auth gate, unauthenticated visits should land on /setup.
  const finalUrl = page.url();
  const pathname = new URL(finalUrl).pathname;
  const result = await page.evaluate(() => {
    const root = document.getElementById("root");
    return { rootLen: root?.innerHTML.length ?? 0 };
  });
  const tag = route.padEnd(12);
  const new401s = console401.length - before;
  console.log(`  ${tag} -> ${pathname.padEnd(12)} root=${String(result.rootLen).padStart(5)} 401s=${new401s}`);
  if (label === "pre" && pathname !== "/setup") {
    errors.push(`pre-auth ${route}: expected redirect to /setup, got ${pathname}`);
  }
  if (label === "pre" && new401s > 4) {
    // 2-3 = expected: Shell + Dashboard each consume the ["status"]
    // query and React Query doesn't always dedupe across consumers
    // that mount in different ticks before the first request resolves.
    // Anything > 4 would mean retry-storm regression (the bug that
    // produced the infinite loop the user reported).
    errors.push(`pre-auth ${route}: ${new401s} 401s (gate should suppress retries)`);
  }
}

try {
  await waitForServer();
  browser = await chromium.launch({ headless: true });

  console.log("\n=== pre-auth (must redirect to /setup, no retry storm) ===");
  {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    page.on("console", (msg) => {
      if (msg.type() === "error" && msg.text().includes("401")) {
        console401.push(msg.text());
      }
    });
    for (const route of ROUTES) await checkRoute(page, route, "pre");
    console.log(`  total 401s logged: ${console401.length}`);
    await ctx.close();
  }

  if (token) {
    console.log("\n=== post-auth (real content, no gate) ===");
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.addInitScript((t) => {
      window.localStorage.setItem("mcp-tools-token", t);
    }, token);
    const postAuth401s = [];
    page.on("pageerror", (err) => errors.push("pageerror: " + err.message));
    page.on("console", (msg) => {
      if (msg.type() === "error" && msg.text().includes("401")) {
        postAuth401s.push(msg.text());
      }
    });
    for (const route of ROUTES) {
      await page.goto(URL_BASE + route, { waitUntil: "networkidle" });
      await page.waitForTimeout(800);
      const result = await page.evaluate(() => {
        const root = document.getElementById("root");
        return { rootLen: root?.innerHTML.length ?? 0 };
      });
      console.log(`  ${route.padEnd(12)} root=${String(result.rootLen).padStart(5)}`);
      if (result.rootLen === 0) errors.push(`post-auth ${route}: root empty`);
    }
    if (postAuth401s.length > 0) {
      errors.push(`post-auth: ${postAuth401s.length} unexpected 401s (token may be invalid)`);
    }
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
