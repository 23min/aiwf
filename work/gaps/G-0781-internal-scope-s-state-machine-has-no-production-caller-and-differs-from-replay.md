---
id: G-0781
title: internal/scope's state machine has no production caller and differs from replay
status: addressed
discovered_in: M-0364
addressed_by_commit:
    - c615eef3d11a3f4f3e3ff7b21d644b22ad8165ae
---
## What's missing

The scope state machine in `internal/scope` has no production caller. `LoadScope`
(`internal/scope/scope.go:113`) and `IsLegalScopeTransition` (`scope.go:85`), which `LoadScope`
calls at `scope.go:155`, are reached only from tests (the hit in
`integration_scenarios_test.go` is a comment); scopes are replayed in production by `ReplayScopes`
(`internal/cli/cliutil/scopes.go:62`). The two differ on a pause or resume with no scope in the
state it leaves: `LoadScope` refuses the history ("scope %s: illegal transition %s → %s at
commit %s", `scope.go:156`), and `ReplayScopes` skips the commit.

```
$ grep -rn "LoadScope\b\|IsLegalScopeTransition" --include=*.go . | cut -d: -f1 | sort | uniq -c
      1 ./internal/cli/integration/integration_scenarios_test.go
      7 ./internal/scope/scope.go
     20 ./internal/scope/scope_test.go
```

Measured 2026-11-21 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a tree whose code is `main` at 0c00b37c4 (`aiwf version`: `v0.42.1-0.20261121183124-0c00b37c47b3+dirty`), in a disposable repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs, and `milestone/M-0001-first-milestone` checked out. Ids in the block name entities in that repository. A hand-made commit with `aiwf-verb: authorize`, `aiwf-entity: M-0001` and
`aiwf-scope: resumed` was added while the scope was active.

```
$ aiwf show M-0001
    2a59b18  human/t → ai/claude  state: active   opened 2026-11-21  events: 1
$ aiwf check                      # exit 0, no provenance finding
```

## Why it matters

The scope's legal transitions are tested on code that nothing runs, so those tests pass whatever
the replay in use does: a pause or resume that the tested state machine refuses as illegal is
skipped by the replay aiwf uses, and the tests cannot tell.
