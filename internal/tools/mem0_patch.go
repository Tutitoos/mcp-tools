package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// mem0-mcp-selfhosted v0.3.2 (pin a4f538a) passes user_id/agent_id/run_id as
// top-level kwargs to mem0ai's search()/get_all(), but mem0ai >= 2.0.0
// rejects those ("Top-level entity parameters ... are not supported. Use
// filters=...") — search_memories and get_memories fail on every call while
// add/get/delete keep working. Upstream has been inactive since the 0.3.2
// release (2026-03-13), so mcp-tools patches the two call sites
// post-install instead of downgrading mem0ai (a 2.x -> 1.x downgrade risks
// the memories already persisted in qdrant).
//
// Idempotent: an already-patched server.py is recognized and skipped. If
// NEITHER the original NOR the patched block is found, the pinned source
// drifted (pin bump without re-reviewing this patch) — fail loudly.
var mem0EntityFilterPatches = []struct {
	name    string
	orig    string
	patched string
}{
	{
		name: "search_memories",
		orig: `        kwargs: dict[str, Any] = {"user_id": uid, "query": query}
        if agent_id:
            kwargs["agent_id"] = agent_id
        if run_id:
            kwargs["run_id"] = run_id
        if filters:
            kwargs["filters"] = filters
        if limit is not None:
            kwargs["limit"] = limit
        if threshold is not None:
            kwargs["threshold"] = threshold
        if rerank is not None:
            kwargs["rerank"] = rerank
`,
		patched: `        filt: dict[str, Any] = dict(filters) if filters else {}
        filt.setdefault("user_id", uid)
        if agent_id:
            filt["agent_id"] = agent_id
        if run_id:
            filt["run_id"] = run_id
        kwargs: dict[str, Any] = {"query": query, "filters": filt}
        if limit is not None:
            kwargs["top_k"] = limit
        if threshold is not None:
            kwargs["threshold"] = threshold
        if rerank is not None:
            kwargs["rerank"] = rerank
`,
	},
	{
		name: "get_memories",
		orig: `        kwargs: dict[str, Any] = {"user_id": uid}
        if agent_id:
            kwargs["agent_id"] = agent_id
        if run_id:
            kwargs["run_id"] = run_id
        if limit is not None:
            kwargs["limit"] = limit
`,
		patched: `        filt: dict[str, Any] = {"user_id": uid}
        if agent_id:
            filt["agent_id"] = agent_id
        if run_id:
            filt["run_id"] = run_id
        kwargs: dict[str, Any] = {"filters": filt}
        if limit is not None:
            kwargs["top_k"] = limit
`,
	},
}

// mem0ServerPy locates the installed server.py inside the uv tool venv.
// The python3.* segment is globbed so a future uv/python bump keeps working.
func mem0ServerPy(home string) (string, error) {
	pattern := filepath.Join(home, ".local", "share", "uv", "tools",
		"mem0-mcp-selfhosted", "lib", "python3.*", "site-packages",
		"mem0_mcp_selfhosted", "server.py")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("mem0 server.py no encontrado (%s) — ¿instalación uv incompleta?", pattern)
	}
	return matches[0], nil
}

// patchMem0EntityFilters rewrites the two broken call sites in the installed
// server.py. Runs after every install/upgrade (both reinstall from the pin,
// so the pristine source is always what this patch expects).
func patchMem0EntityFilters(home string, log func(string)) error {
	path, err := mem0ServerPy(home)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	src := string(raw)
	changed := false
	for _, p := range mem0EntityFilterPatches {
		switch {
		case strings.Contains(src, p.patched):
			log(fmt.Sprintf("  parche mem0 %s: ya aplicado", p.name))
		case strings.Contains(src, p.orig):
			src = strings.Replace(src, p.orig, p.patched, 1)
			changed = true
			log(fmt.Sprintf("  parche mem0 %s: aplicado (entity params -> filters)", p.name))
		default:
			return fmt.Errorf("parche mem0 %s: el código fuente no coincide con el pin %s — revisa el parche tras el bump", p.name, mem0GitURL)
		}
	}
	if !changed {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(src), info.Mode().Perm())
}
