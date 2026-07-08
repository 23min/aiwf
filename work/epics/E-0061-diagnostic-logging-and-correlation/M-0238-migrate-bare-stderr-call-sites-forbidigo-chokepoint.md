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
      status: met
      tdd_phase: done
    - id: AC-3
      title: A non-allowlisted bare print call fails CI via forbidigo and a policy test
      status: met
      tdd_phase: done
    - id: AC-4
      title: aiwf.yaml's logging block is parsed, validated, and surfaced by aiwf doctor
      status: met
      tdd_phase: done
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
`logger.Info("verb.completed", …)` call at its outcome point, bound via
`WithVerb` (which field carries the verb name — never baked into the
event string itself, matching ADR-0017's own worked example
`logger.Info("verb.commit", "verb", "promote", …)`: a stable, generic
event name plus structured fields, never string interpolation). Every
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

`forbidigo` is configured in `.golangci.yml` banning the three
destination-unambiguous bare forms — `fmt.Println`, `fmt.Print`,
`fmt.Printf` — outside the sanctioned writers (`cmd/aiwf/main.go`,
`internal/cli/cliutil/outputformat.go`'s text-mode branch,
`internal/cli/cliutil/textio.go` itself). forbidigo matches only a call's
callee expression, never its arguments, so it cannot distinguish
`fmt.Fprintln(os.Stdout, …)` from `fmt.Fprintln(someOtherWriter, …)` —
banning `Fprintln`/`Fprintf` outright would also catch legitimate
writer-parameterized code. `internal/policies/logging_chokepoint.go`
(companion test `logging_chokepoint_test.go`) is the independent
AST-walking backstop — same shape as `PolicyNoHardcodedEntityPaths` — that
inspects the actual first argument and flags `fmt.Fprintln`/`fmt.Fprintf`
specifically when it is the literal `os.Stdout` or `os.Stderr` selector,
plus redundantly re-flagging the three forbidigo-covered forms so the
discipline holds even if the linter rule is ever disabled.

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

### AC-1 / AC-2 — Diagnostic breadcrumbs on the five named verbs

Added `cliutil.ResolveLogger` (env-var-only precedence tier; falls back to a
discard logger on any resolve/open failure so diagnostic logging can never
affect a verb's own exit code) and a `cliutil.Errorf`/`Errorln`/`Printf`/
`Println`/`Print` wrapper set. Instrumented `cancel`, `move`, `upgrade`, and
the statusline scaffold/remove flows with one `WithVerb`-bound
`logger.Info("verb.completed", …)` call each, firing only on a genuinely
successful outcome; migrated all five files' pre-existing operator-facing
prints to the new wrappers; migrated `root.go`'s prints too (no diagnostic
event there — it is pure dispatch). Each site's AC-2 seam test drives the
real Cobra dispatcher (`cli.Execute`) with `AIWF_LOG=info`, reads the
resulting JSON log line, and separately confirms a failed run emits no
event and a disabled run creates no log file at all. `wf-vacuity` mutation
probes (unconditional emission, swapped verb/entity/actor argument order,
wrong stream, closing the real `os.Stderr`) all caught by the test suite;
one probe (the stderr-close guard) found and fixed a real test gap where
the assertion targeted the wrong (capture-swapped) stream. The
independent wrap review (see Reviewer notes) found two further gaps,
both closed as corrective commits: `upgrade`'s event fires before the
reexec attempt (by design — a successful reexec replaces the process
via `syscall.Exec`, so no code path after it could ever emit; commit
`7b31e85e` adds the missing test proving this is intentional, not an
oversight) and `statusline` was missing the disabled/failed-run seam
tests the other four sites carry (commit `cbf86e0e`). · commits
`14c81e3a`, `7b31e85e`, `cbf86e0e` · full `internal/cli/...` tree green,
`check-fast` clean.

### AC-3 — forbidigo ban + logging_chokepoint policy backstop

Mechanically migrated every remaining bare `fmt.Println`/`Print`/`Printf`/
`Fprintln(stdout|stderr)`/`Fprintf(stdout|stderr)` call site (38 files) to
the `cliutil` wrapper set — pure substitution, zero behavior change,
confirmed via the full test suite passing unchanged (commit `2ac84846`).
Added three `forbidigo` rules banning the destination-unambiguous bare
forms (`fmt.Println`/`Print`/`Printf`); confirmed empirically (a throwaway
probe rule + file, since forbidigo's actual matching semantics weren't
documented in this repo) that forbidigo matches only a call's callee
expression, never its arguments — it structurally cannot distinguish
`fmt.Fprintln(os.Stdout, …)` from `fmt.Fprintln(someOtherWriter, …)`.
Added `internal/policies/logging_chokepoint.go` (+ companion test) as the
independent AST-walking backstop that inspects the actual first argument,
mirroring `atomic_write_chokepoint.go`'s shape; wired into
`policies_test.go`'s self-check registry, confirmed clean against aiwf's
own repo. `wf-vacuity` mutation probes (disable the allowlist, narrow the
bare-form switch, always-match the writer check) all caught; one probe
survived undetected on first attempt (the writer-check mutation, since the
"arbitrary writer" fixture used a bare identifier that fails an earlier
guard before ever reaching the mutated line) — added a
non-stdio-but-still-`os.*`-selector fixture to close the gap. A later
pass (commit `cbc7f296`) closed one more real gap `wf-vacuity` had not:
`isOSStdioWriter`'s `pkg.Name != "os"` arm (a writer that's a selector
under some other package) had no test — found while investigating the
epic-level coverage-gate run for AC-4's Deferrals entry below, not part
of the original mutation-probe pass. · commits `14c81e3a`, `2ac84846`,
`83afce24`, `cbc7f296` · `check-fast` clean, self-check policy passes
against aiwf's own tree, `logging_chokepoint.go` 100% covered.

### AC-4 — Logging config moved into internal/config; doctor reporting

Added `Config.Logging` (schema-registered per G-0382, with `Schema()`/
`fieldDescriptions`/`AcceptedKeys()`/`GenerateExample()` anti-drift
coverage, not a second private decode path). `internal/logger` cannot
import `internal/config` (layering direction), so `cliutil.ResolveLogger`
copies the three parsed strings across as plain values after loading
`aiwf.yaml` itself. Added `logger.ResolveConfigWithSources` (per-field
env/yaml/default provenance, `ResolveConfig` now a thin wrapper over it)
so `aiwf doctor` can report not just the resolved level/format/destination
but which tier supplied each one — the `logging:` line, informational,
never a problem when disabled (the documented default-off state), an
error-severity problem when a value is genuinely invalid. `wf-vacuity`
mutation probes (skip yaml loading, invert the enabled/disabled branch,
disable the destination-display guard, ignore a non-nil config) all
caught. The independent design review (see Reviewer notes) found the
doc comments explaining why `Config.Logging` isn't typed as
`logger.YAMLConfig` directly stated the layering constraint backwards
(config importing logger is legal; it's the reverse that isn't), and
that the field-by-field copy was already duplicated in two call sites
with no compiler check against drift — commit `f31880da` corrects the
claim, gives the real reason (`schema.go`'s reflection walker needs a
`config.*`-prefixed type name to detect a nested block), and extracts
`Logging.ToYAMLConfig()` as the one conversion point both call sites
now use. · commits `dfcdd96b`, `f31880da` · full `internal/config` +
`internal/cli/doctor` + `internal/logger` suites green, `check-fast`
clean.

## Decisions made during implementation

- `upgrade`'s "verb.completed" fires the moment install succeeds, not
  on the verb's final exit code — unlike the other four instrumented
  sites, which gate on `code == cliutil.ExitOK`. This is a deliberate
  per-site classification (per AC-1's own design notes: made once per
  site, not re-litigated later), forced by `reexecUpdate`'s use of
  `syscall.Exec`: a successful reexec replaces the process image, so
  no Go code after a successful reexec ever runs — gating on the final
  exit code would mean the event almost never fires in a real
  production upgrade, only in the `AIWF_NO_REEXEC` test-only path.
  Surfaced by the wrap review (finding B1); pinned by a test in commit
  `7b31e85e`.

## Validation

`make check-fast` green throughout (build, lint 0 issues, full race test
suite) after every commit. `internal/cli/...` (172+ tests) and
`internal/config`/`internal/logger`/`internal/policies` all green.
`internal/policies.TestPolicy_LoggingChokepoint` (the self-check against
aiwf's own repo) passes clean, confirming the AC-3 migration left zero
non-allowlisted bare stdio prints anywhere in `internal/` or `cmd/`.

An epic-level `make coverage-gate` run (base `origin/main`, not required
for this milestone's own wrap per CLAUDE.md's cadence rule — only
`make check-fast` gates milestone work on an epic branch) surfaced ~194
pre-existing, never-exercised error-handling branches across 31 files
that AC-3's mechanical print-call rename incidentally touched (confirmed
via `git log -p` on a sample: only the print-call text changed, no
logic). Filed as G-0386 rather than fixed here — genuinely a different
concern (CLI-verb error-path coverage hygiene, unrelated to diagnostic
logging) at a scale (194 lines, 31 unrelated files) disproportionate to
this milestone. Two directly-in-scope gaps this same investigation
turned up were fixed inline instead of deferred: `logging_chokepoint.go`
had two real coverage gaps of its own (the writer-check mutation-probe
blind spot, and the `pkg.Name != "os"` arm), both closed with real tests
in commits `83afce24` and `cbc7f296`.

## Deferrals

- G-0386 — backfill test coverage for ~194 pre-existing, untested CLI
  verb error-handling branches across 31 files, surfaced (not caused) by
  AC-3's mechanical print-call migration. Recommended as a new epic
  branched from `main` directly, running in parallel with `E-0061` — the
  affected lines exist on `main` today in their pre-rename form, so the
  fix needs no coordination with this epic and will be inherited
  automatically via the coverage gate's merge-base recomputation if it
  lands first.

## Reviewer notes

Independent wrap review ran five fresh-context passes: three code-quality
slices (AC-1/AC-2, AC-3, AC-4) and two design-quality passes (the new
`cliutil` text-output convention; the config/logger layering-boundary
split). Verdicts and how each finding was handled:

- **AC-1/AC-2 — REQUEST CHANGES, both blocking findings fixed, plus two
  track-for-later items revisited and also fixed** (initially judged
  non-blocking and left as-is; reconsidered before this milestone merged
  anywhere, since both were cheap and directly relevant to what M-0239
  builds on next). B1 (`upgrade`'s event fires before the reexec,
  untested) and B2 (`statusline` missing seam tests the Work log claimed
  existed) — see the Decisions entry and the AC-1/AC-2 Work log update
  above. T2 (`statusline`'s bound `entity` field carried the project
  root path rather than a meaningful, correlation-worthy value — flagged
  independently by both the AC-1/AC-2 and the design reviewer) is now
  the `--scope` value ("user"/"project") instead, matching the other
  four sites' pattern of a real, small value — commit `b9631b76`. T3
  (AC-1's own body text described the event as
  `logger.Info("verb.<name>.completed", …)`, implying a per-verb event
  *string*, while the shipped implementation uses a constant
  `"verb.completed"` message plus a `verb` field — matching ADR-0017's
  own worked example, not a deviation from it) is reconciled in AC-1's
  text above. One informational observation, deliberately left as-is:
  (H1) one non-reproducing flake of
  `TestResolveLogger_EnvBeatsYAMLLoggingBlock`
  under a combined `-v` run, never reproduced across ~30 follow-up runs
  including `-race -count=20` — noted here in case it recurs, not
  chased further without a second occurrence.
- **AC-3 — APPROVE.** Independently re-verified (not just re-read) the
  G-0386 deferral judgment via `git log -p` on sampled files and forbidigo's
  actual matching semantics, and agreed it was the right call. Two stale-text
  nits: one (the AC-3 acceptance prose) turned out to already be corrected
  in the version the reviewer had checked out — no action needed; the other
  ("40 unrelated verbs" vs. G-0386's "31 files") is fixed in the Validation
  section above. Also flagged, as an accepted trade-off rather than a
  finding: the two forbidigo-allowlisted files (`outputformat.go`,
  `textio.go`) lose the pre-existing panic/os.Exit ban too, since
  golangci-lint's path exclusion is whole-linter, not per-rule — matches
  the existing `cmd/aiwf/main.go`/`verb/apply.go` precedent, surface is two
  trivial files, currently clean.
- **AC-4 — APPROVE**, no findings.
- **Design (`cliutil` textio convention) — keep as-is.** One real,
  evidence-backed finding not acted on: `cliutil.Errorf`/`Errorln` share a
  name with `fmt.Errorf` (different semantics — one writes to stderr, the
  other builds an `error` value), and 6 files already call both. Renaming
  would touch 280+ call sites for a readability improvement with no
  correctness risk (Go's package qualification prevents an actual compiler
  mixup) — judged disproportionate for this milestone; worth reconsidering
  if `textio.go` is touched again.
- **Design (config/logger boundary) — keep the type split, two real bugs
  fixed.** The inverted layering-direction doc comments and the
  already-duplicated field copy — both closed. See the AC-4 Work log
  update and Decisions-adjacent commit `f31880da` above.

G-0386's own body was updated (a separate `aiwf edit-body G-0386` commit)
with a caveat the AC-3 review raised: the "epic inherits the fix
automatically" framing needs the gap's own fix to land on `main` before
`E-0061` pushes — a real precondition to confirm at epic-wrap time, not an
automatic guarantee.
