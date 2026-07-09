import { describe, expect, it } from "bun:test";
import type { ExtensionAPI, ExtensionContext } from "@oh-my-pi/pi-coding-agent";
import { makeMem0SearchFirstGuard } from "../src/guards/mem0-search-first.js";
import { createFakePi, fakeCtx, toolCallEvent, type FakePi } from "./helpers/fake-pi.js";
import type { Guard } from "../src/common/types.js";

const ADD_TOOL = "mcp__mcp_tools_mem_add_memory";
const SEARCH_TOOL = "mcp__mcp_tools_mem_search_memories";

interface GuardFixture {
  pi: FakePi;
  guard: Guard;
}

function guardWith(tools: string[]): GuardFixture {
  const pi = createFakePi({ tools });
  const guard = makeMem0SearchFirstGuard(pi as unknown as ExtensionAPI);
  return { pi, guard };
}

describe("mem0-search-first guard", () => {
  it("blocks add_memory before any search this session", async () => {
    const { guard } = guardWith([ADD_TOOL, SEARCH_TOOL]);
    const ctx = fakeCtx("s1") as unknown as ExtensionContext;
    const result = await guard.check(toolCallEvent(ADD_TOOL, {}), ctx);
    expect(result?.block).toBe(true);
  });

  it("passes add_memory after search_memories in the same session", async () => {
    const { guard } = guardWith([ADD_TOOL, SEARCH_TOOL]);
    const ctx = fakeCtx("s2") as unknown as ExtensionContext;
    await guard.check(toolCallEvent(SEARCH_TOOL, {}), ctx);
    const result = await guard.check(toolCallEvent(ADD_TOOL, {}), ctx);
    expect(result).toBeUndefined();
  });

  it("blocks add_memory again after session_shutdown resets state", async () => {
    const { pi, guard } = guardWith([ADD_TOOL, SEARCH_TOOL]);
    const ctx = fakeCtx("s3") as unknown as ExtensionContext;
    await guard.check(toolCallEvent(SEARCH_TOOL, {}), ctx);
    await pi.emit("session_shutdown", { type: "session_shutdown" }, ctx);
    const result = await guard.check(toolCallEvent(ADD_TOOL, {}), ctx);
    expect(result?.block).toBe(true);
  });

  it("is inert when no mem0 server is connected", async () => {
    const { guard } = guardWith([]);
    const ctx = fakeCtx("s4") as unknown as ExtensionContext;
    const result = await guard.check(toolCallEvent(ADD_TOOL, {}), ctx);
    expect(result).toBeUndefined();
  });
});
