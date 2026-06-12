---
id: G-0179
title: Enforce full local CI gate (golangci-lint) at wrap on unpushed branches
status: addressed
discovered_in: M-0151
addressed_by_commit:
    - 3f5f075b
---
## What

Milestones implemented on a long-lived **epic integration branch** can accumulate latent `golangci-lint` failures that no gate catches until the branch is finally pushed: CI only runs on push, and the per-milestone self-review discipline (as practiced through E-0038's M-0148→M-0151) validated with `go vet` + `go test`, **not** `golangci-lint run`.

## Evidence

During M-0151's "are you 100% confident?" self-audit, `golangci-lint run ./...` surfaced **9 CI-blocking findings** on the epic branch — 5 `govet shadow` (the repo enables `govet: enable-all`, which `go vet` alone does not) and 4 `gocritic stringXbytes` — introduced as far back as M-0149 and invisible through three milestone wraps. A clean-cache run confirmed `main` was clean, so the debt was entirely epic-branch-local and would have failed the CI `lint` job the moment the epic was pushed/merged.

The CLAUDE.md "How to validate changes" section already lists `golangci-lint run` as a required check; the gap is that nothing **mechanically** enforces it per-milestone on an unpushed branch, so it depended on operator memory and was skipped.

## Desired resolution

A mechanical local full-gate check run at (or before) every milestone/epic wrap on branches that have not been pushed — minimally `golangci-lint run ./...`, ideally the full set CI runs (`go vet ./...`, `go test -race ./...`, `aiwf doctor --self-check`, `govulncheck`). Candidate shapes: a `make wrap-check` target the wrap ritual invokes, a pre-wrap step in `aiwfx-wrap-milestone`/`aiwfx-wrap-epic`, or a pre-merge-to-main hook. The chokepoint must not depend on the operator remembering to run the linter.

## Notes

- Surfaced and remediated within M-0151 (the 9 findings are fixed on the M-0151 branch); this gap captures the *systemic* fix so it does not recur on the next long-lived epic branch.
- Related discipline already documented: CLAUDE.md § "Worktree binary discipline" (stale-PATH binary) is the analogous "validate against the right thing" hazard for a different surface.
