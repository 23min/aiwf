---
id: M-083
title: Drift check, normative-doc amendments, and skill content refresh
status: draft
parent: E-23
depends_on:
    - M-081
    - M-082
tdd: required
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

The marketplace cache test (already in place per E-14) compares each fixture against the local marketplace cache and fires if they diverge — skipping cleanly when the cache is absent. M-C's wrap is gated on this test being green.

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
- G-091's preventive check rule for path-form refs — related but separate.
- Any §07 TDD architecture proposal advancement — tracked separately.
- New cycle skills or other rituals additions beyond example refreshes — out of this epic's scope.
