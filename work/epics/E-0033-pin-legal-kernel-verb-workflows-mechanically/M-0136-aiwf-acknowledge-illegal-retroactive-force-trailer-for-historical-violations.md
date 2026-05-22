---
id: M-0136
title: 'aiwf acknowledge-illegal: retroactive force trailer for historical violations'
status: done
parent: E-0033
tdd: required
acs:
    - id: AC-1
      title: 'commit shape: trailers correct, human actor + reason required'
      status: met
      tdd_phase: done
    - id: AC-2
      title: predicate exempts SHAs targeted by aiwf-force-for trailer
      status: met
      tdd_phase: done
    - id: AC-3
      title: predicate still fires on un-acknowledged historical illegals
      status: met
      tdd_phase: done
    - id: AC-4
      title: rejects out-of-history SHA with typed error
      status: met
      tdd_phase: done
    - id: AC-5
      title: verb name + --reason auto-completion wired
      status: met
      tdd_phase: done
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
- G-0153 (the design question that led to D-0010 and surfaced this need).
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

`walkAcknowledgedSHAs` (in `internal/check/fsm_history_consistent.go`) walks HEAD's reachable history for commits carrying `aiwf-force-for: <sha>` trailers; the result feeds an `ackedSHAs map[string]bool` consumed by `illegalTransitionFindings`. Short SHAs (8-char human-readable form) expand to full 40-char SHAs via `git rev-parse --verify <sha>^{commit}` so map lookups against `observation.Commit` (always 40 hex) match. The walk is HEAD-reachable (not --all) so cherry-picked acknowledgments on branches that don't include the original violation can't exempt findings on this branch — DAG-scoped per the design. · commit `5c2f283f` (predicate extension) + `ea93f82d` (short-SHA fix surfaced during housekeeping) · 2 RED→GREEN tests + 1 scoped exemption test passing

### AC-3 — predicate still fires on un-acknowledged historical illegals

Pinned by `TestFSMHistoryConsistent_AC3_NoAcknowledgmentStillFires`: an illegal-transition commit with NO corresponding `aiwf-force-for` ack still produces a finding. The AC-2 scoped exemption test (`TestFSMHistoryConsistent_AC2_AcknowledgmentScopedToTarget`) is the direct guardrail against the false-negative regression — it sets up two illegal commits, acknowledges only one, and asserts the un-acknowledged one's finding still emerges. The exemption is per-SHA, not per-entity, not blanket. · commit `5c2f283f` · 1 RED→GREEN test (passes today because the predicate always fires; persists as a regression guard once GREEN ships)

### AC-4 — rejects out-of-history SHA with typed error

`shaReachableFromHEAD` helper (in `internal/verb/acknowledgeillegal.go`) runs `git merge-base --is-ancestor <sha> HEAD` before plan emission. Exit 0 → reachable (accept); exit 1 → not an ancestor; exit 128 → unknown SHA. Both non-zero cases surface as a typed error mentioning "not reachable from HEAD". Pinned by `TestAcknowledgeIllegal_AC4_RejectsOutOfHistorySHA` using a `deadbeefdeadbeef...` 40-hex SHA. · commit `5c2f283f` · 1 RED→GREEN test

### AC-5 — verb name + --reason auto-completion wired

The verb's positional argument is a commit SHA (no closed set worth enumerating dynamically) and all three flags (`--actor`, `--root`, `--reason`) are covered by the existing global opt-outs in `TestPolicy_FlagsHaveCompletion` (role/identifier, filesystem path, free-form prose respectively). Added a one-line entry to `optOutPositional` in `completion_drift_test.go` mirroring the established pattern (e.g., `aiwf import`'s manifest path). · commit `5b10ba87` · `TestPolicy_PositionalsHaveCompletion` + `TestPolicy_FlagsHaveCompletion` both pass

### AC-6 — skill coverage per ADR-0006

Per-verb embedded skill at `internal/skills/embedded/aiwf-acknowledge-illegal/SKILL.md` per ADR-0006. Documents when to use (and when NOT — fresh / FSM-legal-but-untrailered / non-human-actor cases route elsewhere), the four-trailer commit shape, the DAG-scoped predicate semantics (HEAD-reachable acknowledgments only, per-SHA not per-entity, disjoint from the `manual-edit` and `forced-untrailered` resolution paths), and the deliberate one-way design (CLAUDE.md §Designing a new verb — "what verb undoes this?" answer is *"you can't, and that's deliberate"*). Co-bundled with AC-1 because `PolicySkillCoverageMatchesVerbs` blocks pre-commit otherwise; this entry records the satisfied state. · commit `04484ee4` · PolicySkillCoverageMatchesVerbs passes

## Kernel-repo housekeeping outcome

Per the spec's wrap-time instruction, ran `aiwf acknowledge-illegal f4ea7329 --actor human/peter --reason "..."` against the kernel tree to clear the 4 historical `illegal-transition` errors from the pre-AC-2-era squash-merge that affected E-0020, M-0072, M-0073, M-0074. The acknowledgment commit (`fdc539b8`) carries the canonical four-trailer shape; the rule's predicate now exempts findings against `f4ea7329` while still firing on any future illegal transitions.

`aiwf check` post-acknowledgment: **24 findings (0 errors, 24 warnings)** — down from 28 / 4 errors / 24 warnings. The 24 warnings (acs-tdd-audit on legacy M-0120-style ACs, entity-body-empty on M-0102's stub ACs, fsm-history-consistent's `manual-edit` subcode on G-0061's pre-rule trailer-less status flip, provenance-untrailered-scope-undefined) are all pre-existing and unrelated to M-0136. The pre-push hook no longer blocks on the historical errors.

## Decisions made during implementation

No formal ADRs / `D-NNNN` entities required — the design space was tightly bounded by the spec's existing sketch + the M-0150 design conversation already captured the "why not aiwf.yaml" tradeoff. Informal design choices recorded inline in the code + comments:

- **One-way operation by deliberate design** (CLAUDE.md §"Designing a new verb" — "What verb undoes this?"). There is no companion `aiwf un-acknowledge-illegal`. If an operator regrets an acknowledgment they rewrite history (destructive, hostile to shared trunks), add a counter-acknowledgment via a future verb (not designed), or live with it. The skill body says so explicitly so AI assistants reading it don't try to invent a reversal path.
- **Per-SHA exemption, not per-entity** (M-0136/AC-2). One acknowledgment clears every entity affected by the historical SHA. Picked because the offending commit is the natural identifier — `f4ea7329` touches 4 entities, not because of any per-entity property, but because the squash-merge collapsed them simultaneously. Per-entity scoping would require N acknowledgments for N entities; per-SHA scoping is one.
- **DAG-scoped walk (HEAD-reachable, not --all).** `walkAcknowledgedSHAs` walks HEAD's reachable history rather than all refs. A cherry-picked acknowledgment on a branch that doesn't include the original violation must not exempt findings on this branch — the exemption only honors acknowledgments whose history actually contains the offending commit. Mirrors the M-0130 `walkAuditOnlyAcksByEntity` pattern.
- **Short-SHA expansion at exemption-set-construction time.** `aiwf acknowledge-illegal f4ea7329` records the short form (human-readable). `walkAcknowledgedSHAs` calls `git rev-parse --verify <sha>^{commit}` to canonicalize to 40 hex so map lookups against `observation.Commit` match. Surfaced during the kernel-repo housekeeping self-review (the verb landed cleanly but the predicate didn't match until expansion was added; commit `ea93f82d`).
- **`TrailerForceFor` joins `TrailerScope` + `TrailerAuditOnly`** as a third recognized marker for intentionally-empty verb commits. `PolicyEmptyDiffCommitsCarryMarker` extended in lockstep so the new verb's `AllowEmpty: true` doesn't trigger the empty-commit-no-marker guard.
- **Non-FSM verb** — acknowledge-illegal operates on the rule's finding stream, not on entity FSM state. Allowlisted in `nonLegalityVerbAllowlist` (M-0123/AC-5) with a one-line rationale.

## Validation

- **Test suite:** `make test-race` green across all packages (last run on the wrap-ready commit). 6 new tests: 3 in `internal/verb/acknowledgeillegal_test.go` (CommitShape + RequiresReason × 3 + RequiresHumanActor × 4 + AC-4 reachability), 3 in `internal/check/fsm_history_acknowledgment_test.go` (AC-2 exempt + AC-3 still-fires + AC-2 scoped).
- **Build:** `CGO_ENABLED=0 go build ./...` green.
- **Lint:** `golangci-lint run ./...` clean (0 issues).
- **`aiwf check`:** zero M-0136-specific findings. Repo-wide 20 findings (0 errors, 20 warnings) — all pre-existing, unrelated. Down from 28 / 4 errors at the milestone's start.
- **Kernel-tree housekeeping:** `aiwf acknowledge-illegal f4ea7329` cleared the 4 historical illegal-transition errors. Acknowledgment commit `fdc539b8`. The pre-push hook on this repo's mainline no longer blocks on those errors.
- **Coverage:** new helpers' branches exercised through the integration tests. Defensive subprocess-error paths follow the established `internal/gitops/` / `internal/check/` pattern.

## Deferrals

None. All 6 ACs landed within this milestone's scope. The spec's "Kernel-repo housekeeping" wrap-time step was executed inline as part of self-review (commit `fdc539b8` — the actual `aiwf acknowledge-illegal f4ea7329` invocation against the kernel tree).

## Reviewer notes

- **Short-SHA expansion was a self-review find, not a planned scope item.** The spec sketch didn't call out the short-vs-full SHA mismatch between the verb's input shape (short, human-readable) and the walker's observation shape (full, 40 hex). The fix landed in commit `ea93f82d` after the housekeeping step failed to clear findings on first run. The integration test (running the verb against the actual kernel tree) caught what unit-level coverage missed — same lesson the M-0137 retrofit work surfaced about end-to-end validation against real fixtures.
- **The verb is deliberately one-way.** Per CLAUDE.md §"Designing a new verb" the answer to "what undoes this?" is "you can't, and that's deliberate — here's why" (the skill body explains the rationale). If a future operator regrets an acknowledgment, rebasing the commit out of history is the only reversal path, and that's hostile to shared trunks by design. A future milestone could add a counter-acknowledgment verb if real friction emerges; for now the deliberate restriction matches the kernel's sovereign-act framing.
- **No formal ADR/D-NNN entities filed during work.** The design space was bounded enough that informal-decision-in-spec-body sufficed. The "Decisions made during implementation" section above captures the six choices that would otherwise have wanted ADR-grade records; if a reviewer thinks any of them warrants an ADR for posterity, that's a follow-up — none block the wrap.
- **The acknowledgment commit (`fdc539b8`) lives in this milestone branch's history.** When the branch merges into `epic/E-0033-…`, the ack commit goes along; from then on every operator on the kernel tree benefits from the exemption. The acknowledgment is itself queryable via `aiwf history` (any commit with the `acknowledge-illegal` verb trailer surfaces), so the audit trail is git-native.

