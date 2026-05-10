---
id: M-0083
title: Drift check, normative-doc amendments, and skill content refresh
status: in_progress
parent: E-0023
depends_on:
    - M-0081
    - M-0082
tdd: required
acs:
    - id: AC-1
      title: Drift-check rule entity-id-narrow-width with tree-state detection
      status: met
      tdd_phase: done
    - id: AC-2
      title: Normative docs amended for canonical 4-digit policy
      status: met
      tdd_phase: done
    - id: AC-3
      title: Doc-tree narrow-id sweep complete
      status: met
      tdd_phase: done
    - id: AC-4
      title: Skill content refreshed in kernel and rituals plugin
      status: met
      tdd_phase: done
    - id: AC-5
      title: Active-tree drift check green on this repo post-M-C
      status: met
      tdd_phase: done
---
## Goal

Lock the canonical 4-digit policy in normative docs, ship the drift-check rule that catches future width regressions, and refresh skill examples in this repo and the rituals plugin. After M-C ships, the kernel's "what aiwf commits to" §2 reads as a single uniform rule; ADR-0003 reflects F-NNNN; the rituals plugin shows canonical examples; `aiwf check` produces no width-related findings on this repo's tree.

This is the policy-locking and drift-prevention milestone — the load-bearing kernel changes (M-A) and migration verb + repo migration (M-B) are already in place. M-C makes the policy permanent in the normative documentation layer and adds the chokepoint that prevents quiet regressions.

## Context

ADR-0008's "Drift control" subsection specifies the rule shape (tree-state-based detection); §"Migration" specifies the cross-repo skill-refresh coordination point. With M-A and M-B in place, M-C's job is the documentation sweep + chokepoint addition that closes the policy out.

The drift check is sequenced last so it fires against an already-canonical active tree (in this repo). On a fresh consumer repo post-upgrade, the rule is silent until the consumer either allocates a canonical entity (mixing the tree → warning fires) or runs `aiwf rewidth --apply` (uniformly canonicalizing → silent). That asymmetry is exactly the on-demand framing.

## Acceptance criteria

(ACs allocated separately via `aiwf add ac` after milestone creation; bodies seeded at allocation time.)

## Constraints

- **TDD: required.** Each AC drives a red→green→refactor cycle. AC-1 (drift rule) is net-new check logic; AC-2 (doc amendments) and AC-3 (doc sweep) are structural-assertion-driven content edits per CLAUDE.md's AC-promotion-evidence rule for `tdd: required` milestones.
- **Structural assertions, not substring matches.** Per CLAUDE.md's "substring assertions are not structural assertions" rule: doc-amendment ACs (AC-2) verify content under named section headings, not flat over the file.
- **Cross-repo coordination point.** The wrap commit records the rituals-plugin commit SHA; M-C's epic does not promote to `done` until that SHA is reachable from the marketplace.
- **Forget-by-default for archives.** ADR-0004's principle for entity archives extends to documentation archives (`docs/pocv3/archive/`). The doc sweep (AC-3) preserves archive content; the drift rule (AC-1) excludes archive entries from the active-tree state computation.

## Design notes

### Drift-check rule mixed-state computation

The rule's logic: enumerate active-tree entity files (per kind, excluding `<kind>/archive/`); classify each as narrow-width or canonical-width; if both classes are non-empty, the tree is "mixed" and the rule fires on each narrow-width file. Else the tree is uniform (silent).

Edge cases:

- Empty active tree (no entities): uniform; silent.
- Single entity at narrow width: uniform-narrow; silent.
- Single entity at canonical width: uniform-canonical; silent.
- Two entities, one each width: mixed; one warning on the narrow one.

Archive entries never participate in the classification — they're filtered out before counting.

### Doc-tree sweep boundaries

The sweep covers normative and exploratory doc trees (anywhere outside `<kind>/archive/` analogues). The rule of thumb: "if a future reader could mistake this for current authoritative content, canonicalize it." Historical text-record archives keep their birth-width.

A small allowlist captures intentional narrow-id references — e.g., a release note that says "v0.1.0 introduced E-NN" should keep the narrow form because that's literally what the historical state was. Each allowlist entry has a rationale comment.

### Cross-repo skill refresh

Per CLAUDE.md "Cross-repo plugin testing": rituals-plugin skill bodies are authored as fixtures in this repo at `internal/policies/testdata/<skill>/SKILL.md`. AC-4 updates those fixtures. The rituals repo's actual `SKILL.md` files are updated as a separate commit there; the SHA is recorded in M-C's wrap.

The marketplace cache test (already in place per E-0014) compares each fixture against the local marketplace cache and fires if they diverge — skipping cleanly when the cache is absent. M-C's wrap is gated on this test being green.

## Surfaces touched

- `internal/check/` — new rule `entity-id-narrow-width` with tree-state-based logic.
- `docs/adr/ADR-0003-...md` — amended for F-NNNN.
- `CLAUDE.md` — commitment #2 collapsed to uniform rule.
- `docs/explorations/`, `docs/pocv3/design/`, `docs/pocv3/plans/`, `README.md`, `CHANGELOG.md` — narrow-id sweep.
- `internal/policies/testdata/<skill>/SKILL.md` — embedded skill fixtures refreshed.
- Rituals plugin: 5 files at `/Users/peterbru/Projects/ai-workflow-rituals/plugins/aiwf-extensions/...` — refreshed (cross-repo).
- M-C wrap-side spec — Validation section records rituals-plugin SHA.

## Out of scope

- Width 5 or 6 future-proofing — YAGNI per ADR-0008.
- Per-kind width tuning — rejected in ADR-0008.
- Marker-based drift detection — rejected in ADR-0008.
- Doc-archive content rewriting — preserved per forget-by-default.
- G-0091's preventive check rule for path-form refs — related but separate.
- Any §07 TDD architecture proposal advancement — tracked separately.
- New cycle skills or other rituals additions beyond example refreshes — out of this epic's scope.

### AC-1 — Drift-check rule entity-id-narrow-width with tree-state detection

`aiwf check` includes the rule `entity-id-narrow-width` with tree-state-based detection:

- **Uniform narrow active tree** → consumer hasn't run `aiwf rewidth` yet → silent.
- **Uniform canonical active tree** → consumer has migrated cleanly → silent.
- **Mixed active tree** (some canonical alongside some narrow) → warning fires on each narrow file. Effective message: "narrow-width id detected in mixed-state active tree; run `aiwf rewidth` to complete the migration."

Archive entries (`<kind>/archive/`) are excluded from the mixed-state computation entirely — archive width never participates in the active-tree state assessment.

The signal works for both directions:
- A consumer who upgrades, allocates one canonical entity (via M-A's allocator), and runs `aiwf check` sees the warning prompting them to run `aiwf rewidth`.
- A consumer who has migrated, then somehow ends up with a narrow file (hand-edit, allocator regression) sees the same warning prompting investigation.

A consumer who upgrades and never allocates anything new stays uniform-narrow indefinitely. The rule is silent. That matches the on-demand framing.

Verified by table-driven test with three fixture trees: uniform-narrow (rule silent), uniform-canonical (rule silent), mixed (rule fires on narrow files only). Edge cases: empty tree (silent); single-entity tree at either width (silent). Archive subdirectories tested both empty and populated with narrow entries — neither case affects the active-tree state assessment.

### AC-2 — Normative docs amended for canonical 4-digit policy

Two normative-doc updates with structural assertions:

- **ADR-0003 §"Id and storage"** reads `F-NNNN` (was `F-NNN`) with a cross-reference to ADR-0008. The composite "F-NNN; same family as G-NNN, D-NNN" sentence is rewritten to reflect the unified width.
- **CLAUDE.md "What aiwf commits to" §2** reads as a single uniform rule (every id is 4 digits) with parser-tolerance note and a mention of `aiwf rewidth` for legacy migration. The previous per-kind list of widths is removed.

Verified by structural assertions over named sections (per CLAUDE.md "substring assertions are not structural assertions" rule — assertions scoped to the section heading hierarchy, not flat over the file). Each test parses the markdown, navigates to the named section, and asserts the new content appears under that heading. The structural test fires if either amendment is present in the wrong section, absent, or split across multiple sections.

The amended ADR-0003 stays at `status: proposed` (no status change here); only its body content is updated. CLAUDE.md is plain markdown (not an aiwf entity) and updates via standard commit.

### AC-3 — Doc-tree narrow-id sweep complete

Hardcoded narrow-id mentions in non-entity tracked files are updated to canonical width. Scope:

- `docs/explorations/` (active exploratory docs).
- `docs/pocv3/design/` and `docs/pocv3/plans/` (design docs).
- `README.md`, `CHANGELOG.md` (top-level).
- Other tracked files referencing entity ids in prose.

**Excluded from sweep:**

- `docs/pocv3/archive/` (historical text-record archive; preserves birth-width per ADR-0004 forget-by-default).
- Code-fence content (literal id text in technical documentation).
- Inline backtick spans (literal id text).

Verified by structural grep: post-sweep, narrow-id pattern matches in non-archive tracked files appear only inside code fences, inline spans, or the explicit allowlist of historical mentions. Allowlist is small, named, and committed alongside the sweep — each entry has a rationale comment (e.g., "release note describing v0.1.0's narrow-width origin").

The sweep can be performed by a small script (similar to `aiwf rewidth`'s reference-rewrite engine but scoped to docs prose) or by careful manual edits. Either approach satisfies the AC; the structural-grep assertion is what matters at promotion.

### AC-4 — Skill content refreshed in kernel and rituals plugin

Skill bodies in two locations refresh to canonical 4-digit id examples:

- **Kernel-embedded skills** at `internal/policies/testdata/<skill>/SKILL.md` — examples rewritten to canonical. The drift-prevention test in `internal/policies/` (per M-0074) passes against the local marketplace cache when present, skipping cleanly when absent.
- **Rituals plugin skills** at `/Users/peterbru/Projects/ai-workflow-rituals/`, the 5 enumerated files (27 narrow-width refs total):
  - `plugins/aiwf-extensions/templates/epic-spec.md` (1 ref)
  - `plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md` (10 refs)
  - `plugins/aiwf-extensions/skills/aiwfx-whiteboard/SKILL.md` (14 refs)
  - `plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md` (1 ref)
  - `plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md` (1 ref)

The rituals-plugin commit SHA is recorded in M-C's wrap-side spec Validation per CLAUDE.md "Cross-repo plugin testing." M-C's epic does not promote to `done` until the rituals SHA is reachable from the marketplace.

Verified by:
- Embedded fixtures: structural grep for narrow-width patterns returns empty in `internal/policies/testdata/<skill>/SKILL.md`.
- Rituals: marketplace-cache drift test green when cache present (skipped cleanly when absent).
- Cross-repo SHA recorded in M-C's wrap commit body and Validation section.

### AC-5 — Active-tree drift check green on this repo post-M-C

After M-C ships and `aiwf check` runs on this repo's active tree (uniform-canonical post-M-B), the new `entity-id-narrow-width` rule produces zero warnings.

Verified by: `aiwf check --format=json` post-M-C ship; assert that no findings carry the code `entity-id-narrow-width`. This is the load-bearing assertion for the milestone outcome — the rule fires only on mixed state, M-B made the active tree uniform-canonical, so the rule is silent.

If this AC fails, it indicates either (a) M-B's apply step missed some active-tree files (regression in M-B), or (b) the rule's tree-state computation is wrong (regression in AC-1). The combination of AC-1's table-driven fixture tests + AC-5's actual-tree assertion catches both directions.

The same assertion is part of the epic's overall success criteria: "no `entity-id-narrow-width` warnings on this repo's tree post-M-C ship (uniform-canonical active tree)." AC-5 is M-C's discharge of that epic-level criterion.

