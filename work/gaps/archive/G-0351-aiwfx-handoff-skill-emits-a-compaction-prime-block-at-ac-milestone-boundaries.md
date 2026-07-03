---
id: G-0351
title: aiwfx-handoff skill emits a compaction prime block at AC/milestone boundaries
status: addressed
addressed_by_commit:
    - e6f60d81
---
## Problem

Priming a `/compact` is a manual copy/paste ritual. When context grows, the operator
triggers `/compact` and hand-writes a steering parameter — a short "here's where we
are, here's what's next" handoff so the post-compaction session hits the ground
running. Composing that block from scratch, every time, is friction, and it happens at
an unpredictable moment (whenever tokens get high) rather than at a natural boundary.

Two observations reshape the fix:

1. **The natural moment is a boundary.** An AC or milestone just closed → state was
   just committed, the mental model is fresh, and it is a natural pause. That is exactly
   when a handoff is cheapest and most accurate to write.
2. **Most of "where are we" is already durable.** `aiwf status`, the milestone spec,
   `aiwf show <M>`, and `aiwf history <M>` already reconstruct the committed state. A
   handoff that re-summarizes that is a second source of truth that rots (violates the
   single-source-of-truth force) and is what makes these prompts balloon. The handoff's
   *only* job is to carry the **volatile** half git/aiwf cannot reconstruct — the live
   reasoning thread, gotchas found this session, the exact next action — and to *point
   into* the tree for the rest.

`/compact` itself is a Claude Code harness command whose argument is human-typed; aiwf
cannot hook it or inject the argument. What aiwf *can* remove is the compose step.

## Proposed capability — an `aiwfx-handoff` ritual skill

A new ritual skill, `aiwfx-handoff`, under the `aiwf-extensions` plugin (sibling to
`aiwfx-whiteboard`, its nearest shape: read-only synthesis over the tree). It emits a
short (≤10-line) paste-ready "compaction prime" block. Invoked two ways, both via its
`description` frontmatter:

- **On request, anywhere** — "give me a handoff", "prime the compact", "where are we for
  /compact" match the description, so the skill fires mid-conversation, not only at a
  boundary. This is the on-demand affordance; it is not a special mode, just the skill's
  normal trigger surface.
- **At a boundary** — the two boundary rituals *reference* it (see Insertion points)
  rather than inlining the block format, keeping one source of truth for the format.

The operator still copies the emitted block into `/compact <paste>` — the single paste
is irreducible (the skill cannot call the harness command). The skill removes the
*compose* burden and makes priming invokable on demand.

## Block format (draft)

Volatile-first, points into the tree, six lines:

```
Continue E-NNNN / M-NNNN — <milestone one-liner>.
Just landed: <AC-N / what's done>.  Next: <AC-N — the exact next action>.
State lives in the tree: `aiwf show M-NNNN` · `aiwf status` · `aiwf history M-NNNN`.
Branch M-NNNN/<slug>: <clean | WIP: what's uncommitted>.
Watch out: <gotcha found this session — approach X rejected because Y; don't re-open G-NNNN>.
Decisions this session: <ADR-NNNN / D-NNNN, or "none">.
```

Lines 5–6 are the payload — the part `aiwf status` cannot reconstruct. Lines 1–4 are
pointers, not re-summaries. If a draft runs past ~10 lines, that is the tell it is
duplicating committed state and should be replaced with an `aiwf show` pointer. The cap
is a symptom of doing it right, not an arbitrary limit.

## Insertion points

- **`aiwfx-start-milestone` step 6** (the per-AC loop, after the Work-log-append
  bullet): add a bullet — at the AC boundary, if the user asks for a handoff or context
  is getting long before the next AC, invoke `aiwfx-handoff`.
- **`aiwfx-wrap-milestone` "Next step"** (end of the skill, before the
  `→ aiwfx-start-milestone <next-M>` pointer): add — emit `aiwfx-handoff`; the milestone
  close is the natural compact point.
- **`aiwfx-handoff` itself** carries the block format + the volatile/durable rule, so
  both references stay one-liners.

Note: the AC *loop* boundary lives in `aiwfx-start-milestone` step 6, **not** in
`wf-tdd-cycle` (which is a single inner red/green/refactor iteration that returns to the
loop). `wf-tdd-cycle` needs no change.

## Cadence

- **Milestone boundary: always emit** — infrequent, a natural compact point.
- **AC boundary: on request, or when context is visibly long** — every-AC emission is
  noise, since compaction is not per-AC.

If a configurable cadence is ever wanted, an `aiwf.yaml` knob is the home — YAGNI until
friction shows.

## Deliberate non-goals

- **No `HANDOFF.md` cache.** Unlike `aiwfx-whiteboard` (which caches a large landscape
  you revisit), a handoff is ephemeral and consumed immediately into the `/compact`
  argument. A file would add a stale surface for no gain. Emit inline only.
- **No new kernel verb (for now).** The valuable half of a handoff is LLM judgment (the
  volatile reasoning), which cannot be mechanized — so it belongs in the advisory ritual
  layer, not the kernel. If handoffs later drift in quality, the escape hatch is a
  mechanical skeleton — e.g. `aiwf status --format=handoff` emitting the durable half
  (current epic/milestone, next AC, recent history, open findings) — with the ritual
  filling the volatile half. Split-not-move, matching the existing hybrid-guidance
  pattern. Do not build this speculatively.
- **`/compact` stays human-driven.** No hook injects a computed argument; the paste is
  irreducible. The win is deleting the compose step, not the paste.

## Build requirements

- **Author location:**
  `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-handoff/SKILL.md`
  (embedded via `go:embed`, materialized into consumers' `.claude/` by `aiwf init` /
  `aiwf update`).
- **Referencing structural test** under `internal/policies/` is mandatory: the
  `skill-edit-structural-test-backstop` policy fails CI for an embedded-rituals
  `SKILL.md` that no policy test references, and `skill-coverage` requires valid
  `name:`/`description:` frontmatter. v1 granularity is file-existence + skill-reference;
  prefer a stronger structural assertion (block-format section present; both boundary
  references present in the two rituals).
- **`skill-body-id` constraint:** the shipped skill body cites no real entity id, path,
  or inline status — illustrative ids use canonical `<prefix>-NNNN` placeholders (as in
  the block format above) or shape-descriptions. The check fires pre-push over
  `internal/skills/embedded{,-rituals}/**`.
- **Two ritual edits** (`aiwfx-start-milestone`, `aiwfx-wrap-milestone`) each need a
  referencing structural test touch under `internal/policies/` per the same backstop.
- **Sizing:** small and self-contained — a `wf-patch` fits if done as one skill + two
  one-line references + tests; a single milestone fits if the drafter wants the
  structural tests treated as ACs.

## Suggested acceptance criteria for the implementing milestone

1. `aiwfx-handoff` `SKILL.md` exists at the author location with valid
   `name:`/`description:` frontmatter; the `description` includes the on-request trigger
   phrases; `skill-coverage` passes.
2. The skill body contains the block-format section (≤10-line volatile-first template)
   and the volatile-vs-durable rule; asserted by a structural test under
   `internal/policies/`.
3. `aiwfx-start-milestone` step 6 references `aiwfx-handoff` at the AC boundary;
   `aiwfx-wrap-milestone` "Next step" references it at the milestone close — both
   asserted structurally.
4. `skill-body-id` and the full `aiwf check` are clean over the new and edited skills.
