---
id: G-0679
title: A branch's first push is never judged, and no workflow runs aiwf check
status: open
discovered_in: M-0331
---
## What's missing

`ResolveUntrailedRange` in `internal/cli/check/provenance.go` returns no range
when the branch has no upstream and no `--since` is passed. The provenance audit
is then skipped with a `provenance-untrailered-scope-undefined` warning, taking
the untrailered-entity audit and the dropped-body-section gate with it. A branch
started from a local ref is in that state at its first push — `git push -u` runs
the pre-push hook before it sets the upstream — while one started from a
remote-tracking ref already has one.

No workflow under `.github/workflows/` runs `aiwf check`. CI runs
`aiwf doctor --self-check` and nothing that judges a commit range.

## Why it matters

Content pushed on such a branch and merged on the server — a pull request merged
or squashed on GitHub — is compared by nothing. The push that carried it had no
range, and no later local push carries it. A branch merged locally is judged when
the branch receiving the merge is pushed against its upstream, which is what
keeps the maintainer flow covered; the contributor flow through pull requests is
not.
