---
id: E-0075
title: Structured-state verbs refuse to commit frontmatter they do not own
status: proposed
---
## Goal

Stop a mutating verb from committing frontmatter it does not own.

Two distinct mechanisms produce this, and a guard that closes one leaves the other
open.

**Serializing verbs** read the entity from the loaded tree, re-serialize the whole
frontmatter, and commit. Nothing compares that frontmatter against HEAD first, so a
hand-edited field rides into the next verb's commit under that verb's trailer.
Measured: a gap's priority set to `high` through `aiwf set-priority`, hand-edited to
`low`, then `aiwf retitle` run — the retitle commit carried
`-priority: high / +priority: low` under `aiwf-verb: retitle`, and `aiwf history`
still shows `set-priority high` as the last priority act, so a reader concludes
`high` when it is `low`.

**Move-shaped verbs never re-serialize at all.** `rename`'s plan is an `OpMove`
plus link rewrites on *other* entities; the renamed entity is never re-serialized.
The laundering enters through `gatherCommitOps` → `addFile`, which commits the moved
file's on-disk bytes verbatim, and which walks a moved directory recursively. So an
epic rename commits every nested entity's on-disk bytes.

That nested case is the worst vector and it changes the required shape of the fix.
Measured: `tdd: none` hand-edited to `tdd: required` on a milestone — a policy field
that decides whether `acs-tdd-audit` fires — then `aiwf rename` run on the *parent
epic*. The change landed in a commit trailered `aiwf-entity: <the epic>`, and
`aiwf history <the milestone>` shows only its creation, with no event for the change
at all. The field moved, it is attributed to a different entity, and the committed
guarantee that `aiwf history <id>` reads the log is wrong for the entity that
actually changed.

**Two blocking rules are defeated, not one.** `provenance-untrailered-entity-commit`
skips any commit carrying a non-empty `aiwf-verb:` trailer, and it reads changed
*paths* — it never inspects frontmatter — so which fields moved is invisible to it.
And `fsm-history-consistent/illegal-transition` is evaded whenever the laundered
field is `status:`: its walker skips a commit that both renames and changes status,
documented in `internal/check/fsm_history_walker.go` on the reasoning that "pure
renames don't change status, so no observation is lost on the typical path." This
defect is precisely what falsifies that premise. An illegal terminal-to-open
transition routed through a rename passes a blocking check that catches the same
edit committed by hand.

Addresses G-0466 and G-0463.

## Scope

Every route that commits frontmatter. Grouped by what a guard has to do about them:

- **Single-entity field writes** — `promote`, `cancel`, `move`, `retitle`, `rename`,
  `reallocate`, `set-priority`, `set-area`, `milestone tdd`,
  `milestone depends-on`, `add ac`, and `edit-body --body-file` (G-0463's instance).
- **A second entity's frontmatter** — `promote --superseded-by` writes `supersedes`
  on the ADR being superseded, so ownership spans two entities.
- **Multi-entity sweeps** — `rename-area` writes `area:` on every tagged entity plus
  `aiwf.yaml` in one commit; `rewidth --apply` rewrites `id:` tree-wide; `import`
  with `--on-collision update` rewrites existing entities; `archive` moves files.
  Each needs an explicit in-or-out call rather than inheriting the single-entity
  answer.
- **Nested paths under a moved directory** — the case above. No verb names these
  entities, so a guard keyed on the verb's target cannot see them.

Deliberately out, so nobody re-derives it: `authorize`, `acknowledge illegal`,
`acknowledge mistag`, and `promote` / `cancel --audit-only` write no files — they
land empty-diff trailer-only commits, so there is nothing to launder.

**The precedent to extend is `checkStagedConflict`**, not `edit-body`'s bless-mode
message. `checkStagedConflict` already refuses this exact operator sequence when the
edit is *staged*, naming the overlapping path and telling the operator to
`git restore --staged` or `git stash` and re-run. It has the right message shape and
it sits in the one function every write route passes through. That also settles
whether the comparison is against HEAD or the index: index overlap already refuses,
so the open hole is specifically the **unstaged** working-copy edit. Bless mode's
message points the operator at the structured-state verbs, which is the wrong
direction for someone already running one, and it names four of the routes above.

### Decisions to land first

1. **Where the precondition runs, relative to the same-state NoOp comparison.** This
   is the epic-defining one. Run it in the verb prelude, before the same-state
   check, and a NoOp guard can never be reached with HEAD-divergent frontmatter.
   Run it at `verb.Apply` — the seam that covers all the routes above — and it is
   too late: the NoOp guards return from the verb body before any plan exists, so
   `Apply` is never called and the defects in Out of scope below stay live. The
   choice is shared with E-0074, which adds convergence guards to more verbs, and
   should be settled before either epic writes code.
2. **Entity-scoped or committed-path-scoped.** The nested case forces this. A guard
   comparing only the named entity's frontmatter misses a nested milestone's.
3. **Refuse or warn.** Weigh this against the illegal-FSM-transition escape, not
   against a laundered `priority`. Refusing blocks a workflow that currently
   succeeds; permitting one lets a blocking check be bypassed.
4. **Whether an escape hatch exists, and what it costs.** Not a question of reusing
   an existing lever: only `promote` and `cancel` expose `--force` at all. Adding it
   to the rest is a surface expansion with a completion-drift obligation, and
   `--force` already carries two meanings in this CLI — sovereign FSM bypass when
   paired with `--reason`, and force-replace on `contract bind`. A third would need
   arguing.

The never-committed-entity case is an implementation note rather than a decision:
the comparison degrades when there is no HEAD version. `edit-body` already contains
both answers — bless mode refuses such an entity, explicit mode proceeds — so the
raw material for the call is in one file.

### Shape

Roughly: an ADR settling decisions 1-4; one milestone for the shared precondition at
the chosen seam covering single-entity routes; one for the multi-entity sweeps and
the nested-path case; and an `internal/policies/` invariant so the discipline cannot
rot as new verbs land, mirroring what M-0281 did for same-state convergence.

## Out of scope

- **A HEAD conjunct inside each same-state NoOp guard.** The reason is that the
  comparison belongs at one shared precondition ahead of the guards, not duplicated
  into eleven of them — decision 1 settles where. It is *not* that the converging
  path is harmless, which measurement refutes: with HEAD at `priority: high` and the
  working copy hand-edited to `low`, asking for `low` reports "already set to
  `low`; nothing to change" — false about the record, and the operator's requested
  mutation is silently dropped — while asking for `high` commits a tree byte-identical
  to its parent, an empty-diff commit of the class M-0281 existed to eliminate. A
  loaded-only comparison produces a false negative and a false positive.
- **A check rule for laundering already in history.** A genuine companion rather
  than an alternative: it catches what is already committed, which a precondition
  cannot, and it is the only thing that catches the nested case if the precondition
  ends up entity-scoped.
- **The FSM walker's rename-plus-status blind spot.** This epic's precondition would
  incidentally mask it, but the rule stays wrong for any other route to such a
  commit. Filed separately.
- **Merge commits.** Already handled: `checkNoGitOperationInProgress` refuses any
  verb while a merge, cherry-pick, revert or rebase is in progress, and the
  untrailered audit skips multi-parent non-squash merges.
- **The same-state convergence remainder** — G-0458, G-0459, G-0460. Same layer,
  different axis, tracked as E-0074.
