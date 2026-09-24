---
id: M-0333
title: Fence project guidance while preserving managed updates
status: in_progress
parent: E-0092
tdd: required
acs:
    - id: AC-1
      title: Guidance commits reject unrelated changes and allow managed outputs
      status: open
      tdd_phase: done
    - id: AC-2
      title: Guidance commits require a resolving entity trailer
      status: open
    - id: AC-3
      title: Guidance removals require a disposition block
      status: open
    - id: AC-4
      title: The always-on set above the ceiling fails the policy
      status: open
    - id: AC-5
      title: Finding codes and config fields remain discoverable outside guidance
      status: open
    - id: AC-6
      title: New prose-presence assertions over development guidance fail the gate
      status: open
---

## Goal

Fence changes to both hosts' project instructions and their canonical development guidance, measure each host's upfront load, and stop instruction documents serving as discoverability substitutes.

## Closes

- G-0676 — all four surfaces that let `CLAUDE.md` grow inside ordinary work: the commit seam (AC-1 to AC-3), the discoverability channel list and the principle text (AC-5), and the AC-evidence rule's scope (AC-6).

## Context

The development-guidance set is the root `CLAUDE.md`, the root `AGENTS.md`, `.guidance/project.md`, and the on-demand documents `.guidance/project.md` routes to. aiwf owns the managed blocks in the two entry points — `aiwf:guidance` (the Claude import, and the fragment rendered inline for Codex) and `aiwf:engineering-guidance` (the routing block) — and the files listed in `.guidance/.aiwf-owned`. Everything else in the set is handwritten: `CLAUDE.md` and `AGENTS.md` outside those blocks, and `.guidance/project.md`. Generated blocks are judged through their owning sources; expected rendered copies are not independent rule restatements. E-0092 defines the primed/on-demand split and the two ceilings.

G-0676 measured how `CLAUDE.md` grew and named the surfaces that let it. D-0091 decided that no AC is evidenced by a sentence pinned there, enforced diff-scoped. The skill-edit provenance backstop already judges commits in the base-to-HEAD range by what they touch and what trailer they carry, so the commit-seam gate takes its shape. Both discoverability policies pass with `CLAUDE.md` removed from their channel list, so that removal costs no new doc mention anywhere.

## Acceptance criteria

### AC-1 — Guidance commits reject unrelated changes and allow managed outputs

A commit changing repository guidance alongside unrelated implementation files fails the gate. Related guidance sources, configuration and generated host outputs may change together as one logical update. **Pass criterion**: fixtures cover an unrelated code edit being refused, a source-plus-rendered-output update passing, and an unrelated file hidden beside that update being refused. The owned set is the managed blocks and `.guidance/.aiwf-owned` named in Context; do not exempt arbitrary files by directory alone. Cover both host entry points, `.guidance/project.md`, the on-demand documents, renames and merge commits. Reuse the existing commit-range policy machinery.

### AC-2 — Guidance commits require a resolving entity trailer

A development-guidance commit carrying no `aiwf-entity` trailer, or one whose value resolves to no entity, fails the gate with the commit and the value in the detail. **Pass criterion**: fixture commits for the missing and the unresolvable case each produce one violation; one naming a real entity produces none. **Edge cases**: a narrow-width legacy id resolves after canonicalization; an archived entity resolves, since the loader spans the archive; a composite `M-NNNN/AC-N` resolves to its milestone. **Code references**: the same policy, resolving through `tree.Load` and `Tree.ByID`.

### AC-3 — Guidance removals require a disposition block

A development-guidance commit whose diff removes lines and whose message body carries no disposition block fails the gate. A block is a `Removed:` line followed by a `Disposition:` line whose value is one of `copy of <path>`, `relocated to <path>`, `pointer to <id>`, or `deleted`. **Pass criterion**: a removing commit without a block, or with a `Disposition:` value outside the closed set, produces one violation; a pure addition needs no block; a removing commit with at least one well-formed block passes. The policy checks shape, not coverage; whether every removed passage has its block is held at review. **Edge cases**: a rewording is a removal plus an addition and needs a block; a block in a trailer position is still a block. **Code references**: the same policy.

### AC-4 — The always-on set above the ceiling fails the policy

Measure Claude and Codex separately. A host's primed load is its entry point's automatically loaded content plus every document a reference requires reading in full before any task — today `AGENTS.md`'s preamble makes `CLAUDE.md` such a read for Codex. Count the handwritten primed words and the rendered fragment's words as two figures; the ceiling applies to the handwritten figure, and the fragment figure is reported for M-0339. Conditional task reads and personal/global material are reported separately. **Pass criterion**: fixtures cover both host entry points, transitive required reads, shared targets, cycles, missing targets, and a host above its ceiling. Unresolved routing is reported rather than silently omitted. Each host's initial ceiling equals its measured handwritten primed size; later milestones lower it. The measure is an explicit model of configured routing, checked against live observations in M-0334, not a claim to inspect hidden model context.

### AC-5 — Finding codes and config fields remain discoverable outside guidance

The discoverability channel list contains neither host entry point nor any on-demand development document, and both `finding-codes-are-discoverable` and `config-fields-are-discoverable` pass on the tree. **Pass criterion**: a test over the channel list asserts the entry is absent; the two policies' own tests stay green. **Edge cases**: the policies' fixtures, if they seed a `CLAUDE.md` channel, are updated. **Code references**: `internal/policies/discoverability.go`, `internal/policies/config_fields_discoverable.go` (comments only).

### AC-6 — New prose-presence assertions over development guidance fail the gate

A test file added or modified in the gate's range that reads a document in the development-guidance set and asserts a string literal is present in its content fails the gate, naming the test. **Pass criterion**: a fixture test with such an assertion produces one violation; an absence assertion, and an expectation derived from code or from another artefact, produce none; the tests G-0676's floor command lists are carried in a grandfather ledger that only shrinks, in the shape of `firing_fixture_presence.go`'s ledger, and produce none while listed. **Edge cases**: a nested `CLAUDE.md` path counts; a helper that reads the file for a test that then asserts is caught at the assertion. **Code references**: `internal/policies/shipped_prose_assertion.go`, extended or given a sibling; D-0091 is the decision.

## Constraints

- Every guidance change this milestone makes carries an entity trailer and may include related sources and generated outputs together, including the two sentences the principle text loses.
- The ceiling constant is the measured current size, not a target. Lowering it is later milestones' work.
- No prose is cut here beyond the principle text; the gate lands on a file that still has everything in it.
- Measured figures in Validation carry their command (G-0668).

## Design notes

- Apply D-0091's scan to both host entry points, `.guidance/project.md` and the on-demand documents; retain the shrinking-ledger approach for existing pins.
- Keep disposition blocks in commit bodies and reuse the commit-range machinery.
- Generated updates must be possible without hand-editing rendered blocks; the owned set in Context is the delivered implementation's.
- Re-run discoverability tests with both host instruction files excluded before claiming the removal needs no replacement channel. Record command and result.

## Surfaces touched

- `internal/policies/` — two new policies and their tests; `discoverability.go`
- `Makefile`, `.github/workflows/go.yml` — the gate regex
- Both host entry points, `.guidance/project.md` and the on-demand documents — source changes and owned regeneration kept together

## Out of scope

- Cutting, moving, or rewording any guidance; that is M-0336, M-0335 and M-0337.
- A pre-push hook for the gate.
- A ceiling on any consumer's `CLAUDE.md`; the policy never ships.

## Dependencies

- E-0094 — done; it delivered the owned set and routing this fence judges.
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

- D-0091 accepted before implementation; AC-6 enforces an accepted decision.
- The principle text rides with AC-5: removing "this file" from §"Engineering principles" is a separate trailered commit, and AC-5's test asserts the absence.
- D-0091's two statements in `CLAUDE.md` — the AC-evidence section and the substring-assertion bullet stating the extended scope — ride with AC-6 as separate trailered commits. They are held at review, since D-0091 rules out pinning them.

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
