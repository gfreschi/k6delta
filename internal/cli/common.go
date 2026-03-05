package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/provider/compose"
	"github.com/gfreschi/k6delta/internal/provider/ecs"
	"github.com/gfreschi/k6delta/internal/provider/mock"
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

// createProvider selects the infrastructure provider based on config.
func createProvider(ctx context.Context, cfg *config.Config, resolved config.ResolvedApp) (provider.InfraProvider, error) {
	prov := cfg.Provider
	if prov == "" {
		prov = "ecs"
	}
	switch prov {
	case "ecs":
		return ecs.New(ctx, resolved)
	case "docker-compose":
		return compose.New(resolved, resolved.ComposeProject)
	case "mock":
		return mock.New(resolved.MockScenario)
	default:
		return nil, fmt.Errorf("unknown provider: %q (supported: ecs, docker-compose, mock)", prov)
	}
}

// progressSetter is optionally implemented by providers that support progress callbacks.
type progressSetter interface {
	SetOnProgress(fn func(id string, current, total int))
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
