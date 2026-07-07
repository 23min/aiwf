---
id: M-0238
title: Migrate bare-stderr call sites; forbidigo chokepoint
status: draft
parent: E-0061
depends_on:
    - M-0237
tdd: required
acs:
    - id: AC-1
      title: Named bare-stderr call sites emit diagnostic events through the bound logger
      status: open
      tdd_phase: red
    - id: AC-2
      title: A migrated verb run with AIWF_LOG=info fires the expected structured event
      status: open
      tdd_phase: red
    - id: AC-3
      title: A non-allowlisted bare print call fails CI via forbidigo and a policy test
      status: open
      tdd_phase: red
    - id: AC-4
      title: aiwf.yaml's logging block is parsed, validated, and surfaced by aiwf doctor
      status: open
      tdd_phase: red
---

## Goal

Migrate the codebase's known bare-stderr diagnostic call sites onto
`internal/logger` (M-0237), and make the discipline mechanical: a
`forbidigo` lint rule plus a policy-test backstop so a future bare print
call outside the allowlist fails CI even if the linter is ever disabled.

## Context

M-0237 shipped `internal/logger` as a standalone, unused package. This
milestone is the first thing to actually call it. Five known call sites
carry bare `fmt.Fprintln(os.Stderr, …)` diagnostic output today:
`internal/cli/statusline.go`, `internal/cli/root.go`,
`internal/verb/move.go`, `internal/verb/cancel.go`,
`internal/verb/upgrade.go`.

## Acceptance criteria

### AC-1 — Named bare-stderr call sites emit diagnostic events through the bound logger

Per call site, the classification decision (is this call a diagnostic event
that belongs on the opt-in logger, or a genuinely operator-facing
warning/error that must stay visible on stderr regardless of `AIWF_LOG`) is
made individually — not predetermined by this spec. A site classified as
diagnostic routes through `logger.Info("verb.<event>", …)` with structured
fields, never string interpolation. A site classified as operator-facing
stays on the existing `internal/cli/output` stderr path unchanged.

### AC-2 — A migrated verb run with AIWF_LOG=info fires the expected structured event

Per migrated verb, a test drives it through its real dispatcher with
`AIWF_LOG=info` set, captures the `slog` handler, and asserts the expected
`verb.<event>` fires with the bound fields (`verb`, `entity`, `actor`,
`run_id`) — "test the seam, not just the layer" (CLAUDE.md §Go conventions),
applied to the logging seam specifically.

### AC-3 — A non-allowlisted bare print call fails CI via forbidigo and a policy test

`forbidigo` is configured in `.golangci.yml` banning bare `fmt.Println`,
`fmt.Print`, and `fmt.Fprintln(os.Stdout|os.Stderr, …)` outside an explicit
allowlist (`cmd/aiwf/main.go`, the human-text branch in
`internal/cli/output/outputformat.go`, golden-file regeneration helpers).
`internal/policies/logging_chokepoint_test.go` AST-walks `internal/` and
`cmd/` for the same pattern independently — same shape as
`PolicyNoHardcodedEntityPaths` — so the discipline holds even if the linter
rule is ever disabled.

### AC-4 — aiwf.yaml's logging block is parsed, validated, and surfaced by aiwf doctor

Parsing and validation of the `logging:` block itself landed in M-0237 (a
dependency of that milestone's precedence-resolution AC); what's new here is
`aiwf doctor` reporting the currently *active*, fully-resolved logging
configuration (level, format, destination, and which source — env, yaml, or
default — won) so an operator can confirm what's on without reading source.

## Constraints

- The `forbidigo` rule and `logging_chokepoint_test.go` land together, in
  the same commit sequence — the rule alone, without the policy-test
  backstop, is not sufficient (CLAUDE.md's own constraint on this ADR).
- No call site is migrated without its own AC-2-shaped test; a "migrate now,
  test later" partial state does not close this milestone.

## Design notes

- ADR-0017 is the locked design. The per-site diagnostic-vs-operator-facing
  classification is this milestone's own judgment call, made once per site
  and recorded in that site's commit, not re-litigated later.

## Surfaces touched

- `internal/cli/statusline.go`, `internal/cli/root.go`
- `internal/verb/move.go`, `internal/verb/cancel.go`, `internal/verb/upgrade.go`
- `.golangci.yml` (forbidigo config)
- `internal/policies/logging_chokepoint_test.go` (new)
- `internal/cli` (`aiwf doctor` output)

## Out of scope

- `correlation_id` / envelope wiring, mutating-verb metadata, `--trace` —
  M-0239.
- Any call site not in the named list — a new one discovered during this
  milestone is migrated too if trivial, or filed as a gap if it's a larger
  detour.

## Dependencies

- M-0237 — `internal/logger` must exist before anything can call it.

## References

- `docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md`
- G-0223 — implement ADR-0017 opt-in slog logging; migrate bare-stderr call sites

---

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
