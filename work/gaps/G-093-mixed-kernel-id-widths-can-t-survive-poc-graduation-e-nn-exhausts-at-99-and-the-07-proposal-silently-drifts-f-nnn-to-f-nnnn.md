---
id: G-093
title: Mixed kernel ID widths can't survive PoC graduation; E-NN exhausts at 99 and the §07 proposal silently drifts F-NNN to F-NNNN
status: open
---
`internal/verb/import.go::canonicalPadFor` encodes the kernel's current id-width policy: pad 2 for `epic`, pad 4 for `ADR`, pad 3 for everything else. That choice is the source of two compounding problems.

## Existing state

- **Epic** — `E-NN`, max 99. **`E-22` is the highest currently in use** in PoC self-hosting.
- **Milestone** — `M-NNN`, max 999. `M-080` is the highest in use.
- **Gap** — `G-NNN`, max 999. `G-092` is the highest and rising.
- **Decision / Contract** — `D-NNN` / `C-NNN`. Light usage; ample headroom.
- **ADR** — `ADR-NNNN`. Already 4 digits.
- **Finding (proposed by ADR-0003)** — declared as **F-NNN** in the ADR; the §07 TDD architecture proposal silently uses **F-NNNN** in its frontmatter examples (`id: F-1023`) and entity table while keeping F-NNN in surrounding prose, with no rationale for the divergence.

CLAUDE.md commitment #2 lists the widths as a fact: *"`E-NN`, `M-NNN`, `ADR-NNNN`, `G-NNN`, `D-NNN`, `C-NNN`. The id is the primary key."* The list is mixed by kind, the rationale per width is undocumented, and the policy isn't a single kernel rule — `canonicalPadFor` is the de facto authority but doesn't read as a policy declaration.

## Why it matters

1. **`E-NN` will rollover.** Epics are the kernel's load-bearing structural kind — the parent of milestones, the unit of work-shape declaration. Hitting 99 mid-consumer-lifetime breaks commitment #2 (stable id is the primary key). The PoC has consumed 22% of the space in self-hosting alone; any real downstream consumer will hit it within a year or two.
2. **Mixed widths break commitment #2's "primary key" feel.** When the same tree shows `E-22`, `M-080`, `ADR-0007`, the visual pattern is irregular. A reader scanning `aiwf list` reads three different format conventions for one kind of thing.
3. **The proposed F kind is already drifting.** ADR-0003 says F-NNN; the §07 TDD architecture proposal uses F-NNNN in its example frontmatter and entity table while keeping F-NNN in surrounding prose. Without a kernel-side policy declaration, every new kind relitigates the width question.
4. **Path-form refs in body prose are width-coupled.** Body content that links `[E-22](work/epics/E-22-foo)` keeps working at the file level today. If new entities allocate at 4 digits while old keep narrow widths, references in active prose either canonicalize-and-break or stay-narrow-and-drift.

## Fix shape

Three layers, sequenced via a companion ADR and a single implementing epic:

1. **Single kernel policy** — every kind canonicalizes to 4 digits. Parser accepts narrower legacy widths on input (so `aiwf-entity: E-22` in old commit trailers keeps validating). Allocator emits 4-digit going forward. Renderer always renders 4-digit form.
2. **One-shot rename pass for this repo** — files in `work/` and active `docs/adr/` rename to canonical widths; in-body references rewrite. **No new verb** — the migration runs once for this repo, with no other consumers; an ad-hoc script (or careful `git mv` + sed pass) does the work, verified by `aiwf check`. New consumers post-graduation are born canonical.
3. **Drift-check rule** — `aiwf check` rule `entity-id-narrow-width` warns on *new* files at non-canonical width (archive entries grandfathered per ADR-0004's forget-by-default).

The full plan lives in **ADR-NEW** (policy) and **E-NEW** (implementation). **E-NEW is a prerequisite of §07 TDD architecture's Slice 2** (F-kind allocation), so this work sequences ahead of any F-related milestone — F is born canonical rather than allocated narrow and rewidth'd later. This gap is the discovery framing.

## References

- **CLAUDE.md** "What aiwf commits to" §2 — current id-width list (per-kind exception list).
- **ADR-0003** — F-NNN as 7th kind; affected by this gap.
- **ADR-0004** — uniform archive convention; archives grandfathered.
- **`docs/explorations/07-tdd-architecture-proposal.md`** — review of this exploratory doc surfaced the gap; uses F-NNN ↔ F-NNNN inconsistently.
- **`internal/verb/import.go::canonicalPadFor`** — current de facto policy site.
