---
id: E-0070
title: 'AC contract-first discipline: plan-time content, red-first ordering'
status: proposed
---

# E-0070 — AC contract-first discipline: plan-time content, red-first ordering

## Goal

Close the two remaining gaps in the kernel's "TDD discipline must not depend
on the LLM's behavior" guarantee: acceptance-criterion contracts land on
main before implementation starts, and red-first test-then-code ordering is
mechanically checked rather than trusted from a self-reported phase
timeline.

## Context

Two related gaps surfaced from a single design conversation (recorded in
D-0047) about closing structural discipline holes the kernel's own ritual
text already names but doesn't enforce:

- **G-0252** — `wf-tdd-cycle` asks for a failing test before the
  implementation, but nothing enforces the ordering: the phase-transition
  commits (`--phase red`/`--phase green`/`--phase done`) are metadata-only,
  and the actual test-plus-implementation code lands in one combined commit
  at the end of the cycle. The skill's own text names the gap ("a phase
  ladder stamped in a batch later is indistinguishable from one
  back-stamped after the fact") without closing it.
- **G-0440** — `aiwfx-plan-milestones` merges planning to main without ever
  calling `aiwf add ac` — that happens inside `aiwfx-start-milestone`'s
  preflight instead, one FSM stage after the milestone is already visible on
  main. The result: draft milestones sit on main with zero or unpopulated
  AC entities, invisible to any reader without the epic's worktree checked
  out.

This builds on the already-shipped `G-0216`/`D-0039` AC-completeness guard
family (`internal/check/acs.go`), which established the
block-at-transition/warn-at-rest pattern this epic's mechanisms extend one
FSM stage earlier (G-0440) and apply to a new dimension — ordering, not
just presence (G-0252).

## Scope

### In scope

- A working-tree diff-shape check on `aiwf promote M-NNN/AC-N --phase
  red|green` (`internal/verb/ac.go`): `--phase red` refuses when the
  working-tree diff against HEAD touches any non-test path; `--phase green`
  refuses unless the diff has grown to include a non-test path since red.
  Test-path classification via a glob, reusing (or extending) the areas
  `paths:` oracle pattern.
- Moving `aiwf add ac` and AC-body content-filling from
  `aiwfx-start-milestone`'s preflight into `aiwfx-plan-milestones`, before
  its merge-to-main step.
- A new warning-severity check-time finding (extending
  `internal/check/acs.go` alongside `milestoneDoneIncompleteACs`) surfacing
  a `draft` milestone with zero ACs or empty AC bodies.
- Updating `wf-tdd-cycle` and `aiwfx-plan-milestones`/`aiwfx-start-milestone`
  skill text to match the new mechanics.

### Out of scope

- Running the test suite at promote-time to verify fail-then-pass (rejected
  in D-0047 on cost/language-coupling grounds, the same objection as
  D-0038).
- A `--evidence`-style flag or an `aiwf-red-commit` SHA trailer (rejected in
  D-0047 — self-reported claims with an "existence not relevance" gap).
- Any change to `wf-vacuity` or `wf-review-code` — the semantic "does this
  test actually matter" judgment stays there by design (D-0047).
- Blocking (as opposed to warning) a `draft` milestone with incomplete ACs —
  draft is a legitimate mid-planning state (D-0047, mirroring D-0039's
  `done`-side reasoning).

## Constraints

- The diff-shape check must add zero friction to an honest cycle — it
  validates existing working-tree state, no new commit or trailer.
- Must remain stack-agnostic — test-path classification is glob-based, not
  toolchain-coupled (no `go test -list` equivalent).
- The AC-content-timing fix must not turn `aiwfx-plan-milestones` into a
  hard block — `draft` stays warn-only per D-0039's precedent.

## Success criteria

- [ ] `aiwf promote M-NNN/AC-N --phase red` refuses when a non-test path is
      already dirty in the working tree; `--phase green` refuses when no
      non-test path has become dirty since red.
- [ ] A milestone drafted via `aiwfx-plan-milestones` has its AC entities
      (`acs[]` + `### AC-N` bodies) created and populated before the
      ritual's merge-to-main step, not deferred to `aiwfx-start-milestone`.
- [ ] `aiwf check`/`aiwf status` surface a warning for any non-archived
      `draft` milestone with zero ACs or an empty AC body.
- [ ] G-0252 and G-0440 are both promoted to `addressed`.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Exact test-path glob convention (per-language default vs. per-milestone declared) | yes | settled at milestone-planning time |
| Does the new warning finding need its own grandfather/archive-scoping, or reuse `entity.IsArchivedPath` like its siblings | no | almost certainly reuse, per D-0039 precedent; confirm at milestone-planning |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Test-path glob misclassifies a legitimate file (e.g. a shared fixture under a non-test-looking path), causing a false-positive refusal | medium | `--force --reason` override (human-only, sovereign), same as every other FSM guard; glob configurable per project |
| An in-flight `tdd: required` milestone mid-cycle when this lands could trip the new phase check on its next promote | low | confirm at milestone-planning time whether any are currently mid-cycle (today: zero, per the current tree) |

## Milestones

<!-- filled at aiwfx-plan-milestones time -->

## References

- D-0047 — Contract-first AC timing and red-first ordering enforcement
- G-0252 — wf-tdd-cycle red-first ordering unguarded for consumer
  tdd:required AC cycles
- G-0440 — AC entities not created until start-milestone; milestones land
  bare on main
- G-0216 / D-0039 — the AC-completeness guard precedent this epic extends
