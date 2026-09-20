---
id: M-0333
title: Fence project guidance while preserving managed updates
status: draft
parent: E-0092
tdd: required
acs:
    - id: AC-1
      title: Guidance commits reject unrelated changes and allow managed outputs
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

Fence changes to both hosts' project instructions and their canonical development guidance, measure each host's upfront load, and stop instruction documents serving as discoverability substitutes.

## Closes

- G-0676 — the four surfaces that let `CLAUDE.md` grow inside ordinary work; three close here, and the fourth (D-0091's scan) lands as AC-6.

## Context

The development-guidance set includes both host entry points and their canonical repository-development documents. Generated operating and language blocks are judged through their owning sources; expected rendered copies are not independent rule restatements. E-0092 defines the per-host upfront measurement and delivery prerequisite.

G-0676 measured how `CLAUDE.md` grew and named the surfaces that let it. D-0091 decided that no AC is evidenced by a sentence pinned there, enforced diff-scoped. The skill-edit provenance backstop already judges commits in the base-to-HEAD range by what they touch and what trailer they carry, so the commit-seam gate takes its shape. Both discoverability policies pass with `CLAUDE.md` removed from their channel list, so that removal costs no new doc mention anywhere.

## Acceptance criteria

### AC-1 — Guidance commits reject unrelated changes and allow managed outputs

A commit changing repository guidance alongside unrelated implementation files fails the gate. Related guidance sources, configuration and generated host outputs may change together as one logical update. **Pass criterion**: fixtures cover an unrelated code edit being refused, a source-plus-rendered-output update passing, and an unrelated file hidden beside that update being refused. Identify the finite owned path set from the completed delivery implementation; do not exempt arbitrary files by directory alone. Cover both host entry points, relocated development documents, renames and merge commits. Reuse the existing commit-range policy machinery.

### AC-2 — A CLAUDE.md commit without a resolving entity trailer fails the gate

A development-guidance commit carrying no `aiwf-entity` trailer, or one whose value resolves to no entity, fails the gate with the commit and the value in the detail. **Pass criterion**: fixture commits for the missing and the unresolvable case each produce one violation; one naming a real entity produces none. **Edge cases**: a narrow-width legacy id resolves after canonicalization; an archived entity resolves, since the loader spans the archive; a composite `M-NNNN/AC-N` resolves to its milestone. **Code references**: the same policy, resolving through `tree.Load` and `Tree.ByID`.

### AC-3 — A CLAUDE.md commit removing text without a disposition block fails the gate

A development-guidance commit whose diff removes lines and whose message body carries no disposition block fails the gate. A block is a `Removed:` line followed by a `Disposition:` line whose value is one of `copy of <path>`, `relocated to <path>`, `pointer to <id>`, or `deleted`. **Pass criterion**: a removing commit without a block, or with a `Disposition:` value outside the closed set, produces one violation; a pure addition needs no block; a removing commit with at least one well-formed block passes. The policy checks shape, not coverage; whether every removed passage has its block is held at review. **Edge cases**: a rewording is a removal plus an addition and needs a block; a block in a trailer position is still a block. **Code references**: the same policy.

### AC-4 — The always-on set above the ceiling fails the policy

Measure Claude and Codex separately using E-0092's upfront-project definition: automatic entry-point content plus project documents required before any task. A reference requiring a full upfront read cannot move those words outside the count. Personal/global material is reported separately; selected project guidance counts whenever its routing makes it upfront. Conditional task reads are reported separately. **Pass criterion**: fixtures cover both host entry points, transitive required reads, shared targets, cycles, missing targets, and a host above its ceiling. Unresolved routing is reported rather than silently omitted. Each host's initial ceiling equals its measured post-delivery size; later milestones lower it. The measure is an explicit model of configured routing, checked against live observations in M-0334, not a claim to inspect hidden model context.

### AC-5 — Finding codes and config fields stay discoverable without CLAUDE.md as a channel

The discoverability channel list contains neither host entry point nor relocated development guidance and both `finding-codes-are-discoverable` and `config-fields-are-discoverable` pass on the tree. **Pass criterion**: a test over the channel list asserts the entry is absent; the two policies' own tests stay green. **Edge cases**: the policies' fixtures, if they seed a `CLAUDE.md` channel, are updated. **Code references**: `internal/policies/discoverability.go`, `internal/policies/config_fields_discoverable.go` (comments only).

### AC-6 — A new test asserting a phrase in a CLAUDE.md file fails the gate

A test file added or modified in the gate's range that reads a document in the development-guidance set and asserts a string literal is present in its content fails the gate, naming the test. **Pass criterion**: a fixture test with such an assertion produces one violation; an absence assertion, and an expectation derived from code or from another artefact, produce none; the tests G-0676's floor command lists are carried in a grandfather ledger that only shrinks, in the shape of `firing_fixture_presence.go`'s ledger, and produce none while listed. **Edge cases**: a nested `CLAUDE.md` path counts; a helper that reads the file for a test that then asserts is caught at the assertion. **Code references**: `internal/policies/shipped_prose_assertion.go`, extended or given a sibling; D-0091 is the decision.

## Constraints

- Every guidance change this milestone makes carries an entity trailer and may include related sources and generated outputs together, including the two sentences the principle text loses.
- The ceiling constant is the measured current size, not a target. Lowering it is later milestones' work.
- No prose is cut here beyond the principle text; the gate lands on a file that still has everything in it.
- Measured figures in Validation carry their command (G-0668).

## Design notes

- Apply D-0091's scan to both host entry points and their relocated development documents; retain the shrinking-ledger approach for existing pins.
- Keep disposition blocks in commit bodies and reuse the commit-range machinery.
- Generated updates must be possible without hand-editing rendered blocks. Resolve ownership from the delivered implementation before writing the fence.
- Re-run discoverability tests with both host instruction files excluded before claiming the removal needs no replacement channel. Record command and result.

## Surfaces touched

- `internal/policies/` — two new policies and their tests; `discoverability.go`
- `Makefile`, `.github/workflows/go.yml` — the gate regex
- Both host entry points and canonical development guidance — source changes and owned regeneration kept together

## Out of scope

- Cutting, moving, or rewording any `CLAUDE.md` section; that is M-0335 through M-0337.
- A pre-push hook for the gate.
- A ceiling on any consumer's `CLAUDE.md`; the policy never ships.

## Dependencies

- E-0092's external delivery and repository migration prerequisite must be complete.
- G-0676 — the defect this closes
- D-0091 — the pin decision AC-6 enforces
- D-0089 — external content ownership; personal instructions are measured separately

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
