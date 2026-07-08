package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteJSONAtomic verifies the atomicity + permissions contract of WriteJSON:
//  1. The target file is created with valid JSON content.
//  2. The mode is 0o600 (private — configs of MCP clients may carry tokens).
//  3. No .tmp file is left behind on the happy path.
func TestWriteJSONAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	payload := map[string]any{"k": "v", "n": float64(42)}
	if err := WriteJSON(path, payload); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	// 1. exists + parses
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got["k"] != "v" {
		t.Fatalf("content mismatch: %v", got)
	}

	// 2. mode 0o600
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := st.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm = %o, want 0o600", perm)
	}

	// 3. no .tmp leftover
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("expected no .tmp file, stat err = %v", err)
	}
}

// TestWriteJSONOverwrite verifies that an existing file is replaced atomically.
func TestWriteJSONOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	first := map[string]any{"v": 1}
	if err := WriteJSON(path, first); err != nil {
		t.Fatalf("first WriteJSON: %v", err)
	}
	second := map[string]any{"v": 2}
	if err := WriteJSON(path, second); err != nil {
		t.Fatalf("second WriteJSON: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got["v"] != float64(2) {
		t.Fatalf("expected v=2, got %v", got["v"])
	}
}
