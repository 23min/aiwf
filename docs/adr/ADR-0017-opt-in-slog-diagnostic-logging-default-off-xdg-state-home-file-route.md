---
id: ADR-0017
title: Opt-in slog diagnostic logging, default off, XDG state-home file route
status: proposed
---
## Context

CLAUDE.md's Go conventions §CLI conventions currently states: *"Logging: `log/slog` to stderr (default level `INFO`). Tool output goes to stdout."* The codebase doesn't reflect that prescription. `internal/cli/output` carries the structured JSON envelope for operator-facing tool output; warnings and errors go to stderr via bare `fmt.Fprintln(os.Stderr, …)` interpolation across `statusline.go`, `root.go`, `move.go`, `cancel.go`, `upgrade.go`, and elsewhere. There is no `log/slog` import, no structured-field discipline, no log-event capture in tests, and no opt-in surface for diagnostic logs.

The 2026-06-04 codebase health scorecard (E1 verdict: Weak; `docs/pocv3/health-scorecard-2026-06-04.md`) named the mismatch directly: design intent and implementation disagree on the diagnostic-log surface, and the disagreement has persisted through every kernel milestone that touched stderr. Either path closes the gap; neither has been chosen.

Two related-but-distinct things are bundled together in the current prose:

1. **Operator-facing CLI output** — what `aiwf check`, `aiwf promote`, `aiwf doctor` print so the human sees what happened. Stdout for verb payload (text by default, JSON envelope under `--format=json`); stderr for warnings and errors the operator must read right now. This is UI. It is not opt-in; it is the verb's primary output surface.

2. **Diagnostic logs** — what a developer reaches for when answering *"why did this verb do that on someone else's repo?"* Structured-field events bound to a verb / entity / actor / run-id. Necessary when debugging a flaky `aiwf check` across a consumer's tree, or when correlating a multi-verb session in CI.

The current prose conflates these by sending both to "stderr at INFO." A CLI tool's stderr is for #1; #2 belongs on a separately-routed surface that the operator opts into.

Comparable CLIs (git, gh, kubectl, terraform, docker, cargo) all default diagnostic logging OFF, route via env or a flag, and offer both stderr and file destinations the operator picks. None of them mix the two on the same channel by default.

## Decision

aiwf's diagnostic-log surface is **opt-in, default OFF**, distinct from the verb's operator-facing stdout/stderr output. The decision narrows CLAUDE.md's "log/slog to stderr default INFO" prescription to:

1. **`log/slog` is the diagnostic-log library.** No bare `fmt.Println` / `fmt.Fprintln` for diagnostic events outside `cmd/aiwf/main.go` and the operator-facing stdout path in `internal/cli/output`. Operator-facing stderr (warnings, errors) remains the verb's responsibility and stays on `os.Stderr` via the same `internal/cli/output` surface; that channel is not the diagnostic-log channel.

2. **Default level OFF (effectively `error`-only).** The kernel emits no diagnostic events at `info` or `debug` unless the operator opts in. Errors and warnings the operator must see remain on the verb's stderr surface, not the diagnostic-log surface.

3. **Three routes to opt in, in precedence order (env beats YAML beats default):**
   - `AIWF_LOG=error|warn|info|debug` — level. Env-only is enough for ad-hoc debugging.
   - `AIWF_LOG_FORMAT=text|json` — text by default, JSON when piped to a log aggregator.
   - `AIWF_LOG_FILE=/absolute/path` or `AIWF_LOG_FILE=stderr` — explicit destination override.
   - `aiwf.yaml` may carry the same three keys under a top-level `logging:` block for repo-scoped defaults (per-repo opt-in, e.g. a contract-verification repo that wants `debug`-level logs in CI). All keys optional; absent means default-off.

4. **Default destination when `AIWF_LOG` is set but `AIWF_LOG_FILE` is not: `$XDG_STATE_HOME/aiwf/logs/aiwf-YYYY-MM-DD.log`** (UTC date, one file per day, per host, per user). Fallback to `~/.local/state/aiwf/logs/aiwf-YYYY-MM-DD.log` when `XDG_STATE_HOME` is unset. Retention: 30 days, oldest deleted on the next aiwf invocation that touches the dir. No log file is ever created under the consumer's repo or `.claude/`; diagnostic logs are per-user state, never committed, never shared.

5. **Structured fields, not interpolation.** Every emit binds context: `logger.Info("verb.commit", "verb", "promote", "entity", "M-0090", "sha", short)`. Never `logger.Info(fmt.Sprintf("promote committed %s for %s", sha, entity))`. The structured form is greppable, JSON-renderable, and survives a future "I want this in a dashboard" pivot.

6. **Per-verb bound logger.** Every top-level verb starts with `logger := slog.With("verb", v, "entity", id, "actor", actor, "run_id", uuid())`. Sub-functions log against the bound logger. The `run_id` is the same value that goes into the JSON envelope's `metadata.correlation_id` so an envelope line and a log line are cross-referenceable.

7. **Logs are testable.** Tests capture the slog handler and assert `"verb.commit fired once with entity=M-0090"` — the rubric's E1 "capture log events in tests so you can assert this event fired with these fields" requirement.

8. **`forbidigo` chokepoint** bans bare `fmt.Println` / `fmt.Print` / `fmt.Fprintln(os.Stdout|os.Stderr, …)` outside an explicit allowlist (`cmd/aiwf/main.go`, the human-text branch in `internal/cli/output/outputformat.go`, golden-file regeneration helpers). The forbidigo rule is the load-bearing piece; without it, the discipline rots back to one-of.

## Consequences

- **CLAUDE.md amended.** The Go conventions §CLI conventions paragraph on logging is rewritten to reflect this decision: opt-in default-off, three env knobs, XDG-state-home daily file, slog with structured fields, forbidigo enforcement. The "tool output goes to stdout" half of the current paragraph stays — that part is correct and unchanged.

- **`internal/logger` package introduced.** Wraps `log/slog` with the env/YAML resolution, the XDG-path default, the daily rotation, the retention sweep, and a `WithVerb(verb, entity, actor)` constructor. Single dependency; standard library only.

- **`logging:` block added to `aiwf.yaml` schema.** Optional. All three keys optional. The validator under `internal/aiwfyaml/` recognizes the block; absence means default-off; positive `AIWF_LOG_*` env vars override.

- **No log file is created when the operator hasn't opted in.** An empty `$XDG_STATE_HOME/aiwf/logs/` directory is never materialized as a side effect of running `aiwf` — only as a side effect of the operator setting `AIWF_LOG=…` (or the `aiwf.yaml` `logging:` block).

- **The `internal/cli/output` operator-facing surface is unaffected.** Warnings and errors continue to flow through it to stderr. The JSON envelope is untouched. This ADR introduces a parallel diagnostic surface; it does not refactor the existing operator-facing one.

- **Performance.** When logging is off (the default), the slog handler is a no-op discard handler — zero allocations at the emit site beyond the closed-form `Info` call. The cost is paid only when an operator opts in.

- **Secrets and path-leak hygiene.** Diagnostic logs respect the same path-leak discipline gitleaks polices at the pre-commit hook (CLAUDE.md §What's enforced and where). The logger's `WithVerb` constructor scrubs `os.Args` of `/Users/<name>/` and `/home/<name>/` paths before binding them. Stack traces and full file paths log only at `debug` level.

- **No per-invocation log files.** aiwf verbs run frequently inside TDD sessions (dozens of `aiwf check` calls per cycle); per-invocation timestamped log files would litter the directory. Daily rotation gives `grep` a natural shard and bounds the directory size.

- **No journald / syslog / log-shipper integration in scope.** aiwf is a developer tool; operators reach for files (`tail -f`) and stderr, not service-log conventions. A future ADR can add a shipper if a real consumer needs it.

- **The implementation work is a separate concern.** A gap covers the migration from the current bare-`fmt.Fprintln` state to the slog surface this ADR ratifies. The gap names the call sites to migrate, the forbidigo wiring, the policy test, and the CLAUDE.md prose edit. This ADR does not schedule that work.
