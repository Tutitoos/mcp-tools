import { readFileSync } from "node:fs";
import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { loadConfig } from "../common/config.js";
import { sessionMap } from "../common/session-state.js";

/**
 * Proactively offers post-task maintenance (tokensave sync, codebase-memory
 * reindex, mem0 durable-memory capture) once a turn that actually changed
 * code settles -- instead of relying on the model to remember a documented
 * convention. See ~/mcp-tools/RULES.md for the underlying per-MCP policy
 * this feeds into (search-before-add for mem0, moderate indexing for
 * codebase-memory, `tokensave sync` for incremental reindex).
 *
 * Mechanism: a `tool_result` listener marks the session "dirty" whenever a
 * native `edit`/`write` call or a known serena mutation succeeds. `agent_end`
 * then checks that flag once the turn settles and, if set, clears it
 * (debounce -- no repeat asks until the next real mutation) and injects a
 * hidden `nextTurn` message that forces one more assistant turn asking the
 * user whether to run the three maintenance actions. The ask itself is
 * code-guaranteed to fire; only its wording/timing/scope is left to the
 * model, since deciding what's "durable enough" for mem0 or whether more
 * work is still queued genuinely needs judgment.
 *
 * Deliberately excludes bare `bash` from the mutation signal: a shell
 * command can't be statically classified as mutating vs. read-only
 * (`ls`/`git status` vs. `sed -i`) without an unreliable command-verb
 * denylist, and a missed offer is far cheaper than nagging after every
 * read-only shell call.
 */

const MUTATING_TOOLS: ReadonlySet<string> = new Set(["edit", "write"]);

const MUTATING_MCP_PATTERN =
  /^mcp__mcp_tools_serena_(replace_content|replace_symbol_body|insert_after_symbol|insert_before_symbol|rename_symbol|delete_symbol|safe_delete_symbol|create_text_file)$/;

function isMutatingToolResult(toolName: string): boolean {
  return MUTATING_TOOLS.has(toolName) || MUTATING_MCP_PATTERN.test(toolName);
}

const OFFER_NUDGE = readFileSync(new URL("../nudges/post-task-maintenance.md", import.meta.url), "utf8").trimEnd();

export default function postTaskMaintenanceOffer(pi: ExtensionAPI): void {
  const config = loadConfig(process.cwd());
  if (!config.postTaskMaintenance.enabled) return;

  const dirty = sessionMap<{ dirty: boolean }>(pi, () => ({ dirty: false }));

  pi.on("tool_result", (event, ctx) => {
    if (event.isError) return;
    if (!isMutatingToolResult(event.toolName)) return;
    dirty.get(ctx.sessionManager.getSessionId()).dirty = true;
  });

  pi.on("agent_end", (_event, ctx) => {
    const state = dirty.get(ctx.sessionManager.getSessionId());
    if (!state.dirty) return;
    state.dirty = false;
    pi.sendMessage(
      { customType: "post-task-maintenance-offer", content: OFFER_NUDGE, display: false, attribution: "agent" },
      { deliverAs: "nextTurn", triggerTurn: true },
    );
  });
}
