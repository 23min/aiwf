---
id: M-0171
title: Area field on root kinds and aiwf.yaml areas block with validation
status: in_progress
parent: E-0043
tdd: required
acs:
    - id: AC-1
      title: Five root kinds accept optional area frontmatter field; absent parses clean
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf.yaml areas block declares member set + optional default label, validated
      status: met
      tdd_phase: done
    - id: AC-3
      title: Milestone and AC derive area from parent epic at load, exposed in model
      status: met
      tdd_phase: done
    - id: AC-4
      title: 'With no areas block the area field is inert: parses but nothing validates'
      status: met
      tdd_phase: done
---
## Goal

Add the optional `area` frontmatter field to the five root entity kinds (epic, ADR, gap, decision, contract) and the `aiwf.yaml: areas` block that declares the closed member set. This is the data + config foundation the rest of E-0043 builds on; the flat, globally-unique id space is untouched.

## Context

Per E-0043's converged design, `area` is a validated grouping tag, not a directory axis or an id-space change. This milestone makes the field *exist and parse* and the config block *exist and validate* — it does not yet add the `area-unknown` check finding (next milestone), the write path, or any read surface. Until the `areas` block is present the field is inert.

Milestones and ACs do **not** store `area`; they derive it from their parent epic, so "milestone disagrees with its epic" is unrepresentable rather than policed.

## Acceptance criteria

### AC-1 — Five root kinds accept optional area frontmatter field; absent parses clean

The five root kinds (epic, ADR, gap, decision, contract) accept an optional `area:` string field in frontmatter. Absent or empty parses cleanly — no error, and no default value is written back.

Forward-compat note (folded from the fifth candidate criterion in the original draft): the field site documents that a pre-`area` binary rejects a file using `area` via the generic `KnownFields(true)` strict-decoder window — the same behavior as every prior frontmatter field, not special to `area`.

Evidence: a parse test over all five kinds (present / absent / empty); a strict-decode test asserting an unknown sibling field is rejected; a structural assertion that the field-site doc comment names the forward-compat window.

### AC-2 — aiwf.yaml areas block declares member set + optional default label, validated

`aiwf.yaml` accepts an `areas` block: a closed member set plus an optional `default:` key that is a display label only (never a member of the tag set, never written to an entity). Schema validation rejects a malformed block (non-string members, wrong shape) at config-load time.

Evidence: config-load tests over a valid block; an absent block; a quoted-numeric member accepted (string) while an unquoted non-string scalar is rejected; and a malformed-block table — non-string member, null member, non-sequence members, whitespace-padded member, whitespace-only / whitespace-padded default, default colliding with a member, default with no members — each rejected with a clear error.

### AC-3 — Milestone and AC derive area from parent epic at load, exposed in model

A milestone (and an AC) resolves its `area` by deriving from its parent epic at load time — the field is not stored on the milestone (the loader blanks any stored value) — and the derived value is exposed through the loaded model so downstream read surfaces can group without re-deriving.

Evidence: a loader test asserting a milestone under an epic carrying an `area` reports that area, one under an untagged epic reports none, and an orphan/nil parent yields none; that a stored milestone `area` is blanked at load and auto-stripped on serialize; and `ResolvedAreaByID` resolving a composite AC id to the parent epic's area through one seam (the AC-derivation surface added after review, so the two downstream consumer milestones do not re-derive it).

### AC-4 — With no areas block the area field is inert: parses but nothing validates

With no `areas` block in `aiwf.yaml`, the `area` field is inert: present values parse but nothing validates or groups. (Validation lands as the `area-unknown` finding in the next milestone.)

Evidence: a metamorphic check test — two trees identical except for area values on the root kinds produce exactly the same findings, none area-related — which goes red the instant any rule reads `area` when no block is declared (vacuity-proven by a temporary mutation).

## Constraints

- **Commitment #2 (stable flat ids) untouched.** No change to the allocator, references, trailers, `aiwf history`, or `reallocate`. `area` never reshapes the on-disk tree, so the loader and the ADR-0004 archive convention are untouched.
- **Single source of truth** for the member set is `aiwf.yaml: areas`; no parallel registry.
- **Zero migration.** Every existing entity (no `area`) keeps parsing and rendering exactly as today.

## Out of scope

- The `area-unknown` check finding (next milestone).
- The `aiwf add --area` write path and completion (later milestone).
- Any read-surface filter or grouping (later milestones).

## Dependencies

- E-0043 epic spec (committed). No prior milestones — this is the foundation.

## References

- [E-0043 epic](epic.md) — converged design and scope.
- [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md) — the gap this epic implements.

## Work log

Implementation landed in a single feat commit `221e4b58` on the milestone branch; the per-AC TDD phase timeline is in `aiwf history M-0171/AC-<N>`.

- **AC-1** — `area` field added to the shared `Entity` struct with the forward-compat doc note; five-kind parse + strict-decode + doc-comment tests. · `221e4b58`
- **AC-2** — `Areas` config block with `UnmarshalYAML` (node-type rejection of non-string members) and a hardened `validate()`; round-trip / absent / quoted-numeric + 8-case malformed table. · `221e4b58`
- **AC-3** — `Tree.ResolvedArea` (root → own, milestone → parent epic) + loader blank-at-load + `Tree.ResolvedAreaByID` composite-AC seam via `entity.CompositeRoot`; full-branch + clear-at-load + AC-rollup tests. · `221e4b58`
- **AC-4** — metamorphic inert guard in the check package (area-tagged vs untagged trees yield identical findings); vacuity-proven. · `221e4b58`

## Validation

- `make check-fast` (go vet + all `internal/...` tests + golangci-lint, full set): green.
- `go build ./...` (CGO_ENABLED=0): green.
- `aiwf check` (worktree-built diag binary): 0 errors (only the benign `provenance-untrailered-scope-undefined` warning — no upstream configured in the worktree).
- Diff-scoped coverage: every new conditional branch (UnmarshalYAML node-type, `validate()` whitespace/default arms, `ResolvedArea` nil/milestone/orphan/root) has a traversing test; the CI coverage-gate confirms on push.

## Reviewer notes

- **Independent two-lens review (wrap step 2), run early at the operator's request.** A fresh-context `reviewer` subagent (`wf-review-code`) returned REQUEST-CHANGES; `wf-rethink` on the `tree.go` area-resolution unit returned KEEP. All findings were addressed before closure:
  - **B1** — AC-3 claimed "milestone *and AC* derive area" but AC resolution was prose-only → added `Tree.ResolvedAreaByID` (composite-AC seam) + test.
  - **B2** — `areas` validation silently coerced non-string members → `UnmarshalYAML` rejects non-`!!str` nodes.
  - **B3** — validation was not whitespace-normalized (padded/whitespace-only members and default passed) → rejected with cases.
  - **N1** strict-decode behavior test; **N2** default-with-no-members rejected; **N3** null member rejected; **rethink-#2** co-located `omitempty` note at the `Area` field.
- **AC-2 and AC-3 were force-demoted (`met → open`, sovereign `--force`) and re-promoted** after the review hardened their evidence — visible as `met → open → met` in `aiwf history`. The premature `met` (before the independent pass) is exactly the discipline the re-validation corrects.
- **Design note.** Option 2 (clear stored milestone `area` at load) was chosen deliberately over Option 1; the loader blanks a milestone's stored value and `omitempty` auto-strips it on the next write-verb. A stored milestone `area` is invalid by design, so the auto-strip is cleanup, not data loss.
- A **skills-lifecycle gap** (start-milestone review framing + the start↔wrap commit-timing contradiction) was filed and then corrected during this wrap, once `aiwfx-wrap-milestone` step 2 revealed the independent review *is* prescribed. It was filed on trunk; its `discovered_in: M-0171` carries the back-reference, so it is not id-linked from this branch (where it does not yet resolve).

## Deferrals

None. The `area-unknown` finding, the `aiwf add --area` write path, and read-surface filter/grouping are out of scope by design — they are E-0043's subsequent milestones M-0172 through M-0175 (already planned), not deferrals.
