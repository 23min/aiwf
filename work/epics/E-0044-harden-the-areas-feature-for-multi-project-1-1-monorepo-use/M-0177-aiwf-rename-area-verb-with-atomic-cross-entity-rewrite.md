---
id: M-0177
title: aiwf rename-area verb with atomic cross-entity rewrite
status: done
parent: E-0044
tdd: required
acs:
    - id: AC-1
      title: rename-area rewrites the member and all referencing entities atomically
      status: met
      tdd_phase: done
    - id: AC-2
      title: the rename commit carries rename-area trailers and aiwf history renders it
      status: met
      tdd_phase: done
    - id: AC-3
      title: rename-area refuses undeclared old or already-declared new; no partial write
      status: met
      tdd_phase: done
    - id: AC-4
      title: rename-area <new> <old> reverses a prior rename
      status: met
      tdd_phase: done
    - id: AC-5
      title: rename-area ships tab-completion for old, --help, and skill-coverage
      status: met
      tdd_phase: done
---
## Goal

Make renaming a declared area safe: `aiwf rename-area <old> <new>` renames the `aiwf.yaml` member and atomically rewrites every entity that references it, in one trailered commit — the same referential-integrity discipline `aiwf reallocate` applies to ids.

## Context

Today, renaming an area in `aiwf.yaml` (or removing one) leaves every entity that still carries the old value orphaned: `area-unknown` flags them at warning, and the grouping view silently buckets them into the complement. No verb rewrites the references. This milestone adds it, closing the Tier-0 referential-integrity hole on the area closed set.

## Acceptance criteria

Formalized at start-milestone as AC-1–AC-5 (frontmatter `acs[]`; full statements and pinning tests under the AC sections below). Summary:

- **AC-1** — `aiwf rename-area <old> <new>` renames the member in `aiwf.yaml` and rewrites the `area` frontmatter of every referencing entity in a single commit.
- **AC-2** — the commit carries `aiwf-verb: rename-area` + entity/actor trailers; `aiwf history` renders the rename.
- **AC-3** — refuses when `<new>` already names a declared member, or `<old>` is not declared — clear error, no partial write.
- **AC-4** — the rename reverses via the same verb (`rename-area <new> <old>`).
- **AC-5** — tab-completion offers declared members for `<old>`; skill-coverage (allowlist) + `--help` ship with it.

## Constraints

- Atomic: either the `aiwf.yaml` member and all entity rewrites land, or nothing does — one commit, abort-before-commit on any failure.
- Single source of truth: the member set in `aiwf.yaml` is the authority; the verb never invents members.
- "What undoes this?" — the same verb with swapped args; documented at design.

## Design notes

- Mirror `aiwf reallocate`'s tree-walk-and-rewrite + trailer-stamp shape.

## Out of scope

- `paths:` (Tier 1) — rename-area operates on the label; the keystone milestone owns carrying any paths along.
- Renaming the display-only `areas.default` label (not a member).

## Dependencies

- None. Independent Tier-0; parallel with the other Tier-0 milestones.

## References

- `internal/config/config.go` — the `Areas` member set rewritten.
- `aiwf reallocate` — the precedent for atomic cross-tree reference rewrite + trailers.
- ADR-0006 — skills policy (the verb satisfies coverage via a `skillCoverageAllowlist` entry — the "--help suffices" case).

### AC-1 — rename-area rewrites the member and all referencing entities atomically

**Property.** `aiwf rename-area <old> <new>` renames the `areas.members` entry in `aiwf.yaml` and rewrites the `area:` frontmatter of every entity tagged `<old>` to `<new>` — in ONE git commit, member display-order preserved; entities tagged other areas are untouched.

**Mechanical assertion.** `TestRenameArea_AC1_RewritesMemberAndEntitiesAtomically` (`internal/cli/integration/renamearea_test.go`) renames in a fixture with ≥2 `platform` + 1 `billing` entity and asserts the new member set, every `platform` entity's frontmatter, the untouched `billing` entity, and exactly one new commit. Verb-level `TestRenameArea_RewritesMemberAndEntities` + `TestRenameArea_NoReferencingEntities` pin the Plan shape (one `OpWrite` for `aiwf.yaml` + one per referencing entity; aiwf.yaml-only when nothing references `<old>`). Vacuity: the reviewer's "rewrite only the first matching entity" mutation reddens it.

### AC-2 — the rename commit carries rename-area trailers and aiwf history renders it

**Property.** The single commit carries `aiwf-verb: rename-area`, `aiwf-actor:`, and one `aiwf-entity:` trailer per rewritten entity (canonicalized, id-sorted) — and no trailer for an untouched-area entity. `aiwf history <rewritten-entity>` renders the rename for each affected entity.

**Mechanical assertion.** `TestRenameArea_AC2_TrailersAndHistory` (`internal/cli/integration/renamearea_test.go`) asserts the exact trailer set and that `aiwf history E-0001` shows the `rename-area` row, plus the negative (the untouched entity gets no trailer). `aiwf-verb: rename-area` is auto-recognized by `trailer-verb-unknown` via the Cobra registration, and `aiwf-verb` alone suppresses the untrailered-entity audit.

### AC-3 — rename-area refuses undeclared old or already-declared new; no partial write

**Property.** The verb refuses when `<old>` is not declared, when `<new>` already names a member, and when `<old>`/`<new>` are empty or identical — a clear error naming the declared set, writing nothing (no `aiwf.yaml` change, no entity change, no commit). All validation precedes any write, so refusal is atomic by construction.

**Mechanical assertion.** `TestRenameArea_AC3_RefusesAndNoPartialWrite` (integration) asserts both refusal paths leave `aiwf.yaml` byte-identical and the commit count unchanged. Verb-level `TestRenameArea_ValidationRefusals` + `TestRenameArea_UndeclaredErrorNamesDeclaredSet` + `TestRenameArea_DocWithoutAreasBlockErrors` exhaust the refusal cases. Vacuity: the reviewer's "invert the `<new>` collision check" mutation reddens it.

### AC-4 — rename-area <new> <old> reverses a prior rename

**Property.** The verb is its own inverse — after `rename-area platform infra`, `rename-area infra platform` restores the prior member name and every entity tag. This is the verb's "what undoes this?" answer.

**Mechanical assertion.** `TestRenameArea_AC4_ReverseRestoresInitialState` (integration) renames forward then back and asserts the final `aiwf.yaml` and entity frontmatter equal the initial state (byte-identical, via the remarshalled block + deterministic `entity.Serialize`).

### AC-5 — rename-area ships tab-completion for old, --help, and skill-coverage

**Property.** `<old>` tab-completes to exactly the declared `areas.members` (nothing at `<new>`'s position); `aiwf rename-area --help` ships, carrying the orphan-trap warning (use the verb, never a hand-edit, or referencing entities are silently orphaned); and the verb satisfies the skill-coverage chokepoint via a `skillCoverageAllowlist` entry — the ADR-0006 "--help suffices" case, no dedicated skill (the start-milestone decision).

**Mechanical assertion.** `TestRenameArea_AC5_Discoverability` (integration) asserts `CompleteAreaArg(0)` returns the declared members and the allowlist entry is present; the `skill_coverage` and completion-drift policy tests fail CI if the verb lacks coverage or positional completion. `TestNewCmd_SmokeShape` pins the command shape + the orphan-trap `Long`.

## Work log

### AC-1 / AC-2 / AC-3 / AC-4 / AC-5 — rename-area verb

Implemented `aiwf rename-area <old> <new>` mirroring `reallocate` (atomic tree-walk-rewrite) + `contract bind` (aiwf.yaml mutation via the comment-preserving `aiwfyaml.Doc`). New: `internal/verb/renamearea.go`, `internal/cli/renamearea/`, `internal/aiwfyaml` `SetAreas` (areas-block byte-range splice), `cliutil.CompleteAreaArg`; allowlist entries in `skill_coverage.go` + `m0123_ac5_drift_test.go`.

- implementation commit: `5ba52c20` (`feat(rename-area): atomic area rename verb with cross-entity rewrite`)
- tests: verb + cli + integration + aiwfyaml + policies green; `-race` green on new code
- lint: `golangci-lint run` on all touched packages → 0 issues
- coverage: aiwfyaml `SetAreas`/marshal/replace 100%; `RenameArea` 97.7% (one `//coverage:ignore` defensive serialize path)

## Decisions made during implementation

- **Provenance posture: human-only.** Routed through the scope-gated `DecorateAndFinish` with an empty `TargetID`, so an authorized AI cannot run it (no single target entity to satisfy `VerbAct` scope-reachability). Rationale: a rename ripples across entities in areas an agent's scope doesn't cover, so refusing it for a scope-bound agent is correct, not accidental — a sovereign-flavored, referential-integrity-sensitive config act. Diverges from `contract bind` (one contract target, `FinishVerb`). Pinned by `TestRenameArea_AuthorizedAIRefused` (a scoped `ai/...` actor is refused with `provenance-authorization-out-of-scope`; setting `TargetID` reddens it). Ratified by the human at review.
- **aiwf.yaml writer: extend the comment-preserving `aiwfyaml.Doc`.** Added `SetAreas` mirroring the `SetContracts` byte-range splice rather than a whole-file marshal, so comments/formatting outside the `areas:` block survive. Same trade-off as `SetContracts`: a future hand-added in-block field (e.g. the out-of-scope `paths:`) would be canonicalized away by the remarshal — not reachable with today's `{members, default}` schema. Candidate for an ADR if the Tier-1 `paths:` evolution (M-0179) revisits this.
- **Skill coverage via allowlist, not a dedicated skill** — ADR-0006 "--help suffices" case; the orphan-trap warning lives in `--help`.

## Validation

- `go test` on `internal/verb`, `internal/cli/renamearea`, `internal/cli/integration`, `internal/aiwfyaml`, `internal/policies` — green; `-race` green on `internal/verb` + `internal/cli/renamearea`.
- `golangci-lint run` on all touched packages — 0 issues. `go build ./...` clean. No new external dependency.
- `aiwf check` (worktree binary) on a real post-rename tree — 0 errors; no `area-unknown` / `provenance-untrailered-entity-commit` / `trailer-verb-unknown`.
- Independent fresh-context review — APPROVE (all 5 ACs mutation-verified; the `aiwf.yaml` splice probed with 9 edge-case inputs, zero corruption; atomicity, trailers, history verified live; no scope creep).

## Deferrals

None. The Tier-1 `paths:` evolution that may revisit the `aiwfyaml` in-block canonicalization is M-0179, already planned in this epic.

## Reviewer notes

- **Two-lens review: APPROVE.** Code-quality lens mutation-tested every AC and adversarially probed the byte-range splice. `wf-rethink` lens: the verb is a local mirror of `reallocate`/`contractbind` — no novel module or abstraction warranting a redesign pass.
- **Non-blocking, accepted:** the per-entity trailer canonicalization (`entity.Canonicalize(e.ID)`) is exercised only with already-canonical ids (defensive; ids are always canonical in practice) — left unpinned per YAGNI.
- **Build process:** the implementation was authored by a builder subagent against a design locked in conversation, then independently reviewed; the red→green cycle (with vacuity) genuinely ran in that session, and phases were recorded red→green→done per AC.

