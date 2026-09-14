---
id: M-0333
title: 'Fence CLAUDE.md: one commit per edit, a ceiling, no discoverability channel'
status: draft
parent: E-0092
tdd: required
acs:
    - id: AC-1
      title: A commit modifying CLAUDE.md with another path fails the gate
      status: open
    - id: AC-2
      title: A CLAUDE.md commit without a resolving entity trailer fails the gate
      status: open
    - id: AC-3
      title: A CLAUDE.md commit removing text without a disposition block fails the gate
      status: open
    - id: AC-4
      title: The always-on set above the ceiling fails the policy
      status: open
    - id: AC-5
      title: Finding codes and config fields stay discoverable without CLAUDE.md as a channel
      status: open
    - id: AC-6
      title: A new test asserting a phrase in a CLAUDE.md file fails the gate
      status: open
---

## Goal

Make a `CLAUDE.md` edit impossible to land except as its own trailered commit, cap the always-on set at its current size, and stop `CLAUDE.md` counting as a discoverability channel.

## Closes

- G-0676 — the four surfaces that let `CLAUDE.md` grow inside ordinary work; three close here, and the fourth (D-0091's scan) lands as AC-6.

## Context

G-0676 measured how `CLAUDE.md` grew and named the surfaces that let it. D-0091 decided that no AC is evidenced by a sentence pinned there, enforced diff-scoped. The skill-edit provenance backstop already judges commits in the base-to-HEAD range by what they touch and what trailer they carry, so the commit-seam gate takes its shape. Both discoverability policies pass with `CLAUDE.md` removed from their channel list, so that removal costs no new doc mention anywhere.

## Acceptance criteria

### AC-1 — A commit modifying CLAUDE.md with another path fails the gate

A commit in the gate's range whose diff touches a file named `CLAUDE.md` (root or nested) together with any other path fails the profile-driven gate, the detail naming the commit and the extra paths. **Pass criterion**: a fixture repo with one such commit produces exactly one violation; a commit touching only `CLAUDE.md` files produces none. **Edge cases**: merge commits are judged the way the backstop judges them; a rename of `CLAUDE.md` counts as touching it; a commit touching two nested `CLAUDE.md` files and nothing else passes. **Code references**: a new policy under `internal/policies/` in the shape of `skill_edit_provenance_backstop.go`, its test name added to the gate regex in the `Makefile` coverage-gate target and in `.github/workflows/go.yml`.

### AC-2 — A CLAUDE.md commit without a resolving entity trailer fails the gate

A `CLAUDE.md` commit carrying no `aiwf-entity` trailer, or one whose value resolves to no entity, fails the gate with the commit and the value in the detail. **Pass criterion**: fixture commits for the missing and the unresolvable case each produce one violation; one naming a real entity produces none. **Edge cases**: a narrow-width legacy id resolves after canonicalization; an archived entity resolves, since the loader spans the archive; a composite `M-NNNN/AC-N` resolves to its milestone. **Code references**: the same policy, resolving through `tree.Load` and `Tree.ByID`.

### AC-3 — A CLAUDE.md commit removing text without a disposition block fails the gate

A `CLAUDE.md` commit whose diff removes lines and whose message body carries no disposition block fails the gate. A block is a `Removed:` line followed by a `Disposition:` line whose value is one of `copy of <path>`, `relocated to <path>`, `pointer to <id>`, or `deleted`. **Pass criterion**: a removing commit without a block, or with a `Disposition:` value outside the closed set, produces one violation; a pure addition needs no block; a removing commit with at least one well-formed block passes. The policy checks shape, not coverage; whether every removed passage has its block is held at review. **Edge cases**: a rewording is a removal plus an addition and needs a block; a block in a trailer position is still a block. **Code references**: the same policy.

### AC-4 — The always-on set above the ceiling fails the policy

The always-on set above the ceiling fails the policy. The set is root `CLAUDE.md` plus every file its `@` imports resolve to inside the repository, followed transitively to the depth the harness follows; the count is whitespace-split words. A home-relative import is listed in the detail and not counted, because its content is the operator's (D-0089). **Pass criterion**: a fixture tree with a two-hop import chain counts all three files; one over the constant produces a violation naming the count and each file's share; the live tree passes at the constant this milestone sets, which is the count the policy itself reports at the milestone's start, recorded in Validation with its command. **Edge cases**: an import target that does not exist is a violation naming it; an `@path` inside a code span or fenced block is not an import; a cycle terminates; depth beyond the harness limit is ignored. **Code references**: a new policy under `internal/policies/` with a fixture tree in `testdata/`.

### AC-5 — Finding codes and config fields stay discoverable without CLAUDE.md as a channel

`discoverability.go`'s channel list contains no `CLAUDE.md` entry and both `finding-codes-are-discoverable` and `config-fields-are-discoverable` pass on the tree. **Pass criterion**: a test over the channel list asserts the entry is absent; the two policies' own tests stay green. **Edge cases**: the policies' fixtures, if they seed a `CLAUDE.md` channel, are updated. **Code references**: `internal/policies/discoverability.go`, `internal/policies/config_fields_discoverable.go` (comments only).

### AC-6 — A new test asserting a phrase in a CLAUDE.md file fails the gate

A test file added or modified in the gate's range that reads a path ending in `CLAUDE.md` and asserts a string literal is present in its content fails the gate, naming the test. **Pass criterion**: a fixture test with such an assertion produces one violation; an absence assertion, and an expectation derived from code or from another artefact, produce none; the tests G-0676's floor command lists are carried in a grandfather ledger that only shrinks, in the shape of `firing_fixture_presence.go`'s ledger, and produce none while listed. **Edge cases**: a nested `CLAUDE.md` path counts; a helper that reads the file for a test that then asserts is caught at the assertion. **Code references**: `internal/policies/shipped_prose_assertion.go`, extended or given a sibling; D-0091 is the decision.

## Constraints

- Every `CLAUDE.md` edit this milestone makes is its own commit with an entity trailer, including the two sentences the principle text loses.
- The ceiling constant is the measured current size, not a target. Lowering it is later milestones' work.
- No prose is cut here beyond the principle text; the gate lands on a file that still has everything in it.
- Measured figures in Validation carry their command (G-0668).

## Design notes

- D-0091 decides AC-6's scan and its diff-scoped enforcement with a shrinking ledger.
- The disposition block lives in the commit body, not in a trailer: trailer keys are a closed set the kernel enforces, and a new key would be a kernel change for a repo-local rule.
- The gate is CI-tier like the backstop, because an uncommitted edit carries no provenance to judge; a pre-push variant is a separate decision.
- Removal of the discoverability channel was measured before this milestone was planned: in a scratch worktree, delete the `filepath.Join(root, "CLAUDE.md")` line from `discoverability.go` and run the two discoverability tests; both pass.

## Surfaces touched

- `internal/policies/` — two new policies and their tests; `discoverability.go`
- `Makefile`, `.github/workflows/go.yml` — the gate regex
- `CLAUDE.md` — the principle sentence and one pointer line, each its own commit

## Out of scope

- Cutting, moving, or rewording any `CLAUDE.md` section; that is M-0335 through M-0337.
- A pre-push hook for the gate.
- A ceiling on any consumer's `CLAUDE.md`; the policy never ships.

## Dependencies

- G-0676 — the defect this closes
- D-0091 — the pin decision AC-6 enforces
- D-0089 — why home-relative imports are not counted

## Coverage notes

- (none)

## References

- G-0676, D-0091, D-0089
- `internal/policies/skill_edit_provenance_backstop.go` — the gate's shape
- `internal/policies/firing_fixture_presence.go` — the shrinking-ledger shape
- `internal/policies/shipped_prose_assertion.go` — the scan AC-6 extends

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
