import { describe, expect, it } from "bun:test";
import type { ExtensionAPI, ExtensionContext } from "@oh-my-pi/pi-coding-agent";
import { makeSerenaSymbolGuard } from "../src/guards/serena-symbol.js";
import { createFakePi, fakeCtx, toolCallEvent } from "./helpers/fake-pi.js";

const SERENA_TOOL = "mcp__mcp_tools_serena_find_symbol";
const ctx = fakeCtx() as unknown as ExtensionContext;

function guardWith(tools: string[]) {
  return makeSerenaSymbolGuard(createFakePi({ tools }) as unknown as ExtensionAPI);
}

describe("serena-symbol guard", () => {
  it("blocks grep with a symbol-like pattern when serena is connected", async () => {
    const guard = guardWith([SERENA_TOOL]);
    const result = await guard.check(toolCallEvent("grep", { pattern: "foo_bar" }), ctx);
    expect(result?.block).toBe(true);
    expect(result?.reason).toContain('"foo_bar" looks like a named symbol');
  });

  it("passes grep with the same pattern when serena is not connected", async () => {
    const guard = guardWith([]);
    const result = await guard.check(toolCallEvent("grep", { pattern: "foo_bar" }), ctx);
    expect(result).toBeUndefined();
  });

  it("passes grep whose pattern has a space (not symbol-like)", async () => {
    const guard = guardWith([SERENA_TOOL]);
    const result = await guard.check(toolCallEvent("grep", { pattern: "hello world" }), ctx);
    expect(result).toBeUndefined();
  });

  it("blocks a bash-wrapped grep on the extracted pattern", async () => {
    const guard = guardWith([SERENA_TOOL]);
    const result = await guard.check(toolCallEvent("bash", { command: "cd /tmp && grep foo main.go" }), ctx);
    expect(result?.block).toBe(true);
    expect(result?.reason).toContain('"foo" looks like a named symbol');
  });

  it("blocks ast_grep with a qualified symbol pattern", async () => {
    const guard = guardWith([SERENA_TOOL]);
    const result = await guard.check(toolCallEvent("ast_grep", { pat: "Foo::bar" }), ctx);
    expect(result?.block).toBe(true);
    expect(result?.reason).toContain('"Foo::bar" looks like a named symbol');
  });

  it("passes grep with a glob wildcard pattern", async () => {
    const guard = guardWith([SERENA_TOOL]);
    const result = await guard.check(toolCallEvent("grep", { pattern: "*.ts" }), ctx);
    expect(result).toBeUndefined();
  });
});
