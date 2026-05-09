---
id: ADR-0008
title: Canonicalize kernel IDs to 4 digits; parsers tolerate narrower legacy widths on input
status: proposed
---
## Context

Kernel principle #2 says ids are stable primary keys: *"`E-NN`, `M-NNN`, `ADR-NNNN`, `G-NNN`, `D-NNN`, `C-NNN`. The id is the primary key; the slug is just display."* The widths are mixed by kind — 2-digit for epic, 3-digit for milestone/gap/decision/contract, 4-digit for ADR — encoded in `internal/verb/import.go::canonicalPadFor`. There is no declared policy; the function is the de facto authority.

The companion gap [G-093](../../work/gaps/G-093-mixed-kernel-id-widths-can-t-survive-poc-graduation-e-nn-exhausts-at-99-and-the-07-proposal-silently-drifts-f-nnn-to-f-nnnn.md) documents the symptoms: epics will exhaust at 99 (E-22 is the current high-water mark from PoC self-hosting); ADR-0003 declares F-NNN but the §07 TDD architecture proposal silently drifts to F-NNNN; CLAUDE.md commitment #2 enumerates the widths as historical fact, not as a rule a future kind would consult.

The kernel's own self-hosting pressure plus pre-graduation timing make this the right moment to lock the policy. Multiple downstream consumers have already adopted aiwf with narrow-width trees; each will need to migrate to canonical width when they upgrade past the kernel version that ships this change.

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

Every display surface — `aiwf list`, `aiwf status`, `aiwf show`, `aiwf history`, `aiwf render --format=html`, JSON envelope output — emits canonical 4-digit form. Existing files on disk keep their birth-width filename until the consumer runs the migration verb (next subsection); the renderer's canonical form is consistent regardless.

### Migration — `aiwf rewidth` verb

A new top-level verb sweeps a consumer's active tree from narrow to canonical width.

```
aiwf rewidth [--apply] [--root <path>]
```

- **Default is dry-run.** Without `--apply`, the verb prints the planned moves and reference rewrites and exits without touching the tree. Re-run with `--apply` to commit.
- **Single commit per `--apply` invocation.** Per kernel principle #7, one verb invocation produces one commit. The commit message body lists per-kind rename counts, reference-rewrite counts, and the canonical-width policy version. Trailer is `aiwf-verb: rewidth` (no `aiwf-entity:` trailer — multi-entity sweeps are a special case in the trailer-keys policy, same shape as `aiwf archive`).
- **Active-tree only.** Files in `<kind>/archive/` keep their birth-width filenames per ADR-0004's forget-by-default. Archive references in body prose stay narrow; active references rewrite to canonical.
- **Idempotent.** Running on an already-canonical tree is a no-op; the verb prints "no changes needed" and exits without commit.
- **Rerunnable.** A consumer who runs `--apply` once, then later allocates new entities (canonical), will see no changes from a subsequent dry-run because the tree is already uniform-canonical.

Concretely, `aiwf rewidth --apply` walks each kind's active directory, computes the canonical-width filename for each entity, performs the `git mv`, then sweeps in-body references (markdown links + prose mentions + composite ids) to canonical form, then produces one commit. Verification: `aiwf check` is green afterwards plus the rule's tree-state-based detection (next subsection) shows uniform-canonical active tree.

### Reversal — what verb undoes rewidth?

**You don't.** The change is forward-only at the allocator (always 4-digit) and structurally one-shot per consumer. The migration commit is reversible by `git revert` like any other content commit. There is no "narrow it back" verb because there is no use case — the kernel only emits canonical form going forward.

### Drift control — `entity-id-narrow-width` finding

`aiwf check` includes a tree-state-based rule that distinguishes pre-migration legacy from post-migration regression without configuration or markers:

- **Uniform narrow active tree** → consumer hasn't run `aiwf rewidth` yet → silent.
- **Uniform canonical active tree** → consumer has migrated cleanly → silent.
- **Mixed active tree** (some canonical alongside some narrow) → warning fires on the narrow files. Effective message: "you've started accruing canonical entities; finish the migration."

Archive entries (`<kind>/archive/`) are excluded from the mixed-state computation entirely — archive width never participates in the active-tree state assessment. Pre-existing narrow files in archive stay narrow forever per ADR-0004's forget-by-default principle.

The signal works for both directions:
- A consumer who upgrades, allocates one canonical entity (via the new allocator), and then runs `aiwf check` sees the warning prompting them to run `aiwf rewidth`.
- A consumer who has migrated, then somehow ends up with a narrow file (hand-edit, allocator regression) sees the same warning prompting investigation.

A consumer who upgrades and never allocates anything new stays uniform-narrow indefinitely. The rule is silent. That matches the on-demand framing — the kernel doesn't nag.

The chokepoint is `internal/check/`, not the allocator alone — defense in depth: even if a future allocator regression emitted narrow widths into a previously-canonical tree, the next `aiwf check` would catch it.

## Consequences

**Positive:**

- **Commitment #2 becomes a single rule, not a per-kind exception list.** Future kinds inherit the policy automatically.
- **No more visible width inconsistency** in `aiwf list`, `aiwf status`, etc. — every id reads the same width.
- **Epic exhaustion is lifted by two orders of magnitude.** 9999 instead of 99.
- **F-NNN/F-NNNN drift in §07 resolves itself** — F is born at canonical width when ADR-0003 implementation lands, with no separate decision needed.
- **Tested, distributed migration path.** Every consumer gets `aiwf rewidth` via `go install`; no per-consumer ad-hoc scripts; one canonical implementation everyone shares.
- **Parser tolerance means zero compatibility break.** Old commits, old branches, old skills with hardcoded narrow widths keep working indefinitely.
- **No nagging on pre-migration trees.** Tree-state-based drift detection means consumers see warnings only when they're already mid-migration or have regressed; uniform-narrow consumers stay silent until they choose to migrate.

**Negative:**

- **Old filenames stay narrow until the consumer runs `aiwf rewidth --apply`** (transient one-shot per consumer). During that window, `ls work/epics/` shows `E-22-foo.md` while `aiwf show E-22` renders `E-0022`.
- **The `aiwf rewidth` verb adds permanent CLI surface for a one-shot ritual.** Mitigation: idempotent, self-documenting via `--help`, sunset path is "remove in a future major version once all known consumers have migrated" — cheap to keep otherwise.
- **Path-form refs in archived bodies stay pointing at narrow filenames forever.** Resolved by **not** renaming files in `<kind>/archive/` — narrow widths in archives are preserved per the forget-by-default principle, and active-tree refs only point at active-tree files (which all migrate at once). G-091's preventive check rule gives long-term protection.
- **One trailing minor detail to coordinate** — the rituals plugin's embedded skills mention narrow forms in 5 files (27 mentions total). The implementing epic's final milestone refreshes these and records the cross-repo SHA per CLAUDE.md "Cross-repo plugin testing".

## Alternatives considered

- **Keep mixed widths; add a width per kind only when the next exhaustion looms.** The "let it bleed" answer. Rejected: every future kind would relitigate the question, and `E-NN` is already exhaustion-imminent within consumer lifetimes. The kernel has no policy declaration for the next kind to consult.
- **No new verb; one-shot manual rename PR per repo.** Initially recommended on the YAGNI argument that the migration runs once for this repo, with no other consumers to support. **Rejected after the downstream-consumer population surfaced** — multiple existing consumer repos already run aiwf with narrow-width trees; each needs migration. A distributed, tested, idempotent verb beats N consumers each inventing their own ad-hoc script. The "what verb undoes this?" question is answered the same way as `aiwf init`: one-shot ritual; reversal is `git revert` if needed.
- **Fold migration into `aiwf update`.** Acceptable but stretches the verb's job description — `aiwf update` is "regenerate framework-owned artifacts" (skills, hooks); rewriting user-owned content (`work/`) is a different category. Rejected in favor of a clearly-named single-purpose verb that consumers explicitly opt into.
- **Kernel-only canonicalization, no rename pass.** Files keep birth-width forever; renderer canonicalizes at display. Smallest possible work. Rejected because path-form refs in body prose break (linking `[E-0022](work/epics/E-22-foo.md)` doesn't resolve at the file level), and `ls work/epics/` shows a permanent visual mix.
- **Width 5 or 6 instead of 4.** Future-proofs further. Rejected as YAGNI — 9999 epics is enough headroom for the kernel's lifetime; ADR-0007 already established 4-digit ADR with no exhaustion concerns.
- **Per-kind policy table (`E:4, M:5, F:5, ...`).** Closer to "right-size each kind." Rejected — a uniform width is simpler, and the marginal cost of extra padding is one character per id-render. KISS beats per-kind tuning here.
- **Marker-based drift detection** (verb writes `aiwf.yaml: migration.id-widths-applied: true`; rule reads marker). Cleaner conceptually but adds config-surface and a piece of state the consumer doesn't directly own. Rejected in favor of tree-state-based detection, which infers the same signal from the tree's actual shape without extra state.

## References

- [G-093](../../work/gaps/G-093-mixed-kernel-id-widths-can-t-survive-poc-graduation-e-nn-exhausts-at-99-and-the-07-proposal-silently-drifts-f-nnn-to-f-nnnn.md) — companion gap that surfaced this work.
- **CLAUDE.md** "What aiwf commits to" §2 — current id-width statement, updated by the implementing epic.
- [ADR-0003](ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md) — F-NNN as 7th entity kind; **amended by this ADR's implementing epic** (the docs-and-drift milestone updates the ADR's id-pattern paragraph from F-NNN to F-NNNN).
- [ADR-0004](ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — Uniform archive convention; archive entities keep their birth-width per forget-by-default.
- `internal/verb/import.go::canonicalPadFor` — current pad-policy site; relocated and broadened by the implementing epic's first milestone.
- `docs/explorations/07-tdd-architecture-proposal.md` — exploratory doc whose review surfaced this; F-NNN ↔ F-NNNN drift resolved by this ADR.
- **G-091** — body-prose path-form refs have no preventive check (related; not blocking; long-term protection layer).
