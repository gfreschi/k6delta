package runtui

import (
	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/verdict"
)

// computeVerdict delegates to the shared verdict package.
func computeVerdict(in verdict.Input, cfg config.VerdictConfig) verdict.Result {
	return verdict.Compute(in, cfg)
}
