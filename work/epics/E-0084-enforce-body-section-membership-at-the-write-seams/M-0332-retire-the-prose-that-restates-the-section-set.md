---
id: M-0332
title: Retire the prose that restates the section set
status: in_progress
parent: E-0084
tdd: required
acs:
    - id: AC-1
      title: No surface restates the section set as a per-kind table
      status: met
      tdd_phase: done
    - id: AC-2
      title: The design doc cites the owning symbol, and the citation resolves
      status: met
      tdd_phase: done
---
## Goal

Retire the prose that restates each kind's required section set, now that a
refusal carries it, so the set is stated in one place instead of three.

## Closes

- (none)

## Context

`entity.RequiredSections` has been the single definition of each kind's body
sections since E-0081, but two surfaces still enumerate that set in prose: the
`aiwf-add` skill's per-kind body-section table, and the body-sections table in
`docs/design/design-decisions.md`. They were safe to keep while nothing
enforced the set, because a reader had no other way to learn it.

M-0329 changed that. The verb seam now refuses a body that omits a required
section, so the set is discoverable by running the verb, and the prose copies
are second sources that can drift. Retiring them is what E-0084 means by the
deletion being the point rather than a tidy-up: the enforcement without the
deletion leaves the duplication in place.

## Acceptance criteria

### AC-1 — No surface restates the section set as a per-kind table

No shipped skill or normative design doc states a kind's required section set
as a table row keyed by that kind.

The check reads each kind's sections from the kernel, scans the shipped-skill
and normative-doc corpus for a table row whose first cell names a kind and
whose remaining cells carry that kind's sections, and asserts it finds none.
Both sides derive: a section added to a kind's set changes what the scan looks
for, so neither artefact can move without the other following.

The claim is scoped to the table shape because that is what states the set as
a set. Prose advising what to write *in* a section is not a second copy of the
membership and survives — the `aiwf-add` skill's per-kind authoring advice is
the case in point.

It is deliberately not stated as "these two passages are deleted". D-0070
retires prose-presence assertions over the shipped skill tree, and naming the
two files would pin the inputs rather than the rule: a copy added to a third
surface would go unreported.

Two surfaces keep a statement of the set and are correct to. The `aiwf-show`
skill's table names the JSON keys an envelope carries, and the prose templates
carry the headings as scaffold. Each derives from the kernel and is already
pinned by its own relationship check, so neither is the free-drifting copy this
milestone retires; this criterion adds no assertion about them.

### AC-2 — The design doc cites the owning symbol, and the citation resolves

The design doc that carried a retired table names the Go declaration owning
the required section set, and the name it writes is really declared in the
file it links to.

Two checks, one per conjunct. The citation's presence is asserted inside the
named section rather than anywhere in the document, because a citation
elsewhere does not help a reader who arrived at that section looking for the
sections. The resolution is asserted over every such citation under the
design-doc tree, reading the name from the prose and the declarations from
the Go source, so a rename on either side turns it red. That the link target
exists is already held by the design-doc anchor rule and is not asserted a
second time.

The skill that carried the other retired table is outside this criterion. Its
route is a command, and that the command resolves is held by the
skill-coverage rule; that the prose still names one is a property of shipped
prose, which D-0070 holds at review rather than by assertion, because an
assertion there pins a reading a reword breaks and nothing catches. Narrowing
the criterion to what is checked is the alternative to writing a check that
reads as evidence for more than it holds.

## Constraints

- Deletion, not correction. A passage rewritten to agree with the kernel is
  still a second copy; the next drift is a rewording away.
- A reader who reached the set through the deleted table must still reach it.
  Removing the table without leaving a route makes the surface worse.

## Design notes

- The evidence here needs care. D-0070 retires prose- and heading-presence
  assertions over the shipped skill tree, and `aiwf-add` is in that tree, so
  "assert the table is gone" is the banned shape wearing a minus sign. What
  survives D-0070 is the relationship check: read the section names from the
  kernel and scan for the table shape that states them. Both sides derive, so
  a reword of the surrounding prose cannot break it and a re-added table
  cannot evade it.
- The scan covers the whole shipped-skill and normative-doc corpus rather than
  the two files being edited. A test pins a rule, not an input: the rule is
  that the set is stated once, and these two files are two of its inputs. The
  reach costs one list of roots, so breadth is not where the cost sits.
- No ledger of surfaces permitted to state the set. Measured, the scan finds
  nothing once the two tables are gone, and an empty expectation needs no
  escape hatch. One is designed if a legitimate case ever appears.

## Surfaces touched

- the `aiwf-add` skill's per-kind body-section table
- the body-sections table in `docs/design/design-decisions.md`

## Out of scope

- Changing what any kind's required set contains. That set is E-0081's answer.
- The push seam. Its own milestone owns it, and neither waits on the other:
  what makes these passages safe to delete is the verb seam M-0329 landed.
- G-0530's question of whether the milestone template's structured-data
  sections should exist at all. That asks whether a section is worth carrying;
  this asks only where the set is written down.

## Dependencies

- M-0329 — delivered the refusal that makes the prose redundant.

## Coverage notes

- (none)

## References

- D-0070 — why the evidence is a relationship check rather than a phrase assertion
- E-0081 — gave the section set one owner
- M-0329 — delivered the verb seam
- G-0571 — the hole this closes, jointly with the push-seam milestone; its own
  body still carries superseded counts and a claim M-0329 falsified
- G-0530 — the adjacent, out-of-scope question

## Release note

The `aiwf-add` skill no longer carries a per-kind table of required body
sections. Run `aiwf template <kind>` to see what a kind requires; `aiwf add`
and `aiwf edit-body` already refuse a body that omits one, and the refusal
names the missing headings. Consumers see the change on their next
`aiwf update`.

## Decisions made during implementation

- None new. The corpus-wide scope of AC-1's census, and the absence of a
  ledger of surfaces permitted to state the set, were both settled into
  `## Design notes` before implementation began.

## Validation

Run against the milestone branch tip in the devcontainer — linux/amd64,
go1.25.11, `aiwf` built from that tip.

- `make check-fast` — `go vet` plain and under `-tags stress` and
  `-tags testpins`; golangci-lint 0 issues; `go test -parallel 8 ./...` ok
  across every package.
- `make coverage-gate` — green. It reads only files carried in the coverage
  profile, which never holds a `_test.go`, so what it judges here is the scan
  itself rather than the tests around it. One arm is annotated unreachable:
  the markdown walk fails only on a mid-walk IO fault.
- `aiwf check` — 0 errors, 12 warnings, none of them on this milestone. The
  warnings are the standing archive backlog and the provenance audit skipping
  itself because this worktree has no upstream configured.
- Mutation probes, each editing one file in memory and restoring it
  byte-identically afterwards. The scan reports an absence, so against a
  clean tree it cannot tell a real absence from a scan that finds nothing.
  Which test catches which mutation is therefore the substance:

  | mutation | scan | fixtures | detector | roots |
  |---|---|---|---|---|
  | a retired table re-added | fails | passes | passes | passes |
  | the scan's report path disarmed | passes | fails | passes | passes |
  | the detector's kind loop narrowed to one kind | passes | fails | fails | passes |
  | a pinned corpus root dropped | passes | fails | passes | passes |
  | a pinned corpus root renamed | passes | fails | passes | fails |
  | an unpinned corpus root renamed | passes | passes | passes | fails |
  | an unpinned corpus root dropped | passes | passes | passes | passes |

  The last row is the limitation, stated rather than closed. The corpus holds
  nine roots; the fixtures pin two — the design-doc tree and the shipped-skill
  tree, the two this milestone deletes from. Dropping any of the other seven
  narrows the scan with every test green. Pinning each would be one fixture
  per root, which enumerates a list rather than deciding a rule; what would
  actually close it is deriving the corpus instead of hand-maintaining it,
  and the reviewer notes say where that leads.

  For AC-2, renaming the cited symbol fails the resolution check alone, and
  deleting the citation fails the section check alone.

## Deferrals

- G-0674 — the `--principal` flag's help text cites an internal iteration
  label, found at this milestone's preflight. It lands in none of the files
  this work touches, so it takes its own change rather than an inline fix.

## Reviewer notes

- Two independent review rounds read the full change-set before this
  milestone closed. The first returned three blocking findings, the second
  five; all eight were corrected here. The scan moved into a non-test file so
  the firing-fixture and statement gates reach it, its detector is driven over
  every kind rather than one, AC-2 was narrowed to the surface it checks, and
  two dangling references to the deleted table were repaired in the shipped
  skill — one of which neither round found, and which surfaced only on a
  tree-wide sweep for the phrase.
- The scan's corpus is a hand-maintained list of nine roots, and dropping one
  the fixtures do not pin narrows the scan with nothing said. It is also a
  third statement of the repo's documentation tiers, beside the root
  `CLAUDE.md` and the documentation-hierarchy policy. Both problems have the
  same fix — derive the corpus from a tier-partitioned list — and no such
  list exists; G-0092 owns the tiering question.
- The `/archive/` skip the scan first carried is gone. It guarded nothing
  measurable (no archived file under any corpus root restates a set), and it
  compared the absolute path, so a checkout under any directory named
  `archive` — a worktree on an `archive/*` branch, for instance — silently
  disabled the whole scan. A frozen snapshot carrying a retired table is a
  real case for it, so if one ever appears the skip returns, comparing the
  repo-relative path.
- The design-doc citation check requires the tree to carry at least one
  Go-symbol citation of that shape. This is permanent and has no owner, and
  it earns that: it is the anti-vacuity guard for the resolution check, which
  would otherwise pass over nothing if the house shape moved and the pattern
  stopped matching.
- The section-scoping check cuts at the next heading of any level, so a
  sub-heading inserted above the citation reports the section as routeless
  while a human still reads the citation inside it. False red, never false
  green, and the message names the section.
- Declined: reporting a bad Go-symbol citation per citation rather than
  fatally. A citation pointing at a file that does not exist is already
  reported by the design-doc anchor rule with a clearer message, so the
  change would improve wording in a path another check already covers.
