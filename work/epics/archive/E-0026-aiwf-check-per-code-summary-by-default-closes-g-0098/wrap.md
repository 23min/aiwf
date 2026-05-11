# Epic wrap — E-0026

**Date:** 2026-05-11
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** milestone/M-0089-per-code-text-render-summary-with-verbose-fallback (single-milestone epic; no separate integration branch)
**Merge commit:** (filled post-merge)

## Milestones delivered

- M-0089 — Per-code text-render summary with `--verbose` fallback (implementation `de94cf9`, promote `0e2112c`)

## Summary

Replaced the flat per-finding text render of `aiwf check` with a per-code summary by default while preserving error-per-instance behaviour and adding a `--verbose` flag that restores the full pre-epic shape byte-for-byte. JSON envelope output is unchanged modulo `metadata.root` (which is environmental). On this repo, the change collapses the post-E-0024 advisory state (~176 near-identical `terminal-entity-not-archived` lines + paired `archive-sweep-pending` aggregate) down to a 5-line default output that fits one screen and answers "is there anything new I should look at?" without scrolling. The friction the epic targets — observable since E-0024 (uniform archive convention) landed — is closed at the render layer alone, with zero changes to check rules, severities, or finding codes.

## ADRs ratified

- none

## Decisions captured

- AC-4 mid-flight relaxation: strict byte-identity → structural-equal modulo `metadata.root` (in `M-0089-…/Decisions made during implementation`). The relaxation is principled (the field is environmental — absolute path of the resolved consumer repo, never identical across hosts), the contract is still tight (`cmp.Diff` on parsed envelopes proves every per-finding field is unchanged; the field's presence + non-emptiness is asserted), and the JSON contract stays intact because the renderer is the stdlib's deterministic `encoding/json`. No ADR — the decision is M-0089-scoped, not durable architecture.

## Follow-ups carried forward

- none — every success criterion in the epic spec is met; no scope was cut or deferred.

## Validation

`aiwf check` on the milestone branch immediately before merge:

```
terminal-entity-not-archived (warning) × 176 — entity ADR-0002 has terminal status "rejected" but file is still in the active tree; awaiting `aiwf archive --apply` sweep
archive-sweep-pending (warning) × 1 — 176 terminal entities awaiting `aiwf archive --apply`. Set `archive.sweep_threshold` in aiwf.yaml to escalate to blocking past N
provenance-untrailered-scope-undefined (warning) × 1 — no upstream configured and no --since <ref>; provenance audit skipped

178 findings (0 errors, 178 warnings)
```

5 lines total, well under the AC-7 ≤10 bound. Zero error-severity findings. `golangci-lint run`: 0 issues. `go build`: clean. Targeted test re-runs (`internal/render/`, the narrow-id policy) green; full-suite re-run skipped per the user's standing G-0097 ack.

## Doc findings

doc-lint scope is the milestone change-set (Go source + the milestone spec). No `docs/` files touched. The milestone spec's *Design notes* and *Surfaces touched* sections still name the pre-implementation path predictions (`internal/render/text.go`, `cmd/aiwf/check.go`); the spec's *Decisions made during implementation* section reconciles them to the actual files (`internal/render/render.go`, `cmd/aiwf/main.go`). Deliberately left as a historical record of the friction — the correction is recorded once, in the right structural place. doc-lint: clean.

## Reviewer notes

The AC-4 relaxation is the only reviewer-relevant decision shape. It is documented in M-0089's *Decisions* section in adequate detail (what was relaxed, why, the structural compare that replaces strict byte-identity, and the contract the relaxation does not weaken). No ADR was opened because the decision is milestone-scoped — it pins how a specific AC is verified, not durable architecture. If a future reader is surprised that JSON output's `metadata.root` differs across hosts, the M-0089 spec is the entry point.

## Handoff

Ready for the next epic. The kernel's archive sweep (the work `archive-sweep-pending` keeps surfacing) remains a deliberate open item — operators run it when the cumulative noise of the warning surface justifies the bookkeeping. With this epic landed, the surface itself is one line per code instead of 176 leaves, so the pressure to sweep at any specific cadence is lower; the warning still names what's pending whenever a check is run.
