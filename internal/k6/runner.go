// Package k6 builds command-line arguments and environment variables
// for executing k6 load tests via os/exec.
package k6

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RunConfig holds the configuration for a k6 test run.
type RunConfig struct {
	TestFile      string
	Env           string
	ResultsPrefix string
	ResultsDir    string
	BaseURL       string // optional override
}

// RunResult holds the outcome of a k6 test run.
type RunResult struct {
	ExitCode  int
	StartTime time.Time
	EndTime   time.Time
}

// BuildArgs constructs k6 CLI arguments from the given RunConfig.
func BuildArgs(cfg RunConfig) []string {
	args := []string{
		"run",
		"--env", fmt.Sprintf("ENV=%s", cfg.Env),
		"--env", fmt.Sprintf("RESULTS_PREFIX=%s", cfg.ResultsPrefix),
	}

	if cfg.BaseURL != "" {
		args = append(args, "--env", fmt.Sprintf("BASE_URL=%s", cfg.BaseURL))
	}

	timeseriesPath := filepath.Join(cfg.ResultsDir, cfg.ResultsPrefix+"-timeseries.json.gz")
	args = append(args, "--out", fmt.Sprintf("json=%s", timeseriesPath))

	summaryPath := filepath.Join(cfg.ResultsDir, cfg.ResultsPrefix+"-summary.json")
	args = append(args, "--summary-export", summaryPath)

	args = append(args, cfg.TestFile)

	return args
}

// BuildEnv constructs environment variables for the k6 process.
func BuildEnv(cfg RunConfig) []string {
	env := os.Environ()

	env = append(env, "K6_WEB_DASHBOARD=true")
	htmlPath := filepath.Join(cfg.ResultsDir, cfg.ResultsPrefix+".html")
	env = append(env, fmt.Sprintf("K6_WEB_DASHBOARD_EXPORT=%s", htmlPath))

	return env
}

// GenerateResultsPrefix generates a timestamp-based prefix for result files.
func GenerateResultsPrefix(appName, phase, env string) string {
	ts := time.Now().UTC().Format("20060102T150405Z")
	return fmt.Sprintf("%s-%s-%s-%s", appName, phase, env, ts)
}

// Run spawns k6 via exec.CommandContext, pipes stdout/stderr to the provided
// writers, and captures the exit code.
func Run(ctx context.Context, cfg RunConfig, stdout, stderr io.Writer) (RunResult, error) {
	args := BuildArgs(cfg)
	cmd := exec.CommandContext(ctx, "k6", args...)
	cmd.Env = BuildEnv(cfg)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	result := RunResult{
		StartTime: time.Now(),
	}

	err := cmd.Run()
	result.EndTime = time.Now()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, fmt.Errorf("run k6: %w", err)
	}

	result.ExitCode = 0
	return result, nil
}
