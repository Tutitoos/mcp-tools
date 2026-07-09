// Package plugins scans <RepoRoot>/plugins/*/package.json (workspace-local
// OMP plugin packages) and merges each entry with its enabled/linked state
// from $HOME/.omp/plugins/omp-plugins.lock.json. Read-only: the CLI (or
// internal/web/plugins.go shelling `omp plugin …`) is the only writer of
// the lockfile.
package plugins

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

// View is one workspace plugin annotated with its OMP-side state.
type View struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Path        string   `json:"path"`       // absolute
	Extensions  []string `json:"extensions"` // from package.json.omp.extensions
	Linked      bool     `json:"linked"`     // present in lockfile.plugins
	Enabled     bool     `json:"enabled"`    // lockfile entry .enabled; false when !Linked
}

// pkg is the subset of package.json fields Scan needs; everything else is
// ignored by the decoder.
type pkg struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Omp         struct {
		Description string   `json:"description"`
		Extensions  []string `json:"extensions"`
	} `json:"omp"`
	Pi struct { // fallback, same order OMP itself uses (omp || pi)
		Description string   `json:"description"`
		Extensions  []string `json:"extensions"`
	} `json:"pi"`
}

// lockfile is the subset of $HOME/.omp/plugins/omp-plugins.lock.json Scan
// needs to determine linked/enabled state.
type lockfile struct {
	Plugins map[string]struct {
		Enabled bool `json:"enabled"`
	} `json:"plugins"`
}

// Scan reads <config.PluginsDir()> and returns one View per subdirectory
// that contains a readable, well-formed package.json. Silently skips
// subdirectories whose package.json is missing/unreadable/malformed (they
// might be scratch dirs or half-initialised); the caller sees only valid
// entries. Missing plugins dir → nil, nil.
//
// The lockfile at <config.OmpPluginsLockfile()> is merged on top: for each
// workspace plugin whose `package.json.name` matches a key under
// lockfile.plugins, `Linked=true` and `Enabled=<entry.enabled>`. Missing or
// malformed lockfile → treated as empty (all entries Linked=false), no error.
//
// Return order: alphabetical by Name (for stable UI rendering).
func Scan() ([]View, error) {
	root := config.PluginsDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	lf := readLockfile(config.OmpPluginsLockfile())

	var views []View
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		p, ok := readPackageJSON(filepath.Join(dir, "package.json"))
		if !ok || p.Name == "" {
			continue
		}

		description := p.Omp.Description
		if description == "" {
			description = p.Pi.Description
		}
		if description == "" {
			description = p.Description
		}

		extensions := p.Omp.Extensions
		if len(extensions) == 0 {
			extensions = p.Pi.Extensions
		}
		if extensions == nil {
			extensions = []string{}
		}

		v := View{
			Name:        p.Name,
			Version:     p.Version,
			Description: description,
			Path:        dir,
			Extensions:  extensions,
		}
		if lfEntry, ok := lf.Plugins[p.Name]; ok {
			v.Linked = true
			v.Enabled = lfEntry.Enabled
		}
		views = append(views, v)
	}

	sort.Slice(views, func(i, j int) bool { return views[i].Name < views[j].Name })
	return views, nil
}

// readPackageJSON reads and decodes a single package.json. ok=false on any
// I/O or parse error (missing/unreadable/malformed) — the caller treats
// that subdirectory as not a plugin.
func readPackageJSON(path string) (pkg, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return pkg{}, false
	}
	var p pkg
	if err := json.Unmarshal(data, &p); err != nil {
		return pkg{}, false
	}
	return p, true
}

// readLockfile reads and decodes the OMP plugins lockfile. Missing or
// malformed → empty lockfile (all workspace plugins report Linked=false).
func readLockfile(path string) lockfile {
	data, err := os.ReadFile(path)
	if err != nil {
		return lockfile{}
	}
	var lf lockfile
	if err := json.Unmarshal(data, &lf); err != nil {
		log.Printf("plugins: lockfile parse: %v", err)
		return lockfile{}
	}
	return lf
}
