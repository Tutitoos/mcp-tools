package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUpdateEnvPreservesDollarSigns reproduces B11: regexp.ReplaceAllString
// interprets `$1`, `$name`, `$$` etc. in the REPLACEMENT text as backrefs.
// Any value containing `$` (e.g. `$foo`, referencing another shell var) got
// silently mangled on write. ReplaceAllLiteralString must be used instead.
func TestUpdateEnvPreservesDollarSigns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("KEY=old\nOTHER=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := UpdateEnv(path, map[string]string{"KEY": "$foo"}); err != nil {
		t.Fatalf("UpdateEnv: %v", err)
	}

	got, err := LoadEnv(path)
	if err != nil {
		t.Fatalf("LoadEnv: %v", err)
	}
	if got["KEY"] != "$foo" {
		t.Errorf("KEY = %q, want literal %q (regexp backref expansion mangled it)", got["KEY"], "$foo")
	}
	if got["OTHER"] != "1" {
		t.Errorf("OTHER = %q, want %q (unrelated line was disturbed)", got["OTHER"], "1")
	}

	// A value containing a capture-group-shaped backref ($1) and a literal
	// $$ must also round-trip untouched.
	if err := UpdateEnv(path, map[string]string{"KEY": `$1-$$-$name`}); err != nil {
		t.Fatalf("UpdateEnv (backref-shaped value): %v", err)
	}
	got, err = LoadEnv(path)
	if err != nil {
		t.Fatalf("LoadEnv: %v", err)
	}
	if want := `$1-$$-$name`; got["KEY"] != want {
		t.Errorf("KEY = %q, want literal %q", got["KEY"], want)
	}
}

// TestUpdateEnvRejectsInvalidKeys reproduces F2 (auditoría 2026-07-11):
// regexp.QuoteMeta escapes regex metacharacters but does NOT neutralise an
// embedded newline, so a key like "SAFE\nINJECTED" appended "INJECTED=x" as
// a brand-new valid env var. UpdateEnv must reject the whole update.
func TestUpdateEnvRejectsInvalidKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	original := "HOST_HOME=/home/test\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"SAFE\nINJECTED", "lower", "1BAD", "A-B", "", "A B"} {
		if err := UpdateEnv(path, map[string]string{key: "x"}); err == nil {
			t.Errorf("UpdateEnv accepted invalid key %q", key)
		}
	}

	// The file must be byte-identical: a rejected update writes nothing.
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Errorf("file mutated by rejected update:\n%s", got)
	}

	// Sanity: a valid key still works.
	if err := UpdateEnv(path, map[string]string{"NEW_KEY": "v"}); err != nil {
		t.Fatalf("UpdateEnv (valid key): %v", err)
	}
	envs, err := LoadEnv(path)
	if err != nil {
		t.Fatal(err)
	}
	if envs["NEW_KEY"] != "v" {
		t.Errorf("NEW_KEY = %q, want %q", envs["NEW_KEY"], "v")
	}
}
