---
id: M-0189
title: Add worktree.dir config knob defaulting to .claude/worktrees
status: in_progress
parent: E-0046
tdd: required
acs:
    - id: AC-1
      title: aiwf.yaml worktree.dir is parsed and exposed through config
      status: met
      tdd_phase: done
    - id: AC-2
      title: Unset, empty, or invalid worktree.dir defaults to .claude/worktrees
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf doctor surfaces the resolved worktree.dir line
      status: open
      tdd_phase: done
---

# M-0189 — Add worktree.dir config knob defaulting to .claude/worktrees

## Goal

Add a `worktree.dir` key to `aiwf.yaml` (default `.claude/worktrees`) giving a project a
persistent default placement for ritual worktrees, surfaced where the start rituals can
read it.

## Acceptance criteria

Tracked in frontmatter `acs[]` and detailed in the `### AC-1` / `### AC-2` / `### AC-3`
sections below. The AC-3 surface was settled at start-milestone as a greppable `aiwf
doctor` line — not a new verb, not a JSON envelope (see *Decisions made during
implementation*).

## Context

`aiwf.yaml` already carries top-level feature keys (`tree:`); a `worktree:` key follows
the established shape. The correct placement is environment-dependent (in-repo for
sandboxed devcontainers, siblings on a bare host), so the default is config-driven, not
hardcoded (E-0046 constraint). This milestone adds only the knob + default + parse; the
rituals consume it in M-0190.

## Constraints

- Minimal surface: a single repo-relative directory value (YAGNI — no absolute paths or
  multiple roots until a consumer needs them).
- Unset / empty / invalid values fall back to the kernel default `.claude/worktrees`.

## Out of scope

- The rituals reading/defaulting to the knob (M-0190); the loader guard (M-0188).

## Dependencies

- None.

## References

- E-0046 epic spec; `aiwf.yaml` `tree:` key precedent.

### AC-1 — aiwf.yaml worktree.dir is parsed and exposed through config

A `worktree.dir` key under a top-level `worktree:` block in `aiwf.yaml` parses into
`config.Config.Worktree.Dir` and is exposed through the nil-tolerant `WorktreeDir()`
getter (mirroring `HTMLOutDir()` / `EntityTitleMaxLength()`).

Evidence: `TestWorktreeDir_DefaultUnset` and `TestWorktreeDir_ExplicitOverride` in
`internal/config/config_test.go` drive `config.Load` → `WorktreeDir()` end-to-end.

### AC-2 — Unset, empty, or invalid worktree.dir defaults to .claude/worktrees

The knob is a single repo-relative directory (E-0046 YAGNI). An unset, empty,
whitespace-only, or absolute `worktree.dir` falls back to the kernel default
`.claude/worktrees` (`DefaultWorktreeDir`); a nil `*Config` does too.

Evidence: `TestWorktreeDir_InvalidFallsBackToDefault` (empty / whitespace / two absolute
shapes) and `TestWorktreeDir_NilReceiver`. Getter branch coverage measured at 100%.

### AC-3 — aiwf doctor surfaces the resolved worktree.dir line

`aiwf doctor` emits a greppable `worktree-dir: <resolved> (default|configured)` line so
the M-0190 ritual reads the placement directory with
`aiwf doctor | grep '^worktree-dir:' | awk '{print $2}'`. No new verb, no JSON envelope
(`aiwf doctor` is human-output-only).

Evidence: `TestDoctorReport_WorktreeDirDefault` and `_Configured` in
`internal/cli/doctor/worktree_test.go` drive the real `DoctorReport` seam, asserting
structurally on the `worktree-dir:` line.

## Work log

### AC-1 / AC-2 — config knob + getter

Added `config.Worktree{Dir}`, `DefaultWorktreeDir`, and the `WorktreeDir()` getter
(empty / whitespace / absolute → default; nil-tolerant). 4 table tests; getter coverage
100%. · tests: 4 new

### AC-3 — aiwf doctor surface

Added a `worktree-dir:` line to `DoctorReport` after the `config:` block, annotated
`(default)` / `(configured)`. 2 seam tests driving `DoctorReport`. · tests: 2 new

**TDD phase note (G-0293).** The test-first discipline was genuinely followed — each test
written first and observed red, then green — but the `tdd_phase` promotes were stamped at
wrap, not contemporaneously. The phase timeline is therefore *not* temporal evidence; the
real mechanical evidence is the test suite plus 100% diff coverage on the new code. The
systemic fix (promote phases live during the cycle) is tracked in G-0293.

## Decisions made during implementation

- **AC-3 surface = a greppable `aiwf doctor` line — not a new verb, not JSON.** The
  rituals (M-0190) are LLM-read markdown that shell out to `aiwf`; they need a CLI surface
  that resolves-and-prints the value (so the default lives in one place, not duplicated in
  markdown). `aiwf doctor` is human-output-only (no `--format=json`), so the surface is a
  human line the ritual greps. A dedicated `aiwf config get` verb was rejected to keep the
  verb surface minimal (operator preference). The AC-3 title was renamed mid-flight to drop
  an erroneous "JSON" claim.
- **Annotation = `resolved != DefaultWorktreeDir`.** Known cosmetic limitation: a consumer
  who explicitly sets `worktree.dir: .claude/worktrees` is labelled `(default)`. The ritual
  reads the path (field 2), not the annotation, so this is harmless; left as a known
  limitation rather than duplicating the getter's validity logic in doctor.

## Validation

- `make ci` — green (vet, lint, test-cov with race + coverage, 29-step self-check).
- `go test ./...` — exit 0, no failures across all packages (config is widely imported; no
  downstream consumer broke).
- `aiwf check` — 0 errors.
- 6 new tests (4 config getter, 2 doctor seam); getter coverage 100%, both doctor
  annotation branches covered.

## Reviewer notes

Independent fresh-context reviewer verdict: **APPROVE**, zero blocking findings — verified
by measuring (built the binary, ran the getter + doctor against fixtures, measured 100%
getter coverage and both annotation arms). Three advisories, none blocking:

1. **Annotation mislabels the explicit-default case** (`doctor.go`) — cosmetic; documented
   under *Decisions* as a known limitation (the ritual reads the path, not the annotation).
2. **`..`-traversal not rejected by the getter** — out of M-0189 scope (the getter rejects
   empty / whitespace / absolute). A repo-relative path that escapes the repo would defeat
   the in-repo guarantee, but the right place to reject it is the *use* site (M-0190), not
   this parse-layer getter. Carried as a deferral below.
3. **Flat table loop without `t.Run`** (`config_test.go`) — left as-is: it mirrors the
   adjacent `TestEntityTitleMaxLength_NonPositiveFallsBackToDefault` pattern in the same
   file; the failing input is named in the error message.

## Deferrals

- **G-0293** — promote `tdd_phase` live during the cycle, not bursted at wrap (the phase-
  ladder methodology issue surfaced while wrapping this milestone).
- **`..`-traversal validation (→ M-0190).** The `WorktreeDir()` getter honors a
  repo-relative path that escapes the repo (`../../foo`). M-0190 consumes the value to place
  worktrees; the escape check belongs at that use site so a worktree can't land outside the
  repo (defeating ADR-0023 / the M-0188 loader guard). Not architecturally distinct from
  M-0190's deliverable, so no separate gap — carried into M-0190 when it starts.

