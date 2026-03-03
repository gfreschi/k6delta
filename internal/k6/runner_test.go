package k6runner

import (
	"regexp"
	"strings"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	cfg := RunConfig{
		TestFile:      "tests/performance/web/smoke.js",
		Env:           "staging",
		ResultsPrefix: "web-smoke-staging-20260303T120000Z",
		ResultsDir:    "tests/performance/results",
	}

	args := BuildArgs(cfg)

	if args[0] != "run" {
		t.Errorf("args[0] = %q, want %q", args[0], "run")
	}

	envFound := false
	for i, a := range args {
		if a == "--env" && i+1 < len(args) && args[i+1] == "ENV=staging" {
			envFound = true
			break
		}
	}
	if !envFound {
		t.Errorf("ENV=staging not found in args: %v", args)
	}

	prefixFound := false
	for i, a := range args {
		if a == "--env" && i+1 < len(args) && strings.HasPrefix(args[i+1], "RESULTS_PREFIX=") {
			prefixFound = true
			break
		}
	}
	if !prefixFound {
		t.Errorf("RESULTS_PREFIX not found in args: %v", args)
	}

	last := args[len(args)-1]
	if last != cfg.TestFile {
		t.Errorf("last arg = %q, want %q", last, cfg.TestFile)
	}

	outFound := false
	for i, a := range args {
		if a == "--out" && i+1 < len(args) {
			val := args[i+1]
			if strings.HasPrefix(val, "json=") && strings.HasSuffix(val, "-timeseries.json.gz") {
				outFound = true
			}
			break
		}
	}
	if !outFound {
		t.Errorf("--out json=...timeseries.json.gz not found in args: %v", args)
	}
}

func TestBuildArgsWithBaseURL(t *testing.T) {
	cfg := RunConfig{
		TestFile:      "tests/performance/web/smoke.js",
		Env:           "staging",
		ResultsPrefix: "web-smoke-staging-20260303T120000Z",
		ResultsDir:    "tests/performance/results",
		BaseURL:       "https://custom.example.com",
	}

	args := BuildArgs(cfg)

	baseURLFound := false
	for i, a := range args {
		if a == "--env" && i+1 < len(args) && args[i+1] == "BASE_URL=https://custom.example.com" {
			baseURLFound = true
			break
		}
	}
	if !baseURLFound {
		t.Errorf("BASE_URL not found in args when BaseURL is set: %v", args)
	}
}

func TestBuildArgsWithoutBaseURL(t *testing.T) {
	cfg := RunConfig{
		TestFile:      "tests/performance/web/smoke.js",
		Env:           "staging",
		ResultsPrefix: "web-smoke-staging-20260303T120000Z",
		ResultsDir:    "tests/performance/results",
		BaseURL:       "",
	}

	args := BuildArgs(cfg)

	for i, a := range args {
		if a == "--env" && i+1 < len(args) && strings.HasPrefix(args[i+1], "BASE_URL=") {
			t.Errorf("BASE_URL should NOT be present when BaseURL is empty, but found in args: %v", args)
			break
		}
	}
}

func TestBuildEnv(t *testing.T) {
	cfg := RunConfig{
		ResultsPrefix: "web-smoke-staging-20260303T120000Z",
		ResultsDir:    "tests/performance/results",
	}

	env := BuildEnv(cfg)

	dashboardFound := false
	exportFound := false

	for _, e := range env {
		if e == "K6_WEB_DASHBOARD=true" {
			dashboardFound = true
		}
		if strings.HasPrefix(e, "K6_WEB_DASHBOARD_EXPORT=") {
			exportFound = true
			val := strings.TrimPrefix(e, "K6_WEB_DASHBOARD_EXPORT=")
			if !strings.HasSuffix(val, ".html") {
				t.Errorf("K6_WEB_DASHBOARD_EXPORT does not end with .html: %q", val)
			}
			if !strings.Contains(val, cfg.ResultsPrefix) {
				t.Errorf("K6_WEB_DASHBOARD_EXPORT does not contain results prefix: %q", val)
			}
		}
	}

	if !dashboardFound {
		t.Error("K6_WEB_DASHBOARD=true not found in env")
	}
	if !exportFound {
		t.Error("K6_WEB_DASHBOARD_EXPORT not found in env")
	}
}

func TestGenerateResultsPrefix(t *testing.T) {
	prefix := GenerateResultsPrefix("web", "smoke", "staging")

	if !strings.HasPrefix(prefix, "web-smoke-staging-") {
		t.Errorf("prefix %q does not start with expected pattern", prefix)
	}

	parts := strings.SplitN(prefix, "-", 4)
	if len(parts) < 4 {
		t.Fatalf("prefix %q has fewer than 4 dash-separated parts", prefix)
	}
	ts := parts[3]

	matched, err := regexp.MatchString(`^\d{8}T\d{6}Z$`, ts)
	if err != nil {
		t.Fatalf("regexp error: %v", err)
	}
	if !matched {
		t.Errorf("timestamp %q does not match YYYYMMDDTHHMMSSZ format", ts)
	}
}
