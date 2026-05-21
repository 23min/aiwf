---
id: M-0136
title: 'aiwf acknowledge-illegal: retroactive force trailer for historical violations'
status: in_progress
parent: E-0033
tdd: required
acs:
    - id: AC-1
      title: 'commit shape: trailers correct, human actor + reason required'
      status: met
      tdd_phase: done
    - id: AC-2
      title: predicate exempts SHAs targeted by aiwf-force-for trailer
      status: open
      tdd_phase: red
    - id: AC-3
      title: predicate still fires on un-acknowledged historical illegals
      status: open
      tdd_phase: red
    - id: AC-4
      title: rejects out-of-history SHA with typed error
      status: open
      tdd_phase: red
    - id: AC-5
      title: verb name + --reason auto-completion wired
      status: open
      tdd_phase: green
    - id: AC-6
      title: skill coverage per ADR-0006
      status: met
      tdd_phase: done
---
## What this milestone delivers

A retroactive sovereign-override mechanism for the `fsm-history-consistent` rule (M-0130): the ability to acknowledge a historical FSM-illegal commit without rewriting history and without listing SHAs in `aiwf.yaml`.

The mechanism is a new verb (working name: `aiwf acknowledge-illegal`) that creates a separate, current-day commit carrying an `aiwf-force-for: <historical-sha>` trailer plus an `aiwf-reason: "..."` and `aiwf-actor: human/<name>`. The `fsm-history-consistent` rule's `illegal-transition` predicate extends to walk acknowledgment commits in HEAD's reachable history; when an offending commit's SHA appears as an `aiwf-force-for` target, the finding is exempted.

## Why this milestone exists

M-0130 lands the `fsm-history-consistent` check rule. Per D-0010, AC-2's `illegal-transition` subcode catches every parent → child edge in HEAD's reachable history that violates the per-kind FSM. The kernel's own repo carries 4 such historical commits — squash-merges from the pre-AC-2 era that collapsed legal feature-branch progressions (e.g., `draft → in_progress → done`) into single FSM-illegal commits (e.g., `draft → done`). Going forward, AC-2 catches these at pre-push, so operators learn to either fix the workflow or use `--force --reason` deliberately. But the legacy commits need a clean acknowledgment path.

The user's framing in the M-0130 design conversation:

> "In the future, we should not have these illegal transitions anymore. But in the rare case that we have them, they should have been `--forced`, with reason, right? But do we really want to have 41 hashes in aiwf.yaml? Can we solve it in a better way?"

This milestone is the answer: a verb that extends `--force --reason` to act retroactively via a separate commit. The acknowledgment lives in git history (queryable via `aiwf history`), aligns with the existing `--force` sovereign-act semantics, and doesn't pollute `aiwf.yaml`. Per-acknowledgment reason + human/ actor preserves the audit trail.

## Design (sketch)

**New verb:** `aiwf acknowledge-illegal <sha> --reason "..."`

- Requires a `human/` actor (sovereign acts trace to a named human, per the kernel's existing rule).
- `--reason` is mandatory (mirrors `--force --reason`).
- Creates one commit per invocation with trailers:
  ```
  aiwf-verb: acknowledge-illegal
  aiwf-force-for: <historical-sha>
  aiwf-reason: <text>
  aiwf-actor: human/<name>
  ```
- The commit may be empty (no file changes) or may touch a small audit-log file at `.aiwf/historical-acknowledgments.md` for human readability — design choice deferred to ACs.

**AC-2 predicate extension:**

The `illegalTransitionFindings` predicate (in `internal/check/fsm_history_consistent.go`) gains a pre-emission step: for each illegal observation, walk HEAD's commit history for any commit carrying `aiwf-force-for: <observation.Commit>`. If found, exempt the finding.

The walk can be cached per `aiwf check` invocation — collect the full set of acknowledged SHAs once at the start, then check membership during predicate iteration. Avoids per-observation git calls.

**Tests:**

- Acknowledgment commit shape (trailers correct, actor required, reason required).
- AC-2 predicate exempts acknowledged SHAs.
- AC-2 still fires on un-acknowledged historical illegals.
- Acknowledgment for a commit that isn't in HEAD's history fails with a typed error (no silent accumulation).
- Auto-completion of the verb name and `--reason` flag.
- Skill coverage per ADR-0006.

**Kernel-repo housekeeping after this milestone lands:**

Run the verb 4 times against the current historical illegals (one per offending SHA — they all happen to share `f4ea7329`, so actually just once):

```
aiwf acknowledge-illegal f4ea7329 --reason "squash-merge from epic/E-21 era; intermediate FSM progression existed on the feature branch but was lost to the squash. Pre-AC-2 era; no longer reachable via a non-squash path."
```

After that, `aiwf check` on the kernel repo returns 0 errors, and pre-push is unblocked.

## Relates to

- M-0130 (parent dependency — lands the rule whose findings this verb exempts).
- D-0010 (the merge-commit policy that left these single-parent illegal commits as the residual real-finding class).
- G-0150 (the design question that led to D-0010 and surfaced this need).
- D-0008 (the audit-only design that explicitly excluded illegal-transition from the existing audit-only suppression; this milestone introduces a *separate*, more deliberate retroactive-force mechanism without contradicting D-0008's framing).

## Scope boundaries

- Out of scope: rewriting history (destructive); reverting the offending commits (impractical).
- Out of scope: a generic "exemption list" in `aiwf.yaml`. The acknowledgment must be a sovereign git-history event, not a config-file entry.
- Out of scope: extending the mechanism to other check rules. If a future rule wants similar retroactive override, design it then.

## Work log

Per-AC outcome notes. Phase + status timeline lives in `aiwf history M-0136/AC-<N>` — not duplicated here.

### AC-1 — commit shape: trailers correct, human actor + reason required

`verb.AcknowledgeIllegal(ctx, root, sha, actor, reason)` in `internal/verb/acknowledgeillegal.go` validates `--reason` non-empty + `--actor` `human/` prefix + the SHA pattern (via `gitops.ValidateTrailer`), then emits an `AllowEmpty` plan with the four trailers (`aiwf-verb: acknowledge-illegal`, `aiwf-force-for: <sha>`, `aiwf-actor: human/<name>`, `aiwf-reason: <text>`). New `TrailerForceFor = "aiwf-force-for"` constant in `internal/gitops/trailers.go` wired into `trailerOrder` + `ValidateTrailer` (7-40 hex). CLI dispatcher in `internal/cli/acknowledgeillegal/` (uses `FinishVerb`, not `DecorateAndFinish`, because the verb operates on a commit SHA not on entity FSM state). Wired into `internal/cli/root.go`. Policy extensions: `PolicyEmptyDiffCommitsCarryMarker` admits `TrailerForceFor` as a third recognized marker for intentionally-empty verb commits; `nonLegalityVerbAllowlist` records the verb's non-FSM nature. · commit `04484ee4` · 3 verb tests (CommitShape + RequiresReason × 3 sub + RequiresHumanActor × 4 sub) passing

### AC-2 — predicate exempts SHAs targeted by aiwf-force-for trailer

### AC-3 — predicate still fires on un-acknowledged historical illegals

### AC-4 — rejects out-of-history SHA with typed error

### AC-5 — verb name + --reason auto-completion wired

### AC-6 — skill coverage per ADR-0006

Per-verb embedded skill at `internal/skills/embedded/aiwf-acknowledge-illegal/SKILL.md` per ADR-0006. Documents when to use (and when NOT — fresh / FSM-legal-but-untrailered / non-human-actor cases route elsewhere), the four-trailer commit shape, the DAG-scoped predicate semantics (HEAD-reachable acknowledgments only, per-SHA not per-entity, disjoint from the `manual-edit` and `forced-untrailered` resolution paths), and the deliberate one-way design (CLAUDE.md §Designing a new verb — "what verb undoes this?" answer is *"you can't, and that's deliberate"*). Co-bundled with AC-1 because `PolicySkillCoverageMatchesVerbs` blocks pre-commit otherwise; this entry records the satisfied state. · commit `04484ee4` · PolicySkillCoverageMatchesVerbs passes

