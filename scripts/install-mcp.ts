#!/usr/bin/env bun
/**
 * Register the mcp-tools MCP servers into Claude Code, OpenCode, and OMP.
 * Idempotent: existing entries are overwritten; other keys preserved.
 *
 *   Claude Code -> `claude mcp add-json --scope user <name> '<json>'`
 *   OpenCode    -> direct merge into $HOME/.config/opencode/opencode.json (.mcp)
 *   OMP         -> direct merge into $HOME/.omp/agent/mcp.json (.mcpServers)
 *
 * Each direct-merge target gets a timestamped .bak.<ISO> before overwrite.
 */
import { $ } from "bun";
import fs from "node:fs";
import path from "node:path";
import os from "node:os";

const HOME = os.homedir();
const LOCAL_BIN = path.join(HOME, ".local/bin");

interface ServerSpec {
  name: string;
  wrapper: string;
  args: string[];
}

const SERVERS: ServerSpec[] = [
  {
    name: "mcp_tools_codebase_memory",
    wrapper: path.join(LOCAL_BIN, "mcp-tools-codebase-memory-docker"),
    args: ["--ui=false"],
  },
  {
    name: "mcp_tools_mem0",
    wrapper: path.join(LOCAL_BIN, "mcp-tools-mem0-docker"),
    args: [],
  },
  {
    name: "mcp_tools_headroom",
    wrapper: path.join(LOCAL_BIN, "mcp-tools-headroom-docker"),
    args: [],
  },
];

for (const s of SERVERS) {
  if (!fs.existsSync(s.wrapper)) {
    console.error(`ERROR: wrapper ${s.wrapper} missing — run installer wrappers step first`);
    process.exit(1);
  }
}

const isoStamp = () => new Date().toISOString().replace(/[:.]/g, "-");

function backup(file: string) {
  if (!fs.existsSync(file)) return;
  const dst = `${file}.bak.${isoStamp()}`;
  fs.copyFileSync(file, dst);
}

function loadJson(file: string, fallback: Record<string, unknown>) {
  if (!fs.existsSync(file)) return fallback;
  return JSON.parse(fs.readFileSync(file, "utf8"));
}

function writeJson(file: string, obj: unknown) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, JSON.stringify(obj, null, 2) + "\n");
}
// --- Claude Code (via CLI, no direct edits) ---
async function configureClaude() {
  const label = "Claude Code";
  const claudeBin = path.join(HOME, ".local/bin/claude");
  if (!fs.existsSync(claudeBin) && !fs.existsSync("/usr/local/bin/claude")) {
    console.log(`SKIP ${label} (claude CLI not found)`);
    return;
  }
  for (const s of SERVERS) {
    // remove is idempotent — ignore failure when the server doesn't exist
    await $`claude mcp remove --scope user ${s.name}`.nothrow().quiet();
    const spec = {
      type: "stdio" as const,
      command: s.wrapper,
      args: s.args,
      env: { HOME },
    };
    const res = await $`claude mcp add-json --scope user ${s.name} ${JSON.stringify(spec)}`.nothrow();
    if (res.exitCode !== 0) {
      console.error(`FAIL ${label} ${s.name}: ${res.stderr.toString()}`);
      process.exit(1);
    }
  }
  console.log(`OK ${label} (${SERVERS.length} servers via claude mcp add-json --scope user)`);
}

// --- OpenCode ---
function configureOpenCode() {
  const label = "OpenCode";
  const file = path.join(HOME, ".config/opencode/opencode.json");
  const parent = path.dirname(file);
  if (!fs.existsSync(parent)) {
    console.log(`SKIP ${label} (${parent} missing — OpenCode not installed?)`);
    return;
  }
  backup(file);
  const cfg: Record<string, unknown> = loadJson(file, {
    $schema: "https://opencode.ai/config.json",
    mcp: {},
  });
  const mcp = (cfg.mcp as Record<string, unknown>) ?? {};
  for (const s of SERVERS) {
    mcp[s.name] = {
      type: "local",
      command: [s.wrapper, ...s.args],
      enabled: true,
      environment: { HOME },
    };
  }
  cfg.mcp = mcp;
  writeJson(file, cfg);
  console.log(`OK ${label} ${file}`);
}

// --- OMP ---
function configureOmp() {
  const label = "OMP";
  const file = path.join(HOME, ".omp/agent/mcp.json");
  const parent = path.dirname(file);
  if (!fs.existsSync(parent)) {
    console.log(`SKIP ${label} (${parent} missing — OMP not installed?)`);
    return;
  }
  backup(file);
  const cfg: Record<string, unknown> = loadJson(file, {
    $schema:
      "https://raw.githubusercontent.com/can1357/oh-my-pi/main/packages/coding-agent/src/config/mcp-schema.json",
    mcpServers: {},
    disabledServers: [],
  });
  const servers = (cfg.mcpServers as Record<string, unknown>) ?? {};
  for (const s of SERVERS) {
    // Omit "type": "stdio" — OMP schema default per mcp-schema.json.
    servers[s.name] = {
      command: s.wrapper,
      args: s.args,
      env: { HOME },
      enabled: true,
    };
  }
  cfg.mcpServers = servers;
  writeJson(file, cfg);
  console.log(`OK ${label} ${file}`);
}

await configureClaude();
configureOpenCode();
configureOmp();
