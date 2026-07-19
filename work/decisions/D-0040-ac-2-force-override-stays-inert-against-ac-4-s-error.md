---
id: D-0040
title: AC-2 force override stays inert against AC-4's error
status: proposed
priority: medium
relates_to:
    - M-0268
    - D-0039
---
## Question

M-0268/AC-2 refuses `aiwf promote <milestone> in_progress` when an AC's body is empty, and its own spec text (and error message, as originally written) promised a `--force --reason` override, mirroring AC-1's own zero-AC guard. M-0268/AC-4 adds a new error-severity check-time finding (`acs-empty-body`) that fires on exactly the state AC-2 guards. `aiwf promote`'s existing `projectionFindings` mechanism runs unconditionally on every call — including forced ones — and refuses to produce a commit plan when the projected post-mutation tree carries a new error-severity finding. Since every AC-2 refusal condition is also an AC-4 refusal condition, `--force` can never actually land the commit for this specific guard, even though its verb-time code path is structurally force-gated the same way AC-1's is. Should AC-4 be downgraded to a warning to restore AC-2's literal force-override promise, should the projection check be special-cased to let force bypass AC-4 specifically, or should this asymmetry be accepted as the correct outcome and AC-2's own contract corrected instead?

## Decision

Accept AC-4's error severity as effectively unconditional for this specific interaction. AC-2's `--force` still skips its own verb-time Go-error refusal (the code path is unchanged, matching AC-1's structure), but the resulting commit still cannot land while any AC's body is empty — `Promote` returns a `Result` carrying the `acs-empty-body` finding and a `nil` Plan, the same shape `TestPromote_ForceStillFailsCoherence` already pins for a status-valid violation. AC-2's error message and the M-0268 spec's AC-2 body no longer claim `--force` overrides this state. No change to AC-4's severity and no new bypass mechanism.

## Reasoning

KISS and YAGNI both favor this over the alternatives. The honest fix for an empty AC body — `aiwf edit-body <milestone-id>`, one line of real prose — is cheaper than constructing a `--force --reason` invocation in the first place, so there is no realistic scenario where an operator genuinely needs force to get unblocked here; building a bypass mechanism (a `skipDuringProjection` entry, or a severity downgrade) to solve a problem that already has a trivial honest path is premature complexity for no real benefit.

Downgrading AC-4 to warning (matching AC-3's own warning-severity pairing) was rejected because it would weaken more than just the force interaction: it also softens the guarantee at `done` for a milestone that started clean (AC-2 satisfied at start) but had its AC body hand-edited back to empty afterward — that drift would then only warn, not block the pre-push hook, contradicting AC-4's own explicit "error finding" title and G-0216's actual guarantee.

Adding `acs-empty-body` to the existing `skipDuringProjection` carve-out was rejected because that mechanism's documented rationale is a different, unrelated problem (in-memory projection hasn't caught up with a verb's own pending disk write, e.g. `add --body-file`). Repurposing it here to mean "let force bypass this specific error" would mix two unrelated semantics under one flag, misleading a future reader who sees the entry and assumes projection staleness.

The asymmetry between AC-1 (force genuinely works) and AC-2 (force is practically inert) is not arbitrary: it tracks a real distinction the two guards protect. AC-1/AC-3 guard whether a milestone has any acceptance criteria at all — D-0039 point 2 already establishes that a permanently zero-AC milestone is a legitimate end state, so a soft, force-overridable stance is correct there. AC-2/AC-4 guard something categorically different — an AC that was explicitly created (a stated promise: "this is a thing we're doing") but never given real content. That is a broken promise, not a legitimate minimalist choice, and D-0039 point 3's "archive-scoped, error-severity, forward-only" framing for AC-4 already signals it should be treated as a genuine defect, not a state to make gracefully overridable.

This mirrors the one existing precedent already in the codebase for the same class of interaction: `TestPromote_ForceStillFailsCoherence` establishes that `--force` relaxes FSM-transition legality and named verb-time preconditions, but never check-time tree coherence. AC-4's error, once it exists, falls into that same "coherence" category — this decision just makes that consequence for AC-2 explicit rather than leaving it as a silent, surprising side effect discovered only by an operator who tries `--force` and gets a different-looking refusal than expected.
