---
id: M-0331
title: Refuse at the push a body that reached a commit without passing a verb
status: in_progress
parent: E-0084
tdd: required
acs:
    - id: AC-1
      title: A body committed without passing a verb is refused at the push
      status: met
      tdd_phase: done
    - id: AC-2
      title: Promote, retitle and archive still succeed against an entity missing a section
      status: met
      tdd_phase: done
    - id: AC-3
      title: The new rule leaves aiwf check's tree-wide output unchanged
      status: met
      tdd_phase: done
    - id: AC-4
      title: A required section absent as a top-level heading is the only violation
      status: open
    - id: AC-5
      title: A template edit cannot change what the gate enforces
      status: open
---
## Goal

Close the last hole in body-section enforcement: a body can reach a commit
without passing any verb, and nothing catches it. Add a gate on the push,
riding the commit range the provenance audit already resolves.

## Closes

- G-0571 — the enforcement hole. After this milestone a body cannot reach a
  commit carrying fewer required sections than it had, whether or not a verb
  wrote it. The bodies committed without one before it stay as they are.

## Context

`entity.RequiredSections` has been the single definition of each kind's body
sections since E-0081, and M-0329 wired a refusal into `aiwf add` and both
modes of `aiwf edit-body`. What remains unenforced is the path that bypasses
those seams entirely: a body written into a commit by plain `git commit`. That
path is not hypothetical — the wrap-milestone ritual writes the milestone spec
that way today.

ADR-0048 places this gate and defines what a violation is; ADR-0043 established
why the enforcement is forward-only by construction rather than by policy.

## Acceptance criteria

### AC-1 — A body committed without passing a verb is refused at the push

A commit that drops a required section from an entity's body is refused at the
push, whether or not it carries verb trailers.

The proof runs against the path that does this today — the wrap-milestone
ritual's plain `git commit` of the milestone spec — rather than a synthetic
commit constructed for the test. That commit carries the ritual's three
trailers and still passes no body-supplying verb, which is why the gate cannot
filter on trailer presence: the range reader yields every commit in the window,
and the filtering the provenance audit does is its own. A synthetic path would
pin the rule against an input nobody produces; the real one is what the gate
exists for.

### AC-2 — Promote, retitle and archive still succeed against an entity missing a section

A status promote, a retitle, and an archive sweep each succeed against an
entity whose body omits a required section.

The gate's scope is entities whose body *content* changed in the range. An
entity merely touched by a frontmatter write or a file move is outside it.
Without this the gate would block ordinary work on debt that work did not
create, and the three verbs above are where that would bite first.

### AC-3 — The new rule leaves aiwf check's tree-wide output unchanged

`aiwf check` reports the same findings on this tree before and after the gate
lands.

The rule does not join `check.Run`; it runs only on the push, over a commit
range. No existing entity gains a finding, which is what keeps the tree's
accumulated omissions out of scope by construction rather than by a
grandfather list.

### AC-4 — A required section absent as a top-level heading is the only violation

A violation is exactly one thing: a required section absent as a top-level
`##` heading.

Sections beyond the declared set are legal and never reported. Order is not
enforced. Stating the rule this narrowly is what lets an author add their own
headings and arrange them freely while the declared set stays mandatory.

### AC-5 — A template edit cannot change what the gate enforces

Editing a prose template does not change what the gate enforces.

The scan's only input is the section set the kernel declares — located today at
`entity.RequiredSections`. The templates describe that set for a human reader
and are not consulted, so the two cannot drift into disagreeing about what is
required.

## Constraints

- The rule does not join `check.Run`. `aiwf check`'s tree-wide output is
  unchanged and no existing entity gains a finding.
- Scope is entities whose *body content* changed in the range, never entities
  merely touched. A touched-path scope would block ordinary work on debt it did
  not create.
- No workflow available today may become unavailable. An author who does not yet
  know a section's content keeps the heading and leaves it empty.

## Design notes

- ADR-0048 is the decision this implements; it supersedes ADR-0043 on what the
  verb seam asks, while the placement, the definition of a violation, and the
  push seam carry forward unchanged.
- The finding code is settled. D-0090 gives each property its own: this gate
  takes a new code naming absence, and the emptiness rule extends the existing
  `entity-body-empty` under E-0083. `entity-body-section-absent` pairs with
  that sibling and is the proposed spelling, open to a better one at
  implementation. A new code owes a row in the shipped `aiwf-check` findings
  table; the discoverability policy enforces that rather than leaving it to
  vigilance, so it needs no criterion of its own here.
- **The gate asks what the verb seams ask: an edit must not regress.** A commit
  that drops a required section an entity's body carried is refused; one that
  keeps an omission already there is not. Holding a push to completeness instead
  would refuse an author over an omission they did not introduce — the cost
  ADR-0048 measured at the edit seam and rejected, and the push has no `--force`
  either. It would also contradict the verb it rides behind: `aiwf edit-body`
  permits an edit that keeps an existing omission, so the push would refuse the
  commit that verb had just made, with no way to satisfy both.
- Creates are not judged here. `aiwf add` already holds them to completeness and
  carries the sovereign `--force` that ADR-0048 makes a standing exemption;
  re-judging the same body at the push would undo that override. An entity file
  written by hand and committed without a verb is caught by
  `provenance-untrailered-entity-commit` before this rule would see it.
- A path appearing in the range does not mean its body changed; the reader
  yields paths, not hunks. Deciding *body content changed* means comparing
  post-frontmatter bytes, which is what keeps a frontmatter-only promote outside
  the scope AC-2 protects.
- Three things the rule must not do, each a way a range carries a body its
  author did not write: refuse over an omission already present at the range
  base, undo an `aiwf add --force` exemption, or re-judge a body a merge
  absorbed from another branch. The provenance audit skips ordinary `--no-ff`
  merges for the third reason and this gate has the same one. Whether that falls
  out of a base-against-HEAD tree comparison or needs the commit reader's
  `ParentSHAs` is for the tests to settle.
- Nothing converges the bodies already committed without a section. ADR-0048's
  Consequences names this seam as what changes that; under the rule above it
  does not, and correcting that sentence is part of this milestone.
- G-0571 carries an inherited obligation: whoever closes it folds the
  `milestone-done-empty-release-note` rule into the general mechanism, or records
  why it stays separate. It stays separate — `Release note` is not in the
  milestone's required set, changing that set is out of scope here, and that rule
  is tree-wide and status-conditional where this gate is range-scoped and
  status-blind. The record belongs in that rule's own doc comment.
- The gate inherits the provenance audit's range resolution, which is skipped
  when no upstream is configured and no `--since` is passed. CI-on-push is the
  backstop; do not claim otherwise in the milestone's own prose.

## Surfaces touched

- `ResolveUntrailedRange` in `internal/cli/check/` — the range resolution the
  gate rides, with `ReadUntrailedCommits` beside it if the merge case needs it
- `internal/check/provenance.go` — where a sibling pass over the same range
  already lives
- `check.AbsentRequiredSections` — the membership scan M-0329 built for the verb
  seams, called here rather than re-derived, which is what makes AC-5 structural
- `internal/check/milestone_release_note.go` — the record of why that rule stays
  separate, per G-0571's inherited obligation
- ADR-0048 — its Consequences sentence naming this seam as what converges the
  existing omissions

## Out of scope

- The existing violations. Both seams read only bytes being written, so a body
  already committed is never in scope. Paying that debt is a migration with its
  own evidence.
- Emptiness. Whether a section that is present carries content is ADR-0042's
  subject and E-0083's work.
- Changing what any kind's required set contains. That set is E-0081's answer
  and this milestone consumes it.
- `aiwf import`, which is excluded on its deprecation pending G-0667.

## Dependencies

- ADR-0048 — accepted; the decision this implements.
- D-0090 — settles the finding code, which ADR-0043 and ADR-0042 both deferred.
- M-0329 — delivered the verb seam this completes.

## Coverage notes

- (none)

## References

- ADR-0048, ADR-0043, ADR-0042
- D-0090 — one code each: this gate names absence, `entity-body-empty` keeps
  emptiness
- E-0081 — gave the section set one owner and deliberately excluded enforcement
- E-0083 — shared the finding-code question, now answered by D-0090
- G-0571 — the hole this closes, jointly with the deletion milestone
- G-0667 — `aiwf import`'s unrecorded deprecation

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
