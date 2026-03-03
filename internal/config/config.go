// Package config handles YAML configuration loading, CLI flag merging,
// and variable interpolation for k6delta.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level k6delta configuration.
type Config struct {
	Provider string              `yaml:"provider"`
	Region   string              `yaml:"region"`
	Defaults Defaults            `yaml:"defaults"`
	Apps     map[string]AppConfig `yaml:"apps"`
}

// Defaults holds default values for CLI flags.
type Defaults struct {
	Env        string `yaml:"env"`
	Phase      string `yaml:"phase"`
	ResultsDir string `yaml:"results_dir"`
}

// AppConfig holds the infrastructure configuration for one application.
// All string fields support ${env} and ${app} interpolation.
type AppConfig struct {
	Cluster          string `yaml:"cluster"`
	Service          string `yaml:"service"`
	ASGPrefix        string `yaml:"asg_prefix"`
	CapacityProvider string `yaml:"capacity_provider"`
	TestFile         string `yaml:"test_file"`
	AlarmPrefix      string `yaml:"alarm_prefix"`
}

// ResolvedApp is an AppConfig with all ${var} interpolations applied.
type ResolvedApp struct {
	Name             string
	Cluster          string
	Service          string
	ASGPrefix        string
	CapacityProvider string
	TestFile         string
	AlarmPrefix      string
	Env              string
	Phase            string
	Region           string
	ResultsDir       string
}

// Load reads and parses a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// LoadOrDefault tries to load k6delta.yaml from the current directory.
// Returns an empty Config (with defaults) if the file does not exist.
func LoadOrDefault() (*Config, error) {
	cfg, err := Load("k6delta.yaml")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			d := DefaultConfig()
			return &d, nil
		}
		return nil, err
	}
	return cfg, nil
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		Provider: "ecs",
		Region:   "us-east-1",
		Defaults: Defaults{
			Env:        "staging",
			Phase:      "smoke",
			ResultsDir: "results",
		},
		Apps: make(map[string]AppConfig),
	}
}

// Interpolate replaces ${env}, ${app}, and ${phase} in all string fields
// of an AppConfig and returns a fully resolved ResolvedApp.
func Interpolate(app AppConfig, name, env, phase, region, resultsDir string) ResolvedApp {
	r := strings.NewReplacer(
		"${env}", env,
		"${app}", name,
		"${phase}", phase,
	)
	return ResolvedApp{
		Name:             name,
		Cluster:          r.Replace(app.Cluster),
		Service:          r.Replace(app.Service),
		ASGPrefix:        r.Replace(app.ASGPrefix),
		CapacityProvider: r.Replace(app.CapacityProvider),
		TestFile:         r.Replace(app.TestFile),
		AlarmPrefix:      r.Replace(app.AlarmPrefix),
		Env:              env,
		Phase:            phase,
		Region:           region,
		ResultsDir:       resultsDir,
	}
}

var validPhases = map[string]bool{
	"smoke":  true,
	"load":   true,
	"stress": true,
	"soak":   true,
}

// ValidatePhase checks that s is one of smoke, load, stress, soak.
func ValidatePhase(s string) error {
	if !validPhases[s] {
		return fmt.Errorf("invalid phase %q: must be one of smoke, load, stress, soak", s)
	}
	return nil
}
