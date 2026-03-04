// k6delta wraps Grafana k6 load test execution with AWS infrastructure
// monitoring and before/after comparison.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

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
	}

	rootCmd.AddCommand(cli.NewRunCmd())
	rootCmd.AddCommand(cli.NewAnalyzeCmd())
	rootCmd.AddCommand(cli.NewCompareCmd())
	rootCmd.AddCommand(cli.NewInitCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
