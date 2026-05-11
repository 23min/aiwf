---
id: G-0108
title: aiwf retitle should sync the on-disk slug, not just frontmatter title
status: open
---
# Problem

`aiwf retitle <id> "<new title>"` updates the entity's frontmatter `title:` field but does **not** rename the on-disk filename to reflect the new title. The slug stays at whatever it was when the entity was created.

This means an entity can drift into a state where:
- Frontmatter title: "Short and current"
- On-disk slug: `G-NNNN-very-long-historical-title-that-was-the-original-shape.md`

The two views (display vs filesystem) diverge silently.

# Why this matters

Surface-by-surface:

| Surface | Reads | Affected by retitle? |
|---|---|---|
| `aiwf list` / `aiwf status` / `aiwf show` | frontmatter title | ✅ updated |
| `aiwf render` HTML | frontmatter title | ✅ updated |
| `git log` subjects | full title written in commit msg | ✅ updated for future commits |
| **On-disk filename / filesystem path** | **slug** | ❌ unchanged |
| Tab-completion of paths | slug | ❌ unchanged |
| Cross-references in design docs (path-form links) | slug | ❌ unchanged |

So retitle gets you partway. The filesystem-pathology case G-0102 raised (filenames approaching the 255-byte limit) is *not* addressed by retitle alone.

# Surfaced by

G-0102's slug-cap cleanup pass. The CLAUDE.md grandfathering note (written at G-0102's wrap) said: *"operators retitle them manually when convenient via `aiwf retitle G-NNNN "..."`, which enforces the cap."* That's only half-true — the cap is enforced on the title, but the slug remains uncapped.

Today's cleanup discovered this when the on-disk filenames still showed long slugs after 22 retitle operations. The two-step workflow (retitle then rename) is the current workaround, but it's separate verbs with separate commits.

# Two related shapes

- **G-0083** — "aiwf retitle does not sync entity body H1 with frontmatter title." H1 sync is the body side; slug sync is the filesystem side. Both are "retitle should propagate the title change to derived surfaces" — could be the same gap with two ACs, or two gaps that close together.
- **The trunk-collision gap (filed alongside this one)** — even if retitle synced the slug, the rename half would still hit the kernel's trunk-collision bug on feature branches. Both kernel changes are needed before retitle-batches become atomic.

# Scope of the fix

Two design options:

1. **Add slug-sync to retitle.** Each `aiwf retitle` re-derives the slug from the new title and (atomically) renames the on-disk file in the same commit. Composes with G-0081's pre-flight + the trunk-collision detector fix.
2. **Leave retitle title-only; document the two-step.** The operator runs `aiwf retitle` then `aiwf rename` deliberately. Status quo, but the CLAUDE.md docs spell it out.

Option 1 matches the operator's mental model (one verb, one observable change). Option 2 is the verb-hygiene-contract-compliant version (each verb does one thing).

The aiwf grain leans toward Option 1: the kernel already has `aiwf rename` as a separate verb when you only want the slug change; consolidating "title + slug" into retitle removes a class of "I forgot to also rename" bugs.

# Decision (locked 2026-05-11)

**Option 1 — extend `aiwf retitle` to atomically sync the on-disk slug** alongside the frontmatter title. Operator-ergonomics over verb-hygiene-strictness: the "I forgot to also rename" class of bug is a real friction surface, observed today during the G-0102 cleanup pass. Verb-hygiene defenders can read `aiwf retitle` as "rename the entity end-to-end" — one observable change to the operator, even if the implementation touches two fields. `aiwf rename` stays as the slug-only verb for the "I just want a slug tweak" case.

# Why not urgent

The two-step workflow exists; the docs (after this gap surfaces) can describe it. No active work is blocked. Promote when retitle-batches become frequent enough that the two-step friction matters.

# Suggested resolution

Wf-patch shape: extend `aiwf retitle` to call the slug-rename logic atomically (one verb = one commit, two file changes: frontmatter + rename). Depends on the trunk-collision detector fix landing first (otherwise retitle-batches still hit the same wall as today).
