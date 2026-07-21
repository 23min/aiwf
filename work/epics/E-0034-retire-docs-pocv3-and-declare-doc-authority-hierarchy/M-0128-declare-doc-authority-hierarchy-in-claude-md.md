---
id: M-0128
title: Declare doc-authority hierarchy in CLAUDE.md
status: in_progress
parent: E-0034
depends_on:
    - M-0127
tdd: none
acs:
    - id: AC-1
      title: Documentation hierarchy section tags every active docs/ subtree
      status: met
---

## Goal

Add a "Documentation hierarchy" section to CLAUDE.md naming each active `docs/` subtree by authority tier (normative / forward-looking / exploratory / archival). The section is written *once* against the post-M-0127 layout, so the tier labels match the actual path layout a reader will see.

## Context

This is the move G-0092 originally proposed as the minimum (CLAUDE.md table). Writing it before M-0127 would have meant either two passes of the same content or labels that don't match the directory layout; writing it now means once-and-correct.

The mechanical evidence shape — a structural assertion under `internal/policies/` that the named section exists in CLAUDE.md and lists each active `docs/` subtree — is drafted at `aiwfx-start-milestone` time.

## Acceptance criteria

### AC-1 — Documentation hierarchy section tags every active docs/ subtree

CLAUDE.md gains a `## Documentation hierarchy` section naming every currently-active `docs/` subtree and top-level narrative file group, each tagged with exactly one of four closed-set tiers: **normative**, **forward-looking**, **exploratory**, **archival**.

Tier assignment (post-M-0127 layout):

- **Normative** — `docs/adr/`, `docs/design/`, and the top-level operational references (`architecture.md`, `overview.md`, `workflows.md`, `skill-author-guide.md`, `migration/`). Current-truth, kept in lockstep with the code.
- **Forward-looking** — `docs/initiatives/`. Captured ideas awaiting promotion to a real epic/gap entity.
- **Exploratory** — `docs/explorations/` (including `loom/`, `surveys/`), `docs/research/`, `working-paper.md`. Synthesis/thesis/proposal genre; not kernel-binding regardless of internal rigor.
- **Archival** — `docs/archive/` (includes `docs/archive/pocv3/`). Frozen historical snapshot per ADR-0004.

A structural test under `internal/policies/` parses CLAUDE.md's heading hierarchy, locates the `## Documentation hierarchy` section by name, and asserts each subtree name and each top-level narrative file above appears within it, tagged with a valid tier label from the closed set. Per the epic's own out-of-scope note, this is a fixed snapshot assertion against the tree as it exists today — not a live drift check against `docs/`'s actual layout (that's G-0092's deferred kernel-rule follow-on).

## Out of scope

- Drift-checking that hierarchy labels match `docs/`'s actual layout at runtime (deferred to G-0092's full kernel-rule follow-on, listed as out of scope in the epic spec).
- Per-tree `_AUTHORITY.md` marker files (option 2 from G-0092's gap body; not the chosen layer per the planning conversation).

## Work log

### AC-1 — Documentation hierarchy section tags every active docs/ subtree

CLAUDE.md's `## Documentation hierarchy` section added before `## What aiwf commits to`, naming all 7 active `docs/` subtrees under the four closed-set tiers; `PolicyM0128DocumentationHierarchy` parses the section structurally and asserts subtree coverage + closed-set tier vocabulary, with firing-fixture rows covering all five violation classes · commit 3241453e · tests 1/1

### Post-review correction (AC-1)

Independent review (dispatched at wrap) approved with one non-blocking finding: AC-1's own text promises the section names top-level narrative files alongside `docs/` subtrees, but the mechanical test only pinned subtree coverage, leaving a narrative-file omission undetectable. Fixed by extending `PolicyM0128DocumentationHierarchy` to also assert each of the 5 narrative files, with a matching firing-fixture update · commit a6f6f6d1 · tests: full `internal/policies` package green, `make lint` 0 issues, 100% statement coverage on the policy function

## Decisions made during implementation

None rising to an ADR/`D-NNN`-worthy architectural decision. The tier-assignment judgment call (which subtree belongs to which of the four tiers, and where the two ambiguous cases — `docs/research/`'s "defended-position" thesis arc and `docs/migration/`'s how-to content — land) was surfaced to and confirmed by the operator before AC-1 drafting; recorded here rather than as a standalone `D-NNN` since it's milestone-scoped content, not a durable cross-cutting decision.

## Validation

`make lint` (golangci-lint, full set) — 0 issues. `go build ./...` — green. `go test ./...` — green (one `TestWorktreeRitualsCheckHook_NotAWorktreeExitsZeroSilently` "text file busy" flake observed once in the full-suite run, the same pre-existing flake documented in M-0127's Work log; confirmed unrelated to this diff via 3x isolated re-run and a clean re-run of the full `internal/policies` package). `make coverage-gate` — diff-scoped branch coverage clean (100% on the new policy function), firing-fixture presence clean (all 5 violation classes covered by the 3 fixture rows), no stale allowlist entries. `aiwf check` — clean except the two pre-existing, already-tracked findings unrelated to this milestone (the `promote-on-wrong-branch` reused-id pattern on M-0126, and the expected no-upstream provenance advisory).

Independent review: one fresh-context reviewer dispatched over the full milestone diff (small, single-AC change-set; no concern-slicing needed at this scale). Verdict **approve**, with one non-blocking finding fixed in the "Post-review correction" Work log entry above and confirmed mechanically (re-running the affected tests, `make lint`). `wf-rethink` (design-quality lens) was not run — this milestone introduced no new module/package boundary, core abstraction, or data model, only a CLAUDE.md content section and a structural test following an existing repo pattern (the same shape as the `m0134`/`m0132`/`m0228` CLAUDE.md-content policies).

Doc-lint sweep (scoped to this milestone's change-set): clean — 0 findings, all 15 paths cited in the new section verified to resolve on disk.

## Deferrals

- (none) — AC-1's scope landed in full; no residual M-0128 work was punted.

## Reviewer notes

- **The mechanical test is a fixed snapshot, not a live drift check.** `PolicyM0128DocumentationHierarchy` pins the `docs/` subtree and narrative-file list as they exist today; it does not re-derive the list from `docs/`'s actual directory contents at test time. If a new `docs/` subtree is added later without a CLAUDE.md update, nothing catches the drift automatically. This is deliberate — the epic's own out-of-scope note defers live drift-checking to G-0092's kernel-rule follow-on — not an oversight.
- **`docs/research/` and `docs/migration/` were the two ambiguous tier calls.** `docs/research/` (a "defended-position" thesis arc) went to **exploratory** rather than normative, since CLAUDE.md doesn't cite it as a binding kernel commitment despite its internal rigor. `docs/migration/` (how-to guides for importing from prior systems) went to **normative** alongside the other operational references, since it's current-truth operational content, not a proposal. Both calls were surfaced to and confirmed by the operator before implementation.

## Dependencies

- M-0127 (Relocate) — done. The hierarchy section labels must match the post-Relocate layout.

## References

- **E-0034** — parent epic.
- **G-0092** — superseded by E-0034; this milestone is the concrete realization.
