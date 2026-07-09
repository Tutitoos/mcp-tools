import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { detectMcpServers } from "../common/mcp-detect.js";
import type { BlockResult, Guard } from "../common/types.js";

/**
 * Blocks `task(agent: "explore")` -- the project's MANDATORY prose rule
 * already forbids spawning an Explore agent when tokensave is available;
 * this makes that a hard block instead of a prompt the model can skip.
 * Gated on tokensave being a connected MCP server this session.
 */
export function makeTokensaveExploreGuard(pi: ExtensionAPI): Guard {
  return {
    key: "tokensave-explore",
    check(event): BlockResult | undefined {
      if (event.toolName !== "task") return undefined;
      const input = event.input as Record<string, unknown>;
      if (input.agent !== "explore") return undefined;
      if (!detectMcpServers(pi.getAllTools()).tokensave) return undefined;

      return {
        block: true,
        reason:
          "tokensave is available in this project. Use tokensave_context directly with your question in plain " +
          "English instead of spawning an explore subagent -- do not call Read/glob/grep/list_directory either; " +
          "the source sections tokensave_context returns ARE the relevant code. Call search_tool_bm25 first if " +
          "tokensave_context isn't in your active tools yet.",
      };
    },
  };
}
