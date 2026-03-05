package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gfreschi/k6delta/internal/config"
)

func TestLoadVerdictConfig(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		wantCPUWarn float64
		wantCPUFail float64
		want5xxWarn int
		want5xxFail int
	}{
		{
			name:        "defaults when missing",
			yaml:        "provider: ecs\napps:\n  api:\n    cluster: c\n    service: s\n    test_file: t.js\n",
			wantCPUWarn: 90.0,
			wantCPUFail: 98.0,
			want5xxWarn: 1,
			want5xxFail: 10,
		},
		{
			name:        "custom overrides",
			yaml:        "provider: ecs\nverdicts:\n  cpu_peak_warn: 85\n  cpu_peak_fail: 95\n  error_5xx_warn: 5\n  error_5xx_fail: 20\napps:\n  api:\n    cluster: c\n    service: s\n    test_file: t.js\n",
			wantCPUWarn: 85.0,
			wantCPUFail: 95.0,
			want5xxWarn: 5,
			want5xxFail: 20,
		},
		{
			name:        "partial overrides use defaults",
			yaml:        "provider: ecs\nverdicts:\n  cpu_peak_warn: 80\napps:\n  api:\n    cluster: c\n    service: s\n    test_file: t.js\n",
			wantCPUWarn: 80.0,
			wantCPUFail: 98.0,
			want5xxWarn: 1,
			want5xxFail: 10,
		},
		{
			name:        "explicit zero is honored",
			yaml:        "provider: ecs\nverdicts:\n  cpu_peak_warn: 0\n  error_5xx_warn: 0\napps:\n  api:\n    cluster: c\n    service: s\n    test_file: t.js\n",
			wantCPUWarn: 0.0,
			wantCPUFail: 98.0,
			want5xxWarn: 0,
			want5xxFail: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "k6delta.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0o644); err != nil {
				t.Fatal(err)
			}

			cfg, err := config.Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			vc := cfg.Verdicts.WithDefaults()
			if vc.CPUPeakWarn != tt.wantCPUWarn {
				t.Errorf("CPUPeakWarn = %v, want %v", vc.CPUPeakWarn, tt.wantCPUWarn)
			}
			if vc.CPUPeakFail != tt.wantCPUFail {
				t.Errorf("CPUPeakFail = %v, want %v", vc.CPUPeakFail, tt.wantCPUFail)
			}
			if vc.Error5xxWarn != tt.want5xxWarn {
				t.Errorf("Error5xxWarn = %v, want %v", vc.Error5xxWarn, tt.want5xxWarn)
			}
			if vc.Error5xxFail != tt.want5xxFail {
				t.Errorf("Error5xxFail = %v, want %v", vc.Error5xxFail, tt.want5xxFail)
			}
		})
	}
}
