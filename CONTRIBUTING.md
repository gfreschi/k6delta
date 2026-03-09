# Contributing to k6delta

Thank you for considering contributing to k6delta! This guide explains how to get started.

## Code of Conduct

Be respectful and constructive. We are all here to build something useful.

## Getting Started

### Prerequisites

- Go 1.25+
- GNU Make
- [k6](https://k6.io/) (for integration tests)

### Setup

```bash
git clone https://github.com/gfreschi/k6delta.git
cd k6delta
make build
make test
```

### Useful Commands

```bash
make build        # Build binary with version injection
make test         # Run all unit tests
make lint         # Run go vet
make clean        # Remove build artifacts
```

## Development Workflow

1. Fork the repository and clone your fork
2. Create a feature branch from `main`:

   ```bash
   git checkout -b feat/my-feature
   ```

3. Make your changes
4. Ensure all checks pass:

   ```bash
   make lint && make test
   ```

5. Commit using [Conventional Commits](#commit-conventions)
6. Push and open a Pull Request against `main`

## Commit Conventions

All commits **must** follow [Conventional Commits](https://www.conventionalcommits.org):

```
type(scope): short description
```

- **Subject line:** imperative mood, lowercase, no period, max 72 characters
- **Body (optional):** wrap at 72 characters, explain *why* not *what*

### Types

| Type | Usage |
|------|-------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `test` | Adding or updating tests |
| `docs` | Documentation only |
| `ci` | CI/CD configuration |
| `build` | Build system or dependencies |
| `perf` | Performance improvement |
| `chore` | Maintenance tasks |

### Scope

Use the package or area name: `config`, `provider`, `tui`, `cli`, `report`, `k6`, `release`.

### Examples

```
feat(tui): add elapsed timers to step tracker
fix(provider): handle missing ASG prefix gracefully
ci(release): add GoReleaser release pipeline
test(report): add comparison edge case tests
docs: update README installation instructions
```

### Sign-off (DCO)

All commits **must** include a sign-off line for the [Developer Certificate of Origin](https://developercertificate.org/):

```bash
git commit -s -m "feat(tui): add elapsed timers to step tracker"
```

This adds:

```
Signed-off-by: Your Name <your.email@example.com>
```

If you forget, amend the last commit:

```bash
git commit --amend -s --no-edit
```

## Pull Requests

### Before Submitting

- [ ] Code compiles: `make build`
- [ ] All tests pass: `make test`
- [ ] Linter is clean: `make lint`
- [ ] New code includes tests
- [ ] Commits follow [Conventional Commits](#commit-conventions) with sign-off

### PR Guidelines

- Keep PRs focused - one feature or fix per PR
- Write a clear description of *what* and *why*
- Reference related issues (e.g., `Closes #42`)
- For larger changes, open an issue first to discuss the approach

## Code Standards

- Format with `gofmt` (standard Go formatting)
- All packages live under `internal/` - nothing is exported
- Follow existing patterns: interfaces for providers, `ResolvedApp` for config, graceful degradation for optional fields
- Prefer table-driven tests

## Reporting Issues

- Use [GitHub Issues](https://github.com/gfreschi/k6delta/issues)
- Include: Go version, OS, steps to reproduce, expected vs actual behavior
- For feature requests, describe the use case before the solution

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
