---
id: E-0076
title: Chokepoints for three documented rules that have no detector
status: proposed
---
## Goal

Give three documented conventions the detectors they lack.

This repo holds that framework correctness must not depend on an assistant — or a
human — remembering a rule, and that a guarantee depending on someone remembering
is not a guarantee. Three rules currently depend on exactly that. Each is stated in
an authoritative surface, each is held at review, and none has a mechanical
chokepoint. Two produced live evidence during E-0073; the third was surfaced by a structural
sweep after it wrapped.

The common shape matters more than any one instance: a rule without a detector
reads as enforced, so the next reader stops looking. Closing them together makes
the pattern visible rather than treating each as an isolated oversight.

Addresses G-0465, G-0471 and G-0474.

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
typed interface assertions and the deliberate policy fixtures that must not fire.

**G-0465 — shipped surfaces drifting from the verbs they describe.** Three separate
E-0073 review rounds each found more `--help` text, skill prose and doc claims that
no longer matched behaviour. Reading is the only detector, and it does not scale.
This is the hardest of the three and may resolve to a narrower mechanical subset
plus an explicit review obligation for the rest.

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
