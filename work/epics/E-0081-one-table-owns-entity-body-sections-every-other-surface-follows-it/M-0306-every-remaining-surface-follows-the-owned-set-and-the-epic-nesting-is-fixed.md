---
id: M-0306
title: Every remaining surface follows the owned set, and the epic nesting is fixed
status: in_progress
parent: E-0081
depends_on:
    - M-0305
tdd: required
acs:
    - id: AC-1
      title: Each shipped prose template contains its kind's required sections at top level
      status: met
      tdd_phase: done
    - id: AC-2
      title: A template-drafted entity's body keys name sections, not optionality markers
      status: met
      tdd_phase: done
    - id: AC-3
      title: No shipped template presents the title heading as mandatory
      status: met
      tdd_phase: done
    - id: AC-4
      title: The normative body-sections table states what the add scaffold writes
      status: cancelled
      tdd_phase: done
    - id: AC-5
      title: The self-check and coverage fixtures derive their bodies from the owned set
      status: cancelled
      tdd_phase: done
---

## Goal

Make the shipped prose templates a checked superset of the owned section set, so a
body drafted by following the ritual satisfies the same rule as one the verb scaffolds.

## Context

The prose templates materialize into a consumer's `.claude/templates/` and are what
the planning rituals instruct an author to fill from. They are richer than the verb
scaffold by design — commentary, optional sections, fill-in guidance — and that part
is worth keeping. What is not worth keeping is their freedom to disagree with the
required set.

The epic template is the live instance. It places out-of-scope one heading level below
the flat `## Out of scope` that the check, the `aiwf add` scaffold, and the JSON body
map all name. Measured: every epic in this tree drafted from the prose template carries
no `out_of_scope` key in `aiwf show --format=json`; epics carrying the flat form do.
Because `entity-body-empty` skips an absent heading, neither side reports it.

A second issue sits on the same artifacts, and it is not a disagreement. The prose
templates open with a title H1; the verb scaffold writes none. Both are correct: the
kernel treats the H1 as optional, and `aiwf retitle` implements that — it keeps a
canonical `# <id> — <title>` in sync when one is present and is a no-op when it is
not, leaving an operator-shaped heading alone rather than clobbering it. The tree
shows the same, unevenly: a minority of ADRs and decisions carry one, almost no gaps,
and no epic or milestone at all.

What is missing is the word "optional". A template that opens with the heading and
says nothing reads as mandatory to whoever fills it in, while the repo's own entities
mostly omit it.

A third issue reaches the read path rather than the author. Every template marks its
optional sections by suffixing the heading — `## Risks (optional)` — and `SectionSlug`
folds that suffix into the key, so the section surfaces as `risks_optional` in
`aiwf show --format=json`. The marker is guidance about whether to keep the section;
it is not part of the section's name, and a key that carries it names the guidance.

Two axes of the template are already pinned against the real production oracle:
frontmatter decodes through `entity.Parse`, and prose is scanned by the real
`body-prose-id`. The section axis is the one with no such test.

## Approach

Extend the existing pattern rather than inventing one. The frontmatter test drives the
accepted-key set from the `Entity` struct through the real decoder; the section test
drives the required set from M-0305's owned definition, so it needs no second list to
maintain and a change to the set fails here without anyone remembering to update it.

Containment, not equality — the templates are a superset by design, and asserting
equality would forbid the optional sections that are the reason they exist. Heading
level is part of containment: a required section nested a level down does not count,
which is what makes the epic template fail on day one.

The H1 needs no decision — the kernel already made it, and both surfaces already
comply. The deliverable is that a reader can tell: the templates say the heading is
optional, in the form they already use for their other optional sections, and a test
keeps the shipped surfaces from later contradicting the kernel's stance.

## Acceptance criteria

### AC-1 — Each shipped prose template contains its kind's required sections at top level

For each shipped prose template, every section its kind's owned set names is present
as a top-level heading. The required set is read from M-0305's definition rather than
restated here, so adding a section to the set fails this test until the templates
follow. Optional and extra sections are permitted — the assertion is containment, not
equality. A required section present at a deeper heading level fails.

### AC-2 — A template-drafted entity's body keys name sections, not optionality markers

An entity drafted from a shipped prose template yields, in `aiwf show --format=json`,
a body key for every section its kind's owned set names, and no key carrying an
optionality marker. The test drives the real projection over real entity files built
from the shipped template bytes — through `aiwf add`, the loader, and the envelope —
rather than reading the template in memory, which is AC-1's subject.

`SectionSlug` folds a heading's whole text into its key, so `## Risks (optional)`
yields `risks_optional`. The marker is authoring guidance about whether to keep the
section at all; folding it into a data key makes the key name that guidance instead of
the section, and two entities of one kind then disagree on their key set according to
whether each author happened to delete the parenthetical. The templates state
optionality in the prose beneath the heading, where it reaches the author and not the
read path.

The `out_of_scope` key AC-1's fix produces is asserted here too, on the end-to-end
path. That half is green once AC-1 lands; the optionality half is what fails first.

### AC-3 — No shipped template presents the title heading as mandatory

A shipped template either omits the opening `# <id> — <title>` heading or marks it
optional. One rule, no per-kind list: a template carrying the heading unmarked fails,
whichever kind it serves.

The kernel's stance is not changed and no surface is made to match another. `aiwf
retitle` already treats the heading as optional — syncing a canonical one, no-oping
when absent, leaving a non-canonical one alone — and the scaffold writing none is that
stance, not a contradiction of it. What the templates lack is the word, and a reader
filling one in has no way to know the heading is theirs to skip.

Which side of the rule each template satisfies follows the measurement rather than a
preference for uniformity. No epic and no milestone in this tree carries the heading,
against templates that open with one — an instruction every author has silently
overridden — so those two templates drop it. Roughly a quarter of ADRs and decisions
carry one and every one of them is canonical, which is the heading earning its place:
those files are read as documents outside aiwf, where the id and title are all that
identify them. Those two templates keep the heading and mark it.

The heading reaches no read path either way. It sits above the first `## `, so the
body map drops it and no key depends on this criterion — which is why the rule is
about what a template presents to its reader, and why the assertion is scoped to the
region between the heading and the first section rather than to the file.

### AC-4 — The normative body-sections table states what the add scaffold writes

`docs/design/design-decisions.md`'s body-sections table names, for each kind, exactly
the sections `aiwf add` scaffolds — which is what its own caption claims it lists. A
test compares the table against the owned set and fails if either moves alone.

Equality here, where AC-1 asserts containment, because the two surfaces answer
different questions. A prose template is a superset by design: it carries commentary
and optional sections a scaffold has no business writing. This table is captioned as
the scaffold's output, so a section it names that `aiwf add` does not write is not
extra detail — it is the caption being false. The milestone row is where that shows:
it lists five sections the rich template contributes, none of which the verb writes.

The doc's claim that bodies are not validated is corrected in the same change.
`entity-body-empty` reports an empty required section, and the born-complete kinds
refuse one at creation, so the sentence describes a kernel that has not existed since
those landed. That correction is prose and no test pins it; it is caught at review.

### AC-5 — The self-check and coverage fixtures derive their bodies from the owned set

`selfCheck*Body` in `internal/cli/doctor/selfcheck.go` and `bornCompleteFixtureBody`
in `internal/cellcoverage/fixture.go` stop spelling out section headings and render
them from `entity.RequiredSections` instead. Both build bodies for the four
born-complete kinds, which refuse a bare scaffold at creation and so need real prose
supplied.

This is the one criterion in the milestone that retires a surface rather than checking
one. Every other route here leaves a copy in place and adds a test to watch it; these
two can stop being copies, and then there is nothing to watch. That is the cheaper
outcome and the one the epic prefers wherever it is available.

The failure it forecloses is the worst-shaped in the inventory. Both fixtures are
correct today. If a kind's set later gains a section, each would build an entity
missing it — and because nothing reports an absent heading, `aiwf doctor --self-check`
would go on passing while creating exactly the defect the epic exists to prevent, in
the one place a consumer runs to ask whether their install is healthy.

## Constraints

- Containment, not equality, **for the prose templates** — they stay a superset and
  keep their commentary and optional sections. The normative table is the exception
  and AC-4 says why: it is captioned as the scaffold's output, so a section it names
  that the scaffold does not write makes the caption false rather than adding detail.
- The required set is read from M-0305's owned definition; this milestone introduces
  no second copy of it.
- The templates keep materializing where D-0015 puts them; this milestone changes
  their content, not their destination.
- Every `SKILL.md` edit under the embedded rituals lands with its referencing
  structural test, per the repo's backstop policy. That policy matches `SKILL.md`
  paths only, so a template edit carries no mechanical backstop; the template edits
  here are covered by AC-1's and AC-2's assertions instead, which read the shipped
  bytes through `skills.ListRitualTemplates`.

## Design notes

The four milestone-template sections that duplicate structured data (G-0530) are not
touched here. Containment permits them, and whether they earn their place is a
separate judgment with its own evidence — one of them carries the wrap ritual's
commit-SHA trail.

Whether the containment check covers optional sections is deliberately unanswered:
only the required set is checked unless a second case appears.

## Surfaces touched

`internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/`,
`docs/design/design-decisions.md`, `internal/cli/doctor/selfcheck.go`,
`internal/cellcoverage/fixture.go`, `internal/policies/`.

## Out of scope

- The always-on guidance's scaffold instruction, which M-0307 routes.
- Whether the milestone template's duplicating sections should exist at all (G-0530).
- Collapsing the prose templates into the verb scaffold — four rituals read them.

## Dependencies

M-0305 — the owned section set this milestone checks the templates against.

## References

- G-0479 — epic template nests out-of-scope below the level three surfaces require
- D-0015 — ritual templates materialize to the templates dir
- E-0081 — parent epic

## Work log

### AC-1 — Each shipped prose template contains its kind's required sections at top level

The epic template's out-of-scope heading moved to top level, and `### In scope` was
dropped with its bullets rising under `## Scope`; a containment test over every
shipped template reads the required set from the owned definition · commit 01b1cf758 ·
tests all green

Heading level is the load-bearing half of the assertion, not a stylistic preference.
`ParseBodySections` — the production parser behind the JSON body map and the
`entity-body-empty` rule — matches `## ` alone, so the nested form yielded no
`out_of_scope` key on any read path.

Neither side of the comparison is restated here. The kind is resolved from each
template's own placeholder id through the kernel's id-prefix table: `KindFromID`
matches the full id pattern, which requires digits, so it resolves nothing for
`E-NNNN` and cannot serve. That resolver is pinned by its own test, since a
placeholder binding to the wrong kind would compare a template against another
kind's set and pass vacuously.

### AC-2 — A template-drafted entity's body keys name sections, not optionality markers

Every optionality marker left the four templates' headings, and the ADR template's
status-vocabulary section became a comment; an integration test drafts one entity per
templated kind through `aiwf add` and asserts the show envelope's key set names the
template's sections · commit 9d1b529bb · tests all green

The criterion was rewritten before it was built. As specified it asserted that an epic
drafted from the template carries `out_of_scope`, which is AC-1's conclusion reached
through a second function — the same fix produces both, and AC-1 covers every template
rather than the epic alone. It had no failing test available to it, and under
`tdd: required` the phase FSM admits no entry but `red`. Reframing kept the end-to-end
coverage the criterion existed for — a real file through `aiwf add`, the loader, and
the envelope, none of which AC-1 touches — and attached it to a defect still live, so
the red was genuine. The `out_of_scope` assertion survives as the criterion's first
half.

Eight headings carried a parenthetical that `SectionSlug` folded into the key, across
all four templates: seven `(optional)` markers plus the ADR template's
`## Status vocabulary (aiwf)`. The latter is authoring guidance rather than a section
of any ADR's content, so it became an HTML comment above the first heading, where the
body map drops it, rather than a section renamed.

The expected key set derives from each template heading with a trailing parenthetical
removed. That stripping is inert against the repaired templates and is the whole
guard: measured, neutering it while a marker is re-added makes expected and actual
agree on the leaked key and the assertion passes silently. It has its own test for
that reason.

Six live entities had already drawn from the templates and were repaired in the same
pass, distinguished by measurement rather than swept by pattern · commits 8ad54942f
through dba3ea840. Four ADRs carried the status-vocabulary block verbatim, no authored
word among them, and lost it. `D-0033` and two ADRs carried real prose under a marked
heading and kept every word of it. Scope stopped at live entities: an author's own
`## Risks (to weigh at the ADR)` is a heading rather than a marker, several gaps carry
such headings legitimately, and archived entities are forget-by-default.

### AC-3 — No shipped template presents the title heading as mandatory

The epic and milestone templates dropped the title heading; the ADR and decision
templates kept it and now say it is optional; one test admits either shape and
refuses an unmarked heading · commit 7815c361e · tests all green

The measurement decided which template lands on which side, and it was one-sided. No
epic and no milestone in the tree carries the heading, against two templates that
opened with one — an instruction overridden every time it was given. Among ADRs and
decisions roughly a quarter carry one and all 26 are canonical, with no unfilled
placeholder anywhere in 253 live entities. So one surface stops prescribing what
nobody wants and the other keeps what its readers use.

The rule is scoped to the region between the heading and the first `## `. Measured, a
marking moved below that bound leaves the test red, which is the difference between
this assertion and a file-wide search for the word.

What the test cannot do is read. For the two templates that keep the heading it
bottoms out in the presence of a word in a region — prose checking prose, satisfiable
by a rewrite that says something else. Judged acceptable against building a parser for
template commentary; the deletion half carries the weight, since absence is exact.

### AC-4 — The normative body-sections table states what the add scaffold writes

The milestone row lost the five sections the prose template contributes and the verb
does not write; a test compares every row against the owned set · commit fb47c361f ·
tests all green

Only that row was wrong — the other five already named what `aiwf add` scaffolds. The
row had documented a different artifact than its caption claims, so the richer shape
is now attributed to the prose template in a sentence beneath the table, which is
where it is true.

The doc's claim that bodies are not validated was false in both directions and is
corrected. `entity-body-empty` reports a required section present and empty — warning
for epic and milestone, error for the born-complete kinds — and those kinds refuse an
empty body at creation; membership is enforced nowhere, which the doc now states and
attributes to G-0571 rather than leaving a reader to infer a stricter kernel than
exists. No test pins that prose, and the first pass corrected one copy of four: the
sentence below the table, leaving the "not enforced structure" lead-in three lines
above it and the same claim in `legal-workflows-first-principles.md` and twice in
`legal-workflows-audit.md`. Review caught it; all four now agree.

The parse is anchored on the table's header row and bounded by the table. Measured, a
reworded header fails with that diagnosis rather than passing over an empty haystack —
the failure mode that matters for a test reading a document it does not own.

This is the milestone's one ongoing obligation: a future change to the owned set must
carry this doc with it or CI fails. Deleting the table would have retired the copy
instead, and was rejected — `docs/design/` is Normative tier, and replacing a six-row
table with a pointer to a command costs a reader more than the mandate costs a writer.

### AC-5 — The self-check and coverage fixtures derive their bodies from the owned set

Three fixture builders stopped transcribing section headings and render from the owned
set; an AST walk refuses a heading literal in any of them · commit 80095df91 ·
tests all green

The criterion's own comparison would not have failed. Both fixtures named the right
sections already, so checking their output against the owned set passes before the
change and is a tautology after it, both sides calling one function. What can fail is
the spelling, since a transcription is the thing that goes stale — so the test forbids
transcribing rather than checking the transcript.

The sweep found three copies where the epic's inventory named two: `internal/verb`
carried a byte-identical twin of the cellcoverage builder. Both now derive their kinds
from `entity.IsBornComplete` as well as their sections, retiring a second enumeration
nobody had asked about. Output verified byte-identical to all four retired constants
and all four retired switch arms before the swap, and `aiwf doctor --self-check` passes
29 steps either side of it.

Measured payoff rather than argued: adding a section to gap's owned set makes both
fixtures carry it with no edit. Before the change they would have built a gap without
it, and `aiwf doctor --self-check` would have kept passing — the failure this
forecloses, in the one command a consumer runs to ask whether their install is healthy.

### Review corrections — commit 2815c74e8

Four fresh-context reviewers over the full change-set returned request-changes. Two
defects reached shipped surfaces that no criterion's test could see, because every
assertion here reads template bytes and these are the templates' *readers*:
`aiwfx-record-decision` named `Validation (optional)` and `Consequences (optional)`
after those markers left the templates — aiwf's own ritual instructing an author to
produce the key AC-2 exists to foreclose — and three epic rituals prescribed the
`in / out` substructure AC-1 flattened, which is G-0479 recreated in a consumer's
repo. `TestRitualsNameTheOwnedSectionsWithoutMarkers` now pins both, scoped to the
instruction passage and reading the expected names from the owned set.

That scoping is the finding against the fix. The first version searched the whole
file, and probing showed it did not catch the epic defect — the phrase appears
elsewhere in that skill, so the check passed while the instruction stayed wrong. The
same looseness the review had just reported, in the test written to answer it.

Also corrected: the branch-coverage gate was red on `bornCompleteFixtureBody`'s
nil arm, which the refactor pulled into diff scope uncovered — `make check-fast` does
not run that gate, so "tests all green" was true of what had been run and misleading
about what it implied. `make ci` is green now and is what the remaining ACs are held
to. `titleHeadingPreamble` treated any `# ` line as the title heading; the search now
stops at the first section. `sectionNameSlugs` did not deduplicate while the key map
it is compared against does. Four package-level `var`s went back to call-site calls.

## Decisions made during implementation

- **AC-2 asserts the optionality-marker leak rather than the `out_of_scope` key alone.**
  As originally written it restated AC-1's conclusion through a second function: the
  same fix produces both, and AC-1 is the stronger assertion because it covers every
  template rather than the epic's. Reframing it kept the end-to-end read-path
  coverage the criterion was for — a real file, a real load, a real envelope, none of
  which AC-1 touches — and attached it to a defect that was still live, so the
  criterion has a genuine failing test rather than one green on arrival. The
  alternative considered and rejected was cancelling AC-2 as subsumed, which would
  have left the read path unpinned and the leak unfound.
- **AC-3 became one rule about what a template presents, not a marking mandate.** As
  specified it required every template to mark the title heading optional, and its test
  could only assert a sentence was present — prose pinned against prose, failing solely
  if someone deleted the sentence it looked for. Measured, there was no defect behind
  it: the kernel treats the heading as optional, `retitle` implements that, and all 26
  live entities carrying one carry a canonical one, with no unfilled placeholder among
  253. What the measurement did show is that the epic and milestone templates prescribe
  a heading no epic or milestone has ever kept. Restating the criterion as "omit it or
  mark it" gives one rule with a real failure mode, lets those two templates drop the
  line rather than annotate it, and leaves the ADR and decision templates — where a
  quarter of authors keep the heading, all correctly — to carry it and say so.
- **An optionality marker belongs in a section's prose, not its heading.** A heading is
  the section's name and `SectionSlug` turns the whole of it into a key. Guidance about
  whether to keep a section reaches the author from the prose beneath it just as well,
  and from there it cannot reach a consumer's data.

## Validation

## Deferrals

## Reviewer notes
