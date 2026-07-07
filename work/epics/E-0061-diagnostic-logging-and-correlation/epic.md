---
id: E-0061
title: Diagnostic logging and correlation
status: active
---

# E-0061 — Diagnostic logging and correlation

## Goal

Give aiwf a retrace-ready diagnostic surface: opt-in structured logging plus
a correlation id that ties one invocation's JSON envelope to its own log
lines, so "why did this verb do that on someone else's repo?" has a real
answer instead of whatever the operator happened to capture from stderr.

## Context

CLAUDE.md's Go conventions currently prescribe `log/slog` to stderr at
default `INFO`; the code does something else — bare `fmt.Fprintln(os.Stderr,
…)` call sites across `internal/cli` and several verbs, no `slog` import
anywhere, and a JSON envelope (`render.Envelope`) whose
`metadata.correlation_id` slot is declared but never populated. ADR-0017
already resolved the design question (opt-in, default off, three env knobs,
one daily-rotated file under `$XDG_STATE_HOME`, structured fields, a
`forbidigo` chokepoint) and was subsequently amended to specify the one
interaction it had left open: the log file is a shared, append-only,
multi-writer stream, so its writer needs `O_APPEND` + one `Write()` call per
record, not this repo's usual temp+rename atomic-write discipline — and
`internal/policies/atomic_write_chokepoint.go` needs an explicit allowlist
entry to permit that.

This epic is the implementation half of that ADR, plus G-0232's
correlation-id wiring. It's scoped as its own epic — not folded into a larger
"robustness" epic — specifically so it can ship and merge to main on its own
schedule: it has standalone operator value regardless of what else gets
built afterward. It is epic 1 of 2 named in
[`docs/initiatives/robustness-correctness-stress-testing.md`](../../../docs/initiatives/robustness-correctness-stress-testing.md)'s
"Foundation: making findings retrace-able" section; a second, larger epic (a
correctness stress harness) depends on this one's milestones being done,
since the harness's own RCA value rests on the correlation id and logger
this epic builds.

## Scope

### In scope

- Ratify ADR-0017: promote it `proposed → accepted` once the implementation
  below lands and matches it.
- `internal/logger` package (G-0223): wraps `log/slog`; resolves
  `AIWF_LOG`/`AIWF_LOG_FORMAT`/`AIWF_LOG_FILE` then `aiwf.yaml`'s `logging:`
  block, falling back to a no-op discard handler when both are absent;
  derives the `$XDG_STATE_HOME/aiwf/logs/aiwf-YYYY-MM-DD.log` default
  destination with 30-day retention; exposes `WithVerb(verb, entity, actor)`
  binding a fresh `run_id`.
- The `logging:` block in `aiwf.yaml`, parsed and validated by
  `internal/aiwfyaml/`, surfaced through `aiwf doctor`.
- Migrate the named bare-stderr call sites (`internal/cli/statusline.go`,
  `internal/cli/root.go`, `internal/verb/move.go`, `internal/verb/cancel.go`,
  `internal/verb/upgrade.go`) to the bound logger where the call is a
  diagnostic event, leaving genuinely operator-facing warnings/errors on the
  existing `internal/cli/output` stderr path.
- The `forbidigo` rule banning bare `fmt.Println`/`fmt.Print`/
  `fmt.Fprintln(os.Stdout|os.Stderr, …)` outside the allowlist, landed
  together with `internal/policies/logging_chokepoint_test.go` so the
  discipline holds even if the linter rule is ever disabled.
- The `atomic_write_chokepoint.go` allowlist entry for `internal/logger`'s
  file writer, with a rationale comment pointing back at ADR-0017.
- G-0232: wire `render.Envelope.Metadata.correlation_id` end-to-end — the
  Cobra root mints a per-invocation id, every verb threads it into the
  envelope, and the same id becomes `logger.WithVerb`'s `run_id`; add
  per-verb-appropriate mutating-verb metadata (`promote` reports
  `entity_id`/`from`/`to`/`commit_sha`, etc.); land the `--trace` flag G-0232
  deferred until the logger existed.
- `internal/policies/envelope_structural_assertion.go`, pinning the
  envelope's required-key set against the `Envelope` struct's field tags.

### Out of scope

- The correctness stress harness itself and its scenario catalog — epic 2,
  which depends on this epic's milestones being done.
- Any performance work — a sibling initiative
  ([`check-performance-incremental-revwalk-cache.md`](../../../docs/initiatives/check-performance-incremental-revwalk-cache.md)),
  unrelated to this scope.
- journald/syslog/log-shipper integration — ADR-0017 explicitly defers this;
  operators reach for files and `tail -f`, not service-log conventions.
- Any CI or scheduling change; this epic changes what aiwf logs, not when
  anything runs.

## Constraints

- **Default-off, zero side effects.** No log file, no directory, is ever
  created under `$XDG_STATE_HOME/aiwf/logs/` unless the operator opts in via
  `AIWF_LOG` or the `logging:` block. An opted-out invocation pays no
  allocation cost beyond a discard-handler `Info` call.
- **`O_APPEND`, one `Write()` per record — non-negotiable.** Any buffered
  `io.Writer` that could split a single log record across two `Write()`
  calls voids the concurrent-append safety the ADR relies on and must not be
  used for this file handle.
- **The `forbidigo` rule and its policy-test backstop land together.** The
  rule alone, without `logging_chokepoint_test.go`, is not sufficient — if
  the linter is ever disabled, the discipline still needs to hold.
- **No secrets or path-leak regressions.** `WithVerb`'s field binding scrubs
  `os.Args` of `/Users/<name>/` and `/home/<name>/` paths before binding
  them, matching the gitleaks path-leak discipline already enforced
  elsewhere.
- **Never `==` against `correlation_id` for anything but exact-match
  correlation.** It's an opaque per-invocation identifier, not a
  business-logic key.

## Success criteria

- [ ] An operator can set `AIWF_LOG=debug` (optionally `AIWF_LOG_FILE`) and
      get a structured, greppable diagnostic trace for any verb, with no log
      file created when the operator hasn't opted in.
- [ ] An invocation's JSON envelope `metadata.correlation_id` matches the
      `run_id` bound into that same invocation's diagnostic-log lines.
- [ ] ADR-0017 reads `accepted`, with no remaining mismatch between its
      prescription and the shipped code.
- [ ] Every bare-stderr call site named in this spec's *Scope* is migrated or
      explicitly kept on the operator-facing path, with the `forbidigo`
      chokepoint and its policy-test backstop both active.
- [ ] The `internal/logger` file writer's concurrent-append safety (`O_APPEND`,
      one `Write()` per record) has a real concurrent-writer test proving no
      line is ever interleaved or torn under simultaneous processes.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Exact per-call-site classification (diagnostic event vs. operator-facing warning) for each named bare-stderr site | no | Decided per-site during the migration milestone, per G-0223's own note that this isn't predetermined |
| Whether epic 2's harness needs anything from this epic beyond the shipped `AIWF_LOG*`/`correlation_id` surface (e.g. a stable internal Go API) | no | Revisited when epic 2 is planned; not blocking this epic's completion |

## Risks (optional)

| Risk | Impact | Mitigation |
|---|---|---|
| `forbidigo` ships ahead of full call-site migration, tripping CI on a legitimate future print site with no clear escape hatch | med | Land the rule, the migration, and the policy test in the same milestone sequence; allowlist new legitimate sites with a one-line rationale, never a bare `//nolint` |
| The `O_APPEND` concurrent-append safety property is asserted in the ADR but stays unverified until a real test exists | med | Required as mechanical evidence (per CLAUDE.md's AC-promotion rule) before any AC claiming this property is promoted `met` |

## Milestones

- `M-0237` — Logger core: `internal/logger` package, env/YAML resolution
  precedence, no-op-when-off discard handler, XDG-state-home daily-rotated
  file, the `O_APPEND`/one-`Write()`-per-record concurrent-append discipline
  and its concurrent-writer test. · depends on: —
- `M-0238` — Migrate bare-stderr call sites; forbidigo chokepoint: migrate
  the named bare-stderr call sites; wire the `forbidigo` rule plus
  `logging_chokepoint_test.go`; add the `atomic_write_chokepoint.go`
  allowlist entry; `aiwf doctor` surfaces the resolved `logging:`
  configuration. · depends on: `M-0237`
- `M-0239` — Correlation id wiring; ratify ADR-0017: wire `correlation_id`
  end-to-end, mutating-verb metadata, the `--trace` flag,
  `envelope_structural_assertion.go`; ratify ADR-0017. · depends on:
  `M-0237`, `M-0238`

## ADRs produced (optional)

- ADR-0017 — Opt-in slog diagnostic logging, default off, XDG state-home
  file route (ratified by this epic)

## References

- [`docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md`](../../../docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md)
- [`docs/initiatives/robustness-correctness-stress-testing.md`](../../../docs/initiatives/robustness-correctness-stress-testing.md)
- G-0223 — implement ADR-0017 opt-in slog logging; migrate bare-stderr call sites
- G-0232 — envelope enrichment: correlation_id wiring + mutating-verb metadata
