---
id: M-0331
title: Refuse at the push a body that reached a commit without passing a verb
status: done
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
  body that carried it, or out of an entity created other than by `aiwf import`
  or a forced `aiwf add`, is refused.
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
range created other than by `aiwf import` or a forced `aiwf add`, any required
section. A trailer on a commit that edits an entity makes no difference.

The proof runs against the history the wrap-milestone ritual produces rather than
a synthetic commit. The spec is edited on a milestone branch by a plain
`git commit` carrying the ritual's trailers, merged into the epic branch with
`--no-ff`, and the epic branch is judged against its upstream. A test built on a
commit sitting directly on the pushed branch would pin the rule against a
history nobody produces.

### AC-2 — Promote, retitle and archive still succeed against an entity missing a section

A status promote, a retitle, and an archive sweep each succeed against an
entity whose body omits a required section.

A write that changes only an entity's frontmatter, or moves its file, leaves the
sections its body carries unchanged, so it can report nothing the entity did not
already lack. Without this the gate would block ordinary work on debt that work
did not create, and the three verbs above are where that would bite first.

### AC-3 — The new rule leaves aiwf check's tree-wide output unchanged

`aiwf check`'s tree-wide pass reports the same findings on this tree before and
after the gate lands.

The rule does not join `check.Run`; it runs wherever `aiwf check` resolves a
commit range — the pre-push hook, or `--since`. No existing entity gains a finding, which is what keeps the tree's
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
- A finding is a section the entity carried at its starting point and lacks at
  HEAD, never one it already lacked. Judging every touched entity for
  completeness would block ordinary work on debt it did not create.
- No workflow available today may become unavailable. An author who does not yet
  know a section's content keeps the heading and leaves it empty.

## Design notes

- ADR-0049 is the decision this implements. It supersedes ADR-0048 on what the
  push seam asks and keeps ADR-0043's definition of a violation; D-0092, which
  first answered that question, is superseded by it.
- The finding code is `entity-body-section-dropped`, under D-0090's split: this
  gate names a section left out, and emptiness stays with `entity-body-empty`.
  It covers an entity created other than by `aiwf import` or a forced `aiwf add`
  as well as a removal, so its message reads "leaves required section … out of"
  to be true of both. A new code owes a
  row in the shipped `aiwf-check` findings table, which the discoverability
  policy enforces.
- The gate judges every entity that differs between its starting point and HEAD,
  by id, rather than only the entities git lists a write for: a merge can adopt
  one parent's copy of a file wholesale, and git lists no write for it. The
  starting point is where the branch left its base — judging against the base's
  current tip charges the pusher for whatever moved there after the fork.
- A reported section is credited by comparing a commit with its own parents: the
  newest commit whose version lacks it while every parent carried it, then a
  merge that adopted a copy lacking it, so a finding names a commit
  `aiwf acknowledge illegal <sha>` can exempt. Comparing each write with the
  previous one in log order instead names a clean merge whenever both sides
  edited the file. ADR-0049 records where the named commit can still be the
  wrong one.
- Two exemptions keep the gate off debt the pushing author did not create: a
  section the entity already lacks on the configured trunk under its own id, and
  a create by `aiwf import` or a forced `aiwf add`, which starts from the body it
  wrote. An `aiwf add` trailer without `aiwf-force` is not trusted, because that
  verb refuses to write an incomplete body unforced. Trunk is matched by the
  entity's own id alone, so an unrelated entity holding a prior id after a
  collision exempts nothing.
- G-0571's inherited obligation — fold the `milestone-done-empty-release-note`
  rule into the general mechanism, or record why it stays separate — is
  discharged as the second, in that rule's own doc comment.
- The gate inherits the provenance audit's range, which is skipped when no
  upstream is configured and no `--since` is passed. No workflow runs
  `aiwf check`, so nothing backs that up; G-0679 tracks it.

## Surfaces touched

- `internal/check/entity_body_section_dropped.go` — the walker and the rule
- `internal/verb/import.go` — a per-entity import plan stamps `aiwf-verb: import`,
  as the single-commit plan does, so the push sees an import in either mode
- `entity.RequiredSections`'s doc comment — the seams it names
- `RunProvenanceCheck` in `internal/cli/check/provenance.go` — hands the gate the
  base of the range `ResolveUntrailedRange` resolves
- `check.AbsentRequiredSections` — the membership scan the verb seams use, called
  here rather than re-derived, which is what makes AC-5 structural
- `gitops.BlobReader` — every body read, over one `git cat-file --batch`
- `internal/check/hint.go`, the `aiwf-check` skill's findings table, and
  `aiwf acknowledge illegal --help` — the remedy and the escape
- `internal/check/milestone_release_note.go` — why that rule stays separate
- `docs/design/legal-workflows-first-principles.md` and
  `docs/design/legal-workflows-audit.md` — the rule rosters that named only the
  two write seams
- `docs/design/design-decisions.md`, the `aiwf-add` skill, `aiwf add --force` help
  and the self-check's fixture comment — each said no rule reports an absent
  section
- ADR-0049, which supersedes ADR-0048 and D-0092

## Out of scope

- The existing violations. Each seam judges only what a write or a push changes,
  so a body committed before the pushed range is never in scope. Paying that debt is a migration with its
  own evidence.
- Emptiness. Whether a section that is present carries content is ADR-0042's
  subject and E-0083's work.
- Changing what any kind's required set contains. That set is E-0081's answer
  and this milestone consumes it.
- `aiwf import` at the write seam, excluded on its deprecation pending G-0667. The
  push exempts what an import created, in either commit mode.

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
- G-0571 — the hole this closes
- G-0667 — `aiwf import`'s unrecorded deprecation
- G-0679 — the ranges the push seam cannot judge

## Release note

`aiwf check`, and so the pre-push hook, now refuses a push that leaves a required
section — `## Goal`, `## What's missing`, and the rest of each kind's set — out of
an entity's body, reporting `entity-body-section-dropped` at error severity. It
catches edits that never passed through `aiwf edit-body`, including a plain
`git commit` carrying aiwf trailers, and it judges everything the checked-out
branch adds over its upstream as one push: a section removed on a branch merged
in, or in a file a later commit renamed, archived or reallocated, is reported, and
one added and removed again within the push is not. A section already missing
where the branch left its base, or already missing on trunk, is never reported.
An entity the push creates must carry every required section unless
`aiwf import` or `aiwf add --force` created it; an import in per-entity commit
mode now stamps `aiwf-verb: import` on each commit, so `aiwf history` and the
`aiwf status` digest show those entities as imported rather than added. Restore
the heading with its content to clear the finding — for a gap, decision, ADR or
contract that is not terminal, an empty required section is itself an error — or
keep a removal with
`aiwf acknowledge illegal <sha> --reason "..."`, adding `--for-entity <id>` when
the same commit is also reported by `provenance-untrailered-entity-commit`. The
check runs only when the branch has an upstream or `--since <ref>` is passed, and
the pre-push hook runs it on the branch checked out where the push is made.

## Decisions made during implementation

- ADR-0049 — the push seam holds each entity to its starting point rather than
  to the whole set; it supersedes ADR-0048 and replaces D-0092.

## Validation

Measured 2026-09-16 in the devcontainer (linux/amd64, go1.25.11, git 2.54.0), on
the milestone branch at `eed5ceb21`, whose last build input is `3de2153bd`. The
binary was built from that commit.

| Command | Expected | Observed |
|---|---|---|
| `AIWF_COVERAGE_BASE=09d2058cc make ci` | exit 0 | exit 0 — vet, lint clean, `go test -race` across every package, the diff-scoped gates over `09d2058cc`, self-check |
| `aiwf check --since 09d2058cc`, which runs the gate over this milestone's own range | 0 errors, no finding from this rule | 0 errors, 18 warnings, none from this rule |
| scratch repo in the wrap ritual's shape: a milestone branch drops a section in a commit carrying the ritual's trailers, merged `--no-ff` into an epic branch with an upstream; `aiwf check` | one `entity-body-section-dropped` naming the milestone-branch commit, exit 1 | exactly that, naming the drop rather than the merge |
| scratch repo: a branch with an upstream merges a `main` on which another commit dropped a section; `aiwf check`, then again with `origin/main` moved back to where the section existed | exit 0, then exit 1 | exit 0, then exit 1 naming the trunk commit |
| scratch repo: a commit with no aiwf trailers drops a section; `aiwf acknowledge illegal <sha> --for-entity <id> --reason "..."`; `aiwf check --since <base>` | two errors before, exit 0 after | `entity-body-section-dropped` and `provenance-untrailered-entity-commit` before, exit 0 after |
| scratch repo: a commit whose message carries the log's record separator drops a section, and later work follows it; `aiwf check --since <base>`, `aiwf acknowledge illegal` on the commit it names, then `aiwf check --since <base>` again | the drop commit named, then exit 0 | exactly that |

A manual mutation probe ran against this walker, since no mutation tool is wired
for a local diff, restoring the file byte-identical after each mutant, against
the tests in `internal/check`, `internal/cli/check` and `internal/policies`. The
mutants that survive: the skip for an entity byte-identical at both ends of the
range, which is equivalent — identical files carry identical sections; the memo
of a create commit's trailers, which changes only how often one commit is read;
the sort of reported paths, which survives a single run and fails under
repetition, since without it the order is a map's; and the two last-resort lines
of the credit, which no history reaches, as the argument above them states.
Each of these fails a test: putting message bytes into the record, blinding the
read of a create's trailers, reading a create's start from the wrong add or from
the trailers of a merge that wrote it in its own resolution, dropping either
credit pass, reading the range in clock order rather than
topological order, letting a prior id's file or another entity's file stand in
for the entity's own, linking a chain through an unrelated deletion, sharing the
section cache across entities, leaving `docs/adr` out of the tree scan, dropping
merge candidacy, or dropping `--no-renames`.

Every AC is `met` with `tdd_phase: done`. AC-1's seam test, built on the ritual's
merge shape, fails against a gate that reads only the first-parent line.

## Deferrals

- G-0679 — a branch started from a local ref is not judged at its first push, so
  a pull request merged on the server can carry content nothing compared.
- G-0684 — a path git quotes is invisible to the gate at both ends of the range,
  so an entity at such a path is judged by nothing.
- G-0685 — the pre-push hook runs the check on the branch checked out, not on the
  refs being pushed, so a push made from another branch's checkout is judged by
  nothing.
- G-0686 — a move the push makes outside one add-and-delete commit reads as a
  create, and is held to every required section.
- G-0687 — `aiwf acknowledge illegal --for-entity` refuses a merge commit, so a
  merge this rule credits that is also reported untrailered has no clearing form.

## Reviewer notes

- Verdict: the gate's design stands as reviewed; what the review declined or
  left in place is recorded here.
- Declined: restating ADR-0049's Decision as the one rule the code implements
  rather than one bullet per seam. An operator meets the seams one at a time,
  and the bullets are what they look up.
- Declined: routing the walker's git calls through a shared runner.
  `internal/check` has none — each walker there shells out for itself — and this
  one follows that shape.
- Declined: collapsing `sortedPaths` into an iterator one-liner; eight plain lines
  read better than the chain that replaces them.
- Limit left in place: the commit a finding names is best-effort (ADR-0049). An
  acknowledgment keyed to it can be re-raised when a later merge is credited
  instead; acknowledging the commit newly named clears it, and no acknowledgment
  moves the commit a finding names.
- Limit left in place: an acknowledgment exempts every section this rule reports
  on that commit, whichever entity `--for-entity` binds it to; it is a record
  about the commit (ADR-0049).
- Limit left in place: the `aiwf acknowledge illegal --help` row, the
  `aiwf check --since` and `aiwf add --force` help sentences, the two range-skip
  warnings that name this gate, and the `aiwf-add` skill's account of what
  `aiwf check` reports for an acceptance criterion's body are held by no test.
- Limit left in place: the gate's base is the range argument the provenance
  audit resolves, minus its `..HEAD` suffix — the only two shapes that resolver
  returns; a third shape would need its own base.
- Declined: a report that `aiwf reallocate <path>` is refused while trunk holds
  the duplicate id. Measured on a branch that had not merged trunk, the verb
  renumbers the branch's file at exit 0 with the trunk-collision finding standing.
- Limit left in place: the AC-5 test depends on at least one shipped template
  carrying a section beyond its declared set. The demand is named in the test's
  header and retires with the test.
