package golden

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequireEqual_createsFileOnUpdate(t *testing.T) {
	// Only run in update mode
	if !shouldUpdate() {
		t.Skip("not in update mode")
	}

	dir := t.TempDir()

	RequireEqualIn(t, dir, []byte("hello golden"))

	name := strings.ReplaceAll(t.Name(), "/", "_")
	path := filepath.Join(dir, name+".golden")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file not created: %v", err)
	}
	if string(data) != "hello golden" {
		t.Errorf("content = %q, want %q", string(data), "hello golden")
	}
}
