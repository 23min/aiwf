# `docs/archive/` — pre-PoC design documents

This directory holds design documents from the framework's *original* ambition — an event-sourced kernel with hash-verified projections, monotonic IDs, RFC 8785 canonicalization. The research arc in [`../research/`](../research/) walked most of that back; the working PoC under `poc/aiwf-v3` (graduating to trunk via the `PROMOTION-PLAN.md` procedure) is materially smaller in shape.

These documents are kept because the reasoning is useful — a future version may revisit parts of it — but they do not describe the active design. New readers should start with [`../working-paper.md`](../working-paper.md).

## Contents

- [`architecture.md`](architecture.md) — original event-sourced kernel design.
- [`build-plan.md`](build-plan.md) — original sequenced build plan that targeted the event-sourced design.
