---
id: E-0076
title: Chokepoints for three documented rules that have no detector
status: cancelled
---
## Goal

Give three documented conventions the detectors they lack, so each stops reading
as enforced when nothing enforces it.

Addresses G-0465, G-0471 and G-0474.

## Context

This repo holds that framework correctness must not depend on an assistant — or a
human — remembering a rule, and that a guarantee depending on someone remembering
is not a guarantee. Three rules currently depend on exactly that. Each is stated
in an authoritative surface, each is held at review, and none has a mechanical
chokepoint. Two produced live evidence during E-0073; the third was surfaced by a
structural sweep after it wrapped.

The common shape matters more than any one instance: a rule without a detector
reads as enforced, so the next reader stops looking. Closing them together makes
the pattern visible rather than treating each as an isolated oversight.

The three are not equally tractable, and the epic should not pretend otherwise.
Two have a measured, cheap detector. The third has no obviously-mechanical core
and may resolve to a narrower detectable subset plus a named review obligation
for the rest — which is a legitimate outcome, provided it is recorded rather than
left implicit.

## Scope

**G-0471 — a verb run by a binary older than the worktree's source.** Working on
the kernel means the `aiwf` on PATH predates the tree. Every verb then runs older
logic, reads *and writes*, with no signal. Measured during E-0073: a met acceptance
criterion appeared to fail because the PATH binary predated its convergence guard,
and `aiwf update` materialized stale skills including the one for the verb that
milestone had just changed.

`doctor` reports skill drift at error severity, so that half is already detected;
what it does not report is the staleness itself. Its binary-staleness check compares
the running binary against `refs/remotes/origin/main`, never against the working
tree — the decisive miss, given the gap is titled for a binary older than the
*worktree's* source. It additionally skips tagged releases by shape, so a developer
many commits past a tag is the case it most declines to examine. And it lives in a
verb nobody runs at the moment the failure arrives.

Two predecessors bound this without covering it. G-0147 — titled for the missing
mechanical chokepoint — closed by shipping a `make diag-aiwf` convenience target
plus a documented discipline, which is a tool an operator must remember to reach for
rather than a chokepoint. G-0176 shipped real detection that skips this case twice
over, on the comparison target and on the tagged-release shape.

**G-0474 — blank-identifier unused-silencers.** CLAUDE.md bans `var _ = <ident>`
kept solely to quiet `unused`, a rule G-0451 asked for and G-0449 acted on by hand.
One instance survives, in a `_test.go` file the hand-scoped sweep did not cover.
Only whole-program reachability sees past the alias. Measured: a bare-identifier
regex matches exactly one site tree-wide with no false positives, including the
typed interface assertions and the deliberate policy fixtures that must not fire —
so a cheap detector is viable and does not require adding reachability machinery.

**G-0465 — shipped surfaces drifting from the verbs they describe.** Three separate
E-0073 review rounds each found more `--help` text, skill prose and doc claims that
no longer matched behaviour. Reading is the only detector, and it does not scale.
This is the hardest of the three and may resolve to a narrower mechanical subset
plus an explicit review obligation for the rest.

G-0479 is a concrete instance already in hand, and a good calibration target for
whatever detector emerges: the shipped epic template nests its out-of-scope
section one heading level below what the `entity-body-empty` rule's required
sections, the `aiwf add` scaffold, and the `aiwf show --format=json` body
contract all name, so an epic drafted from the template satisfies none of the
three. It is referenced here rather than absorbed because it shows the class is
mechanically detectable at least in part — the disagreement is between four
surfaces with fixed shapes, not between prose and behaviour.

## Out of scope

- **Deleting the other unreachable methods.** A whole-program reachability run —
  `deadcode` is in no linter set, no Make target and no workflow, so this is not
  something the gate reports — finds `PreflightBranchNotFoundError.Error` and
  `.Code` alongside the instance above. Both are retained by accepted D-0018 and
  scheduled for removal by open G-0417, coupled to a spec-table cleanup. Removing
  them here would preempt tracked work and contradict a recorded decision.
- **Making the code-health rubric enforcing.** ADR-0019 ships it as advisory
  deliberately. An `internal/policies/` test is kernel-internal and adds no consumer
  surface, so it does not contradict that ADR — but the ADR is not being revisited.
- **A general drift detector for all prose.** G-0465 is scoped to shipped surfaces
  describing verb behaviour, not to documentation correctness at large.
- **Adding whole-program reachability to the gate.** `deadcode` is in no linter
  set, Make target or workflow, so wiring it is new machinery with its own
  runtime and false-positive budget. The measurement above shows G-0474 does not
  need it. If a later rule does, that is its own decision.

## Constraints

- G-0471's detector compares against the **working tree**, not
  `refs/remotes/origin/main`, and does not skip tagged releases by shape. Those
  are precisely the two misses that make the existing check silent on this case.
- A detector that only fires from `aiwf doctor` does not close G-0471. The
  failure arrives during an ordinary verb run, and `doctor` is a verb nobody runs
  at that moment.
- Any reachability-shaped detector must not fire on deliberately-retained code.
  `PreflightBranchNotFoundError.Error` and `.Code` are held by accepted D-0018
  and scheduled by open G-0417; a detector that flags them contradicts a recorded
  decision.
- ADR-0019 ships the code-health rubric as advisory. This epic does not revisit
  it; a kernel-internal `internal/policies/` test adds no consumer surface and so
  does not contradict it.
- If part of G-0465 resolves to a review obligation rather than a detector, that
  obligation is written down in an authoritative surface. An unrecorded one
  reproduces the condition this epic exists to remove.

## Success criteria

- [ ] Running a verb from a binary older than the worktree's source produces a
      signal without the operator having to remember to run `aiwf doctor` — the
      comparison is against the working tree, and a developer many commits past a
      tag is covered rather than skipped.
- [ ] A `var _ = <ident>` unused-silencer added anywhere in the tree, `_test.go`
      files included, fails a gate; the surviving instance is gone.
- [ ] That detector does not fire on the typed interface assertions, the
      deliberate policy fixtures, or the code D-0018 retains.
- [ ] For G-0465, either a detector covers a named mechanical subset, or a
      recorded decision states that no subset is worth mechanizing — and in
      either case the residue is a written review obligation rather than an
      implicit one.
- [ ] G-0465, G-0471 and G-0474 are promoted to `addressed`.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Where the binary-staleness signal fires — every verb's prelude, the pre-push hook, or a build-time check | yes | milestone-planning. A check nobody runs at failure time is the defect being fixed, so the seam is the substance of the fix |
| What the staleness comparison costs per invocation, and whether that rules out the prelude | yes | measured at milestone-planning, alongside the seam choice |
| What mechanical subset of "a shipped surface describes current verb behaviour" is detectable, and what stays a review obligation | yes | G-0465 is the hardest of the three; scoped at milestone-planning after the two cheap detectors land |
| Whether the unused-silencer detector is the measured regex or something reachability-based | no | measured: the regex matches exactly one site tree-wide with no false positives, so the cheap option is viable |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| G-0465 has no obviously-mechanical core and absorbs the epic | high | it is scheduled last, after the two measured detectors; a recorded decision that no subset is worth mechanizing is an acceptable outcome, provided the review obligation is written down |
| A staleness check in every verb's prelude costs a `git` call per invocation and is disabled for being slow | medium | the seam is an open question with cost measured before it is chosen; the pre-push hook is a cheaper seam that still fires before work leaves the machine |
| A detector fires on deliberately-retained code and gets grandfathered wholesale, hollowing it out | medium | the retained set is named up front (D-0018 / G-0417); the measurement shows the cheap detector already excludes it |
| Three detectors land and each is individually cheap, so the shared pattern goes unstated and the next instance is filed as another isolated oversight | low | the shape is the epic's premise and belongs in whatever surface the review obligation lands in |

## Milestones

Not yet allocated; ids are assigned when `aiwfx-plan-milestones` runs. Candidate
deliverables, in execution order:

- The binary-staleness detector, at the seam chosen from the open question above
  — compared against the working tree, tagged releases included.
- The unused-silencer detector, plus removal of the surviving instance. Cheapest
  and fully measured; independent of the first.
- Shipped-surface drift — scope the mechanical subset, build its detector, and
  record the review obligation for the residue. Last, as the hardest and least
  defined.

## References

- G-0465 — shipped surfaces drift from the verbs they describe
- G-0471 — a verb run by a binary older than the worktree's source, with no signal
- G-0474 — blank-identifier unused-silencers have no detector
- G-0479 — the epic template's out-of-scope heading level; a measured instance of G-0465's class
- G-0147 — the predecessor that closed with a convenience target and a discipline
- G-0176 — the predecessor whose detection skips this case twice over
- G-0417 / D-0018 — the deliberately-retained methods a reachability detector must not flag
- ADR-0019 — the code-health rubric ships advisory
