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
with ECS infrastructure snapshots, CloudWatch metrics collection, and
a unified report - collapsing five manual steps into one command.

## Features

- **One-command load tests:** run k6 with infrastructure monitoring in a single invocation
- **Pre/post snapshots:** captures ECS task counts and ASG instance counts before and after your test
- **CloudWatch metrics:** CPU, memory, reservation, ALB request rates, response times, 5xx errors, healthy hosts
- **Scaling activity tracking:** ASG scaling events and CloudWatch alarm history during the test window
- **Unified JSON reports:** k6 results + infrastructure metrics in one portable file
- **Report comparison:** diff two runs with percentage deltas and direction indicators (better/worse)
- **Standalone analysis:** query CloudWatch metrics for any time window, no k6 required
- **Interactive TUI:** terminal UI powered by Bubble Tea for results display
- **Graceful degradation:** optional config fields are silently skipped, not errored

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

Executes a k6 test while capturing infrastructure state before and after. Collects CloudWatch metrics and produces a unified JSON report.

```bash
k6delta run --app <name> --phase <smoke|load|stress|soak> [flags]
```

Output files:

- `{prefix}.json` - k6 summary
- `{prefix}.html` - k6 web dashboard
- `{prefix}-timeseries.json.gz` - k6 time-series data
- `{prefix}-report.json` - unified report (k6 + infra + scaling)

### `k6delta analyze`

Queries CloudWatch metrics for a time window without running k6.

```bash
k6delta analyze --app <name> --env <env> [flags]
```

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

Create a `k6delta.yaml` - see [k6delta.example.yaml](k6delta.example.yaml) for a full reference.

```yaml
provider: ecs
region: eu-west-1

apps:
  web:
    cluster: "myapp-${env}"
    service: "myapp-web-${env}"
    test_file: "tests/performance/web/${phase}.js"
```

`${env}`, `${app}`, `${phase}` are interpolated at runtime from CLI flags or defaults.
Optional fields (`asg_prefix`, `capacity_provider`, `alarm_prefix`) are silently skipped when missing.

## Prerequisites

- **Go 1.25+:** for building from source
- **[k6](https://k6.io/docs/get-started/installation/):** must be available in your `PATH`
- **AWS credentials:** configured via any method supported by the [AWS SDK default credential chain](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configuring-sdk.html) (environment variables, `~/.aws/credentials`, IAM roles, etc.)

## Development

```bash
make build        # build binary with version injection
make test         # go test ./... -v
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
