---
id: G-0107
title: reorganize cmd/aiwf into idiomatic per-verb packages
status: open
---
## What's missing

`cmd/aiwf/` holds ~103 files in a single flat `package main`: ~31 source files (verb implementations + cross-cutting helpers) and ~72 test files. Three files exceed 25 KB — [admin_cmd.go](../../../cmd/aiwf/admin_cmd.go) at 54 KB containing ~12 sub-verbs, [main.go](../../../cmd/aiwf/main.go) at 27 KB, [status_cmd.go](../../../cmd/aiwf/status_cmd.go) at 26 KB. Cross-cutting helpers ([actor.go](../../../cmd/aiwf/actor.go), [flags.go](../../../cmd/aiwf/flags.go), [lock.go](../../../cmd/aiwf/lock.go), [treeload.go](../../../cmd/aiwf/treeload.go), [provenance.go](../../../cmd/aiwf/provenance.go), [scopes.go](../../../cmd/aiwf/scopes.go), [platform.go](../../../cmd/aiwf/platform.go)) sit alongside verb files with no naming or directory distinction. There is no per-verb encapsulation: every helper, every verb constructor, every test lives in the same package namespace.

## Why it matters

The shape is what `cobra-cli init` gives you and works fine for ~10-verb CLIs; aiwf has ~30 top-level verbs plus sub-commands, well past the size where the Go community splits things up. Idiomatic Cobra CLIs at this scale — kubectl, gh, hugo, terraform — give each verb its own sub-package and keep `main.go` ~30 lines. The current shape costs us: (a) contributors (human or LLM) cannot orient by directory listing — finding "where does the `promote` verb live?" requires grepping over a flat 103-file directory; (b) `admin_cmd.go` is a 54 KB mixed-concern monolith that no test or reviewer can reason about as a unit; (c) shared helpers and verbs are indistinguishable, so the boundary between "framework toolbelt" and "verb implementation" never gets enforced by the compiler; (d) the contrast with [internal/](../../../internal/) — already cleanly factored into 24 domain packages (entity, tree, gitops, check, render, scope, contractverify, …) — makes `cmd/aiwf/` the structural outlier in an otherwise well-organized module.

## Target shape (sketch, not prescription)

```
cmd/aiwf/main.go                  # ~30 lines: parse args, call cli.Execute()
internal/cli/
  root.go                         # newRootCmd, version stamping, exit codes
  cliutil/                        # shared helpers, today loose in cmd/aiwf/
    actor.go  flags.go  lock.go  treeload.go  provenance.go  scopes.go
  <verb>/                         # one package per top-level verb
    <verb>.go  <verb>_test.go
  admin/                          # the 54 KB monolith, split per sub-verb
    admin.go  canonicalize.go  auditonly.go  rituals.go  selfcheck.go  ...
  integration/                    # cross-verb tests
    binary_integration_test.go  completion_drift_test.go  ...
```

Each verb package exports `func New(deps Deps) *cobra.Command`; tests that currently poke at unexported `package main` helpers either move with the helper or shift to same-package `_test.go` in the new sub-package.

## Suggested sequencing (independent commits, in leverage order)

1. **Split [admin_cmd.go](../../../cmd/aiwf/admin_cmd.go)** into one-file-per-sub-verb under `cmd/aiwf/admin/`, still `package main`. Biggest readability win; no API change; no test churn.
2. **Move pure helpers** ([actor.go](../../../cmd/aiwf/actor.go), [flags.go](../../../cmd/aiwf/flags.go), [lock.go](../../../cmd/aiwf/lock.go), [treeload.go](../../../cmd/aiwf/treeload.go), [provenance.go](../../../cmd/aiwf/provenance.go), [scopes.go](../../../cmd/aiwf/scopes.go), [platform.go](../../../cmd/aiwf/platform.go)) into `internal/cli/cliutil/`. Mechanical; exports the helper API; some tests that referenced unexported names move with their helper.
3. **Extract verbs one at a time** into `internal/cli/<verb>/` as the verbs are next touched. Lazy migration; no big-bang.

Step 2 is probably the highest ROI: it pulls the largest visual clutter out of `cmd/aiwf/` without forcing every test file to move.

## Code references

- [cmd/aiwf/](../../../cmd/aiwf/) — the flat 103-file directory under discussion
- [cmd/aiwf/main.go](../../../cmd/aiwf/main.go) — `newRootCmd()` registers ~30 verbs via `AddCommand`
- [cmd/aiwf/admin_cmd.go](../../../cmd/aiwf/admin_cmd.go) — 54 KB, ~12 sub-verbs
- [internal/](../../../internal/) — the reference for how factored-by-domain looks in this repo (24 packages)
