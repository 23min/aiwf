---
id: M-0329
title: An entity body that omits a required section is refused at the write
status: done
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

Nothing. G-0571 names the hole this milestone narrows, but closing it is
E-0084's: that epic's scope carries the push seam and the prose retirement, and
its success criteria name the gap. This milestone delivers E-0084's first
scope item early, from a different epic, and leaves the gap open — a write-time
rule reads only bytes it is writing, so the 55 entities already missing a section
are untouched by it.

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

Refusal at the write, at three of the four seams that produce a body: `aiwf add`
(`--body` and `--body-file`), `aiwf edit-body --body-file`, and `aiwf edit-body`
in bless mode.

`aiwf add` demands a complete body, because a new entity has no history to be
held to. Both of `aiwf edit-body`'s modes refuse a *regression* only — a write
dropping a section HEAD carries — so an entity already omitting one stays
editable, and an operator changing one section is not refused over an omission
they did not introduce.

One predicate answers "is this section absent", and every rule asking that
question routes through it.

## Out of scope

`aiwf import`, the fourth seam. It supplies caller-authored body bytes from a
manifest and is left ungated because the verb is deprecated — measured, the same
body `aiwf add` refuses lands through `import` at exit 0. The deprecation is
recorded on no surface, and two shipped skills still instruct a consumer to use
it, which is G-0667; until that resolves, this exclusion rests on a decision a
reader cannot verify from the tree.

A tree-wide `aiwf check` rule. At error severity it raises the 109 findings
E-0081 already declined; at warning severity it raises them against the four
warnings this tree carries today, and a warning nobody can act on trains a reader
to skip the output.

That is a blast-radius decision, not a claim that the omissions are historical.
Measured: none of the 55 is terminal — 30 open gaps, 24 accepted decisions, and
one proposed epic. They are live records carrying live debt, and a write-time
rule leaves every one of them standing. What closes them is a tree-side rule with
a baseline, which ADR-0048 and E-0084 own; this milestone reaches the write seams
and no further.

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
  rather than one rule at all three seams. Ratified as ADR-0048, which supersedes
  ADR-0043 on that clause alone — the placement, the definition of a violation,
  the split from emptiness, and the unbuilt push seam all carry forward. A new entity has no committed body to
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

## Release note

`aiwf add` and `aiwf edit-body` now refuse a body that is missing a section its
kind requires. The two verbs ask different questions, because a create and an
edit are in different positions:

- **`aiwf add`** requires a complete body — every `## <Section>` the kind
  declares — for every kind. A new entity has no history to be judged against,
  and the scaffold writes every heading, so only an explicit `--body` or
  `--body-file` can drop one. `--force --reason` still bypasses it, and now
  stamps its trailer on epic and milestone too, where it was previously inert.
- **`aiwf edit-body`** refuses only a write that *drops* a section the committed
  body carries, in both bless and `--body-file` mode. An entity whose body
  already omits a section stays editable, so an author changing one section is
  never refused over an omission they did not introduce. There is no `--force`
  here; record a deliberate removal with `aiwf acknowledge illegal`.

Both refusals name the section they missed. `aiwf check` is unchanged — it still
reports a required section that is present and empty, and still says nothing
about one that is absent, so a tree carrying that debt is unaffected.

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

Run on the final tree, against base
`epic/E-0091-hold-each-spec-section-to-what-it-uniquely-holds-and-check-what-ships`:

    make ci                exit 0   (vet, lint 0 issues, race suite, coverage gate, self-check 29 steps)
    make stress-tests      exit 0   (internal/stresstest, cmd/stresstest)
    aiwf check             exit 0   6 findings, 0 errors

The six are all warnings and none is this milestone's subject: two archive-sweep
advisories raised by superseding ADR-0043, an epic with no drafted milestones, and
an undefined provenance range.

Verdict parity for the parser swap was measured before it landed, over every
entity body in this tree: `EmptyRequiredSections` against the same function
rewritten onto `entity.ParseBodySections`, 7,284 (file, kind) pairs, zero
differences. A reviewer re-derived it independently over a superset — 18,126
pairs — with a negative control confirming the harness fires on `##\t` and on an
H1 mid-section. `aiwf check --format=json` on this tree reports the same findings
before and after.

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

Two rounds of independent fresh-context review ran over the full change-set, each
three lenses wide: code-quality, design, and the wrap's shape measurements. Both
rounds returned request-changes; both are closed. The deciding round found nothing
wrong in the shipped code path — every blocking finding was in the record, and the
mechanism survived 14 cross-mode input classes, an independent re-derivation of
every census figure, and a 20-spelling attack on the ban.

### Decided against

- **A tree-wide rule**, at either severity. The reasoning is in `## Out of scope`;
  what belongs here is that review pressed it twice and the answer held on
  blast radius, not on the omissions being historical — they are not, and the
  first version of that argument borrowed a premise this set does not satisfy.
- **Completeness at the edit seams**, which ADR-0043 had ratified. A reviewer
  argued the honest version of that case: adding `--force` to `aiwf edit-body`
  would let completeness apply everywhere and converge the tree, and reasoning
  from "the flag does not exist" to "so the rule must be weaker" is circular.
  The answer that survives is one ADR-0048 should carry and does not: `--force`
  is sovereign and human-only, so completeness at an edit would hard-block an AI
  actor from touching any of the 54 incomplete born-complete entities.
- **Unifying the two seams onto one call**, passing the required set as the
  create's baseline so `sectionsDroppedSince` serves both. It would collapse two
  rules into a parameter. Declined here because the two refusals need different
  messages and the create's is load-bearing for the operator, but it is the
  better shape if a third seam ever arrives.

### Known and left

- `aiwf import` is a fourth body-producing seam, ungated on its deprecation
  (G-0667). Until that is recorded somewhere, the exclusion rests on a decision
  the tree does not carry.
- An entity absent from HEAD takes the nil-baseline path, so `aiwf edit-body
  --body-file` will commit a body `aiwf add` would refuse. Reachable by
  `git reset --mixed` after a create. Pinned in neither direction.
- A `## <Section>` inside a fenced code block satisfies every seam. Uniform
  across all of them, so not a divergence — but the gate is stated as requiring
  a complete body, and this is a way past it.
- The ban has no owner and no retirement trigger, and its by-name exemption map
  costs per subject. One entry, and deleting it makes the policy fire.
- The `prefixTests`/`searchTests` split is pinned by no test: collapsing both
  into one map with a single containment test passes every row. It is defensible
  on false-positive grounds, and that argument is written down nowhere.

### Attacked and survived — ground the next round can skip

Both rounds recorded these; the second reviewer read the list and skipped it,
which is what it is for.

- The add gate across all six kinds, via `--body`, `--body-file`, stdin, an empty
  file and headless prose: refused every time, every missing heading named.
- `--force`'s `bypassed` accounting in three states: a trailer only where a real
  refusal was overridden, none for a no-op.
- Cross-mode agreement over 14 input classes, driven through the CLI: identical
  verdict and identical message on every one.
- Same-state convergence: NoOp, exit 0, zero commits, and a dirty working copy
  correctly does *not* converge.
- Every `//coverage:ignore` in the change-set, checked against git's own exit
  codes and the `go/ast` contract rather than read.
- All six guards, each with a caller named that reaches it. None proven dead.
- The six prose templates, which still satisfy the gate the rituals now hit.
- Composition with `retitle`, `reallocate`, `move` and the archive sweep: none
  can drop a required heading.
- ADR-0048's load-bearing measurement and the 55/109 census, both re-derived
  independently with a separate parser.
