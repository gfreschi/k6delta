// Command k6delta wraps Grafana k6 with AWS infrastructure monitoring
// and before/after comparison.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/gfreschi/k6delta/internal/cli"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:           "k6delta",
		Short:         "Run k6 load tests. See what your infrastructure did.",
		Long:          "k6delta wraps k6 execution with infrastructure monitoring and before/after comparison.\nIt correlates k6 test results with ECS, ASG, ALB, and CloudWatch metrics in a single command.",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !term.IsTerminal(int(os.Stdout.Fd())) {
				return cmd.Help()
			}
			configFile, _ := cmd.Flags().GetString("config")
			return cli.RunDashboard(configFile)
		},
	}

	rootCmd.PersistentFlags().String("config", "", "config file path (default: k6delta.yaml)")

	rootCmd.AddCommand(cli.NewRunCmd())
	rootCmd.AddCommand(cli.NewAnalyzeCmd())
	rootCmd.AddCommand(cli.NewCompareCmd())
	rootCmd.AddCommand(cli.NewInitCmd())
	rootCmd.AddCommand(cli.NewDemoCmd())
	rootCmd.AddCommand(cli.NewDashboardCmd())

	if err := rootCmd.Execute(); err != nil {
		var exitErr *cli.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
