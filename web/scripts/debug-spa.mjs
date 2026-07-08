// debug-spa.mjs - investigate post-auth content
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

page.on("console", (msg) => {
  console.log(`  console.${msg.type()}: ${msg.text().slice(0, 200)}`);
});
page.on("pageerror", (err) => console.log("  pageerror:", err.message));

await page.goto(URL + "/tools", { waitUntil: "networkidle" });
await page.waitForTimeout(2000);

const html = await page.evaluate(() => {
  const root = document.getElementById("root");
  return root ? root.innerHTML : "(no root)";
});

console.log("=== /tools root.innerHTML (first 1000 chars) ===");
console.log(html.slice(0, 1000));
console.log("\n=== root.innerHTML.length:", html.length);

await browser.close();