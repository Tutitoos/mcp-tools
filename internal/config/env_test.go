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
