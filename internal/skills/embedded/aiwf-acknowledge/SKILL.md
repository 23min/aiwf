---
name: aiwf-acknowledge
description: Use when `aiwf check` reports a finding you've judged to be intentional and want a sovereign, reasoned exemption recorded in git. Two subverbs — `aiwf acknowledge illegal <sha>` exempts a historical commit from the fsm-history-consistent / provenance audit rules; `aiwf acknowledge mistag <id>` accepts an `area-mistag` warning as legitimate cross-cutting work. Each records a current-day empty commit carrying `aiwf-verb` / `aiwf-actor` / `aiwf-reason` (+ the target trailer) that the matching check rule walks to exempt the named target. Both require a `human/...` actor and a non-empty `--reason` — sovereign acts trace to a named human with written rationale.
---

# aiwf-acknowledge

`aiwf acknowledge` is the sovereign-acknowledgement verb group: each subcommand records a human-authored, reasoned acceptance of something a kernel audit rule flagged. Both share one shape — a current-day **empty commit** (`git commit --allow-empty`) carrying `aiwf-verb` / `aiwf-actor: human/...` / `aiwf-reason: "..."` plus a target trailer, which the matching check rule walks at check time to exempt the named target. The acknowledgement lives in git (queryable via `aiwf history`), aligns with the existing `--force` sovereign-act semantics, and does not pollute `aiwf.yaml`.

Two subverbs today:

- **`aiwf acknowledge illegal <sha>`** — exempt a historical commit from the FSM-history / provenance audit rules.
- **`aiwf acknowledge mistag <id>`** — accept an `area-mistag` warning as intentional cross-cutting work.

Both refuse a non-`human/` actor and an empty `--reason` at the gate — the judgment is the human's, recorded with a written rationale.

## aiwf acknowledge illegal

The `aiwf acknowledge illegal <sha>` verb is the retroactive sovereign-override mechanism for the `fsm-history-consistent` rule's `illegal-transition` subcode (and the other rules that consume the acknowledged-SHA set). It exists for historical commits that violate the per-kind FSM but cannot be cleanly fixed (squash-merges from the pre-rule era; force-pushed history; etc.).

### When to use

- `aiwf check` reports `fsm-history-consistent/illegal-transition` against a commit you've confirmed is intentional / unfixable (the typical case is a pre-rule squash-merge whose intermediate FSM steps were collapsed away).
- Pre-push is blocked by the error and rewriting history is wrong (shared trunk, force-push to main is forbidden, etc.).

### When NOT to use

- The illegal transition is fresh / fixable — re-route through `aiwf promote` or `aiwf cancel` (which only accept FSM-legal moves), or use `aiwf <verb> --force --reason "..."` at the time of the change.
- The transition is FSM-legal but lacks an `aiwf-verb:` trailer — that's the `manual-edit` subcode, cleared via `aiwf <verb> <id> --audit-only --reason "..."`.

### What to run

```bash
# Acknowledge a historical illegal commit with a written rationale.
aiwf acknowledge illegal f4ea7329 \
  --reason "pre-AC-2 era squash-merge from epic/E-21; intermediate FSM progression existed on the feature branch but was lost to the squash"

# An untrailered entity-edit commit (per-(SHA, entity) ack) needs --for-entity.
aiwf acknowledge illegal 6a1e70cc --for-entity ADR-0007 \
  --reason "post-E-0038 terminology refresh landed inline; should have used aiwf edit-body"
```

The verb refuses with a typed error when `--reason` is empty, `--actor` is not `human/...`, `<sha>` doesn't match the 7-40-hex shape, or `<sha>` is **neither** reachable from HEAD **nor** present in the local object database (the typo guard).

### What the commit looks like

```
aiwf acknowledge illegal <short-sha>

<your reason text>

aiwf-verb: acknowledge-illegal
aiwf-force-for: <historical-sha>
aiwf-actor: human/<name>
aiwf-reason: <text>
aiwf-entity: <id>           (only when --for-entity is supplied)
```

### Exemption semantics

The consuming rules walk HEAD's reachable history for `aiwf-force-for:` trailers and exempt findings whose offending commit appears in that set. The exemption is **DAG-scoped** (only trailers reachable from HEAD count, so a cherry-pick onto a branch lacking the original violation doesn't exempt it) and **per-SHA** (one ack covers every entity the historical SHA touched). The `aiwf-verb: acknowledge-illegal` trailer value is unchanged by the subverb regroup — the command path `acknowledge illegal` enumerates to the same string, so history validates with no shim.

## aiwf acknowledge mistag

The `aiwf acknowledge mistag <id>` verb records a sovereign acceptance that an entity's `area` tag and its commits' landing zone legitimately disagree — suppressing the `area-mistag` warning for that entity. Mistag fires when an entity's area-claimed work landed entirely in a *foreign* area's `paths:` territory; sometimes that is genuinely intentional (e.g. moving code into a shared area), not a mis-file.

### When to use

- `aiwf check` reports `area-mistag` for an entity whose cross-cutting work you've confirmed is deliberate.
- The right fix is *not* a re-tag — the work really does span areas. (If the tag is simply wrong, run `aiwf set-area <id> <member>` instead; the mistag then no longer fires, and there's nothing to acknowledge.)

### What to run

```bash
aiwf acknowledge mistag G-0301 \
  --reason "moving billing's auth into the shared platform lib; cross-cutting by design"
```

The verb refuses with a typed error when `--reason` is empty, `--actor` is not `human/...`, or `<id>` resolves to no entity in the tree (the typo guard — a composite `M-NNNN/AC-N` id rolls up to its milestone).

### What the commit looks like

```
aiwf acknowledge mistag <id>

<your reason text>

aiwf-verb: acknowledge-mistag
aiwf-entity: <id>
aiwf-actor: human/<name>
aiwf-reason: <text>
```

### Suppression semantics

The `area-mistag` rule walks HEAD's reachable history for `aiwf-verb: acknowledge-mistag` commits and exempts the entities they name (via `aiwf-entity`, canonicalized). The exemption is **per-entity** (not per-SHA): once acknowledged, that entity never fires `area-mistag` again, regardless of which commits its work lands in. `area-mistag` is warning-only and never escalates, so the acknowledge path — not a strictness bump — is the sanctioned escape valve for legitimate cross-cutting work.

## Why empty-commit + trailer (vs. aiwf.yaml entry)

- **Audit trail is git-native.** `aiwf history` already walks commits for `aiwf-verb:` events; acknowledgements fit naturally. A YAML entry would need a parallel surfacing mechanism.
- **Per-acknowledgement rationale.** Each ack carries its own `aiwf-reason` — a YAML allowlist would either drop the reason or replicate it redundantly.
- **Sovereign-act alignment.** Existing `--force --reason "..."` records the human in the commit's trailer block; acknowledge is the same shape.
- **No allowlist drift.** A YAML allowlist accumulates entries nobody knows are still needed; a commit-trailer set lives in history and grows monotonically with reasons attached.

## What these verbs can't do

- **Reverse themselves.** There's no companion "un-acknowledge" verb — the acts are one-way by deliberate design (the "What verb undoes this?" answer is *"you can't, and that's deliberate"*). To undo a mistag ack, re-tag with `aiwf set-area` so the finding no longer fires; an illegal ack is lived with or, in extremis, rewritten out of history.
- **Acknowledge an absent target.** `illegal` refuses a SHA reachable from neither HEAD nor the object DB; `mistag` refuses an id that resolves to no entity — both rather than silently recording a no-op ack.
- **Acknowledge across rule families.** `illegal` clears the FSM-history / provenance set; `mistag` clears `area-mistag`. Other findings (`forced-untrailered`, `manual-edit`, …) have their own resolution paths.

## Related

- **`aiwf acknowledge illegal`** — exempts historical commits from FSM-history / provenance audit rules.
- **`aiwf acknowledge mistag`** — accepts an `area-mistag` warning as intentional cross-cutting work; includes the regroup into the `aiwf acknowledge` subverb namespace.
- **`fsm-history-consistent`** — the check rule whose `illegal-transition` subcode `acknowledge illegal` exempts.
- **Skills policy ADR** — the per-verb-skill / topical-skill / allowlist judgment rule this topical skill satisfies.
