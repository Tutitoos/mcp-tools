import { execFile } from "node:child_process";
import { promisify } from "node:util";

/**
 * Resolve the git toplevel for `dir`, or `null` when it's not a directory,
 * not inside any git repo, timed out, or git is missing.
 *
 * Defensive timeout: a hook that guards every glob/bash call must never
 * hang the agent turn on a stuck subprocess (stale NFS mount, etc.). Wraps
 * `execFile` with `promisify` at call time (not module load) so tests can
 * swap `node:child_process` via `mock.module` before this ever runs.
 */
export async function gitToplevel(dir: string): Promise<string | null> {
  try {
    const { stdout } = await promisify(execFile)("git", ["-C", dir, "rev-parse", "--show-toplevel"], { timeout: 3000 });
    return stdout.trim() || null;
  } catch {
    return null;
  }
}

export interface RepoRootCache {
  /** Resolve (and cache) the git toplevel for a session's current working directory. */
  getRoot(cwd: string): Promise<string | null>;
  /** Resolve (and cache) the git toplevel for a candidate path found in a tool call. */
  getOther(path: string): Promise<string | null>;
  clear(): void;
}

/**
 * Per-instance repo-root cache. Each guard/session gets its own instance
 * (via the guard factory) instead of sharing a module-global map, so state
 * from one project/session never bleeds into another.
 */
export function makeRepoRootCache(): RepoRootCache {
  const rootCache = new Map<string, string | null>();
  const otherCache = new Map<string, string | null>();

  async function resolve(cache: Map<string, string | null>, dir: string): Promise<string | null> {
    if (!cache.has(dir)) cache.set(dir, await gitToplevel(dir));
    return cache.get(dir) ?? null;
  }

  return {
    getRoot: (cwd) => resolve(rootCache, cwd),
    getOther: (path) => resolve(otherCache, path),
    clear() {
      rootCache.clear();
      otherCache.clear();
    },
  };
}
