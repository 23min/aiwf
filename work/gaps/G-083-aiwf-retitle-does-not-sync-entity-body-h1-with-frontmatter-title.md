---
id: G-083
title: aiwf retitle does not sync entity body H1 with frontmatter title
status: open
discovered_in: E-21
---

# G-083 — aiwf retitle does not sync entity body H1 with frontmatter title

## What's missing

`aiwf retitle <id> <new-title>` updates the entity's frontmatter `title:` field. For ACs (composite ids like `M-NNN/AC-N`), the verb also updates the body heading (`### AC-N — <title>`) atomically — this is documented in the verb's help and works as advertised. For top-level entities (epics, milestones, gaps, ADRs, decisions, contracts), the body's H1 line (`# <ID> — <title>`) is left at its old value, creating a frontmatter-vs-body divergence that the next reader has to manually reconcile via a follow-up `aiwf edit-body` commit.

The verb's help text says *"Update an entity's or AC's frontmatter title"* without naming the asymmetry, and the AC example explicitly highlights *"(updates frontmatter and body heading atomically)"*. A reader scanning the help reasonably expects the same atomic-body-update for the entity case.

## Why it matters

1. **Discoverable surface drift.** The H1 is what a human reading the entity file sees first; many AI assistants will surface it ahead of frontmatter. After a retitle, the frontmatter says one thing and the H1 says another until someone hand-edits the body. The two diverge silently.
2. **AI assistants reading the entity file get inconsistent signals.** Some surfaces (`aiwf show`, `aiwf status`, roadmap render) read the frontmatter `title:`; others (raw file inspection, doc-lint, web render) read the body. The two should agree.
3. **Two-commit follow-up adds noise to history.** Each entity retitle ends up requiring a `aiwf retitle ...` commit AND a separate `aiwf edit-body ...` commit just to sync the H1. `aiwf history <id>` then shows two events when the operator's intent was one.
4. **The asymmetry contradicts the verb's spirit.** `aiwf retitle` for ACs *does* sync the body heading. For entities it doesn't. Same verb, two behaviors.

## Reproducer

```
aiwf add epic --title "Original title"
# epic.md frontmatter:  title: 'Original title'
# epic.md body H1:      # E-NN — Original title

aiwf retitle E-NN "New title"
# epic.md frontmatter:  title: 'New title'        ← updated
# epic.md body H1:      # E-NN — Original title   ← STALE
```

Surfaced live during E-21 milestone planning on 2026-05-08. After `aiwf retitle E-21 "Open-work synthesis: aiwfx-whiteboard skill replaces critical-path.md"`, the H1 still read the old `recommended-sequence skill` title and required a separate `aiwf edit-body E-21` to sync — folded into a broader audit-sweep commit, but the underlying need was a manual H1 edit.

## Resolution shape

`aiwf retitle <entity-id>` should update **both** the frontmatter `title:` field and the body's H1 line in the same atomic commit, matching the AC behavior. The H1 pattern is well-defined for entities: `# <ID> — <title>`. The verb already locates the file by id; identifying the matching H1 is mechanical (first `# ` line in the body, optionally pattern-checked against the `<ID> — ` prefix).

If the body H1 doesn't match the expected pattern (operator hand-edited the H1 into a different shape, or the body has no H1), the verb has two reasonable choices:

- **Refuse with a hint** — *"M-080's body H1 doesn't match the canonical `# <ID> — <title>` shape. Run `aiwf edit-body M-080` to bring it into shape, then retry, or pass `--frontmatter-only` to bypass."* Safer; preserves operator intent.
- **Warn and proceed with frontmatter-only** — emits the warning to stderr but completes the title change. Less paternalistic; loses the chokepoint guarantee.

Lean: refuse-with-hint, with `--frontmatter-only` as the explicit opt-out for the rare case where the H1 has been intentionally diverged.

## Out of scope

- **Updating other body references to the entity's title.** Prose in the body that says *"this epic does X"* using the old title is not the verb's responsibility — that's an operator-driven prose update.
- **Updating cross-entity references.** Other entities' bodies that mention this entity by old title don't get auto-rewritten. Out of scope; would require a body-grep + rewrite pass that isn't this verb's job.
- **Updating ROADMAP.md / STATUS.md.** Those regenerate from the entity tree's frontmatter on the next render; they pick up the new title automatically.

## Discovered in

- E-21 milestone planning, 2026-05-08. After invoking `aiwf retitle` on E-21 to lock the new `aiwfx-whiteboard` skill name, the H1 still read the old title; a follow-up `aiwf edit-body E-21` commit was required to sync. The asymmetry was unexpected — the verb's example for ACs explicitly highlights body-heading-sync, suggesting the same shape for entities.

## References

- **ADR-0005** — *"Verb hygiene contract: complete, consistent, pre-flighted aiwf verbs"*. This gap implements obligation 2 (**atomic completeness over consistent surfaces**); filed under umbrella gap G-084.
- M-077 (E-22) — the milestone that added `aiwf retitle`. The AC-aware body-sync was implemented; entity-body-sync was not. The asymmetry may be a deliberate scope choice or an oversight; either way it warrants closing.
- G-081 — sibling gap on `aiwf rename`'s lack of pre-flight. Both gaps touch the rename / retitle family of verbs.
- G-065 (closed via M-077) — the original gap that motivated `aiwf retitle`. This new gap is a follow-up to its scope, not a re-litigation.
