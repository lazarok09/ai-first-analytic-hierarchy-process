# Open-sourcing the `ahp` CLI

Plan to publish this Go CLI + MCP tool as a public open-source project. Aligned with [VISION.md](./VISION.md) and [ROADMAP.md](./ROADMAP.md): local binary, CSV workspace, no hosted Auth/DB/web app.

## Current readiness

| Item | Status |
|------|--------|
| MIT [`LICENSE`](../LICENSE) | Done (Copyright 2026 Lazaro Souza) |
| Root [`README.md`](../README.md) + docs | Done (CLI-focused; badges/install for `lazarok09/ahp-method`) |
| Engine / workspace tests | Pass (`go test ./...`) |
| Example workspace | [`examples/vendor-selection`](../examples/vendor-selection) |
| CI / releases | Workflows added (`.github/workflows/ci.yml`, `release.yml` + GoReleaser); not yet green on GitHub |
| Community files | Done (`CONTRIBUTING`, `SECURITY`, CoC, `CHANGELOG`, issue templates) |
| Module path ↔ GitHub remote | Module is `github.com/lazarok09/ahp-method`; remote still `…/ai-first-analytic-hierarchy-process` (rename pending) |
| Local `.env` | Relocated off-tree; gitignored; must never be committed |

### Critical mismatch (remaining)

- **Module / docs**: `github.com/lazarok09/ahp-method`
- **Git remote today**: `github.com/lazarok09/ai-first-analytic-hierarchy-process`

Rename the GitHub repo to `ahp-method` (and update `git remote`) so strangers can run:

```bash
go install github.com/lazarok09/ahp-method/cmd/ahp@latest
```

---

## Phase 0 — Decide identity (do first)

**Status: locked** — see [Decisions (locked)](#decisions-locked) (`A` / keep history / full day-one).

Pick one naming strategy and stick to it:

| Option | Action | Pros | Cons |
|--------|--------|------|------|
| **A** | Rename/move GitHub repo to `lazarok/ahp-method` (or `lazarok09/ahp-method`) | Matches `go.mod` + README | URL change / redirects |
| **B** | Keep current repo name; rewrite `go.mod` + all imports to the long path | No GitHub rename | Ugly module path; README churn |
| **C** | New public repo (`ahp` / `ahp-method`); push clean Go-only history | Clean OSS story; drop Next.js git baggage | Extra setup |

**Recommendation:** A or C with a short public name (`ahp-method` or `ahp`). Keep MIT unless there is a reason to change.

Also decide history strategy:

- **Keep history** — includes deleted Next.js / Auth / ChatKit commits (no `.env` found in history; optional hygiene only).
- **Orphan / squash to Go-only** — cleaner first impression for public readers.

---

## Phase 1 — Hygiene before public

1. Confirm `.env` stays untracked; under `.env.*` in `.gitignore`, add `!.env.example` so the example is not ignored by accident.
2. Delete or relocate the local `.env` before any broad `git add`.
3. Trim `.gitignore` for a Go OSS tree: keep `bin/`, `dist/`, `.env`; drop stale Python/Next noise if desired.
4. Keep `.cursor/` untracked (already the case).
5. Soften or verify docs: no promise of hosted Auth/ChatKit ([VISION.md](./VISION.md) already local-first).
6. Final secret scan on the working tree and, if keeping history, on old commits for API keys / `AUTH_SECRET`.

---

## Phase 2 — Installability and quality bar

1. Align module path with the public GitHub URL; `go mod tidy`; fix imports.
2. Pin and document a real minimum Go version (README says 1.22+; local toolchain may be newer — state the supported minimum clearly).
3. Add GitHub Actions CI:
   - `go test ./...`
   - `go vet ./...`
   - optional `golangci-lint`
   - matrix: linux (+ macOS if budget allows)
4. Smoke test in CI: `go build -o ahp ./cmd/ahp` then `./ahp compute -w examples/vendor-selection`.
5. Versioning: keep `v0.3.0` in sync with git tags; later inject version via `-ldflags` so `ahp version` matches the release tag.

---

## Phase 3 — Community and docs pack

| File | Purpose |
|------|---------|
| `CONTRIBUTING.md` | Build, test, PR expectations; note MCP and CLI share `internal/workspace` (no forked logic) |
| `SECURITY.md` | How to report vulnerabilities (email or GitHub private advisory) |
| `CODE_OF_CONDUCT.md` | Contributor Covenant or short custom |
| `CHANGELOG.md` | Keep a Changelog from `v0.3.0` onward |
| README polish | Badges (CI, license, Go version), one-liner pitch, install, 30-second demo, link to `docs/`, license footer |
| GitHub metadata | Description + topics: `ahp`, `mcp`, `cli`, `decision-making`, `golang` |

Optional: short `docs/QUICKSTART.md` walking through the vendor-selection example end-to-end.

Agent rule reminder for contributors ([AGENTS.md](../AGENTS.md)): prefer `ahp status` / `suggest_repairs`; `propose_pairwise` must keep `status=proposal`.

---

## Phase 4 — Distribution

1. **GoReleaser** + workflow on `v*` tags → GitHub Releases (linux/darwin/windows, amd64/arm64) with checksums.
2. README install section:
   - `go install …@latest`
   - download release binary
   - (later) Homebrew tap if non-Go users show up
3. After the first public tag, confirm the module appears on [pkg.go.dev](https://pkg.go.dev) (may need a proxy fetch).
4. Optional: MCP directory / Cursor marketplace listing once install is reliable.

---

## Phase 5 — Flip the switch

1. Push the aligned branch and tags.
2. Set the repository **Public**; add description (and homepage if useful).
3. Create a GitHub Release for `v0.3.0` (“local Go AHP CLI + MCP; CSV workspace; HTML report”).
4. Announce with the vendor-selection demo and/or a screenshot of `output/report.html`.
5. Enable Discussions or issue templates (bug / feature) so early contributors land soft.

---

## Phase 6 — Post-launch (do not block day one)

- Catalog / `ahp docs` from [ROADMAP.md](./ROADMAP.md) (better agent onboarding).
- CLI parity wrappers (`missing-pairs`, `suggest-repairs`).
- Dependabot for Go modules.
- CodeQL or basic security scanning.
- Homebrew / Scoop only if demand appears.

---

## Execution checklist

```text
[x] Choose public name + module path (A / B / C)
[x] Decide history strategy (keep vs Go-only root)
[x] Secret hygiene (.env, scan)
[x] Align go.mod / imports / README install
[ ] CI green on PR + main
[x] CONTRIBUTING + SECURITY + CHANGELOG (+ CoC)
[ ] GoReleaser + tag v0.3.0
[ ] Make repo public + GitHub Release
[ ] Verify go install + pkg.go.dev
[ ] Announce + watch first issues
```

---

## Decisions (locked)

1. **Module / repo name** — **A**: public name `ahp-method` under GitHub user `lazarok09`. Module path `github.com/lazarok09/ahp-method` (was `github.com/lazarok/ahp-method`). Rename remote from `ai-first-analytic-hierarchy-process` → `ahp-method`.
2. **History** — **Keep** existing history (secret scan found only placeholder Auth keys in old `.env.example`; no real secrets committed).
3. **Day-one scope** — **Full**: MIT + CI + community pack + GoReleaser + CoC in the same change set.
