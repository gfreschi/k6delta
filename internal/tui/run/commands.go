package runtui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gfreschi/k6delta/internal/k6"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/report"
)

func (m Model) checkAuth() tea.Cmd {
	return func() tea.Msg {
		if err := m.provider.CheckCredentials(context.Background()); err != nil {
			return errMsg{err: err}
		}
		return authOKMsg{}
	}
}

func (m Model) fetchSnapshot(label string) tea.Cmd {
	return func() tea.Msg {
		snap, err := m.provider.TakeSnapshot(context.Background())
		if err != nil {
			return errMsg{err: fmt.Errorf("snapshot (%s): %w", label, err)}
		}
		return snapshotMsg{snapshot: snap, label: label}
	}
}

func (m Model) fetchAnalysis() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		metrics, err := m.provider.FetchMetrics(ctx, m.startTime, m.endTime, 60)
		if err != nil {
			return errMsg{err: fmt.Errorf("fetch metrics: %w", err)}
		}

		activities, err := m.provider.FetchActivities(ctx, m.startTime, m.endTime)
		if err != nil {
			return errMsg{err: fmt.Errorf("fetch activities: %w", err)}
		}

		return analysisMsg{metrics: metrics, activities: activities}
	}
}

func buildInfraMetrics(metrics []provider.MetricResult) *report.InfraMetrics {
	infra := &report.InfraMetrics{}
	for _, m := range metrics {
		pa := &report.PeakAvg{Peak: m.Peak, Avg: m.Avg}
		switch m.ID {
		case "service_cpu":
			infra.ServiceCPU = pa
		case "service_memory":
			infra.ServiceMemory = pa
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

func (m Model) buildReport() tea.Cmd {
	return func() tea.Msg {
		info := report.RunInfo{
			App:             m.app.Name,
			Env:             m.app.Env,
			Phase:           m.app.Phase,
			Start:           m.startTime.Format(time.RFC3339),
			End:             m.endTime.Format(time.RFC3339),
			DurationSeconds: int(m.endTime.Sub(m.startTime).Seconds()),
		}
		if m.k6Result != nil {
			info.K6Exit = m.k6Result.ExitCode
		}

		k6SummaryPath := filepath.Join(m.app.ResultsDir, m.resultsPrefix+"-summary.json")
		if _, err := os.Stat(k6SummaryPath); err != nil {
			k6SummaryPath = ""
		}

		infra := buildInfraMetrics(m.metrics)

		r, err := report.BuildUnifiedReportFromInfra(
			info, k6SummaryPath, infra,
			m.preSnapshot.TaskRunning, m.postSnapshot.TaskRunning,
			m.preSnapshot.ASGInstances, m.postSnapshot.ASGInstances,
		)
		if err != nil {
			return errMsg{err: fmt.Errorf("build report: %w", err)}
		}

		r.ScalingActivities = m.activities
		r.AlarmHistory = m.activities.Alarms

		var reportPath string
		if m.app.ResultsDir != "" {
			reportPath = filepath.Join(m.app.ResultsDir, m.resultsPrefix+"-report.json")
			if err := report.WriteReport(r, reportPath); err != nil {
				return errMsg{err: fmt.Errorf("write report: %w", err)}
			}
		}

		return reportMsg{report: r, path: reportPath}
	}
}

func (m Model) exportReport() tea.Cmd {
	return func() tea.Msg {
		if m.report == nil {
			return errMsg{err: fmt.Errorf("no report to export")}
		}
		path := m.reportPath
		if err := report.WriteReport(m.report, path); err != nil {
			return errMsg{err: err}
		}
		return exportDoneMsg{path: path}
	}
}

func (m Model) openHTML() tea.Cmd {
	return func() tea.Msg {
		htmlPath := filepath.Join(m.app.ResultsDir, m.resultsPrefix+".html")
		if _, err := os.Stat(htmlPath); err != nil {
			return errMsg{err: fmt.Errorf("HTML report not found: %s", htmlPath)}
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", htmlPath)
		default:
			cmd = exec.Command("xdg-open", htmlPath)
		}
		_ = cmd.Start() // fire and forget
		return openDoneMsg{path: htmlPath}
	}
}

func (m Model) runK6Streaming(k6Ctx context.Context) tea.Cmd {
	cfg := k6.RunConfig{
		TestFile:      m.app.TestFile,
		Env:           m.app.Env,
		ResultsPrefix: m.resultsPrefix,
		ResultsDir:    m.app.ResultsDir,
	}
	if m.baseURL != "" {
		cfg.BaseURL = m.baseURL
	}
	ch := m.k6PointChan
	startTime := m.startTime
	return func() tea.Msg {
		result, err := k6.RunStreaming(k6Ctx, cfg, ch)
		if err != nil {
			return errMsg{err: fmt.Errorf("k6 streaming: %w", err)}
		}
		result.StartTime = startTime
		return k6DoneMsg{result: result}
	}
}

func (m Model) runK6Demo() tea.Cmd {
	ch := m.k6PointChan
	speed := m.demoSpeed
	scenario := m.demoScenario
	startTime := m.startTime
	duration := 30 * time.Second

	return func() tea.Msg {
		result := k6.FakeStream(duration, speed, scenario, ch)
		result.StartTime = startTime
		return k6DoneMsg{result: result}
	}
}

func (m Model) waitForK6Point() tea.Cmd {
	ch := m.k6PointChan
	return func() tea.Msg {
		point, ok := <-ch
		if !ok {
			return nil
		}
		return k6PointMsg{point: point}
	}
}

func (m Model) runK6Fallback() tea.Cmd {
	cfg := k6.RunConfig{
		TestFile:      m.app.TestFile,
		Env:           m.app.Env,
		ResultsPrefix: m.resultsPrefix,
		ResultsDir:    m.app.ResultsDir,
	}
	if m.baseURL != "" {
		cfg.BaseURL = m.baseURL
	}

	args := k6.BuildArgs(cfg)
	env := k6.BuildEnv(cfg)

	c := exec.Command("k6", args...)
	c.Env = env

	startTime := m.startTime

	return tea.ExecProcess(c, func(err error) tea.Msg {
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return errMsg{err: fmt.Errorf("k6 exec: %w", err)}
			}
		}
		return k6DoneMsg{result: k6.RunResult{
			ExitCode:  exitCode,
			StartTime: startTime,
			EndTime:   time.Now().UTC(),
		}}
	})
}
