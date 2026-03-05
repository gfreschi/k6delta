package k6

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// RunStreaming executes k6 with JSON output piped to stdout and sends
// parsed data points to the provided channel. The caller is responsible
// for closing the channel after RunStreaming returns.
func RunStreaming(ctx context.Context, cfg RunConfig, points chan<- K6Point) (RunResult, error) {
	args := BuildArgs(cfg)
	// Insert --out json=- before the test file (last element) for live streaming
	testFile := args[len(args)-1]
	args = args[:len(args)-1]
	args = append(args, "--out", "json=-", testFile)

	cmd := exec.CommandContext(ctx, "k6", args...)
	cmd.Env = BuildEnv(cfg)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return RunResult{}, fmt.Errorf("create stdout pipe: %w", err)
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	result := RunResult{StartTime: time.Now()}

	if err := cmd.Start(); err != nil {
		return RunResult{}, fmt.Errorf("start k6: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		point, parseErr := ParseJSONLine(scanner.Text())
		if parseErr == nil && point != nil {
			select {
			case points <- *point:
			default:
				// Drop if channel full
			}
		}
	}

	err = cmd.Wait()
	result.EndTime = time.Now()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, fmt.Errorf("k6 process: %w", err)
	}

	return result, nil
}
