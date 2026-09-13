---
id: M-0332
title: Retire the prose that restates the section set
status: in_progress
parent: E-0084
tdd: required
acs:
    - id: AC-1
      title: No surface restates the section set as a per-kind table
      status: met
      tdd_phase: done
    - id: AC-2
      title: Each surface that carried a retired table routes to the owner
      status: met
      tdd_phase: done
---
## Goal

Retire the prose that restates each kind's required section set, now that a
refusal carries it, so the set is stated in one place instead of three.

## Closes

- (none)

## Context

`entity.RequiredSections` has been the single definition of each kind's body
sections since E-0081, but two surfaces still enumerate that set in prose: the
`aiwf-add` skill's per-kind body-section table, and the body-sections table in
`docs/design/design-decisions.md`. They were safe to keep while nothing
enforced the set, because a reader had no other way to learn it.

M-0329 changed that. The verb seam now refuses a body that omits a required
section, so the set is discoverable by running the verb, and the prose copies
are second sources that can drift. Retiring them is what E-0084 means by the
deletion being the point rather than a tidy-up: the enforcement without the
deletion leaves the duplication in place.

## Acceptance criteria

### AC-1 — No surface restates the section set as a per-kind table

No shipped skill or normative design doc states a kind's required section set
as a table row keyed by that kind.

The check reads each kind's sections from the kernel, scans the shipped-skill
and normative-doc corpus for a table row whose first cell names a kind and
whose remaining cells carry that kind's sections, and asserts it finds none.
Both sides derive: a section added to a kind's set changes what the scan looks
for, so neither artefact can move without the other following.

The claim is scoped to the table shape because that is what states the set as
a set. Prose advising what to write *in* a section is not a second copy of the
membership and survives — the `aiwf-add` skill's per-kind authoring advice is
the case in point.

It is deliberately not stated as "these two passages are deleted". D-0070
retires prose-presence assertions over the shipped skill tree, and naming the
two files would pin the inputs rather than the rule: a copy added to a third
surface would go unreported.

Two surfaces keep a statement of the set and are correct to. The `aiwf-show`
skill's table names the JSON keys an envelope carries, and the prose templates
carry the headings as scaffold. Each derives from the kernel and is already
pinned by its own relationship check, so neither is the free-drifting copy this
milestone retires; this criterion adds no assertion about them.

### AC-2 — Each surface that carried a retired table routes to the owner

Each surface that carried a retired table routes the reader to the owner of
the set, and the route resolves.

A reader who previously learned the section set from the table must still be
able to reach it. The citation is checked rather than assumed: a route naming
a heading or symbol that no longer exists is reported.

## Constraints

- Deletion, not correction. A passage rewritten to agree with the kernel is
  still a second copy; the next drift is a rewording away.
- A reader who reached the set through the deleted table must still reach it.
  Removing the table without leaving a route makes the surface worse.

## Design notes

- The evidence here needs care. D-0070 retires prose- and heading-presence
  assertions over the shipped skill tree, and `aiwf-add` is in that tree, so
  "assert the table is gone" is the banned shape wearing a minus sign. What
  survives D-0070 is the relationship check: read the section names from the
  kernel and scan for the table shape that states them. Both sides derive, so
  a reword of the surrounding prose cannot break it and a re-added table
  cannot evade it.
- The scan covers the whole shipped-skill and normative-doc corpus rather than
  the two files being edited. A test pins a rule, not an input: the rule is
  that the set is stated once, and these two files are two of its inputs. The
  reach costs one list of roots, so breadth is not where the cost sits.
- No ledger of surfaces permitted to state the set. Measured, the scan finds
  nothing once the two tables are gone, and an empty expectation needs no
  escape hatch. One is designed if a legitimate case ever appears.

## Surfaces touched

- the `aiwf-add` skill's per-kind body-section table
- the body-sections table in `docs/design/design-decisions.md`

## Out of scope

- Changing what any kind's required set contains. That set is E-0081's answer.
- The push seam. Its own milestone owns it, and neither waits on the other:
  what makes these passages safe to delete is the verb seam M-0329 landed.
- G-0530's question of whether the milestone template's structured-data
  sections should exist at all. That asks whether a section is worth carrying;
  this asks only where the set is written down.

## Dependencies

- M-0329 — delivered the refusal that makes the prose redundant.

## Coverage notes

- (none)

## References

- D-0070 — why the evidence is a relationship check rather than a phrase assertion
- E-0081 — gave the section set one owner
- M-0329 — delivered the verb seam
- G-0571 — the hole this closes, jointly with the push-seam milestone; its own
  body still carries superseded counts and a claim M-0329 falsified
- G-0530 — the adjacent, out-of-scope question

## Release note

## Decisions made during implementation

- None new. The corpus-wide scope of AC-1's census, and the absence of a
  ledger of surfaces permitted to state the set, were both settled into
  `## Design notes` before implementation began.

## Validation

Run against the milestone branch tip in the devcontainer — linux/amd64,
go1.25.11, `aiwf` built from that tip.

- `make check-fast` — `go vet` plain and under `-tags stress` and
  `-tags testpins`; golangci-lint 0 issues; `go test -parallel 8 ./...` ok
  across every package.
- `make coverage-gate` — green. Covers the diff-scoped statement gate and the
  firing-fixture meta-gate.
- `aiwf check` — 0 errors, 12 warnings, none of them on this milestone. The
  warnings are the standing archive backlog and the provenance audit skipping
  itself because this worktree has no upstream configured.
- Vacuity probes, each mutating one file in memory and restoring it
  byte-identically afterwards. AC-1: a detector rewritten to match nothing
  leaves the census passing and is caught only by `TestKindStatedByRow`;
  removing the section-containment check fails both. AC-2: renaming the cited
  symbol fails the resolve test alone, deleting the link fails the
  section-route test alone.

The first probe is the load-bearing one. The census asserts an absence, so it
cannot distinguish a clean tree from a detector that matches nothing, and the
pair is sufficient only together.

## Deferrals

- G-0674 — the `--principal` flag's help text cites an internal iteration
  label, found at this milestone's preflight. It lands in none of the files
  this work touches, so it takes its own change rather than an inline fix.

## Reviewer notes

- (none)
