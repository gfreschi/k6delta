// Package cli defines the Cobra subcommands for the k6delta CLI.
package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/gfreschi/k6delta/internal/config"
	k6runner "github.com/gfreschi/k6delta/internal/k6"
	"github.com/gfreschi/k6delta/internal/provider/ecs"
	runtui "github.com/gfreschi/k6delta/internal/tui/run"
)

// NewRunCmd creates the "run" subcommand.
func NewRunCmd() *cobra.Command {
	var (
		appName     string
		phase       string
		env         string
		region      string
		configFile  string
		skipAnalyze bool
		dryRun      bool
		baseURL     string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run k6 + monitor infrastructure + generate unified report",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(configFile)
			if err != nil {
				return err
			}

			resolved, err := resolveApp(cfg, appName, env, phase, region)
			if err != nil {
				return err
			}

			if err := config.ValidatePhase(resolved.Phase); err != nil {
				return err
			}

			if dryRun {
				return printDryRun(resolved, baseURL)
			}

			ctx := context.Background()
			prov, err := ecs.New(ctx, resolved)
			if err != nil {
				return err
			}

			m := runtui.NewModel(resolved, prov, baseURL, skipAnalyze)
			p := tea.NewProgram(m)
			prov.SetOnProgress(func(id string, current, total int) {
				p.Send(runtui.ProgressMsg{ID: id, Current: current, Total: total})
			})
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("TUI error: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&appName, "app", "", "application name (required)")
	cmd.Flags().StringVar(&phase, "phase", "", "test phase: smoke, load, stress, soak (required)")
	cmd.Flags().StringVar(&env, "env", "", "environment (default from config)")
	cmd.Flags().StringVar(&region, "region", "", "AWS region (default from config)")
	cmd.Flags().StringVar(&configFile, "config", "", "config file path (default: k6delta.yaml)")
	cmd.Flags().BoolVar(&skipAnalyze, "skip-analyze", false, "skip CloudWatch analysis after k6 run")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print k6 command without executing")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "override base URL for k6 test")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("phase")

	return cmd
}

func printDryRun(app config.ResolvedApp, baseURL string) error {
	prefix := k6runner.GenerateResultsPrefix(app.Name, app.Phase, app.Env)
	cfg := k6runner.RunConfig{
		TestFile:      app.TestFile,
		Env:           app.Env,
		ResultsPrefix: prefix,
		ResultsDir:    app.ResultsDir,
		BaseURL:       baseURL,
	}

	args := k6runner.BuildArgs(cfg)

	var envVars []string
	envVars = append(envVars, "K6_WEB_DASHBOARD=true")
	envVars = append(envVars, fmt.Sprintf("K6_WEB_DASHBOARD_EXPORT=%s",
		filepath.Join(app.ResultsDir, prefix+".html")))

	fmt.Println("k6delta dry-run -- command that would be executed:")
	fmt.Println()
	fmt.Printf("  %s \\\n", strings.Join(envVars, " \\\n    "))
	fmt.Printf("    k6 %s\n", strings.Join(args, " "))
	fmt.Println()
	fmt.Println("Output files (would be created):")
	fmt.Printf("  k6 summary:      %s\n", filepath.Join(app.ResultsDir, prefix+".json"))
	fmt.Printf("  HTML report:      %s\n", filepath.Join(app.ResultsDir, prefix+".html"))
	fmt.Printf("  Time-series:      %s\n", filepath.Join(app.ResultsDir, prefix+"-timeseries.json.gz"))
	fmt.Printf("  Unified report:   %s\n", filepath.Join(app.ResultsDir, prefix+"-report.json"))
	fmt.Println()
	fmt.Println("No test executed (--dry-run).")

	return nil
}
