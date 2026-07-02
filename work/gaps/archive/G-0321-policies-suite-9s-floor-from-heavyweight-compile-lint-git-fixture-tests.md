---
id: G-0321
title: policies suite ~9s floor from heavyweight compile/lint/git fixture tests
status: wontfix
discovered_in: M-0196
---
## What's missing

After G-0320 fixture-pinned `TestM080_AC6` (removing the 82s live-tree
`aiwf check` shell-out), the `internal/policies` suite floor is ~9s wall.
That floor is no longer one outlier — it is the aggregate of ~150 tests
running at `-parallel 8`, with wall-clock gated by a handful of
heavyweight tests:

- `TestM0162_AC2_BuildTagExclusion` (~4.7s) — `go build ./cmd/aiwf` +
  `go tool nm` to prove a build-tag symbol is absent from the production
  binary. The compile *is* the assertion.
- `TestM0147_AC3_GlobalRuleExercised` (~4.1s) — walks the impl source
  tree to collect finding codes.
- `TestM0146_ScopeReachMachinery` (~4.0s) — git-fixture scope-reachability
  fan-out.
- `TestGolangciConfigRulesFire` (~2.6s) — runs real `golangci-lint`
  against fixtures to prove config rules fire. Running the linter *is*
  the assertion.
- ~30 `promote`/`authorize`/`cancel` subtests (~1.5–2.2s each) — each
  builds a git repo and runs real verbs (one commit per mutation).

## Why it matters

~9s is paid on every Go-touching commit (the `pre-commit.local` G-0280
gate runs `go test ./internal/policies/...`) and in the CI policy-test
job. It is a smaller tax than the pre-G-0320 84s, but still felt in the
inner loop.

## Why this is hard / may not be worth it

Unlike G-0320's single pathological outlier, the residual cost is paid
for the thing each test actually verifies: a real `go build` (build-tag
exclusion), a real `golangci-lint` run (config-rule firing), real git
history (verb/scope fixtures). These are not waste — they are
integration assertions that lose their value if faked. The next tier is
~4s, not 82s, so the leverage is an order of magnitude lower and the
correctness risk per rewrite is higher.

## Proposed investigation (if pursued)

1. Profile the git-fixture verb subtests — is there a shared
   `sync.Once` repo fixture they could read-only reuse instead of each
   building its own git repo? (Mirrors `sharedRepoTree`.) The
   per-mutation commits make full sharing hard, but a shared *base*
   repo cloned per-test may beat building from scratch.
2. `TestM0147_AC3` walks the source tree — confirm it isn't re-parsing
   on every call; cache the parse behind a `sync.Once` if shared.
3. Leave `BuildTagExclusion` and `GolangciConfigRulesFire` alone — the
   compile / lint run is irreducible and is the assertion.

Measure before building. The realistic floor after (1)+(2) is probably
~5s; whether the delta is worth the rewrite risk is the open question.

## Discovered in

G-0320 — measured the residual `internal/policies` suite floor (~9s)
after the M-080 AC-6 fixture fix landed; the operator asked what the
remaining time was spent on.
