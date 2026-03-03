package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gfreschi/k6delta/internal/config"
)

// loadConfig loads config from the given path, or falls back to k6delta.yaml / defaults.
func loadConfig(path string) (*config.Config, error) {
	if path != "" {
		return config.Load(path)
	}
	return config.LoadOrDefault()
}

// resolveApp looks up the app in config and produces a ResolvedApp with CLI overrides applied.
func resolveApp(cfg *config.Config, appName, env, phase, region string) (config.ResolvedApp, error) {
	if appName == "" {
		return config.ResolvedApp{}, fmt.Errorf("--app is required")
	}

	appCfg, ok := cfg.Apps[appName]
	if !ok {
		return config.ResolvedApp{}, fmt.Errorf("app %q not found in config (available: %s)", appName, availableApps(cfg))
	}

	if env == "" {
		env = cfg.Defaults.Env
	}
	if env == "" {
		env = "staging"
	}

	if phase == "" {
		phase = cfg.Defaults.Phase
	}
	if phase == "" {
		phase = "smoke"
	}

	if region == "" {
		region = cfg.Region
	}
	if region == "" {
		region = "us-east-1"
	}

	resultsDir := cfg.Defaults.ResultsDir
	if resultsDir == "" {
		resultsDir = "results"
	}

	return config.Interpolate(appCfg, appName, env, phase, region, resultsDir), nil
}

func availableApps(cfg *config.Config) string {
	if len(cfg.Apps) == 0 {
		return "none"
	}
	names := make([]string, 0, len(cfg.Apps))
	for k := range cfg.Apps {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
