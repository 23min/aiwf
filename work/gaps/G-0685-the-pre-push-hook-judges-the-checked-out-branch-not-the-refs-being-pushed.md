---
id: G-0685
title: The pre-push hook judges the checked-out branch, not the refs being pushed
status: open
discovered_in: M-0331
---
## What's missing

The pre-push hook `aiwf init` installs — `preHookScript` in
`internal/initrepo/initrepo.go` — ends in `exec "$AIWF" check` with no arguments
and reads nothing from stdin, where git hands a pre-push hook the refs being
pushed. `aiwf check` then resolves its commit range from the checked-out branch's
upstream, so the hook judges the branch that is checked out, not the branch being
pushed. Measured in a scratch repository with a bare remote and the hook installed:
`feature`, whose one commit past its upstream drops `## Why it matters` from a gap,
is refused when pushed from the `feature` checkout — `entity-body-section-dropped`
naming that commit — and pushed from the `main` checkout the same command prints
`ok — no findings` and the drop lands on the remote.

## Why it matters

Every range-scoped rule the hook carries — the provenance audit as much as the
body-section gate — is a refusal only for the branch an operator happens to have
checked out. `git push origin <other-branch>` from anywhere else is judged by
nothing, and the push that leaves the machine is the one boundary these rules
exist to hold.
