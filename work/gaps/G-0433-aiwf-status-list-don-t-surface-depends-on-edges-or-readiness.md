---
id: G-0433
title: aiwf status/list don't surface depends_on edges or readiness
status: open
priority: low
discovered_in: M-0126
---
## Problem

Milestones can declare `depends_on` edges (other milestones they wait on) in frontmatter, but the primary `aiwf status` and `aiwf list` output doesn't surface them — they're only visible via the `--worktrees` lens or by reading raw frontmatter. A reader (human or AI) checking "what's blocking milestone X" has no direct answer from the primary surfaces. Surfaced by `docs/pocv3/plans/observability-surfaces-plan.md`'s Phase 1 (self-described "unconditional... standing intention," as opposed to Phases 2-3 which are explicitly gated on shown need). Distinct from G-0073 (which is about widening what `depends_on` can express across entity kinds — a schema question); this is purely a display question over data that already exists today.

## Direction

- Surface existing `depends_on` edges for milestones directly in the primary `aiwf status` and `aiwf list` output (text and JSON), not just `--worktrees`.
- Add a readiness marker: an in-flight/draft milestone is **ready** when every `depends_on` entry is terminal, else **blocked**, naming the open blocker(s).

Local-vs-origin delta (the plan's third Phase-1 item) is explicitly out of scope for this gap — deferred, not filed.