---
id: M-0336
title: Remove duplicated guidance and move each rule to its audience's home
status: done
parent: E-0092
depends_on:
    - M-0349
    - M-0334
tdd: advisory
acs:
    - id: AC-1
      title: Development guidance has no independent copy of operating rules
      status: met
    - id: AC-2
      title: Development guidance cites entity ids only through links
      status: met
    - id: AC-3
      title: The generic Go conventions section is deleted
      status: met
    - id: AC-4
      title: The ceiling constant steps down to the post-deletion size
      status: met
    - id: AC-5
      title: Every removal commit traces to an inventory row and its decided home
      status: met
    - id: AC-6
      title: An audience move adds the rule to its new home, then removes the old copy
      status: met
---

## Goal

Remove from both hosts' development guidance the text that already loads from another home — the rules the shipped fragment carries, the generic Go conventions, and the entity-id asides — and move each rule the inventory places in the other home, so every rule ends once, where its audience says.

## Closes

- (none)

## Context

The shared operating fragment and the selected language packs under `.guidance/packs/` are canonical sources. Development guidance must not restate them. M-0349's inventory records which passages are copies and of what, and which rules belong to the other audience; this milestone deletes exactly those copies and makes exactly those moves, before M-0335 relocates what remains. A move into the shipped fragment or a shipped skill changes what every consumer reads, so it carries a CHANGELOG entry under E-0092 and follows the shipped-surface rules: no entity ids, no repository paths, no development history. From this milestone on, a test holds every removal commit to the inventory. Generated host copies are expected delivery outputs and are excluded from duplicate-authoring checks by ownership, not by ignoring an entire host file. D-0102 governs the evidence restriction.

## Acceptance criteria

### AC-1 — Development guidance has no independent copy of operating rules

For each operating anchor, no independently authored copy remains in either host's repository-development guidance. Exclude only the exact generated operating blocks; the generated Codex block legitimately carries the shared source's rules. **Pass criterion**: fixtures distinguish an expected rendered block from an extra handwritten copy in the same file, and the live tree passes. Phrase checks cover only known anchors; semantic duplication remains a review question.

### AC-2 — Development guidance cites entity ids only through links

No id-shaped token (gap, epic, milestone, decision, or ADR) appears in handwritten development guidance outside a markdown link destination. **Pass criterion**: a structural scan in the shape of the `skill-body-id` check, with link carriers masked, reports each stray id; on the tree it reports none. **Edge cases**: a placeholder in the letter-N form inside backticks is syntax, not a citation; a command example cites a placeholder, never a real id. **Code references**: `internal/check/skill_body_id.go` for the masking; the new scan under `internal/policies/`.

### AC-3 — The generic Go conventions section is deleted

Remove from `CLAUDE.md` the Go conventions the selected Go pack (`.guidance/packs/go/cobra/guide.md`) already carries, as the inventory marks them, retaining the pack and its route for both hosts. Record the removal commit and the canonical source in its disposition. Retain aiwf-specific conventions, and any rule the inventory marks as a genuine project override of the pack. This is observational evidence, supported by the measured reduction.

### AC-4 — The ceiling constant steps down to the post-deletion size

Lower each host's ceiling to its measured post-deletion upfront size. The current tree passes at the lowered ceiling. Record the ceiling measure's command and each host's figures before and after the deletion; the ceiling's own test already pins that a load above it fails, so the step-down adds no fixture.

### AC-5 — Every removal commit traces to an inventory row and its decided home

Each commit that removes text from development guidance or the shipped fragment names, in its disposition block, the inventory row it carries out, and the block agrees with the row's decision: the same kind of disposition and the same destination. **Pass criterion**: the inventory test (M-0349's design notes) reads each disposition block in E-0092's commit range and the inventory table in M-0349's body through the loader, and reports a removal that names no row, names a row that does not exist, disagrees with the row's decision, or names a destination that does not exist; fixtures cover each, and the tree passes. Whether the destination states the rule is held at review, since D-0070 and D-0102 bar pinning guidance wording. **Code references**: the inventory test under `internal/policies/`, reading commits through the existing commit-range machinery.

### AC-6 — An audience move adds the rule to its new home, then removes the old copy

A rule the inventory moves between homes — `move to the shipped fragment`, `move to the shipped skill <path>` or `move to repository guidance <document>` — moves in two commits: the first adds it to its new home, the second removes it from the old with `Disposition: copy of <new home>`, so it is briefly in both homes and never in neither. The instruction-file fence keeps shipped sources out of a commit that changes handwritten instructions, so the two halves cannot share a commit. A commit that changes the shipped fragment regenerates the managed `AGENTS.md` block with it. **Pass criterion**: AC-5's check also reports a move whose removal commit has no earlier commit in E-0092's range changing its destination file; a fixture covers it, and the tree passes.

## Constraints

- Deletion of copies and audience moves only; what stays or moves is not reworded.
- Each guidance commit may include its related source and generated outputs together, with a `copy of <path>` disposition per removed passage; an id aside whose reasoning is nowhere else goes to the entity that owns it first, and its block says `relocated to`; an audience move's removal says `copy of <new home>`, as AC-6 sets out. Every block names its inventory row.
- One row appended to the iteration log when this lands.
- A pin on a passage this milestone deletes is retired with its reason, and its entry leaves the guidance-reader list.

## Design notes

- D-0102 shapes AC-1 and AC-2 as absence and structure checks; neither can hold a sentence in place, and each is listed as a guidance reader.
- AC-3 is observational, so this milestone runs under `tdd: advisory`; its other criteria carry tests regardless.
- AC-5's check maps each inventory decision to the block dispositions it allows: `delete as copy of` and the three audience moves to `copy of`, `move on demand to` to `relocated to`, `pointer to` to `pointer to`, and `merge into` and `tighten` to `reworded`. Until M-0338 AC-3, a row not yet carried out is not a finding.
- A block names its row as the first word of its `Removed:` value, ahead of the quoted text: `Removed: kiss "…"`. An edit AC-2 requires names the id as the second word, `Removed: write-executable G-0491`, with any disposition the fence accepts; AC-5's check confirms that id is a token of the row's text at the inventory's freeze and that a destination it names resolves, and holds it to no decision, while every other block must agree with its row's decision. The check reads blocks only from commits after this milestone's promote to `in_progress`, since earlier blocks in E-0092's range predate the label.

## Surfaces touched

- Both host entry points and `.guidance/project.md`, respecting generated ownership.
- The existing anchor policy and structural reference scan.
- `internal/skills/embedded-guidance/aiwf-guidance.md`, the shipped skills a rule moves into, and `CHANGELOG.md`.
- The inventory test under `internal/policies/`, which this milestone extends with the reconciliation.

## Out of scope

- The pointer cut (M-0337).
- Rewording or shortening the shipped fragment's rules; the inventory records any such suggestion for a later epic.

## Dependencies

- M-0349 — the inventory records which passages are copies, where their canonical home is, and which rules move between audiences
- M-0334 — the baseline, recorded before any decision is carried out

## Coverage notes

- (none)

## References

- D-0102, ADR-0054
- `internal/check/skill_body_id.go` — the link-masking shape

## Release note

The always-on guidance `aiwf init` and `aiwf update` install adds four clauses to the rules it already states: any deviation from a declared-sequence gate's list aborts the sequence and re-gates; a remote-branch delete is an outward action that never batches; a one-at-a-time decision list ends with "something else", the pick is confirmed in one line, and "Q&A" or "interview me" asks for the same; and what you noticed is surfaced, whether it is fixed in place or filed. The `aiwf-promote` skill says a milestone's `tdd:` setting never waives the evidence an AC needs and that the obligation binds at the promote. The `aiwf-reallocate` skill says how to rule out a stale branch before reallocating on a trunk collision, that the new id is allocated against every local branch and remote-tracking ref as well as trunk, and that the verb rewrites mentions of the old id in body prose rather than reporting them. The `id-rename-untrailered` finding and its hint point at that finding's entry in the `aiwf-check` skill rather than at a section of aiwf's own `CLAUDE.md`.

## Decisions made during implementation

- Where a disposition block names its inventory row, and how an edit M-0336 AC-2 requires is told apart from one that carries out the row's decision, as Design notes state. Decided by the maintainer, 2026-10-10.
- M-0336 AC-1's check reads a copy as every fragment of an operating anchor within one paragraph or list item, and the `body-prose-id` anchor gains the fragment's "canonical width", so naming the check as an enforcement pointer is not read as restating its rule; the presence policy reads the stronger anchor too. Decided by the maintainer, 2026-10-10.
- The shipped fragment is held by M-0336 AC-5 through its words: those at the inventory's freeze must survive, in order, at HEAD, since every fragment row is kept and the fence does not judge the fragment. Decided by the maintainer, 2026-10-10.
- The handwritten routed documents are defined once, in `projectguidance`, and the fence, both development-guidance checks and the guidance-readers policy read that definition, so an output aiwf owns is never judged as handwritten. Decided by the maintainer, 2026-10-10.
- The `aiwf-reallocate` skill is corrected where it is the destination of a passage this milestone removed: the allocation scope the code runs, that the verb rewrites body-prose mentions rather than reporting them, and that a trunk rename lacking an aiwf rename trailer is what makes a stale branch collide, since `aiwf retitle` and `aiwf rename` on trunk are recognised. The `id-rename-untrailered` hint keeps saying the verb rewrites cross-references, which the verb's tests pin. Decided by the maintainer, 2026-10-10.
- Defects found here outside what this milestone moved are recorded as gaps, under Deferrals: the `aiwf-add` skill's step 1 understating the allocation scope, its claim that `body-prose-id` resolves only the working tree and trunk, and the design documents and `AllocateID`'s comment stating the same narrower allocation scope. Decided by the maintainer, 2026-10-10 and 2026-10-11.
- M-0349's inventory note on the `id-allocation` row is corrected: the verb rewrites body-prose mentions as well as frontmatter references, as `CLAUDE.md` said. Decided by the maintainer, 2026-10-10.

## Validation

Environment: the aiwf devcontainer, Linux, `go1.25.12 linux/amd64`, on `milestone/M-0336-remove-duplicated-guidance-and-move-each-rule-to-its-audience-s-home`. The gates ran at `12e3c978b`; each primed load was measured in a `git archive` export of the commit named beside it.

- `make check-fast` at `12e3c978b`. Expected exit 0. Observed exit 0.
- `aiwf check` at `12e3c978b`, built from that tree. Expected no error. Observed 0 errors and 15 warnings: `acs-tdd-audit` on each met criterion of M-0349 and of this milestone, both `tdd: advisory` and promoted without phases; two terminal decisions awaiting the archive sweep and its summary; and `provenance-untrailered-scope-undefined`, since the milestone branch has no upstream.
- M-0336 AC-1: `TestPolicy_GuidanceNoOperatingCopies` passes on the tree, and `TestOperatingCopyViolations` pins its rules — a rendered block carrying a rule passes while a handwritten copy in the same file fails, fragments count only within one paragraph or list item, whichever CommonMark list marker opens it, and are read across line wraps and indented continuations, and the router and the handwritten documents it routes to are guidance while a routed output aiwf owns is not.
- M-0336 AC-2: `TestPolicy_GuidanceIDsThroughLinks` passes on the tree, and `TestGuidanceIDViolations` pins its rules — an id in a link destination passes and one in its text fails, the canonical placeholder passes, and a code span is in scope.
- M-0336 AC-3, an observation. `bd98a3a17` removes the copies of the Go pack and the code-health pack, and each of its disposition blocks names the pack it copies. `.guidance/packs/go/cobra/guide.md` remains, and both hosts' managed routing blocks are byte-identical before and after this milestone. `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/`: expected the handwritten primed load to fall by the removed rows' words and the heading of the subsection they emptied. Observed 7995 to 7779 words for Claude and 8124 to 7908 for Codex, from `5f0ee60bf` to `bd98a3a17`: the rows' 213 words and the 3 of `### Error handling`.
- M-0336 AC-4: the same command. Before the deletions, at `aba8c2732`, it measures 9647 words for Claude and 9776 for Codex; after them, at `630edc920`, 7801 and 7930, which are the lowered ceilings, and the test passes there and at `12e3c978b`.
- M-0336 AC-5 and AC-6: `TestE0092Inventory` reconciles every disposition block since `aba8c2732` with its inventory row, reports a malformed block, and requires the fragment's words at the freeze to survive at HEAD; it passes. `TestInventoryReconcileFindings` and `TestFragmentLossFindings` pin each rule. A fresh reviewer's mutation run at `5b9fc3d0b` replaced one line at a time across the two development-guidance checks, the shared set functions and the reconciliation, and re-ran `go test -count=1` over `./internal/policies/`, `./internal/projectguidance/` and `./internal/check/`. Six mutants that change behaviour survived it — list numbers past nine, the space a list marker needs, blank lines of spaces or ending in CRLF, a last paragraph with no closing newline, a block naming only its row, and an id edit read from a quoted passage's last word — and each now has a case that fails it. The rest that survived leave behaviour unchanged, or sit in the live test's wiring, which has no fixture: the trunk exclusion, the malformed-block report, the call to the fragment rule, and the move ordering's look-back ending at the parent of the removal commit, which no commit can tell apart from one ending at the commit itself, since the fence refuses a commit that changes an instruction file and a shipped home together.

## Deferrals

- G-0732 — the `aiwf-add` skill's step 1 says allocation scans only the working tree and trunk.
- G-0733 — the `aiwf-add` skill says `body-prose-id` resolves only the working tree and trunk.
- G-0734 — the design documents and `AllocateID`'s comment say allocation reads only two trees.

## Reviewer notes

- Obligations this milestone adds for later changes, counted from `git diff b33921f03..HEAD`:
  - two standing policies, `guidance-no-operating-copies` and `guidance-ids-through-links`, each a ban paid once rather than per subject; owner: this repository's development-guidance rules; no retirement, since they hold M-0336 AC-1 and AC-2 for as long as the guidance exists;
  - every anchor added to `m0211OperatingAnchors` is now also an absence rule over development guidance, which the anchor type's comment states;
  - every entity citation in development guidance is a link, and every link from primed text needs a row in the ceiling's read table, ten of them added here; owner: the ceiling policy;
  - the ceilings at 7801 and 7930 words and the fragment's line budget at 172, each with no headroom, so a primed addition needs a removal or a deliberate raise; owner: M-0337, which steps the ceilings down further;
  - the removal reconciliation and the fragment rule in the inventory test, requiring every disposition block since `aba8c2732` to name its row, the destinations it names to keep existing, and the fragment's frozen words to survive; owner: E-0092, whose wrap deletes the test once M-0338 records its last run;
  - `check.ParseDispositionBlocks` and `check.WordsSurviveInOrder` are exported; the fence calls both, so they outlive the inventory test.
- Declined:
  - a denser rewrite of the copy check, and a class rule replacing the read table's per-link rows for ADRs and decisions;
  - reading a table or a nested list as more than one paragraph: a false match there fails loudly;
  - taking the flagged token from the kernel scan's message by pattern, which the id check's cases pin;
  - three clauses of removed passages restated nowhere else, each emphasis or an example: "especially one you just authored", the dependency-injection example, and "a Go test under `internal/policies/`";
  - dated documents that cite `CLAUDE.md` sections removed here or earlier — the legal-workflows design tables, an initiative and a gap body — which record their own time;
  - rewriting commit messages already superseded by a later commit's body.
- Handed to M-0335, whose tighten rows carry them: the `AGENTS.md` preamble and the router still name `CLAUDE.md` as the home of approval-gate rules, and the AC-promotion section now holds one sentence with no lead-in.
- The review's result: the independent review covered code quality, design and subtraction over the full change set, then ran further full passes until one found no defect in the code; that pass's remaining findings were evidence and fixtures, which are fixed, and by the maintainer's decision the close was a fresh confirmation scoped to them rather than another full pass. Its findings — one mutant still surviving and the Validation sentence recording the mutation run — are fixed: the mutant fails its new case, and Validation records the run as made. Verdict: approve.

