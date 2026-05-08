---
id: M-077
title: aiwf retitle verb for entities and ACs (closes G-065)
status: in_progress
parent: E-22
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
      status: open
      tdd_phase: red
    - id: AC-4
      title: aiwf-rename skill body redirects to retitle
      status: open
      tdd_phase: red
    - id: AC-5
      title: Closed-set completion for retitle id argument
      status: met
      tdd_phase: done
    - id: AC-6
      title: Verb-level integration test drives the dispatcher
      status: open
      tdd_phase: red
---

# M-077 — aiwf retitle verb for entities and ACs (closes G-065)

## Goal

Ship `aiwf retitle <id|composite-id> <new-title> [--reason ...]` so scope refactors can correct entity titles without leaving frontmatter `title:` permanently misleading. Title only — no body changes, no slug renames (those go through `aiwf rename`). Composite-id support covers AC titles inside parent milestone frontmatter. Closes G-065 — the asymmetry where `aiwf rename` exists for slugs but no verb exists for titles.

## Context

Scope refactors that change an entity's intent leave the frontmatter `title:` stale. The slug can be corrected via `aiwf rename`, but the title — which is what humans read in `aiwf status`, `aiwf show`, `aiwf history`, and the roadmap — stays misleading. Today the operator either lives with the drift or hand-edits frontmatter (which `aiwf edit-body` refuses, same chokepoint as G-072). G-065's body explicitly cites *"entity's or AC's intent,"* so this milestone covers both top-level entities and AC sub-elements.

## Acceptance criteria

### AC-1 — aiwf retitle works for all top-level kinds with --reason

### AC-2 — Composite-id retitle updates frontmatter and body atomically

### AC-3 — aiwf-retitle skill exists with title-shaped phrasings

### AC-4 — aiwf-rename skill body redirects to retitle

### AC-5 — Closed-set completion for retitle id argument

### AC-6 — Verb-level integration test drives the dispatcher

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

- No prior milestones in E-22.
- Existing patterns: `aiwf rename`'s frontmatter-edit flow; composite-id parser used by `aiwf show`/`history`/`promote`; `aiwf edit-body`'s `--reason` handling (the trailer-shape part).

## Coverage notes

- (filled at wrap)

## References

- E-22 epic spec (parent).
- G-065 — names the missing verb; cites *"entity's or AC's intent."*
- `aiwf rename` — the existing slug-rename verb; precedent for the frontmatter-edit flow but operates on a different field.
- `aiwf edit-body` — precedent for the `--reason` flag and trailer pattern.
- `aiwf promote` / `aiwf cancel` — additional precedent for `--reason`.
- `internal/check/entity_body.go` — the `h3ACHeading` regex for locating AC body headings.

---

## Work log

(filled during implementation)

## Decisions made during implementation

- (none — all decisions pre-locked above)

## Validation

(pasted at wrap)

## Deferrals

- (none)

## Reviewer notes

- (filled at wrap)
