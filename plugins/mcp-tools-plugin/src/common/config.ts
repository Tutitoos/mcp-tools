import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { DEFAULT_CONFIG, type GuardKey, type PluginConfig } from "./types.js";

const GUARD_KEYS: readonly GuardKey[] = [
  "serena-symbol",
  "codebase-memory-cross-repo",
  "tokensave-explore",
  "mem0-search-first",
];

function readJsonLayer(path: string): Record<string, unknown> {
  try {
    const parsed: unknown = JSON.parse(readFileSync(path, "utf8"));
    return parsed !== null && typeof parsed === "object" && !Array.isArray(parsed) ? (parsed as Record<string, unknown>) : {};
  } catch {
    // Missing file (ENOENT) or malformed JSON -- treat the layer as absent.
    return {};
  }
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

/** Apply one settings layer on top of an already-resolved config. Unknown keys are dropped; invalid values keep the prior layer's value. */
function applyLayer(base: PluginConfig, layer: Record<string, unknown>): PluginConfig {
  const guards = { ...base.guards };
  if (isPlainObject(layer.guards)) {
    for (const key of GUARD_KEYS) {
      const entry = layer.guards[key];
      if (isPlainObject(entry) && typeof entry.enabled === "boolean") {
        guards[key] = { enabled: entry.enabled };
      }
    }
  }

  let postTaskMaintenance = base.postTaskMaintenance;
  if (isPlainObject(layer.postTaskMaintenance) && typeof layer.postTaskMaintenance.enabled === "boolean") {
    postTaskMaintenance = { enabled: layer.postTaskMaintenance.enabled };
  }

  return { guards, postTaskMaintenance };
}

/**
 * Load and merge plugin settings: `DEFAULT_CONFIG <- user layer <- project layer`.
 * Called once at extension load (no hot reload).
 */
export function loadConfig(cwd: string): PluginConfig {
  const userLayer = readJsonLayer(join(homedir(), ".omp", "agent", "mcp-tools-plugin.config.json"));
  const projectLayer = readJsonLayer(join(cwd, ".omp", "mcp-tools-plugin.config.json"));

  return applyLayer(applyLayer(DEFAULT_CONFIG, userLayer), projectLayer);
}
