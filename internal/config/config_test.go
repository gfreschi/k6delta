package config

import (
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "full.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Provider != "ecs" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "ecs")
	}
	if cfg.Region != "eu-west-1" {
		t.Errorf("Region = %q, want %q", cfg.Region, "eu-west-1")
	}
	if cfg.Defaults.Env != "staging" {
		t.Errorf("Defaults.Env = %q, want %q", cfg.Defaults.Env, "staging")
	}
	if cfg.Defaults.Phase != "smoke" {
		t.Errorf("Defaults.Phase = %q, want %q", cfg.Defaults.Phase, "smoke")
	}
	if cfg.Defaults.ResultsDir != "results" {
		t.Errorf("Defaults.ResultsDir = %q, want %q", cfg.Defaults.ResultsDir, "results")
	}
	if len(cfg.Apps) != 2 {
		t.Fatalf("len(Apps) = %d, want 2", len(cfg.Apps))
	}
	web, ok := cfg.Apps["web"]
	if !ok {
		t.Fatal("Apps[\"web\"] not found")
	}
	if web.Cluster != "myapp-${env}" {
		t.Errorf("web.Cluster = %q, want %q", web.Cluster, "myapp-${env}")
	}
	if web.ASGPrefix != "myapp-${env}-ecs-" {
		t.Errorf("web.ASGPrefix = %q, want %q", web.ASGPrefix, "myapp-${env}-ecs-")
	}
}

func TestLoadMinimal(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "minimal.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", cfg.Region, "us-east-1")
	}
	if len(cfg.Apps) != 1 {
		t.Fatalf("len(Apps) = %d, want 1", len(cfg.Apps))
	}
	api := cfg.Apps["api"]
	if api.ASGPrefix != "" {
		t.Errorf("api.ASGPrefix = %q, want empty", api.ASGPrefix)
	}
	if api.CapacityProvider != "" {
		t.Errorf("api.CapacityProvider = %q, want empty", api.CapacityProvider)
	}
	// Defaults struct fields should be zero values when not specified
	if cfg.Defaults.Env != "" {
		t.Errorf("Defaults.Env = %q, want empty", cfg.Defaults.Env)
	}
}

func TestLoadOrDefault(t *testing.T) {
	// When no k6delta.yaml exists in cwd, returns default config without error.
	// This test runs from the testdata-less config package dir where no k6delta.yaml exists.
	cfg, err := LoadOrDefault()
	if err != nil {
		t.Fatalf("LoadOrDefault: %v", err)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", cfg.Region, "us-east-1")
	}
	if cfg.Defaults.Env != "staging" {
		t.Errorf("Defaults.Env = %q, want %q", cfg.Defaults.Env, "staging")
	}
}

func TestInterpolate(t *testing.T) {
	app := AppConfig{
		Cluster:          "myapp-${env}",
		Service:          "myapp-web-${env}",
		ASGPrefix:        "myapp-${env}-ecs-",
		CapacityProvider: "myapp-${env}-ec2",
		TestFile:         "tests/${app}/${phase}.js",
		AlarmPrefix:      "myapp-${env}",
	}
	resolved := Interpolate(app, "web", "staging", "load", "eu-west-1", "results")

	if resolved.Name != "web" {
		t.Errorf("Name = %q, want %q", resolved.Name, "web")
	}
	if resolved.Cluster != "myapp-staging" {
		t.Errorf("Cluster = %q, want %q", resolved.Cluster, "myapp-staging")
	}
	if resolved.Service != "myapp-web-staging" {
		t.Errorf("Service = %q, want %q", resolved.Service, "myapp-web-staging")
	}
	if resolved.ASGPrefix != "myapp-staging-ecs-" {
		t.Errorf("ASGPrefix = %q, want %q", resolved.ASGPrefix, "myapp-staging-ecs-")
	}
	if resolved.CapacityProvider != "myapp-staging-ec2" {
		t.Errorf("CapacityProvider = %q, want %q", resolved.CapacityProvider, "myapp-staging-ec2")
	}
	if resolved.TestFile != "tests/web/load.js" {
		t.Errorf("TestFile = %q, want %q", resolved.TestFile, "tests/web/load.js")
	}
	if resolved.AlarmPrefix != "myapp-staging" {
		t.Errorf("AlarmPrefix = %q, want %q", resolved.AlarmPrefix, "myapp-staging")
	}
	if resolved.Env != "staging" {
		t.Errorf("Env = %q, want %q", resolved.Env, "staging")
	}
	if resolved.Phase != "load" {
		t.Errorf("Phase = %q, want %q", resolved.Phase, "load")
	}
	if resolved.Region != "eu-west-1" {
		t.Errorf("Region = %q, want %q", resolved.Region, "eu-west-1")
	}
	if resolved.ResultsDir != "results" {
		t.Errorf("ResultsDir = %q, want %q", resolved.ResultsDir, "results")
	}
}

func TestInterpolateOptionalEmpty(t *testing.T) {
	app := AppConfig{
		Cluster:  "myapp-${env}",
		Service:  "myapp-worker-${env}",
		TestFile: "tests/${app}/${phase}.js",
		// ASGPrefix, CapacityProvider, AlarmPrefix intentionally empty
	}
	resolved := Interpolate(app, "worker", "prod", "smoke", "us-east-1", "results")

	if resolved.ASGPrefix != "" {
		t.Errorf("ASGPrefix = %q, want empty", resolved.ASGPrefix)
	}
	if resolved.CapacityProvider != "" {
		t.Errorf("CapacityProvider = %q, want empty", resolved.CapacityProvider)
	}
	if resolved.AlarmPrefix != "" {
		t.Errorf("AlarmPrefix = %q, want empty", resolved.AlarmPrefix)
	}
	if resolved.Cluster != "myapp-prod" {
		t.Errorf("Cluster = %q, want %q", resolved.Cluster, "myapp-prod")
	}
}

func TestValidatePhase(t *testing.T) {
	valid := []string{"smoke", "load", "stress", "soak"}
	for _, p := range valid {
		if err := ValidatePhase(p); err != nil {
			t.Errorf("ValidatePhase(%q) = %v, want nil", p, err)
		}
	}
	if err := ValidatePhase("benchmark"); err == nil {
		t.Error("ValidatePhase(\"benchmark\") = nil, want error")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Provider != "ecs" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "ecs")
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", cfg.Region, "us-east-1")
	}
	if cfg.Defaults.Env != "staging" {
		t.Errorf("Defaults.Env = %q, want %q", cfg.Defaults.Env, "staging")
	}
	if cfg.Defaults.Phase != "smoke" {
		t.Errorf("Defaults.Phase = %q, want %q", cfg.Defaults.Phase, "smoke")
	}
	if cfg.Apps == nil {
		t.Error("Apps is nil, want initialized map")
	}
}
