---
id: M-0083
title: Drift check, normative-doc amendments, and skill content refresh
status: done
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

## Work log

Phase timeline lives in `aiwf history M-0083/AC-N` for every AC; the entries below are the post-cycle outcome and the kernel `met` commit SHA. The production-code diff for the in-repo ACs (AC-1, AC-3, AC-5 implementations + AC-2 amendments + AC-4 kernel-fixture sweep) is bundled in this milestone's wrap commit. The rituals-side AC-4 commit lives in the rituals plugin repo at `808ad70bb368c7d687a207cc7b749e0b11529323`.

### AC-1 — Drift-check rule `entity-id-narrow-width` with tree-state detection

`internal/check/entity_id_narrow_width.go` introduces the warning rule with tree-state classification: silent on uniform-narrow / uniform-canonical / empty trees; fires once per narrow active file when mixed. Archive entries (`<kind>/archive/...`) excluded from the active-tree state computation. ADR is exempt from classification because its grammar (`ADR-\d{4,}`) was always 4-digit canonical — including it would taint pre-migration trees as "mixed." Wired into `check.Run`; hint added to `internal/check/hint.go`; rule mentioned in the embedded `aiwf-check` SKILL.md so it's AI-discoverable. Kernel met commit: `5c9405f`. Tests in `internal/check/entity_id_narrow_width_test.go` cover empty/uniform-narrow/uniform-canonical/mixed trees plus archive-only narrow + active-mixed + narrow-archive cases.

### AC-2 — Normative docs amended for canonical 4-digit policy

`docs/adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md` §"Id and storage" amended `F-NNN → F-NNNN`, family references updated to `G-NNNN`/`D-NNNN`/`C-NNNN`, ADR-0008 cross-reference added. `CLAUDE.md` "What aiwf commits to" §2 collapsed from per-kind width list to a single uniform rule citing canonical 4-digit width, parser tolerance for narrow legacy widths, and `aiwf rewidth` for migration. Kernel met commit: `9ed80aa`. Tests in `internal/policies/m083_test.go` use markdown-section walking (`extractMarkdownSection`) for structural assertions per CLAUDE.md "substring assertions are not structural assertions" — assertions scoped to named section headings, not flat over the file.

### AC-3 — Doc-tree narrow-id sweep complete

9 documents under `docs/pocv3/design/` and `docs/pocv3/plans/` swept for ~70 narrow-id mentions including markdown-link path-form refs. Code-fence content, inline backtick spans, and the explicit allowlist of historical/illustrative references preserved. Kernel met commit: `7fa17b9`. Mechanical chokepoint in `internal/policies/m083_doc_sweep_test.go`: `TestPolicy_DocTreeNarrowIDsCanonicalized` greps for narrow-id patterns under `docs/`, `README.md`, `CHANGELOG.md` (excluding archive paths) and fails on any non-allowlisted match. The 16-entry allowlist covers foreign-project surveys (FlowTime, Liminara), illustrative mining docs, the hypothetical worked-example proposal `07-tdd-architecture-proposal.md`, and `CHANGELOG.md` (release notes are historical record per the spec).

### AC-4 — Skill content refreshed in kernel and rituals plugin

Two-part deliverable per CLAUDE.md "Cross-repo plugin testing":

1. **Kernel-embedded fixture** at `internal/policies/testdata/aiwfx-whiteboard/SKILL.md` — 13 narrow-id mentions canonicalized to 4-digit form (G-071→G-0071, E-20→E-0020, F-NNN→F-NNNN, E-NN/M-NNN placeholders → E-NNNN/M-NNNN). Manual sweep using a small Python regex script.

2. **Rituals plugin** — 5 files refreshed and committed at `808ad70bb368c7d687a207cc7b749e0b11529323` in `https://github.com/23min/ai-workflow-rituals`:
   - `plugins/aiwf-extensions/templates/epic-spec.md` (1 ref)
   - `plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md` (5 refs)
   - `plugins/aiwf-extensions/skills/aiwfx-whiteboard/SKILL.md` (13 refs)
   - `plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md` (1 ref)
   - `plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md` (1 ref)

Kernel met commit: `4b6aadc`. The marketplace cache was reloaded and the active install bumped to the new SHA in `~/.claude/plugins/installed_plugins.json`. The `TestAiwfxWhiteboard_AC8_MaterialisationDriftCheck` test (added in M-0080) compares the kernel fixture against the active install and is now green. One width-fragile substring assertion in `aiwfx_whiteboard_test.go::TestAiwfxWhiteboard_AC3_TierRubric` updated to use the `containsIDForm` helper (same width-tolerant pattern as M-0082's m080_test.go refactor).

### AC-5 — Active-tree drift check green on this repo post-M-0083

`internal/policies/this_repo_drift_check_clean_test.go::TestPolicy_ThisRepoDriftCheckClean` loads this repo's tree via `tree.Load`, runs `check.Run`, and asserts no finding carries the code `entity-id-narrow-width`. Standalone `aiwf check` confirms: 0 errors, 1 unrelated warning (`provenance-untrailered-scope-undefined`). The active tree is uniform-canonical post-M-0082's rewidth apply, so the new rule is silent. Kernel met commit: `a92e960`.

## Decisions made during implementation

- **ADR exempt from mixed-state classification.** ADR's grammar was always 4-digit canonical (`ADR-\d{4,}`); including it in the rule's classification would taint pre-migration trees (E-01 + ADR-0001) as "mixed," which is not the signal ADR-0008 calls for. The rule operates only on kinds with a narrow legacy form (E, M, G, D, C, F). Documented in code comments and tested explicitly in `internal/check/entity_id_narrow_width_test.go`.

- **Doc-sweep allowlist breadth.** 16 doc paths went into the allowlist for foreign / illustrative / historical reasons. `CHANGELOG.md` was treated as historical record per the spec's "release-note rule" — entries describing v0.1.0's narrow-width origin keep narrow form. `docs/explorations/07-tdd-architecture-proposal.md`'s E-12 / M-042..M-052 are hypothetical worked-example IDs in a proposal doc, not real entities; allowlisted to preserve the worked-example shape.

- **Width-fragile substring assertion in `aiwfx_whiteboard_test.go::TestAiwfxWhiteboard_AC3_TierRubric` migrated to `containsIDForm`.** Same theme as M-0082's m080_test.go refactor. The exemplar list in the test stays at narrow form (`G-071`, `G-059`, etc.) as the canonical query identifier — the helper matches either rendering. Documented in a comment near the call site.

- **Programmatic active-install bump in `~/.claude/plugins/installed_plugins.json`.** After committing + pushing the rituals plugin and reloading plugins (which pulls the new SHA into cache but leaves the active install pointing at the previous SHA), the JSON was edited to point at `808ad70bb368`. This is operator-side state mutation but mirrors what the `/plugin` menu's update flow does — the cleanest path to a green drift test in the local development environment without waiting on an interactive update.

No ADRs filed mid-implementation. ADR-0008 was the policy precedent for the entire epic; ADR-0003 was amended (per AC-2) but its status stayed `proposed` — only its body content was updated.

## Validation

- `go build -o /tmp/aiwf ./cmd/aiwf` — clean.
- `go test -race ./...` — 25 packages, 0 failures.
- `golangci-lint run` — 0 issues.
- `aiwf doctor --self-check` — 30/30 steps.
- `aiwf check` on this repo's tree — 0 errors, 1 unrelated warning (`provenance-untrailered-scope-undefined`, no upstream configured).
- `aiwf show M-0083` — all 5 ACs `met` with `phase: done`; no findings.
- `TestAiwfxWhiteboard_AC8_MaterialisationDriftCheck` (the cross-repo materialisation drift test) — green after marketplace reload + active-install bump.
- Coverage on the new code: `entityIDNarrowWidth` 95.8% (one defensive `IDFromPath` fall-through marked `//coverage:ignore` with rationale); `isArchivePath` 100%; `isNarrowID` 100%.

**Cross-repo SHA recorded:** rituals plugin commit `808ad70bb368c7d687a207cc7b749e0b11529323` on `main` at `https://github.com/23min/ai-workflow-rituals` is the canonical reference. The epic's overall success criterion ("All 5 rituals-plugin files refresh; SHA recorded in M-C's wrap") is satisfied by this entry.

## Deferrals

None for M-0083 itself. The epic's ADR-0008 policy is now fully implemented across all three milestones (M-0081, M-0082, M-0083).

## Reviewer notes

- **The drift-check rule is sequenced last for a reason.** ADR-0008 specifies tree-state detection: silent on uniform tree (any width), fires only on mixed state. With M-0082's rewidth already applied to this repo, AC-5 becomes a verification (the rule is silent on the post-rewidth canonical tree). On a fresh consumer repo post-upgrade, the rule stays silent until the consumer either allocates a canonical entity (mixing the tree → warning fires) or runs `aiwf rewidth --apply` (uniformly canonicalizing → silent). That asymmetry is the on-demand framing.

- **The `installed_plugins.json` programmatic bump is a developer-environment concession.** The kernel's drift test is part of the cross-repo plugin testing convention from CLAUDE.md, and it works correctly when the active plugin install matches the marketplace's current state. The interactive `/plugin` menu's update flow is the user-facing path; the JSON edit here is the equivalent. CI without the plugin install skips the test cleanly (the test's `t.Skipf` branch handles missing manifests).

- **One width-fragile assertion remained in `aiwfx_whiteboard_test.go`.** The `TestAiwfxWhiteboard_AC3_TierRubric` test was substring-matching `"G-071"` etc. against the skill body. After the AC-4 sweep, the skill body cites canonical `"G-0071"`, breaking the substring match. Fixed by switching to `containsIDForm` (the regex-based width-tolerant matcher introduced in M-0082's m080_test.go refactor). If a third such assertion surfaces, promote `containsIDForm` to a shared helper rather than copy-pasting again.

- **ADR-0008 still references `internal/verb/import.go::canonicalPadFor`** at lines 8, 38, 117 — the function deleted in M-0081's AC-1. Flagged in M-0081 wrap as deliberately preserved historical commentary. Same posture here: ADR bodies describe the state at authoring time; reviewers can decide whether to add a post-migration footnote pointing at `entity.CanonicalPad`.

- **15 mechanical aiwf state-transition commits** sit ahead of the wrap commit (red→green→done→met for each of 5 ACs). They modify only the milestone spec's frontmatter and STATUS.md. The production-code diff is bundled in this wrap commit.

- **Epic E-0023 is now ready to wrap.** This was the third and final milestone (M-0081 done, M-0082 done, M-0083 done). After this wrap, invoke `aiwfx-wrap-epic E-0023` to scaffold the wrap artefact, harvest ADR candidates, and merge the epic branch into mainline.

