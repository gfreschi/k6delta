// Package cli defines the Cobra subcommands for the k6delta CLI.
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/k6"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/provider/ecs"
	"github.com/gfreschi/k6delta/internal/report"
	runtui "github.com/gfreschi/k6delta/internal/tui/run"
	"github.com/gfreschi/k6delta/internal/verdict"
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
		ciMode      bool
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

			if ciMode {
				return runCI(ctx, resolved, prov, baseURL, skipAnalyze, cfg.Verdicts.WithDefaults())
			}

			m := runtui.NewModel(resolved, prov, baseURL, skipAnalyze, cfg.Verdicts.WithDefaults())
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
	cmd.Flags().BoolVar(&ciMode, "ci", false, "CI mode: no TUI, JSON to stdout, exit code = verdict")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("phase")

	return cmd
}

func runCI(ctx context.Context, app config.ResolvedApp, prov provider.InfraProvider, baseURL string, skipAnalyze bool, vcfg config.VerdictConfig) error {
	fmt.Fprintf(os.Stderr, "k6delta: checking credentials...\n")
	if err := prov.CheckCredentials(ctx); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "k6delta: pre-snapshot...\n")
	presnap, err := prov.TakeSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("pre-snapshot: %w", err)
	}

	fmt.Fprintf(os.Stderr, "k6delta: running k6...\n")
	prefix := k6.GenerateResultsPrefix(app.Name, app.Phase, app.Env)
	k6Cfg := k6.RunConfig{
		TestFile:      app.TestFile,
		ResultsPrefix: prefix,
		ResultsDir:    app.ResultsDir,
		BaseURL:       baseURL,
	}
	k6Result, err := k6.Run(ctx, k6Cfg, os.Stderr, os.Stderr)
	if err != nil {
		return fmt.Errorf("k6 run: %w", err)
	}

	fmt.Fprintf(os.Stderr, "k6delta: post-snapshot...\n")
	postsnap, err := prov.TakeSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("post-snapshot: %w", err)
	}

	var metrics []provider.MetricResult
	var activities provider.Activities
	if !skipAnalyze {
		fmt.Fprintf(os.Stderr, "k6delta: fetching metrics...\n")
		metrics, err = prov.FetchMetrics(ctx, k6Result.StartTime, k6Result.EndTime, 60)
		if err != nil {
			return fmt.Errorf("fetch metrics: %w", err)
		}

		fmt.Fprintf(os.Stderr, "k6delta: fetching activities...\n")
		activities, err = prov.FetchActivities(ctx, k6Result.StartTime, k6Result.EndTime)
		if err != nil {
			return fmt.Errorf("fetch activities: %w", err)
		}
	}

	infra := buildInfraMetricsCI(metrics)

	info := report.RunInfo{
		App:             app.Name,
		Env:             app.Env,
		Phase:           app.Phase,
		Start:           k6Result.StartTime.Format(time.RFC3339),
		End:             k6Result.EndTime.Format(time.RFC3339),
		K6Exit:          k6Result.ExitCode,
		DurationSeconds: int(k6Result.EndTime.Sub(k6Result.StartTime).Seconds()),
	}

	k6SummaryPath := filepath.Join(app.ResultsDir, prefix+"-summary.json")
	if _, statErr := os.Stat(k6SummaryPath); statErr != nil {
		k6SummaryPath = ""
	}

	unifiedReport, err := report.BuildUnifiedReportFromInfra(
		info, k6SummaryPath, infra,
		presnap.TaskRunning, postsnap.TaskRunning,
		presnap.ASGInstances, postsnap.ASGInstances,
	)
	if err != nil {
		return fmt.Errorf("build report: %w", err)
	}

	unifiedReport.ScalingActivities = activities
	unifiedReport.AlarmHistory = activities.Alarms

	reportPath := filepath.Join(app.ResultsDir, prefix+"-report.json")
	if writeErr := report.WriteReport(unifiedReport, reportPath); writeErr != nil {
		fmt.Fprintf(os.Stderr, "k6delta: warning: could not write report: %v\n", writeErr)
	} else {
		fmt.Fprintf(os.Stderr, "k6delta: report written to %s\n", reportPath)
	}

	// Compute verdict
	var cpuPeak *float64
	if infra != nil && infra.ECSCPU != nil {
		cpuPeak = infra.ECSCPU.Peak
	}
	v := verdict.Compute(verdict.Input{
		K6Exit:      k6Result.ExitCode,
		ALB5xx:      infra.ALB5xx,
		ECScPUPeak:  cpuPeak,
		TasksBefore: presnap.TaskRunning,
		TasksAfter:  postsnap.TaskRunning,
		Activities:  activities,
	}, vcfg)

	output := map[string]any{
		"verdict":     v.Level.String(),
		"reasons":     v.Reasons,
		"report":      unifiedReport,
		"report_path": reportPath,
	}
	if err := ciOutput(output); err != nil {
		return err
	}

	os.Exit(v.ExitCode())
	return nil
}

func buildInfraMetricsCI(metrics []provider.MetricResult) *report.InfraMetrics {
	infra := &report.InfraMetrics{}
	for _, m := range metrics {
		pa := &report.PeakAvg{Peak: m.Peak, Avg: m.Avg}
		switch m.ID {
		case "ecs_cpu":
			infra.ECSCPU = pa
		case "ecs_memory":
			infra.ECSMemory = pa
		case "cluster_cpu_reservation":
			infra.ClusterCPUReservation = pa
		case "cluster_memory_reservation":
			infra.ClusterMemoryReservation = pa
		case "capacity_provider_reservation":
			infra.CapacityProviderReservation = pa
		case "alb_response_time":
			infra.ALBResponseTimeP95 = m.Peak
		case "alb_5xx":
			if m.Peak != nil {
				infra.ALB5xx = int(*m.Peak)
			}
		}
	}
	return infra
}

func printDryRun(app config.ResolvedApp, baseURL string) error {
	prefix := k6.GenerateResultsPrefix(app.Name, app.Phase, app.Env)
	cfg := k6.RunConfig{
		TestFile:      app.TestFile,
		Env:           app.Env,
		ResultsPrefix: prefix,
		ResultsDir:    app.ResultsDir,
		BaseURL:       baseURL,
	}

	args := k6.BuildArgs(cfg)

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
