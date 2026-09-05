---
id: M-0329
title: An entity body that omits a required section is refused at the write
status: in_progress
parent: E-0091
tdd: required
acs:
    - id: AC-1
      title: edit-body --body-file refuses a body that drops a required section HEAD carries
      status: met
      tdd_phase: done
    - id: AC-2
      title: The add-time gate refuses an omitted required heading, not only an empty one
      status: met
      tdd_phase: done
    - id: AC-3
      title: A blessed body that drops a required section HEAD carried is refused
      status: met
      tdd_phase: done
    - id: AC-4
      title: One absence predicate serves every rule that asks whether a section is there
      status: open
      tdd_phase: done
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

`aiwf add` demands a complete body, because a new entity has no history to be
held to. Both of `aiwf edit-body`'s modes refuse a *regression* only — a write
dropping a section HEAD carries — so an entity already omitting one stays
editable, and an operator changing one section is not refused over an omission
they did not introduce.

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

### AC-1 — edit-body --body-file refuses a body that drops a required section HEAD carries

The refusal names the missing section. Naming it is what the test asserts, not
merely a non-zero exit: an unresolvable id and a working copy with drifted
frontmatter both refuse on this path already, so an exit-code assertion passes
with the guard absent.

The comparison is against the committed body, not against completeness. A body
that omits a section HEAD also omits is committed — the state 55 entities in this
tree are in, and the verb offers no `--force` to get back out of a refusal.

Evidence: an epic whose committed body carries all three required sections,
handed a body carrying `## Goal` and `## Scope` and no `## Out of scope` through
this path, refused with the section named; today the same input commits and
`aiwf check` reports zero errors.

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

One rule, asked at both of the verb's seams. The two modes take different inputs
— bytes handed to the verb, and a working copy it reads — but produce the same
write, so an answer that differs by mode means the same edit to the same entity
is refused or committed depending on which flag the operator reached for.

The verb offers no `--force`, so an operator who cannot satisfy the refusal has
only a sovereign acknowledgement. Refusing on a regression makes that tolerable:
the edit in hand is the cause.

Bless mode already refuses on body content — `body-prose-id` runs there under the
same write block — so this adds a precondition of a kind the verb already has.

Evidence: both halves driven through both modes and asserted to agree — a
committed body carrying every required section against an edit dropping one,
refused by each; a committed body already missing one against an edit still
missing it, committed by each.

### AC-4 — One absence predicate serves every rule that asks whether a section is there

`internal/check/milestone_release_note.go` counts an absent section as empty.
`check.EmptyRequiredSections` skips one. Two answers to one question live in the
same package, and the guards this milestone adds would be a third.

Evidence: the release-note rule and the write-time guards resolved to the same
predicate, asserted by a check that fails if a second definition of section
absence is reachable from either.

## Decisions made during implementation

- Completeness is demanded at `aiwf add` and non-regression at `aiwf edit-body`,
  rather than one rule at all three seams. A new entity has no committed body to
  be judged against, so completeness is the only question there; an edit has one,
  and holding it to completeness would refuse an operator over an omission they
  did not introduce. Measured on a gap already omitting `## Why it matters`: with
  the paths scoped differently, `--body-file` refused an edit that kept the
  omission while bless mode committed it — one verb, two answers, for the same
  edit to the same entity.

- Absence is refused for every kind; emptiness stays born-complete-only. The two
  halves of the add-time gate are scoped differently because the workflows they
  must leave alone differ. A draft epic is meant to land with its headings empty
  and be filled in, so emptiness keeps the born-complete scope it has. No kind is
  meant to land without its headings at all — the scaffold writes every one, so
  absence is reachable only from an explicit `--body`/`--body-file`. Scoping it
  the narrower way would leave `aiwf add` accepting bytes `aiwf edit-body`
  refuses, which is the asymmetry this milestone exists to remove. Measured cost:
  18 of the 51 fixture fixes were the draft-bearing kinds.

- The `milestone-done-empty-release-note` rule stays separate rather than folding
  into the general mechanism. Its trigger is a status — it reports only a `done`
  milestone — and the required-section table has no status axis. Folding it in
  would demand the section from a draft's first body edit. Measured: 6 of 8 live
  milestones lack the heading and none of those 6 is `done`. What it does share is
  the absence predicate, which AC-4 unifies.

## Work log

### AC-1 — edit-body --body-file refuses a body that drops a required section HEAD carries

Refused with the missing section named; the comparison against the committed body
that keeps an already-omitting entity editable arrived with AC-3 · commit 1386448
· check-fast and coverage gate green

### AC-2 — The add-time gate refuses an omitted required heading, not only an empty one

Both halves now run at the gate, scoped as the decision above records, and the
absent-section refusal no longer promises the check will block · commit 4d2a589 ·
make ci, the stress-tagged lane, and the coverage gate green

### AC-3 — A blessed body that drops a required section HEAD carried is refused

Both modes now route through one rule and one refusal, and the test asserts they
agree rather than checking each alone · commit d43ddbb · make ci, the
stress-tagged lane, and the coverage gate green

## Validation

## Deferrals

- G-0666 — a body line over 1 MB makes the section scanner report that section
  as empty. Pre-existing and error severity, so the pre-push hook already blocks
  on a body that is not empty; the guards this milestone adds refuse the write on
  the same input, naming a heading the file carries. The fix is a choice between
  three shapes across five call sites, which is why it is not taken here.

## Reviewer notes
