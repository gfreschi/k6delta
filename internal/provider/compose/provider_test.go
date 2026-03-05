package compose_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/provider/compose"
)

func TestProviderImplementsInterface(t *testing.T) {
	var _ provider.InfraProvider = (*compose.Provider)(nil)
}
