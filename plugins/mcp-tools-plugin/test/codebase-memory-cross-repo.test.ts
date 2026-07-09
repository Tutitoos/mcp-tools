import { homedir } from "node:os";

import { afterAll, describe, expect, it, mock } from "bun:test";
import type { ExtensionAPI, ExtensionContext } from "@oh-my-pi/pi-coding-agent";
import { makeCodebaseMemoryCrossRepoGuard } from "../src/guards/codebase-memory-cross-repo.js";
import { createFakePi, fakeCtx, toolCallEvent } from "./helpers/fake-pi.js";

/**
 * `git.ts` promisifies `execFile` at call time (not module load), so
 * `mock.module` here reliably intercepts it regardless of import order --
 * `gitToplevel` maps a directory to a synthetic repo root instead of
 * shelling out to real git.
 */
const REPO_ROOTS: Record<string, string> = {
  "/repo": "/repo",
  "/other/repo": "/other/repo",
  [`${homedir()}/other-project`]: `${homedir()}/other-project`,
};

mock.module("node:child_process", () => ({
  execFile: (
    _cmd: string,
    args: string[],
    _opts: unknown,
    cb: (err: Error | null, res: { stdout: string; stderr: string }) => void,
  ) => {
    const dir = args[1]; // ["-C", dir, "rev-parse", "--show-toplevel"]
    const root = REPO_ROOTS[dir];
    if (root) cb(null, { stdout: `${root}\n`, stderr: "" });
    else cb(new Error("not a git repository"), { stdout: "", stderr: "" });
  },
}));

afterAll(() => {
  mock.restore();
});

const CODEBASE_MEMORY_TOOL = "mcp__mcp_tools_codebase_memory_search_code";

function guardWith(tools: string[]) {
  return makeCodebaseMemoryCrossRepoGuard(createFakePi({ tools }) as unknown as ExtensionAPI);
}

describe("codebase-memory-cross-repo guard", () => {
  it("blocks a glob whose path is a different git repo", async () => {
    const guard = guardWith([CODEBASE_MEMORY_TOOL]);
    const ctx = fakeCtx("s1", "/repo") as unknown as ExtensionContext;
    const result = await guard.check(toolCallEvent("glob", { path: "/other/repo/**" }), ctx);
    expect(result?.block).toBe(true);
    expect(result?.reason).toContain("/other/repo");
  });

  it("passes a glob path that is already inside the current repo", async () => {
    const guard = guardWith([CODEBASE_MEMORY_TOOL]);
    const ctx = fakeCtx("s2", "/repo") as unknown as ExtensionContext;
    const result = await guard.check(toolCallEvent("glob", { path: "/repo/src/**" }), ctx);
    expect(result).toBeUndefined();
  });

  it("blocks a bash cd into a different repo before grep", async () => {
    const guard = guardWith([CODEBASE_MEMORY_TOOL]);
    const ctx = fakeCtx("s3", "/repo") as unknown as ExtensionContext;
    const result = await guard.check(
      toolCallEvent("bash", { command: "cd ~/other-project && grep foo" }),
      ctx,
    );
    expect(result?.block).toBe(true);
  });

  it("passes a cross-repo path when codebase-memory is not connected", async () => {
    const guard = guardWith([]);
    const ctx = fakeCtx("s4", "/repo") as unknown as ExtensionContext;
    const result = await guard.check(toolCallEvent("glob", { path: "/other/repo/**" }), ctx);
    expect(result).toBeUndefined();
  });
});
