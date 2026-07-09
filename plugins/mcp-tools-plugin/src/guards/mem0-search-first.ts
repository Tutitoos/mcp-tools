import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { detectMcpServers } from "../common/mcp-detect.js";
import { sessionMap } from "../common/session-state.js";
import type { BlockResult, Guard } from "../common/types.js";

/**
 * Blocks `add_memory` until `search_memories` (or `get_memories`) has been
 * called at least once earlier this session -- mem0's own policy is
 * search-before-add, to avoid storing duplicate memories. Session-scoped
 * state (not a per-call heuristic): resets when the session ends/switches.
 * Gated on a mem0 server being connected; inert if none is.
 */
export function makeMem0SearchFirstGuard(pi: ExtensionAPI): Guard {
  const searched = sessionMap<{ searched: boolean }>(pi, () => ({ searched: false }));

  return {
    key: "mem0-search-first",
    check(event, ctx): BlockResult | undefined {
      const mem0 = detectMcpServers(pi.getAllTools()).mem0;
      if (!mem0) return undefined;

      const state = searched.get(ctx.sessionManager.getSessionId());

      if (mem0.searchTools.has(event.toolName)) {
        state.searched = true;
        return undefined;
      }

      if (event.toolName === mem0.addTool && !state.searched) {
        return {
          block: true,
          reason:
            "Call mcp_tools_mem0's search_memories (or get_memories) first this session before add_memory -- " +
            "mem0's own policy is search-before-add, so this doesn't store a duplicate of something already saved.",
        };
      }

      return undefined;
    },
  };
}
