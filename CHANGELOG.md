# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Commit-level release notes are generated with [`goreleaser/chglog`](https://github.com/goreleaser/chglog)
from `changelog.yml` (`chglog format -t release`) and attached to GitHub Releases.

## [Unreleased]

## [0.3.0] - 2026-09-18

First public release of the local Go `ahp` CLI + MCP server.

### Added

- Local Go `ahp` CLI: CSV/TOML workspace, eigenvector weights, consistency ratio, repair hints
- HTML report (`output/report.html`) and `ahp open`
- Stdio MCP server (`ahp mcp`) sharing `internal/workspace` with the CLI
- Orientation commands: `status`, `doctor` (`validate` alias), `tree`, `next`
- Judgment workflow: `pair` (missing/set/repairs/ask/import), `plan`, `apply`, `get`/`describe`, `catalog`
- Decision-quality: `explain`, `sensitivity`
- Attribute bridge: `ahp rate` / `pair suggest-from-attributes` (including `--refresh`)
- `ahp constrain` eligibility bands; out-of-band alts excluded from synthesis
- `ahp doctor --purchase`: FX/foreign source, missing unit/source, pairwise vs attribute direction
- MCP tools: `constrain`, `import_pairwise`; `suggest_from_attributes.refresh`
- Example workspace: `examples/vendor-selection`
- GitHub Actions CI (`go test`, `go vet`, build + vendor-selection smoke) on Linux and macOS
- GoReleaser workflow for tagged `v*` releases with `chglog` release notes
- Community docs: `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, issue templates
- `docs/QUICKSTART.md` vendor-selection walkthrough

### Changed

- `ahp apply` defaults to a summary (`committed`, `cr_ok`, ranking peek); use `--verbose` for full rows
- Status readiness is structure-aware (`complete`, proposal coverage, `ranking_mode`)
- Module path aligned to `github.com/lazarok09/ahp-method` for public `go install`
- Documented minimum Go version as **1.25.5** (required by `mcp-go`)
- `ahp version` can be injected at release via `-ldflags -X main.version=…`

[Unreleased]: https://github.com/lazarok09/ahp-method/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/lazarok09/ahp-method/releases/tag/v0.3.0
