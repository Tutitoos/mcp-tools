import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { loadConfig } from "../common/config.js";
import type { Guard, GuardKey } from "../common/types.js";
import { makeCodebaseMemoryCrossRepoGuard } from "../guards/codebase-memory-cross-repo.js";
import { makeMem0SearchFirstGuard } from "../guards/mem0-search-first.js";
import { makeSerenaSymbolGuard } from "../guards/serena-symbol.js";
import { makeTokensaveExploreGuard } from "../guards/tokensave-explore.js";

/**
 * Force-routes code intelligence work to connected MCP servers instead of
 * native tools / subagents, at the code level (not prompting).
 *
 * All four guards check *connection* (`pi.getAllTools()`, the full
 * discovered set) rather than *activation* (`pi.getActiveTools()`) -- with
 * `mcp.discoveryMode: true`, MCP tools start hidden behind
 * `search_tool_bm25` to keep the per-turn tool schema small, so gating on
 * "active" would make every guard permanently inert. The model discovers
 * the redirect target itself via `search_tool_bm25` on first use each
 * session; that's a one-time round trip, not a standing per-turn token cost.
 *
 * Order matters: serena-symbol first (cheapest, sync), then
 * codebase-memory-cross-repo (async git), then tokensave-explore, then
 * mem0-search-first. Guards disabled via settings are skipped at
 * construction time -- no per-call cost for a disabled guard.
 */
const GUARD_FACTORIES = {
  "serena-symbol": makeSerenaSymbolGuard,
  "codebase-memory-cross-repo": makeCodebaseMemoryCrossRepoGuard,
  "tokensave-explore": makeTokensaveExploreGuard,
  "mem0-search-first": makeMem0SearchFirstGuard,
} as const satisfies Record<GuardKey, (pi: ExtensionAPI) => Guard>;

const GUARD_ORDER = Object.keys(GUARD_FACTORIES) as GuardKey[];

export default function mcpRouting(pi: ExtensionAPI): void {
  const config = loadConfig(process.cwd());

  const guards: Guard[] = GUARD_ORDER.filter((key) => config.guards[key].enabled).map((key) => GUARD_FACTORIES[key](pi));

  pi.on("tool_call", async (event, ctx) => {
    for (const guard of guards) {
      const result = await guard.check(event, ctx);
      if (result) return result;
    }
  });
}
