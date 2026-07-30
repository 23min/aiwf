---
id: E-0075
title: Structured-state verbs refuse to commit frontmatter they do not own
status: active
---
## Goal

Stop a mutating verb from committing frontmatter it does not own, so a
hand-edited field can no longer land under another verb's trailer and be
attributed to an act that did not make it.

Addresses G-0466 and G-0463.

## Context

Two distinct mechanisms produce this, and a guard that closes one leaves the
other open.

**Serializing verbs** read the entity from the loaded tree, re-serialize the
whole frontmatter, and commit. Nothing compares that frontmatter against HEAD
first, so a hand-edited field rides into the next verb's commit under that verb's
trailer. Measured: a gap's priority set to `high` through `aiwf set-priority`,
hand-edited to `low`, then `aiwf retitle` run — the retitle commit carried
`-priority: high / +priority: low` under `aiwf-verb: retitle`, and `aiwf history`
still shows `set-priority high` as the last priority act, so a reader concludes
`high` when it is `low`.

**Move-shaped verbs launder through a different path.** `rename`'s plan leads
with an `OpMove`, and the laundering enters through `gatherCommitOps` →
`addFile`, which commits the moved file's on-disk bytes verbatim and walks a
moved directory recursively — so an epic rename commits every nested entity's
on-disk bytes. The two mechanisms are not disjoint verb classes: the link-rewrite
pass re-serializes *any* entity whose body links to a moved path, including the
moved entity itself, and `retitle` builds both an `OpMove` and an `OpWrite`, so
it sits in both.

That nested case is the worst vector and it changes the required shape of the
fix. Measured: `tdd: none` hand-edited to `tdd: required` on a milestone — a
policy field that decides whether `acs-tdd-audit` fires — then `aiwf rename` run
on the *parent epic*. The change landed in a commit trailered
`aiwf-entity: <the epic>`, and `aiwf history <the milestone>` shows only its
creation, with no event for the change at all. The field moved, it is attributed
to a different entity, and the committed guarantee that `aiwf history <id>` reads
the log is wrong for the entity that actually changed.

**Two blocking rules are defeated, not one.**
`provenance-untrailered-entity-commit` skips any commit carrying a non-empty
`aiwf-verb:` trailer, and it reads changed *paths* — it never inspects
frontmatter — so which fields moved is invisible to it. And
`fsm-history-consistent/illegal-transition` is evaded when the laundering rides a
commit that *also* changes the file's path — measured: an illegal
terminal-to-open edit passes the check through `rename`, and is caught through
`set-priority`. Its walker skips a commit that both renames and changes status,
documented in `internal/check/fsm_history_walker.go` on the reasoning that "pure
renames don't change status, so no observation is lost on the typical path." This
defect is precisely what falsifies that premise. So the escape is confined to the
path-changing routes — `rename`, `retitle` when the slug re-derives, `reallocate`,
`archive`, `rewidth`, `move`. On the serializing routes a laundered `status:` is
already a hard error today, which narrows this rule's exposure without narrowing
the provenance rule's.

## Scope

Every route that commits frontmatter. Grouped by what a guard has to do about
them:

- **Single-entity field writes** — `promote`, `cancel`, `move`, `retitle`, `rename`,
  `reallocate`, `set-priority`, `set-area`, `milestone tdd`,
  `milestone depends-on`, `add ac`, and `edit-body --body-file` (G-0463's instance).
- **A second entity's frontmatter** — `promote <id> superseded --superseded-by <other>`
  writes `superseded_by` on the promote target and the reciprocal `supersedes` on
  `<other>`, the superseding ADR, so ownership spans two entities.
- **Multi-entity sweeps** — `rename-area` writes `area:` on every tagged entity plus
  `aiwf.yaml` in one commit; `rewidth --apply` rewrites `id:` tree-wide; `import`
  with `--on-collision update` rewrites existing entities; `archive` moves files.
  Each needs an explicit in-or-out call rather than inheriting the single-entity
  answer.
- **Nested paths under a moved directory** — not only `rename`. `retitle`,
  `reallocate`, `archive` and `rewidth --apply` all emit directory `OpMove`s
  through the same helper, and each carries the identical vector; reproduced for
  `reallocate`. No verb names the nested entities, so a guard keyed on the verb's
  target cannot see them.

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

## Out of scope

- **A HEAD conjunct inside each same-state NoOp guard.** The reason is that the
  comparison belongs at one shared precondition ahead of the guards, not duplicated
  into each of them — the first decision below settles where. It is *not* that the
  converging path is harmless, which measurement refutes: with HEAD at
  `priority: high` and the working copy hand-edited to `low`, asking for `low`
  reports "already set to `low`; nothing to change" — false about the record, and
  the operator's requested mutation is silently dropped — while asking for `high`
  commits a tree byte-identical to its parent, an empty-diff commit of the class
  M-0281 existed to eliminate. A loaded-only comparison produces a false negative
  and a false positive.
- **A check rule for laundering already in history.** A genuine companion rather
  than an alternative: it catches what is already committed, which a precondition
  cannot, and it is the only thing that catches the nested case if the precondition
  ends up entity-scoped. It is G-0466's third option, tracked as G-0480, so
  closing G-0466 under this epic no longer orphans it.
- **The FSM walker's rename-plus-status blind spot.** This epic's precondition would
  incidentally mask it, but the rule stays wrong for any other route to such a
  commit. Tracked as G-0475.
- **Merge commits.** A verb cannot produce one: `checkNoGitOperationInProgress`
  refuses while a merge, cherry-pick, revert or rebase is in progress. Note the
  untrailered audit *skips* multi-parent non-squash merges, so a merge carrying
  laundered frontmatter goes unaudited — that is a reason the after-the-fact rule
  above matters, not a reason merges are safe.
- **The same-state convergence remainder** — G-0458, G-0459, G-0460. Same layer,
  different axis, tracked as E-0074. The coupling that is not separable is the
  first decision below, which E-0074 waits on.

## Constraints

- The guard has to reach nested paths under a moved directory. No verb names
  those entities, so a guard keyed on the verb's target cannot see them, and the
  nested case is the worst vector.
- `checkStagedConflict` is the precedent: its message shape, and its position in
  the one function every write route passes through. A second, differently-shaped
  refusal for the unstaged case would leave the operator with two messages for
  one condition.
- The comparison degrades when there is no HEAD version. `edit-body` already
  contains both answers — bless mode refuses a never-committed entity, explicit
  mode proceeds — so the raw material for the call sits in one file. This is an
  implementation note, not an open decision.
- Whatever lands must not be read as fixing the FSM walker's blind spot. The
  precondition would incidentally mask it; the rule stays wrong for any other
  route to such a commit, and G-0475 stays open on its own terms.
- G-0480 carries the after-the-fact check rule and stays open on its own terms.
  Promoting G-0466 to `addressed` does not close it.
- An `internal/policies/` invariant so the discipline cannot rot as new verbs
  land, mirroring what M-0281 did for same-state convergence.

## Success criteria

<!-- Observable at epic close. Deliberately phrased so they hold under either
     answer to the refuse-or-warn decision: what goes away is the *silence*. -->

- [ ] A structured-state verb run against an entity whose frontmatter differs
      from HEAD in the unstaged working copy never commits that difference
      silently — the operator is refused or told, per the ADR's decision, before
      the commit lands.
- [ ] The measured nested reproduction no longer succeeds silently: `tdd:`
      hand-edited on a milestone, then `aiwf rename` on the parent epic, no
      longer produces a commit that attributes the change to the epic while
      `aiwf history` on the milestone shows no event for it.
- [ ] Every route listed in *In scope* is covered by the guard or carries a
      recorded reason for being exempt — including each multi-entity sweep,
      which gets an explicit call rather than inheriting the single-entity one.
- [ ] An `internal/policies/` invariant fails when a new frontmatter-writing
      route is added that does not pass through the precondition seam.
- [ ] Neither failure direction of a loaded-only comparison survives: a verb no
      longer reports "already set; nothing to change" while HEAD disagrees, and
      no longer commits a tree byte-identical to its parent when HEAD's value is
      the one requested.
- [ ] An ADR records the four decisions listed in *Open questions*.
- [ ] The after-the-fact laundering check rule exists as its own entity — G-0480.
- [ ] G-0466 and G-0463 are promoted to `addressed`.

## Open questions

The first four are the decisions to land before any code. They are the epic's
substance, not scheduling detail.

| Question | Blocking? | Resolution path |
|---|---|---|
| Where the precondition runs, relative to the same-state NoOp comparison. In the verb prelude, before the same-state check, a NoOp guard can never be reached with HEAD-divergent frontmatter. At `verb.Apply` — the seam covering every route above — it catches the empty-diff case, which does produce a plan, but not the false NoOp: that guard returns from the verb body before any plan exists, so `Apply` is never reached | yes | ADR, first. Shared with E-0074, which waits on it |
| Entity-scoped or committed-path-scoped. The nested case forces this — a guard comparing only the named entity's frontmatter misses a nested milestone's | yes | ADR |
| Refuse or warn. Weigh against the illegal-FSM-transition escape, not against a laundered `priority`: refusing blocks a workflow that currently succeeds, permitting one lets a blocking check be bypassed | yes | ADR |
| Whether an escape hatch exists, and what it costs. Not a question of reusing an existing lever — of the routes above, only `promote` and `cancel` expose `--force`. Adding it to the rest is a surface expansion with a completion-drift obligation, and the flag already carries several distinct meanings across the CLI | yes | ADR |
| Whether each multi-entity sweep — `rename-area`, `rewidth --apply`, `import --on-collision update`, `archive` — is in or out | yes | explicit per-sweep call at milestone-planning; does not inherit the single-entity answer |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| An entity-scoped guard ships and silently misses the nested case, which is the worst vector and the one that defeats a blocking check | high | scope is the second decision, made explicitly and before code; the after-the-fact check rule is the only backstop if the precondition ends up entity-scoped, so it is filed regardless |
| Refusing blocks an operator workflow that currently succeeds, and the friction lands on every verb rather than the rare laundering case | medium | the third decision weighs it against the FSM escape; the fourth covers the escape hatch. `checkStagedConflict` already imposes this shape for staged edits, so the cost is bounded by observed practice |
| The precondition incidentally masks the FSM walker's blind spot, so the walker looks fixed while staying wrong for any other route | medium | G-0475 tracked separately, stated in *Out of scope*, and stays open on its own terms |
| G-0466 is closed under this epic and the after-the-fact rule — its third option — is orphaned with it | medium | filed as G-0480 at milestone-planning, ahead of any promotion, so the orphaning window never opens |
| A merge commit carrying laundered frontmatter goes unaudited, since the untrailered audit skips multi-parent non-squash merges | low | verbs cannot produce a merge commit; the after-the-fact rule is what covers this route |

## Milestones

In execution order. The chain is linear rather than partly parallel: the seam has
to exist before routes reach it, and the invariant would fail against unrouted
sweeps if it landed first.

- **M-0282** — settle the seam, scope, verdict and escape hatch in one ADR.
  Nothing else starts first, and E-0074 waits on the first of its decisions.
  `tdd: none`.
- **M-0283** — the shared precondition at the chosen seam, covering the
  single-entity routes. Depends on M-0282. `tdd: required`.
- **M-0284** — the nested-path vector and the multi-entity sweeps, each with its
  explicit in-or-out call. Depends on M-0283. `tdd: required`.
- **M-0285** — an `internal/policies/` invariant so a newly-added write route
  cannot bypass the seam, mirroring what M-0281 did for same-state convergence.
  Depends on M-0284. `tdd: required`.

## References

- G-0466 — a verb commits frontmatter it does not own (the laundering; `high`)
- G-0463 — `edit-body --body-file` instance of the same write-scope question
- G-0475 — the FSM history walker skips a commit that both renames and changes status
- G-0480 — after-the-fact detection of laundering already in history
- ADR-0036 — same-status FSM transitions converge to NoOp, not refusal
- E-0074 — same-state convergence remainder; waits on this epic's first decision
- `internal/verb/apply.go` — `checkStagedConflict`, the precedent to extend
- `internal/check/fsm_history_walker.go` — the walker whose premise this defect falsifies
- CLAUDE.md §"Same-state convergence — resolve, then converge"
