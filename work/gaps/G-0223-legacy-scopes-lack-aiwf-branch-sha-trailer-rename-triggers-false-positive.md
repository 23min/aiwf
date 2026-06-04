---
id: G-0223
title: Legacy scopes lack aiwf-branch-sha trailer; rename triggers false positive
status: open
prior_ids:
    - G-0221
discovered_in: M-0161
---
## What's missing

M-0161's AC-6 (closes [G-0206](G-0206-branchoracle-false-positive-on-branch-renames-after-authorize.md)) introduces `aiwf-branch-sha:` trailer + SHA-fallback resolution in `BranchOracle` so a `git branch -m oldname newname` rename is transparent to the `isolation-escape` rule. The fix works for authorize commits made AFTER AC-6 lands.

**Authorize commits made BEFORE AC-6 ("legacy scopes") lack the `aiwf-branch-sha:` trailer.** When the bound branch of such a scope is renamed AND the old name no longer resolves AND there are AI-actor commits on the renamed branch, AC-6's SHA-fallback path has no SHA to fall back to — the rule's name-only resolution finds the (now-unreachable) old name, falls back to AC-3's `isolation-escape-oracle-failure` advisory, but EVERY AI commit on the renamed branch still hits the false-positive path of the original G-0206 failure mode for the duration the scope stays open.

## When does this actually trigger?

ALL of these must hold:

1. There is an OPEN authorize scope (no `authorize-end` event has fired on it).
2. The authorize commit was emitted before AC-6 (no `aiwf-branch-sha:` trailer).
3. The bound branch has been renamed via `git branch -m` *after* AC-6 landed.
4. The old branch name no longer resolves to any current ritual branch.
5. There are AI-actor commits on the renamed branch within the pre-push range.

If any condition is false, no false positive — silent operation.

**Practical scope of the failure mode:**
- For an aiwf consumer who never renames ritual branches with open scopes: never triggers; AC-6 ships transparently.
- For consumers who rename mid-flight: every AI commit on the renamed branch shows the false positive until either (a) every AI commit gets `aiwf acknowledge-illegal <sha>`, or (b) the legacy scope is ended and re-authorized post-AC-6 (the new authorize commit carries `aiwf-branch-sha:` and rename transparency kicks in for future commits).

## Why parked

M-0161 chose path (a) — "downgrade closure language + file follow-up gap" — over path (c) (ship `aiwf scope rebind` as part of AC-6). The reviewer flagged path (c) as scope-balloon for AC-6 ("belongs in a separate AC"). The legacy false-positive class is documented in AC-6's body as a known limitation; this gap captures the architectural completion path.

The cadence matches the kernel's other rule-introductions:
- M-0107's `fsm-history-consistent` rule shipped at warning; legacy FSM-illegal commits use `aiwf acknowledge-illegal` rather than being auto-rewritten.
- M-0125 ratchet pattern: introduce, observe, tighten.

`aiwf scope rebind` would be the kernel's analogous "graceful migration verb" for legacy authorize scopes that need rename transparency without ending and re-authorizing.

## Proposed fix shape

A new verb `aiwf scope rebind <id> --to <new-branch> [--reason "..."]`:

- Records a follow-up commit on the authorize scope carrying:
  - `aiwf-verb: scope-rebind`
  - `aiwf-entity: <scope-target-id>`
  - `aiwf-actor: <committer>`
  - `aiwf-branch: <new-branch>` (the new ritual branch name)
  - `aiwf-branch-sha: <new-tip-sha>` (the new branch's tip SHA at rebind time)
  - `aiwf-prior-branch: <old-branch>` (the old name being replaced)

- The `isolation-escape` rule's scope-branch resolution reads `aiwf-branch-sha:` from the MOST RECENT rebind commit on the scope (if any), falling back to the original authorize commit's trailers (which AC-6 may or may not have).

- The rebind verb does NOT end the scope — it updates the binding metadata. The scope stays open at the FSM level; only the bound-branch identity refreshes.

This is the symmetric companion to AC-6's `aiwf-branch-sha:` trailer: AC-6 records SHA at authorize-open; `scope-rebind` records SHA at rebind-time. Together they cover the full lifecycle of a scope's bound branch.

## Test surface

When the verb lands:

- Scenario: open legacy scope → rename branch → `aiwf scope rebind` → AC-6 rule resolves via rebind's SHA → silent (false positive resolved without scope end-and-re-authorize)
- Scenario: open legacy scope → rename branch → AI commit on renamed branch → false positive fires → `aiwf scope rebind` → re-run check → silent (rebind eliminates false positive for subsequent and existing commits)
- Scenario: scope opened post-AC-6 (has `aiwf-branch-sha:`) → rebind → rebind's SHA OVERRIDES the original authorize SHA (rebind is the more recent record)
- Sabotage: removing the rebind-precedence in scope-branch resolution → rebind has no effect, AC-6's original path fires for legacy scopes

## Workaround

Until `aiwf scope rebind` lands, operators with legacy authorize scopes have two options when they need to rename a ritual branch:

1. **End-and-re-authorize**: `aiwf authorize <id> end` to close the scope, then `aiwf authorize <id> --to ai/<agent> --branch <new-name>` to re-open. The new authorize commit gets `aiwf-branch-sha:` per AC-6 and rename transparency kicks in for future commits. Doesn't help with already-landed AI commits on the renamed branch.

2. **Acknowledge-illegal sweep**: for each AI commit on the renamed branch that the rule false-positives on, `aiwf acknowledge-illegal <sha> --reason "legacy scope rename"`. Silences the rule on those specific commits; new commits made post-rename still need acknowledging until the scope ends or rebind lands.

Both workarounds are operator-discipline; neither is automated. `aiwf scope rebind` would replace both with a single commit per scope.

## Closing this gap

When the verb lands:
- `aiwf scope rebind` registered with Cobra completion + skill
- AC-6's body updated to remove the "legacy is documented carve-out" language and reference `scope rebind` as the upgrade path
- This gap promoted to `addressed` with `--by M-NNNN`

## Discovered in

M-0161 — during AC-6 contract review (reviewer pass on M-0161 9-AC body set, pre-implementation). The legacy-scope false-positive class was a known consequence of choosing path (a) over path (c); G-0223 captures the architectural completion path the deferred path (c) names.
