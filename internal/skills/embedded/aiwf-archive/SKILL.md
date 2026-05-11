---
name: aiwf-archive
description: Use when terminal-status entities have accumulated in the active tree and the operator wants to sweep them into per-kind `archive/` subdirs, or when `aiwf check` reports `archive-sweep-pending`. Explains dry-run vs `--apply`, the no-reverse rule, the `archive.sweep_threshold` knob, merge edge cases, and the per-kind storage layout.
---

# aiwf-archive

The `aiwf archive` verb sweeps terminal-status entities into per-kind `archive/` subdirectories so the active tree reflects what's currently in-flight. Movement is **decoupled from FSM promotion** — `aiwf promote` and `aiwf cancel` flip status only; this verb performs the structural projection later, as a single commit per invocation.

The kernel principle the verb embodies: location is a redundant projection of status. The decoupling buys (1) a grace period after close for inspection and wrap rituals, and (2) one-purpose promotion verbs with no file-move side effects to test. The cost is bounded drift (the period between promotion and the next sweep); the drift is policed via the `archive-sweep-pending` advisory finding and the `archive.sweep_threshold` knob below.

See [ADR-0004](../../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) for the full design rationale.

## When to use

- `aiwf check` reports `archive-sweep-pending` and the operator wants the tree back to convergence.
- The active directory listing (a plain `ls work/gaps/`, GitHub tree, or IDE file pane) is cluttered with terminal entries.
- A milestone or epic just closed and the operator is doing wrap rituals — running a dry-run now previews what the next sweep will move.
- First-time migration on a pre-ADR-0004 tree: the same verb covers the bulk first sweep and every recurring small sweep that follows.

## What to run

```bash
# Preview the planned moves (default behavior — no commit, read-only).
aiwf archive

# Same thing, explicit form.
aiwf archive --dry-run

# Commit the sweep as a single git commit. The trailer is
# `aiwf-verb: archive` (no `aiwf-entity:` trailer — multi-entity
# sweeps are a special case in the trailer-keys policy).
aiwf archive --apply

# Scope the sweep to one kind. Useful when one kind is the volume
# offender (typically gaps or findings).
aiwf archive --apply --kind gap
```

**Dry-run is the default.** The verb prints the planned moves and exits without touching the tree. Re-run with `--apply` to commit. This is the single-flag-flip safety pattern that keeps the verb hard to misuse — the destructive shape requires a deliberate flag, not the default.

**Idempotent.** Re-running on a clean tree produces no commit and exits 0. There is no "force a sweep" mode; if the tree is already converged, there is nothing to do.

## Reversal — there is none

**You don't reverse the sweep, deliberately.** Per ADR-0004 §"Reversal — what verb undoes archive?", the FSM is one-directional and archive is the structural projection of FSM-terminality. The kernel does not provide an "aiwf reactivate" verb, an "un-archive" verb, or any reverse-sweep mode.

The canonical pattern when a closed entity needs revisiting is to **file a new entity that references the archived one**. `Resolves: G-0018` from a new gap remains valid because the loader resolves ids across both active and archive directories — references stay live indefinitely.

If a contributor hand-edits frontmatter to take a status off-terminal on an already-archived file, `aiwf check` fires `archived-entity-not-terminal` (blocking). The remediation is to revert the hand-edit, not to relocate the file.

## Drift control

The tree is never strictly in convergence between promotion and the next sweep. Three layers bound the drift; the threshold knob below is the operator-tunable one.

**`archive.sweep_threshold` (the knob this skill exists to document).** Set in `aiwf.yaml`:

```yaml
archive:
  sweep_threshold: 5
```

When the active-tree pending-sweep count *exceeds* the configured value, `aiwf check` escalates `archive-sweep-pending` from a warning to an error and the pre-push hook blocks the push. At-or-below the threshold the finding stays advisory.

- **Default: unset.** No threshold; `archive-sweep-pending` stays advisory regardless of count. Permissive by default — teams that don't want the kernel to nag don't have to opt out.
- **Strictest setting: `sweep_threshold: 0`.** Any single pending sweep blocks. Use when the consumer prefers a strictly-converged tree.
- **Common pattern: `sweep_threshold: 20` or similar.** Allows the natural grace period for inspection without letting the backlog grow indefinitely.

The escalated message names both the count and the configured threshold so the human reading the failed push sees the magnitude of the breach and the policy they crossed.

## Merge edge cases

The decoupled model creates one merge-conflict shape worth knowing.

**Rename + modify.** Branch A archives `G-0018` (renames the file from `work/gaps/G-0018-...md` to `work/gaps/archive/G-0018-...md`). Branch B edits `G-0018` in place. When the branches merge, git's rename detection usually handles this cleanly — the edit goes to the renamed path. Occasionally it surfaces as a "rename+modify" conflict; the resolution is mechanical: take the rename, take the edit. Standard kernel pattern of "merge, run check, fix findings" handles the rest.

**Cross-references in body prose.** Body-prose references that use file paths (rather than ids) become stale when the target archives. Standard kernel discipline prefers id-form references; G-0091 tracks the work item for a preventive check rule.

## Per-kind storage layout

| Kind | Active location | Archive location | Trigger |
|---|---|---|---|
| Epic | `work/epics/<epic>/` (directory) | `work/epics/archive/<epic>/` (whole subtree) | terminal status (`done`, `cancelled`) |
| Milestone | `work/epics/<epic>/M-NNNN-<slug>.md` | rides with parent epic — does **not** archive independently | n/a |
| Contract | `work/contracts/<contract>/` | `work/contracts/archive/<contract>/` (whole subtree) | terminal status (`retired`, `rejected`) |
| Gap | `work/gaps/G-NNNN-<slug>.md` | `work/gaps/archive/G-NNNN-<slug>.md` | terminal status (`addressed`, `wontfix`) |
| Decision | `work/decisions/D-NNNN-<slug>.md` | `work/decisions/archive/D-NNNN-<slug>.md` | terminal status (`superseded`, `rejected`) |
| ADR | `docs/adr/ADR-NNNN-<slug>.md` | `docs/adr/archive/ADR-NNNN-<slug>.md` | terminal status (`superseded`, `rejected`) |

Milestones don't archive independently because they live as flat files inside their parent epic's directory. A `done` milestone under an `active` epic stays put until the epic itself archives, at which point the whole subtree moves in a single rename.

`internal/entity/transition.go::IsTerminal` is the source of truth for which statuses are terminal per kind. The verb consults it directly; the table above mirrors the rule for reference.

## Don't

- **Don't hand-move files into `archive/`.** The verb's commit carries the `aiwf-verb: archive` trailer so `aiwf history` recognizes the move. A hand-`mv` leaves an untrailered commit that the provenance audit flags.
- **Don't try to "un-archive" by editing frontmatter.** Status is the source of truth; flipping a terminal status off-terminal on a file under `archive/` produces an `archived-entity-not-terminal` finding. The remediation is to revert the hand-edit and file a new entity referencing the archived one.
- **Don't sweep before wrap rituals.** Just-closed entities benefit from the grace period — running wrap skills, browsing `aiwf show <id>` paths, and skimming the dust just settled all work most naturally when the entity is still at its active path.
- **Don't bypass the dry-run preview on a bulk migration.** The first sweep on a pre-ADR-0004 tree typically moves dozens or hundreds of files. Read the dry-run output, confirm the counts, then `--apply`.
