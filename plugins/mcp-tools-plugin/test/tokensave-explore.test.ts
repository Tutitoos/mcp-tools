import { describe, expect, it } from "bun:test";
import type { ExtensionAPI, ExtensionContext } from "@oh-my-pi/pi-coding-agent";
import { makeTokensaveExploreGuard } from "../src/guards/tokensave-explore.js";
import { createFakePi, fakeCtx, toolCallEvent } from "./helpers/fake-pi.js";

const TOKENSAVE_TOOL = "mcp__tokensave_context";
const ctx = fakeCtx() as unknown as ExtensionContext;

function guardWith(tools: string[]) {
  return makeTokensaveExploreGuard(createFakePi({ tools }) as unknown as ExtensionAPI);
}

describe("tokensave-explore guard", () => {
  it("blocks task(agent: explore) when tokensave is connected", async () => {
    const guard = guardWith([TOKENSAVE_TOOL]);
    const result = await guard.check(toolCallEvent("task", { agent: "explore" }), ctx);
    expect(result?.block).toBe(true);
    expect(result?.reason).toContain("tokensave_context");
  });

  it("passes task(agent: explore) when tokensave is not connected", async () => {
    const guard = guardWith([]);
    const result = await guard.check(toolCallEvent("task", { agent: "explore" }), ctx);
    expect(result).toBeUndefined();
  });

  it("passes task(agent: tester) even when tokensave is connected", async () => {
    const guard = guardWith([TOKENSAVE_TOOL]);
    const result = await guard.check(toolCallEvent("task", { agent: "tester" }), ctx);
    expect(result).toBeUndefined();
  });
});
