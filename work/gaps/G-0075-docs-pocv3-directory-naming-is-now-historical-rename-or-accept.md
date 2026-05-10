---
id: G-0075
title: docs/pocv3/ directory naming is now historical; rename or accept
status: open
---
The `docs/pocv3/` directory name is now historical — it was the working name during the PoC era. After the trunk promotion (PROMOTION-PLAN.md, Step 5, commit `e0a7fe5`), the framework lives on `main` and the contents of `pocv3/` are the active design docs for the trunk.

Decision 1 of the promotion plan deliberately deferred the rename: *"docs/pocv3/ keeps its name for now. Might refactor docs later."* Two paths:

- **Rename** to `docs/framework/` or `docs/v3/` — more accurate going forward; requires cross-link updates throughout the tree (a doc-lint pass would surface them).
- **Leave as-is** — cheap; the directory name signals vintage but reads correctly enough that fresh readers don't get confused.

Surfaced during the trunk-promotion procedure (PROMOTION-PLAN.md, Step 6.G).
