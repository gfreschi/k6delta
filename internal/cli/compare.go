package cli

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/report"
	comparetui "github.com/gfreschi/k6delta/internal/tui/compare"
	"github.com/gfreschi/k6delta/internal/verdict"
)

// NewCompareCmd creates the "compare" subcommand.
func NewCompareCmd() *cobra.Command {
	var (
		jsonOutput bool
		ciMode     bool
		configFile string
	)

	cmd := &cobra.Command{
		Use:   "compare <report-a.json> <report-b.json>",
		Short: "Compare two unified reports side-by-side with percentage deltas",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathA := args[0]
			pathB := args[1]

			if jsonOutput && !ciMode {
				return comparetui.RunJSON(pathA, pathB)
			}

			if ciMode {
				cfg, err := loadConfig(configFile)
				if err != nil {
					return err
				}
				return compareCI(pathA, pathB, cfg.Verdicts.WithDefaults())
			}

			m := comparetui.NewModel(pathA, pathB)
			p := tea.NewProgram(m)
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("TUI error: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output JSON instead of TUI")
	cmd.Flags().BoolVar(&ciMode, "ci", false, "CI mode: JSON to stdout, exit code = regression verdict")
	cmd.Flags().StringVar(&configFile, "config", "", "config file path (default: k6delta.yaml)")

	return cmd
}

func compareCI(pathA, pathB string, vcfg config.VerdictConfig) error {
	comp, err := report.CompareReports(pathA, pathB)
	if err != nil {
		return err
	}

	level := verdict.Pass
	var reasons []string

	for _, row := range comp.K6Rows {
		pct := parseDeltaPct(row.Delta)
		if pct == 0 || row.Direction != "worse" {
			continue
		}
		absPct := math.Abs(pct)
		switch row.Metric {
		case "p95":
			if absPct >= vcfg.P95RegFail {
				level = verdict.Fail
				reasons = append(reasons, fmt.Sprintf("p95 regressed %.1f%% (threshold: %.0f%%)", absPct, vcfg.P95RegFail))
			} else if absPct >= vcfg.P95RegWarn {
				if level < verdict.Warn {
					level = verdict.Warn
				}
				reasons = append(reasons, fmt.Sprintf("p95 regressed %.1f%% (threshold: %.0f%%)", absPct, vcfg.P95RegWarn))
			}
		case "error_rate":
			if absPct >= vcfg.ErrorRateRegWarn {
				if level < verdict.Warn {
					level = verdict.Warn
				}
				reasons = append(reasons, fmt.Sprintf("error_rate regressed %.1f%% (threshold: %.0f%%)", absPct, vcfg.ErrorRateRegWarn))
			}
		}
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "No significant regressions detected")
	}

	output := map[string]any{
		"verdict":    level.String(),
		"reasons":    reasons,
		"comparison": comp,
	}
	if err := ciOutput(output); err != nil {
		return err
	}

	r := verdict.Result{Level: level, Reasons: reasons}
	if code := r.ExitCode(); code != 0 {
		return &ExitError{Code: code}
	}
	return nil
}

func parseDeltaPct(delta string) float64 {
	s := strings.TrimSuffix(delta, "%")
	s = strings.TrimPrefix(s, "+")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
