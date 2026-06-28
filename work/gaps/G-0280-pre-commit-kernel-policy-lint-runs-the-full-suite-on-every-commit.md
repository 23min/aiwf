---
id: G-0280
title: Pre-commit kernel-policy lint runs the full suite on every commit
status: open
prior_ids:
    - G-0279
discovered_in: M-0177
---
## Problem

The pre-commit hook's kernel-policy-lint step runs the full `go test ./internal/policies/` suite (~66–73s observed) on **every** commit, including planning-state commits that touch no Go code. `aiwf promote`, `edit-body`, `retitle`, and `render` only mutate entity markdown / `ROADMAP.md`, so the policy suite cannot have changed — re-running it is pure waste.

## Impact

- Every planning commit pays ~70s. For a `tdd: required` milestone this compounds badly: M-0177's phase dance was 15 frontmatter-only commits (`green→done→met` × 5 ACs + a retitle) — roughly 20 minutes of policy-lint runs for zero Go change.
- The latency caused a concrete failure during M-0177: an `aiwf promote --phase done` was SIGTERM'd by a 2-minute command timeout *mid-commit*. Go deferred cleanup (apply.go's rollback and the repo-lock release) does not run on signal-kill, so it left a staged-but-uncommitted frontmatter edit and a stale `.aiwf.lock` that needed manual cleanup before the sequence could resume.

## Proposed fix

Gate the policy-lint step on staged Go/build inputs: skip `go test ./internal/policies/` when `git diff --cached --name-only` contains no `*.go` / `go.mod` / `go.sum` / `Makefile` / `.github/workflows/*` path. This mirrors the CLAUDE.md "skip the redundant `make ci` run when no Go/build input changed" cadence rule, applied at the pre-commit chokepoint. Planning-only commits become near-instant; Go-touching commits keep the full gate.

## Notes

- The hook is the authoritative chokepoint, so the gate must be conservative — when any Go-shaped path is staged, run the suite.
- Secondary hardening worth considering: a signal handler (or `aiwf`-side lock with a staleness check) so a SIGTERM/SIGKILL mid-commit doesn't strand a `.aiwf.lock` + partial staged write. The lock-staleness recovery is the more general fix; the hook-gating above removes the latency that makes the timeout likely in the first place.
- Surfaced during M-0177 (E-0044). Distinct subsystem from the area feature; filed as its own gap rather than folded into the milestone.
