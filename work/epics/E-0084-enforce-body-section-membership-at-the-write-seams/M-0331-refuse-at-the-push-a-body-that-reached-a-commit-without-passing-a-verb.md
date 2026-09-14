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
      status: met
      tdd_phase: done
    - id: AC-5
      title: A template edit cannot change what the gate enforces
      status: met
      tdd_phase: done
---
## Goal

Close the last hole in body-section enforcement: a body can reach a commit
without passing any verb, and nothing catches it. Add a gate on the push,
riding the commit range the provenance audit already resolves.

## Closes

- G-0571 — the enforcement hole: a push that leaves a required section out of a
  body that carried it, or out of an entity created without a verb, is refused.
  ADR-0049 records what no seam reaches.

## Context

`entity.RequiredSections` has been the single definition of each kind's body
sections since E-0081, and M-0329 wired a refusal into `aiwf add` and both
modes of `aiwf edit-body`. What remains unenforced is the path that bypasses
those seams entirely: a body written into a commit by plain `git commit`. That
path is not hypothetical — the wrap-milestone ritual writes the milestone spec
that way today.

ADR-0049 decides what this gate asks, superseding ADR-0048 on the push seam;
ADR-0043 defines what a violation is and why the enforcement is forward-only by
construction rather than by policy.

## Acceptance criteria

### AC-1 — A body committed without passing a verb is refused at the push

A push that leaves a required section out of an entity's body is refused: a
section the entity carried when the pushed range started, or, for an entity the
range created without `aiwf add` or `aiwf import`, any required section. Verb
trailers on the commit make no difference.

The proof runs against the history the wrap-milestone ritual produces rather than
a synthetic commit. The spec is edited on a milestone branch by a plain
`git commit` carrying the ritual's trailers, merged into the epic branch with
`--no-ff`, and the epic branch is judged against its upstream. A test built on a
commit sitting directly on the pushed branch would pin the rule against a
history nobody produces.

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

- ADR-0049 is the decision this implements. It supersedes ADR-0048 on what the
  push seam asks and keeps ADR-0043's definition of a violation; D-0092, which
  first answered that question, is superseded by it.
- The finding code is `entity-body-section-dropped`, under D-0090's split: this
  gate names a section left out, and emptiness stays with `entity-body-empty`.
  It covers an entity created without a verb as well as a removal, so its message
  reads "leaves required section … out of" to be true of both. A new code owes a
  row in the shipped `aiwf-check` findings table, which the discoverability
  policy enforces.
- The gate judges each entity by id from its starting point to HEAD, reading the
  whole range with `git log --cc`: a side branch's commits count, and a merge
  counts only where it resolved content itself. Reading along the first-parent
  line commit by commit cannot see a drop merged in with `--no-ff`, loses one a
  later rename carries, and refuses a push whose section came and went.
- A reported section is credited to the newest write that took it out, so the
  finding names a commit `aiwf acknowledge illegal <sha>` can exempt.
- G-0571's inherited obligation — fold the `milestone-done-empty-release-note`
  rule into the general mechanism, or record why it stays separate — is
  discharged as the second, in that rule's own doc comment.
- The gate inherits the provenance audit's range, which is skipped when no
  upstream is configured and no `--since` is passed. No workflow runs
  `aiwf check`, so nothing backs that up; G-0679 tracks it.

## Surfaces touched

- `internal/check/entity_body_section_dropped.go` — the walker and the rule
- `RunProvenanceCheck` in `internal/cli/check/provenance.go` — hands the gate the
  base of the range `ResolveUntrailedRange` resolves
- `check.AbsentRequiredSections` — the membership scan the verb seams use, called
  here rather than re-derived, which is what makes AC-5 structural
- `gitops.BlobReader` — every body read, over one `git cat-file --batch`
- `internal/check/hint.go`, the `aiwf-check` skill's findings table, and
  `aiwf acknowledge illegal --help` — the remedy and the escape
- `internal/check/milestone_release_note.go` — why that rule stays separate
- ADR-0049, which supersedes ADR-0048 and D-0092

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

- ADR-0049 — accepted; the decision this implements.
- D-0090 — settles the finding code, which ADR-0043 and ADR-0042 both deferred.
- M-0329 — delivered the verb seam this completes.

## Coverage notes

- (none)

## References

- ADR-0049, ADR-0048, ADR-0043, ADR-0042
- D-0090 — one code each for a section left out and a section left empty
- D-0092 — superseded by ADR-0049
- E-0081 — gave the section set one owner and deliberately excluded enforcement
- E-0083 — shared the finding-code question, now answered by D-0090
- G-0571 — the hole this closes, jointly with the deletion milestone
- G-0667 — `aiwf import`'s unrecorded deprecation
- G-0679 — the ranges the push seam cannot judge

## Release note

`aiwf check`, and so the pre-push hook, now refuses a push that leaves a required
section — `## Goal`, `## What's missing`, and the rest of each kind's set — out of
an entity's body, reporting `entity-body-section-dropped` at error severity. It
catches edits that never passed through `aiwf edit-body`, including a plain
`git commit` carrying aiwf trailers, and it judges the whole push: a section
removed on a branch merged in, or in a file a later commit renamed, is reported,
and one added and removed again within the push is not. A section already
missing before the push is never reported. An entity created by hand rather than
with `aiwf add` must carry every required section. Restore the heading with its
content to clear the finding — for a gap, decision, ADR or contract an empty
required section is itself an error — or keep a removal with
`aiwf acknowledge illegal <sha> --reason "..."`, which exempts every finding on
that commit. The check runs only when the branch has an upstream or
`--since <ref>` is passed.

## Decisions made during implementation

- ADR-0049 — the push seam holds each entity to its starting point rather than
  to the whole set; it supersedes ADR-0048 on that seam and replaces D-0092.

## Validation

Measured 2026-09-14 in the devcontainer (linux/amd64, go1.25.11), on the
milestone branch with its last build input at `92996688b`.

| Command | Expected | Observed |
|---|---|---|
| `make check-fast` | exit 0 | exit 0 — vet, `go test` across 71 packages, lint clean |
| `AIWF_COVERAGE_BASE=09d2058cc make coverage-gate` | exit 0 | exit 0 |
| `aiwf check` | 0 errors | 0 errors; 15 warnings, none from this rule |
| worktree binary, scratch repo in the wrap ritual's shape — a milestone branch drops `## Acceptance criteria` in a commit carrying the ritual's trailers, merged `--no-ff` into an epic branch with an upstream — then `aiwf check` | an `entity-body-section-dropped` error naming the milestone-branch commit, exit 1 | exactly that, naming `7b29a4f6` rather than the merge; the run's other error is `refs-resolve`, for the fixture's missing epic file |
| worktree binary, scratch repo: drop a section, `aiwf acknowledge illegal <sha> --reason "meant it"`, then `aiwf check --since <base>` | the finding before, exit 0 after | one error before, exit 0 after |

A manual mutation probe ran 16 mutants against the walker and rule, since no
mutation tool is wired for a local diff, restoring each file byte-identical. All
but one fail a test. The survivor removes the check that skips an entity gone at
HEAD: the body read after it finds no entity there and skips it anyway, and the
check stays as the statement of the rule.

Every AC is `met` with `tdd_phase: done`. AC-1's seam test, built on the ritual's
merge shape, fails against a gate that reads only the first-parent line.

## Deferrals

- G-0679 — a branch's first push, and a pull request merged on the server, are
  judged by nothing.

## Reviewer notes

- (none)
