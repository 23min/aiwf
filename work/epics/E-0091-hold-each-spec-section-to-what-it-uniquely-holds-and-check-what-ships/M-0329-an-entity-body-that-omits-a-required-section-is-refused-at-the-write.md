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
      status: met
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

Measured, the split runs one layer deeper than the rules: the two functions read
different parsers. On `##\tGoal` the write-time guards report the section
present and the release-note rule reports it absent — one body, two answers,
from functions sitting eight lines apart.

The unification is that all three read one parser, not that all three call one
helper. The release-note rule needs no absence test of its own: a heading that is
not there produces no key, and the empty string the lookup yields is what its
emptiness classifier already reports as unwritten.

Evidence: all three surfaces driven over the heading spellings the parsers
disagreed on and required to answer alike, each on a body shaped so its verdict
turns on nothing but the heading; and a ban that fails when a package deciding
section presence carries a heading scan of its own. The agreement test is
mutation-verified — re-introducing a tolerant scanner inside
`EmptyRequiredSections` fails it on `##\t`, which the earlier version of this
test, comparing two callers that read the same parser, did not notice.

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

### AC-4 — One absence predicate serves every rule that asks whether a section is there

`check.SectionsAbsent` is the one answer; the check package's own heading scanner
is deleted rather than left with a single caller, and a ban keeps a second from
being written · commit cf22a95 · make ci, the stress-tagged lane, and the
coverage gate green

## Validation

Run on the milestone branch at AC-4's close, against base
`epic/E-0091-hold-each-spec-section-to-what-it-uniquely-holds-and-check-what-ships`:

    make ci                exit 0   (vet, lint 0 issues, race suite, coverage gate, self-check 29 steps)
    make stress-tests      exit 0   (internal/stresstest, cmd/stresstest)
    aiwf check             exit 0   5 findings, 0 errors

Verdict parity for the parser swap was measured before it landed, over every
entity body in this tree: `EmptyRequiredSections` against the same function
rewritten onto `entity.ParseBodySections`, 7,284 (file, kind) pairs across 1,272
files, zero differences. `aiwf check --format=json` on this tree reports the same
five findings before and after.

## Deferrals

- G-0666 — a section whose content carries one line of 65,536 bytes or more is
  reported empty, because the classifier deciding that reads through a scanner
  at bufio's default buffer and never consults `scanner.Err()`. Pre-existing and
  error severity, so the pre-push hook blocks on a section that is full. AC-4
  closed the heading half of the same class by retiring the capped scanner that
  decided which sections a body carries; three siblings still raise their
  ceiling rather than remove it, and what they need to agree on — whether a body
  they cannot finish reading is judged silently at all — is a decision, not a
  patch.

## Reviewer notes

Three independent fresh-context lenses ran over the full change-set before the
milestone closed: code-quality, design, and the wrap's shape measurements. Two
returned request-changes. The corrective round they ask for is unfinished — what
follows is the worklist, not a closing summary.

### Findings that block

- **The seam census is wrong.** `## Scope` claims all three body-producing seams;
  there are four. `aiwf import` supplies caller-authored body bytes and is
  ungated — measured, identical bytes refused by `aiwf add` and accepted by
  `aiwf import` at exit 0 with no `--force` and no record. The verb is
  deprecated, which is the reason to exclude it, but no surface says so.
- **Ten surfaces describe behaviour the code no longer has.** A worked example in
  the shipped `aiwf-add` skill is refused by the gate it documents; `--force` is
  documented as inert on epic and milestone and now stamps a trailer;
  `aiwf edit-body`'s new hard refusal appears in no `--help` or skill; and two
  normative design docs plus the doc comment on the file that defines the section
  set still assert that nothing enforces membership.
- **AC-4's evidence is thinner than the criterion states.** The `absent` term
  added to the release-note rule is dead logic: an absent section yields empty
  content, which the emptiness classifier already reports, so the term changes no
  verdict — measured by deleting it, package green. The agreement test compares
  two callers that both read the shared parser; re-introducing a second scanner
  inside `EmptyRequiredSections` leaves it passing. Only the ban covers the
  function that actually swapped parsers.
- **The ban catches two heading-scan spellings out of eleven.** `(?m)^## ` evades
  it, and that idiom is already the sitting model inside a package the ban scans.
  One of its two by-name exemptions carves out nothing — measured by deleting the
  entry: still zero violations — so it is a dead entry that would silently excuse
  that function if it were ever rewritten into a real scan.
- **Three tests pin nothing.** Two rows of the agreement test are byte-identical,
  and the second's name describes a whitespace its input does not carry. The
  `cross-worktree-id-race` row of the seeding test survives both mutations of the
  property it claims, because that scenario's Setup runs only `git`. The
  AC-1-era single-mode refusal test is subsumed by the cross-mode test that
  replaced it.

### Claims of this milestone that review overturned

- `## Out of scope` justified declining a tree-wide rule by the epic's reasoning
  about terminal milestones. Measured: none of the 55 entities is terminal — 30
  open gaps, 24 accepted decisions, 1 proposed epic. The conclusion may hold; the
  argument recorded for it does not.
- The `mid-write-kill` comment states G-0666 as a 1 MB ceiling in the section
  scanner. It is 64 KB, in the emptiness classifier. The gap body was corrected
  and the correction did not reach this copy.
- `## Validation` reports 7,284 pairs across 1,272 files, which do not reconcile.
  A second reviewer re-derived the same zero over a superset — 18,126 pairs —
  with a negative control that fires, so the finding is stronger than stated.

### The governing decisions this milestone was planned without

ADR-0043 is accepted and decides this design: membership enforced at the write
seams and nowhere else, a scan called by every body-supplying verb refusing for
every kind, and a second seam on the push that it calls the authority. E-0084 is
proposed, carries the same goal, and names closing G-0571 — which this milestone
also claims. Neither was consulted when this milestone was planned or when its
edit-seam rule was chosen.

Two substantive divergences. This milestone refuses a *regression* at the edit
seams where ADR-0043 refuses *incompleteness*; and it builds no push seam, so no
surface can answer which entities are incomplete. ADR-0043's argument for
completeness rests on a remedy it describes as free — keep the heading, leave it
empty — which is measurably not free for the born-complete kinds carrying 54 of
the 55 omissions: adding the empty heading converts a silent omission into an
error-severity finding that blocks the push.

### Attacked and survived — ground the next round can skip

- The add gate across all six kinds, via `--body-file` omitting one section and
  via a headless body: refused every time, every missing heading named.
- `--force`'s `bypassed` accounting in three states: a trailer only where a real
  refusal was overridden, none for a no-op.
- Both `edit-body` modes agreeing on drop-versus-keep, driven through the CLI on
  an epic and a milestone: identical message, identical exit.
- Both `//coverage:ignore` annotations, checked against git's own exit codes
  rather than read: accurate.
- All three arms of the add-time gate, each with a caller named that reaches it.
- The six prose templates, which all still satisfy the gate the rituals now hit.
- AC-4's parity claim, re-derived independently over a superset with a working
  negative control.
