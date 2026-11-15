---
id: G-0679
title: A branch started from a local ref is never judged at its first push
status: open
priority: medium
discovered_in: M-0331
---
## What's missing

`ResolveUntrailedRange` in `internal/cli/check/provenance.go` returns no range
when the branch has no upstream and no `--since` is passed. `aiwf check` then
skips its range-scoped checks with one `provenance-untrailered-scope-undefined`
warning: the untrailered-entity audit, the dropped-body-section gate and
`trailer-verb-unknown`, which `RunProvenanceCheck` runs after the range
resolves, and the instruction-file fence, which resolves its range through the
same function (`internal/cli/check/instruction_fence.go`). The warning's text
says "provenance audit" for the first and does not name `trailer-verb-unknown`:
`no upstream configured and no --since <ref>; provenance audit, body-section
gate and instruction-file fence skipped`. The standing provenance rules, which
walk every commit reachable from HEAD, still run. A branch started from a local
ref is in that state at its first push — `git push -u` runs the pre-push hook
before it sets the upstream — while one started from a remote-tracking ref
already has one.

No workflow under `.github/workflows/` runs `aiwf check`, and no CI job runs
the untrailered-entity audit, the dropped-body-section gate or
`trailer-verb-unknown` over this repository's commits. The instruction-file
fence does run in CI, as a Go policy test compiled from this tree: the
profile-driven gate step in `.github/workflows/go.yml` runs it over the commits
since its base ref, on a pull request, or a push to `main` or a `poc/**`
branch, that touches a path the workflow's filter lists.

## Why it matters

Content pushed on such a branch and merged on the server — a pull request merged
or squashed on GitHub — is compared by none of the checks the skip takes,
except the instruction-file fence where this repository's CI runs it. The push
that carried it had no range, and no later local push carries it.

A branch merged locally is judged only in part when the branch receiving the
merge is pushed against its upstream: the dropped-body-section gate and the
instruction-file fence see the merged commits, and a `--no-ff` merge hides them
from the untrailered-entity audit and `trailer-verb-unknown` (G-0765).

Measured at `3c5bd7546` (git 2.51.1, Linux) in a throwaway repository whose
`main` tracks a bare remote. A branch carries an untrailered hand edit to a gap,
a commit with `aiwf-verb: implement`, a commit dropping `## Why it matters`, and
a commit changing `CLAUDE.md` together with an unrelated file. Merged into `main`
with `git merge --no-ff`, it draws `entity-body-section-dropped`,
`instruction-commit-entity-missing` and `instruction-commit-unrelated-file` from
`aiwf check`. Fast-forwarded instead, it also draws
`provenance-untrailered-entity-commit` and `trailer-verb-unknown`.
