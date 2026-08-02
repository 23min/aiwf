---
id: M-0286
title: Close the archive sweep's referrer and destination gaps
status: in_progress
parent: E-0075
tdd: required
acs:
    - id: AC-1
      title: A referrer absent from the loaded tree but present at HEAD declines the move
      status: met
      tdd_phase: done
    - id: AC-2
      title: An archived referrer mid-edit does not block an unrelated candidate
      status: met
      tdd_phase: done
    - id: AC-3
      title: Both ends of a move are enumerated
      status: met
      tdd_phase: done
    - id: AC-4
      title: A working-copy-only link declines that candidate, and --dry-run predicts --apply
      status: met
      tdd_phase: done
    - id: AC-5
      title: The decline predicate and the rewrite predicate derive from one enumeration
      status: met
      tdd_phase: done
---
## Goal

Make the archive sweep's per-candidate decline decide from the record, so the class M-0284
recorded as closed actually is.

## Context

M-0284 gave `archive` a per-candidate decline: a move whose verdict rests on a mid-edit file is
declined and reported, while the rest of the sweep proceeds. The carried half of that decision
was corrected to read the record as well as the working tree. The referrer half was not, and it
drifts in both directions.

Measured after M-0284 shipped, a referrer whose frontmatter is momentarily unparseable — the
ordinary bless rhythm — lets the target's move land without its link rewrite, and HEAD keeps a
link to a path that no longer exists. No re-run repairs it, because an archived target leaves
the scan for good, and `aiwf check` reports zero errors.

G-0499 carries the measurement and the neighbouring defects that share its root cause.

## Acceptance criteria

### AC-1 — A referrer absent from the loaded tree but present at HEAD declines the move

A referrer that HEAD records and the loaded tree does not — deleted on disk,
hand-renamed, or carrying momentarily unparseable frontmatter — makes the move it
links into undecidable, so that candidate is declined and named in the sweep's
skip report while the rest of the sweep proceeds.

The unparseable-frontmatter route is the one that matters, because it is the
ordinary rhythm rather than an accident: mid-edit YAML is briefly invalid, and
editing an entity body in the working tree before blessing it is what the shipped
guidance recommends.

This is the criterion with unrecoverable damage behind it. The move currently
commits without its link rewrite, HEAD keeps a link to a path nothing occupies,
and no later run repairs it — `IsArchivedPath` excludes the archived target from
every subsequent scan, so the sweep reports the tree converged. Restoring the
referrer changes nothing.

### AC-2 — An archived referrer mid-edit does not block an unrelated candidate

A mid-edit file already under an `archive/` subdirectory is not a blocker. It
cannot lose a link that a live sweep would rewrite, because the rewrite pass
excludes archived entities; treating it as a blocker declines a candidate that has
nothing to do with it.

The two predicates disagree here today, and that disagreement is the whole defect:
the rewrite pass filters archived paths and the decline pass does not.

### AC-3 — Both ends of a move are enumerated

A move's destination is examined alongside its source. A file sitting at the
destination — untracked, or recorded and divergent — declines that candidate by
name.

Enumerating the source alone leaves the condition invisible to the decline and
live at the commit, where the guard refuses the whole verb. That whole-verb
refusal for one participant is exactly the behaviour the per-candidate decline
exists to replace, so reaching it through an unenumerated path is a defect in the
decline rather than in the guard.

### AC-4 — A working-copy-only link declines that candidate, and --dry-run predicts --apply

A link present in the working copy and absent at HEAD is a divergence the decline
accounts for. It currently is not: the decline reads HEAD's body and finds no
committed link to lose, while the rewrite pass reads the working copy and emits an
op for it — so the move stays in the plan and the commit-side guard refuses the
verb, naming no candidate.

The observable form is the agreement between the two modes: the candidates a dry
run reports as sweeping are the ones an apply sweeps, and the ones it reports as
skipped are the ones an apply skips. A dry run that promises a sweep the apply
refuses is a defect on its own terms, independent of which condition caused it.

### AC-5 — The decline predicate and the rewrite predicate derive from one enumeration

A candidate is declined if and only if a file its verdict rests on is mid-edit.
No arrangement of the tree makes one predicate count an entity the other does not.

The other four criteria each name one arrangement where the two disagree today.
This one is the claim that no fifth exists — which is why its evidence is a
property test over constructed tree states rather than a reproduction. Asserting
structurally that one function calls the other would pin the implementation and
leave the behaviour free to drift, which is the failure mode a ledger recording
names rather than behaviour already demonstrated in this epic.

## Constraints

- The decline predicate and the rewrite predicate are one rule, not two that agree by
  inspection. `moveBlockers` and `planArchiveRewrites` currently disagree on which entities are
  candidates, which archived entities count, and which end of a move is enumerated.
- No fix may reintroduce whole-verb refusal for one mid-edit participant; that is the behaviour
  the per-candidate decline exists to replace.
- `--dry-run` must predict `--apply`. A dry run that promises a sweep the apply refuses is a
  defect in its own right.

## Out of scope

- The comparison primitive (`gitops.DivergentPaths`) and the claim-side guard, both settled in
  M-0284.
- The commit path's filter-blindness (G-0498), which is about what blobs aiwf stores rather than
  which candidates a sweep declines.

## Dependencies

- M-0284 — the primitive and the decline machinery this corrects.

## References

- G-0499 — the measured defect and its three neighbours
- ADR-0038 — the per-candidate scoping
- M-0284 — where the class was recorded as closed
## Work log

### AC-1 — A referrer absent from the loaded tree but present at HEAD declines the move

`dirtyEntityPaths` unions the loaded tree's entity paths with HEAD's, classified
by `entity.PathKind` · commit f8b80cf8a, extended by bd9cc14e0

### AC-2 — An archived referrer mid-edit does not block an unrelated candidate

`moveBlockers` applies the archived-path filter the rewrite pass already
applied · commit 8af98880f

### AC-3 — Both ends of a move are enumerated

`moveEnds` supplies source and destination to the carried walk and the blocker
match alike · commit c5124c9ed

### AC-4 — A working-copy-only link declines that candidate, and --dry-run predicts --apply

A link in either copy blocks, with the absent-from-HEAD exemption kept so the
decline and the commit-side guard accept the same set · commit ce05fa725

### AC-5 — The decline predicate and the rewrite predicate derive from one enumeration

Three properties over 32 enumerated arrangements, plus a non-gap arrangement
holding the scan to every kind · commits 0ae83e98f, bd9cc14e0

## Decisions made during implementation

- **The absent-from-HEAD exemption is kept, not removed.** Measured: the
  commit-side guard exempts an absent-from-HEAD divergence at an `OpWrite`'s own
  destination, which is exactly a referrer's rewrite, so that write lands.
  Blocking here would decline a candidate the commit accepts — a disagreement in
  the opposite direction.
- **A non-gap arrangement, not a kind dimension.** Adding referrer kind to the
  property grammar doubles it for the same protection. One ADR referrer detects
  both narrowings that left the suite green.
- **A directory at a recorded entity path is a divergence, not an error.** The
  byte-wise comparison refuses a directory outright, which fails the verb over
  one participant. Reporting it keeps the decline per-candidate.

## Validation

- `make ci` — exit 0
- `AIWF_COVERAGE_BASE=645d482d4 make coverage-gate` — exit 0
- `aiwf check` — 0 errors
- Mutation matrix, eight reversions of production behaviour, all detected.
  Four tests were deleted after measuring each was never a unique detector;
  the matrix stayed 8/8 and coverage stayed green.

## Deferrals

- G-0511 — `LsTreePaths` filters in Go rather than passing a pathspec to git.
  Throughput, not correctness. Filed rather than fixed here because the helper
  is shared with callers outside the sweep.
- G-0512 — a directory occupying a move's destination is invisible to the
  decline, so the sweep offers a plan that cannot land. The decline judges
  destination *divergence*, and a directory is neither untracked nor modified;
  "something already occupies this path" is a filesystem question the
  comparison does not ask.
- G-0513 — a candidate terminal in the record and unparseable on disk is
  neither swept nor reported as masked. The masked-terminal report still walks
  the loaded tree only, so the two halves of one decision disagree about which
  entities exist.

## Reviewer notes

**Two independent lenses ran before closure, then a deciding pass.** The design
lens reconstructed the sweep from intent before reading the implementation and
arrived at the same two predicates, which is the evidence that the shape is
essential rather than residue: the decline must read a strict superset of the
rewrite's domain and both versions of each body, because it answers whether the
working copy is trustworthy enough for the rewrite's answer to be right. A single
shared enumerator would need two knobs whose only two call sites set them to
opposite values.

**The round's blocking finding was an insufficient pin, not a wrong fix.**
Narrowing `entity.PathKind` to one kind, or the HEAD listing to one directory
prefix, left every test green while ADR and milestone referrers stranded the
dangling link AC-1 exists to prevent. The evidence covered one of six kinds
while the criterion's text is kind-agnostic.

**AC-5's text asserts more than its evidence establishes.** "No arrangement of
the tree makes one predicate count an entity the other does not" is a universal
over an unbounded space; what the properties check is that every plan the sweep
offers is one it can land, and that no active entity is left linking to a path
nothing occupies, across the referrer states an operator reaches. The narrower
claim is the one that holds.

**The two sides now read the same slice of a file.** The record side compared
whole-file bytes while the working side and the rewrite pass compared
body-only. That was taken for a harmless over-block until it was measured:
the link scan tracks fenced regions by counting ```-leading lines, so
frontmatter carrying one opens a region before the body and hides every link
behind it. With the working copy mid-edit and its link dropped, neither
predicate counts the referrer and the move lands with the record still
pointing at a path it vacated — AC-1's damage class, reached by an entity
that parses. Both sides now scan the body alone.

**AC-1's phase ladder is compressed and AC-5's `met` precedes its evidence
commit.** AC-1's `green` and `done` were stamped together after the audit rather
than at first pass; AC-5 was promoted `met` before the commit carrying its
property file. Both read oddly in `aiwf history`.

**The per-candidate skip was operator-visible and documented nowhere.** Fixed
here rather than deferred: `--help` and the verb skill now state that a skip is
normal, what causes it, and what to do next.

**One review finding is declined.** The deciding pass proposed deleting the
attributability property, having measured it as never the unique detector
across thirteen mutants. That argument generalizes to any property test: a
property earns its place against the cases nobody enumerated, and the two
properties the same pass kept are non-unique by the identical measure. This
one carries the "only if" half of AC-5 — that a candidate is declined *only*
when a file its verdict rests on is mid-edit — which no other test states in
general form. Deleting it would narrow a met criterion's evidence after the
fact.

**A mutation probe reported a survivor twice, and both were compile failures.**
Replacing a loop's range expression left a variable unused, so the package did
not build and a scan for failing tests found none. A probe that cannot build
is not evidence of a surviving mutant; the build is checked before the verdict
is read.
