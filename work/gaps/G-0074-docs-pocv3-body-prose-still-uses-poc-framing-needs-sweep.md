---
id: G-0074
title: docs/pocv3/ body prose still uses PoC framing; needs sweep
status: open
---
docs/pocv3/ body prose throughout (overview, design, plans, archive subdirectories) still reads as if the PoC is the current state and main is elsewhere — references like "the PoC will…", "post-PoC we will…", "the PoC commits to…" remain.

Step 3 of PROMOTION-PLAN.md (commit `987273d`) deliberately scoped the framing-edit pass to the root docs (CLAUDE.md, README.md, CHANGELOG.md) plus a single cross-anchor fix in `docs/pocv3/overview.md`. The pocv3 body prose was deferred per Decision 1 of the promotion ("docs/pocv3/ keeps its name for now; may refactor docs later").

Two resolution paths:

- **Sweep:** scrub pocv3 body prose to reframe "PoC" as historical when it refers to the now-merged trunk; preserve "PoC" only where it correctly names the historical era.
- **Accept:** leave as-is on the principle that the directory name `pocv3` already signals the doc-set's vintage; readers will infer correctly.

Surfaced during the trunk-promotion procedure (PROMOTION-PLAN.md, Step 6.G).
