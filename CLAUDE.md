# mcp-tools agent instructions

This repository ships an installer + registry for MCP servers and operational skills.

## Serena-first code exploration (READ THIS FIRST)

**`mcp_tools_serena` is the default for any code operation on a named symbol.** Use it for: finding a function/class/method/struct/constant, reading its body, listing who calls it, locating its declaration, getting a file's symbol outline, and editing a symbol's body. It is LSP-accurate, returns ~60% fewer tokens than `rtk grep` and ZERO false positives (no string/comment noise).

Workflow (EVERY session):
1. `activate_project("/absolute/path")` once per project. For this repo: `activate_project("/home/tutitoos/mcp-tools")`.
2. Then use `find_symbol` / `find_referencing_symbols` / `find_declaration` / `get_symbols_overview` / `find_symbol(include_body: true)` as the situation demands.

Default tool mapping (overrides "use native Read" instinct):
- "show me function X" / "cómo funciona X" / "muéstrame el cuerpo" → `find_symbol(name_path_pattern: "X", include_body: true)`
- "where is X used" / "quién llama a X" / "references of X" → `find_referencing_symbols(name_path_pattern: "X")`
- "where is X defined" / "declaración de X" → `find_declaration(name_path_pattern: "X")`
- "outline of file.go" / "symbols in file" → `get_symbols_overview(relative_path: "internal/.../file.go")`
- "rename X to Y" / "replace body of X" → `replace_symbol_body` / `rename_symbol` (LSP-accurate)

Fall back to native `Read` only for: raw config, docs, `.env`, logs, JSON dumps, or when the file is not in an LSP-indexable language. **Never** fall back to native `Read` to "see how function X works" — that is exactly what serena is for.

If serena returns an error or "not connected" → see the escalation list at the bottom of this file.

## Other MCP servers

Before working with local code repositories, read:

- `skills/codebase-memory/SKILL.md`
- `skills/serena/SKILL.md`
- `skills/tokensave/SKILL.md`
- `skills/mem0/SKILL.md`

Use `mcp_tools_serena` for symbol-level code operations (see block above — DEFAULT for any named symbol).

Use `tokensave` (`tokensave_context`) for natural-language exploration in a `tokensave init`'d project ("how does X work" open-ended questions).

Use `mcp_tools_codebase_memory` for cross-repo architecture, ADR, community detection, dependency graphs.

Use `mcp_tools_mem0` for persistent cross-session memory (facts, preferences, decisions). Always call `search_memories` before `add_memory` to avoid duplicates. NOTE: `search_memories` and `get_memories` are BROKEN upstream (lib mem0 API change) — see RULES.md "Known bugs" before relying on them.

Important rules:

- The MCP servers run as host binaries (not Docker). `codebase-memory-mcp` lives at `~/.local/bin/codebase-memory-mcp`; `mem0-mcp-selfhosted` runs behind the `~/.local/bin/mem0-launcher` wrapper (sources `.env.mem0`); `serena` lives at `~/.local/bin/serena` (installed from the mcp-tools web panel, `/tools` → serena → install, or `uv tool install -p 3.13 serena-agent`).
- Do not spawn old `mcp-tools-*-docker` wrappers — they were removed. If a client still references them, re-run mcp-config from the web panel (`/settings` → "Re-run mcp-config", i.e. `POST /api/mcp-config/sync`). The `mcp-tools` CLI no longer has a `mcp-config` subcommand.
- Persistent data lives under `$HOME/mcp-tools-data/` (subdirs per MCP: `mem0`, `ollama`, plus `state.json`). Per-project serena state lives at `<project>/.serena/`.
- NEVER fall back to native `Grep`/`Read`/`find`/`bash grep` for repo-wide code search — use serena (named symbol), tokensave (open question), or codebase-memory (cross-repo).
- NEVER use `rtk grep` to find references of a named symbol — `rtk grep` matches strings/comments, not symbol identity. Use serena.
- NEVER write local `notes.md`/scratchpad files to persist facts across sessions — that's what `mcp_tools_mem0` is for.

## Serena activation reminder

If you have not called `activate_project` for the current project in this session, the FIRST code-operation call you make must be `activate_project("/absolute/path")`. After that, all serena tools work without re-activation for the rest of the session. If you find yourself about to call `Read` on a `.go` / `.ts` / `.py` / `.rs` / `.java` file, stop and use serena instead.

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