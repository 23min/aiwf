---
id: M-0163
title: Embed and materialize the guidance fragment
status: done
parent: E-0040
tdd: required
acs:
    - id: AC-1
      title: Materialized guidance file contains every full-set rule and the id-shape line
      status: met
      tdd_phase: done
    - id: AC-2
      title: Materialized guidance file declares the aiwf version it was generated from
      status: met
      tdd_phase: done
    - id: AC-3
      title: init and update materialize the gitignored guidance file idempotently
      status: met
      tdd_phase: done
    - id: AC-4
      title: Guidance fragment stays within its per-turn line budget
      status: met
      tdd_phase: done
---
# M-0163 — Embed and materialize the guidance fragment

## Goal

Ship the version-pinned aiwf guidance fragment as an embedded artifact that
`aiwf init` and `aiwf update` materialize to `.claude/aiwf-guidance.md`
(gitignored), without yet wiring the consumer's `CLAUDE.md`. After this milestone
the guidance file exists in a consumer tree but is not yet imported — a coherent
intermediate state.

## Context

E-0040 reaches the consumer's per-turn surface in two steps: materialize the
content (here), then wire the import line (the next milestone). The fragment
content and the rule for what may enter it are settled in ADR-0018. The shipping
mechanism is the embed-and-materialize manifest established by ADR-0014 / E-0038;
this milestone adds one more materialized artifact to that set.

## Acceptance criteria

### AC-1 — Materialized guidance file contains every full-set rule and the id-shape line

The fragment carries each rule named in ADR-0018's inclusion set — per-action
gate discipline; never suggest the user pause; collision resolves via
`aiwf reallocate` not `git mv`; AC promotion requires mechanical evidence; Q&A
one decision at a time; finish in context rather than papering over — plus the
id-shape one-liner. Asserted against the embedded bytes via a path constant (the
G-0182 pattern), not a flat grep.

### AC-2 — Materialized guidance file declares the aiwf version it was generated from

The fragment carries a version marker; the test asserts it equals the binary's
own version (`version.Current()`), so a stale materialized copy is detectable.

### AC-3 — init and update materialize the gitignored guidance file idempotently

A fixture-tree test drives `aiwf init` and `aiwf update`, asserting
`.claude/aiwf-guidance.md` is written and the gitignore entry is present; a second
run produces no diff.

### AC-4 — Guidance fragment stays within its per-turn line budget

A line-count assertion holds the fragment under a fixed budget so it stays terse
enough to re-anchor on every turn.

## Constraints

- The materialized file lives in-repo under `.claude/` — never `~/.claude/` or an
  absolute path (ADR-0018).
- The fragment obeys ADR-0018's inclusion principle and its hard boundary (it
  says nothing about the consumer's own code).
- Authored in the embedded snapshot (ADR-0014 / ADR-0016), not a retired
  upstream; AC tests assert against the embedded bytes.

## Design notes

- ADR-0018 — the consent decision and the inclusion principle that govern this
  fragment's content.
- Reuses the embed-and-materialize manifest mechanism (E-0038); the guidance file
  joins the unconditionally-refreshed artifact set (unlike the scaffold-once
  statusline).

## Surfaces touched

- `internal/skills/embedded-guidance/aiwf-guidance.md` — the authored fragment source.
- `internal/skills/guidance.go` — the embed + `RenderGuidance` / `MaterializeGuidance`.
- `internal/initrepo/initrepo.go` — `ensureGuidance` wired into the init/update pipeline.

## Out of scope

- The marker-wrapped `CLAUDE.md` import line and its consent flow — the wiring
  milestone.
- The `aiwf doctor` unwired-guidance finding — the doctor-finding milestone.

## Dependencies

- None. (ADR-0018 is `accepted`; the embed-and-materialize mechanism already
  exists.)

## References

- ADR-0018 — risk-calibrated consent + the fragment inclusion principle.
- G-0243 — the gap E-0040 closes.
- ADR-0014 / E-0038 — the embed-and-materialize precedent.
- G-0182 — the embedded-path-constant test pattern.

---

## Work log

<!-- Phase/met timeline per AC is authoritative in `aiwf history M-0163/AC-<N>`;
     the implementation landed in this milestone's single wrap commit. -->

### AC-1 — Fragment contains every rule

`RenderGuidance` returns the embedded 31-line fragment (7 rules + id-shape line);
asserted against embedded bytes. · tests 1/1 · `aiwf history M-0163/AC-1`

### AC-2 — Materialized file declares the binary version

`RenderGuidance` substitutes the `__AIWF_VERSION__` sentinel; `MaterializeGuidance`
stamps `version.Current()`. · tests 2/2 · `aiwf history M-0163/AC-2`

### AC-3 — init/update materialize the gitignored file, idempotently

`ensureGuidance` wired into the init/update pipeline; `.claude/aiwf-guidance.md`
added to `GitignorePatterns()`; seam test + both IO error branches. · tests 5/5 ·
`aiwf history M-0163/AC-3`

### AC-4 — Fragment within the per-turn line budget

Line-budget guard over `GuidanceBytes()` (31/50 lines). · tests 1/1 ·
`aiwf history M-0163/AC-4`

## Decisions made during implementation

- (none)

## Validation

- `go build ./...` — green; `golangci-lint run` (full module) — 0 issues; `go vet` — clean.
- `go test ./internal/skills/ ./internal/initrepo/` — green. `guidance.go` and
  `ensureGuidance` at 100% line coverage; every reachable branch (both IO error
  paths and the error-wrap) has an explicit test.
- Full `go test ./...` — green; `internal/cli/integration` flaked on the known
  TempDir-cleanup race under the parallel full run but passes isolated (~79s).
- `aiwf check` — 0 errors (3 pre-existing / worktree-benign warnings:
  archive-sweep-pending, terminal-entity-not-archived, untrailered-scope-undefined).

## Deferrals

- (none)

## Reviewer notes

- AC-2 (version marker) and AC-4 (line budget) are verification + regression
  guards, not test-driven REDs — their behavior was realized in AC-1/AC-3's
  GREEN. This is inherent to content-property ACs on an artifact created earlier
  in the same milestone.
- The consumer-facing CLAUDE.md / design-doc description of "what aiwf
  materializes" is intentionally NOT updated here: the guidance file is inert
  until M-0164 wires the import line, so that doc-completeness update belongs with
  M-0164 / the epic wrap.
- `MaterializeGuidance` mirrors the `ScaffoldStatusline` IO pattern and
  `ensureGuidance` mirrors `ensureSkills`; the difference is the guidance file is
  byte-refreshed on every update (in `GitignorePatterns` + the refresh pipeline),
  unlike the scaffold-once statusline.
