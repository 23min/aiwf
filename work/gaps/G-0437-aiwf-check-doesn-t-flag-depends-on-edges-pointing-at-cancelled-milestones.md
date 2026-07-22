---
id: G-0437
title: aiwf check doesn't flag depends_on edges pointing at cancelled milestones
status: addressed
addressed_by_commit:
    - d315c492906ed6e1d6998281f47410ef31e139af
---
## What's missing

`aiwf check` validates that `depends_on` ids resolve (`refs-resolve`) and that the milestone DAG has no cycles (`no-cycles/depends_on`), but it never inspects the *status* of the referent. When a milestone is cancelled while another non-terminal milestone still lists it in `depends_on`, the dependency can never be satisfied — the edge is either permanently unsatisfiable or silently means nothing — and `aiwf check` reports no finding at all.

## Why it matters

This is not hypothetical: a planning triage cancelled several epics and their milestones via `aiwf cancel`, and five still-active draft milestones kept `depends_on` edges pointing at the cancelled milestones. `aiwf check` reported clean for six days until a manual frontmatter walk surfaced the rot (recorded in G-0073's friction-evidence log, 2026-06-12). The fix is squarely within the already-shipped milestone-to-milestone `depends_on` scope — no cross-kind schema work is needed to close it. It is the narrow "dangling-on-cancel detection" follow-on G-0073 named but deferred as part of a larger, still-speculative cross-kind generalization; extracting it lets the concrete, already-proven pain get fixed without waiting on that broader epic.