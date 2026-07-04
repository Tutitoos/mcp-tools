# Headroom MCP

Use this skill when the user mentions Headroom, headroom, compress text, reduce context, save tokens, compress logs, compress JSON, compress output, retrieve compressed content, or Headroom stats.

## Mandatory rule

Use the MCP tools directly.

Available MCP tools:
- `headroom_compress`
- `headroom_retrieve`
- `headroom_stats`

Do not use shell, Docker, Python package internals, or CLI discovery for normal Headroom tasks.

Forbidden for normal usage:
- `which headroom`
- `headroom --help`
- `headroom mcp --help`
- `headroom mcp serve`
- `docker exec ... headroom`
- `docker exec ... python`
- importing `headroom` from Python
- reading Headroom package source
- searching for Headroom files
- creating synthetic expanded test inputs unless the user explicitly asks

## Compress

When the user asks to compress content with Headroom:

Call `headroom_compress` with:

```json
{
  "content": "<exact user content>"
}
```

Use the exact content the user provided. Do not rewrite it, expand it, replicate it, or generate a larger test unless explicitly requested.

Then report:
- whether output changed
- tokens saved / savings if returned
- transform or passthrough if returned
- hash if returned

If the result is passthrough or 0 savings, explain briefly:
- content may be too small
- content may be protected, especially error logs
- compression would not save tokens

Do not run extra experiments to prove Headroom works elsewhere.

## Retrieve

When the user asks to retrieve original content from a Headroom hash:

Call `headroom_retrieve` with:

```json
{
  "hash": "<hash>"
}
```

Do not search files first.

## Stats

When the user asks for Headroom savings, stats, usage, or compression history:

Call `headroom_stats`.

## If MCP tools are not visible

If `headroom_compress`, `headroom_retrieve`, and `headroom_stats` are not available in the current session, stop.

Ask the user to run:

```text
/mcp list
/mcp test headroom
/mcp reload
/mcp reconnect headroom
```

Expected result:

```text
headroom connected [stdio]
Tools:
- headroom_compress
- headroom_retrieve
- headroom_stats
```

Do not fall back to Docker or Python unless the user explicitly asks for low-level debugging.

## Expected runtime

Configured command:

```text
/home/tutitoos/.local/bin/headroom-mcp-docker
```

Container:

```text
mcp-custom-headroom-mcp
```

This runtime is already managed by Docker. Normal user tasks must use MCP tools, not direct container commands.

## Good inputs

Headroom works best on large or repetitive content:
- long logs
- large JSON
- grep/find output
- command output
- file listings
- long tool responses
- long documentation snippets

Headroom may not compress:
- one sentence
- tiny text
- already concise text
- protected diagnostic/error output
- content where compression would add more tokens than it saves

## Output discipline

For Headroom compression requests:
- Use the exact user-provided content.
- Do not generate a bigger synthetic sample.
- Do not run multiple experimental variants.
- Do not inspect Headroom source code.
- Do not explain internal implementation unless the user asks.
- Keep the answer focused on the compression result.

## Example

User:
Compress this with Headroom:

```text
<large log>
```

Assistant:
Call `headroom_compress` with the full log as `content`.

Then respond with:
- compressed output or passthrough
- savings/tokens if returned
- retrieval hash if returned
- a short explanation if no compression happened
