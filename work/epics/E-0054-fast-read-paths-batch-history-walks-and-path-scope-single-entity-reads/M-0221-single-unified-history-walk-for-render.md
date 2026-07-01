---
id: M-0221
title: Single unified history walk for render
status: draft
parent: E-0054
tdd: required
acs:
    - id: AC-1
      title: render resolves all entity histories from a single git-history pass
      status: open
      tdd_phase: red
    - id: AC-2
      title: provenance and scope views resolve from the shared pass, not per-milestone
      status: open
      tdd_phase: red
    - id: AC-3
      title: rendered site byte-identical before and after the refactor
      status: open
      tdd_phase: red
    - id: AC-4
      title: measured render wall-time delta recorded in Validation
      status: open
      tdd_phase: red
---
## Goal

Replace render's per-entity git-history fan-out with one shared single-pass walk,
covering **both** walk families the spike identified:

1. **Per-entity events** — `resolver.history(id)`, called N+2× per milestone (once
   per AC composite `M-NNNN/AC-N`, plus the commits table, plus the provenance
   timeline).
2. **Provenance/scopes** — `show.LoadEntityScopeViews(m.ID)`, which itself runs a
   per-milestone `history.ReadHistory` plus a full `readAllAuthorizeOpeners` grep.

On the kernel tree that is ~1,000+ subprocesses and 28 minutes. Feed the per-entity
event lists (bucketed by `aiwf-entity` / `aiwf-prior-entity`) and the authorize-opener
map from one `BulkRevwalk`-shaped HEAD walk. The spike proved 12.8s, byte-identical
across all 657 pages.

## Notes

- Ref scope must match `ReadHistoryChain` (HEAD, not `--all`) so output is preserved.
- The bare-milestone query must still fold in its `M-NNNN/AC-N` AC events; width
  tolerance (`E-22` vs `E-0022`) handled by canonicalizing both sides.
- Batching history alone still timed out — the provenance/scope family must be
  batched in the same change, ideally from the same pass.
- The throwaway spike (`resolver_bulkspike.go`, reverted) is the reference
  implementation; productionize with tests, don't ship the env-gated form.

### AC-1 — render resolves all entity histories from a single git-history pass

### AC-2 — provenance and scope views resolve from the shared pass, not per-milestone

### AC-3 — rendered site byte-identical before and after the refactor

### AC-4 — measured render wall-time delta recorded in Validation

