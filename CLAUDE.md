<!-- GENERATED FILE - do not edit. Source: instructions/. Regenerate: mcp-tools instructions sync -->
# mcp-tools agent instructions

This repository contains an installer + registry for MCP servers and operational skills.

**Tool routing (which MCP for which intent) is defined ONCE in `RULES.md`** — installed globally in every supported client. It is not duplicated here; if it is not loaded in your session, read `RULES.md` at the repo root first.

## Serena-first for this repo

`mcp_tools_serena` is the default for ANY operation on a named symbol (LSP-accurate, no string/comment false positives). First code-operation call of every session: `activate_project("/home/tutitoos/mcp-tools")`. Then:

- body of X / "cómo funciona X" → `find_symbol(name_path_pattern: "X", include_body: true)`
- who uses X / "quién llama a X" → `find_referencing_symbols`; declaration → `find_declaration`
- file outline → `get_symbols_overview(relative_path: "internal/.../file.go")`
- semantic edit / rename → `replace_symbol_body` / `rename_symbol`

Native `Read` is only for raw config, docs, `.env`, logs, JSON dumps, and non-LSP languages. Never to "see how function X works" — about to `Read` a `.go`/`.ts`/`.py`/`.rs`/`.java` file for that? Stop and use serena. If serena errors or shows "not connected" → escalation list in `RULES.md`.

## Per-MCP skills

Read as needed before repo work: `skills/serena/SKILL.md`, `skills/tokensave/SKILL.md`, `skills/codebase-memory/SKILL.md`, `skills/mem0/SKILL.md`. Note: mem0 `search_memories`/`get_memories` are broken upstream — workarounds in `skill://mem0` §Known state.

## Repo facts

- MCP servers run as host binaries (not Docker): `~/.local/bin/codebase-memory-mcp`, `~/.local/bin/mem0-launcher` (sources `.env.mem0`), `~/.local/bin/serena` (install: web panel `/tools` → serena, or `uv tool install -p 3.13 serena-agent`), `~/.cargo/bin/tokensave`.
- Old `mcp-tools-*-docker` wrappers were removed. If a client still references them, re-run mcp-config from the web panel (`/settings` → "Re-run mcp-config" = `POST /api/mcp-config/sync`). The `mcp-tools` CLI has no `mcp-config` subcommand.
- Persistent data: `$HOME/mcp-tools-data/` (subdirs `mem0`, `ollama`, plus `state.json`). Per-project serena state: `<project>/.serena/`.

## OMP tool discovery

OMP may expose MCP servers as discoverable tools instead of preloading them. If an `mcp__...` tool is not visible, do NOT declare it unavailable: call `search_tool_bm25` with the capability as query (e.g. `serena find symbol activate project`, `tokensave context code exploration`, `mem0 add memory remember preference`), then call the activated tool. Do not fall back to bash, Docker, Python, or host binaries for normal MCP tasks unless explicitly debugging MCP setup.

<!-- rtk-instructions v2 -->
# RTK (Rust Token Killer) - Token-Optimized Commands

## Golden Rule

**Always prefix commands with `rtk`**. If RTK has a dedicated filter, it uses it. If not, it passes through unchanged. This means RTK is always safe to use.

**Important**: Even in command chains with `&&`, use `rtk`:
```bash
# ❌ Wrong
git add . && git commit -m "msg" && git push

# ✅ Correct
rtk git add . && rtk git commit -m "msg" && rtk git push
```

## RTK Commands by Workflow

### Build & Compile (80-90% savings)
```bash
rtk cargo build         # Cargo build output
rtk cargo check         # Cargo check output
rtk cargo clippy        # Clippy warnings grouped by file (80%)
rtk tsc                 # TypeScript errors grouped by file/code (83%)
rtk lint                # ESLint/Biome violations grouped (84%)
rtk prettier --check    # Files needing format only (70%)
rtk next build          # Next.js build with route metrics (87%)
```

### Test (60-99% savings)
```bash
rtk cargo test          # Cargo test failures only (90%)
rtk go test             # Go test failures only (90%)
rtk jest                # Jest failures only (99.5%)
rtk vitest              # Vitest failures only (99.5%)
rtk playwright test     # Playwright failures only (94%)
rtk pytest              # Python test failures only (90%)
rtk rake test           # Ruby test failures only (90%)
rtk rspec               # RSpec test failures only (60%)
rtk test <cmd>          # Generic test wrapper - failures only
```

### Git (59-80% savings)
```bash
rtk git status          # Compact status
rtk git log             # Compact log (works with all git flags)
rtk git diff            # Compact diff (80%)
rtk git show            # Compact show (80%)
rtk git add             # Ultra-compact confirmations (59%)
rtk git commit          # Ultra-compact confirmations (59%)
rtk git push            # Ultra-compact confirmations
rtk git pull            # Ultra-compact confirmations
rtk git branch          # Compact branch list
rtk git fetch           # Compact fetch
rtk git stash           # Compact stash
rtk git worktree        # Compact worktree
```

Note: Git passthrough works for ALL subcommands, even those not explicitly listed.

### GitHub (26-87% savings)
```bash
rtk gh pr view <num>    # Compact PR view (87%)
rtk gh pr checks        # Compact PR checks (79%)
rtk gh run list         # Compact workflow runs (82%)
rtk gh issue list       # Compact issue list (80%)
rtk gh api              # Compact API responses (26%)
```

### JavaScript/TypeScript Tooling (70-90% savings)
```bash
rtk pnpm list           # Compact dependency tree (70%)
rtk pnpm outdated       # Compact outdated packages (80%)
rtk pnpm install        # Compact install output (90%)
rtk npm run <script>    # Compact npm script output
rtk npx <cmd>           # Compact npx command output
rtk prisma              # Prisma without ASCII art (88%)
```

### Files & Search (60-75% savings)
```bash
rtk ls <path>           # Tree format, compact (65%)
rtk read <file>         # Code reading with filtering (60%)
rtk grep <pattern>      # Search grouped by file (75%). Format flags (-c, -l, -L, -o, -Z) run raw.
rtk find <pattern>      # Find grouped by directory (70%)
```

### Analysis & Debug (70-90% savings)
```bash
rtk err <cmd>           # Filter errors only from any command
rtk log <file>          # Deduplicated logs with counts
rtk json <file>         # JSON structure without values
rtk deps                # Dependency overview
rtk env                 # Environment variables compact
rtk summary <cmd>       # Smart summary of command output
rtk diff                # Ultra-compact diffs
```

### Infrastructure (85% savings)
```bash
rtk docker ps           # Compact container list
rtk docker images       # Compact image list
rtk docker logs <c>     # Deduplicated logs
rtk kubectl get         # Compact resource list
rtk kubectl logs        # Deduplicated pod logs
```

### Network (65-70% savings)
```bash
rtk curl <url>          # Compact HTTP responses (70%)
rtk wget <url>          # Compact download output (65%)
```

### Meta Commands
```bash
rtk gain                # View token savings statistics
rtk gain --history      # View command history with savings
rtk discover            # Analyze Claude Code sessions for missed RTK usage
rtk proxy <cmd>         # Run command without filtering (for debugging)
rtk init                # Add RTK instructions to CLAUDE.md
rtk init --global       # Add RTK to ~/.claude/CLAUDE.md
```

## Token Savings Overview

| Category | Commands | Typical Savings |
|----------|----------|-----------------|
| Tests | vitest, playwright, cargo test | 90-99% |
| Build | next, tsc, lint, prettier | 70-87% |
| Git | status, log, diff, add, commit | 59-80% |
| GitHub | gh pr, gh run, gh issue | 26-87% |
| Package Managers | pnpm, npm, npx | 70-90% |
| Files | ls, read, grep, find | 60-75% |
| Infrastructure | docker, kubectl | 85% |
| Network | curl, wget | 65-70% |

Overall average: **60-90% token reduction** on common development operations.
<!-- /rtk-instructions -->
