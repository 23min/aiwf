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
