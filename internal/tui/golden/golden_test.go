package golden

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRequireEqual_createsFileOnUpdate(t *testing.T) {
	// Only run in update mode
	if !shouldUpdate() {
		t.Skip("not in update mode")
	}

	dir := t.TempDir()
	original := testdataDir
	testdataDir = dir
	defer func() { testdataDir = original }()

	RequireEqual(t, []byte("hello golden"))

	path := filepath.Join(dir, t.Name()+".golden")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file not created: %v", err)
	}
	if string(data) != "hello golden" {
		t.Errorf("content = %q, want %q", string(data), "hello golden")
	}
}
