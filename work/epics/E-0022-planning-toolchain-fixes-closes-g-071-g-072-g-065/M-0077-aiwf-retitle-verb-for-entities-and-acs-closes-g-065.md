---
id: M-0077
title: aiwf retitle verb for entities and ACs (closes G-0065)
status: done
parent: E-0022
tdd: required
acs:
    - id: AC-1
      title: aiwf retitle works for all top-level kinds with --reason
      status: met
      tdd_phase: done
    - id: AC-2
      title: Composite-id retitle updates frontmatter and body atomically
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf-retitle skill exists with title-shaped phrasings
      status: met
      tdd_phase: done
    - id: AC-4
      title: aiwf-rename skill body redirects to retitle
      status: met
      tdd_phase: done
    - id: AC-5
      title: Closed-set completion for retitle id argument
      status: met
      tdd_phase: done
    - id: AC-6
      title: Verb-level integration test drives the dispatcher
      status: met
      tdd_phase: done
---

# M-0077 — aiwf retitle verb for entities and ACs (closes G-0065)

## Goal

Ship `aiwf retitle <id|composite-id> <new-title> [--reason ...]` so scope refactors can correct entity titles without leaving frontmatter `title:` permanently misleading. Title only — no body changes, no slug renames (those go through `aiwf rename`). Composite-id support covers AC titles inside parent milestone frontmatter. Closes G-0065 — the asymmetry where `aiwf rename` exists for slugs but no verb exists for titles.

## Context

Scope refactors that change an entity's intent leave the frontmatter `title:` stale. The slug can be corrected via `aiwf rename`, but the title — which is what humans read in `aiwf status`, `aiwf show`, `aiwf history`, and the roadmap — stays misleading. Today the operator either lives with the drift or hand-edits frontmatter (which `aiwf edit-body` refuses, same chokepoint as G-0072). G-0065's body explicitly cites *"entity's or AC's intent,"* so this milestone covers both top-level entities and AC sub-elements.

## Acceptance criteria

### AC-1 — aiwf retitle works for all top-level kinds with --reason

`aiwf retitle <id> "<new-title>" [--reason "..."]` updates the frontmatter `title:` field for any of the six top-level kinds (epic, milestone, ADR, gap, decision, contract). Title-only mutation: no body changes, no slug renames, no other frontmatter fields touched. Empty new title (after trimming whitespace) is rejected with a usage error. Same-as-current title is rejected — there's no diff to commit. Unknown entity id is rejected. The optional `--reason` flag lands in the commit body and surfaces in `aiwf history`. Verb implementation lives in `internal/verb/retitle.go`; cmd in `cmd/aiwf/retitle_cmd.go`.

### AC-2 — Composite-id retitle updates frontmatter and body atomically

`aiwf retitle M-NNN/AC-N "<new-title>"` updates the AC's `title:` inside the parent milestone's `acs[]` array AND regenerates the matching `### AC-N — <new-title>` body heading. Both happen in one atomic file write, so the commit captures both changes. Reuses `lookupAC`, `withACMutation`, and `rewriteACHeading` from `internal/verb/ac.go` (the same helpers `aiwf rename M-NNN/AC-N` consumes for its composite-id arm) — no new parser, no new heading-rewriter. The trailer is `aiwf-verb: retitle` so `aiwf history M-NNN/AC-N` distinguishes retitle invocations from rename's composite-id arm.

### AC-3 — aiwf-retitle skill exists with title-shaped phrasings

New `internal/skills/embedded/aiwf-retitle/SKILL.md` with frontmatter description densely populated with title-shaped phrasings: *"the title doesn't match anymore"*, *"fix the title"*, *"retitle to reflect new scope"*, *"correct the title"*, *"rename the title"*. The last phrasing intentionally overlaps with `aiwf-rename` to cover the natural confusion; the body redirects to the right verb in either skill. The `internal/skills/skills_test.go` `TestList_AllShippedSkillsPresent` table is extended to expect 12 skills including `aiwf-retitle`.

### AC-4 — aiwf-rename skill body redirects to retitle

`internal/skills/embedded/aiwf-rename/SKILL.md` gains a short blockquote near the top: *"Looking to change a title? For changing an entity's title (the prose label, distinct from the slug), use `aiwf retitle <id> <new-title>` — that is the dedicated verb for title mutations. This skill covers slug renames only."* The redirect ensures AI assistants land on the right verb regardless of which skill they invoke first.

### AC-5 — Closed-set completion for retitle id argument

The positional `<id>` arg on `aiwf retitle` registers `completeEntityIDArg("", 0)` so shell completion proposes any entity id (all six top-level kinds, plus composite ids since the empty filter accepts both). New flags (`--actor`, `--principal`, `--root`, `--reason`) are covered by the existing completion-drift opt-out table; the test passes without modification.

### AC-6 — Verb-level integration test drives the dispatcher

Per CLAUDE.md "Test the seam, not just the layer": `TestRetitle_DispatcherSeam_TopLevel` and `TestRetitle_DispatcherSeam_Composite` drive `run([]string{"retitle", ...})` end-to-end through cmd → verb → projection → apply → git, then assert (a) on-disk frontmatter title changed AND (b) `aiwf history <id>` finds the trailered retitle commit (proving the trailer chain reached git). A regression where the cmd flag is read but never copied into the verb call slips past unit tests but trips these.

## Constraints

- **Title only.** No body changes, no slug renames, no frontmatter mutations beyond `title:`. The single-mutation rule keeps the verb's reasoning local.
- **`--reason` is supported** per the epic's locked decision. Lands in commit body and surfaces in `aiwf history` per the existing pattern from `aiwf promote`/`cancel`/`authorize`/`edit-body`.
- **Composite-id support** for AC titles. `aiwf retitle M-NNN/AC-N "<new-title>"` updates the AC's `title:` inside the parent milestone's `acs:` array AND regenerates the corresponding `### AC-N — <new-title>` body heading. Both happen in one atomic commit per kernel rule.
- **Reuse the existing composite-id parsing** from `aiwf show <M-NNN/AC-N>`, `aiwf history <M-NNN/AC-N>`, `aiwf promote <M-NNN/AC-N>`. No new parser; consume the existing helper.
- **All top-level kinds supported.** `E-NN`, `M-NNN`, `ADR-NNNN`, `G-NNN`, `D-NNN`, `C-NNN`. The dispatcher resolves the id to its file path and updates the frontmatter `title:`.
- **One commit per invocation** per kernel rule. Trailers: `aiwf-verb: retitle`, `aiwf-entity: <id>` (or `<id>/AC-N` for composite ids), `aiwf-actor: <derived>`.
- **TDD-required.** Each AC drives a red→green→refactor cycle. AC-6 is the seam test driving `run([]string{"retitle", ...})`.
- **`aiwf rename` is unaffected.** The two verbs stay separate; neither invokes the other. `aiwf rename` keeps slug-only semantics.

## Design notes

- **Verb signature:** `aiwf retitle <id|composite-id> <new-title> [--reason "..."]`. Two positional arguments (id, new-title) plus the optional flag. Matches `aiwf rename <id> <new-slug>`'s positional shape.
- **Skill placement:** new `internal/skills/embedded/aiwf-retitle/SKILL.md`, not addition to `aiwf-rename`. Per the epic's locked decision (Q1). Description densely populated with title-shaped phrasings: *"the title doesn't match anymore"*, *"fix the title"*, *"retitle to reflect new scope"*, *"correct the title"*, *"rename the title"*. The last phrasing intentionally overlaps with `aiwf-rename` to cover the natural confusion — the body redirects to the right verb in either skill.
- **`aiwf-rename` skill body update** for AC-4: a short section near the top — *"For changing an entity's title (the prose label, distinct from the slug), use `aiwf retitle <id> <new-title>` — that is the dedicated verb for title mutations. This skill covers slug renames only."*
- **Composite-id handling for AC-2:** the verb resolves `M-NNN/AC-N` to the parent milestone file. Two coordinated edits inside that file:
  1. Frontmatter: update `acs[i].title` for the matching AC.
  2. Body: replace the `### AC-N — <old-title>` heading with `### AC-N — <new-title>`. Use the existing `h3ACHeading` regex from `internal/check/entity_body.go` to locate; replace by line.
  Both happen via a single file write; the commit captures both in one diff.
- **Top-level kind dispatch for AC-1:** the verb loads the entity by id, edits its frontmatter `title:` field, writes the file, commits. Reuses the loader and writer machinery from `aiwf rename` (which does the same shape — load, edit frontmatter, write, commit) but on a different field.
- **Validation:** the new title must be non-empty after trimming whitespace. Reject empty new-titles with a clear error. No length cap or character validation beyond the existing tree-shape rules.
- **No old-title preservation.** The frontmatter `title:` is overwritten. Old titles live in `git log`'s diffs and `aiwf history` shows the retitle event with the prior commit's title visible from the diff. A "show me previous titles" feature is deferred (out of scope per epic spec).

## Surfaces touched

- `cmd/aiwf/retitle_cmd.go` (new) — the verb implementation.
- `cmd/aiwf/completion_drift_test.go` — completion wiring assertion.
- `internal/skills/embedded/aiwf-retitle/SKILL.md` (new) — focused skill per locked decision.
- `internal/skills/embedded/aiwf-rename/SKILL.md` — body addition with the redirect.
- Existing composite-id parser (location TBD; reused without modification).

## Out of scope

- Combined "rename + retitle" verb.
- Title history rendering (old titles in some persistent form beyond git history).
- Any mutation other than frontmatter `title:` and (for composite-id) the AC heading line.
- Any change to `aiwf rename`'s slug-only semantics.

## Dependencies

- No prior milestones in E-0022.
- Existing patterns: `aiwf rename`'s frontmatter-edit flow; composite-id parser used by `aiwf show`/`history`/`promote`; `aiwf edit-body`'s `--reason` handling (the trailer-shape part).

## Coverage notes

- `internal/verb/retitle.go:Retitle` — 87% statement. Branch audit: composite-id false (top-level path) and true (composite path) both covered; empty-title arm via `TestRetitle_EmptyTitleRejected`; unknown-id via `_UnknownIdRejected`; same-title via `_SameTitleRejected`; success arm via every kind in `TestRetitle_AllKinds`. Uncovered: `readBody` IO failure, `entity.Serialize` failure, `projectionFindings` error arm — defensive paths consistent with the project's pattern (M-0075 `entityBodyEmpty`, M-0076 `MilestoneDependsOn` follow the same convention).
- `internal/verb/retitle.go:retitleAC` — 76% statement. Branch audit: `lookupAC` error covered via `_AC_UnknownRejected`; same-title arm reachable via the top-level same-title test path's logic; success arm via `_AC_FrontmatterAndBody`, `_DispatcherSeam_Composite`. Uncovered: same defensive IO arms (readBody/Serialize/projection).
- `cmd/aiwf/retitle_cmd.go:newRetitleCmd` — 100%.
- `cmd/aiwf/retitle_cmd.go:runRetitleCmd` — 65%. The uncovered region is the lock/loadTree/actor-resolve error arms which require filesystem failure or concurrent-locked state to exercise — same convention as `runRenameCmd`, `runEditBodyCmd`, etc.
- `internal/skills` — `TestList_AllShippedSkillsPresent` table extended to 12 entries; the new `aiwf-retitle` skill is asserted present alongside the existing 11.

## References

- E-0022 epic spec (parent).
- G-0065 — names the missing verb; cites *"entity's or AC's intent."*
- `aiwf rename` — the existing slug-rename verb; precedent for the frontmatter-edit flow but operates on a different field.
- `aiwf edit-body` — precedent for the `--reason` flag and trailer pattern.
- `aiwf promote` / `aiwf cancel` — additional precedent for `--reason`.
- `internal/check/entity_body.go` — the `h3ACHeading` regex for locating AC body headings.

---

## Work log

### AC-1 — retitle works for all top-level kinds

Added `Retitle` to `internal/verb/retitle.go` and `aiwf retitle` cmd to `cmd/aiwf/retitle_cmd.go`. Top-level path: load entity → mutate `title:` field → re-serialize with existing body → project + check → emit `aiwf-verb: retitle` plan. Reuses the same load/serialize machinery as `aiwf rename` but on a different field. Tests in `cmd/aiwf/retitle_cmd_test.go`: `TestRetitle_AllKinds` (6 sub-cases, one per kind), `TestRetitle_Reason`, `TestRetitle_EmptyTitleRejected`, `TestRetitle_SameTitleRejected`, `TestRetitle_UnknownIdRejected`.

### AC-2 — Composite-id retitle (frontmatter + body heading)

Added `retitleAC` helper in the same file. Reuses `lookupAC`, `withACMutation`, and `rewriteACHeading` from `internal/verb/ac.go` — no new parser, no new heading-rewriter. The shape parallels `renameAC`'s composite-id arm but emits a `retitle` trailer so `aiwf history M-NNN/AC-N` distinguishes the two invocation paths. Test: `TestRetitle_AC_FrontmatterAndBody` asserts both the `acs[].title` mutation AND the body heading regeneration in the same atomic commit; `TestRetitle_AC_UnknownRejected` covers the missing-AC arm.

### AC-3 — aiwf-retitle skill

New `internal/skills/embedded/aiwf-retitle/SKILL.md` with title-shaped phrasings densely populated in the frontmatter description: *"the title doesn't match anymore"*, *"fix the title"*, *"retitle to reflect new scope"*, *"correct the title"*, *"rename the title"*. The last phrasing intentionally overlaps with `aiwf-rename`; the body redirects in either skill. `internal/skills/skills_test.go` updated to expect 12 skills.

### AC-4 — aiwf-rename skill body redirect

Inserted a short blockquote near the top of `internal/skills/embedded/aiwf-rename/SKILL.md` pointing operators to `aiwf retitle` for title changes. Symmetric with the redirect in `aiwf-retitle`'s body — AI assistants land on the right verb regardless of which skill they invoke first.

### AC-5 — Closed-set completion

The positional `<id>` arg uses `completeEntityIDArg("", 0)` which proposes any entity id (all six kinds plus composite ids). New flags (`--actor`, `--principal`, `--root`, `--reason`) are covered by the existing completion-drift opt-out table — `TestPolicy_FlagsHaveCompletion` and `TestPolicy_PositionalsHaveCompletion` pass without modification.

### AC-6 — Verb-level seam tests

`TestRetitle_DispatcherSeam_TopLevel` and `TestRetitle_DispatcherSeam_Composite` drive `run([]string{"retitle", ...})` end-to-end; assert (a) on-disk frontmatter title changed AND (b) `aiwf history <id>` finds the trailered commit. Per CLAUDE.md "Test the seam, not just the layer."

### Side cleanup — G-0079 body backfilled, README updated

While on the M-0077 branch I noticed two stale items left over from M-0076 wrap and an earlier session that needed cleanup before this branch could land:

1. `G-079` was filed during M-0076 wrap as the deferred-skill-update gap, but its body was never filled. Backfilled the prose via `aiwf edit-body G-079` so the gap reads cleanly when consumed by future planning.
2. `README.md`'s verb table missed `aiwf retitle` (this milestone) and `aiwf milestone depends-on` (M-0076). The skills-list line said "ten skill files" while the actual count is twelve (now including `aiwf-edit-body` and `aiwf-retitle`). Both updated atomically with this wrap so the README reflects what aiwf actually ships.

## Decisions made during implementation

- (none — all decisions pre-locked above)

## Validation

- `go test -race ./...` — green. All packages pass.
- `go build -o /tmp/aiwf ./cmd/aiwf` — green.
- `golangci-lint run ./cmd/aiwf/ ./internal/verb/ ./internal/skills/` — 0 issues (after `gofumpt -w` on `retitle_cmd_test.go`).
- `aiwf check` — 0 errors, 2 unrelated warnings (`provenance-untrailered-scope-undefined` because the milestone branch has no upstream; `unexpected-tree-file` on `work/epics/critical-path.md` is E-0021's scope).
- `aiwf show M-077` — every AC at `met` + `tdd_phase: done`.
- Real-tree dogfood: `aiwf retitle <id>` and `aiwf retitle M-NNN/AC-N` both round-trip through `aiwf history` with the right trailers; the seam tests pin this end-to-end.

## Deferrals

- (none)

## Reviewer notes

- **Why a separate `aiwf retitle` rather than extending `aiwf rename`.** Per the epic's locked decision (M-0077 spec design notes; epic spec §"Design decisions" Q1): title and slug are parallel mutations on different frontmatter fields, not topically related. `aiwf-rename` and `aiwf-retitle` are kept separate so the single-mutation rule keeps reasoning local. The skill-level confusion ("rename the title") is mitigated by the redirect in both skills' bodies — same pattern E-0020/M-0073 used for `aiwf-status` ↔ `aiwf-list`.
- **Composite-id arm shares helpers with `renameAC` deliberately.** The lookup, mutation, and body-heading-rewrite are identical; only the trailer differs. Re-implementing from scratch would invite drift. The shared helpers (`lookupAC`, `withACMutation`, `rewriteACHeading`) live in `internal/verb/ac.go` and are stable. If a future change touches AC body-heading mechanics, both `rename` and `retitle` benefit automatically.
- **`aiwf rename M-NNN/AC-N "<title>"` still works** — that's the existing path, untouched. Operators with muscle memory aren't broken by this milestone. Going forward, `aiwf retitle` is the dedicated verb; the skill redirects steer humans and AI assistants to it.
- **`Retitle`'s same-title rejection is a kindness, not a kernel-level invariant.** A no-op produces an empty diff, which the kernel would handle, but rejecting at the verb boundary means the operator notices the typo immediately rather than producing a no-op commit. Same convention as `renameAC`'s same-title check (`internal/verb/ac.go:283`).
- **Forward-compat note for G-0073's eventual cross-kind generalisation.** This verb is unaffected — `retitle` operates on `title:` which is universal across kinds. Cross-kind dependency mechanics (G-0073's territory) are orthogonal.
- **README updates and G-0079 body backfill rode along** rather than splitting into a separate `wf-patch`. Both were on-topic and minor: the README updates document M-0077's and M-0076's verbs (the spec calls for AI-discoverable surfaces, README counts), and G-0079's body was the receiving gap from M-0076's deferred skill-update — leaving it with empty `## What's missing` / `## Why it matters` would be dead weight. Reviewers may prefer the patch route in the future; in this case the bundled wrap was lower friction.
- **No ADR/D-NNN produced.** Every locked design choice (verb name, separate skill, `--reason` flag, composite-id support, title-only mutation) was pre-locked in the epic spec.
