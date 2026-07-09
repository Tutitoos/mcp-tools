import { afterEach, describe, expect, it } from "bun:test";
import { mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import type { AgentEndEvent, ExtensionAPI, ExtensionContext, ToolResultEvent } from "@oh-my-pi/pi-coding-agent";
import postTaskMaintenanceOffer from "../src/extensions/post-task-maintenance-offer.js";
import { createFakePi, fakeCtx } from "./helpers/fake-pi.js";

// `loadConfig(process.cwd())` is real, unmocked code here (no `mock.module`
// on a module other test files also import -- that leaks across files
// within the same `bun test` process). Instead each test `chdir`s into a
// fresh temp project, optionally seeded with a project-layer settings file.
const originalCwd = process.cwd();

afterEach(() => {
  process.chdir(originalCwd);
});

function chdirToFreshProject(config?: Record<string, unknown>): void {
  const dir = mkdtempSync(join(tmpdir(), "mcp-tools-plugin-ptm-"));
  if (config) {
    mkdirSync(join(dir, ".omp"), { recursive: true });
    writeFileSync(join(dir, ".omp", "mcp-tools-plugin.config.json"), JSON.stringify(config), "utf8");
  }
  process.chdir(dir);
}

function toolResultEvent(toolName: string, isError = false): ToolResultEvent {
  return { type: "tool_result", toolCallId: "t1", toolName, input: {}, content: [], isError, details: undefined } as ToolResultEvent;
}

const AGENT_END: AgentEndEvent = { type: "agent_end", messages: [] };

describe("post-task-maintenance-offer extension", () => {
  it("sends the nudge once per mutation, not on a repeat agent_end", async () => {
    chdirToFreshProject();
    const pi = createFakePi();
    const ctx = fakeCtx("s1") as unknown as ExtensionContext;
    postTaskMaintenanceOffer(pi as unknown as ExtensionAPI);

    await pi.emit("tool_result", toolResultEvent("edit"), ctx);
    await pi.emit("agent_end", AGENT_END, ctx);
    expect(pi.sentMessages.length).toBe(1);

    await pi.emit("agent_end", AGENT_END, ctx);
    expect(pi.sentMessages.length).toBe(1);
  });

  it("does not mark the session dirty on a read tool_result", async () => {
    chdirToFreshProject();
    const pi = createFakePi();
    const ctx = fakeCtx("s2") as unknown as ExtensionContext;
    postTaskMaintenanceOffer(pi as unknown as ExtensionAPI);

    await pi.emit("tool_result", toolResultEvent("read"), ctx);
    await pi.emit("agent_end", AGENT_END, ctx);
    expect(pi.sentMessages.length).toBe(0);
  });

  it("wires nothing when postTaskMaintenance is disabled", async () => {
    chdirToFreshProject({ postTaskMaintenance: { enabled: false } });
    const pi = createFakePi();
    const ctx = fakeCtx("s3") as unknown as ExtensionContext;
    postTaskMaintenanceOffer(pi as unknown as ExtensionAPI);

    await pi.emit("tool_result", toolResultEvent("edit"), ctx);
    await pi.emit("agent_end", AGENT_END, ctx);
    expect(pi.sentMessages.length).toBe(0);
  });
});
