---
id: G-0161
title: 'AntiRules() negative coverage: assert kernel does NOT enforce each ANTI'
status: open
---
## Problem

`spec.AntiRules()` carries 12 anti-rules (`ANTI-0001..ANTI-0012`) — prose statements pinning what the kernel deliberately does NOT police (e.g. *"a milestone is NOT required to have ≥1 AC"*, *"there is NO global AC allocator"*, *"--force cannot be wielded by a non-human actor"*). Each anti-rule is the canonical record of a deliberate non-rule.

M-0125 covers `Rules()` Illegal cells (18 cells, verb-time + check-time rejection shapes). AntiRules() are **explicitly out of scope** for M-0125 because they are not (Kind, FromState, Verb)-keyed — they're scope-by-negation prose, requiring a different test shape (assert *absence* of enforcement, not presence of rejection).

## Why it matters

Without mechanical coverage, an AntiRule can quietly become a Rule via implementation drift — somebody adds a new check rule that fires on a pattern the AntiRule says the kernel doesn't police, and nothing catches it. The AntiRule's status changes from "the kernel deliberately doesn't" to "the kernel actually does" without anyone noticing. The whole point of cataloguing AntiRules is to make these inversions visible; without mechanical coverage, the catalog is documentation that the code is free to diverge from.

## Fix outline

For each `ANTI-NNNN`, write a positive-of-the-negation test: construct a fixture/state that *would* trigger the anti-pattern if enforced, and assert the kernel **does not** treat it as illegal — verb succeeds (no exit-code rejection), `aiwf check --format=json` produces no finding code keyed on the anti-pattern.

Specific shapes per anti-rule:

- **ANTI-0001** (no `≥1 AC` requirement): create a milestone with 0 ACs, confirm `aiwf check` produces no `acs-required` finding.
- **ANTI-0002** (no `red-on-entry` rule): promote a milestone to `in_progress` with all ACs lacking `tdd_phase: red`, confirm no error.
- **ANTI-0003** (no global AC allocator): create two milestones, each with AC-1; confirm both coexist without a uniqueness finding.
- **ANTI-0004** (no AC tombstone): cancel an AC and confirm its position in `acs[]` is retained.
- **ANTI-0005** (no `reactivate` verb): invoke `aiwf reactivate`, assert "unknown command" exit.
- **ANTI-0006** (no event log / projection / hash chain / monotonic id): confirm `aiwf init` does not create `events.jsonl`, `.aiwf-graph.json`, `.aiwf-hash`, or `id-counter` files.
- **ANTI-0007** (no kernel branch-of-verb rule): invoke a mutating verb on any branch (main / feature / detached HEAD), confirm verb succeeds with no branch-specific check.
- **ANTI-0008** (`recommended_plugins` default empty): inspect `aiwf init`-produced `aiwf.yaml` and confirm `doctor.recommended_plugins:` is empty / absent.
- **ANTI-0009** (no shipped validators): inspect the binary or release artifacts; confirm no `cue` / `ajv` binary is bundled.
- **ANTI-0010** (`--force` not wielded by non-human actor): invoke a mutating verb with `--force` while the operator email matches an `ai/*` pattern, confirm rejection.
- **ANTI-0011** (no `pre-fail-tests-before-in_progress` rule): same as ANTI-0002 essentially — already covered there; this AntiRule can collapse to a citation under ANTI-0002.
- **ANTI-0012** (epic → active with zero milestones legal): promote an epic with zero child milestones to `active`, confirm no `epic-active-no-drafted-milestones` finding.

Most of these are short tests; collectively they form a 9–11 test addition under `internal/policies/`.

## Discovered in

M-0125 planning (AC scope discussion). The milestone body explicitly names `Rules()` Illegal cells; AntiRules() coverage is the symmetric concern that needs its own scope.

## Status

`open`.
