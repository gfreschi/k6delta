package cli

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	analyzetui "github.com/gfreschi/k6delta/internal/tui/analyze"
)

// NewAnalyzeCmd creates the "analyze" subcommand.
func NewAnalyzeCmd() *cobra.Command {
	var (
		appName    string
		env        string
		region     string
		configFile string
		startFlag  string
		endFlag    string
		duration   int
		period     int32
		jsonOutput bool
		outputFile string
		ciMode     bool
		refreshSec int
	)

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Query infrastructure metrics for a time window (no k6 execution)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(configFile)
			if err != nil {
				return err
			}

			resolved, err := resolveApp(cfg, appName, env, "", region)
			if err != nil {
				return err
			}

			startTime, endTime, err := resolveTimeWindow(startFlag, endFlag, duration)
			if err != nil {
				return err
			}

			ctx := context.Background()
			prov, err := createProvider(ctx, cfg, resolved)
			if err != nil {
				return err
			}

			if ciMode || jsonOutput {
				return analyzetui.RunJSON(resolved, prov, startTime, endTime, period, outputFile)
			}

			m := analyzetui.NewModel(resolved, prov, startTime, endTime, period, jsonOutput, outputFile, refreshSec)
			p := tea.NewProgram(m)
			if ps, ok := prov.(progressSetter); ok {
				ps.SetOnProgress(func(id string, current, total int) {
					p.Send(analyzetui.ProgressMsg{ID: id, Current: current, Total: total})
				})
			}
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("TUI error: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&appName, "app", "", "application name (required)")
	cmd.Flags().StringVar(&env, "env", "", "environment (default from config)")
	cmd.Flags().StringVar(&region, "region", "", "AWS region (default from config)")
	cmd.Flags().StringVar(&configFile, "config", "", "config file path (default: k6delta.yaml)")
	cmd.Flags().StringVar(&startFlag, "start", "", "start time (RFC3339)")
	cmd.Flags().StringVar(&endFlag, "end", "", "end time (RFC3339)")
	cmd.Flags().IntVar(&duration, "duration", 0, "duration in minutes (alternative to --start/--end)")
	cmd.Flags().Int32Var(&period, "period", 60, "CloudWatch metric period in seconds")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output JSON instead of TUI")
	cmd.Flags().StringVar(&outputFile, "output", "", "write JSON output to file")
	cmd.Flags().BoolVar(&ciMode, "ci", false, "CI mode: JSON to stdout, no TUI")
	cmd.Flags().IntVar(&refreshSec, "refresh", 0, "auto-refresh interval in seconds (0=disabled)")

	_ = cmd.MarkFlagRequired("app")

	return cmd
}

func resolveTimeWindow(startFlag, endFlag string, duration int) (string, string, error) {
	if startFlag != "" && endFlag != "" {
		if _, err := time.Parse(time.RFC3339, startFlag); err != nil {
			return "", "", fmt.Errorf("invalid --start: %w", err)
		}
		if _, err := time.Parse(time.RFC3339, endFlag); err != nil {
			return "", "", fmt.Errorf("invalid --end: %w", err)
		}
		return startFlag, endFlag, nil
	}

	if duration > 0 {
		end := time.Now().UTC()
		start := end.Add(-time.Duration(duration) * time.Minute)
		return start.Format(time.RFC3339), end.Format(time.RFC3339), nil
	}

	return "", "", fmt.Errorf("must specify --start/--end or --duration")
}
