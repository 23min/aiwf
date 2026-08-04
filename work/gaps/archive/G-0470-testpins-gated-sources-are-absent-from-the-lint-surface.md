---
id: G-0470
title: testpins-gated sources are absent from the lint surface
status: addressed
priority: low
addressed_by_commit:
    - 5e1d58d80
---
## What's missing

`.golangci.yml` declares `run.build-tags: [stress]`, so the `stress`-tagged scenario drivers under `internal/stresstest` and `cmd/stresstest` are linted. The `testpins` tag is not declared, so every file gated behind it is invisible to `golangci-lint` — in `make lint`, in the pre-push hook, and in the `lint` CI job alike.

The affected sources are `internal/workflows/spec/branch/branchtest/` (the Pin registry and its tests), `internal/cli/integration`'s `pin_testpins_test.go` / `bijection_*_testpins_test.go`, and `internal/policies/m0162_ac4_sabotage_testpins_test.go`.

Adding the tag reports two findings that have never been surfaced:

```
internal/workflows/spec/branch/branchtest/pin_test.go:30:8: ST1023: should omit type func(string, string) from declaration; it will be inferred from the right-hand side (staticcheck)
internal/workflows/spec/branch/branchtest/pin_test.go:37:8: ST1023: should omit type func() map[string][]string from declaration; it will be inferred from the right-hand side (staticcheck)
```

## Why it matters

A tag that no linter configuration names is a hole in the lint boundary, and the boundary is the point: G-0179 records what long-lived unlinted code costs, and the pre-push hook exists specifically so lint debt cannot accumulate invisibly until a push. Code behind `testpins` accumulates exactly that way today.

The scope is bounded and the exposure is low — the compile-level hole is already closed, because `.github/workflows/go.yml`'s `vet` job runs `go vet -tags testpins ./...` alongside `-tags stress`, so a type error behind either tag fails CI. What remains uncovered is the style and correctness surface `golangci-lint` adds over `go vet`: `staticcheck`, `errcheck`, `gocritic`, `revive`, `gosec`, and the rest of the enabled set.

## Resolution shape

Add `testpins` to `.golangci.yml`'s `run.build-tags`, then clear whatever the enabled linter set reports — the two findings above, plus anything in the integration and policy test files that has never been linted.

Note that `golangci-lint` lints the union of the declared tags in one pass, not each tag configuration separately. A file gated behind `//go:build !testpins` therefore stays unlinted once `testpins` is declared. `internal/cli/integration` carries two such files (`bijection_posthook_nontestpins_test.go`, `pin_nontestpins_test.go`), so the negated arm needs its own decision: accept it as out of scope, or add a second lint invocation for that configuration.

## Where to fix

- `.golangci.yml` — the `run.build-tags` list, and the comment there explaining why `testpins` was deliberately left out.
- `internal/workflows/spec/branch/branchtest/pin_test.go` — the two ST1023 findings.
- `internal/cli/integration/`, `internal/policies/` — the remaining `testpins`-gated files, once they are linted for the first time.
