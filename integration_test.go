//go:build integration

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	bin := filepath.Join(tmpDir, "k6delta")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/k6delta")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func TestDryRun(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "run", "--app", "web", "--phase", "smoke", "--dry-run",
		"--config", "k6delta.example.yaml")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dry-run failed: %v\n%s", err, out)
	}
	output := string(out)

	if !strings.Contains(output, "k6delta dry-run") {
		t.Errorf("output missing dry-run header:\n%s", output)
	}
	if !strings.Contains(output, "k6 run") {
		t.Errorf("output missing k6 command:\n%s", output)
	}
	if !strings.Contains(output, "No test executed (--dry-run)") {
		t.Errorf("output missing dry-run footer:\n%s", output)
	}
	if !strings.Contains(output, "tests/performance/web/smoke.js") {
		t.Errorf("output missing test file path:\n%s", output)
	}
}

func TestCompareJSON(t *testing.T) {
	bin := buildBinary(t)

	pathA := filepath.Join("internal", "report", "testdata", "report-a.json")
	pathB := filepath.Join("internal", "report", "testdata", "report-b.json")

	cmd := exec.Command(bin, "compare", "--json", pathA, pathB)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compare --json failed: %v\n%s", err, out)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}
	for _, key := range []string{"run_a", "run_b", "comparison"} {
		if _, ok := result[key]; !ok {
			t.Errorf("JSON output missing key %q", key)
		}
	}
}

func TestRunMissingApp(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "run", "--phase", "smoke")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for missing --app, got nil")
	}
	if !strings.Contains(string(out), "required") {
		t.Errorf("expected 'required' in error output:\n%s", out)
	}
}

func TestRunMissingPhase(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "run", "--app", "web")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for missing --phase, got nil")
	}
	if !strings.Contains(string(out), "required") {
		t.Errorf("expected 'required' in error output:\n%s", out)
	}
}

func TestAnalyzeMissingFlags(t *testing.T) {
	bin := buildBinary(t)

	// Missing --app
	cmd := exec.Command(bin, "analyze", "--duration", "10")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for missing --app, got nil")
	}
	if !strings.Contains(string(out), "required") {
		t.Errorf("expected 'required' in error output:\n%s", out)
	}
}

func TestCompareWrongArgCount(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "compare", "only-one.json")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for wrong arg count, got nil")
	}
	if !strings.Contains(string(out), "accepts 2 arg(s)") {
		t.Errorf("expected arg count error:\n%s", out)
	}
}

func TestInitCreatesFile(t *testing.T) {
	bin := buildBinary(t)

	tmpDir := t.TempDir()
	cmd := exec.Command(bin, "init")
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader("\n\n\n\n\n") // accept all defaults

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	configPath := filepath.Join(tmpDir, "k6delta.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("k6delta.yaml not created: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "provider: ecs") {
		t.Errorf("config missing provider:\n%s", content)
	}
	if !strings.Contains(content, "web:") {
		t.Errorf("config missing default app name:\n%s", content)
	}
}

func TestRunCIModeHelp(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "run", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "--ci") {
		t.Error("expected --ci flag in run help output")
	}
}

func TestCompareCIModeHelp(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "compare", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "--ci") {
		t.Error("expected --ci flag in compare help output")
	}
}

func TestAnalyzeCIModeHelp(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "analyze", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "--ci") {
		t.Error("expected --ci flag in analyze help output")
	}
}

func TestInitRefusesOverwrite(t *testing.T) {
	bin := buildBinary(t)

	tmpDir := t.TempDir()

	// Create first
	cmd := exec.Command(bin, "init")
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader("\n\n\n\n\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("first init failed: %v\n%s", err, out)
	}

	// Try again -- should fail
	cmd = exec.Command(bin, "init")
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader("\n\n\n\n\n")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error on second init, got nil")
	}
	if !strings.Contains(string(out), "already exists") {
		t.Errorf("expected 'already exists' error:\n%s", out)
	}
}
