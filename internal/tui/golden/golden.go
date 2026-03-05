// Package golden provides golden file test assertions for TUI output.
//
// Set UPDATE_GOLDEN=1 to regenerate golden files:
//
//	UPDATE_GOLDEN=1 go test ./internal/tui/...
package golden

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const defaultTestdataDir = "testdata"

func shouldUpdate() bool {
	return os.Getenv("UPDATE_GOLDEN") != ""
}

// RequireEqual compares actual output against a golden file derived from t.Name().
// Set UPDATE_GOLDEN=1 to write actual output to the golden file instead.
func RequireEqual(t *testing.T, actual []byte) {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	requireGolden(t, defaultTestdataDir, name, actual)
}

// RequireEqualNamed compares actual output against a golden file with a custom name.
func RequireEqualNamed(t *testing.T, name string, actual []byte) {
	t.Helper()
	safeName := strings.ReplaceAll(name, "/", "_")
	requireGolden(t, defaultTestdataDir, safeName, actual)
}

// RequireEqualIn compares actual output against a golden file in the given directory.
func RequireEqualIn(t *testing.T, dir string, actual []byte) {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	requireGolden(t, dir, name, actual)
}

func requireGolden(t *testing.T, dir, name string, actual []byte) {
	t.Helper()

	path := filepath.Join(dir, name+".golden")

	if shouldUpdate() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create testdata dir: %v", err)
		}
		if err := os.WriteFile(path, actual, 0o644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		return
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file %s not found. Run with UPDATE_GOLDEN=1 to create it:\n  UPDATE_GOLDEN=1 go test ./... -run %s", path, name)
	}

	if string(actual) != string(expected) {
		t.Fatalf("output differs from golden file %s.\n"+
			"Run with UPDATE_GOLDEN=1 to regenerate:\n  UPDATE_GOLDEN=1 go test ./... -run %s\n\n"+
			"--- expected (golden) ---\n%s\n--- actual ---\n%s",
			path, name, string(expected), string(actual))
	}
}
