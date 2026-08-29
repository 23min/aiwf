---
id: M-0325
title: Add an operator-driven end to aiwf authorize
status: in_progress
parent: E-0090
depends_on:
    - M-0323
tdd: required
acs:
    - id: AC-1
      title: The end mode ends the targeted scope without changing its entity's status
      status: met
      tdd_phase: done
    - id: AC-2
      title: Re-ending converges to a NoOp; naming no resolvable scope is refused
      status: met
      tdd_phase: done
    - id: AC-3
      title: The end mode is reachable from --help, tab-completion, and the authorize skill
      status: met
      tdd_phase: done
    - id: AC-4
      title: The automatic end covers paused scopes, not only active ones
      status: met
      tdd_phase: done
---
## Goal

Give a human a way to end an authorization scope deliberately, without changing the status of the entity it was opened on.

## Context

`AuthorizeMode` is a closed three-value set — open, pause, resume. A scope's only exit today is the terminal promote or cancel of its own entity, which stamps `aiwf-scope-ends` as a side effect of the status change. So withdrawing a delegation requires closing the work: there is no way to record that a human ended a delegation while the entity it was opened on keeps living.

G-0022 reserved an `aiwf-revoked-by:` trailer slot for a revoke verb that was never built. This milestone builds that capability, though not through that slot: the end writes `aiwf-scope-ends:`, which the replay already reads. The automatic end fires exactly where it already fires; the one behaviour that changes is which scopes it covers, so a terminal promote or cancel of an entity carrying a paused scope now ends that scope rather than stranding it — which is AC-4.

ADR-0047 settles the semantics. An end names its scope by authorize-commit SHA and defaults to the entity's sole candidate; it takes a required reason, as pause and resume do; ending covers scopes in `active` or `paused` state; and nothing undoes an end.

## Acceptance criteria

### AC-1 — The end mode ends the targeted scope without changing its entity's status

After the verb runs, the targeted scope's replayed state is `ended` and the entity's status is what it was before. The two assertions together are the point: an end that only worked by moving the entity to a terminal status would be the behavior this milestone exists to replace.

"Targeted" is what ADR-0047 defines: the scope named by `--scope <auth-sha>`, or the entity's sole non-ended scope when `--scope` is absent. More than one candidate and no `--scope` is a refusal listing them, which AC-2 covers.

### AC-2 — Re-ending converges to a NoOp; naming no resolvable scope is refused

Two behaviors, one rule ordering. Re-running the verb against a scope already `ended` returns exit 0, writes no commit, and reports the state as already holding. Naming a scope that does not resolve — a bad sha, an entity with no scope ever opened — is refused, not converged.

That ordering is the kernel's R1-before-R2: an argument must name something real before the verb asks whether the request is already satisfied. Converging on an unresolvable target would assert success for state that cannot exist.

### AC-3 — The end mode is reachable from --help, tab-completion, and the authorize skill

The flag appears in `aiwf authorize --help`, it is tab-completable, and the `aiwf-authorize` skill documents it. The completion half is already policed by the drift test that fails CI on a flag added without completion wiring.

### AC-4 — The automatic end covers paused scopes, not only active ones

An entity carrying one paused scope reaches a terminal status; afterwards that
scope's replayed state is `ended`. Today it stays `paused` forever, because
`loadActiveScopeAuthSHAsForEntity` collects only scopes in `active` state and
nothing else ever emits their `aiwf-scope-ends:` trailer.

The assertion is over the replayed state rather than the emitted trailer, so it
holds whichever way the predicate is written. `paused → ended` is already legal
in the scope FSM, so this closes an exit the FSM permits and no code fires.

## Constraints

- The automatic scope-end fires where it already fires; only the predicate choosing which scopes it covers changes. The one invocation that behaves differently is a terminal promote or cancel of an entity carrying a paused scope, which AC-4 requires.
- One mutation, one commit, or none — a converging re-run writes nothing and carries no `commit_sha`.
- The mode is a peer of `--to` / `--pause` / `--resume`, which are mutually exclusive; the new one joins that exclusion rather than combining with them. `--scope` modifies the end mode and is not itself a mode.
- What undoes an end is answered in ADR-0047, not invented here.

## Design notes

Scope state is a projection over git trailers, not stored state — `internal/scope` replays commits forward from the authorize commit. So ending a scope means writing a trailer that the replay reads, in the same shape the automatic end already writes, rather than mutating a record. `ReplayScopes` therefore needs no change: it already resolves `aiwf-scope-ends: <auth-sha>` by SHA.

An operator end stays distinguishable from an automatic one in history without a new trailer, because it rides an `aiwf-verb: authorize` commit rather than a `promote` or `cancel`.

The paused-scope fix is one predicate in `loadActiveScopeAuthSHAsForEntity` (`internal/cli/cliutil/provenance.go`), which today collects only scopes in `active` state.

The CHANGELOG entry belongs to this milestone: the surface is new and consumer-visible.

## Out of scope

- Removing or re-timing the automatic end. Only its predicate changes.
- Same-state convergence for a duplicate `authorize --to` re-grant (G-0460). It shares the targeting question and is a separate defect.
- Time-bound scopes, verb-set restrictions, and the rest of G-0022's extension list.

## Dependencies

- M-0323 — produced ADR-0047, which settles which scope the mode targets, what "ending" covers, and what undoes an end.

## Work log

### AC-1 — The end mode

`--end` lands as a fourth `AuthorizeMode`, with `--scope <auth-sha>` as its modifier · commit `9dd1a2d7e` · tests in `internal/verb/authorize_end_test.go` and `internal/cli/integration/authorize_end_test.go`.

The mode resolves its target, then emits `aiwf-scope-ends: <full-sha>` on an otherwise ordinary authorize commit. `ReplayScopes` needed no change, as the spec predicted.

### AC-2 — Target resolution and convergence

Resolution searches every scope, ended ones included, so naming an ended scope converges while naming nothing refuses · commit `51254f8dc` · tests in `internal/verb/authorize_end_resolution_test.go` and `internal/cli/integration/authorize_end_resolution_test.go`.

No production code: the arms this AC describes shipped inside AC-1's commit, because `authorizeEnd` cannot emit a trailer without first deciding which scope. The phase ladder records that — see *Decisions made during implementation*.

### AC-3 — Discoverability

`--end` and `--scope` reach `aiwf authorize --help`, the root banner, tab-completion, and the skill · commit `70d64e768` · tests in `internal/cli/integration/authorize_skill_surface_test.go` and `internal/cli/authorize/authorize_scope_completion_test.go`, plus the two pre-existing drift tests this turned green.

### AC-4 — The automatic end covers paused scopes

Predicate widened from `== StateActive` to `!= StateEnded`; the function is renamed to state the rule it now holds · commit `00012965f` · tests in `internal/cli/cliutil/provenance_scope_ends_test.go` and `internal/cli/integration/authorize_auto_end_paused_test.go`.

### Supporting commits

`b0476f25a` records the mode in `docs/design/provenance-model.md` (normative, and it described a three-mode verb) and in `CHANGELOG.md`. `33d4e0db2` backfills three `Run`-level tests the diff-scoped coverage gate named.

## Decisions made during implementation

**AC-2's TDD phase was forced to `done`, with the reason on the commit.** Its tests pass the moment they are written, because the behaviour they describe shipped in AC-1. The AC decomposition is a specification split — two distinct claims — not an implementation split, and no promote sequence makes "a failing test preceded the code" true here. The alternative considered was reverting AC-2's arms to manufacture a red; it was rejected because AC-1's commit already contains that code, so the ladder and `git log` would then disagree about when it was written, and the record would look routine while being no more true. `aiwf history M-0325/AC-2` carries the forced transition and its reason.

## Validation

Run on the milestone branch at `33d4e0db2`, in the devcontainer (linux/amd64, go 1.25):

- `AIWF_COVERAGE_BASE=epic/E-0090-… make ci` — green. Covers `go vet`, the full `golangci-lint` set, `go test -race` with coverage, the diff-scoped coverage gate, the firing-fixture meta-gate, and `aiwf doctor --self-check` (29 steps).
- `aiwf check` — 0 errors, 4 warnings: the unpushed-branch provenance skip, the G-0646 archive sweep pair, and `epic-active-no-drafted-milestones`. The last is new and expected — M-0325 was the epic's final `draft` milestone, so the warning is E-0090 reporting it is ready to wrap.

An earlier `make ci` on the same branch failed `TestPolicy_BranchCoverageAudit` on three lines of `internal/cli/authorize/authorize.go`; `33d4e0db2` is the fix.

## Deferrals

- **G-0651** (filed on `main`) — a backtick-quoted span in a flag's usage string is read by cobra as that flag's value-placeholder name, so `--help` prints the phrase where the value's type belongs. Five shipped flags are affected. Found while verifying this milestone's own `--help` output, where the same mistake had just been made and fixed. Deferred rather than fixed here because the uniform repair needs a convention decision — strip the backticks, or name a proper placeholder — across three CLI packages this milestone does not otherwise touch.

## Reviewer notes

**The `--branch` flag was undocumented in the `aiwf-authorize` skill.** The AC-3 relationship check found it, not a reading of the skill: the check derives its expectation from the Cobra tree, so it names every flag the skill omits rather than the one flag someone thought to look for. Fixed in `70d64e768` under the cheap-fix rule — same file, same test.

**A reverse-direction check was written and cut.** It asserted that every `--flag` the skill mentions still resolves against the Cobra tree. It cannot distinguish a flag the skill tells an assistant to pass to `authorize` from one it mentions while explaining another verb, so it fired on `--principal` in the tool-mode example and on `--allow-force`, a deliberately-documented future flag. Making it precise would mean parsing which command each mention belongs to, which pins a reading — the fragility D-0070 retires. The forward direction is also the only one AC-3 claims.

**Two unreachable branches carry `//coverage:ignore` rather than a test.** `authorizeEnd`'s `CheckTrailerCoherence` error return: every coherence rule keys on a trailer the end mode never emits, and `Authorize` has already required a `human/` actor, so no input reaches it. The `LoadEntityScopes` error returns in `loadEndableScopeAuthSHAsForEntity` and `completeScopeFlag`: the loader short-circuits to `(nil, nil)` when `HasCommits` is false, so a repo-less or empty root never errors, and an error means `git log` failed after `HasCommits` succeeded on the same root. Both calls stay in place so the checks still run if their inputs grow.

**`internal/verb` now imports `internal/entityview`** for `ShortHash`. A downward edge (tier 2 → tier 4) the layering policy permits. The alternative was a third local copy of a seven-character truncation, next to the two that exist; routing through the call `aiwf show` renders with also makes the "same abbreviation the operator read" claim true by construction rather than by two literals agreeing.

**A `--help` rendering defect was introduced and fixed inside AC-3.** Only running the binary caught it. Worth repeating at the next render-touching milestone: the surface a test asserts on and the surface an operator reads are not the same artefact.

**The manual branch-coverage audit was wrong again, in the same way as M-0324.** Three CLI lines were verified through the integration tests and cleared; those drive the binary as a subprocess and contribute nothing to the profile, so only `make ci` caught them. The pattern across two milestones: every claim a command could check was right first time, and the ones resting on reading were not.
