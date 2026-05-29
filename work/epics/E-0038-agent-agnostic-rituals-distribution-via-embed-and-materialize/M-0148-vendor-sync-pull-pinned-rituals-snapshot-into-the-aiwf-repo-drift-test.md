---
id: M-0148
title: 'Vendor-sync: pull pinned rituals snapshot into the aiwf repo + drift test'
status: done
parent: E-0038
tdd: required
acs:
    - id: AC-1
      title: Pin and vendor the upstream rituals into the repo as committed files
      status: met
      tdd_phase: done
    - id: AC-2
      title: Record the pinned upstream commit SHA in one discoverable location
      status: met
      tdd_phase: done
    - id: AC-3
      title: Drift test fails when the snapshot diverges; skips when upstream absent
      status: met
      tdd_phase: done
---
## Goal

Establish a reproducible mechanism that pulls a pinned snapshot of the upstream rituals repo (`23min/ai-workflow-rituals`) into a path inside the aiwf repo as real committed files, records the pinned upstream commit SHA, and guards the snapshot against drift with a CI test.

## Context

Before this milestone, the rituals reach consumers only via the Claude marketplace (G-0177). ADR-0014 chooses build-time embed of a vendored snapshot as the new distribution mechanism. This milestone lands the *source-of-truth* half: getting the upstream content into the aiwf repo in a form `go:embed` can consume later (M2+), pinned and drift-checked. It builds on nothing — it is the foundation the embed/materialize milestones depend on.

## Acceptance criteria

## Constraints

- The snapshot is **real committed files, not a git submodule** — the Go module proxy does not fetch submodule contents, so a submodule embeds empty under `go install` (ADR-0014 §2). Load-bearing.
- The upstream authoring home is unchanged; aiwf only vendors. No hand-edits to vendored content beyond the sync — the drift test enforces this.
- The pinned upstream SHA is recorded in a single discoverable location.

## Design notes

- ADR-0014 §2 (source of truth; pinned snapshot; submodule caveat).
- Mirrors the existing cross-repo SKILL.md fixture discipline (CLAUDE.md § "Cross-repo plugin testing"): author upstream, vendor a snapshot here, a drift test fires on divergence and skips cleanly when the upstream source is absent (CI without a checkout).
- The sync mechanism (`git subtree` pull vs scripted copy vs `go:generate`) and the SHA-record location are open questions resolved in this milestone; default lean: scripted copy + committed snapshot + a `rituals.lock`-style SHA record.

## Surfaces touched

- New vendored `rituals/` tree (path TBD in this milestone).
- A `make sync-rituals` target (or equivalent) and the drift-check test under `internal/policies/`.

## Out of scope

- Embedding the snapshot (`go:embed`) — M2.
- Materializing anything into a consumer repo — M2+.

## Dependencies

- ADR-0014 (the decision). No prior milestone.

## References

- **ADR-0014** — the decision this milestone implements (§2).
- **G-0177** — the motivating gap.
- **E-0038** — parent epic.
- **CLAUDE.md** § "Cross-repo plugin testing" — the vendoring + drift-test pattern reused.

### AC-1 — Pin and vendor the upstream rituals into the repo as committed files

### AC-2 — Record the pinned upstream commit SHA in one discoverable location

### AC-3 — Drift test fails when the snapshot diverges; skips when upstream absent

## Work log

- **AC-1 — vendor + sync mechanism.** `scripts/sync-rituals.sh` + `make sync-rituals` fetch upstream `plugins/` at the ref in `rituals.lock` and vendor it (committed files, not a submodule) under `internal/skills/embedded-rituals/plugins/` — 24 files (13 skills, 4 agents, 4 templates, 2 plugin.json). · tests: `TestRituals_VendoredSnapshotPresent`
- **AC-2 — pinned SHA record.** `rituals.lock` records the upstream URL + pinned ref `f72036c4`; trivial `key=value` format parsed by both the shell sync script and the Go drift test. · tests: `TestRituals_LockPinsUpstreamRef`
- **AC-3 — drift test.** `internal/policies/rituals_drift_test.go` fetches upstream@ref and byte-compares the vendored tree; skips under `-short`/offline; plant-and-revert verified it catches a corrupted vendored byte. · tests: `TestRituals_VendoredMatchesUpstream`, `TestRituals_DiffTrees`

The per-phase timeline (red→green→done→met) is the authoritative record in `aiwf history M-0148/AC-<N>`.

## Validation

- `go test ./...` — all packages pass, 0 failures.
- `go vet ./internal/policies/` — clean. `gofmt -l` — clean. `golangci-lint run ./internal/policies/` — 0 issues.
- `aiwf check` (worktree-built `aiwf-diag`) — 0 errors.
- Drift test plant-and-revert: corrupted a vendored byte → `TestRituals_VendoredMatchesUpstream` failed at the right file; re-synced → green. Confirms the guard is not a false-green.

## Deferrals

- None at the milestone level. (Epic-level: the non-Claude target proof is deferred to **G-0178**; `go:embed` is **M-0149**; materialization into `.claude/` is **M-0150**.)

## Reviewer notes

- **Foundation only.** M-0148 vendors + drift-guards. It deliberately does **not** wire `go:embed` (M-0149) or materialize into a consumer's `.claude/` (M-0150). The freshly-built binary does not yet contain the rituals — confirmed by grep (only the drift test references `embedded-rituals`; `skills.go` still embeds only `embedded`).
- **Drift test is non-flaky by construction.** It fetches over the network but treats any fetch failure as a *skip*, not a failure (transient network blip → skip; only a successful fetch that differs → fail). Gated off under `-short`. This follows CLAUDE.md § "Contract tests for upstream-cached systems".
- **Committed files, not a submodule** (ADR-0014 §2): the Go module proxy does not fetch submodule contents, so a submodule would embed empty under `go install`.
- **End-to-end install smoke is M-0150/AC-4** (human-verified `make install` + `aiwf update` in this repo, eyeballing `.claude/`), since no unit test substitutes for it. Surfaced during this milestone's review.

