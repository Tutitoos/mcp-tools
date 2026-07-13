package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// envLine matches KEY=VALUE lines with a valid shell-style key.
var envLine = regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)=(.*)$`)

// envKey matches a valid shell-style variable name. UpdateEnv rejects any
// other key: regexp.QuoteMeta escapes regex metacharacters but does NOT
// neutralise an embedded newline, so an unvalidated key like "SAFE\nINJECTED"
// would let a caller inject arbitrary KEY=VALUE lines into the file.
var envKey = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// LoadEnv parses a .env-style file. Missing file returns an empty map and no error.
func LoadEnv(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		m := envLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		out[m[1]] = m[2]
	}
	return out, nil
}

// UpdateEnv rewrites lines matching keys in `updates`, appending any missing key.
// Every key must match envKey; the whole update is rejected otherwise. Returns
// os.ErrNotExist if the file is missing (idempotent replacement is not defined without a base).
func UpdateEnv(path string, updates map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(data)
	for k := range updates {
		if !envKey.MatchString(k) {
			return fmt.Errorf("clave de entorno inválida %q (debe cumplir ^[A-Z_][A-Z0-9_]*$)", k)
		}
	}
	for k, v := range updates {
		re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(k) + `=.*$`)
		newLine := k + "=" + v
		if re.MatchString(text) {
			text = re.ReplaceAllLiteralString(text, newLine)
		} else {
			if !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			text += newLine + "\n"
		}
	}
	return os.WriteFile(path, []byte(text), 0o600)
}

// WriteEnv writes a full .env with the exact key order + blank-line separators produced
// by the legacy scripts/init-env.sh. Only the keys defined in initEnvOrder are written;
// any extra keys in `contents` are appended at the end. Keys missing from `contents`
// are skipped silently.
func WriteEnv(path string, contents map[string]string) error {
	// Exact layout: matches scripts/init-env.sh lines 10-24 (blank lines between groups).
	groups := [][]string{
		{"HOST_HOME", "HOST_UID", "HOST_GID"},
		{"MCP_TOOLS_ROOT", "MCP_TOOLS_DATA", "MCP_TOOLS_BIND"},
		{"MEM0_USER_ID"},
	}
	written := map[string]bool{}
	var b strings.Builder
	for i, group := range groups {
		if i > 0 {
			b.WriteByte('\n')
		}
		for _, k := range group {
			v, ok := contents[k]
			if !ok {
				continue
			}
			fmt.Fprintf(&b, "%s=%s\n", k, v)
			written[k] = true
		}
	}
	// Append any extras (not in groups) to preserve them if caller passed more.
	for k, v := range contents {
		if written[k] {
			continue
		}
		fmt.Fprintf(&b, "%s=%s\n", k, v)
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}
