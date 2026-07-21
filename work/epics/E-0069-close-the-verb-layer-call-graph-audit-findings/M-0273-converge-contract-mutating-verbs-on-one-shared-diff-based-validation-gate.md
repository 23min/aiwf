---
id: M-0273
title: Converge contract-mutating verbs on one shared diff-based validation gate
status: in_progress
parent: E-0069
tdd: required
acs:
    - id: AC-1
      title: a shared gate reports only findings introduced by the projected mutation
      status: met
      tdd_phase: done
    - id: AC-2
      title: bind, unbind, recipe install, and recipe remove route through the shared gate
      status: met
      tdd_phase: done
---
## Goal

Give the contract-mutating verbs one shared validation gate: findings
introduced by a mutation are computed as a before/after diff of contract-check
findings on the projected config, and all four verbs route through it.

## Context

The audit found three unrelated gate styles across bind, recipe install, and
recipe remove (finding F10 of `docs/initiatives/verb-layer-cleanup.md`), with
unbind ungated. The convergence decision (see References) chose a diff-based
gate because id-filtered scoping cannot generalize to verbs that mutate the
validators map. Bind's current filter is not a true before/after diff; this
milestone makes the diff the shared semantics.

## Acceptance criteria

### AC-1 — a shared gate reports only findings introduced by the projected mutation

A new `internal/verb/contractgate.go` holds `contractMutationGate(t
*tree.Tree, current, next *aiwfyaml.Contracts, repoRoot string)
[]check.Finding`: it runs `contractcheck.Run` once against `current`
and once against `next`, then returns the findings present in the
`next` run that are not already present in the `current` run — a true
multiset before/after diff, not an id-filtered subset. A finding
present in both runs (a pre-existing issue on an entry the mutation
didn't touch) is excluded; a finding introduced by the mutation, on
any entry, is returned; a finding the mutation *resolves* (present
before, absent after) is not reported — the gate only reports
additions.

Evidence: `internal/verb/contractgate_test.go` exercises the diff
directly — a mutation that changes nothing produces zero introduced
findings even when `current` already carries pre-existing findings; a
mutation that adds an entry with a missing schema/fixtures path
surfaces exactly those two findings; a pre-existing finding on an
untouched entry is excluded from the introduced set even when the
mutation is otherwise "dirty" elsewhere.

### AC-2 — bind, unbind, recipe install, and recipe remove route through the shared gate

`ContractBind`, `ContractUnbind`, `RecipeInstall`, and `RecipeRemove`
(`internal/verb/contractbind.go`, `contractrecipe.go`) all call
`contractMutationGate` before writing, in place of bind's existing
id-filtered `contractCheckForBinding` (removed) and unbind/recipe-
install/recipe-remove's previous lack of any contract-check gate.
Each verb function gains the `t *tree.Tree` and `repoRoot string`
parameters needed to run the check; the CLI dispatchers that didn't
already load a tree (`internal/cli/contract/unbind.go`, `recipes.go`'s
install and remove paths) now do, mirroring `bind.go`'s existing
`tree.Load` call. Recipe remove keeps its `bindingsReferencing`
referential-integrity error ahead of the gate call, per the
milestone's constraint — the gate is an additional safety net there,
not a replacement for that specific error message.

Evidence: the migrated `internal/verb/contractbind_test.go` /
`contractrecipe_test.go` suites (updated call sites, still green) plus
new coverage proving the previously-ungated verbs' safety net is live
— a case constructed so unbind's or recipe-remove's projected mutation
would introduce a contract-config finding, confirming the gate
actually fires rather than just being wired in dead. `go vet`/`go
build` clean repo-wide; the CLI-level `internal/cli/integration`
contract-verb tests (`single_commit_invariant_test.go`,
`trailer_shape_test.go`, `remaining_verbs_diag_test.go`,
`verb_metadata_test.go`) pass unmodified, confirming existing verb
envelopes and exit codes are unchanged for normal operation.

## Constraints

- Test-first per AC (`tdd: required`).
- Remove keeps its precise "referenced by bindings: <ids>" error on top of the
  shared gate — the gate replaces gate *logic*, not better error messages.
- Existing verb envelopes and exit codes unchanged; pre-existing findings on
  untouched entries never block a mutation (the diff guarantees this by
  construction).

## Design notes

- Gate shape: run the contract check on current and projected configs, report
  only findings present in the projection and absent from current.
- The converged-gate decision entity in References carries the full rationale.

## Out of scope

- `contract verify`'s external-validator pipeline (deliberately separate).
- Any change to what the underlying contract check validates.

## Dependencies

- None — parallel-safe with the sibling E-0069 milestones.

## References

- `docs/initiatives/verb-layer-cleanup.md` §F10; D-0041, the convergence decision
  entity; `internal/verb/contractbind.go`, `internal/verb/contractrecipe.go`.

---

## Work log

### AC-1 — a shared gate reports only findings introduced by the projected mutation

`internal/verb/contractgate.go` created: `contractMutationGate` plus
the pure `diffIntroducedFindings` helper (split out during a `wf-
vacuity` audit) · commit 94b5515f · tests
`internal/verb/contractgate_test.go`, all green.

### AC-2 — bind, unbind, recipe install, and recipe remove route through the shared gate

`ContractBind`, `ContractUnbind`, `RecipeInstall`, `RecipeRemove`, and
the atomic add+bind path (`internal/verb/add.go`) wired onto the
shared gate; the three CLI dispatchers lacking a tree load
(`internal/cli/contract/unbind.go`, `recipes.go`) now have one. Landed
alongside a correctness fix to AC-1's gate itself (D-0046) and a set
of `wf-vacuity`-driven tests closing gaps the wiring's own review
surfaced · commit da14e458 · tests `internal/verb/contractbind_test.go`,
`contractrecipe_test.go`, `contractgate_test.go`, all green; full
`internal/cli/integration` contract-verb suite unmodified and green.

## Decisions made during implementation

- D-0046 — the shared gate diffs findings by identity
  (Code/Severity/EntityID/Subcode/Path), not full-struct equality,
  because `contractcheck.Run`'s `Message` embeds a positional index
  that shifts on entry insert/remove.

## Validation

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `gofmt -l internal/verb/*.go internal/cli/contract/*.go` — clean.
- `go test -race -parallel 8 ./...` — all packages green.
- `make lint` (golangci-lint, worktree-scoped cache) — 0 issues.
- `make coverage-gate` — M-0273's own diff (the milestone branch's
  `7ae0713e..HEAD` range) produces zero branch-coverage findings; the
  gate's overall failure is entirely the pre-existing M-0272
  epic-scoping artifact (see Reviewer notes).
- Manual branch-coverage walk of every new/changed conditional in
  `contractgate.go`, `contractbind.go`, `contractrecipe.go`,
  `unbind.go`, `recipes.go` — every reachable branch has an explicit
  test; the three branches that are unreachable given
  `contractcheck.Run`'s current rules (`ContractUnbind`'s,
  `RecipeInstall`'s, and `RecipeRemove`'s gate-block lines) carry
  `//coverage:ignore` with rationale, independently re-verified by
  both reviewers by reading `contractcheck.Run` directly rather than
  trusting the comments.
- `wf-vacuity` mutation probes on both ACs: AC-1's probes found and
  closed two real gaps (multiset-multiplicity logic untestable through
  `contractcheck.Run`'s real output, closed by extracting
  `diffIntroducedFindings` as a directly-testable pure function; the
  original struct-identity diff itself, closed by D-0046). AC-2's
  probes found and closed two more (`EntityID` wasn't load-bearing in
  any existing test, closed by an adversarial-ordering test; the three
  previously-ungated verbs' wiring was unobservable through normal
  behavior, closed by three nil-tree panic probes proving each verb
  actually invokes the gate).

## Deferrals

- (none)

## Reviewer notes

- Independent two-lens review (fresh-context subagents, code-quality
  and design-quality) both returned **approve, no blocking findings**.
- Code-quality lens: independently re-derived (not trusted from
  comments) that the three `//coverage:ignore`d gate-block branches
  are genuinely unreachable given `contractcheck.Run`'s current rules,
  and that `findingIdentity`'s field choice can't conflate two
  genuinely different problems on the same entity. Confirmed
  `RecipeRemove`'s referential-integrity error still runs ahead of the
  gate, unchanged, and that the CLI-integration test suite passed
  unmodified. Surfaced one epic-level heads-up: `make coverage-gate`
  fails overall, but the 11 flagged lines are all sibling M-0272 work
  (`internal/cli/history/`, `internal/entityview/`) in the range
  before M-0273's own fork point — the same known epic-scoping
  artifact M-0272's own Reviewer notes documented (the local coverage
  gate compares against `origin/main`, which predates the whole
  epic). Not introduced by, or in scope for, M-0273; the epic wrap
  will need to clear it before the E-0069→main merge gate passes.
  Also noted a minor, non-blocking behavior narrowing: a `bind
  --force` that repoints an *already-broken* binding to a
  differently-broken path no longer re-blocks, since that entry's
  `missing-schema`/`missing-fixtures` identity already existed in
  `current` — a direct, correct consequence of "only what this
  mutation introduces" (arguably more correct than the old id-filter,
  which over-blocked on any bound-entity error regardless of whether
  bind's own values were being changed), but a subtle narrowing of
  the "envelopes unchanged" constraint worth being aware of. No test
  pinned the old over-blocking behavior, so nothing broke.
- Design-quality lens: approved the gate's shape, the pure-function
  split, `findingIdentity`'s locality to `internal/verb` (deliberately
  not hoisted next to `check.Finding` — "what makes two findings the
  same" is a property of the specific finding *producer*, not of the
  `check.Finding` type itself), and D-0046's identity choice as all
  correct. One documented, non-blocking judgment call: the reviewer
  would have wired only the two load-bearing sites (`ContractBind`,
  the atomic add+bind path) and left `RecipeInstall`/`RecipeRemove`
  gate-free with an inline "exempt until this verb can mutate
  entries[]" comment, rather than threading `t`/`repoRoot` through all
  three previously-ungated verbs for a branch that can't fire today.
  Called this a genuine coin-flip against D-0041's uniform-convergence
  reasoning, not a defect — the milestone's chosen shape (uniform
  wiring as a safety net against a future `contractcheck.Run` change,
  so no verb's gate needs retrofitting later) is defensible and
  already reasoned through in D-0041. Also noted `ContractUnbind`'s
  gate computes a real `no-binding` warning that the caller currently
  discards (only `check.HasErrors` gates the write) — intentional,
  not a defect, but worth naming since the gate's own doc comment
  ("reports only additions") reads as more consumed than it is at
  this one call site.
