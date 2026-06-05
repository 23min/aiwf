---
id: G-0232
title: 'Envelope enrichment: correlation_id wiring + mutating-verb metadata'
status: open
---
## What's missing

Two declared-but-unwired envelope slots and one optional debug surface:

1. **`render.Envelope.Metadata.correlation_id` is dead code.** The slot is documented at `internal/render/render.go:26-49` but no caller populates it. Either wire it through (Cobra root mints a per-invocation UUID; every verb threads it into `render.Envelope.Metadata.correlation_id`; the same id goes into the ADR-0017 `logger.WithVerb(... "run_id", id)` binding) or remove the declaration. The "wire it" path is cleaner because it makes envelopes and (future) slog lines cross-referenceable; the "remove it" path is cleaner if no consumer asks for it.
2. **Mutating verbs emit no per-invocation metadata.** Today only read-only verbs populate the envelope's `metadata` map (e.g. `aiwf check` reports counts/timing). Mutating verbs (`promote`, `archive`, `add`) emit nothing. Add per-verb-appropriate metadata: `aiwf promote` reports `entity_id`, `from`, `to`, `commit_sha`; `aiwf archive` reports `swept_count`, `commit_sha`; etc. The shape is per-verb but the discipline is uniform.
3. **Optional `--trace` debug flag** emitting per-phase timings via slog. Depends on ADR-0017 (G-0223) landing first — the trace output is a logger consumer, not an envelope consumer. Defer until then; cite the dependency.
4. **`internal/policies/envelope_structural_assertion.go`** — pins the envelope's required-key set against the `Envelope` struct field tags. Catches the case where a future field rename breaks the JSON shape that downstream tooling consumes.

## Why it matters

G3's verdict was Strong "by virtue of the git-commit-as-record model," with the adversarial pass noting the declared-but-unwired `correlation_id` and the asymmetric metadata coverage as real-but-immaterial-for-a-one-commit-CLI gaps. They become material the moment any downstream tool (CI dashboard, log aggregator, future `aiwf trace`) wants to correlate envelopes across a multi-verb session.

## Source

`docs/pocv3/health-scorecard-2026-06-04.md` §G3 (moves 1–3; refuting evidence on correlation_id + mutating-verb metadata), §B2 (move 1: envelope structural policy).
