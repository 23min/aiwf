---
name: aiwf-acknowledge-illegal
description: Use when `aiwf check` reports `fsm-history-consistent/illegal-transition` errors against historical commits (typically pre-`fsm-history-consistent`-era squash-merges that collapsed legal feature-branch progressions into a single FSM-illegal commit). The verb records a current-day empty commit carrying `aiwf-force-for: <historical-sha>` plus `aiwf-actor: human/...` and `aiwf-reason: "..."`, which the rule's predicate walks at check time to exempt the named SHA. Requires a `human/...` actor and a non-empty `--reason` — sovereign acts trace to a named human with written rationale.
---

# aiwf-acknowledge-illegal

The `aiwf acknowledge-illegal <sha>` verb is the retroactive sovereign-override mechanism for the `fsm-history-consistent` rule's `illegal-transition` subcode. It exists for historical commits that violate the per-kind FSM but cannot be cleanly fixed (squash-merges from the pre-rule era; force-pushed history; etc.) — running it produces a separate empty commit with a special trailer set, and the rule's predicate walks HEAD's history for those trailers to exempt the named SHAs.

The verb is the answer to the design question: *"In the future, we should not have illegal transitions. But for the legacy commits that pre-date the rule, do we really want to list SHAs in `aiwf.yaml`? Can we solve it in a better way?"* — yes: the acknowledgment lives in git (queryable via `aiwf history`), aligns with the existing `--force` sovereign-act semantics, and doesn't pollute `aiwf.yaml`.

## When to use

- `aiwf check` reports `fsm-history-consistent/illegal-transition` against a commit you've confirmed is intentional / unfixable (the typical case is a pre-rule squash-merge whose intermediate FSM steps were collapsed away).
- Pre-push is blocked by the error and rewriting history is wrong (shared trunk, force-push to main is forbidden, etc.).
- The operator is a named human with written rationale for accepting the violation.

## When NOT to use

- The illegal transition is fresh / fixable — re-route through `aiwf promote` or `aiwf cancel` (which only accept FSM-legal moves), or use `aiwf <verb> --force --reason "..."` at the time of the change.
- The transition is FSM-legal but lacks an `aiwf-verb:` trailer — that's the `manual-edit` subcode, cleared via `aiwf <verb> <id> --audit-only --reason "..."`. `acknowledge-illegal` is specifically for FSM-illegal flips that need post-hoc sovereign acceptance.
- You're acting as a non-human actor (LLM, bot) — the verb refuses non-`human/` actors at the gate, by design. If an automated process surfaces an illegal-transition finding, the human reviewing the result decides whether to acknowledge.

## What to run

```bash
# Acknowledge a historical illegal commit with a written rationale.
aiwf acknowledge-illegal f4ea7329 \
  --reason "pre-AC-2 era squash-merge from epic/E-21; intermediate FSM progression existed on the feature branch but was lost to the squash"

# The verb derives --actor from `git config user.email` by default
# (must resolve to a human/... identity); pass --actor human/<name> explicitly
# when needed.
aiwf acknowledge-illegal f4ea7329 \
  --actor human/peter \
  --reason "..."
```

The verb refuses with a typed error when:

- `--reason` is empty or whitespace-only.
- `--actor` is not `human/...`.
- `<sha>` doesn't match the 7-40-hex SHA shape (verified at write time via the trailer validator).
- (M-0136/AC-4) `<sha>` is not a commit reachable from HEAD.

## What the commit looks like

One empty commit (`git commit --allow-empty`) carrying:

```
aiwf acknowledge-illegal <short-sha>

<your reason text>

aiwf-verb: acknowledge-illegal
aiwf-force-for: <historical-sha>
aiwf-actor: human/<name>
aiwf-reason: <text>
```

The acknowledgment commit's SHA is itself in `aiwf history` going forward — queryable via `aiwf history <historical-sha>` once the cross-reference resolver lands (future scope).

## Predicate semantics

The `fsm-history-consistent` rule's `illegal-transition` predicate walks HEAD's reachable history at check time and builds a set of SHAs targeted by any `aiwf-force-for:` trailer. For each illegal-transition observation, if the offending commit's SHA appears in that set, the finding is exempted.

Properties of the exemption:

- **DAG-scoped**: only `aiwf-force-for` trailers in HEAD's reachable history count. A cherry-picked acknowledgment on a branch that doesn't include the original violation doesn't exempt findings on this branch.
- **Per-SHA**, not per-entity: one acknowledgment commit covers every entity touched by the historical SHA. A single `f4ea7329` ack clears illegal-transition findings against M-0072, M-0073, M-0074, and E-0020 in one shot.
- **Disjoint from `manual-edit`**: the `aiwf-audit-only` mechanism still clears the `manual-edit` subcode independently; `aiwf-force-for` clears `illegal-transition` only. The two cover different failure modes (FSM-legal-but-untrailered vs. FSM-illegal-but-sovereign).
- **Disjoint from `forced-untrailered`**: that subcode catches sovereign-act-shape transitions lacking the inline `aiwf-force:` trailer at the time of the act. Retroactive acknowledgment via `acknowledge-illegal` is for the rarer case where the commit can't be re-done (already merged, force-push forbidden) — typically squash-merges.

## Why empty-commit + trailer (vs. aiwf.yaml entry)

- **Audit trail is git-native.** `aiwf history` already walks commits for `aiwf-verb:` events; acknowledgments fit naturally. A YAML entry would need a parallel surfacing mechanism.
- **Per-acknowledgment rationale.** Each ack carries its own `aiwf-reason` — the YAML alternative would either drop the reason (lossy) or replicate the trailer in a comment field (redundant).
- **Sovereign-act alignment.** Existing `--force --reason "..."` records the human in the commit's trailer block. `acknowledge-illegal` is the same shape, just retroactive.
- **No allowlist drift.** A YAML allowlist accumulates entries that nobody knows whether they're still needed; a commit-trailer set lives in history and grows monotonically with reasons attached.

## What can't this verb do

- **Reverse itself.** There's no companion "un-acknowledge" verb. If you regret an acknowledgment, your options are: rewrite history (destructive, hostile to shared trunks); add a counter-acknowledgment via a future verb (not designed); or live with it. The verb is one-way by deliberate design — the spec's "What verb undoes this?" answer is *"you can't, and that's deliberate."*
- **Acknowledge a commit not reachable from HEAD.** Per M-0136/AC-4, the verb fails with a typed error rather than silently accumulating no-op acknowledgments. Reachable means: `git merge-base --is-ancestor <sha> HEAD` succeeds.
- **Acknowledge findings other than `illegal-transition`.** `forced-untrailered`, `manual-edit`, and `history-walk-error` each have their own resolution paths (`--force --reason` at the time of the act, `--audit-only`, and re-run / fix git store respectively).

## Related

- **M-0136** — this milestone.
- **M-0130** — implements `fsm-history-consistent` whose findings this verb exempts.
- **M-0137** — retrofits the rule to batched walker + surfaces walker errors as findings (the silent-swallow fix); a prerequisite to M-0136 so the predicate fires reliably.
- **D-0010** — merge-commit policy that left these single-parent illegal commits as the residual real-finding class M-0136 addresses.
- **G-0150** — the design conversation that surfaced the need for this verb (vs. an aiwf.yaml allowlist).
- **D-0008** — explicitly excluded `illegal-transition` from the existing `audit-only` suppression; M-0136 introduces the separate, more deliberate retroactive-force mechanism.
- **ADR-0006** — the per-verb-skill / topical-skill / allowlist judgment rule M-0136/AC-6 satisfies via this skill.
