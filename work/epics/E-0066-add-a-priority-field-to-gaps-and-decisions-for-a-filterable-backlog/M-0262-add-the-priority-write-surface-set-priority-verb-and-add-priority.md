---
id: M-0262
title: 'Add the priority write surface: set-priority verb and add --priority'
status: in_progress
parent: E-0066
depends_on:
    - M-0261
tdd: required
acs:
    - id: AC-1
      title: aiwf set-priority sets a gap/decision priority in one trailered commit
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf set-priority refuses an out-of-range level and a non-gap/decision target
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf add --priority sets it at creation, gated on kind like --area
      status: met
      tdd_phase: done
    - id: AC-4
      title: set-priority ships completion wiring and an aiwf-set-priority skill
      status: met
      tdd_phase: done
---

# M-0262 — Add the priority write surface: set-priority verb and add --priority

## Goal

Give operators two trailered ways to set a gap's or decision's `priority`: a dedicated `aiwf set-priority <id> <level>` verb for changing it later, and a `--priority` flag on `aiwf add` for setting it at creation.

## Context

The field and its validation land in the field milestone; this milestone makes it writable through verb routes so a value gets in without hand-editing frontmatter (which trips `provenance-untrailered-entity-commit`). `set-priority` is a deliberate second member of a `set-X` family alongside `set-area`, not a general-purpose edit verb — the codebase has no generic "edit a frontmatter field" verb and isn't gaining one here.

## Acceptance criteria

<!-- Seeded via `aiwf add ac`; each starts at tdd_phase: red. -->

### AC-1 — aiwf set-priority sets a gap/decision priority in one trailered commit

### AC-2 — aiwf set-priority refuses an out-of-range level and a non-gap/decision target

### AC-3 — aiwf add --priority sets it at creation, gated on kind like --area

### AC-4 — set-priority ships completion wiring and an aiwf-set-priority skill

## Constraints

- `set-priority` follows the two-file verb pattern (`internal/verb` + `internal/cli/…`), emits `aiwf-verb` / `aiwf-entity` / `aiwf-actor` trailers, and refuses a no-op set.
- `aiwf add --priority` is gated on kind exactly the way `--area` already is — legal on gap/decision, refused elsewhere.
- Both writers route validation through the field milestone's closed-set predicate; neither re-implements the value check.

## Design notes

- Verb wiring must satisfy the discoverability chokepoints: a new `aiwf-set-priority` skill (per `skill_coverage.go`) and completion wiring for the `<level>` arg (per the completion-drift test), mirroring `set-area`'s `CompleteAreaValueArg`.
- The `aiwf-add` skill gains a `--priority` line; the new `aiwf-set-priority` skill documents the verb.

## Surfaces touched

- `internal/verb/`, `internal/cli/` — the `set-priority` verb and its cobra wiring; the `--priority` flag on `aiwf add`.
- `internal/skills/embedded/` — the new `aiwf-set-priority` skill and the `aiwf-add` update.

## Out of scope

- Reading or filtering by priority (the read-surface milestone) and rendering it (the render milestone).
- A general-purpose `aiwf set <id> --field=value` verb.

## Dependencies

- M-0261 — the field, closed-set predicate, and validation must exist first.

## References

- G-0078 — the ratified design decisions (verb choice, creation-time flag).
- The `set-area` verb — `internal/cli/setarea/` — the pattern this verb copies.

## Work log

### AC-1 — aiwf set-priority sets a gap/decision priority in one trailered commit

`SetPriority` verb (`internal/verb/setpriority.go`) and the `set-priority` CLI command (`internal/cli/setpriority/`) land, wired into `root.go` · commit 91f42294 · tests 24/24 new (7 verb, 4 CLI-unit, 13 integration incl. a diag-logging case), 6/6 mutants killed.

The verb also ships a `--clear` flag, beyond AC-1's literal title — added deliberately per CLAUDE.md's "what verb undoes this?" design rule: without it, the very first set (unset→set) would have no reversal path. Mirrors `set-area`'s established set/clear precedent rather than opening a fresh design question.

Two discoverability chokepoints needed same-commit fixes to keep the build green: `nonLegalityVerbAllowlist` (M-0123/AC-5's FSM-drift policy) gained a `set-priority` entry mirroring `set-area`'s ("FSM state is preserved"); `skillCoverageAllowlist` gained a *temporary* entry noting the real `aiwf-set-priority` skill lands in AC-4 — remove the allowlist entry when that skill ships.

**Readiness-check follow-up** (commit e02ab400): `make coverage-gate` — run for the first time only at the milestone's readiness check, not per-AC — caught a real gap the manual branch-coverage audit missed: the diag-logging `runID == ""` fallback-mint line is unreachable via `cli.Execute` (which always mints a real correlation id), so it only fires on a direct `Run()` call carrying a zero-value `cliutil.OutputFormat{}`. Every other wired verb has a dedicated `*Diag_FallsBackWhenOutputFormatCarriesNone` test for exactly this path (`remaining_verbs_fallback_test.go`); `set-priority` was missing its instance. Added `TestSetPriorityDiag_FallsBackWhenOutputFormatCarriesNone` mirroring `TestSetAreaDiag_FallsBackWhenOutputFormatCarriesNone`.

### AC-2 — aiwf set-priority refuses an out-of-range level and a non-gap/decision target

No new code: the refusal logic was written alongside AC-1's set path in the same commit (91f42294), since both live in the same `SetPriority` function body — `TestSetPriority_ValidationRefusals/{non-gap/decision_target,out-of-range_level}` and `TestSetPriority_OutOfRangeErrorNamesAllowedSet` already covered AC-2's exact claims. Closing this AC formally rather than silently folding it into AC-1, since the milestone spec tracks it as its own unit.

Added one mechanical gap this AC's own audit surfaced: the `wf-vacuity` pass for AC-1 hadn't specifically mutated the `IsAllowedPriorityLevel` guard (the core of AC-2's "refuses an out-of-range level" claim). Ran it now — inverting the guard produced 7 test failures including the direct out-of-range case — killed, no code change needed.

### AC-3 — aiwf add --priority sets it at creation, gated on kind like --area

`AddOptions.Priority`, the kind gate + closed-set value check in `validateAddOptsForKind`, `applyAddOpts` wiring, and the `--priority` flag on `aiwf add` (with completion) land · commit cd25bb89 · tests 5/5 new, 3/3 mutants killed.

Unlike `--area`, `--priority` needs no CLI-side config lookup — the level set is Go-hardcoded (`entity.IsAllowedPriorityLevel`), so the CLI layer passes the flag straight through and `Add` owns both the kind gate and the value check itself, routing through the same SSOT predicate `set-priority` uses. This is a real (not superficial) simplification versus `--area`'s `validateAreaMember` CLI-side helper, not an oversight.

Updating `AddOptions` with a new field is a signature-shape change to `verb.Add`'s existing test call sites (`internal/cli/add/add_error_paths_test.go`'s `runArgs` struct, plus two positional-arg call sites in `internal/cli/contract/diag_fallback_internal_test.go` and `internal/cli/integration/correlation_id_test.go`) — all three needed a mechanical update to insert the new parameter; `go vet ./...` (not just `go build`, which skips test files) is what actually caught them.

### AC-4 — set-priority ships completion wiring and an aiwf-set-priority skill

Completion wiring for `<level>` landed with AC-1's `ValidArgsFunction`; this AC's remaining scope is the real `aiwf-set-priority` skill (mirroring `aiwf-retitle`'s shape) and an `aiwf-add` update documenting `--priority` on the gap/decision rows · commit 15fbc6b1 · the new skill's `name:` matching its directory satisfies `skill-coverage` mechanically, so the AC-1 temporary allowlist entry is removed in the same commit — no `set-priority` special-case remains in `skillCoverageAllowlist`.

Pure documentation/registry change — no new branching code, so neither the branch-coverage audit nor `wf-vacuity`'s mutation probe applies (both need code with conditionals to walk/mutate). Mechanical evidence instead: `TestPolicy_SkillCoverageMatchesVerbs` and `TestM0123_AC5_ImplToSpec_VerbsCovered` both pass with the allowlist entry gone, and `TestList_AllShippedSkillsPresent`'s hardcoded skill roster (a pre-existing chokepoint unrelated to this milestone) needed a matching update — caught by the full test sweep, not anticipated in the Design notes.

### Independent pre-wrap review

An independent fresh-context reviewer audited the full diff against nine load-bearing claims (kind-gate correctness, `--clear`'s omitempty behavior and dual-layer mutex, single-source-of-truth level validation, AC-3's kind gate end-to-end, AC-2's "no new code" honesty, the allowlist cleanup, completion wiring reachability, absence of a divergent kind-check, and real (non-tautological) test assertions) — all nine held up under independent measurement. One non-blocking finding: `aiwf add --priority` had statement-level coverage but no CLI-seam integration test (every priority-behavior test drove `verb.Add` directly, bypassing the flag → `Run()` param → `AddOptions.Priority` wiring the CLI layer owns). Fixed in-review with `TestRunAdd_PrioritySetViaDispatcher` and `TestRunAdd_PriorityRejectedForNonCarryingKind` (commit 74700b42), mirroring `add_area_test.go`'s existing dispatcher-level pattern. Re-verified: full build, `go vet`, full test suite, `make lint`, `make coverage-gate` all green after the fix.

No `wf-rethink` design-quality unit applies — the milestone introduces `internal/cli/setpriority/` as a mechanical extension of the pre-existing `set-area` verb-family pattern (per this spec's own Context section), not a new module boundary, core abstraction, or data model.

## Decisions made during implementation

None — all decisions are pre-locked in `## Design notes` and G-0078's ratified decisions; the `--clear` flag addition (noted under AC-1 above) is a direct application of the already-referenced `set-area` precedent, not a fresh design fork.

## Validation

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test -race -parallel 8 ./...` (`make test-race`) — all packages pass. (Two isolated, unrelated flakes surfaced across the session's full-suite runs — `TestWorktreeRitualsCheckHook_NotAWorktreeExitsZeroSilently`, `TestCheckListInvariant_RealBinary_DetectsAGenuineDivergence`, `TestRun_FixtureRejected_OneFailingValid` — none touch priority/add/setpriority code; each passed cleanly in isolation, consistent with this repo's known full-parallel race/git-subprocess fan-out flakiness.)
- `make lint` (full `golangci-lint` set) — 0 issues.
- `make coverage-gate` (diff-scoped branch-coverage audit + firing-fixture meta-gate) — clean, including after the independent review's fix.
- `aiwf check` — 0 error findings; 1 pre-existing warning (`provenance-untrailered-scope-undefined`, no upstream configured for this unpushed branch — expected).
- Manual branch-coverage audits (per AC) and `wf-vacuity` mutation probes: AC-1 — 6/6 mutants killed (verb + CLI layers); AC-2 — 1 additional targeted mutation (`IsAllowedPriorityLevel` guard) killed; AC-3 — 3/3 mutants killed (`validateAddOptsForKind`'s three new conditionals). AC-4 introduced no branching code, so neither audit applies there.

## Deferrals

- (none)

## Reviewer notes

- **`--clear` was added beyond AC-1's literal title.** Deliberate, not scope creep: CLAUDE.md's "what verb undoes this?" rule requires a full reversal story, and without `--clear` the very first set (unset→set) would have no way back. Mirrors `set-area`'s established set/clear shape.
- **AC-2 shares its implementation and initial test evidence with AC-1** (both refusal paths were written in the same commit as AC-1's set path, since they live in the same `SetPriority` function body). Closed as its own AC per the spec's tracking, with one additional targeted mutation-kill pass specific to AC-2's claim, rather than silently folding it into AC-1's Work log entry.
- **`--priority` needs no CLI-side config lookup**, unlike `--area`'s `validateAreaMember` helper — the level set is Go-hardcoded, so `aiwf add`'s CLI layer passes the flag straight through and the verb layer (`Add`) owns the full kind-gate + value-check itself. A real simplification versus the `--area` precedent, not an oversight worth flagging as a follow-up.
- **AC-1 shipped with one real coverage gap** (the diag-logging `runID == ""` fallback-mint branch, unreachable via `cli.Execute` and only reachable through a direct `Run()` call) — caught by `make coverage-gate` at the readiness check, not by the manual audit or `wf-vacuity`; fixed with a test mirroring every other wired verb's `*Diag_FallsBackWhenOutputFormatCarriesNone` pattern. Worth naming as a standing gap in the manual-audit process: the diag-logging fallback line needs its own explicit checklist item, since it's easy to treat "the block ran" as sufficient when only the *inner* fallback path is actually the untested one.
- **One CLI-seam gap surfaced by the independent review** (`aiwf add --priority`'s dispatcher-level wiring had no integration test, only direct `verb.Add` tests) — fixed in-review; see "Independent pre-wrap review" above.
