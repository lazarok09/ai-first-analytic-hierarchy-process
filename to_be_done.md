# To be done — human operator

Agent finished local OSS prep but **could not talk to GitHub** from this environment (no `gh` login; SSH key is passphrase-locked / not accepted). Do the remote Phase 5 steps yourself.

## Already done (local)

- Module path: `github.com/lazarok09/ahp-method`
- CI + GoReleaser workflows, community docs, README polish
- Commit on `main`: `fa7ad67 Prepare the CLI for public open-source publishing.`
- Branch is **3 commits ahead** of `origin/main` (also includes Phase A orientation + roadmap docs)
- Working tree may still have a small edit to `docs/OPEN_SOURCE.md` — commit or discard before push

## Your job

### 1. Authenticate in this repo’s shell

```bash
cd /home/lazarok/github/ai-first-analytic-hierarchy-process
gh auth login
# HTTPS + login via browser is fine
```

Or unlock SSH and ensure the pubkey is on your GitHub account:

```bash
ssh-add ~/.ssh/id_ed25519
ssh -T git@github.com
```

### 2. Commit leftover doc tweak (if any)

```bash
git status
git add docs/OPEN_SOURCE.md   # if still modified
git commit -m "Note remaining open-source remote checklist items."
# or: git restore docs/OPEN_SOURCE.md
```

### 3. Rename repo → `ahp-method`

Matches `go.mod` / README install path.

```bash
gh repo rename ahp-method --yes
git remote set-url origin https://github.com/lazarok09/ahp-method.git
# or SSH: git@github.com:lazarok09/ahp-method.git
```

### 4. Push `main`

```bash
git push -u origin HEAD:main
```

### 5. Make the repo public + metadata

```bash
gh repo edit lazarok09/ahp-method \
  --visibility public \
  --accept-visibility-change-consequences

gh repo edit lazarok09/ahp-method \
  --description "Local Go CLI + MCP for Analytic Hierarchy Process (CSV workspace, CR, HTML report)" \
  --add-topic ahp --add-topic mcp --add-topic cli --add-topic decision-making --add-topic golang
```

### 6. Tag `v0.3.0` (triggers GoReleaser)

```bash
git tag -a v0.3.0 -m "v0.3.0: local Go AHP CLI + MCP; CSV workspace; HTML report"
git push origin v0.3.0
gh run watch   # or: gh run list --workflow=release.yml
gh release view v0.3.0
```

### 7. Verify install + pkg.go.dev

```bash
GOPROXY=https://proxy.golang.org,direct go install github.com/lazarok09/ahp-method/cmd/ahp@v0.3.0
ahp version
# open https://pkg.go.dev/github.com/lazarok09/ahp-method
# (proxy may lag a few minutes after the first fetch)
```

### 8. Optional day-one

- Confirm Actions CI is green on `main`
- Enable Discussions or leave the issue templates as-is
- Announce with `examples/vendor-selection` + `output/report.html`

## Do not

- Commit `.env` (legacy Auth secrets were relocated under `~/.ahp-legacy-secrets/`)
- Force-push or rewrite history unless you explicitly choose a Go-only orphan root
- Change the module path again after the rename — keep `github.com/lazarok09/ahp-method`

## Reference

Full plan: [docs/OPEN_SOURCE.md](./docs/OPEN_SOURCE.md)
