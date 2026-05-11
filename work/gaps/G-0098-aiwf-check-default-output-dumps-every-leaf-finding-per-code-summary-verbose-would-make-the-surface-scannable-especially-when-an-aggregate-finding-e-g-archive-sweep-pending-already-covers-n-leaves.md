---
id: G-0098
title: aiwf check default output dumps every leaf finding; per-code summary + --verbose would make the surface scannable, especially when an aggregate finding (e.g. archive-sweep-pending) already covers N leaves
status: addressed
discovered_in: M-0086
addressed_by:
    - E-0026
---

## What's missing

`aiwf check` renders one line per leaf finding, with no aggregation. On the kernel tree in its post-E-0024 worked example the verb emitted 176 advisories — 174 of them near-identical `terminal-entity-not-archived` lines sharing the same hint, paired with one `archive-sweep-pending` aggregate that already summarises the same surface. A scannable default would group warnings by `Code`, render one summary line per code (`<code> (warning) × N — <sample message>`), and reserve per-instance rendering for `--verbose`. Errors stay per-instance regardless — each error is per-instance-actionable. JSON envelope output is unaffected; the change is text-render only. This is shape B from the post-E-0024 discussion: collapse-by-code at the render layer rather than touching the check rules themselves. E-0026 / M-0089 is the implementation.

## Why it matters

The 176-line default output is the surface every consumer (human or AI) reads first when validating planning state. When the only signal in 176 lines is "the archive sweep is pending — you already know," the noise drowns the signal: a real finding hiding in a different code is hard to spot, and operators learn to ignore the output. The aggregate-paired finding shape (`archive-sweep-pending` summarising `terminal-entity-not-archived` leaves) is the new normal until the historical migration runs, so the friction is structural, not transitional. A per-code summary collapses the 176 lines to roughly the count of distinct warning codes (here, 4) and makes "is there a new finding I should look at?" answerable in one screen.
