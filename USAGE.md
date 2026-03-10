# Usage

Complete CLI reference for k6delta. Run any command with `--help` for inline help.

---

## Table of Contents

- [k6delta dashboard](#k6delta-dashboard) - interactive workspace
- [k6delta run](#k6delta-run) - load test + infra monitoring
- [k6delta analyze](#k6delta-analyze) - query infra metrics
- [k6delta compare](#k6delta-compare) - diff two runs
- [k6delta demo](#k6delta-demo) - try without infrastructure
- [k6delta init](#k6delta-init) - generate config
- [CI Mode](#ci-mode) - JSON output + exit codes
- [Configuration](#configuration) - k6delta.yaml reference
- [Workflows](#workflows) - common usage patterns

---

## k6delta dashboard

Interactive workspace for browsing configured apps, selecting test phases, and launching commands. This is the default entrypoint -- running `k6delta` with no arguments opens the dashboard.

```bash
k6delta                    # opens dashboard (default)
k6delta dashboard          # explicit form
```

### Flags

| Flag       | Type   | Default      | Description                  |
|------------|--------|--------------|------------------------------|
| `--config` | string | k6delta.yaml | Config file path             |

### Keybindings

| Key              | Action                     |
|------------------|----------------------------|
| `j/k` or `up/dn` | Navigate app list         |
| `h/l` or `left/right` | Previous/next test phase |
| `enter`     | Run selected app + phase   |
| `a`         | Analyze selected app       |
| `c`         | Compare reports            |
| `r`         | View reports               |
| `d`         | Launch demo                |
| `?`         | Help overlay               |
| `q`         | Quit                       |

### Behavior

- **TTY detected:** opens the full-screen dashboard TUI
- **Non-TTY (piped/redirected):** prints the help text instead
- The dashboard reads `k6delta.yaml` and lists all configured apps with their provider type
- Phase picker cycles through smoke, load, stress, soak for the selected app

---

## k6delta run

Execute a k6 test while capturing infrastructure state before and after. Collects metrics from your provider and produces a unified JSON report.

```bash
k6delta run --app <name> --phase <smoke|load|stress|soak> [flags]
```

### Flags

| Flag             | Type   | Default      | Description                                  |
|------------------|--------|--------------|----------------------------------------------|
| `--app`          | string | required     | Application name (must exist in config)      |
| `--phase`        | string | required     | Test phase: smoke, load, stress, soak        |
| `--env`          | string | from config  | Environment name                             |
| `--region`       | string | from config  | AWS region                                   |
| `--config`       | string | k6delta.yaml | Config file path                             |
| `--base-url`     | string | -            | Override base URL for k6 test                |
| `--skip-analyze` | bool   | false        | Skip CloudWatch analysis after k6            |
| `--dry-run`      | bool   | false        | Print k6 command without executing           |
| `--ci`           | bool   | false        | JSON to stdout, exit code 0/1 = verdict      |

### Examples

```bash
# Smoke test with TUI dashboard
k6delta run --app web --phase smoke

# Load test against production
k6delta run --app web --phase load --env production

# Stress test with custom base URL
k6delta run --app api --phase stress --base-url https://staging.api.example.com

# Preview the k6 command without running
k6delta run --app web --phase load --dry-run

# Skip infra analysis (k6 only)
k6delta run --app web --phase smoke --skip-analyze

# CI mode (JSON output, exit code)
k6delta run --app web --phase smoke --ci

# Custom config file
k6delta run --app web --phase smoke --config ./configs/staging.yaml
```

### Output Files

After a successful run, these files are created in the results directory:

```
results/
  {app}-{env}-{phase}.json               # k6 summary
  {app}-{env}-{phase}.html               # k6 web dashboard
  {app}-{env}-{phase}-timeseries.json.gz  # k6 time-series data
  {app}-{env}-{phase}-report.json         # unified report (k6 + infra + scaling)
```

### TUI Keybindings

| Key       | Action                            |
|-----------|-----------------------------------|
| `tab`     | Cycle panel focus                 |
| `1`-`4`   | Jump to panel                     |
| `+`       | Expand/collapse focused panel     |
| `up/down` | Scroll within panel               |
| `e`       | Export report                     |
| `o`       | Open report file                  |
| `r`       | Toggle raw JSON view              |
| `?`       | Help overlay                      |
| `q`       | Quit                              |

---

## k6delta analyze

Query infrastructure metrics for a time window without running k6. Useful for investigating incidents or checking infra state independently.

```bash
k6delta analyze --app <name> [flags]
```

### Flags

| Flag         | Type   | Default      | Description                                    |
|--------------|--------|--------------|------------------------------------------------|
| `--app`      | string | required     | Application name (must exist in config)        |
| `--env`      | string | from config  | Environment name                               |
| `--region`   | string | from config  | AWS region                                     |
| `--config`   | string | k6delta.yaml | Config file path                               |
| `--start`    | string | -            | Start time (RFC3339)                           |
| `--end`      | string | -            | End time (RFC3339)                             |
| `--duration` | int    | -            | Duration in minutes (alternative to start/end) |
| `--period`   | int32  | 60           | CloudWatch metric period in seconds            |
| `--json`     | bool   | false        | Output JSON instead of TUI                     |
| `--output`   | string | -            | Write JSON output to file                      |
| `--ci`       | bool   | false        | JSON to stdout, no TUI                         |

Either `--start` + `--end` (both RFC3339) or `--duration` (minutes) is required. Cannot mix both.

### Examples

```bash
# Analyze last 30 minutes with TUI dashboard
k6delta analyze --app web --duration 30

# Analyze specific time window
k6delta analyze --app web \
  --start 2026-03-09T10:00:00Z \
  --end 2026-03-09T11:00:00Z

# With 5-minute metric resolution
k6delta analyze --app web --duration 60 --period 300

# JSON output to file
k6delta analyze --app web --duration 30 --json --output metrics.json

# Production environment
k6delta analyze --app web --env production --duration 15

# CI mode
k6delta analyze --app web \
  --start 2026-03-09T10:00:00Z \
  --end 2026-03-09T11:00:00Z \
  --ci
```

### TUI Keybindings

| Key       | Action                            |
|-----------|-----------------------------------|
| `tab`     | Cycle panel focus                 |
| `1`-`3`   | Jump to panel                     |
| `+`       | Expand/collapse focused panel     |
| `up/down` | Scroll within panel               |
| `e`       | Export report                     |
| `?`       | Help overlay                      |
| `q`       | Quit                              |

---

## k6delta compare

Compare two unified report JSON files side-by-side with percentage deltas, direction indicators, and regression verdict.

```bash
k6delta compare <report-a.json> <report-b.json> [flags]
```

### Flags

| Flag       | Type   | Default      | Description                                  |
|------------|--------|--------------|----------------------------------------------|
| `--json`   | bool   | false        | Output JSON instead of TUI                   |
| `--ci`     | bool   | false        | JSON + exit code 0/1 = regression verdict    |
| `--config` | string | k6delta.yaml | Config file (for verdict thresholds in CI)   |

### Examples

```bash
# Compare two runs with TUI
k6delta compare results/baseline-report.json results/latest-report.json

# JSON output
k6delta compare results/run-a.json results/run-b.json --json

# CI mode (exit code 1 if regression detected)
k6delta compare results/run-a.json results/run-b.json --ci

# CI with custom thresholds from config
k6delta compare results/a.json results/b.json --ci --config staging.yaml
```

### TUI Keybindings

| Key       | Action                                   |
|-----------|------------------------------------------|
| `tab`     | Cycle panel focus                        |
| `1`-`2`   | Jump to panel                            |
| `+`       | Expand/collapse focused panel            |
| `enter`   | Drill down into selected metric          |
| `s`       | Toggle sort order                        |
| `d`       | Side-by-side diff mode (>= 140 width)   |
| `up/down` | Scroll within panel                      |
| `e`       | Export report                            |
| `?`       | Help overlay                             |
| `q`       | Quit                                     |

---

## k6delta demo

Run a simulated load test with synthetic data. No AWS, Docker, or k6 binary needed.

```bash
k6delta demo [flags]
```

### Flags

| Flag         | Type    | Default      | Description                          |
|--------------|---------|--------------|--------------------------------------|
| `--scenario` | string  | happy-path   | Scenario to simulate                 |
| `--speed`    | float64 | 1.0          | Time multiplier (2.0 = 2x faster)   |
| `--list`     | bool    | false        | List available scenarios and exit    |

### Examples

```bash
# Default demo (happy-path scenario, 1x speed)
k6delta demo

# Fast cascade failure demo
k6delta demo --scenario cascade-failure --speed 2

# CPU spike scenario at normal speed
k6delta demo --scenario cpu-spike

# Scale-out scenario (shows ASG autoscaling)
k6delta demo --scenario scale-out --speed 3

# List all available scenarios
k6delta demo --list
```

### Available Scenarios

| Scenario          | Duration | Verdict | Description                              |
|-------------------|----------|---------|------------------------------------------|
| `happy-path`      | 60s      | PASS    | Steady state, all metrics healthy        |
| `cpu-spike`       | 60s      | WARN    | CPU sine wave peaks at 92%, no scaling   |
| `scale-out`       | 90s      | PASS    | ASG fires 2->5 tasks, CPU recovers      |
| `cascade-failure` | 60s      | FAIL    | 3/4 tasks OOMKilled, 5xx spike          |

---

## k6delta init

Generate a starter `k6delta.yaml` config file interactively.

```bash
k6delta init
```

Prompts for app name, cluster/service patterns, AWS region, and test file path. Creates `k6delta.yaml` in the current directory. Fails if the file already exists.

---

## CI Mode

All main commands support `--ci` for pipeline integration. CI mode disables the TUI, writes JSON to stdout, and sets the exit code based on the verdict.

### Exit Codes

| Command   | Exit 0           | Exit 1        |
|-----------|------------------|---------------|
| `run`     | PASS or WARN     | FAIL          |
| `analyze` | Always           | Error only    |
| `compare` | No regression    | Regression    |

### Pipeline Examples

```bash
# Run load test, fail pipeline on FAIL verdict
k6delta run --app web --phase load --ci || exit 1

# Compare against baseline, fail on regression
k6delta compare baseline.json latest.json --ci || exit 1

# Analyze and save metrics to file
k6delta analyze --app web --duration 30 --ci > metrics.json

# Full CI workflow
k6delta run --app web --phase load --ci
RESULT=$?
if [ $RESULT -ne 0 ]; then
  echo "Load test failed"
  exit 1
fi
```

### Verdict Thresholds

Configure in `k6delta.yaml`:

```yaml
verdicts:
  cpu_peak_warn: 90         # CPU % to trigger WARN
  cpu_peak_fail: 98         # CPU % to trigger FAIL
  p95_regression_warn: 10   # p95 regression % for WARN
  p95_regression_fail: 25   # p95 regression % for FAIL
  error_rate_warn: 1.0      # Error rate % for WARN
```

---

## Configuration

### k6delta.yaml

```yaml
provider: ecs                  # ecs | docker-compose | mock
region: eu-west-1              # AWS region (ECS only)

defaults:
  env: staging                 # default --env value
  phase: smoke                 # default --phase value
  results_dir: results         # output directory

verdicts:
  cpu_peak_warn: 90
  p95_regression_warn: 10

apps:
  web:
    cluster: "web-${env}"
    service: "web-${env}"
    test_file: "tests/web/${phase}.js"
    # Optional (silently skipped if missing):
    # asg_prefix: "web-${env}-ecs-"
    # capacity_provider: "web-${env}-ec2"
    # alarm_prefix: "web-${env}"

  api:
    cluster: "api-${env}"
    service: "api-${env}"
    test_file: "tests/api/${phase}.js"
```

### Variable Interpolation

Config values support `${var}` expansion at runtime:

| Variable   | Source                                  |
|------------|-----------------------------------------|
| `${app}`   | Application name (from `--app` or config) |
| `${env}`   | Environment (from `--env` or defaults)  |
| `${phase}` | Test phase (from `--phase` or defaults) |

Example: `test_file: "tests/${app}/${phase}.js"` with `--app web --phase load` expands to `tests/web/load.js`.

### Provider Configurations

**AWS ECS**

```yaml
provider: ecs
region: us-east-1

apps:
  web:
    cluster: "web-${env}"
    service: "web-${env}"
    test_file: "tests/web/${phase}.js"
    asg_prefix: "web-${env}-ecs-"
    capacity_provider: "web-${env}-ec2"
    alarm_prefix: "web-${env}"
```

**Docker Compose**

```yaml
provider: docker-compose

apps:
  web:
    compose_project: "myapp"
    test_file: "tests/${phase}.js"
```

**Mock (for testing/demos)**

```yaml
provider: mock

apps:
  web:
    mock_scenario: "happy-path"
    test_file: "tests/${phase}.js"
```

### Default Fallback Chain

| Setting     | Priority                                       |
|-------------|------------------------------------------------|
| Config file | `--config` flag > `k6delta.yaml` in cwd        |
| Environment | `--env` flag > `defaults.env` > `staging`       |
| Region      | `--region` flag > `region` in config > `us-east-1` |
| Phase       | `--phase` flag > `defaults.phase` > `smoke`     |
| Results dir | `defaults.results_dir` > `results`              |

---

## Workflows

### First-Time Setup

```bash
k6delta init                          # create k6delta.yaml
vim k6delta.yaml                      # edit with your settings
k6delta run --app web --phase smoke   # first smoke test
```

### Baseline + Regression Check

```bash
# Establish a baseline
k6delta run --app web --phase load

# ... deploy changes ...

# Run again and compare
k6delta run --app web --phase load
k6delta compare results/baseline-report.json results/latest-report.json
```

### Incident Investigation

```bash
# Check what happened in the last hour
k6delta analyze --app web --duration 60

# Check specific time window with fine granularity
k6delta analyze --app web \
  --start 2026-03-09T14:00:00Z \
  --end 2026-03-09T14:30:00Z \
  --period 10
```

### CI/CD Pipeline

```bash
# Run load test and fail on FAIL verdict
k6delta run --app web --phase load --ci || exit 1

# Or with comparison against a committed baseline
k6delta compare baseline.json latest.json --ci || exit 1
```
