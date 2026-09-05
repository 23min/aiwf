---
id: M-0329
title: An entity body that omits a required section is refused at the write
status: in_progress
parent: E-0091
tdd: required
acs:
    - id: AC-1
      title: edit-body --body-file refuses a body that omits a required section
      status: open
      tdd_phase: done
    - id: AC-2
      title: The add-time gate refuses an omitted required heading, not only an empty one
      status: open
    - id: AC-3
      title: A blessed body that drops a required section HEAD carried is refused
      status: open
    - id: AC-4
      title: One absence predicate serves every rule that asks whether a section is there
      status: open
---

## Goal

Make a body that omits a section its kind requires impossible to write, at the
three seams where one is written. Nothing enforces it today: `entity-body-empty`
reports a required section present and empty and skips one absent outright, and
`aiwf edit-body` consults nothing at all.

## Closes

- G-0571 — no surface enforces that an entity body carries its kind's required
  sections.

## Context

The set is called required on five surfaces and no mechanism makes it true. The
sharpest consequence is at `aiwf add`: handed a body whose required section is
present and empty, the gate refuses and tells the operator `aiwf check` will
block until it is filled. An operator can satisfy that refusal by deleting the
heading instead, and then neither the gate nor the check says anything. The
stricter body is the one that is harder to land.

Measured on this tree: 55 live entities omit at least one required section, 109
omissions in all — 30 gaps, 24 decisions, and one epic missing `## Out of scope`.
They concentrate in the born-complete kinds, which have no reachable scaffold.

## Scope

Refusal at the write, at all three seams that produce a body: `aiwf add
--body-file`, `aiwf edit-body --body-file`, and `aiwf edit-body` in bless mode.
Bless mode refuses a *regression* only — a body dropping a section HEAD carried —
so an entity that already omits one stays editable.

One predicate answers "is this section absent", and every rule asking that
question routes through it.

## Out of scope

A tree-wide `aiwf check` rule. At error severity it raises the 109 findings
E-0081 already declined; at warning severity it raises them against the four
warnings this tree carries today, and a warning nobody can act on trains a reader
to skip the output. The measurement above is the record instead. The same
reasoning the epic applies to the Work logs of terminal milestones applies here:
the historical record stays as written.

Promoting `## Release note` into the kernel required set. That rule triggers on a
status and `requiredSectionsByKind` has no status axis; see the decision recorded
below.

## Acceptance criteria

### AC-1 — edit-body --body-file refuses a body that omits a required section

The refusal names the missing section. Naming it is what the test asserts, not
merely a non-zero exit: an unresolvable id and a working copy with drifted
frontmatter both refuse on this path already, so an exit-code assertion passes
with the guard absent.

The claim covers a body offered for the first time. Re-supplying a body identical
to the one HEAD carries converges to a no-op whether or not it omits a section,
which is the state every already-omitting entity is in.

Evidence: an epic body carrying `## Goal` and `## Scope` and no `## Out of
scope`, driven through this path, refused with the section named; today the same
input commits and `aiwf check` reports zero errors.

### AC-2 — The add-time gate refuses an omitted required heading, not only an empty one

The fixture carries exactly one required section empty and every other filled.
The scaffold will not do: it leaves every required heading empty, so deleting one
leaves a survivor to refuse, and the test passes with no guard at all.

A body carrying no headings at all is refused too. Today the gate inspects only
headings that are present, so prose with none lands.

The gate's message stops promising that `aiwf check` will block until the section
is filled. That is true of a section present and empty and false of a deleted
heading, which is the asymmetry the operator currently exploits.

Evidence: the discriminating fixture refused before and after the heading is
deleted; a headless body refused; the message asserted against what the check
actually reports.

### AC-3 — A blessed body that drops a required section HEAD carried is refused

Scoped to a regression. A working copy lacking a section HEAD also lacked is
committed as before — that half is what the test pins, and it is what keeps the
55 already-omitting entities editable by whoever next touches one.

The verb offers no `--force`, so an operator who cannot satisfy the refusal has
only a sovereign acknowledgement. Refusing on a regression makes that tolerable:
the edit in hand is the cause.

Bless mode already refuses on body content — `body-prose-id` runs there under the
same write block — so this adds a precondition of a kind the verb already has.

Evidence: a HEAD body carrying every required section, a working copy dropping
one, refused; a HEAD body already missing one, a working copy still missing it,
committed.

### AC-4 — One absence predicate serves every rule that asks whether a section is there

`internal/check/milestone_release_note.go` counts an absent section as empty.
`check.EmptyRequiredSections` skips one. Two answers to one question live in the
same package, and the guards this milestone adds would be a third.

Evidence: the release-note rule and the write-time guards resolved to the same
predicate, asserted by a check that fails if a second definition of section
absence is reachable from either.

## Decisions made during implementation

- The `milestone-done-empty-release-note` rule stays separate rather than folding
  into the general mechanism. Its trigger is a status — it reports only a `done`
  milestone — and the required-section table has no status axis. Folding it in
  would demand the section from a draft's first body edit. Measured: 6 of 8 live
  milestones lack the heading and none of those 6 is `done`. What it does share is
  the absence predicate, which AC-4 unifies.

## Work log

## Validation

## Deferrals

## Reviewer notes
