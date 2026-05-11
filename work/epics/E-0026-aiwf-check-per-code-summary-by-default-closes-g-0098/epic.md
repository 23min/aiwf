---
id: E-0026
title: aiwf check per-code summary by default (closes G-0098)
status: done
---

# E-0026 — `aiwf check` per-code summary by default

## Goal

Change the default text output of `aiwf check` from one line per finding to one line per finding-code with a count and a sample message. Errors continue to print per-instance (each is actionable); warnings collapse to per-code summaries; `--verbose` prints the full unaggregated detail. JSON envelope output is unchanged — machines still get every finding.

## Context

E-0024 (uniform archive convention) landed on 2026-05-11. The kernel tree now produces a pre-sweep advisory state of **176 `terminal-entity-not-archived` warnings + 1 `archive-sweep-pending` aggregate**, by design — until the consumer runs the historical migration via `aiwf archive --apply`. Until that moment, every `aiwf check` invocation dumps 177 lines, of which 176 are near-identical "entity X has terminal status Y but file is still in the active tree" advisories with the same actionable hint.

The friction surfaced immediately post-merge: `aiwf check` became unscannable. The root issue is older than E-0024 — the text renderer has always concatenated every finding flat — but the dogfooded warning count was 0–2 historically, so the friction was latent. E-0024 made it observable.

The aggregate finding pattern (one `archive-sweep-pending` summarizing 176 `terminal-entity-not-archived` leaves) is the right shape; the renderer just needs to lean on it. Per-code summary by default makes the surface match `aiwf status`'s one-screen discipline.

## Scope

### In scope

- New default text-render shape for `aiwf check`: one line per finding-code with `(severity) × N` count and one representative sample message.
- Errors continue to print per-instance — each error is per-instance-actionable. Summary form applies only to warnings.
- `--verbose` flag on `aiwf check` that flips back to the current full-detail behavior.
- JSON envelope unchanged — machines still receive every finding via `--format=json`.
- `--quiet` is **not** in scope; that's an additive discoverability change deferred to a downstream gap if/when a CI consumer asks for it.
- Tests covering the four observable shapes: default-summary, `--verbose`, error-still-full, JSON-full.

### Out of scope

- Aggregating findings in the JSON envelope. The JSON view is machine-consumed; consumers can summarize themselves. No envelope-shape change.
- Severity-tiered output beyond "errors full vs. warnings summarized." A `--quiet` mode, a `--severity error` filter, etc. are downstream gaps.
- Re-rendering the existing `archive-sweep-pending` aggregate. The aggregate stays an emitted finding; this epic just changes how the renderer groups peers.
- Changes to any check rule (no new findings, no rule-scoping changes).
- Changes to `aiwf status`'s output (already one-screen by design).
- Removing the per-leaf finding for `terminal-entity-not-archived`. The leaves are addressable detail under `--verbose`; consumers may want them.

## Constraints

- **JSON output stays unchanged.** The full finding list is the machine-readable contract; consumers depend on it.
- **Exit code semantics unchanged.** `0` ok, `1` findings — the renderer is independent of exit-code policy.
- **Errors continue to print per-instance in default text.** A consumer scanning for errors should not have to use `--verbose` to see what failed.
- **Backwards-compatible CLI surface.** Adding `--verbose` is a new flag; existing flag-less invocations get the new (more terse) default. Anyone who needs the old behavior writes `--verbose`. No silent drop.
- **One mutating verb produces exactly one commit** still holds (this epic doesn't add mutating verbs; the check verb is read-only).

## Success criteria

- [ ] `aiwf check` against the kernel tree in its current pre-sweep state outputs ≤10 lines total (well under one screen), names each finding-code present, and reports the count for each.
- [ ] `aiwf check --verbose` against the same tree reproduces the current 177-line output verbatim — same finding text, same ordering.
- [ ] `aiwf check --format=json` (with or without `--verbose`) produces an envelope whose `findings` array contains every finding individually. Byte-identical to the pre-epic baseline.
- [ ] A fixture tree containing both an error-severity and a warning-severity finding produces default output that shows the error in full and the warning summarized.
- [ ] `aiwf check --help` documents the new `--verbose` flag with one-line description; the example block shows both default and `--verbose` invocation.
- [ ] The drift-prevention test in `cmd/aiwf/completion_drift_test.go` passes — the new flag is wired through Cobra's flag set.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Should the per-code summary sort by count (descending) or by code (alphabetical)? | no | Pick during the milestone; descending-by-count surfaces the volume offenders first, which is what scanning is for. |
| Should the summary line include a hint excerpt, or just the code + count + sample message? | no | Pick during the milestone; the existing hint text is long, and including it inflates each summary line back toward the noise we're escaping. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| A CI consumer or script parses default text output and breaks on the new shape. | low | The kernel ships JSON output for machine consumption; text output has no stability commitment. Document the shape change in CHANGELOG once tagged. |
| A future aggregate finding emits a different per-leaf shape and the summary heuristic mis-groups them. | low | The summary groups strictly by `Code` field; aggregates and leaves have distinct codes by convention. New rules opt into the leaf-shape via code naming, not via renderer special-casing. |

## Milestones

- [M-0089](work/epics/E-0026-aiwf-check-per-code-summary-by-default-closes-g-0098/M-0089-per-code-text-render-summary-with-verbose-fallback.md) — Per-code text-render summary + `--verbose` flag · depends on: —

## References

- G-0098 — gap this epic closes.
- E-0024 (done) — surfaced the friction observably; the 176-advisory pre-sweep state is the worked example.
- `internal/check/check.go` — `Run` returns `[]Finding`; the call sites in `cmd/aiwf/check.go` (or wherever the check verb lives) render the result.
- `internal/render/` — text and JSON render adapters.
- CLAUDE.md "CLI conventions" §JSON envelope — the machine-readable contract that stays unchanged.
- CLAUDE.md "Render output must be human-verified before the iteration closes" — the rule M-0087's visibility regression hit; applies here too (run the binary against the kernel tree before declaring the milestone done).
