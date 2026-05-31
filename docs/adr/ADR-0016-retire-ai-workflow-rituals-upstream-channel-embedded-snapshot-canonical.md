---
id: ADR-0016
title: Retire ai-workflow-rituals upstream channel; embedded snapshot canonical
status: accepted
---
## Context

[ADR-0014](./ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) retired the Claude marketplace plugin as the rituals distribution channel and replaced it with embed-and-materialize: the rituals are vendored from upstream `https://github.com/23min/ai-workflow-rituals` into `internal/skills/embedded-rituals/`, embedded into the aiwf binary via `go:embed`, and materialized into the consumer's `.claude/` at `aiwf init` / `aiwf update`.

That landing left the *distribution* channel single (embedded) but kept the *authoring* channel split: ritual content is authored upstream at `ai-workflow-rituals`, pinned in `rituals.lock`, vendored via `make sync-rituals`, and policed by `TestRituals_VendoredMatchesUpstream` against the pinned ref.

The split-authoring model was a load-bearing assumption when marketplace distribution was live — third-party consumers might pin different upstream refs; a separate authoring repo let the marketplace artifact evolve independently of the kernel. Post-ADR-0014, none of those conditions hold:

- **Single consumer.** aiwf is the only known consumer of `ai-workflow-rituals`. The rituals are embedded into the aiwf binary; no other tool fetches the upstream.
- **Single distribution channel.** Embed-and-materialize means the binary IS the distribution. There is no parallel channel that would pin a different ref.
- **Cross-repo coordination is real friction.** Every ritual edit becomes a 2-repo dance: edit upstream → commit upstream → push upstream → bump `rituals.lock` → `make sync-rituals` → commit kernel-side. The G-0192 wf-patch session hit this directly while trying to add a missing trailer stamp to `aiwfx-wrap-milestone`.
- **Three authoring surfaces.** Ritual content currently lives at:
  1. Upstream `23min/ai-workflow-rituals/plugins/<plugin>/skills/<skill>/SKILL.md` (the *de jure* canonical source).
  2. Vendored snapshot at `internal/skills/embedded-rituals/plugins/<plugin>/skills/<skill>/SKILL.md` (what the binary embeds).
  3. Per-AC content-assertion fixtures at `internal/policies/testdata/<skill>/SKILL.md` (what AC tests assert against; [G-0182](../../work/gaps/G-0182-consolidate-testdata-ritual-fixtures-onto-the-embedded-snapshot-dedupe.md) calls this out as the third drift surface).

  The drift between (1) and (2) is policed by `TestRituals_VendoredMatchesUpstream`. G-0182's resolution collapses (3) onto (2). With this ADR's resolution, (1) collapses onto (2) as well — leaving exactly one authoring surface.
- **The kernel principle that nails it.** *"No plugin architectures for a single implementation."* The upstream authoring channel IS a plugin architecture for a single implementation.

The trade-off the split-authoring model still nominally provides — *"someone could author a ritual without cloning the whole aiwf kernel"* — is speculative. No one has done so. If a future hypothetical consumer needed to fork the rituals, the embedded snapshot directory is just markdown; forking it is no harder than forking a separate repo.

## Decision

**Retire `https://github.com/23min/ai-workflow-rituals` as an authoring channel. Make `internal/skills/embedded-rituals/` THE canonical authoring location for ritual content.**

Concretely:

- Ritual edits are authored directly in `internal/skills/embedded-rituals/plugins/<plugin>/skills/<skill>/SKILL.md` as one commit on the aiwf kernel repo. No cross-repo coordination.
- `rituals.lock`, `scripts/sync-rituals.sh`, the `sync-rituals` Make target, and `TestRituals_VendoredMatchesUpstream` retire. The drift class they policed goes away because there is only one source.
- The GitHub repo `23min/ai-workflow-rituals` is archived (read-only) with a README pointer to `23min/aiwf` at `internal/skills/embedded-rituals/`. The git history is preserved in archive form; the implementing work may also choose to import the upstream history into the kernel repo at the embedded path so `git log` over ritual edits stays continuous (decision deferred to the implementing gap).
- G-0182's testdata consolidation gets simpler: collapsing (3) onto (2) is now collapsing onto THE authoring source rather than onto a vendored mirror.

The decision concerns the *authoring channel only*. Distribution (embed-and-materialize) is unaffected — `aiwf init` / `aiwf update` continue to materialize from the embedded snapshot exactly as today.

This ADR is a follow-up to ADR-0014 (which retired the distribution channel) and completes the same simplification arc.

## Consequences

**Positive.**

- One authoring surface. A ritual edit is one commit in one repo.
- One drift-class shrunk. `TestRituals_VendoredMatchesUpstream` and the upstream-vs-vendored drift it polices both retire.
- Closer alignment with the kernel principle "no plugin architectures for a single implementation."
- G-0182's resolution becomes purely a test-repointing exercise (point AC tests at embedded bytes) rather than a triangulation between three authoring surfaces.
- Future ritual fixes (e.g. the deferred wrap-milestone trailer-stamp landing from G-0190) become trivially one-commit changes.
- AI-discoverability improves: the LLM and human reader both find ritual content in one place.

**Negative.**

- Hypothetical third-party consumers who want to fork the ritual content lose a clean external pinning point. (No such consumers exist today; if one shows up, forking the embedded directory works the same as forking a standalone repo.)
- Upstream git history for ritual edits is no longer at `git log` of the kernel repo unless the implementing work imports it. Most consumers won't care; ritual-history-spelunkers either consult the archived GH repo or get the history imported into the kernel repo at the embedded path (implementer's call).
- One-time work to retire the upstream surface — script removal, drift-test removal, GH repo archive, optional history import. The implementing gap names the punch list.

**Relationship to other decisions.**

- ADR-0014 retired the *distribution* channel (marketplace plugin); this ADR retires the *authoring* channel (upstream-as-source). Together they collapse the entire ritual-content pipeline onto one source.
- G-0182 (testdata consolidation) becomes substantially simpler. The two resolutions can land in sequence: ADR-0016 implementing work first, then G-0182's repointing.
- G-0190 (already landed in G-0192's wf-patch as commit `ab75d376`) intentionally deferred the wrap-milestone trailer-stamp fix to the post-ADR-0016 window so it could land cleanly as one embedded-snapshot commit.

**Status: proposed.** Ratification waits on the implementing gap producing a credible punch list and the operator confirming the GH-repo-archive step is acceptable.
