# k6delta

[![CI](https://github.com/gfreschi/k6delta/actions/workflows/ci.yaml/badge.svg)](https://github.com/gfreschi/k6delta/actions/workflows/ci.yaml)
[![Go 1.25+](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Run k6 load tests. See what your infrastructure did.**

k6delta wraps [Grafana k6](https://k6.io/) load test execution with AWS infrastructure monitoring and before/after comparison. It answers the question DevOps engineers ask after every load test: *"My test ran -- did autoscaling fire? How high did CPU go? Any 5xx during scale-out?"*

It collapses five manual steps (record infra state, run k6, check CloudWatch, record post-state, compare) into one command.

| Tool | Load Generation | Infra Monitoring | Before/After Delta | Single Command |
|------|:-:|:-:|:-:|:-:|
| k6 native | Yes | No | No | Partial |
| k6 + Grafana stack | Yes | Yes (with setup) | No | No |
| **k6delta** | **Yes (via k6)** | **Yes** | **Yes** | **Yes** |

## Features

- **One-command load tests** -- run k6 with infrastructure monitoring in a single invocation
- **Pre/post snapshots** -- captures ECS task counts and ASG instance counts before and after your test
- **CloudWatch metrics** -- CPU, memory, reservation, ALB request rates, response times, 5xx errors, healthy hosts
- **Scaling activity tracking** -- ASG scaling events and CloudWatch alarm history during the test window
- **Unified JSON reports** -- k6 results + infrastructure metrics in one portable file
- **Report comparison** -- diff two runs with percentage deltas and direction indicators (better/worse)
- **Standalone analysis** -- query CloudWatch metrics for any time window, no k6 required
- **Interactive TUI** -- terminal UI powered by Bubble Tea for results display
- **Graceful degradation** -- optional config fields are silently skipped, not errored

## Prerequisites

- **Go 1.25+** -- for building from source
- **[k6](https://k6.io/docs/get-started/installation/)** -- must be available in your `PATH`
- **AWS credentials** -- configured via any method supported by the [AWS SDK default credential chain](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configuring-sdk.html) (environment variables, `~/.aws/credentials`, IAM roles, etc.)

## Installation

### From source

```bash
go install github.com/gfreschi/k6delta/cmd/k6delta@latest
```

### Build locally

```bash
git clone https://github.com/gfreschi/k6delta.git
cd k6delta
make build    # produces ./k6delta
```

## Quick Start

```bash
# 1. Generate a starter config
k6delta init

# 2. Edit k6delta.yaml with your cluster/service names

# 3. Run a load test with infrastructure monitoring
k6delta run --app web --phase smoke

# 4. Compare two runs
k6delta compare results/report-baseline.json results/report-latest.json
```

## Commands

### `k6delta run`

Executes a k6 test while capturing infrastructure state before and after. Collects CloudWatch metrics during the test window. Produces a unified JSON report.

```bash
k6delta run --app <name> --phase <smoke|load|stress|soak> [flags]
```

| Flag | Description |
|------|-------------|
| `--app` | Application name as defined in `k6delta.yaml` (required) |
| `--phase` | Test phase: `smoke`, `load`, `stress`, `soak` (required) |
| `--env` | Environment override (default from config) |
| `--region` | AWS region override (default from config) |
| `--config` | Config file path (default: `k6delta.yaml`) |
| `--base-url` | Override base URL for k6 test |
| `--skip-analyze` | Skip CloudWatch analysis after k6 run |
| `--dry-run` | Print k6 command without executing |

**Workflow:** Check AWS credentials -> pre-snapshot (ECS tasks, ASG instances) -> hand terminal to k6 -> post-snapshot -> fetch CloudWatch metrics + scaling activities -> build unified report -> render TUI summary.

**Output files:**

- `{prefix}.json` -- k6 summary
- `{prefix}.html` -- k6 web dashboard
- `{prefix}-timeseries.json.gz` -- k6 time-series data
- `{prefix}-report.json` -- unified report (k6 + infra + scaling)

### `k6delta analyze`

Queries CloudWatch metrics for a time window without running k6.

```bash
k6delta analyze --app <name> --env <env> [flags]
```

| Flag | Description |
|------|-------------|
| `--app` | Application name (required) |
| `--env` | Environment (required) |
| `--duration` | Duration in minutes (alternative to `--start`/`--end`) |
| `--start` | Start time in RFC3339 format |
| `--end` | End time in RFC3339 format |
| `--period` | CloudWatch metric period in seconds (default: `60`) |
| `--json` | Output JSON instead of TUI |
| `--output` | Write JSON output to file |
| `--region` | AWS region override |
| `--config` | Config file path |

### `k6delta compare`

Compares two unified report JSON files side-by-side with percentage deltas and direction indicators.

```bash
k6delta compare <report-a.json> <report-b.json> [--json]
```

### `k6delta init`

Generates a starter `k6delta.yaml` config file.

```bash
k6delta init
```

## Configuration

Create a `k6delta.yaml` in your project root. CLI flags override any config value.

```yaml
provider: ecs          # v1 only supports "ecs"
region: eu-west-1      # AWS region (overridable with --region)

defaults:
  env: staging         # default --env value
  phase: smoke         # default --phase value
  results_dir: tests/performance/results

apps:
  web:
    cluster: "myapp-${env}"
    service: "myapp-web-${env}"
    asg_prefix: "myapp-${env}-ecs-"              # optional
    capacity_provider: "myapp-${env}-ec2"         # optional
    test_file: "tests/performance/web/${phase}.js"
    alarm_prefix: "myapp-${env}"                  # optional
```

**Variable interpolation:** `${env}`, `${app}`, and `${phase}` are replaced at runtime from CLI flags or defaults.

**Optional fields:** Missing `asg_prefix`, `capacity_provider`, or `alarm_prefix` causes those metrics to be skipped -- no errors, fewer data points.

**Config resolution order:** Built-in defaults -> `k6delta.yaml` -> CLI flags (highest priority).

## How It Works

```
Config load + CLI flags
       |
       v
  Interpolate ${env}, ${app}, ${phase}
       |
       v
  AWS credential check (STS)
       |
       v
  Pre-test snapshot (ECS task counts, ASG instance counts)
       |
       v
  k6 run (hands terminal to k6)
       |
       v
  Post-test snapshot
       |
       v
  CloudWatch metrics:
    - ECS: CPUUtilization, MemoryUtilization
    - Cluster: CPUReservation, MemoryReservation
    - Capacity Provider: CapacityProviderReservation
    - ASG: GroupDesiredCapacity, GroupInServiceInstances
    - ALB: RequestCountPerTarget, TargetResponseTime(p95), 5XX, HealthyHosts
       |
       v
  Scaling activities + alarm history
       |
       v
  Unified JSON report + TUI summary
```

## Architecture

```
cmd/k6delta/main.go           Entry point (Cobra root command)
internal/
  cli/                         Cobra subcommand definitions (run, analyze, compare, init)
  config/                      YAML config loader + CLI flag merge + ${var} interpolation
  provider/
    provider.go                InfraProvider interface + shared types
    ecs/                       AWS ECS implementation of InfraProvider
  k6/                          k6 process execution (generic, no AWS coupling)
  report/                      Report schema + comparison logic (generic, no AWS coupling)
  tui/                         Bubble Tea TUI models + Lip Gloss styles
```

**Key design decisions:**

- **InfraProvider interface** -- all infrastructure interaction goes through `provider.InfraProvider`. v1 only has ECS; the interface enables EKS/Prometheus providers without changing CLI or TUI code.
- **Config resolution** -- `k6delta.yaml` -> CLI flag overrides -> `config.Interpolate()` produces a `ResolvedApp`. All downstream code uses `ResolvedApp`, never raw config.
- **TUI accepts interfaces** -- TUI models take `config.ResolvedApp` + `provider.InfraProvider`, not concrete AWS types.
- **k6 as subprocess** -- k6 is invoked via `os/exec`, not imported as a library. The terminal is handed directly to k6 during execution.

## Development

```bash
make build        # go build -o k6delta ./cmd/k6delta
make test         # go test ./... -v
make lint         # go vet ./...
make clean        # rm -f k6delta
```

Run a single package's tests:

```bash
go test ./internal/config/ -v
```

### Tech Stack

- **Go 1.25+** with modules
- **[Cobra](https://github.com/spf13/cobra)** -- CLI framework
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** + [Bubbles](https://github.com/charmbracelet/bubbles) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) -- TUI
- **[AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)** -- ECS, CloudWatch, Auto Scaling, ELBv2, STS
- **[gopkg.in/yaml.v3](https://github.com/go-yaml/yaml)** -- YAML config parsing

## Contributing

Contributions are welcome. Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Write tests for new functionality
4. Ensure all tests pass (`make test`)
5. Ensure code passes linting (`make lint`)
6. Open a pull request against `main`

## License

[MIT](LICENSE)
