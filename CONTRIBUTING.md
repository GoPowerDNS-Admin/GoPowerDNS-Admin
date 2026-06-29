# Contributing to GoPowerDNS-Admin

Thanks for your interest in contributing! This document explains how to get a development environment running, the conventions the project follows, and how to submit changes.

By participating, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## Ways to contribute

- **Report bugs** and **request features** via [GitHub issues](https://github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/issues) (please use the templates).
- **Ask questions** or share ideas in the issue tracker.
- **Improve docs** — both this repository and the documentation site.
- **Submit code** — bug fixes and features via pull requests.

For security vulnerabilities, **do not** open a public issue — follow the [Security Policy](SECURITY.md).

## Development setup

Prerequisites:

- Go (see the `go` directive in [`go.mod`](go.mod) for the required version) — no C toolchain required; all database drivers are pure Go.
- Docker and Docker Compose (optional, for a local PowerDNS / LDAP).

Optional contributor tooling (needed only for linting / pre-commit):

- [`golangci-lint`](https://golangci-lint.run/) — Go linting (`make linter`)
- [Biome](https://biomejs.dev/) — JavaScript linting for `internal/web/static/js/` (`make linter-js`)
- [`pre-commit`](https://pre-commit.com/) — runs golangci-lint, Biome, prettier, a secret scan, `go mod tidy`, and the conventional-commit check (`make pre-commit`)

Clone, build, and run:

```bash
git clone https://github.com/GoPowerDNS-Admin/GoPowerDNS-Admin.git
cd GoPowerDNS-Admin

# optional: start a local PowerDNS for development
docker compose up -d

# run in development mode (templates reloaded from disk on each request)
go run . start --dev
```

Configuration lives in `etc/main.toml`. For local testing, copy it into the gitignored `etc/local/` directory and adjust:

```bash
mkdir -p etc/local
cp etc/main.toml etc/local/main.toml
# edit etc/local/main.toml to match your setup
```

See the [documentation site](https://docs.gopowerdnsadmin.org) and the README for more detail on configuration, database backends, and authentication.

## Before you open a pull request

1. **Discuss large changes first.** For anything beyond a small fix, please open an issue to align on the approach before investing significant effort.
2. **Format and lint.** Run `make pre-commit` (or at minimum `gofmt`, `make linter`, and `make linter-js`) and make sure it passes.
3. **Add tests.** New behavior should come with tests; run `make test` (and `go test ./...`) and ensure everything is green.
4. **Keep PRs focused.** One logical change per PR is much easier to review than a large, mixed diff.
5. **Update docs.** If you change behavior, update the README and/or the docs site accordingly.

## Commit messages and PR titles

This project uses [Conventional Commits](https://www.conventionalcommits.org/). PR titles are validated by CI and must match:

```
^(feat|fix|docs|test|ci|chore)!?(\(scope\))?!?: summary
```

Allowed types: `feat`, `fix`, `docs`, `test`, `ci`, `chore`. Use a scope where it adds clarity (e.g. `feat(zone):`, `fix(dashboard):`), and append `!` for a breaking change. Examples:

- `feat(zone): add bulk record import`
- `fix(auth): reject expired OIDC tokens`
- `docs(readme): document reverse-zone search`

## Submitting the pull request

1. Fork the repository and create a branch from `main` (e.g. `feat/bulk-import` or `fix/oidc-expiry`).
2. Make your changes following the conventions above.
3. Push to your fork and open a PR against `main`, filling out the pull request template.
4. CI must pass (build, tests, linters, PR-title check). A maintainer will review and may request changes.

Thanks for helping make GoPowerDNS-Admin better!
