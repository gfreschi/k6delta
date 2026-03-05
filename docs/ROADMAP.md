# k6delta Roadmap Tracker

**Last updated:** 2026-03-05
**Design docs:**

- [Roadmap design](design/roadmap.md) — features & architecture
- [TUI UI/UX design](design/tui-ui-ux.md) — visual & interaction design
**Conventions:** [docs/conventions/go-conventions.md](conventions/go-conventions.md) (Section 31: TUI Architecture)

---

## Phase 0.1 — "Foundation" (COMPLETE)

**Plan:** [docs/phases/0.1-foundation.md](phases/0.1-foundation.md)

- [x] ECS provider (CheckCredentials, TakeSnapshot, FetchMetrics, FetchActivities)
- [x] Config system (YAML loading, variable interpolation, CLI flag merge)
- [x] k6 process execution (BuildArgs, BuildEnv, exit code capture)
- [x] Unified report schema (k6 + infra metrics JSON)
- [x] Report comparison with percentage deltas
- [x] TUI run model (6-phase state machine, progress callbacks, verdict)
- [x] TUI analyze model (standalone CloudWatch queries)
- [x] TUI compare model (side-by-side diff)
- [x] `k6delta run` command
- [x] `k6delta analyze` command
- [x] `k6delta compare` command
- [x] `k6delta init` command
- [x] CI/CD (GitHub Actions: lint, test, build matrix)
- [x] GoReleaser (cross-platform binaries, checksums, changelog)
- [x] Verdict system (PASS/WARN/FAIL with reasons)

---

## Phase 0.1a — "Theme & Layout Engine" (COMPLETE)

**Design:** [docs/design/tui-ui-ux.md](design/tui-ui-ux.md)
**Plan:** [docs/phases/0.1a-theme-layout.md](phases/0.1a-theme-layout.md)
**Goal:** Centralized theme, reusable components, professional visual foundation.
**Branch:** `feat/tui-ui-0.1.1`

### Theme system (gh-dash architecture)

- [x] Create `internal/tui/theme/` package (AdaptiveColor, DefaultTheme)
- [x] Create `internal/tui/context/` package (ProgramContext, Styles, InitStyles)
- [x] Create `internal/tui/common/` package (CommonStyles, BuildStyles, glyphs)
- [x] Create `internal/tui/constants/` package (Dimensions, icons, layout constants)

### Key bindings

- [x] Create `internal/tui/keys/` package (universal KeyMap, per-view key maps)

### Reusable components

- [x] Create `internal/tui/components/table/` (columns, rows, auto-sizing, alternating rows)
- [x] Create `internal/tui/components/footer/` (context-sensitive keybinding bar)
- [x] Create `internal/tui/components/header/` (app/env/phase context bar)
- [x] Create `internal/tui/components/panel/` (bordered panel with focus state)
- [x] Migrate `internal/tui/step.go` → `internal/tui/components/stepper/`

### View migration

- [x] Refactor `internal/tui/run/model.go` to use theme + components
- [x] Refactor `internal/tui/analyze/model.go` to use theme + components
- [x] Refactor `internal/tui/compare/model.go` to use theme + components
- [x] Remove old `internal/tui/styles.go` (replaced by theme/context/common)

---

## Phase 0.1b — "Live Run Dashboard" (COMPLETE)

**Design:** [docs/design/tui-ui-ux.md](design/tui-ui-ux.md)
**Plan:** [docs/phases/0.1b-live-dashboard.md](phases/0.1b-live-dashboard.md)
**Goal:** Split-screen with live ASCII graphs and infrastructure metrics during k6 execution.
**Branch:** `feat/tui-ui-0.1.1`

### Live graphs

- [x] Create `internal/tui/components/linechart/` (Unicode line chart, rolling window, auto-scale Y-axis)
- [x] Create `internal/tui/components/gauge/` (progress bar with threshold coloring)

### Run dashboard

- [x] Implement header bar with elapsed time
- [x] Implement split-screen content: graphs (left) + infrastructure panel (right)
- [x] Implement health bar (live verdict conditions: CPU, tasks, 5xx)
- [x] k6 JSON stream parsing (replace `tea.ExecProcess` with `--out json=-` pipe)
- [x] Infrastructure polling goroutine (15s interval via `tea.Tick`)
- [x] `g` key to toggle graphs ↔ stepper view
- [x] `a` key to abort test (context cancellation to k6)
- [x] Fallback to `tea.ExecProcess` when k6 JSON streaming unavailable

### Responsive layout

- [x] `>= 120` cols: full split (side-by-side)
- [x] `80-119` cols: vertical stack (graphs then infra)
- [x] `< 80` cols: step tracker fallback

---

## Phase 0.1c — "Interactive Report Dashboard" (COMPLETE)

**Design:** [docs/design/tui-ui-ux.md](design/tui-ui-ux.md)
**Plan:** [docs/phases/0.1c-interactive-report.md](phases/0.1c-interactive-report.md)
**Goal:** Navigable post-run dashboard replacing static text dump.
**Branch:** `feat/tui-ui-0.1.1`

### Report panels

- [x] k6 Summary panel (2-column metric grid)
- [x] Infrastructure panel (snapshots + CloudWatch peaks, scrollable)
- [x] Scaling Events panel (chronological list, scrollable)
- [x] Verdict bar (compact inline verdict with reasons)

### Navigation and actions

- [x] Tab/Shift-Tab panel focus cycling
- [x] ↑/↓ scroll within focused panel
- [x] Scroll position in panel title
- [x] `e` export unified report JSON
- [x] `o` open k6 HTML dashboard in browser
- [x] `r` toggle raw text view

### Analyze view

- [x] Interactive panel dashboard (state, metrics, activities panels)

---

## Phase 0.1d — "Compare Heatmap & Polish" (COMPLETE)

**Design:** [docs/design/tui-ui-ux.md](design/tui-ui-ux.md)
**Plan:** [docs/phases/0.1d-compare-polish.md](phases/0.1d-compare-polish.md)
**Goal:** Color intensity heatmap for compare, animation polish, responsive finalization.
**Branch:** `feat/tui-ui-0.1.1`

### Compare heatmap

- [x] Color intensity delta styles (5 tiers per direction: neutral, mild, moderate, severe, strong)
- [x] Percentage delta column alongside absolute delta
- [x] Summary strip (improved/regressed/unchanged counts, worst regression highlight)
- [x] `s` key to sort by delta magnitude

### Animation polish (cross-view)

- [x] Panel border fade-in on focus change (~150ms)
- [x] Step completion flash (1 frame bright, then settle)
- [x] Spinner upgrade (MiniDot or Ellipsis)
- [x] Scroll indicator flash on panel focus

### Responsive reflow (cross-view)

- [x] Finalize all breakpoints across run, report, compare views
- [x] `< 80` fallback to static output in all views

---

## Phase 0.1.1a — "Chart Engine" (COMPLETE)

**Design:** [docs/plans/2026-03-04-0.1.1-tui-quality-design.md](plans/2026-03-04-0.1.1-tui-quality-design.md)
**Plan:** [docs/phases/0.1.1a-chart-engine.md](phases/0.1.1a-chart-engine.md)
**Goal:** Replace garbled custom Unicode chart with ntcharts braille rendering. Fix chart sizing.

### ntcharts integration

- [x] Task 1: Add `ntcharts` v0.4.0 dependency
- [x] Task 2: Create `internal/tui/components/streamchart/` (TimeSeriesChart wrapper, braille, Push+Draw)
- [x] Task 3: Create `internal/tui/components/trendline/` (Sparkline wrapper, braille, Push+Draw)
- [x] Task 4: Create `internal/tui/components/timechart/` (static time-series graphs for reports)

### Migration

- [x] Task 5: Replace `linechart` usage in `run/model.go` with `streamchart` (dynamic sizing, timestamp Push)
- [x] Task 6: Delete `internal/tui/components/linechart/`
- [x] Task 7: Verify full build and lint

---

## Phase 0.1.1b — "Live Dashboard Overhaul" (COMPLETE)

**Design:** [docs/plans/2026-03-04-0.1.1-tui-quality-design.md](plans/2026-03-04-0.1.1-tui-quality-design.md)
**Plan:** [docs/phases/0.1.1b-live-dashboard.md](phases/0.1.1b-live-dashboard.md)
**Goal:** Tab navigation, infra sparklines, gauge 0% fix, visual consistency with report view.
**Branch:** `feat/tui-ui-0.1.2`

### Panel navigation

- [x] Task 1: Add `focus.Manager` + panel wrapping for live dashboard (Tab/Shift+Tab, border animation)

### Infra improvements

- [x] Task 2: Add CPU/memory/reservation trend sparklines below gauges
- [x] Task 3: Fix gauge 0% bug ("—" fallback, 2× CloudWatch lookback)

### Consistency

- [x] Task 4: Update live footer with panel navigation hints
- [x] Task 5: Remove debug "GRAPH" label (not found — already clean)
- [x] Task 6: Responsive chart/panel sizing on terminal resize
- [x] Task 7: Verify full build and lint

---

## Phase 0.1.1c — "Post-Run Report Enhancement" (COMPLETE)

**Design:** [docs/plans/2026-03-04-0.1.1-tui-quality-design.md](plans/2026-03-04-0.1.1-tui-quality-design.md)
**Plan:** [docs/phases/0.1.1c-report-enhancement.md](phases/0.1.1c-report-enhancement.md)
**Goal:** Fix k6 data bug, add time-series graphs, 4-panel layout, graceful degradation.
**Branch:** `feat/tui-ui-0.1.2`

### Bug fixes

- [x] Task 1: Fix k6 data pipeline — added `--summary-export` to k6 args so summary JSON is always written

### Report graphs

- [x] Task 2: Add Graphs panel with braille throughput + latency time-series (timechart wrapper)
- [x] Task 3: Implement 2×2 four-panel report layout (k6 / graphs / infra / events)

### Polish

- [x] Task 4: Graceful degradation for partial data (exit code context, "metrics pending")
- [x] Task 5: Panel styling consistency across live and report views (verified clean)
- [x] Task 6: Verify full build and lint

---

## Phase 0.2a — "Foundations & CI Mode" (COMPLETE)

**Plan:** [docs/phases/0.2a-foundations-ci.md](phases/0.2a-foundations-ci.md)
**Goal:** Codebase fixes, verdict extraction, configurable thresholds, CI mode. No new external dependencies.
**Branch:** `feat/docker-0.1.3`

### Codebase improvements

- [x] Task 1: Fix `errors.As` in k6 runner (+ stream_runner.go)
- [x] Task 2: Add compile-time interface check for ECS provider
- [x] Task 3: Align k6 package name with directory (`k6runner` → `k6`)
- [x] Task 4: Update main package comment
- [x] Task 5: Convert TestValidatePhase to table-driven

### Verdict and config

- [x] Task 6: Add VerdictConfig to config system (WithDefaults, 7 thresholds)
- [x] Task 7: Extract verdict into `internal/verdict/` shared package
- [x] Task 8: Wire VerdictConfig into extracted verdict (TUI + CI)

### CI mode

- [x] Task 9: Add `--ci` flag to run command (JSON to stdout, exit 0=PASS/WARN, 1=FAIL)
- [x] Task 10: Add `--ci` flag to compare command
- [x] Task 11: Add `--ci` flag to analyze command
- [x] Task 12: Integration tests for CI mode (3 help-flag tests)
- [x] Task 13: Update example config with verdict section

---

## Phase 0.2b — "Docker Compose Provider" (COMPLETE)

**Plan:** [docs/phases/0.2b-docker-compose.md](phases/0.2b-docker-compose.md)
**Goal:** Docker Compose provider so anyone can try k6delta without AWS.
**Branch:** `feat/docker-0.1.3`

### Config and dependency

- [x] Task 1: Add `compose_project` to config
- [x] Task 2: Add Docker client dependency (`github.com/moby/moby/client v0.2.2`)

### Provider implementation

- [x] Task 3: Scaffold provider (types, constructor, interface check)
- [x] Task 4: Implement CheckCredentials (Docker ping + project validation)
- [x] Task 5: Implement TakeSnapshot (container counts)
- [x] Task 6: Implement FetchMetrics (CPU/memory via container stats)
- [x] Task 7: Implement FetchActivities (Docker events API)

### CLI and testing

- [x] Task 8: Wire Docker Compose provider into CLI factory
- [x] Task 9: Example config + integration test
- [x] Task 10: Final validation

### Post-0.2b Code Review Fixes

12 items addressed from code review of phases 0.2a + 0.2b:

- [x] I-1: Move `os.Exit()` out of `runCI` into `main.go` via `ExitError` type
- [x] I-2: Implement compare `--ci` regression verdict with exit code (p95/error_rate thresholds)
- [x] I-3: Document why FetchMetrics ignores time parameters in Compose provider
- [x] I-4: Propagate Docker Events errors in FetchActivities (was silently swallowed)
- [x] S-1: Rename ECSScaling/ASGScaling to ServiceScaling/NodeScaling in provider.Activities
- [x] S-2: Use pointer types in VerdictConfig for zero-value support (`verdictYAML` → `VerdictConfig`)
- [x] S-3: Fix ECScPUPeak naming to ECSCPUPeak
- [x] S-4: Define named `dockerEventEmitter` interface for activities
- [x] S-5: Add mock tests for `fetchMetrics/ActivitiesWithClient` (4 new tests)
- [x] S-6: Rename ECSCPU/ECSMemory to ServiceCPU/ServiceMemory, standardize metric IDs to `service_cpu`/`service_memory`
- [x] S-7: Update example config comment to list supported providers
- [x] S-8: Move `config_test.go` to external test package

---

## Phase 0.1.4a — "Mock Provider + Scenario Engine" (COMPLETE)

**Plan:** [docs/phases/0.1.4a-mock-provider.md](phases/0.1.4a-mock-provider.md)
**Goal:** Mock InfraProvider with synthetic scenarios for TUI development and testing without real infrastructure.

### Mock provider

- [x] Task 1: Composable time-series generators (Constant, Sine, Ramp, Step, Noise, Sample)
- [x] Task 2: Scenario definitions (happy-path, cpu-spike, scale-out, cascade-failure)
- [x] Task 3: InfraProvider implementation (scenario-driven snapshots, metrics, activities)

### Config and CLI

- [x] Task 4: Add `mock_scenario` config field to AppConfig/ResolvedApp
- [x] Task 5: Wire mock provider into CLI factory (`provider: mock`)

### Testing

- [x] Task 6: Integration test (mock provider dry-run)
- [x] Task 7: Final validation (full suite, lint, build, integration)

---

## Phase 0.1.4b — "Demo Command" (COMPLETE)

**Plan:** [docs/phases/0.1.4b-demo-command.md](phases/0.1.4b-demo-command.md)
**Goal:** `k6delta demo` command that runs the full TUI pipeline with mock provider and fake k6 stream, requiring zero infrastructure.

### Fake k6 stream

- [x] Task 1: FakeStream generator (synthetic K6Points with speed multiplier and scenario curves)

### Demo command

- [x] Task 2: Demo command scaffold (`--scenario`, `--speed`, `--list` flags)
- [x] Task 3: NewDemoModel in run TUI (reuses full state machine with fake k6 streaming)
- [x] Task 4: Stepper labels for demo mode ("Mock credentials", "Running demo")

### Testing

- [x] Task 5: Integration tests (--list and --help flag verification)
- [x] Task 6: Final validation (full suite, lint, build, integration)

---

## Phase 0.1.4c — "TUI Golden Tests" (COMPLETE)

**Design:** [docs/plans/2026-03-05-test-harness-dev-sandbox-design.md](plans/2026-03-05-test-harness-dev-sandbox-design.md)
**Plan:** [docs/phases/0.1.4c-tui-golden-tests.md](phases/0.1.4c-tui-golden-tests.md)
**Goal:** Deterministic golden file snapshot tests for all TUI views.

### Golden file framework

- [x] Task 1: Skipped teatest dependency (unused — golden tests use manual model driving)
- [x] Task 2: Golden file helper (`internal/tui/golden/`) — RequireEqual with UPDATE_GOLDEN=1 env var
- [x] Task 3: Deterministic test data (`internal/tui/testutil/`) — fixed time, metrics, snapshots

### View golden tests

- [x] Task 4: Compare model golden tests (split 120x40 + stacked 80x24)
- [x] Task 5: Analyze model golden tests (happy-path at 120x40)
- [x] Task 6: Run model golden tests (report dashboard at 120x40, all 4 panels + verdict)
- [x] Task 7: Final validation (full suite, lint, golden regeneration drift check)

---

## Phase 0.1.4d — "CI Fast Path" (IN PROGRESS)

**Plan:** [docs/phases/0.1.4d-ci-fast-path.md](phases/0.1.4d-ci-fast-path.md)
**Goal:** Fast test suite by default, golden file drift detection in CI.

### CI and build

- [x] Task 1: Add Makefile targets (test-all, test-update, test-tui)
- [x] Task 2: Golden file drift check in CI
- [x] Task 3: Responsive breakpoint golden tests (80x24)
- [x] Task 4: Update example config with mock provider
- [x] Task 5: Update ROADMAP.md
- [ ] Task 6: Final validation

---

## Phase 0.1.7 — "Production AWS" (PENDING)

**Plan:** [docs/phases/0.1.7-production-aws.md](phases/0.1.7-production-aws.md)
**Goal:** Complete AWS story with EKS + cost estimation.

### EKS provider

- [ ] Task 1: Scaffold EKS provider package
- [ ] Task 2: Add EKS config fields (namespace, deployment, hpa_name)
- [ ] Task 3: Implement EKS TakeSnapshot (pod/node counts)
- [ ] Task 4: Implement EKS FetchMetrics (Container Insights)
- [ ] Task 5: Implement EKS FetchActivities (HPA events)
- [ ] Task 6: Wire EKS provider into CLI

### Cost estimation

- [ ] Task 7: Create cost estimation package (Fargate pricing)
- [ ] Task 8: Add cost section to unified report

### Report command

- [ ] Task 9: Implement `k6delta report` command
- [ ] Task 10: Final validation

---

## Phase 0.1.8 — "Capacity Planning" (PENDING)

**Plan:** [docs/phases/0.1.8-capacity-planning.md](phases/0.1.8-capacity-planning.md)
**Goal:** k6delta answers "how much infra do I need?"

- [ ] Task 1: Create run history store package (JSON files in ~/.k6delta/runs/)
- [ ] Task 2: Wire store into run command
- [ ] Task 3: Create capacity planning package (linear extrapolation)
- [ ] Task 4: Implement `k6delta plan` command
- [ ] Task 5: Final validation

---

## Phase 0.1.9 — "Universal Metrics" (PENDING)

**Plan:** [docs/phases/0.1.9-universal-metrics.md](phases/0.1.9-universal-metrics.md)
**Goal:** Non-AWS users can use k6delta.

### OpenTelemetry provider

- [ ] Task 1: Scaffold OTel provider package
- [ ] Task 2: Add OTel config fields (prometheus_endpoint, metric_queries)
- [ ] Task 3: Implement CheckCredentials and FetchMetrics (PromQL)
- [ ] Task 4: Implement TakeSnapshot and FetchActivities

### Generic Kubernetes provider

- [ ] Task 5: Create generic K8s provider (client-go + OTel delegation)

### CLI and trend

- [ ] Task 6: Wire OTel and K8s providers into CLI
- [ ] Task 7: Implement `k6delta trend` command
- [ ] Task 8: Final validation

---

## Phase 0.2 — "Stable" (PENDING)

**Plan:** [docs/phases/0.2-stable.md](phases/0.2-stable.md)
**Goal:** Stable API, production-grade for teams.

### Public API and plugins

- [ ] Task 1: Export InfraProvider API (move to public package)
- [ ] Task 2: Implement exec-based plugin system
- [ ] Task 3: Wire plugin provider into CLI

### Documentation and community

- [ ] Task 4: Create documentation site content
- [ ] Task 5: Create GitHub Action (`k6delta-action`)
- [ ] Task 6: Create CONTRIBUTING.md
- [ ] Task 7: Restructure README for 1.0
- [ ] Task 8: Add issue and PR templates
- [ ] Task 9: Final validation

---

## Progress Summary

| Phase | Status | Tasks | Done |
|-------|--------|-------|------|
| 0.1 Foundation | COMPLETE | 15 | 15 |
| 0.1a Theme & Layout | COMPLETE | 14 | 14 |
| 0.1b Live Dashboard | COMPLETE | 12 | 12 |
| 0.1c Interactive Report | COMPLETE | 11 | 11 |
| 0.1d Compare & Polish | COMPLETE | 10 | 10 |
| 0.1.1a Chart Engine | COMPLETE | 7 | 7 |
| 0.1.1b Live Dashboard Overhaul | COMPLETE | 7 | 7 |
| 0.1.1c Report Enhancement | COMPLETE | 6 | 6 |
| 0.2a Foundations & CI | COMPLETE | 13 | 13 |
| 0.2b Docker Compose | COMPLETE | 10 | 10 |
| 0.1.4a Mock Provider | COMPLETE | 7 | 7 |
| 0.1.4b Demo Command | COMPLETE | 6 | 6 |
| 0.1.4c TUI Golden Tests | COMPLETE | 7 | 7 |
| 0.1.4d CI Fast Path | IN PROGRESS | 6 | 5 |
| 0.1.7 Production AWS | PENDING | 10 | 0 |
| 0.1.8 Capacity Planning | PENDING | 5 | 0 |
| 0.1.9 Universal Metrics | PENDING | 8 | 0 |
| 0.2 Stable | PENDING | 9 | 0 |
| **Total** | | **163** | **130** |
