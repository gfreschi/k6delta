package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/gfreschi/k6delta/internal/tui/dashboard"
)

// NewDashboardCmd creates the "dashboard" subcommand.
func NewDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Open the interactive dashboard",
		Long:  "Opens the k6delta dashboard for browsing apps, running tests, and reviewing results.",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("config")
			return runDashboard(configFile)
		},
	}
}

// RunDashboard opens the interactive dashboard. Exported for use by root command.
func RunDashboard(configFile string) error {
	return runDashboard(configFile)
}

func runDashboard(configFile string) error {
	cfg, err := loadConfig(configFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	env := cfg.Defaults.Env
	if env == "" {
		env = "staging"
	}

	m := dashboard.NewModel(cfg, env, nil)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("dashboard: %w", err)
	}
	return nil
}
