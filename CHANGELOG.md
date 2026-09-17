# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- GitHub Actions CI (`go test`, `go vet`, build + vendor-selection smoke) on Linux and macOS
- GoReleaser workflow for tagged `v*` releases
- Community docs: `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, issue templates
- `docs/QUICKSTART.md` vendor-selection walkthrough

### Changed

- Module path aligned to `github.com/lazarok09/ahp-method` for public `go install`
- Documented minimum Go version as **1.25.5** (required by `mcp-go`)
- `ahp version` can be injected at release via `-ldflags -X main.version=…`

## [0.3.0] - 2026-09-16

### Added

- Local Go `ahp` CLI: CSV/TOML workspace, eigenvector weights, consistency ratio, repair hints
- HTML report (`output/report.html`) and `ahp open`
- Stdio MCP server (`ahp mcp`) sharing `internal/workspace` with the CLI
- Orientation commands: `status`, `doctor` (`validate` alias), `tree`, `next`
- Example workspace: `examples/vendor-selection`

[Unreleased]: https://github.com/lazarok09/ahp-method/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/lazarok09/ahp-method/releases/tag/v0.3.0
