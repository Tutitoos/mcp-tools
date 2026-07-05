// Package state persists the set of components a user chose in the
// multi-select installer TUI, plus the versions we knew about after the last
// install/upgrade. `$MCP_TOOLS_DATA/state.json` is the canonical location.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Tutitoos/mcp-tools/internal/config"
)

// SchemaVersion is bumped when the on-disk shape changes incompatibly.
const SchemaVersion = 1

// State is the on-disk shape at $MCP_TOOLS_DATA/state.json.
type State struct {
	Version   int               `json:"version"`
	Selected  []string          `json:"selected"`
	Versions  map[string]string `json:"versions,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Path returns the canonical state file path (respects $MCP_TOOLS_DATA).
func Path() string {
	return filepath.Join(config.DataDir(), "state.json")
}

// Load reads state from Path(). A missing file is not an error — it returns
// the zero State (SchemaVersion, empty Selected). JSON errors surface.
func Load() (State, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{Version: SchemaVersion}, nil
		}
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, fmt.Errorf("state.json: %w", err)
	}
	if s.Version == 0 {
		s.Version = SchemaVersion
	}
	if s.Versions == nil {
		s.Versions = map[string]string{}
	}
	return s, nil
}

// Save writes the state atomically (tempfile + rename). UpdatedAt is set to
// time.Now() before serialising.
func (s *State) Save() error {
	s.Version = SchemaVersion
	s.UpdatedAt = time.Now().UTC()
	if s.Versions == nil {
		s.Versions = map[string]string{}
	}
	if s.Selected == nil {
		s.Selected = []string{}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Has reports whether key is currently selected.
func (s State) Has(key string) bool {
	return slices.Contains(s.Selected, key)
}

// WithSelected returns a copy of s with Selected replaced by keys.
func (s State) WithSelected(keys []string) State {
	out := s
	out.Selected = append([]string(nil), keys...)
	return out
}
