---
id: G-0192
title: 'Post-E-0038 cleanup: doc/test/skill loose ends from embed-and-materialize'
status: addressed
addressed_by_commit:
    - 6a1e70cc
---
## What's missing

E-0038 landed embed-and-materialize per [ADR-0014](../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) (rituals embedded into the binary, marketplace channel retired). The wrap missed several follow-on cleanup items that span docs, tests, kernel-allowlist drift, and ritual-skill prose. Each was filed separately as its surface surfaced; the items have a common cause (the E-0038 wrap's coverage gap) and similar shape (small mechanical edits), and they should land as one focused wf-patch rather than scattered piecemeal work.

### Joined items (already filed)

- **[G-0182](./G-0182-consolidate-testdata-ritual-fixtures-onto-the-embedded-snapshot-dedupe.md)** — Consolidate `internal/policies/testdata/<skill>/SKILL.md` fixtures onto the embedded snapshot. Repoint per-AC content-assertion tests at the embedded bytes (`skills.ListRituals()` / `ListRitualAgents()` / `ListRitualTemplates()` already expose them) and delete the duplicated fixtures. Collapses the drift class to the single `TestRituals_VendoredMatchesUpstream` guard.
- **[G-0190](./G-0190-ritualverbs-allowlist-should-derive-from-embedded-snapshot.md)** — `internal/check/trailer_verb_unknown.go::ritualVerbs` is a hand-maintained allowlist; should derive from the embedded snapshot (a drift test akin to `TestRituals_VendoredMatchesUpstream` extracting `aiwf-verb:` values from embedded skill markdown).
- **[G-0191](./G-0191-wf-patch-and-siblings-hardcode-pr-based-flow-project-flow-belongs-in-claude-md.md)** — `wf-patch`, `wf-review-code`, `wf-doc-lint` skills hardcode PR-based merge flow. Should be project-flow-agnostic; `CLAUDE.md` names the merge mechanism.

### New items (surfaced by the 2026-05-31 doc-lint pass)

**Removed-feature docs (marketplace channel retired by ADR-0014):**

1. **`docs/pocv3/plans/rituals-plugin-plan.md`** is unarchived despite ADR-0014 §"Consequences" explicitly saying *"`rituals-plugin-plan.md`'s marketplace design is superseded and is updated/archived by the implementing epic."* The whole document describes the now-retired marketplace channel. Move to `docs/pocv3/archive/`.
2. **`README.md:142`** — *"aiwf adoption is two steps: install the binary and install the companion rituals plugin."* Directly contradicts the embed-and-materialize model described correctly at `README.md:170+` and `README.md:183`. Likely a missed line during the E-0038 README rewrite.
3. **`docs/adr/ADR-0007-planning-skills-rituals-plugin-pure-skill-default.md`** (status: `accepted`) describes the rituals plugin as marketplace-distributed. ADR-0014 §"Relationship to ADR-0007" explicitly revises the *"delivery channel assumption"* while preserving placement/authoring. ADR-0007 needs a top-of-file note pointing at ADR-0014's partial revision so a reader landing on ADR-0007 first sees stale claims as ratified.

**Stale terminology (lower priority — "rituals plugin" used in pre-ADR-0014 consumer-channel sense):**

4. `CLAUDE.md:41` — *"the rituals plugin's planning skills"* → *"the materialized planning skills"* / *"the planning rituals."*
5. `docs/adr/ADR-0008-canonicalize-kernel-ids-to-4-digits.md:99` — *"the rituals plugin's embedded skills"* — same shape.
6. `docs/adr/ADR-0011-legal-workflow-spec-methodology.md:19` — *"skills in `.claude/skills/` and the rituals plugin"* — same shape.
7. `docs/pocv3/skill-author-guide.md:231` — *"either aiwf or the rituals plugin"* → *"either aiwf core or the embedded rituals snapshot."*

## Why it matters

Two reasons to bundle, not address piecemeal:

1. **One coherent cause.** All seven items are E-0038-wrap-coverage holes. A reader hitting `README.md:142` and then `README.md:170+` sees the project contradict itself within thirty lines — that's the kind of doc-drift that materially damages new-adopter trust. The fix is mechanical; the value of doing it together is that the consumer never sees a half-corrected state.
2. **Patch-shape efficiency.** Each item is genuinely small (a doc edit, a file move, a couple of skill-body prose changes). Filing them separately and wf-patching them separately is more ceremony than the work warrants. The umbrella gap is the audit-trail anchor; one wf-patch is the implementation; the existing gaps get addressed by reference.

The kernel principle *"framework correctness must not depend on LLM behavior"* applies one level out here too: an adopter reading the docs to learn the framework should not have to triangulate between contradictory paragraphs. Pin the docs to the post-E-0038 reality once, mechanically.

## Resolution shape

**One wf-patch** on a branch like `patch/g-0192-e0038-cleanup`. The patch lands:

### Doc edits (items 1–7 above)

- Move `docs/pocv3/plans/rituals-plugin-plan.md` → `docs/pocv3/archive/rituals-plugin-plan.md`. Add a one-line note at the top: *"Superseded by ADR-0014 (2026-05-XX): marketplace channel retired; rituals now embed-and-materialize. See [E-0038](...) and CLAUDE.md §"Operator setup" for the current model."*
- Edit `README.md:142` to drop the *"install the companion rituals plugin"* second-step and align with the embed-and-materialize language used at `README.md:170+`.
- Add a top-of-file header note to `docs/adr/ADR-0007-...md` pointing at ADR-0014's delivery-channel revision: *"> See also: ADR-0014 — Embed-and-materialize rituals distribution retires this ADR's marketplace-channel assumption while preserving the placement/authoring layering described below."*
- Update the terminology in `CLAUDE.md:41`, `ADR-0008:99`, `ADR-0011:19`, `skill-author-guide.md:231` per the per-line rewrites named above.

### Code edits (G-0182, G-0190)

- **G-0182** — Add a helper that exposes embedded skill bytes to per-AC content-assertion tests (`skills.ListRituals()` exposes the bytes; per-AC tests need a thin section-scoped wrapper). Repoint each per-AC content assertion at the embedded snapshot. Delete `internal/policies/testdata/<skill>/SKILL.md` for the migrated skills. Preserve section-scoping per CLAUDE.md §"Substring assertions are not structural assertions"; keep AC6's structural merge-step check and other non-content assertions.
- **G-0190** — Add a drift test `TestRitualVerbs_DerivedFromEmbedded` (or extend an existing test) that extracts `aiwf-verb:` values from embedded skill markdown and asserts they match `internal/check/trailer_verb_unknown.go::ritualVerbs`. Or, more thoroughly, replace the hand-maintained map with a function that derives the set at run time from the embedded bytes (the cleaner shape — eliminates the allowlist as a hand-maintained surface).

### Skill prose edits (G-0191)

- Edit `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md` per G-0191's specific edits (Merge gate replaces PR gate; anti-pattern rewritten; description retitled).
- Same shape for `wf-review-code`, `wf-doc-lint` per G-0191's specific edits.
- Paired testdata fixture edits at `internal/policies/testdata/<skill>/SKILL.md` — *unless G-0182's consolidation lands earlier in the same patch, in which case the testdata side is already gone.* The patch can be ordered to land G-0182 first (deletes the testdata duplication) then the prose edits land in the embedded snapshot only — cleaner.
- Refresh `rituals.lock` via `make sync-rituals` after the matching upstream commits in `23min/ai-workflow-rituals` land (these are pure-prose, so they can be authored together).

### Validation

- `aiwf check` clean.
- `go test ./...` passes.
- `golangci-lint run ./...` clean.
- New drift test from G-0190 passes against the current embedded snapshot.
- `TestRituals_VendoredMatchesUpstream` passes (proves embedded vs upstream alignment).
- `README.md` and `CLAUDE.md` read coherently — no remaining contradiction between "install the companion rituals plugin" and "embed-and-materialize."

## References

- [ADR-0014](../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) — the embed-and-materialize landing whose wrap missed these items.
- **E-0038** — the implementing epic, now `done`/archived.
- [G-0182](./G-0182-consolidate-testdata-ritual-fixtures-onto-the-embedded-snapshot-dedupe.md), [G-0190](./G-0190-ritualverbs-allowlist-should-derive-from-embedded-snapshot.md), [G-0191](./G-0191-wf-patch-and-siblings-hardcode-pr-based-flow-project-flow-belongs-in-claude-md.md) — the joined gaps this umbrella absorbs as its sub-items.
- [ADR-0007](../../docs/adr/ADR-0007-planning-skills-rituals-plugin-pure-skill-default.md) — partially revised by ADR-0014; needs the supersession header note.
- `CLAUDE.md` §"Operator setup" and §"Working in this repo" — the source-of-truth descriptions of the embed-and-materialize model.

Surfaced by the 2026-05-31 wf-doc-lint pass during the G-0129 wf-patch session.
