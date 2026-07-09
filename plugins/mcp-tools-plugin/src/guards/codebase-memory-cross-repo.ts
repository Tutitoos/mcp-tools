import { homedir } from "node:os";
import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { makeRepoRootCache } from "../common/git.js";
import { detectMcpServers } from "../common/mcp-detect.js";
import type { BlockResult, Guard } from "../common/types.js";

// bash commands worth checking for a cross-repo search target. `find` is
// also covered by a separate built-in denylist hook, but is kept here too
// since that hook isn't this guard's to rely on.
const SEARCH_CMD = /\b(grep|rg|ag|find)\b/;

// Path token: absolute (`/a/b`) or home-relative (`~/a/b`). Stops at
// whitespace/quotes or the first glob wildcard (*, ?, [, {) -- so it
// doubles as the static directory prefix when applied to a glob pattern.
const PATH_TOKEN = /(?:~)?(?:\/[\w.-]+)+/g;

// `cd <dir> && grep ...` / `cd <dir>; grep ...` is the idiom models reach
// for over passing an absolute path as a grep/find argument -- the token
// extraction above never sees a bare relative `services/` this way, so the
// `cd` target itself is checked as an extra candidate for bash commands.
const CD_PREFIX = /^\s*cd\s+((?:~)?(?:\/[\w.-]+)+)\s*(?:&&|;)/;

function expandTilde(p: string): string {
  return p.startsWith("~/") ? homedir() + p.slice(1) : p;
}

function extractPathCandidates(text: string): string[] {
  const out: string[] = [];
  for (const m of text.matchAll(PATH_TOKEN)) {
    const p = expandTilde(m[0]).replace(/\/+$/, "");
    if (p.length > 1) out.push(p);
  }
  return out;
}

/**
 * Blocks native `glob`/`bash(grep|rg|ag|find ...)` calls that reach outside
 * the current repo into a *different* git repo, redirecting to
 * codebase-memory's `search_code` / `search_graph` / `trace_path`. Native
 * `read` of an already-known path is left alone -- fetching a file you can
 * already name is legitimate regardless of which repo it lives in; only
 * cross-repo *discovery* is redirected. Gated on codebase-memory being a
 * connected MCP server this session.
 */
export function makeCodebaseMemoryCrossRepoGuard(pi: ExtensionAPI): Guard {
  const repoRoots = makeRepoRootCache();

  async function isCrossRepoPath(candidate: string, ownRoot: string): Promise<boolean> {
    if (candidate === ownRoot || candidate.startsWith(`${ownRoot}/`)) return false;

    // A strict ancestor of the current repo root can reach sibling projects
    // through a recursive pattern even when that ancestor isn't itself a
    // git repo (e.g. a `repositories.json`/`*.code-workspace` multi-repo
    // umbrella dir) -- no need to resolve it, the structural relationship
    // is enough.
    if (ownRoot.startsWith(`${candidate}/`)) return true;

    const root = await repoRoots.getOther(candidate);
    return root !== null && root !== ownRoot;
  }

  async function findCrossRepoEscape(candidates: string[], ownRoot: string): Promise<string | undefined> {
    for (const c of candidates) {
      if (await isCrossRepoPath(c, ownRoot)) return c;
    }
    return undefined;
  }

  return {
    key: "codebase-memory-cross-repo",
    async check(event, ctx): Promise<BlockResult | undefined> {
      const toolName = event.toolName;
      const input = event.input as Record<string, unknown>;
      const isBashSearch = toolName === "bash" && SEARCH_CMD.test(String(input.command ?? ""));
      if (toolName !== "glob" && !isBashSearch) return undefined;

      if (!detectMcpServers(pi.getAllTools()).codebaseMemory) return undefined;

      const ownRoot = await repoRoots.getRoot(ctx.cwd);
      if (!ownRoot) return undefined;

      const text = toolName === "glob" ? String(input.path ?? "") : String(input.command ?? "");
      const candidates = extractPathCandidates(text);
      if (toolName === "bash") {
        const cdMatch = CD_PREFIX.exec(text);
        if (cdMatch) candidates.unshift(expandTilde(cdMatch[1]));
      }
      const escape = await findCrossRepoEscape(candidates, ownRoot);
      if (!escape) return undefined;

      return {
        block: true,
        reason:
          `"${escape}" is outside this repo (a different project). Use mcp_tools_codebase_memory ` +
          `(search_code / search_graph / trace_path) for cross-repo search instead of ${toolName} -- call ` +
          `search_tool_bm25 first if these aren't in your active tools yet, then run list_projects and ` +
          `index_repository if it isn't indexed yet.`,
      };
    },
  };
}
