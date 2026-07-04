package mcp

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LoadJSON parses `path` as JSON. If missing, returns fallback marshalled/unmarshalled
// through json to normalise its shape into map[string]any.
func LoadJSON(path string, fallback any) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return toMap(fallback)
		}
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func toMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// WriteJSON pretty-writes obj to path with a trailing newline. Creates parents.
func WriteJSON(path string, obj any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// Backup copies path to path.bak.<timestamp> if it exists. Silent no-op if missing.
// Timestamp format matches scripts/install-mcp.ts isoStamp: 2026-01-02T15-04-05-000Z.
func Backup(path string) error {
	src, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer src.Close()

	stamp := time.Now().UTC().Format("2006-01-02T15-04-05.000Z")
	stamp = strings.ReplaceAll(stamp, ".", "-")
	dst, err := os.Create(path + ".bak." + stamp)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
