package mcp

import "testing"

func TestClaudeServerSpecOmitsEmptyArgs(t *testing.T) {
	spec := claudeServerSpec(ServerSpec{
		Name:    "mcp_tools_redis",
		Wrapper: "redis-mcp-server",
	}, "/home/test")

	if _, ok := spec["args"]; ok {
		t.Fatal("empty args must be omitted; Claude rejects args: null")
	}
	if got := spec["command"]; got != "redis-mcp-server" {
		t.Fatalf("command = %v, want redis-mcp-server", got)
	}
	env, ok := spec["env"].(map[string]string)
	if !ok || env["HOME"] != "/home/test" {
		t.Fatalf("environment = %#v, want HOME=/home/test", spec["env"])
	}
}
