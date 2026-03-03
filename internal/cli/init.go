package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NewInitCmd creates the "init" subcommand.
func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Generate a starter k6delta.yaml config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			const filename = "k6delta.yaml"

			if _, err := os.Stat(filename); err == nil {
				return fmt.Errorf("%s already exists; remove it first to re-initialize", filename)
			}

			reader := bufio.NewReader(os.Stdin)

			appName := prompt(reader, "App name", "web")
			clusterPattern := prompt(reader, "Cluster naming pattern", "${app}-${env}")
			servicePattern := prompt(reader, "Service naming pattern", "${app}-${env}")
			region := prompt(reader, "AWS region", "us-east-1")
			testFile := prompt(reader, "Test file path", "tests/${app}/${phase}.js")

			content := fmt.Sprintf(`# k6delta.yaml -- k6delta configuration
# Variable interpolation: ${env} and ${app} are replaced at runtime.

provider: ecs
region: %s

defaults:
  env: staging
  phase: smoke
  results_dir: results

apps:
  %s:
    cluster: "%s"
    service: "%s"
    test_file: "%s"
    # Uncomment and customize optional fields:
    # asg_prefix: "%s-${env}-ecs-"
    # capacity_provider: "%s-${env}-ec2"
    # alarm_prefix: "%s-${env}"
`, region, appName, clusterPattern, servicePattern, testFile, appName, appName, appName)

			if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", filename, err)
			}

			fmt.Printf("Created %s\n", filename)
			return nil
		},
	}
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	fmt.Printf("%s [%s]: ", label, defaultVal)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}
