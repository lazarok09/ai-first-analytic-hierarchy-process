# Security Policy

## Supported versions

Security fixes are applied to the latest released `v0.x` tag on `main`.

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

Use one of:

1. **GitHub private vulnerability advisory** on this repository (Security → Advisories → New draft advisory), or
2. Email **lazarok09@gmail.com** with a short description, impact, and steps to reproduce.

You should hear back within a few business days. We will coordinate a fix and public disclosure once a patch is ready.

## Scope notes

`ahp` is a local CLI: it reads/writes workspace files on disk and does not run a hosted multi-tenant service. Reports about unsafe handling of untrusted workspace CSV/TOML, path traversal, or supply-chain issues in release artifacts are in scope.
