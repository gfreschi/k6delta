package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/provider/mock"
	runtui "github.com/gfreschi/k6delta/internal/tui/run"
)

// NewDemoCmd creates the "demo" subcommand.
func NewDemoCmd() *cobra.Command {
	var (
		scenario string
		speed    float64
		list     bool
	)

	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Run a simulated load test with synthetic data (no infrastructure required)",
		Long:  "Demonstrates k6delta's full TUI experience using mock infrastructure data.\nNo AWS, Docker, or k6 binary needed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if list {
				return printScenarios(cmd)
			}

			prov, err := mock.New(scenario)
			if err != nil {
				return err
			}

			app := demoResolvedApp(scenario)
			vcfg := config.DefaultVerdictConfig()

			m := runtui.NewDemoModel(app, prov, speed, scenario, vcfg)
			p := tea.NewProgram(m)
			prov.SetOnProgress(func(id string, current, total int) {
				p.Send(runtui.ProgressMsg{ID: id, Current: current, Total: total})
			})

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("TUI error: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&scenario, "scenario", "happy-path", "scenario to simulate")
	cmd.Flags().Float64Var(&speed, "speed", 1.0, "time multiplier (2.0 = 2x faster)")
	cmd.Flags().BoolVar(&list, "list", false, "list available scenarios")

	return cmd
}

func printScenarios(cmd *cobra.Command) error {
	w := cmd.OutOrStdout()
	scenarios := mock.ListScenarios()
	_, _ = fmt.Fprintln(w, "Available scenarios:")
	_, _ = fmt.Fprintln(w)
	for _, s := range scenarios {
		_, _ = fmt.Fprintf(w, "  %-20s %s\n", s.Name, s.Description)
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage: k6delta demo --scenario <name> [--speed <multiplier>]")
	return nil
}

func demoResolvedApp(scenario string) config.ResolvedApp {
	return config.ResolvedApp{
		Name:         "demo-app",
		Env:          "demo",
		Phase:        "smoke",
		Region:       "local",
		ResultsDir:   "",
		TestFile:     "demo.js",
		MockScenario: scenario,
	}
}
