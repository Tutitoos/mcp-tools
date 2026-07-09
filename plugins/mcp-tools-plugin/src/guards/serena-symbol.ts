import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { detectMcpServers } from "../common/mcp-detect.js";
import type { BlockResult, Guard } from "../common/types.js";

// Bare identifier / dotted or `::` qualified path, no spaces, no glob
// wildcards (*, ?, [, ]) and no quotes -> looks like a symbol reference.
const SYMBOL_LIKE = /^[A-Za-z_$][\w$]*(?:[.:]{1,2}[A-Za-z_$][\w$]*)*$/;

// bash commands worth checking for a symbol-like search target. `find` is
// also covered by a separate built-in denylist hook, but is kept here too
// since that hook isn't this guard's to rely on.
const SEARCH_CMD = /\b(grep|rg|ag|find)\b/;

// Extract the search pattern from a `grep|rg|ag <pattern> ...` bash
// invocation: skip the command word and any `-flag`/`--flag=value` tokens,
// take the first remaining token, and strip a single layer of surrounding
// quotes. Best-effort, deliberately simple tokenization (no nested
// quote/escape handling) -- a miss here just means one bash-wrapped grep
// slips through unblocked, not a functional break.
const BASH_TOKEN = /'[^']*'|"[^"]*"|\S+/g;

function extractGrepPatternFromBash(command: string): string {
  const tokens = command.match(BASH_TOKEN) ?? [];
  const cmdIdx = tokens.findIndex((t) => /^(grep|rg|ag)$/.test(t.replace(/^.*[/\\]/, "")));
  if (cmdIdx === -1) return "";
  for (let i = cmdIdx + 1; i < tokens.length; i++) {
    if (tokens[i].startsWith("-")) continue;
    return tokens[i].replace(/^['"]|['"]$/g, "");
  }
  return "";
}

function symbolLikeSegment(field: string): string | undefined {
  return field
    .split(";")
    .map((s) => s.trim())
    .find((s) => s.length > 0 && SYMBOL_LIKE.test(s));
}

const APPLICABLE_TOOLS: ReadonlySet<string> = new Set(["grep", "ast_grep", "glob"]);

/**
 * Blocks native `grep`/`ast_grep`/`glob`/`bash(grep|rg|ag|find ...)` calls
 * whose pattern/path looks like a bare symbol name, redirecting to serena's
 * `find_symbol` / `find_referencing_symbols`. Gated on serena being a
 * connected (not necessarily active) MCP server this session.
 */
export function makeSerenaSymbolGuard(pi: ExtensionAPI): Guard {
  return {
    key: "serena-symbol",
    check(event): BlockResult | undefined {
      const toolName = event.toolName;
      const input = event.input as Record<string, unknown>;
      const isBashSearch = toolName === "bash" && SEARCH_CMD.test(String(input.command ?? ""));
      if (!APPLICABLE_TOOLS.has(toolName) && !isBashSearch) return undefined;

      if (!detectMcpServers(pi.getAllTools()).serena) return undefined;

      const field =
        toolName === "grep"
          ? String(input.pattern ?? "")
          : toolName === "ast_grep"
            ? String(input.pat ?? "")
            : toolName === "bash"
              ? extractGrepPatternFromBash(String(input.command ?? ""))
              : String(input.path ?? "");
      const hit = symbolLikeSegment(field);
      if (!hit) return undefined;

      return {
        block: true,
        reason:
          `"${hit}" looks like a named symbol, not literal text. Use mcp_tools_serena instead of ${toolName}: ` +
          `find_symbol / find_referencing_symbols / find_declaration (call search_tool_bm25 first if these aren't ` +
          `in your active tools yet, then activate_project if you haven't this session).`,
      };
    },
  };
}
