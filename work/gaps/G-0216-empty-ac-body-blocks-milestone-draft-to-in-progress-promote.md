---
id: G-0216
title: Empty AC body blocks milestone draft to in_progress promote
status: open
discovered_in: M-0160
---

## What's missing

The kernel's existing `entity-body-empty` finding fires at warning severity when an AC's body subsection under `### AC-N — <title>` contains no prose. The warning is advisory; nothing mechanical prevents a milestone from transitioning `draft → in_progress` while its AC bodies are empty.

Per [`docs/pocv3/plans/acs-and-tdd-plan.md`](../../docs/pocv3/plans/acs-and-tdd-plan.md) §1, the AC body is supposed to carry "prose detail (description, examples, edge cases, references)" — the **contract** that defines what the AC requires for it to be `met`. The kernel cross-checks the heading-vs-frontmatter pairing structurally (`acs-body-coherence`); the prose inside the heading is the load-bearing semantic content.

When the prose is empty at start-milestone time, the implementation drifts to "whatever the planning Q&A said" — but the planning Q&A is unrecorded. There is no durable, kernel-tracked contract against which to evaluate whether the implementation actually meets the AC. Reviewers and future readers see only the AC title.

Concrete instance: M-0160 was started with four ACs whose body subsections were empty. Each AC went through RED → GREEN → REFACTOR without the contract being written. The AC bodies were planned to be filled at wrap time (mirroring what M-0159 did) — but wrap-time fills are post-hoc relative to the implementation, which means the contract can be tuned to match whatever the implementation did. This is the "AC gaming" failure mode the contract-first discipline is supposed to prevent.

## Why it matters

The kernel's role per CLAUDE.md is "framework correctness must not depend on the LLM's behavior." Today the contract-first AC discipline depends on the operator (human or LLM) **remembering** to fill the AC body before starting work. The mechanism that should make it mechanical — the `draft → in_progress` promote — doesn't enforce it.

This converts a *load-bearing TDD discipline* (the AC is the test specification; it must be written before the test) into *operator vigilance*. The chokepoint is missing.

The acute symptom: M-0160 (and earlier M-0159) shipped AC bodies populated at wrap time, post-hoc. Discipline holds because the human is paying attention; if attention lapses, contracts can be silently tuned to match implementations.

## Proposed fix shape

A new kernel rule (or extension of an existing one) that refuses the `draft → in_progress` promote on a milestone when any AC's body subsection is empty. Sketch:

- **Verb-time gate.** `aiwf promote M-NNN in_progress` reads each AC's frontmatter entry, derives the corresponding body-subsection bounds (between `### AC-<N>` and the next heading or EOF), and refuses if any subsection contains no non-heading prose. Same `--force --reason "..."` override path the existing FSM uses.
- **Check-time finding.** A complementary finding (e.g. `milestone-in-progress-empty-ac-bodies` or extending `entity-body-empty/ac` with a milestone-status subcode) fires at error severity when a milestone is `in_progress` (or `done`) with any empty AC body subsection. Surfaces the inconsistency on every `aiwf check` until resolved.
- **Forward-only.** Existing `in_progress` milestones at the time of fix are grandfathered (the rule fires from the moment the rule lands; pre-rule milestones with empty AC bodies surface the finding for explicit `--audit-only` backfill or hand-fix). No history rewrite, no retroactive promote-refusal.
- **The contract is what `### AC-N` heading must non-empty body protects.** The Work Log / Validation / Reviewer notes sections (the wrap-time post-hoc sections) are NOT covered by this rule — they're operator narrative, kernel-blind by design.

## Test surface

The new rule needs:
- Verb-time refusal test: `aiwf promote M-NNN in_progress` with an empty AC body returns non-zero, names which AC's body is empty.
- `--force --reason` override test: same fixture with `--force --reason "..."` succeeds; the standing check still surfaces the inconsistency.
- Check-time finding test: a milestone fixture in `in_progress` with one empty AC body produces the expected finding at error severity (or whatever severity the design chooses).
- Grandfather test: a fixture representing the pre-rule state must be diagnosable but not retroactively refused — the rule fires on the post-rule state.

## Workaround

Until the kernel ships the fix, the discipline is **operator vigilance**: fill AC body subsections at `aiwfx-start-milestone` time, BEFORE running `aiwf promote M-NNN in_progress`. The `aiwfx-start-milestone` skill should mention this explicitly as the contract-first lock step.

For milestones started before this gap surfaced (M-0159, M-0160), the AC bodies are back-filled post-hoc with explicit acknowledgment in the commit message. This is the durable record; it does NOT claim the contract was locked ahead of time.

## Closing this gap

When the impl lands:
- Verb-time refusal in `internal/verb/promote.go` (or the milestone-specific promote handler)
- New check rule under `internal/check/` (or extension of existing)
- Hint table entry naming the verb-side fix (`aiwf edit-body M-NNN` to fill the prose, or `--force --reason "..."` for the exceptional skip)
- `aiwfx-start-milestone` skill updated to name the new pre-promote step
- Drift policy verifying the verb refuses the bad transition (sabotage-verifiable)
- Promote G-0216 to `addressed` with `--by M-NNNN`.

## Discovered in

M-0160 — the M-0160 wrap-prep audit surfaced that all four AC body subsections had been empty throughout the milestone's RED/GREEN/REFACTOR cycles, with the plan to fill them at wrap time. The wrap-time fill is post-hoc and violates the contract-first discipline; this gap names the structural fix that makes the discipline mechanical instead of vigilance-dependent.
