# Golden File Tests

Golden file tests snapshot TUI view output and detect visual regressions. Each test drives a Bubble Tea model to a specific state, captures `View()` output, and compares it against a stored `.golden` file.

## Quick Reference

```bash
# Run all golden tests
go test ./internal/tui/... -v

# Run golden tests for a specific view
go test ./internal/tui/run/... -run Golden -v
go test ./internal/tui/analyze/... -run golden -v
go test ./internal/tui/compare/... -run golden -v

# Regenerate ALL golden files
UPDATE_GOLDEN=1 go test ./internal/tui/... -v

# Regenerate golden files for a single view
UPDATE_GOLDEN=1 go test ./internal/tui/run/... -run Golden -v
UPDATE_GOLDEN=1 go test ./internal/tui/analyze/... -run golden -v
UPDATE_GOLDEN=1 go test ./internal/tui/compare/... -run golden -v

# Regenerate a single golden file
UPDATE_GOLDEN=1 go test ./internal/tui/compare/... -run TestCompareModel_goldenDrillDown -v

# Makefile shortcuts
make test-tui     # run all TUI tests (includes golden)
make test-update  # regenerate all golden files (UPDATE_GOLDEN=1)
```

## Test Inventory

### Run view (`internal/tui/run/`)

| Test | Terminal | Layout | State |
|------|----------|--------|-------|
| `TestRunModel_reportDashboard` | 120x40 | Split (2x2 panels) | Post-run report with k6 data, infra metrics, verdict |
| `TestRunModel_reportDashboard_stacked` | 80x24 | Stacked (vertical) | Same data, narrow terminal fallback |

**File:** `internal/tui/run/run_golden_test.go`
**Golden files:** `internal/tui/run/testdata/`

### Analyze view (`internal/tui/analyze/`)

| Test | Terminal | Layout | State |
|------|----------|--------|-------|
| `TestAnalyzeModel_goldenHappyPath` | 120x40 | Split (panels side-by-side) | Full dashboard: state, metrics, activities timeline |
| `TestAnalyzeModel_goldenHappyPath_stacked` | 80x24 | Stacked (vertical) | Same data, narrow terminal fallback |

**File:** `internal/tui/analyze/analyze_test.go`
**Golden files:** `internal/tui/analyze/testdata/`

### Compare view (`internal/tui/compare/`)

| Test | Terminal | Layout | State |
|------|----------|--------|-------|
| `TestCompareModel_goldenSplit` | 120x40 | Split (panels side-by-side) | Delta table with KPI strip, regression verdict |
| `TestCompareModel_goldenStacked` | 80x24 | Stacked (vertical) | Same data, narrow terminal |
| `TestCompareModel_goldenSideBySide` | 150x40 | Wide (diff mode) | Side-by-side A/B diff (triggered by `d` key) |
| `TestCompareModel_goldenDrillDown` | 120x40 | Split + drill-down | Enter key drill-down with A/B sparklines |

**File:** `internal/tui/compare/compare_test.go`
**Golden files:** `internal/tui/compare/testdata/`

## How It Works

### Golden file helper

The `internal/tui/golden` package provides three assertion functions:

- `RequireEqual(t, actual)` - compares against `testdata/{TestName}.golden`
- `RequireEqualNamed(t, name, actual)` - compares against `testdata/{name}.golden`
- `RequireEqualIn(t, dir, actual)` - compares against `{dir}/{TestName}.golden`

When `UPDATE_GOLDEN=1` is set, actual output is written to the golden file instead of compared.

### Deterministic test data

The `internal/tui/testutil` package provides fixed test data:

- `ReferenceTime` - `2026-01-15T10:00:00Z`
- `ResolvedApp()` - deterministic app config (test-app, staging, test-cluster)
- `SampleSnapshot()` - fixed ECS snapshot (4 tasks, 2 ASG instances)
- `SampleMetrics()` - fixed CPU/memory metric series
- `SampleActivities()` - fixed scaling + alarm events
- `VerdictConfig()` - default verdict thresholds

### Test pattern

Each golden test follows the same pattern:

1. Create a model with deterministic data
2. Send `tea.WindowSizeMsg` to set terminal dimensions
3. Drive the model through its state machine (auth, snapshot, metrics, activities)
4. Capture `model.View()` output
5. Compare against the golden file via `golden.RequireEqual`

## CI Integration

The CI pipeline (`.github/workflows/ci.yaml`) includes a golden drift check:

```yaml
- name: Verify golden files are up to date
  run: |
    UPDATE_GOLDEN=1 go test ./internal/tui/... -count=1
    if ! git diff --quiet internal/tui/**/testdata/*.golden 2>/dev/null; then
      git diff internal/tui/**/testdata/*.golden
    fi
```

This regenerates golden files and fails if the output differs from what's committed, catching cases where code changes weren't accompanied by golden file updates.

## When to Regenerate

Regenerate golden files (`make test-update`) after any change that affects TUI rendering:

- Layout changes (breakpoints, panel dimensions, spacing)
- Style changes (colors, borders, alignment)
- Content changes (new tiles, labels, formatting)
- Component changes (table columns, empty states, chart sizing)

Always review the diff after regeneration to confirm changes are intentional:

```bash
make test-update
git diff internal/tui/**/testdata/*.golden
```
