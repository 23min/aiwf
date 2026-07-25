---
id: M-0279
title: Extract the shared ResolveRoot to ResolveActor prelude
status: in_progress
parent: E-0072
depends_on:
    - M-0278
tdd: advisory
acs:
    - id: AC-1
      title: Shared ResolveRoot to ResolveActor prelude helper exists
      status: met
    - id: AC-2
      title: All prelude-duplicating verbs route through the shared helper
      status: met
    - id: AC-3
      title: Prelude behavior including the usage-error arm is unchanged
      status: met
---
# M-0279 — Extract the shared ResolveRoot to ResolveActor prelude

## Goal

Introduce one shared `ResolveRoot → ResolveActor` prelude helper (with its identical usage-error arm) and route the ~23 verbs that duplicate it through it.

## Context

After the diagnostic block is single-sourced, the second-largest per-verb scaffold is the `ResolveRoot → ResolveActor` prelude: resolve the repo root, resolve the actor, and bail with the same usage-error arm when either fails. It is duplicated across ~23 verbs. Builds on the BeginVerbDiag milestone (same `cliutil` surface, same migration shape).

## Acceptance criteria

<!-- ACs created via `aiwf add ac`; contracts filled below. -->

### AC-1 — Shared ResolveRoot to ResolveActor prelude helper exists

One `cliutil` helper resolves the repo root and the actor and returns them (or the shared usage error), replacing the hand-rolled `ResolveRoot → ResolveActor` sequence and its identical usage-error arm.

### AC-2 — All prelude-duplicating verbs route through the shared helper

Every verb that previously duplicated the prelude now calls the shared helper; none reconstructs it inline. Any verb with a legitimately different prelude is documented as an intentional non-member. (Reference-phrased against the set of verbs carrying the prelude today, not a fixed count.)

### AC-3 — Prelude behavior including the usage-error arm is unchanged

Across the migrated verbs, the resolved root, the resolved actor, the usage-error message text, and the exit path are byte-identical to pre-migration — verified by the existing verb and integration suites staying green.

## Constraints

- Behavior-preserving: the resolved root, the resolved actor, the usage-error message, and the exit path are unchanged per verb.
- Extraction lands in today's `cliutil`; no anticipatory restructuring for the G-0227 split.
- Verbs with a legitimately different prelude (if any surface during migration) are documented as intentional non-members rather than forced through the helper.

## Out of scope

- The diagnostic block (its own milestone, a prerequisite).
- The regression-guard structural test (its own milestone).

## Dependencies

- The BeginVerbDiag milestone — same package surface; sequenced after it to keep the two migrations reviewable in isolation.

## References

- E-0072 — parent epic.
- G-0447 — the convergent-duplication cleanup this seam completes.

## Work log

### AC-1 — Shared prelude helpers

Extracted `cliutil.ResolvePrelude` (text usage-error arm: `Errorf` + `ExitUsage`) and `cliutil.ResolvePreludeEnvelope` (envelope arm: `FinishVerbOutcome`, honoring `--format=json`), each owning the `ResolveRoot → ResolveActor` sequence *and* its usage-error arm. Behavior pinned by `prelude_test.go` — success path, both error arms, byte-exact usage-error text, and the envelope `status: error` shape. · commit 39a53c72

### AC-2 / AC-3 — bulk migration

Migrated all 23 prelude sites onto the helpers — 21 text-arm → `ResolvePrelude`, 2 envelope-arm (archive, rewidth) → `ResolvePreludeEnvelope`; no verb reconstructs the prelude inline. `importcmd` remains a documented non-member. `milestone-tdd` / `milestone-depends-on` / `add-ac` took a `code, _ := … DecorateAndFinish` → `code, _ =` change (they have no named `code` return, so the prelude now declares it). Net −138 lines; byte-identical verified by the `internal/cli/integration` diag/JSON suite staying green. · commit be463fdc

## Validation

- `go build ./...`: clean.
- `go test ./internal/cli/...` incl. the integration diag/JSON suite: green.
- `go test ./internal/policies/... ./internal/verb/...`: green.
- `make lint` (full `golangci-lint` set): 0 issues.
- `make coverage-gate` (diff-scoped branch coverage + firing-fixture): green.
- `aiwf check`: 0 error-severity findings on the milestone.
- Runtime byte-diff: the pre-migration and migrated `aiwf` binaries produce identical prelude-error output (stdout / stderr / exit code) across all 23 migrated verbs × {text, json}, modulo the build-provenance `version` field and the random `correlation_id`.

## Reviewer notes

- Independent fresh-context two-lens review before wrap. **Code-quality** (`wf-review-code`): APPROVE — all six load-bearing claims verified by measurement (binary byte-diff for byte-identical behavior, grep for completeness, `make coverage-gate` for the ignore annotations). **Design-quality** (`wf-rethink`) on the `prelude.go` helper unit: DESIGN-SOUND.
- Design decision (two helpers over one): a single helper cannot serve both the text usage-error arm and the `--format=json`-honoring envelope arm without changing behavior; a unified closure-parametrized helper would push the divergent arm back to the call site, defeating AC-1's requirement that the helper own the usage-error arm. The shared-nothing two-entrypoint shape single-sources each arm and mirrors the `BeginVerbDiag` / `BeginReadVerbDiag` precedent. The `(rootDir, actorStr, code, ok)` return mirrors the package's existing `AcquireRepoLock` comma-ok idiom.
- Non-member set: `importcmd` (interleaved manifest parse between root and actor resolution; three-way actor precedence). `whoami` / `doctor` resolve the actor via `ResolveActorWithSource` for identity display — a different shape from the bail-on-usage-error prelude, correctly not migrated.
- Coverage: the four `//coverage:ignore` annotations (archive, cancel, move, rewidth) sit on the `if !ok` short-circuit whose failure the per-verb tests do not reproduce; the shared resolution failure is tested once in `prelude_test.go`. The other sixteen sites' failure branches are covered by pre-existing per-verb tests (confirmed by the coverage-gate). Uniform per-verb prelude-error coverage is a possible future refinement, not a defect here.

## Deferrals

- G-0456 — the prelude resolution error format is not uniform across verbs (~21 text-arm verbs ignore `--format=json` on a prelude root/actor failure; archive / rewidth emit a JSON envelope). A pre-existing inconsistency preserved by this behavior-preserving refactor; unifying it is a behavior change to ~21 sites, out of scope here. Surfaced by the design-lens review.
