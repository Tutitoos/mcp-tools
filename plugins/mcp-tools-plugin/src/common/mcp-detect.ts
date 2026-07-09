/**
 * MCP server detection from the connected tool-name list.
 *
 * `pi.getAllTools()` returns bridged tool names as
 * `mcp__<sanitized_server_name>_<sanitized_tool>`; the sanitizer turns
 * anything outside [a-z_] -- including digits! -- into `_`. A server
 * configured as "mcp_tools_mem0" bridges to `mcp__mcp_tools_mem_*`, not
 * `mcp__mcp_tools_mem0_*`. Detection here uses substring matching instead of
 * a hardcoded prefix so multi-install layouts (different sanitized names)
 * still work; mem0's own add/search tool names are discovered dynamically
 * by scanning for the `_add_memory` / `_search_memories` / `_get_memories`
 * suffixes rather than assuming a fixed server-name prefix.
 */

export interface Mem0Tools {
  addTool: string;
  searchTools: Set<string>;
}

export interface DetectedMcpServers {
  serena: boolean;
  tokensave: boolean;
  codebaseMemory: boolean;
  mem0: Mem0Tools | null;
}

const isSerenaTool = (name: string): boolean => name.startsWith("mcp__") && name.includes("serena");
const isTokensaveTool = (name: string): boolean => name.startsWith("mcp__tokensave_");
const isCodebaseMemoryTool = (name: string): boolean => name.startsWith("mcp__") && name.includes("codebase_memory");

const MEM0_ADD_SUFFIX = /_add_memory$/;
const MEM0_SEARCH_SUFFIXES = [/_search_memories$/, /_get_memories$/];

export function detectMcpServers(tools: readonly string[]): DetectedMcpServers {
  let serena = false;
  let tokensave = false;
  let codebaseMemory = false;
  let addTool: string | undefined;
  const searchTools = new Set<string>();

  for (const name of tools) {
    if (!serena && isSerenaTool(name)) serena = true;
    if (!tokensave && isTokensaveTool(name)) tokensave = true;
    if (!codebaseMemory && isCodebaseMemoryTool(name)) codebaseMemory = true;

    if (name.startsWith("mcp__")) {
      if (!addTool && MEM0_ADD_SUFFIX.test(name)) addTool = name;
      if (MEM0_SEARCH_SUFFIXES.some((re) => re.test(name))) searchTools.add(name);
    }
  }

  return {
    serena,
    tokensave,
    codebaseMemory,
    mem0: addTool ? { addTool, searchTools } : null,
  };
}
