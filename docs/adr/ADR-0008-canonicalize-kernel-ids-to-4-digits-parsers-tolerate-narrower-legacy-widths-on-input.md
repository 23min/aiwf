---
id: ADR-0008
title: Canonicalize kernel IDs to 4 digits; parsers tolerate narrower legacy widths on input
status: proposed
---
## Context

Kernel principle #2 says ids are stable primary keys: *"`E-NN`, `M-NNN`, `ADR-NNNN`, `G-NNN`, `D-NNN`, `C-NNN`. The id is the primary key; the slug is just display."* The widths are mixed by kind — 2-digit for epic, 3-digit for milestone/gap/decision/contract, 4-digit for ADR — encoded in `internal/verb/import.go::canonicalPadFor`. There is no declared policy; the function is the de facto authority.

The companion gap [G-093](../../work/gaps/G-093-mixed-kernel-id-widths-can-t-survive-poc-graduation-e-nn-exhausts-at-99-and-the-07-proposal-silently-drifts-f-nnn-to-f-nnnn.md) documents the symptoms: epics will exhaust at 99 (E-22 is the current high-water mark from PoC self-hosting); ADR-0003 declares F-NNN but the §07 TDD architecture proposal silently drifts to F-NNNN; CLAUDE.md commitment #2 enumerates the widths as historical fact, not as a rule a future kind would consult.

The kernel's own self-hosting pressure plus pre-graduation timing make this the right moment to lock the policy. With no external consumers, the migration cost is bounded by this repo's tree (~100 entities + 27 narrow-width mentions in the rituals plugin). After graduation, the cost compounds with every new adopter.

## Decision

**Every kernel id kind canonicalizes to 4 digits. Parsers accept narrower legacy widths on input. Allocators emit 4-digit form going forward. Renderers canonicalize to 4-digit on every display surface.**

```
Epic       E-NNNN     (was E-NN)
Milestone  M-NNNN     (was M-NNN)
Gap        G-NNNN     (was G-NNN)
Decision   D-NNNN     (was D-NNN)
Contract   C-NNNN     (was C-NNN)
Finding    F-NNNN     (ADR-0003 amended; was F-NNN)
ADR        ADR-NNNN   (unchanged)
```

The composite AC id `M-NNNN/AC-N` follows the milestone width.

### Parser tolerance

Loaders, refs resolvers, and trailer parsers accept both `E-22` and `E-0022` as the same id. The id-resolver canonicalizes on read: an old commit trailer with `aiwf-entity: E-22` continues to match `aiwf history E-22` *and* `aiwf history E-0022`. No git history rewrite. The kernel never re-emits narrow-width ids — it only accepts them on input.

This satisfies commitment #2's "stable id survives rename" — only the *display* width changes; the underlying integer is what's stable.

### Allocator behavior

`canonicalPadFor(kind)` returns 4 for every kind. New ids are always allocated and rendered at 4-digit form. The next epic after E-22 is E-0023, not E-23.

### Renderer canonicalization

Every display surface — `aiwf list`, `aiwf status`, `aiwf show`, `aiwf history`, `aiwf render --format=html`, JSON envelope output — emits canonical 4-digit form. Existing files on disk keep their birth-width filename until the one-shot migration (next subsection); the renderer's canonical form is consistent regardless.

### Migration

**No new verb.** A one-time PR in this repo renames active-tree files (`E-22-foo.md` → `E-0022-foo.md`, etc.) and rewrites in-body references. The rename is performed by an ad-hoc script under `scripts/migrate-id-widths/` (retained for the historical record) or by a careful manual `git mv` + sed pass; either is acceptable. Verification: `aiwf check` is green afterwards plus structural assertions over the post-rename tree.

The kernel ships parser tolerance and renderer canonicalization in the *first* milestone of the implementing epic, so old narrow widths keep working through the entire migration window — old commits validate, old references resolve, old skills don't break.

New consumers (post-graduation) are born at canonical width. They never run a migration; their `aiwf init` allocator is already 4-digit.

### Drift control — `entity-id-narrow-width` finding

After the migration, `aiwf check` fires `entity-id-narrow-width` (warning) on any active-tree file at non-canonical width. Pre-existing files in `<kind>/archive/` keep their narrow width forever (per ADR-0004's forget-by-default principle for archives) and don't fire the rule.

The rule is sequenced last in the implementing epic — after the rename pass — so it fires against an already-canonical active tree. It never warns on pre-migration narrow files because the rename pass has already moved the active tree to canonical form.

The chokepoint is `internal/check/`, not the allocator alone — defense in depth: even if a future allocator regression emitted narrow widths, the next `aiwf check` would catch it.

### Reversal — what verb undoes canonicalization?

**You don't.** The change is monotone-additive at the parser layer (acceptance widens) and forward-only at the allocator (always 4-digit). The migration commit is reversible by `git revert` like any other content commit. There is no "narrow it back" verb because there is no use case — the kernel only emits canonical form going forward, regardless of whether the migration has run.

## Consequences

**Positive:**

- **Commitment #2 becomes a single rule, not a per-kind exception list.** Future kinds inherit the policy automatically.
- **No more visible width inconsistency** in `aiwf list`, `aiwf status`, etc. — every id reads the same width.
- **Epic exhaustion is lifted by two orders of magnitude.** 9999 instead of 99.
- **F-NNN/F-NNNN drift in §07 resolves itself** — F is born at canonical width when ADR-0003 implementation lands, with no separate decision needed.
- **No new verb.** No CLI surface debt for a one-shot ritual.
- **Parser tolerance means zero compatibility break.** Old commits, old branches, old skills with hardcoded narrow widths keep working indefinitely.

**Negative:**

- **Old filenames stay narrow until the migration PR runs** (transient one-shot). During that window, `ls work/epics/` shows `E-22-foo.md` while `aiwf show E-22` renders `E-0022`. Resolved by the migration milestone.
- **The migration PR is a large content rewrite.** ~100 file renames + N reference rewrites. One careful PR; verifiable by `aiwf check` plus structural assertions; not a kernel-correctness risk; just a review-cost risk.
- **Path-form refs in old commits permanently point at narrow filenames.** A body in an archived gap saying `[E-22](work/epics/E-22-foo.md)` still works post-migration only if the *file* keeps a narrow alias or the rename is restricted to active entities. This ADR resolves it by **not renaming files in `<kind>/archive/`** — narrow widths in archives are preserved per the forget-by-default principle, and active-tree refs only point at active-tree files (which all migrate at once). G-091's preventive check rule gives long-term protection.
- **One trailing minor detail to coordinate** — the rituals plugin's embedded skills mention narrow forms in 5 files (27 mentions total). The implementing epic's final milestone refreshes these and records the cross-repo SHA per CLAUDE.md "Cross-repo plugin testing".

## Alternatives considered

- **Keep mixed widths; add a width per kind only when the next exhaustion looms.** The "let it bleed" answer. Rejected: every future kind would relitigate the question, and `E-NN` is already exhaustion-imminent within consumer lifetimes. The kernel has no policy declaration for the next kind to consult.
- **New `aiwf rewidth` verb.** Adds permanent CLI surface for a one-shot operation; no good answer to "what verb undoes this?"; expands skill-coverage obligations under ADR-0006. Rejected — verbs are for repeatable operations, not one-shot rituals.
- **Fold migration into `aiwf update --rewrite-id-widths` flag.** Acceptable fallback if a tested, repeatable migration code path is wanted. Rejected for now — `aiwf update`'s job description is "regenerate framework-owned artifacts" (skills, hooks); rewriting user-owned content (`work/`) crosses that line. Revisit if a real graduating consumer hits the same problem.
- **Kernel-only canonicalization, no rename pass.** Files keep birth-width forever; renderer canonicalizes at display. Smallest possible work. Rejected because path-form refs in body prose break (linking `[E-0022](work/epics/E-22-foo.md)` doesn't resolve at the file level), and `ls work/epics/` shows a permanent visual mix that nags without dollar-stopping.
- **Width 5 or 6 instead of 4.** Future-proofs further. Rejected as YAGNI — 9999 epics is enough headroom for the kernel's lifetime; ADR-0007 already established 4-digit ADR with no exhaustion concerns.
- **Per-kind policy table (`E:4, M:5, F:5, ...`).** Closer to "right-size each kind." Rejected — a uniform width is simpler, and the marginal cost of extra padding is one character per id-render. KISS beats per-kind tuning here.

## References

- [G-093](../../work/gaps/G-093-mixed-kernel-id-widths-can-t-survive-poc-graduation-e-nn-exhausts-at-99-and-the-07-proposal-silently-drifts-f-nnn-to-f-nnnn.md) — companion gap that surfaced this work.
- **CLAUDE.md** "What aiwf commits to" §2 — current id-width statement, updated by the implementing epic's third milestone.
- [ADR-0003](ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md) — F-NNN as 7th entity kind; **amended by this ADR's implementing epic** (the docs-and-drift milestone updates the ADR's id-pattern paragraph from F-NNN to F-NNNN).
- [ADR-0004](ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — Uniform archive convention; archive entities keep their birth-width per forget-by-default.
- `internal/verb/import.go::canonicalPadFor` — current pad-policy site; relocated and broadened by the implementing epic's first milestone.
- `docs/explorations/07-tdd-architecture-proposal.md` — exploratory doc whose review surfaced this; F-NNN ↔ F-NNNN drift resolved by this ADR.
- **G-091** — body-prose path-form refs have no preventive check (related; not blocking; long-term protection layer).
