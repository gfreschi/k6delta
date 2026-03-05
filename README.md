<div align="center">
  <h1>k6delta</h1>
  <p><strong>Run k6 load tests. See what your infrastructure did.</strong></p>
  <p>
    <a href="https://github.com/gfreschi/k6delta/actions/workflows/ci.yaml"><img src="https://github.com/gfreschi/k6delta/actions/workflows/ci.yaml/badge.svg" alt="CI"></a>
    <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go" alt="Go 1.25+"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
  </p>
  <p>
    <a href="#installation">Installation</a> ·
    <a href="#quick-start">Quick Start</a> ·
    <a href="#commands">Commands</a> ·
    <a href="#configuration">Configuration</a>
  </p>
</div>

<!-- TODO: Add demo GIF recorded with VHS/asciinema -->

## Why k6delta?

*"My load test finished - did autoscaling fire? How high did CPU go?
Any 5xx during scale-out?"*

k6delta answers these questions automatically. It wraps [k6](https://k6.io/) execution
with infrastructure monitoring - capturing snapshots, metrics, and scaling events - then
produces a unified report. Works with AWS ECS, Docker Compose, and a built-in mock provider
for demos and testing.

## Features

- **One-command load tests:** run k6 with infrastructure monitoring in a single invocation
- **Multiple providers:** AWS ECS (CloudWatch, ASG, ALB), Docker Compose (container stats, events), and Mock (synthetic data for demos)
- **Pre/post snapshots:** captures task/container counts before and after your test
- **Metrics collection:** CPU, memory, scaling events, and provider-specific metrics
- **Unified JSON reports:** k6 results + infrastructure metrics in one portable file
- **Report comparison:** diff two runs with percentage deltas and direction indicators (better/worse)
- **Verdict system:** configurable PASS/WARN/FAIL thresholds for CPU, 5xx errors, and regressions
- **CI mode:** `--ci` flag on all commands - JSON to stdout, exit code 0 (pass/warn) or 1 (fail)
- **Interactive TUI:** live dashboard with braille charts, panel navigation, and responsive layout
- **Standalone analysis:** query infrastructure metrics for any time window, no k6 required
- **Demo mode:** try the full TUI experience with `k6delta demo` — no infrastructure or k6 binary needed
- **Graceful degradation:** optional config fields are silently skipped, not errored

## Quick Start

```bash
# Try it instantly — no config, no infrastructure, no k6 needed
k6delta demo

# Or with a specific scenario
k6delta demo --scenario cascade-failure --speed 2
```

```bash
# For real infrastructure monitoring:
# 1. Generate a starter config
k6delta init

# 2. Edit k6delta.yaml with your provider and app settings

# 3. Run a load test with infrastructure monitoring
k6delta run --app web --phase smoke

# 4. Compare two runs
k6delta compare results/report-baseline.json results/report-latest.json

# 5. Use in CI (JSON to stdout, exit code = verdict)
k6delta run --app web --phase load --ci
```

## Installation

**From GitHub Releases** - download a prebuilt binary from the [Releases page](https://github.com/gfreschi/k6delta/releases):

```bash
# Example for Linux amd64
curl -Lo k6delta.tar.gz https://github.com/gfreschi/k6delta/releases/latest/download/k6delta_linux_amd64.tar.gz
tar xzf k6delta.tar.gz
sudo mv k6delta /usr/local/bin/
```

**From source**

```bash
go install github.com/gfreschi/k6delta/cmd/k6delta@latest
```

**Build locally**

```bash
git clone https://github.com/gfreschi/k6delta.git
cd k6delta
make build    # produces ./k6delta
```

## Commands

> Run any command with `--help` for full flag reference.

### `k6delta run`

Executes a k6 test while capturing infrastructure state before and after. Collects metrics from your provider and produces a unified JSON report.

```bash
k6delta run --app <name> --phase <smoke|load|stress|soak> [flags]
```

Output files:

- `{prefix}.json` - k6 summary
- `{prefix}.html` - k6 web dashboard
- `{prefix}-timeseries.json.gz` - k6 time-series data
- `{prefix}-report.json` - unified report (k6 + infra + scaling)

### `k6delta analyze`

Queries infrastructure metrics for a time window without running k6.

```bash
k6delta analyze --app <name> --env <env> [flags]
```

### `k6delta compare`

Compares two unified report JSON files side-by-side with percentage deltas and direction indicators. With `--ci`, checks regression thresholds and sets exit code.

```bash
k6delta compare <report-a.json> <report-b.json> [--json | --ci]
```

### `k6delta demo`

Runs a simulated load test with synthetic data. No AWS, Docker, or k6 binary needed.

```bash
k6delta demo [--scenario <name>] [--speed <multiplier>] [--list]
```

Available scenarios: `happy-path`, `cpu-spike`, `scale-out`, `cascade-failure`.

### `k6delta init`

Generates a starter `k6delta.yaml` config file.

```bash
k6delta init
```

## Configuration

Create a `k6delta.yaml` - see [k6delta.example.yaml](k6delta.example.yaml) for a full reference.

```yaml
provider: docker-compose   # or: ecs, mock
region: eu-west-1           # AWS region (ECS only)

verdicts:                   # optional verdict thresholds
  cpu_peak_warn: 90
  p95_regression_warn: 10

apps:
  web:
    compose_project: "myapp"                          # Docker Compose
    # cluster: "myapp-${env}"                         # ECS
    # service: "myapp-web-${env}"                     # ECS
    test_file: "tests/performance/web/${phase}.js"
```

`${env}`, `${app}`, `${phase}` are interpolated at runtime from CLI flags or defaults.
Optional fields (`asg_prefix`, `capacity_provider`, `alarm_prefix`) are silently skipped when missing.

## Prerequisites

- **Go 1.25+:** for building from source
- **[k6](https://k6.io/docs/get-started/installation/):** must be available in your `PATH` (not needed for `k6delta demo`)
- **For Docker Compose provider:** Docker Engine running with your compose project up
- **For ECS provider:** AWS credentials configured via the [AWS SDK default credential chain](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configuring-sdk.html)
- **For Mock provider / demo:** no prerequisites — works out of the box

## Development

```bash
make build        # build binary with version injection
make test         # go test ./... -v
make test-tui     # TUI package tests only
make test-update  # regenerate golden files (UPDATE_GOLDEN=1)
make lint         # go vet ./...
make clean        # rm -f k6delta
```

Run a single package's tests:

```bash
go test ./internal/config/ -v
```

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on commit conventions, PR process, and code standards.

## License

[MIT](LICENSE)
