---
id: M-0312
title: Re-point the skill-edit backstop from content reference to provenance
status: in_progress
parent: E-0087
tdd: required
acs:
    - id: AC-1
      title: An untrailered edit to a watched shipped surface is refused
      status: met
      tdd_phase: done
    - id: AC-2
      title: A trailered, entity-owned edit passes with no policy test naming its path
      status: met
      tdd_phase: done
    - id: AC-3
      title: A trailered edit naming an unresolvable entity is refused
      status: met
      tdd_phase: done
    - id: AC-4
      title: CLAUDE.md no longer states that a SKILL.md edit requires a structural test
      status: met
      tdd_phase: done
---
## Goal

Replace the skill-edit backstop's content-reference predicate with a provenance one, so
an edit to a watched shipped surface must ride a trailered, entity-owned commit instead
of being accompanied by a policy test that happens to name its path.

## Context

M-0196 shipped the current predicate under E-0048, closing G-0220. That gap's complaint
was mostly about provenance — nothing owned the edit, nothing recorded why it happened —
and the predicate that closed it asks a content question instead: does the edited path
appear as a string literal somewhere in the concatenated policy test sources. D-0071
settles the correction and records why the stronger content variants were rejected
rather than adopted.

This milestone lands before its sibling because the current predicate constrains how the
assertion corpus can be removed. While it stands, each watched skill's path must survive
as a literal somewhere in the policy sources, which limits deletion to whole functions
rather than whole files.

## Acceptance criteria

### AC-1 — An untrailered edit to a watched shipped surface is refused

A commit that adds or modifies a `SKILL.md` under the watched set, carrying no
`aiwf-entity` trailer, produces a finding that names the path and says the edit has no
owning entity. This is the arm G-0220 recorded as silent: the edit it describes was an
ordinary `git commit` against a shipped surface, and nothing fired.

The required set is `aiwf-entity` alone. `aiwf-verb` is excluded because a ritual
`SKILL.md` is source, not an entity file, and no aiwf verb commits it: the closed set
`trailer-verb-unknown` enforces — the Cobra verb tree plus the ritual stamps — carries no
value meaning "I edited a shipped surface", so requiring one would mandate the fabricated
trailer G-0150 closed. `aiwf-actor` is excluded because "who ran the verb" is undefined
where no verb ran, git's author field already carries the answer, and an `ai/` actor with
no scope trailers raises `provenance-no-active-scope` at error severity — friction whose
cheapest escape is a false `human/` actor, which is worse provenance than none.

Fixture: a synthetic commit touching a watched surface with a bare Conventional Commits
subject and no trailers. The policy fires.

### AC-2 — A trailered, entity-owned edit passes with no policy test naming its path

The same edit, riding a commit whose `aiwf-entity` trailer is present and resolves to a
real entity, produces no finding — with **zero** policy tests referencing
the edited path anywhere in `internal/policies`.

This is the inversion that retires the content mandate, and it fails today: the current
predicate refuses precisely this case. The fixture must assert silence under the
no-referencing-test condition specifically, since a fixture that happens to run in a tree
where some unrelated test names the path would pass for the wrong reason.

### AC-3 — A trailered edit naming an unresolvable entity is refused

The trailer is present, but `aiwf-entity` names an id that resolves to nothing — a typo,
a fabricated id, an entity never created. The policy refuses.

Provenance that points nowhere is not provenance, and converging here would let any
edit satisfy the gate by inventing an id. This mirrors the kernel's own rule that a verb
resolves its arguments before asking whether the request is already satisfied.

Resolution is the whole of the requirement; the entity's status is not consulted. The
Design notes carry why.

### AC-4 — CLAUDE.md no longer states that a SKILL.md edit requires a structural test

No section of `CLAUDE.md` asserts that an embedded-rituals `SKILL.md` edit must land
alongside a referencing structural test under `internal/policies/`. The check is an
**absence** assertion over the named sections that carry the rule today — the
ritual-authoring section and the enforcement list.

An absence assertion is deliberate, and it is the reason this AC is not the pattern
D-0070 retires. It is a ban: it costs once, there is no wording to maintain as the prose
around it evolves, and it cannot be satisfied accidentally. It fires if the mandate is
ever re-added.

The positive half — that the surviving prose states the provenance rule correctly and
readably — is **not** asserted. That is held at review, per D-0070's disposition for
content correctness. Asserting it would mint exactly the rotting check this epic exists
to remove.

## Constraints

- The watched surface set does not change. This milestone alters what an edit must
  prove, not which edits are watched.
- The predicate must be decidable without judgment — trailers and entity resolution are
  mechanical; "is this edit well-documented" is not.
- The replacement ships with firing fixtures on both arms: an unowned edit is refused, and
  an entity-owned edit passes with no accompanying test.
- Every surface describing the current mandate is updated in this milestone, not left to
  the sibling. A gate whose documentation still states the old rule is half-landed.

## Design notes

- D-0071 carries the decision, the growth-curve evidence, and the rejected alternatives
  (retire outright; narrow to the evidence-backed classes; strengthen to
  section-level reference).
- `provenance-untrailered-entity-commit` already enforces the analogous property over
  entity files. Reuse that shape rather than inventing a second notion of provenance.
- The policy remains an aiwf-repo invariant and stays inert in a consumer tree.
- **The predicate is commit-scoped, not working-tree-scoped.** Provenance is a property a
  commit carries, so an uncommitted edit has none and firing on one would state a fault
  the operator cannot clear. This retires the working-tree arm the content predicate
  needed, where an operator could satisfy the gate before committing by writing a test.
- **The entity's status is not consulted** — E-0087's first open question, settled here.
  Two of the gate's three invocations resolve their base to `git merge-base origin/main
  HEAD` (local `make ci`/`make coverage-gate`, and CI on a pull request), so every skill
  edit on a branch is re-audited on every run for the branch's whole life; only CI's
  push event is incremental. A non-terminal requirement would therefore turn a week-old
  green commit red the moment its owning milestone promoted to `done`, with nothing about
  the commit having changed — the rot class this epic exists to remove — and it would
  refuse the legitimate post-promote wrap edit G-0119 describes. It also fails to prevent
  what it targets: a rubber stamp names a currently-active entity just as easily as a
  closed one. Whether an attribution is *apt* is held at review, per D-0070 and D-0071.
- The policy is renamed with its predicate — `skill-edit-provenance-backstop`. The old id
  names the retired content question, and its violation Detail told operators to add a
  structural test, which is no longer the fix.

## Surfaces touched

- `internal/policies/skill_edit_structural_test_backstop.go` → `skill_edit_provenance_backstop.go`
- `.github/workflows/go.yml` and `Makefile` — the coverage-gate run-pattern names the
  policy test by name
- CLAUDE.md — the ritual-authoring and enforcement sections
- Any shipped guidance that restates the content mandate

## Out of scope

- Deleting the existing prose-assertion corpus. That is the sibling milestone's
  deliverable, and it is sequenced after this one.
- G-0504's separate complaint that `aiwf doctor` byte-checks only verb skills while
  ritual and guidance drift read as healthy.
- Extending provenance enforcement to shippable surfaces the current backstop does not
  already watch.

## Dependencies

- D-0071, accepted.

## Work log

### AC-1..AC-4 — the predicate swap

All four criteria landed in one commit; they are facets of one replacement and each is
broken in isolation — deleting the old policy is what lets AC-2 pass, and the CLAUDE.md
edit is AC-4. · commit 667c8e0fc · `internal/policies` suite green, diff-scoped
branch-coverage audit and firing-fixture meta-gate clean.

Five vacuity probes were run against the new predicate, each red on break and green on
restore: the missing-trailer arm disabled, the unresolvable arm disabled, the resolver
forced always-true, the composite rollup removed, and owned edits refused anyway. The
first found a real defect. With the missing-trailer arm disabled, an empty id fell
through to the unresolvable arm — `resolves("")` is false too — so a violation still
fired on the same path and a file-only assertion could not tell the arms apart. The
second arm now requires a named id, and the Detail assertions discriminate.

AC-4's absence assertion was written after the CLAUDE.md edit rather than before it, so
its red was never observed live. It was established afterwards instead: run against
CLAUDE.md at `ffbd477e5`, every retired signature the ban names fires in at least one of
the two sections, and the assertion passes against the current file.

### Baseline carried in

`TestGuidance_WithinLineBudget` was already failing when this milestone started — the
guidance fragment reached 131 lines against a 120-line budget at `83e85fbaa`, a commit
from a concurrent session on `main`. It is untouched by this work, in a package this
milestone does not touch, and remains red. The budget's own comment sanctions raising the
ceiling rather than compressing the rule, but which lines earn their place belongs to
whoever wrote them.

## Deferrals

- G-0601 — `aiwf history <id>` renders only commits carrying `aiwf-verb`, so a skill-edit
  commit carrying `aiwf-entity` alone satisfies this gate while staying invisible in the
  history projection. Auditability is the point of provenance, so the gap is real; adding
  `aiwf-verb` is what D-0071 rejected, so the fix is not obvious.
- G-0317 asks for the strengthening D-0071 explicitly rejected, against a predicate this
  milestone deletes; closed `wontfix` alongside this work.
- G-0580 stays open. The watched surface set is out of this milestone's scope, and its
  body still describes the retired predicate.
- G-0602 — a merge that resolves a conflict by writing new content into a watched
  `SKILL.md` introduces content no examined commit carries, so the gate is silent on it.
- G-0603 — nothing catches a missing trailer at composition time, when the repair is an
  amend rather than a rebase. A `commit-msg` hook could.

## Reviewer notes

Two independent fresh-context reviewers ran over the full change-set: a code-quality lens
and a design-quality lens on the predicate itself. The design lens returned KEEP against
its five obligations. Both returned findings; the ones that changed the code are below,
each fixed on this branch and each pinned by a test that fails without the fix.

- The violation's `Policy` field was set from a named constant. The firing-fixture
  meta-gate matches a string literal, so the policy had dropped out of its inventory and
  nothing proved it could fire — the regression class that gate exists to catch,
  introduced by this milestone. Restoring the literal returns it to the inventory.
- `--diff-filter=AM` let a rename escape. Git reports a sufficiently similar rename as one
  R entry, so a commit that moved a skill and rewrote part of it passed with no owner
  named. The two reviewers disagreed here and the disagreement was a threshold artifact:
  below git's similarity cutoff a move degrades to delete-plus-add and is caught, above it
  is not. The filter now admits R, and the fixture is deliberately over the cutoff.
- An unparseable owning entity read as a missing one, because the loader records a file it
  cannot parse as a stub rather than an entity. That let an edit to an unrelated file turn
  a landed commit red while advising the operator to name an entity that already exists —
  a drift the design's own non-drift obligation forbids.
- The `prior_ids` arm had no test, and it is the mechanism that keeps older commit
  trailers resolving across `aiwf reallocate`.
- Git escapes a non-ASCII path by default, and the escaped form silently missed the
  `/SKILL.md` suffix test, so such an edit left the watched set entirely.
- The resolver re-inlined `tree.ResolveByCurrentOrPriorID`, which is byte-identical.

Declined, with reasons: the `id != ""` conjunct in the second arm is redundant against
case order and is kept deliberately, because stating each arm's condition in full is what
keeps the two arms independent. The `"structural test"` signature in AC-4's ban is generic
and could catch unrelated prose in either named section; that is the correct trade for a
ban, where a false positive costs one reading and a false negative costs the rule.

Two questions the reviews raised are not resolved here because they are decisions rather
than defects: D-0071's Decision section says the edit must carry "aiwf's verb trailers",
which is not what shipped, and the replacement mandate carries no named retirement
trigger — the bookkeeping D-0071 itself demands of a mandate.
