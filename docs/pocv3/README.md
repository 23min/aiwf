# `pocv3/` — research-from-main landing zone

This branch (`poc/aiwf-v3`) is intentionally isolated from `main`: research documents and the earlier architecture design have been removed here so they do not pollute the working context. They remain on `main` for visitors who want to follow the design trajectory.

When a piece of research from `main` is needed *as input* to PoC iteration — e.g., to inform a feature plan, to settle a design question, to ground a decision — it lands in this directory rather than in `docs/` proper. That keeps the load-bearing PoC docs (`docs/poc-*.md`) flat and self-contained, while still letting research material live alongside the code that uses it.

## What goes here

- Excerpts of research from `main` that a PoC plan references but doesn't restate.
- Brainstorming notes that informed a `docs/poc-*.md` plan and may be useful again.
- Working documents that aren't yet committed to the PoC's load-bearing surface.

## What does *not* go here

- Active PoC plans — those live at `docs/poc-*.md`.
- Skill source — `tools/internal/skills/embedded/`.
- Engineering rules — `CLAUDE.md` and `tools/CLAUDE.md`.

If a document in `pocv3/` is being referenced by an active PoC plan, it should either move up to `docs/` (becoming part of the load-bearing surface) or be summarized inline in the plan that depends on it. `pocv3/` is for source material, not authoritative state.
