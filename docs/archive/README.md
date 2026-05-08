# `docs/archive/` — pre-PoC design documents and historical procedural artifacts

This directory holds two kinds of historical material:

1. **Pre-PoC design documents** describing the framework's *original* ambition — an event-sourced kernel with hash-verified projections, monotonic IDs, RFC 8785 canonicalization. The research arc in [`../research/`](../research/) walked most of that back; the working framework on the trunk today is materially smaller in shape.
2. **Procedural artifacts** from one-time historical events worth preserving the reasoning of — e.g., the trunk-promotion plan that drove the merge of `poc/aiwf-v3` into `main`.

Both kinds are kept because the reasoning is useful, not because they describe the active design. New readers should start with [`../working-paper.md`](../working-paper.md).

## Contents

- [`architecture.md`](architecture.md) — original event-sourced kernel design.
- [`build-plan.md`](build-plan.md) — original sequenced build plan that targeted the event-sourced design.
- [`promotion-2026-05-08.md`](promotion-2026-05-08.md) — working plan that drove the trunk-promotion procedure (`poc/aiwf-v3` → `main`) on 2026-05-08.
