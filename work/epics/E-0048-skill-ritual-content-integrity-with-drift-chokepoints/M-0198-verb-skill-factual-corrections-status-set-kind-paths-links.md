---
id: M-0198
title: Verb-skill factual corrections (status set, kind, paths, links)
status: in_progress
parent: E-0048
depends_on:
    - M-0197
tdd: advisory
acs:
    - id: AC-1
      title: aiwf-check documents the four-status AC set, kernel-derived
      status: met
    - id: AC-2
      title: aiwf-archive drops findings as an archivable kind
      status: met
    - id: AC-3
      title: aiwf-contract cites the real recipe path and complete cancel FSM
      status: met
    - id: AC-4
      title: aiwf-authorize provenance-model doc-link resolves
      status: met
    - id: AC-5
      title: aiwf-add example is self-consistent and cites sections not lines
      status: met
---
## Goal

Five `aiwf-*` verb skills carry verified factual errors that mislead an AI
assistant or operator reading them for authoritative guidance:

- `aiwf-check` understates the acceptance-criterion status set as three
  (`{open, met, cancelled}`); the kernel set is four —
  `{open, met, deferred, cancelled}` — and `deferred` is a live terminal AC
  state the wrap rituals use.
- `aiwf-archive` lists "findings" as an archivable kind; `findings` is not an
  entity kind, and `--kind` accepts only `epic, contract, gap, decision, adr`.
- `aiwf-contract` points contributors at a nonexistent `tools/`-prefixed recipe
  path, and its cancel description omits the `deprecated → retired` cancel case
  the kernel implements.
- `aiwf-authorize` links the provenance-model doc with a relative depth that
  resolves to a nonexistent path.
- `aiwf-add` has a self-contradictory example (identical ids) and cites design
  docs by fragile pinned line numbers.

This milestone corrects all five and pins each with a structural test under
`internal/policies/` — **source-derived** where the skill enumerates a
kernel-defined set (AC statuses, contract cancel targets), so the enumeration
becomes a permanent guard that cannot silently drift from the kernel again,
rather than a one-time fix. The edits are to `embedded/` verb skills, so the
ritual `skill-edit → structural-test` backstop does not apply; `skill-body-id`
(G-0299) does, and the doc-links stay masked/allowed under its carve-out.

Out of scope (owned elsewhere): the finding-code documentation drift
(`G-0283`, delivered by M-0197), and the entity-reference / id-placeholder
hygiene of these skills (the strict skill-body id gap).

Source: G-0301. Parent epic E-0048.

## Acceptance criteria

### AC-1 — aiwf-check documents the four-status AC set, kernel-derived

The `aiwf-check` skill's `acs-shape/status` row names the acceptance-criterion
status set as `{open, met, deferred, cancelled}` (four statuses), matching the
kernel's `entity.AllowedACStatuses()`.

Test: a structural test derives the set from `entity.AllowedACStatuses()` and
asserts the skill's `acs-shape/status` row — scoped to that table row, not a
flat body grep — names exactly those four statuses (and no longer says "three").
Source-derived, so the row cannot drift from the kernel set.

### AC-2 — aiwf-archive drops findings as an archivable kind

The `aiwf-archive` skill no longer presents "findings" as an archivable kind;
its kind vocabulary is consistent with the verb's `--kind` accepted set
(`epic, contract, gap, decision, adr`).

Test: assert the skill body contains no "findings"-as-kind reference (the
specific "gaps or findings" error is gone), and that the kinds it does name are
a subset of the `--kind` set derived from `internal/cli/archive/`.

### AC-3 — aiwf-contract cites the real recipe path and complete cancel FSM

The `aiwf-contract` skill references the upstream-recipe path
`internal/recipe/embedded/` (not the nonexistent `tools/internal/recipe/embedded/`),
and its cancel description documents the `deprecated → retired` cancel case
alongside `proposed`/`accepted → rejected`.

Test: assert the skill references `internal/recipe/embedded/`, that this path
exists on disk, and that no `tools/`-prefixed recipe path remains; assert the
cancel documentation covers every `CancelTarget(KindContract, …)` target
(source-derived — `proposed`/`accepted → rejected` and `deprecated → retired`).

### AC-4 — aiwf-authorize provenance-model doc-link resolves

The `aiwf-authorize` skill's provenance-model markdown link uses depth
`../../../../docs/pocv3/design/provenance-model.md`, which resolves to an
existing file from the skill's source location.

Test: extract the link's relative destination, join it to the skill file's
directory, and assert the resolved path exists — catching broken relative depth
mechanically rather than by eye.

### AC-5 — aiwf-add example is self-consistent and cites sections not lines

The `aiwf-add` skill's "typo" example uses two distinct ids (not the same id
twice), and its design-doc citations name sections rather than pinned `:NN`
line anchors.

Test: assert the typo example names two different ids; assert the skill body
contains no `docs/….md:NN` pinned-line citation (fragile anchors that rot as the
docs change).

## Work log

tdd: advisory — no per-AC phase timeline; this log records the final outcome.

### AC-1 — aiwf-check documents the four-status AC set, kernel-derived
`acs-shape/status` row → `{open, met, deferred, cancelled}` ("four"). Pinned by `TestAiwfCheckSkill_ACStatusSetMatchesKernel` (source-derived from `entity.AllowedACStatuses()`). commit 94f35591.

### AC-2 — aiwf-archive drops findings as an archivable kind
"gaps or findings" → "gaps or decisions". Pinned by `TestAiwfArchiveSkill_NoFindingsAsKind`. commit 94f35591.

### AC-3 — aiwf-contract cites the real recipe path and complete cancel FSM
Recipe path `tools/…` → `internal/recipe/embedded/`; cancel description adds `deprecated → retired`. Pinned by `TestAiwfContractSkill_RecipePathAndCancelFSM` (recipe-path existence + section-scoped cancel FSM source-derived from `CancelTarget`). commit 94f35591.

### AC-4 — aiwf-authorize provenance-model doc-link resolves
Doc-link depth `../../` → `../../../../`. Pinned by `TestAiwfAuthorizeSkill_ProvenanceDocLinkResolves` (resolves the link against the skill dir + `os.Stat`). commit 94f35591.

### AC-5 — aiwf-add example is self-consistent and cites sections not lines
Typo example → distinct ids (`M-008` for `M-007`); citations de-pinned (dropped `:22`/`:139`, kept doc paths). Pinned by `TestAiwfAddSkill_ExampleSelfConsistentAndSectionCites`. commit 94f35591.

## Decisions made during implementation

- **AC-2 replacement kind.** "findings" → "gaps or decisions" (decisions is a real, high-volume archivable kind) rather than dropping the second example — keeps the "one kind is the volume offender" illustration accurate.
- **AC-4 scope: source-tree depth only.** Fixed the doc-link to `../../../../docs/…` so it resolves from the skill's source location. The materialized-in-consumer deadness of design/ADR doc-links (they don't resolve from a consumer's `.claude/skills/…`) is the broader concern owned by G-0315 (an M-0195 deferral), out of scope here.
- **AC-5 ripple: re-point, don't gut.** De-pinning the aiwf-add citations broke `TestSkill_AddCitesDesignIntent` (M-068/AC-2), which pinned the exact `:22`/`:139` anchors. Root-caused (not papered over) and re-pointed it to the stable doc-path form + added a guard that the pinned form can't return, and updated its doc-comment — preserving its citation-traceability intent (the M-0195 "legitimately re-pointed, not gutted" pattern).

No `aiwfx-record-decision` ADRs were needed — these are scoping choices within the milestone.

## Validation

- `go test ./internal/policies/ ./internal/skills/ ./internal/check/` — green.
- `make check-fast` (build / vet / lint / full suite) — green.
- Diff-scoped coverage — no exposure: all changes are `_test.go` (not coverage-instrumented) or markdown.
- `skill-body-id` realtree — green (recipe path + doc-link stay masked / doc-link carve-out).
- Independent adversarial reviewer — [verdict recorded in Reviewer notes].

## Deferrals

- None opened by this milestone. AC-4's consumer-repo doc-link deadness is the pre-existing G-0315, not this milestone's to resolve.

## Reviewer notes

- **Independent adversarial reviewer: APPROVE, no blocking findings.** All 5 fixes verified factually correct against kernel source; all 5 tests confirmed genuine via red-on-old / green-on-new experiments (restore `94f35591~1` content → test reddens naming the defect → restore); the AC-5 ripple re-point confirmed strengthened, not gutted; no real-id leak; full affected suite green.
- **Track-for-later #1 (AC-2 guard narrow) — addressed inline.** `TestAiwfArchiveSkill_NoFindingsAsKind` now also source-derives from `entity.AllKinds()`: every `--kind <x>` the skill demonstrates must name a real entity kind, catching a fabricated flag like `--kind findings` that the prose literal misses. The prose-offender phrasing stays guarded by the `"or findings"` literal (the actual defect was prose, not a flag).
- **Track-for-later #2 (AC-3 iteration domain hardcoded) — accepted.** Only the cancel *targets* are source-derived (`CancelTarget`); the from-status *set* is a literal slice, mirroring the repo convention that a kind's status set is itself hardcoded in Go. The load-bearing part (targets) is derived.
- AC-1 and AC-3 are **source-derived** (iterate `entity.AllowedACStatuses()` / `entity.CancelTarget(KindContract, …)`), so those two skill facts are permanent guards that fail if the kernel set changes and the skill isn't updated — mini-chokepoints, not one-time fixes.
- AC-3's cancel-FSM assertion is scoped to the "Cancel a contract entirely" section (via `sectionUnder`), so an incidental `retired`/`deprecated` mention in the FSM diagram elsewhere does not satisfy it.
