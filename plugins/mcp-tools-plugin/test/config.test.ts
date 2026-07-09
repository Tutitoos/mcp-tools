import { afterAll, describe, expect, it, mock } from "bun:test";
import { mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { loadConfig } from "../src/common/config.js";
import { DEFAULT_CONFIG } from "../src/common/types.js";

let currentHome = mkdtempSync(join(tmpdir(), "mcp-tools-plugin-home-"));

// `loadConfig` looks up `homedir()` at call time (inside the function
// body), so this mock -- applied once at file load -- is picked up by every
// `loadConfig()` call regardless of which fake home is active at the time.
mock.module("node:os", () => ({ homedir: () => currentHome }));

afterAll(() => {
  mock.restore();
});

function freshDir(prefix: string): string {
  return mkdtempSync(join(tmpdir(), prefix));
}

function writeUserConfig(home: string, content: string): void {
  const dir = join(home, ".omp", "agent");
  mkdirSync(dir, { recursive: true });
  writeFileSync(join(dir, "mcp-tools-plugin.config.json"), content, "utf8");
}

function writeProjectConfig(cwd: string, content: string): void {
  const dir = join(cwd, ".omp");
  mkdirSync(dir, { recursive: true });
  writeFileSync(join(dir, "mcp-tools-plugin.config.json"), content, "utf8");
}

describe("loadConfig", () => {
  it("returns DEFAULT_CONFIG when no settings files exist", () => {
    currentHome = freshDir("mcp-tools-plugin-home-");
    const cwd = freshDir("mcp-tools-plugin-cwd-");
    expect(loadConfig(cwd)).toEqual(DEFAULT_CONFIG);
  });

  it("applies a user-layer override", () => {
    currentHome = freshDir("mcp-tools-plugin-home-");
    writeUserConfig(currentHome, JSON.stringify({ guards: { "serena-symbol": { enabled: false } } }));
    const cwd = freshDir("mcp-tools-plugin-cwd-");

    const config = loadConfig(cwd);
    expect(config.guards["serena-symbol"].enabled).toBe(false);
    expect(config.guards["tokensave-explore"].enabled).toBe(true);
  });

  it("lets a project-layer override win over the user layer", () => {
    currentHome = freshDir("mcp-tools-plugin-home-");
    writeUserConfig(currentHome, JSON.stringify({ guards: { "serena-symbol": { enabled: false } } }));
    const cwd = freshDir("mcp-tools-plugin-cwd-");
    writeProjectConfig(cwd, JSON.stringify({ guards: { "serena-symbol": { enabled: true } } }));

    const config = loadConfig(cwd);
    expect(config.guards["serena-symbol"].enabled).toBe(true);
  });

  it("falls back to defaults without throwing on malformed JSON", () => {
    currentHome = freshDir("mcp-tools-plugin-home-");
    writeUserConfig(currentHome, "{ not valid json");
    const cwd = freshDir("mcp-tools-plugin-cwd-");

    expect(() => loadConfig(cwd)).not.toThrow();
    expect(loadConfig(cwd)).toEqual(DEFAULT_CONFIG);
  });

  it("ignores an unknown guard key", () => {
    currentHome = freshDir("mcp-tools-plugin-home-");
    const cwd = freshDir("mcp-tools-plugin-cwd-");
    writeProjectConfig(
      cwd,
      JSON.stringify({ guards: { "nonexistent-guard": { enabled: false }, "serena-symbol": { enabled: false } } }),
    );

    const config = loadConfig(cwd);
    expect(config.guards["serena-symbol"].enabled).toBe(false);
    expect(Object.keys(config.guards).sort()).toEqual(Object.keys(DEFAULT_CONFIG.guards).sort());
  });
});
