package orchestrator

import "regexp"

// ruleBlockRE is a package-level var so the regex isn't recompiled per call.
var ruleBlockRE = func() *regexp.Regexp {
	return regexp.MustCompile(`(?ms)^<!-- mcp-tools:start -->.*?^<!-- mcp-tools:end -->\n?`)
}()