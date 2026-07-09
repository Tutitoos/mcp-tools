package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

// writePluginPkg writes <root>/plugins/<name>/package.json with the given
// content, creating parent directories as needed.
func writePluginPkg(t *testing.T, root, name, content string) {
	t.Helper()
	dir := filepath.Join(root, "plugins", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeLockfile writes <home>/.omp/plugins/omp-plugins.lock.json.
func writeLockfile(t *testing.T, home, content string) {
	t.Helper()
	dir := filepath.Join(home, ".omp", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "omp-plugins.lock.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestScan covers Scan's directory-walk, package.json field precedence,
// and lockfile-merge behavior via isolated MCP_TOOLS_ROOT/HOME env vars
// per subtest.
func TestScan(t *testing.T) {
	t.Run("no-plugins-dir", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("MCP_TOOLS_ROOT", root)
		t.Setenv("HOME", home)

		views, err := Scan()
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(views) != 0 {
			t.Errorf("len(views) = %d, want 0", len(views))
		}
	})

	t.Run("empty-plugins-dir", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("MCP_TOOLS_ROOT", root)
		t.Setenv("HOME", home)
		if err := os.MkdirAll(filepath.Join(root, "plugins"), 0o755); err != nil {
			t.Fatal(err)
		}

		views, err := Scan()
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(views) != 0 {
			t.Errorf("len(views) = %d, want 0", len(views))
		}
	})

	t.Run("one-valid-unlinked", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("MCP_TOOLS_ROOT", root)
		t.Setenv("HOME", home)
		writePluginPkg(t, root, "foo", `{"name":"foo","version":"0.1.0","description":"hi","omp":{"extensions":["src/e.ts"]}}`)

		views, err := Scan()
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(views) != 1 {
			t.Fatalf("len(views) = %d, want 1", len(views))
		}
		v := views[0]
		if v.Linked || v.Enabled {
			t.Errorf("Linked=%v Enabled=%v, want both false", v.Linked, v.Enabled)
		}
		if want := []string{"src/e.ts"}; len(v.Extensions) != 1 || v.Extensions[0] != want[0] {
			t.Errorf("Extensions = %v, want %v", v.Extensions, want)
		}
		if v.Description != "hi" {
			t.Errorf("Description = %q, want %q", v.Description, "hi")
		}
	})

	t.Run("omp-description-wins", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("MCP_TOOLS_ROOT", root)
		t.Setenv("HOME", home)
		writePluginPkg(t, root, "foo", `{"name":"foo","description":"from-root","omp":{"description":"from-omp"}}`)

		views, err := Scan()
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(views) != 1 {
			t.Fatalf("len(views) = %d, want 1", len(views))
		}
		if views[0].Description != "from-omp" {
			t.Errorf("Description = %q, want %q", views[0].Description, "from-omp")
		}
	})

	t.Run("linked-and-enabled", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("MCP_TOOLS_ROOT", root)
		t.Setenv("HOME", home)
		writePluginPkg(t, root, "foo", `{"name":"foo"}`)
		writeLockfile(t, home, `{"plugins":{"foo":{"enabled":true}}}`)

		views, err := Scan()
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(views) != 1 {
			t.Fatalf("len(views) = %d, want 1", len(views))
		}
		if !views[0].Linked || !views[0].Enabled {
			t.Errorf("Linked=%v Enabled=%v, want both true", views[0].Linked, views[0].Enabled)
		}
	})

	t.Run("linked-and-disabled", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("MCP_TOOLS_ROOT", root)
		t.Setenv("HOME", home)
		writePluginPkg(t, root, "foo", `{"name":"foo"}`)
		writeLockfile(t, home, `{"plugins":{"foo":{"enabled":false}}}`)

		views, err := Scan()
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(views) != 1 {
			t.Fatalf("len(views) = %d, want 1", len(views))
		}
		if !views[0].Linked {
			t.Errorf("Linked = %v, want true", views[0].Linked)
		}
		if views[0].Enabled {
			t.Errorf("Enabled = %v, want false", views[0].Enabled)
		}
	})

	t.Run("malformed-package-json-skipped", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("MCP_TOOLS_ROOT", root)
		t.Setenv("HOME", home)
		writePluginPkg(t, root, "foo", `{"name":"foo"}`)
		writePluginPkg(t, root, "bad", `{not valid`)

		views, err := Scan()
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(views) != 1 || views[0].Name != "foo" {
			t.Fatalf("views = %+v, want exactly [foo]", views)
		}
	})

	t.Run("malformed-lockfile-treated-as-empty", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("MCP_TOOLS_ROOT", root)
		t.Setenv("HOME", home)
		writePluginPkg(t, root, "foo", `{"name":"foo"}`)
		writeLockfile(t, home, `{not valid`)

		views, err := Scan()
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(views) != 1 {
			t.Fatalf("len(views) = %d, want 1", len(views))
		}
		if views[0].Linked {
			t.Errorf("Linked = %v, want false", views[0].Linked)
		}
	})

	t.Run("sorted-alphabetically", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		t.Setenv("MCP_TOOLS_ROOT", root)
		t.Setenv("HOME", home)
		writePluginPkg(t, root, "zebra", `{"name":"zebra"}`)
		writePluginPkg(t, root, "alpha", `{"name":"alpha"}`)
		writePluginPkg(t, root, "mango", `{"name":"mango"}`)

		views, err := Scan()
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		var names []string
		for _, v := range views {
			names = append(names, v.Name)
		}
		want := []string{"alpha", "mango", "zebra"}
		if len(names) != len(want) {
			t.Fatalf("names = %v, want %v", names, want)
		}
		for i := range want {
			if names[i] != want[i] {
				t.Errorf("names[%d] = %q, want %q", i, names[i], want[i])
			}
		}
	})
}
