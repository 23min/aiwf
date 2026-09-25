---
id: M-0333
title: Fence project guidance while preserving managed updates
status: in_progress
parent: E-0092
tdd: required
acs:
    - id: AC-1
      title: Guidance commits reject unrelated changes and allow managed outputs
      status: met
      tdd_phase: done
    - id: AC-2
      title: Guidance commits require a resolving entity trailer
      status: met
      tdd_phase: done
    - id: AC-3
      title: Guidance removals require a disposition block
      status: met
      tdd_phase: done
    - id: AC-4
      title: The always-on set above the ceiling fails the policy
      status: met
      tdd_phase: done
    - id: AC-5
      title: Finding codes and config fields remain discoverable outside guidance
      status: met
      tdd_phase: done
    - id: AC-6
      title: Tests reading development guidance are listed and reviewed
      status: met
      tdd_phase: done
---

## Goal

Fence changes to both hosts' project instructions and their canonical development guidance, measure each host's upfront load, and stop instruction documents serving as discoverability substitutes.

## Closes

- G-0676 — all four surfaces that let `CLAUDE.md` grow inside ordinary work: the commit seam (AC-1 to AC-3), the discoverability channel list and the principle text (AC-5), and the AC-evidence rule's scope (AC-6).

## Context

The development-guidance set is the root `CLAUDE.md`, the root `AGENTS.md`, `.guidance/project.md`, and the on-demand documents `.guidance/project.md` routes to. aiwf owns the managed blocks in the two entry points — `aiwf:guidance` (the Claude import, and the fragment rendered inline for Codex) and `aiwf:engineering-guidance` (the routing block) — and the files listed in `.guidance/.aiwf-owned`. Everything else in the set is handwritten: `CLAUDE.md` and `AGENTS.md` outside those blocks, and `.guidance/project.md`. Generated blocks are judged through their owning sources; expected rendered copies are not independent rule restatements. E-0092 defines the primed/on-demand split and the two ceilings.

G-0676 measured how `CLAUDE.md` grew and named the surfaces that let it. D-0091 decided that no AC is evidenced by a sentence pinned there, enforced diff-scoped. The skill-edit provenance backstop already judges commits in the base-to-HEAD range by what they touch and what trailer they carry, so the commit-seam gate takes its shape. Removing `CLAUDE.md` from the discoverability channel list leaves one config field documented nowhere else: `provenance.refuse_coauthors`, whose `commit-msg` refusal no shipped skill describes.

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

### AC-6 — Tests reading development guidance are listed and reviewed

A test that reads a document in the development-guidance set from the repository root — directly or through a helper in its package — fails the gate unless it is listed in the guidance-reader allowlist, whose entry names it a pin to retire, a relationship check, an absence check, or a test that names a guidance document without reading it. **Pass criterion**: a fixture test reading `CLAUDE.md` from the root is reported, and the same test listed is not; a test reading a `CLAUDE.md` in a fixture repository is not a reader; a test holds the list equal to the tree's readers in both directions, so a new reader fails and a retired pin's entry must go. **Edge cases**: a nested `CLAUDE.md` counts; a read through a helper is caught at the test calling it. The tests G-0676's floor command lists that read the guidance are listed. **Code references**: `internal/policies/guidance_readers.go`; D-0102 is the decision.

## Constraints

- Every guidance change this milestone makes carries an entity trailer and may include related sources and generated outputs together, including the two sentences the principle text loses.
- The ceiling constant is the measured current size, not a target. Lowering it is later milestones' work.
- No prose is cut here beyond the principle text; the gate lands on a file that still has everything in it.
- Measured figures in Validation carry their command (G-0668).

## Design notes

- Apply D-0102 to both host entry points, `.guidance/project.md` and the on-demand documents; the existing pins stand as list entries until E-0092 retires them.
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
- D-0102 — the decision AC-6 enforces, superseding D-0091
- D-0089 — external content ownership; personal instructions are measured separately

## Coverage notes

- (none)

## References

- G-0676, D-0091, D-0102, D-0089
- `internal/policies/skill_edit_provenance_backstop.go` — the gate's shape
- `internal/policies/guidance_readers.go` — the reader list AC-6 adds

## Release note

The `aiwf-check` skill documents the `commit-msg` hook: what it refuses as a commit is written, and the `provenance.refuse_coauthors` setting in `aiwf.yaml` that lists addresses a `Co-Authored-By:` line may not name. Everything else in this milestone is internal to aiwf's own repository and changes nothing a consumer runs.

## Decisions made during implementation

- ADR-0053: the commit rules ship in `aiwf check` for every aiwf repository. M-0350 moves AC-1 to AC-3's internal fence into the kernel and removes it; AC-4 and AC-6 stay internal.
- The ceiling follows the routing block: the project router is a required read for both hosts and counts toward the handwritten figure, and the generated pack index toward the aiwf-generated figure. A link from a host entry point's handwritten text must be classified in the read table; a link from the router is conditional unless the table says otherwise.
- A reference is a markdown link in any CommonMark form, or an `@` import in prose outside code; a path named any other way — in backticks, or in a sentence — is outside the ceiling's model.
- D-0102 supersedes D-0091. It holds the evidence rule over the whole guidance set by a list of every test that reads the guidance from the repository root, each entry naming it a pin, a relationship check, an absence check, or a test naming a guidance document without reading it. A new reader fails until an entry is written, so whether a new test is a pin is decided at review of that entry, not by a check of its assertions.
- AC-6 was restated from a scan of phrase-presence assertions to the reader list after its first implementation reached `met`; the phase history recorded under AC-6 belongs to that first implementation.
- The reader rule sees a guidance document named by a literal or a package constant, and a root resolved through a `repoRoot` helper or a literal climbing path. A root resolved any other way, a function reached through a package variable, and a path computed at run time are outside it — `TestPolicy_DesignDocAnchors` resolves links from the design documents into `CLAUDE.md` and is not listed.
- `provenance.refuse_coauthors` is documented in a `commit-msg` row of the `aiwf-check` skill's hook table, since `CLAUDE.md` was its only channel.
- The principle text rides with AC-5 as its own trailered commit and is held at review: a test pinning the principle would pin a heading of `CLAUDE.md`, which D-0102 bars.
- The evidence-rule statements in `CLAUDE.md` ride with AC-6 as trailered commits and cite D-0102; they are held at review.

## Validation

Environment: the aiwf devcontainer, Linux, `go1.25.11 linux/amd64`, on `milestone/M-0333-fence-project-guidance-while-preserving-managed-updates`; the audit base is the epic branch's fork point from `main`, `a61f3d8de`.

- `make check-fast` at `a40ba7453` — expected exit 0; observed exit 0, `golangci-lint` reporting `0 issues.`
- `AIWF_COVERAGE_BASE=a61f3d8de make coverage-gate` over the tree committed as `a40ba7453` — expected exit 0; observed exit 0 across the diff-scoped branch-coverage audit, the firing-fixture meta-gate, the skill-edit provenance backstop and the guidance fence.
- `go test -count=1 -run TestPolicy_GuidanceCeiling -v ./internal/policies/` at `a40ba7453` — the AC-4 measure, in whitespace-separated words. Observed:
  - `claude-code: handwritten primed 9549 (ceiling 9549), aiwf-generated 2393, required reads [.guidance/project.md .guidance/index.md]`
  - `codex: handwritten primed 9678 (ceiling 9678), aiwf-generated 2430, required reads [.guidance/project.md .guidance/index.md CLAUDE.md]`
- AC-5's design note: with `CLAUDE.md` out of the channel list, `go test -count=1 -run 'TestPolicy_FindingCodesAreDiscoverable|TestPolicy_ConfigFieldsAreDiscoverable' ./internal/policies/` reported `provenance.refuse_coauthors` as undocumented; with the `commit-msg` row in the `aiwf-check` skill it passes.
- G-0676's floor command at `da7709c78`, in a detached worktree with `CLAUDE.md` deleted (`go test ./internal/policies/ -count=1 | grep '^--- FAIL'`) — observed 20 failing tests. Nineteen are entries of `guidanceReaderList`: its fifteen pins, its three absence checks, and `TestPolicy_GuidanceCeiling`. The twentieth, `TestPolicy_DesignDocAnchors`, reaches `CLAUDE.md` by a path computed at run time and is outside the reader rule, as Decisions records.
- `AIWF_COVERAGE_BASE=v0.30.0 go test -count=1 -run TestPolicy_GuidanceFence ./internal/policies/` at `da7709c78` — the fence over the repository's history since `v0.30.0`, 3,571 commits; observed `--- FAIL: TestPolicy_GuidanceFence (2.64s)` with 377 `[guidance-fence]` lines, each a historical commit that mixed an instruction-file edit with other files. Only the pushed range is judged in CI, so this is a cost measurement, not a gate.

## Deferrals

- G-0710 — an assertion outside an `if` condition passes the shipped-prose ban.
- G-0711 — `gitops.BlobReader` reports a missing path containing whitespace as a parse error.
- The pins listed in `guidanceReaderList` are retired by M-0335 AC-3 as their passages move.

## Reviewer notes

- The read table's per-link cost for a host entry point's links was raised again at review; the maintainer chose that split, and it stands.
