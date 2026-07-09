import type { ExtensionContext, ToolCallEvent } from "@oh-my-pi/pi-coding-agent";

export type GuardKey =
  | "serena-symbol"
  | "codebase-memory-cross-repo"
  | "tokensave-explore"
  | "mem0-search-first";

export interface BlockResult {
  block: true;
  reason: string;
}

/** A single routing rule: composed by the extension entry, testable in isolation via a fake `pi`. */
export interface Guard {
  key: GuardKey;
  check(event: ToolCallEvent, ctx: ExtensionContext): BlockResult | undefined | Promise<BlockResult | undefined>;
}

export interface PluginConfig {
  guards: Record<GuardKey, { enabled: boolean }>;
  postTaskMaintenance: { enabled: boolean };
}

export const DEFAULT_CONFIG: PluginConfig = {
  guards: {
    "serena-symbol": { enabled: true },
    "codebase-memory-cross-repo": { enabled: true },
    "tokensave-explore": { enabled: true },
    "mem0-search-first": { enabled: true },
  },
  postTaskMaintenance: { enabled: true },
};
