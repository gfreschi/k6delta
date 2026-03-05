package k6_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/k6"
)

func TestSupportsJSONStreaming(t *testing.T) {
	supported, err := k6.SupportsJSONStreaming()
	if err != nil {
		t.Skipf("k6 not available: %v", err)
	}
	if !supported {
		t.Error("expected k6 to support JSON streaming")
	}
}
