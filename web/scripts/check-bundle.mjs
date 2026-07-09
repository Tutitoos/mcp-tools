// scripts/check-bundle.mjs - drive the built SPA through Chromium and
// exercise every route at multiple viewports. The API is open (no auth),
// so we just verify each route renders real content with no runtime
// errors and that the layout fits the viewport (no horizontal scroll).
//
// Run after `vite build` and after the server binary has been built
// (`make build` puts it at `bin/mcp-tools`).

import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import process from "node:process";

const ROOT = resolve(process.cwd(), "..");
const URL_BASE = "http://127.0.0.1:18897";

const VIEWPORTS = [
  { label: "phone",  width: 375,  height: 720 },
  { label: "tablet", width: 768,  height: 1024 },
  { label: "desktop", width: 1280, height: 800 },
];

function discoverChunk() {
  const html = readFileSync(resolve(process.cwd(), "build/client/index.html"), "utf8");
  // Match the legacy Vite default name (`index-<hash>.js`), the
  // interim stable name (`entry.client.js`), and the current
  // content-hashed name (`entry.client-<hash>.js`).
  const m = html.match(/src="\/assets\/(?:index-[^"]+\.js|entry\.client[^"]*\.js)"/);
  if (!m) throw new Error("entry chunk not found in build/client/index.html");
  return m[0];
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
  "/services", "/logs", "/settings", "/plugins", "/jobs", "/jobs?q=test",
];

const server = startServer();
const errors = [];
let browser;
let modelsResponse = { status: null, body: null };

// checkVisibility() walks the ancestor chain (so a child of a
// `display: none` parent is correctly reported as hidden). Falls back
// to getClientRects() for older runtimes.
const visibleScript = `
  (() => {
    const trigger = document.querySelector("[data-mobile-nav-trigger]");
    const navEl = document.querySelector("aside nav");
    const visible = (el) => {
      if (!el) return false;
      if (typeof el.checkVisibility === "function") return el.checkVisibility();
      return el.getClientRects().length > 0;
    };
    return {
      mobileTrigger: visible(trigger),
      desktopNav: visible(navEl),
    };
  })();
`;

async function snapshot(page, route, viewport) {
  await page.setViewportSize({ width: viewport.width, height: viewport.height });
  await page.goto(URL_BASE + route, { waitUntil: "networkidle" });
  await page.waitForTimeout(600);

  const layout = await page.evaluate(() => {
    const html = document.documentElement;
    const overflow = html.scrollWidth > html.clientWidth;
    const root = document.getElementById("root");
    return {
      scrollW: html.scrollWidth,
      clientW: html.clientWidth,
      overflow,
      rootLen: root?.innerHTML.length ?? 0,
    };
  });

  // Per-route, per-breakpoint nav assertions.
  const nav = await page.evaluate(visibleScript);

  const tag = `${viewport.label}@${viewport.width}`;
  if (layout.overflow) {
    errors.push(`${tag} ${route}: horizontal overflow scrollW=${layout.scrollW} clientW=${layout.clientW}`);
  }
  if (layout.rootLen === 0) errors.push(`${tag} ${route}: root empty`);

  if (viewport.width < 768) {
    if (!nav.mobileTrigger) {
      errors.push(`${tag} ${route}: mobile nav trigger missing/hidden`);
    }
    if (nav.desktopNav) {
      errors.push(`${tag} ${route}: desktop nav should be hidden at <768`);
    }
  } else {
    if (!nav.desktopNav) {
      errors.push(`${tag} ${route}: desktop nav missing/hidden`);
    }
    if (nav.mobileTrigger) {
      errors.push(`${tag} ${route}: mobile nav trigger should be hidden at >=768`);
    }
  }

  console.log(
    `  ${tag.padEnd(13)} ${route.padEnd(12)} scrollW=${String(layout.scrollW).padStart(5)}  clientW=${String(layout.clientW).padStart(5)}  overflow=${layout.overflow}  root=${String(layout.rootLen).padStart(5)}  mobile=${nav.mobileTrigger}  desktop=${nav.desktopNav}`,
  );
}

try {
  await waitForServer();
  browser = await chromium.launch({ headless: true });
  const ctx = await browser.newContext();

  // Capture /api/models response so we can verify the empty-array-on-error
  // contract without depending on the dev environment's docker state.
  ctx.route("**/api/models", async (route, request) => {
    const resp = await route.fetch();
    let body = null;
    try {
      body = await resp.json();
    } catch {
      body = null;
    }
    modelsResponse = { status: resp.status(), body };
    await route.fulfill({ response: resp });
  });

  const page = await ctx.newPage();
  page.on("pageerror", (err) => errors.push("pageerror: " + err.message));
  page.on("console", (msg) => {
    if (msg.type() === "error") {
      const t = msg.text();
      if (!t.includes("401")) errors.push("console.error: " + t.slice(0, 200));
    }
  });
  console.log("chunk:", discoverChunk());
  for (const viewport of VIEWPORTS) {
    console.log(`\n--- viewport: ${viewport.label} (${viewport.width}x${viewport.height}) ---`);
    for (const route of ROUTES) await snapshot(page, route, viewport);
  }

  // Exercise the hamburger on the phone viewport: open the Sheet and
  // confirm a Radix dialog is mounted with data-state=open.
  await page.setViewportSize({ width: 375, height: 720 });
  await page.goto(URL_BASE + "/", { waitUntil: "networkidle" });
  const trigger = page.locator("[data-mobile-nav-trigger]").first();
  if ((await trigger.count()) > 0) {
    await trigger.click();
    await page.waitForTimeout(300);
    const open = await page.evaluate(() => {
      const d = document.querySelector("[role=dialog][data-state=open]");
      return !!d;
    });
    if (!open) errors.push("phone /: Sheet did not open after clicking hamburger");
    else console.log("  phone  /         Sheet opened OK");
  } else {
    errors.push("phone /: no [data-mobile-nav-trigger] to click");
  }
} finally {
  if (browser) await browser.close();
  server.kill("SIGTERM");
}

console.log("\n--- runtime check ---");
console.log("errors:", errors.length);
errors.slice(0, 20).forEach((e) => console.log("  ", e));

// /api/models contract: must be 200 + empty array (not 500) when ollama
// is down. The intercept above captures the first request; if the
// server is healthy, the response will be a non-empty array and we
// only check status 200.
if (modelsResponse.status !== 200) {
  errors.push(`/api/models returned status ${modelsResponse.status} (expected 200)`);
} else if (Array.isArray(modelsResponse.body) && modelsResponse.body.length === 0) {
  console.log("/api/models: 200 + [] (ollama down — empty array contract OK)");
} else if (Array.isArray(modelsResponse.body)) {
  console.log(`/api/models: 200 + ${modelsResponse.body.length} model(s) (ollama up)`);
}

if (errors.length > 0) {
  console.error("\nFAIL:", errors.length, "issue(s)");
  process.exit(1);
}
console.log("\nPASS");