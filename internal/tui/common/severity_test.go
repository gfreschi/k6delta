package common_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/common"
)

func TestSeverityFromRatio(t *testing.T) {
	tests := []struct {
		name  string
		ratio float64
		want  common.Severity
	}{
		{"below warn", 0.50, common.SeverityOK},
		{"at warn boundary", 0.80, common.SeverityWarn},
		{"between warn and error", 0.90, common.SeverityWarn},
		{"at error boundary", 0.95, common.SeverityError},
		{"above error", 0.99, common.SeverityError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.SeverityFromRatio(tt.ratio, common.DefaultSeverityThresholds)
			if got != tt.want {
				t.Errorf("SeverityFromRatio(%v) = %v, want %v", tt.ratio, got, tt.want)
			}
		})
	}
}

func TestSeverityString(t *testing.T) {
	tests := []struct {
		sev  common.Severity
		want string
	}{
		{common.SeverityOK, "ok"},
		{common.SeverityWarn, "warn"},
		{common.SeverityError, "error"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.sev.String(); got != tt.want {
				t.Errorf("Severity.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
