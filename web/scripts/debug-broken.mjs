// debug /, /configure pages specifically
import { chromium } from "playwright";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import os from "node:os";

const TOKEN = readFileSync(resolve(os.homedir(), ".mcp-tools-web.token"), "utf8").trim();
const URL = "http://127.0.0.1:18899";

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext();
const page = await ctx.newPage();
await page.addInitScript((t) => {
  window.localStorage.setItem("mcp-tools-token", t);
}, TOKEN);

for (const route of ["/", "/configure"]) {
  console.log("\n=========", route, "=========");
  page.removeAllListeners("pageerror");
  page.removeAllListeners("console");
  page.on("pageerror", (err) => console.log("  pageerror:", err.message));
  page.on("console", (msg) => {
    if (msg.type() === "error" || msg.type() === "warning") {
      console.log(`  console.${msg.type()}: ${msg.text().slice(0, 300)}`);
    }
  });
  await page.goto(URL + route, { waitUntil: "networkidle" });
  await page.waitForTimeout(1500);
  const html = await page.evaluate(() => {
    const r = document.getElementById("root");
    return r?.innerHTML ?? "(no root)";
  });
  console.log(`  root.innerHTML.length: ${html.length}`);
  console.log(`  first 600 chars: ${html.slice(0, 600)}`);
}

await browser.close();