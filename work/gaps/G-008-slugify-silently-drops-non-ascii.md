---
id: G-008
title: Slugify silently drops non-ASCII
status: addressed
addressed_by_commit:
  - 668031c
---

Resolved in commit `668031c` (fix(aiwf): G8 — surface a warning when a non-ASCII title's slug drops chars). New `entity.SlugifyDetailed` returns both the slug and the list of dropped runes; `Slugify` is now a thin wrapper. `verb.Add` and `verb.Rename` surface a `slug-dropped-chars` warning naming the dropped characters and the resulting slug — the verb still succeeds (the YAGNI option per the proposed fix). A user who titled an entity `"Café au Lait"` gets `caf-au-lait` plus a clear one-line notice instead of a silent-then-confusing follow-up rename.

---
