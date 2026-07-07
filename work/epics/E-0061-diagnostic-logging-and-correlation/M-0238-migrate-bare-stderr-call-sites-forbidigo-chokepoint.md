---
id: M-0238
title: Migrate bare-stderr call sites; forbidigo chokepoint
status: in_progress
parent: E-0061
depends_on:
    - M-0237
tdd: required
acs:
    - id: AC-1
      title: Named bare-stderr call sites emit diagnostic events through the bound logger
      status: met
      tdd_phase: done
    - id: AC-2
      title: A migrated verb run with AIWF_LOG=info fires the expected structured event
      status: open
      tdd_phase: done
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
milestone is the first thing to actually call it. Five files anchor AC-1's
work: `internal/cli/cliutil/statusline.go`, `internal/cli/root.go`,
`internal/cli/move/move.go`, `internal/cli/cancel/cancel.go`,
`internal/cli/upgrade/upgrade.go` (paths corrected from the epic-planning
draft, which predates the verb/cli package split). A repo-wide sweep found
bare `fmt.Println`/`fmt.Print`/`fmt.Fprintln(os.Stdout|os.Stderr)`/
`fmt.Fprintf(os.Stdout|os.Stderr)` call sites in roughly forty files, not
five — every one of them operator-facing CLI text, not diagnostic output.
AC-3's forbidigo ban covers all of them via the `cliutil` wrapper migration;
AC-1 is scoped to adding new diagnostic breadcrumbs to the five named
verbs, not converting any existing print (see AC-1 below).

## Acceptance criteria

### AC-1 — Named bare-stderr call sites emit diagnostic events through the bound logger

Per call site, the classification decision (is this call a diagnostic event
that belongs on the opt-in logger, or a genuinely operator-facing
warning/error that must stay visible on stderr regardless of `AIWF_LOG`) is
made individually — not predetermined by this spec. Applied to the five
named call sites, every existing bare-print site is operator-facing (flag
validation, install-progress lines, confirmation prompts, recovery hints) —
none qualify as a diagnostic event, so none convert. What each of these five
verbs gains instead is a genuinely new diagnostic breadcrumb: one
`logger.Info("verb.<name>.completed", …)` call at its outcome point, bound
via `WithVerb`, with structured fields, never string interpolation. Every
existing bare-print site — in these five files and everywhere else in the
tree — is migrated to the `cliutil` text-output wrapper set under AC-3,
which is the actual "operator-facing path" all such calls now share.

### AC-2 — A migrated verb run with AIWF_LOG=info fires the expected structured event

Per instrumented verb, a test drives it through its real dispatcher with
`AIWF_LOG=info` set, captures the `slog` handler, and asserts the expected
`verb.<event>` fires with the bound fields (`verb`, `entity`, `actor`) —
"test the seam, not just the layer" (CLAUDE.md §Go conventions), applied to
the logging seam specifically. `run_id` is not asserted here: `WithVerb`
doesn't bind it yet — minting the per-invocation id and wiring it through
`WithVerb` and the JSON envelope's `metadata.correlation_id` is M-0239's
scope.

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

- `internal/cli/cliutil/statusline.go`, `internal/cli/root.go`
- `internal/cli/move/move.go`, `internal/cli/cancel/cancel.go`, `internal/cli/upgrade/upgrade.go`
- `internal/cli/cliutil` (new `Errorf`/`Errorln`/`Println`/`Print` wrapper set; new `ResolveLogger` helper)
- every other `internal/` and `cmd/` file with a bare-print call site (mechanical migration to the `cliutil` wrappers)
- `.golangci.yml` (forbidigo config)
- `internal/policies/logging_chokepoint_test.go` (new)
- `internal/config`, `internal/cli/doctor` (`aiwf doctor` output)

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
