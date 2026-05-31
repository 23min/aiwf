---
id: G-0193
title: 'Implement ADR-0016: retire upstream rituals authoring channel'
status: addressed
addressed_by_commit:
    - 1e529e5f
---
## What's missing

[ADR-0016](../../docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md) ratifies the architectural decision to retire the `ai-workflow-rituals` upstream as an authoring channel and make `internal/skills/embedded-rituals/` the canonical authoring location. The decision is the choice; this gap is the mechanical punch list to act on it.

The friction this resolves was felt directly during the G-0192 wf-patch session: adding a missing `aiwf-verb: wrap-milestone` trailer stamp to `aiwfx-wrap-milestone/SKILL.md` required a 2-repo coordination dance (edit upstream, push upstream, bump `rituals.lock`, `make sync-rituals`, commit kernel-side) for what should be a single 1-line ritual prose edit. [G-0190](./G-0190-ritualverbs-allowlist-should-derive-from-embedded-snapshot.md) intentionally deferred the wrap-milestone fix to land cleanly post-ADR-0016 as one embedded-snapshot commit.

## Why it matters

Same reasons as ADR-0016 §"Consequences" §positive: one authoring surface, one fewer drift class, kernel-principle alignment, simpler G-0182 resolution, simpler future ritual edits, better AI-discoverability.

This gap is the *mechanical* counterpart to ADR-0016's architectural choice. Until it lands, the authoring-channel friction stays real and the deferred wrap-milestone fix stays parked.

## Resolution shape

One wf-patch or a small milestone (the work is mechanical but spans several mechanical surfaces). The punch list:

### Kernel-side surface retirements

- Delete `rituals.lock` (and the comment scaffolding it carries).
- Delete `scripts/sync-rituals.sh`.
- Delete the `sync-rituals` Make target from `Makefile`.
- Delete `internal/policies/rituals_drift_test.go` (`TestRituals_VendoredMatchesUpstream` and any helpers exclusive to it). The drift class it polices ceases to exist when there is no upstream to drift from.
- Update `CLAUDE.md` §"Cross-repo plugin testing" — most of the section either retires or rewrites to describe the post-ADR-0016 single-source model. The G-0182 testdata-consolidation framing collapses to "testdata fixtures repoint at embedded bytes" with no third surface.

### Authoring-location surface

- Move the canonical authoring story to `CLAUDE.md` §"Operator setup" or a new short §"Authoring rituals" — *"Ritual content is authored directly under `internal/skills/embedded-rituals/plugins/<plugin>/skills/<skill>/SKILL.md`. Edit, test, commit on the kernel repo. No cross-repo coordination."*
- Any other doc surface that points operators at upstream as the authoring location gets retargeted (likely small — most CLAUDE.md references were already framed as "the snapshot is canonical for the binary").

### Optional: import upstream git history

- The upstream repo has a real git history of ritual edits. Two options:
  1. **Archive without import.** The upstream repo's history is preserved in archive form; `git log` over `internal/skills/embedded-rituals/` only sees kernel-side commits from this point forward. Simplest; existing kernel-side history of ritual edits stays unchanged.
  2. **Import history.** Use `git filter-repo` (or similar) to graft the upstream's `plugins/` history under the kernel's `internal/skills/embedded-rituals/plugins/` path, then merge. Preserves `git log` continuity for ritual edits.

  Recommend option 1 (archive without import). The upstream repo stays browsable for history-spelunking; the kernel repo's history starts fresh at the retirement point. Option 2 is doable but invasive (history rewrite, force-push concerns); the value-vs-cost trade-off favors archive. Confirm with operator.

### Upstream GitHub repo archive

- Add a top-level `README.md` (or rewrite the existing one) pointing readers at `https://github.com/23min/aiwf` `internal/skills/embedded-rituals/` as the new canonical location, with a one-line note that the repo is archived as of `<date>` per ADR-0016.
- Use GitHub's "Archive this repository" setting to make the repo read-only. (Sovereign action — requires operator to do this in GH UI; not scriptable from the kernel side.)

### Validation

- `go test ./...` passes after the test deletions.
- `make sync-rituals` is no longer a valid target (and `make help` doesn't list it).
- `aiwf check` clean.
- `golangci-lint run` clean.
- `aiwf init` / `aiwf update` still materialize rituals correctly from the embedded snapshot (no functional change to distribution).
- A test pass against the embedded snapshot's authoring story: any per-AC content-assertion test that previously pointed at testdata fixtures (G-0182 scope) now points at embedded bytes directly; the test surface is one cleaner step closer to single-source.

### Ordering

This gap's implementing work and [G-0182](./G-0182-consolidate-testdata-ritual-fixtures-onto-the-embedded-snapshot-dedupe.md)'s testdata consolidation are coupled but separable:

- G-0193 (this gap) lands first → upstream channel retires, embedded becomes canonical.
- G-0182 lands next → testdata fixtures repoint at embedded bytes, eliminating the third authoring surface that ADR-0016 doesn't explicitly retire (G-0182's surface is intra-kernel).

The wrap-milestone trailer-stamp fix deferred from G-0190 lands as a small post-G-0193 follow-up: one commit on `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md` plus a one-line drift-test expected-set update at `internal/skills/ritual_trailer_verbs_test.go`. No ceremony.

## References

- [ADR-0016](../../docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md) — the ratifying decision this gap implements.
- [ADR-0014](../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) — the predecessor that retired the distribution channel.
- [G-0190](./G-0190-ritualverbs-allowlist-should-derive-from-embedded-snapshot.md) — the gap whose wf-patch session surfaced the cross-repo friction; deferred the wrap-milestone trailer-stamp fix to post-G-0193.
- [G-0182](./G-0182-consolidate-testdata-ritual-fixtures-onto-the-embedded-snapshot-dedupe.md) — the coupled gap that lands next.
- [G-0192](./G-0192-post-e-0038-cleanup-doc-test-skill-loose-ends-from-embed-and-materialize.md) — the umbrella wf-patch gap whose session surfaced this work.
- `CLAUDE.md` §"Cross-repo plugin testing" — the section that retires or rewrites at this gap's landing.

Surfaced by the G-0192 wf-patch session (2026-05-31).
