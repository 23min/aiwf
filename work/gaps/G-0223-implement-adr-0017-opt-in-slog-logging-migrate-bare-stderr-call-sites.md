---
id: G-0223
title: Implement ADR-0017 opt-in slog logging; migrate bare-stderr call sites
status: open
---
## What's missing

The implementation work that ratifies ADR-0017 against the current codebase. ADR-0017 chose opt-in `log/slog` diagnostic logging with default-off behavior, XDG-state-home daily-rotated file destination, and a `forbidigo` chokepoint banning bare `fmt.Fprintln(os.Stderr, …)` outside an allowlist. Today none of that exists: no `internal/logger`, no `logging:` block in `aiwf.yaml`, no slog import anywhere, no policy test, no CLAUDE.md amendment.

Scope of the migration work:

1. **`internal/logger` package.** Wraps `log/slog`. Resolves the operator's opt-in from env (`AIWF_LOG`, `AIWF_LOG_FORMAT`, `AIWF_LOG_FILE`) then from `aiwf.yaml`'s `logging:` block, falling back to a no-op discard handler when both are absent. When opted in with no explicit file destination, derives `$XDG_STATE_HOME/aiwf/logs/aiwf-YYYY-MM-DD.log` (fallback `~/.local/state/aiwf/logs/aiwf-YYYY-MM-DD.log`), creates the directory, opens the file in append mode. Implements 30-day retention by sweeping the directory on logger init. Exposes `WithVerb(verb, entity, actor string) *slog.Logger` that returns a bound logger with a fresh `run_id` matching the envelope's `metadata.correlation_id`.

2. **`logging:` block in `aiwf.yaml`.** Optional top-level block; three optional keys (`level`, `format`, `destination`). Recognized by `internal/aiwfyaml/`'s parser; default-off when absent. Env beats YAML; YAML beats default.

3. **Migrate bare-stderr call sites.** The known call sites (the scorecard's E1 evidence): `internal/cli/statusline.go`, `internal/cli/root.go`, `internal/verb/move.go`, `internal/verb/cancel.go`, `internal/verb/upgrade.go`. Verbs that should keep their operator-facing stderr stay on `internal/cli/output`'s warning/error path; verbs that emit diagnostic events route through `logger.Info("verb.<event>", …)`. The classification per call site is part of the milestone, not predetermined here.

4. **`forbidigo` rule.** Bans bare `fmt.Println`, `fmt.Print`, and `fmt.Fprintln(os.Stdout|os.Stderr, …)` outside an explicit allowlist (`cmd/aiwf/main.go`, the human-text branch in `internal/cli/output/outputformat.go`, golden-file regeneration helpers). Configured in `.golangci.yml` under the existing forbidigo block.

5. **`internal/policies/logging_chokepoint_test.go`.** AST-walks `internal/` and `cmd/` for `fmt.Fprintln(os.Stderr, …)` / `fmt.Println(…)` / `fmt.Print(…)` calls. Fails CI on any non-allowlisted call. Same shape as `PolicyNoHardcodedEntityPaths` — the policy test is the load-bearing companion to the linter rule; if `forbidigo` is ever disabled, the policy still fires.

6. **CLAUDE.md amendment.** Rewrite the Go conventions §CLI conventions paragraph on logging to reflect ADR-0017. Keep the "tool output goes to stdout" half; replace the "log/slog to stderr default INFO" half with the opt-in slog summary plus a link to ADR-0017.

7. **Test the seam, not just the layer** (CLAUDE.md §Go conventions). At least one test per migrated verb that drives the verb through its dispatcher with `AIWF_LOG=info` set, captures the slog handler, and asserts the expected `verb.<event>` fires with the bound fields (verb, entity, actor, run_id). This is the rubric's E1 "capture log events in tests" requirement made concrete.

Reconfirmed by the 2026-06-04 codebase health scorecard (E1 verdict: Weak; see `docs/pocv3/health-scorecard-2026-06-04.md`).

## Why it matters

CLAUDE.md currently states one thing about logging and the code does another. That mismatch is itself the gap: a new reader (human or LLM) believes the prose, reaches for `slog`, finds it absent, and either (a) re-derives the convention by reading the existing stderr call sites or (b) adds the first slog import in a one-off and creates a third pattern. Both outcomes drift the convention further.

Beyond convention hygiene, the practical pain ADR-0017 was written to address is "why did this verb do that on someone else's repo?" Today the only answer is the JSON envelope's `metadata` and whatever the operator captured from stderr at the moment of the run. A flaky `aiwf check` on a consumer's tree with hundreds of entities, against a `tdd: required` milestone with a failing AC promote, leaves the operator with no replay surface. `AIWF_LOG=debug AIWF_LOG_FILE=/tmp/run.log aiwf check` followed by `grep` is the workflow the kernel should support and currently doesn't.

A second, smaller pain is test-side: today an assertion that a verb "logged this thing" is hand-rolled per call site or skipped entirely. The slog handler-capture pattern (D2-shape conformance suites for log events) is what makes "logs are testable" mechanical.

## Path

Plan as a milestone under whichever epic owns it (likely a new "kernel hygiene" epic, or appended to a fitting in-flight one). Suggested AC sequence:

- **AC-1.** `internal/logger` package with no-op-when-off behavior; unit tests over the resolution-precedence matrix (env beats YAML beats default).
- **AC-2.** `logging:` block in `aiwf.yaml` parsed, validated, surfaced through `aiwf doctor` so the operator can confirm what's active.
- **AC-3.** Migrate the named bare-stderr call sites; per-site test asserts the slog event fires with bound fields when `AIWF_LOG=info`.
- **AC-4.** `forbidigo` rule wired; `internal/policies/logging_chokepoint_test.go` fails CI on non-allowlisted bare-print sites.
- **AC-5.** XDG-state-home file destination, daily rotation, 30-day retention sweep.
- **AC-6.** CLAUDE.md prose updated; cross-link to ADR-0017.
- **AC-7.** End-to-end test: a multi-verb session with `AIWF_LOG=info` produces a single dated log file under `$XDG_STATE_HOME/aiwf/logs/` containing structured events for every verb, with `run_id` matching the per-verb JSON envelope.

The forbidigo rule lands together with the policy test (AC-4) — the discipline is mechanical from the moment the first migrated call site commits.
