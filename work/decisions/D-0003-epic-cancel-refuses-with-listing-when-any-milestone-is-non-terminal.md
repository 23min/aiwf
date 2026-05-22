---
id: D-0003
title: Epic cancel refuses with listing when any milestone is non-terminal
status: proposed
relates_to:
    - E-0033
---
## Sources

- First-principles: R-FP-0074 (legal-workflows-first-principles.md, §3a milestone × epic)
- Class: FP-only — Pass A is silent (impl does not enforce cancel-cascade behavior today).

## Resolution

Adopt refuse-with-listing pattern, mirroring `MilestoneCanGoDone` (transition.go:193). `aiwf cancel E-NNNN` refuses if any child milestone has status in `{draft, in_progress}`, printing the offending milestone ids; the operator handles each (cancel, promote-to-done, or otherwise dispose) before retrying.

Rationale:

- Mirrors the existing kernel pattern for completion-coherence (`milestone-done-incomplete-acs`): same shape, same UX, same listing-of-offenders affordance. Uniform mental model across the kernel for *"cancel/done a parent with non-terminal children → refuse."*
- Auto-cascade was considered and rejected: a multi-entity verb violates the *"one verb operates on one entity"* idiom that holds elsewhere; auto-cascade also loses per-milestone audit narrative (each child's history would show "cancelled as part of E-NNNN cancel" rather than a deliberate per-entity disposition).
- No-cascade silence was considered and rejected: leaves the tree in inconsistent state (cancelled epic owning non-terminal milestones) with no kernel chokepoint.
- Warn-and-proceed was considered and rejected: weaker than refuse; the chokepoint should be at write-time, not after.

Forces a per-milestone disposition decision at cancel-time. Some children may need `done` (work already complete; just bureaucratic close-out); others `cancelled`. Refusing surfaces this distinction explicitly.

## Spec cell

`internal/workflows/spec` — `Rule{Kind: entity.KindEpic, FromState: <any non-terminal>, Verb: "cancel", Preconditions: [all-children.status ∈ terminal-set], Outcome: Legal, RejectionLayer: VerbTime, BlockingStrict: true, ExpectedErrorCode: "epic-cancel-non-terminal-children"}`.

## Follow-up

Impl change scope-out of M-0123. File a gap → milestone under E-0033 for: precondition in `aiwf cancel` verb body, new finding code `epic-cancel-non-terminal-children` in `internal/check/`, integration test exercising the refuse path. D-0004's (Q6) milestone-cancel-non-terminal-acs work likely shares the same impl gap.
