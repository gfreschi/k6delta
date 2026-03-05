package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	comparetui "github.com/gfreschi/k6delta/internal/tui/compare"
)

// NewCompareCmd creates the "compare" subcommand.
func NewCompareCmd() *cobra.Command {
	var (
		jsonOutput bool
		ciMode     bool
	)

	cmd := &cobra.Command{
		Use:   "compare <report-a.json> <report-b.json>",
		Short: "Compare two unified reports side-by-side with percentage deltas",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathA := args[0]
			pathB := args[1]

			if ciMode || jsonOutput {
				return comparetui.RunJSON(pathA, pathB)
			}

			m := comparetui.NewModel(pathA, pathB)
			p := tea.NewProgram(m)
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("TUI error: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output JSON instead of TUI")
	cmd.Flags().BoolVar(&ciMode, "ci", false, "CI mode: JSON to stdout, exit code = regression verdict")

	return cmd
}
