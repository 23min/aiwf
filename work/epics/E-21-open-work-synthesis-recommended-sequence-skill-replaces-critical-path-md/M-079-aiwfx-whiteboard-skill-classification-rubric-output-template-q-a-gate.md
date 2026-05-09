---
id: M-079
title: 'aiwfx-whiteboard skill: classification rubric, output template, Q&A gate'
status: done
parent: E-21
depends_on:
    - M-078
tdd: advisory
acs:
    - id: AC-1
      title: Skill scaffolded at aiwfx-whiteboard with frontmatter and SKILL.md
      status: met
      tdd_phase: done
    - id: AC-2
      title: Frontmatter description carries the natural-language query phrasings
      status: met
      tdd_phase: done
    - id: AC-3
      title: Body documents the tier-classification rubric for open-work landscape
      status: met
      tdd_phase: done
    - id: AC-4
      title: 'Body documents output template: landscape, sequence, first decision, pending'
      status: met
      tdd_phase: done
    - id: AC-5
      title: Body documents the Q&A gate flow per CLAUDE.md one-at-a-time convention
      status: met
      tdd_phase: done
    - id: AC-6
      title: 'Body documents anti-patterns: no operator override, no verb invention'
      status: met
      tdd_phase: done
    - id: AC-7
      title: M-074 skill-coverage policy or plugin equivalent accepts the skill
      status: met
      tdd_phase: done
    - id: AC-8
      title: Skill materialised by aiwf init / aiwf update via the rituals plugin
      status: met
      tdd_phase: done
---

# M-079 — aiwfx-whiteboard skill: classification rubric, output template, Q&A gate

## Goal

Ship the `aiwfx-whiteboard` skill — a planning-conversation ritual under `aiwf-extensions/skills/` that loads tree state via existing read verbs (`aiwf status`, `aiwf list`, `aiwf show`, `aiwf history`) and produces a tiered open-work landscape, a recommended sequence, a first-decision fork, and a Q&A-gated walk-through of pending decisions. After this milestone, an operator can ask *"what should I work on next?"* in any Claude Code session opened against this repo and receive structurally consistent direction synthesis without scrolling back through prior planning conversations.

## Context

`aiwf status` and `aiwf render roadmap` render *state* (what entities exist, what their statuses are). Neither renders *direction* (what to do next, in what order, with which dependencies foregrounded). E-21's operator-name-the-gap quote crystallises the missing affordance: *"It's very difficult to keep sequence in my head when there are so many ADRs, epics, milestones, gaps."* The synthesis pattern that produced `work/epics/critical-path.md` on 2026-05-08 is the template; this milestone graduates it into a reproducible skill body.

M-078 (preceding milestone) records the rationale for *where* this skill lives (rituals plugin) and *why* it stays pure-skill (no kernel verb until usage justifies). M-079 is the implementation — body content, classification rubric, output template, Q&A flow. M-080 (following milestone) validates the result against `critical-path.md` as a fixture and retires the holding doc.

The skill body is content, not code, so this milestone is `tdd: advisory`. The substantive testable surface is the structural-agreement check in M-080 — whether running the skill on the live tree produces output close to the existing critical-path.md. Section-presence drift-prevention tests are encouraged but not required at AC level.

## Acceptance criteria

### AC-1 — Skill scaffolded at aiwfx-whiteboard with frontmatter and SKILL.md

Skill lives at `aiwf-extensions/skills/aiwfx-whiteboard/SKILL.md` (path verified against the existing `aiwfx-plan-epic`/`aiwfx-plan-milestones` layout). Frontmatter declares `name: aiwfx-whiteboard` matching the directory; description is non-empty; the file is a single SKILL.md (no template subdirs are required for v1, since the synthesis output is templated in the body, not in side files).

### AC-2 — Frontmatter description carries the natural-language query phrasings

Description text contains, at minimum, these query phrasings (each in quotes or backticks so AI description-match retrievers can lift them as-is):
- *"what should I work on next?"*
- *"give me the landscape"*
- *"where should we focus?"*
- *"what's the critical path?"*
- *"synthesise the open work"*
- *"draw the whiteboard"* (or equivalent metaphor-anchored phrasing)

Total of at least five phrasings. The description is dense by design — Claude Code routes by description-match, and the skill's name (`whiteboard`) is metaphor-shaped not query-shaped, so the description does the routing work.

### AC-3 — Body documents the tier-classification rubric for open-work landscape

Body contains a *Tier classification rubric* section that names each tier explicitly, gives the leverage-on-future-work criterion that places an item in that tier, and gives examples drawn from `critical-path.md`'s actual tier placements. At minimum, five tiers (Tier 1 = compounding fixes, Tier 2 = architecturally foundational, Tier 3 = workflow rituals, Tier 4 = operational debris, Tier 5 = defer). Rubric is reproducible (criterion-based) but acknowledges LLM judgement at the margins (the placement of a borderline item is allowed to vary; the criteria themselves do not).

### AC-4 — Body documents output template: landscape, sequence, first decision, pending

Body contains an *Output template* section specifying the sections the skill emits: (a) tiered landscape table — one row per open item with kind, cost-estimate, what-it-unblocks columns; (b) recommended sequence prose with numbered ordering and explicit "before/after/parallel" framing; (c) first-decision fork — concrete options A/B/C with pros/cons/lean; (d) pending-decisions list — open Q&A items the operator may walk through. Template structure is fixed across runs; content is judgement-driven.

### AC-5 — Body documents the Q&A gate flow per CLAUDE.md one-at-a-time convention

Body contains a *Q&A gate* section. The gate fires after the recommendation is rendered. Gate text: *"Walk through the pending decisions one at a time, or is the recommendation enough?"*. When the operator opts in, the skill walks one decision at a time per CLAUDE.md *Working with the user* §Q&A format — context, options with pros/cons, lean, numbered options, wait for choice. When the operator declines, the skill exits cleanly with a one-line summary.

### AC-6 — Body documents anti-patterns: no operator override, no verb invention

Body contains an *Anti-patterns* section. Lists at minimum: (1) the skill does not replace the operator's judgement — it surfaces and gates; (2) the skill does not invent verbs that don't exist on the kernel surface (per E-21 epic constraint *"no verb invention"*); (3) the skill does not persist its output to a file (on-demand re-derivation is the contract; persisted artefacts go stale within hours); (4) scope is locked to direction-synthesis — adjacent functions ("should I refactor X?", "is this design good?") prompt "should this be its own skill?" not silent extension.

### AC-7 — M-074 skill-coverage policy or plugin equivalent accepts the skill

Run M-074's `internal/policies/skill_coverage.go` policy (or whichever scope-extension covers plugin-side skills) against the new skill and verify zero violations. If M-074's policy is kernel-only and does not yet cover plugin skills, this AC is satisfied by adding a one-line note in the milestone work log explaining the gap and (optionally) filing a follow-up gap rather than expanding M-074's scope from this milestone.

### AC-8 — Skill materialised by aiwf init / aiwf update via the rituals plugin

After installing/updating the rituals plugin in the consumer repo, `aiwf doctor` reports the skill as present (or whatever the equivalent verification surface is for plugin-installed skills). The plugin's marketplace metadata or registration list (whatever points consumers at the new skill) is updated as needed. This AC verifies the distribution path, not just the source file.

## Constraints

- **Pure-skill, no kernel verb.** Per M-078's ADR. No new kernel code; no new verbs. The skill calls only existing read verbs: `aiwf status`, `aiwf list`, `aiwf show`, `aiwf history`. If a verb the skill body would benefit from doesn't exist, file a follow-up gap rather than encoding a hand-edit workaround.
- **Read-only over the planning tree.** No mutations. The Q&A walk-through can suggest mutations to the operator (*"want me to file a gap for that?"*) but does not perform them as a side effect of the skill.
- **One-at-a-time Q&A is non-negotiable.** Per CLAUDE.md *Working with the user* §Q&A format. Batched-question rendering breaks the user's documented preference and the epic's stated mitigation against authoritative-but-brittle output.
- **No persisted artefact.** The skill's output goes to the conversation, not to disk. `critical-path.md` is being deleted in M-080 precisely to remove the persisted-artefact anti-pattern; this milestone must not reintroduce it.
- **Output template is consistent across runs.** Same tree state → structurally identical output. Tier *contents* and *lean* may vary with LLM judgement; tier set, section order, and column headers do not.
- **Description-match routing assumed.** Frontmatter description densely covers query phrasings (AC-2). Skill must work as the destination of natural-language queries even when the user does not type the skill name.

## Design notes

- Skill scaffold layout (refine at start-milestone): single `SKILL.md` under `aiwf-extensions/skills/aiwfx-whiteboard/`. No `templates/` subdirectory in v1 — output template is documented inline in the body, in the *Output template* section. If iteration shows the template benefits from being a separate file (e.g., for reuse across the deferred verb-backed v2), the split lands in a follow-up milestone.
- Frontmatter shape (refine at authorship):
  ```yaml
  name: aiwfx-whiteboard
  description: |
    Use to answer direction-synthesis questions like "what should I work on next?",
    "give me the landscape", "where should we focus?", "synthesise the open work",
    "draw the whiteboard", "what's the critical path?". Loads tree state via
    aiwf status / aiwf list / aiwf show / aiwf history; produces a tiered open-work
    landscape, a recommended sequence, a first-decision fork, and an optional Q&A
    gate over pending decisions. Read-only; no commit.
  ---
  ```
- Body section order (refine at authorship): *What it does* → *When to use* → *Inputs* (which verbs to call, in what order) → *Tier classification rubric* (AC-3) → *Output template* (AC-4) → *Q&A gate* (AC-5) → *Anti-patterns* (AC-6) → *Examples* (one walkthrough drawn from critical-path.md's content) → *References*.
- Inputs section names the read verbs: `aiwf status` for the in-flight summary, `aiwf list <kind> --status <s>` for per-kind enumeration (assumes E-20/M-072 has shipped), `aiwf show <id>` for detail when a referenced item warrants surfacing, `aiwf history <id>` for context on items whose recent activity matters.
- Examples section is one full walk-through using the current planning tree's actual content. The walk-through serves three purposes: (a) demonstrates the output template, (b) seeds the M-080 fixture validation, (c) gives the LLM a worked example to anchor on across runs.
- Q&A gate text (refine at authorship): *"Walk through the pending decisions one at a time, or is the recommendation enough?"* with explicit options 1/2/3 (Q&A walk / one-line summary / further follow-up the operator names).

## Surfaces touched

- `aiwf-extensions/skills/aiwfx-whiteboard/SKILL.md` (new — primary deliverable)
- Possibly `aiwf-extensions/marketplace.json` or equivalent registration surface (verify path at start-milestone)
- No kernel changes (`internal/`, `cmd/aiwf/` untouched)
- No CLAUDE.md edit (M-078's ADR is filed without re-editing CLAUDE.md; this milestone follows the same discipline)

## Out of scope

- Fixture validation against `critical-path.md` — happens in M-080.
- Deletion of `work/epics/critical-path.md` — happens in M-080.
- A `landscape` kernel verb — deferred per M-078's ADR.
- Constraint-aware re-prioritisation (*"we only have 2 days; what changes?"*) — future iteration of the skill.
- Incorporating external signals (calendar, deadlines) — explicitly out of scope per E-21.
- Migrating other planning rituals into the whiteboard metaphor — those are their own skills, unaffected by this milestone.
- Promoting M-078's ADR to `accepted` — separate decision after epic wrap.

## Dependencies

- **M-078** — design ADR must exist (status `proposed` or later) so this milestone's body can cite the placement and tiering rationale by ADR id.
- **E-20 / M-073** — `aiwf-list` skill should exist by now so this skill's *Inputs* section can reference `aiwf list <kind>` without that being a dangling reference. If M-073 hasn't shipped, the *Inputs* section uses the read verbs available at the time and notes the upgrade.
- **E-20 / M-074** — skill-coverage policy must exist so AC-7 has something to run against. If the policy is kernel-scope only, AC-7 is satisfied with a noted gap rather than scope-creeping into this milestone.
- **`aiwf-extensions` rituals plugin** — must be installed in the consumer repo for AC-8 to verify materialisation. The CLAUDE.md *Operator setup* section already requires it.

## Coverage notes

- (filled at wrap)

## References

- E-21 epic spec — scope, constraints, success criteria.
- M-078 — sibling milestone; the design ADR this milestone's skill body cites.
- M-080 — successor milestone; consumes this skill's output as a fixture.
- `work/epics/critical-path.md` — content the skill body's *Examples* section draws from; deleted in M-080.
- `aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md`, `aiwfx-plan-milestones/SKILL.md`, `aiwfx-start-milestone/SKILL.md` — sibling planning rituals; conventions for skill body shape and frontmatter style.
- `internal/skills/embedded/aiwf-status/SKILL.md` — kernel-side sibling. Same job-shape (a one-screen synthesis); different layer (state, not direction).
- CLAUDE.md *Working with the user* §Q&A format — the convention AC-5 honours.
- CLAUDE.md *Engineering principles* §"Kernel functionality must be AI-discoverable" — primary authority for AC-2's dense-description requirement.

---

## Work log

### AC-1 — Skill scaffolded at aiwfx-whiteboard with frontmatter and SKILL.md

Fixture authored at `internal/policies/testdata/aiwfx-whiteboard/SKILL.md` per CLAUDE.md §"Cross-repo plugin testing" — the canonical authoring location during the milestone. Frontmatter declares `name: aiwfx-whiteboard` and a non-empty description. Test: `TestAiwfxWhiteboard_AC1_SkillScaffolded` (`internal/policies/aiwfx_whiteboard_test.go`) · commit 15afafa.

### AC-2 — Frontmatter description carries the natural-language query phrasings

Description embeds all six spec-listed phrasings (*"what should I work on next?"*, *"give me the landscape"*, *"where should we focus?"*, *"what's the critical path?"*, *"synthesise the open work"*, *"draw the whiteboard"*) plus framing about the read verbs the skill calls. Test: `TestAiwfxWhiteboard_AC2_DescriptionPhrasings` (asserts ≥5 of the six phrasings appear case-insensitively) · commit 41e19af.

### AC-3 — Body documents the tier-classification rubric for open-work landscape

§Tier classification rubric carries the five tiers (Tier 1 compounding · Tier 2 foundational · Tier 3 ritual · Tier 4 debris · Tier 5 defer), each with a leverage-on-future-work criterion plus three exemplar ids drawn from the historical `critical-path.md` placements. Source citation removed in commit 07def54 per user direction (the document name is incidental and is being retired in M-080); exemplar ids preserved. Test: `TestAiwfxWhiteboard_AC3_TierRubric` (asserts five tiers, descriptive keywords, and one exemplar per tier) · commits 0591372 (initial) + 07def54 (cleanup).

### AC-4 — Body documents output template: landscape, sequence, first decision, pending

§Output template names the four blocks the skill emits — tiered landscape table, recommended sequence, first-decision fork, pending-decisions list — with explicit columns (Item / Kind / Cost / What it unblocks), ordering frame (before / after / parallel), and fork shape (A/B/C with pros/cons/lean). Test: `TestAiwfxWhiteboard_AC4_OutputTemplate` (asserts named blocks + columns + ordering frame) · commit 0591372.

### AC-5 — Body documents the Q&A gate flow per CLAUDE.md one-at-a-time convention

§Q&A gate carries the canonical gate prompt verbatim and three operator paths (walk one-at-a-time / declare-enough / name-different-followup) per CLAUDE.md *Working with the user* §Q&A format. Test: `TestAiwfxWhiteboard_AC5_QAGate` (asserts gate text, one-at-a-time framing, CLAUDE.md cross-reference) · commit 0591372.

### AC-6 — Body documents anti-patterns: no operator override, no verb invention

§Anti-patterns lists the four spec-named failure modes (replacing operator judgement / inventing verbs / persisting synthesis / scope creep), each with rationale. Test: `TestAiwfxWhiteboard_AC6_AntiPatterns` · commit 0591372. **Note:** anti-pattern #3 is intentionally narrowed in commit 07def54 from "no synthesis snapshot of any kind" to "no checked-in synthesis snapshot" — a follow-up patch (see *Deferrals*) will revisit this with a gitignored-cache option per the user's design pivot.

### AC-7 — M-074 skill-coverage policy or plugin equivalent accepts the skill

Spec's escape valve applied: the kernel's `PolicySkillCoverageMatchesVerbs` (M-074) walks `internal/skills/embedded/` only and does not cover plugin paths. The equivalent invariants (name matches dir, `aiwfx-<topic>` convention, description non-empty, every backticked `aiwf <verb>` mention resolves to a real top-level Cobra verb) are re-applied to the fixture in `TestAiwfxWhiteboard_AC7_SkillCoveragePolicyEquivalent`. Red verified by mutating the fixture body to include `aiwf bogus-verb`; restored byte-for-byte. Follow-up gap **G-088** files the kernel-side scope expansion · commit 69e3bd1.

### AC-8 — Skill materialised by aiwf init / aiwf update via the rituals plugin

Deploy step performed in the rituals repo (`/Users/peterbru/Projects/ai-workflow-rituals/`): feature branch authored, merged `--no-ff` to `main`, pushed (commit 9646984 + post-cleanup commit 333a033). After `/plugin update aiwf-extensions@ai-workflow-rituals` + `/reload-plugins`, the skill appears in the host's available-skills list and the marketplace cache contains the SKILL.md byte-for-byte matching the fixture. Test: `TestAiwfxWhiteboard_AC8_MaterialisationDriftCheck` (skip-if-no-cache; FAIL on missing or drifting skill) · commit 69e3bd1.

## Decisions made during implementation

- **Fixture-first authoring per the new cross-repo doctrine.** Established in CLAUDE.md §"Cross-repo plugin testing" mid-milestone (commit 31c7b43 on the epic branch). The fixture in this repo is the canonical authoring location during the milestone; the rituals-repo SKILL.md is the deploy target. This is the pattern future plugin-skill milestones should follow.
- **Verb name unification: `aiwf whiteboard`, not `aiwf landscape`.** Picked up from M-078's late-cycle correction; the deferred kernel verb's working name follows the skill name to keep the surface unified across plugin and kernel.
- **Anti-pattern #3 narrowing + WHITEBOARD.md gitignored cache as follow-up patch.** The original "no persisted artefact" anti-pattern was over-restrictive (counter-example: `STATUS.md` is a hook-regenerated persisted artefact and is not a problem). The user's design pivot — a gitignored `WHITEBOARD.md` regenerated on each invocation — does not violate the genuine failure mode (checked-in stale snapshot becomes a second source of truth). M-079's anti-pattern was tightened to *"no checked-in synthesis snapshot"* in commit 07def54; the full revision (skill writes WHITEBOARD.md, .gitignore entry, AC test) lands as a follow-up `wf-patch` rather than re-opening M-079 a second time.
- **AC-7 satisfied via test-side equivalent rather than kernel-policy expansion.** The spec's escape valve was used: invariants are mechanically asserted but the kernel policy itself is unchanged. Follow-up gap G-088 tracks the kernel-side scope expansion.

## Validation

- **AC-level tests** — `internal/policies/aiwfx_whiteboard_test.go` (commits 15afafa, 41e19af, 0591372, 69e3bd1, 07def54) carries one `Test*` per AC. Plus `TestFrontmatterField_BranchCoverage` exercises the new helper's reachable branches per CLAUDE.md *Testing* §"Branch-coverage audit". All eight AC tests pass green.
- **Red-evidence per AC** — captured in commit messages: AC-1 (fixture absent), AC-2 (placeholder description had 0 of 6 phrasings), AC-3/4/5/6 (sections absent), AC-7 (mutation `aiwf bogus-verb` injected), AC-8 (cache pre-deploy → post-deploy → drift after fixture cleanup → re-deploy).
- `aiwf check` — 0 errors, 0 warnings on M-079 entities or the deployed skill (the live tree carries 1 standing `unexpected-tree-file` warning for `work/epics/critical-path.md`, retired in M-080; unrelated to this milestone).
- `go build -o /tmp/aiwf ./cmd/aiwf` — clean.
- `go test -race ./...` — all packages green (exit 0).
- `golangci-lint run ./internal/policies/` — 0 issues.
- `wf-doc-lint` — scope empty (M-079 did not touch `docs/`).
- `wf-review-code` — verdict `approve`, 0 blocking findings, 4 track-for-later items (missing body sections per spec design notes; AC-8 branches not unit-tested; `frontmatterField` block-scalar limitation; phase-walk commit volume).
- **Cross-repo deploy verified end-to-end:** rituals repo `main` carries the skill at commit 333a033; marketplace cache holds the matching SKILL.md; the skill appears in this session's available-skills list as `aiwf-extensions:aiwfx-whiteboard`.

## Deferrals

- **Follow-up `wf-patch`: WHITEBOARD.md gitignored local cache.** The skill should write `WHITEBOARD.md` to repo root after producing output; `.gitignore` carries the entry; SKILL.md anti-pattern #3 is revised from absolute "no synthesis snapshot of any kind" to "no checked-in synthesis snapshot, gitignored cache OK". Spec for M-079 set `tdd: advisory` and the AC-6 text explicitly forbade the file — a re-open + re-walk is heavier than the patch is worth, so this rolls forward. **Tracked as gap G-089** (filed at wrap).
- **Body sections per spec design notes** (When to use, Inputs, Examples, References) — none are AC-required; their absence is non-blocking. After discussion, the *Examples* section was actively rejected (the fixture-as-skill is the example; live invocations are the examples; baking a snapshot in is the anti-pattern E-21 exists to retire). The other three sections may land as part of the WHITEBOARD.md follow-up patch or remain unwritten — operator decides at that time.
- **AC-8 branch-coverage hardening** — three branches in the AC-8 test (cache-root absent skip, plugin dir read error skip, fixture/cache drift FAIL) are not unit-tested. The success path and the "skill missing" path were exercised end-to-end by the deploy cycle; the rest are runtime-conditional on home-directory state. **Tracked as gap G-090** (filed at wrap).

## Reviewer notes

- **The fixture *is* the skill body.** During M-079 I initially flagged the absence of an *Examples* section as concerning. After discussion with the user, the right reading is: the fixture is the deployed skill; live invocations produce the examples; a static body-side example would either be a synthetic restatement of §Output template (cosmetic) or a stale snapshot of the live tree (the very anti-pattern this skill exists to retire). M-080's validation captures live transcripts, not body-side static content. Decision recorded above.
- **Phase-walk commit volume.** 24 of the milestone branch's 38 commits are `aiwf promote --phase` walks (3 phases × 8 ACs) added at the very end to clear `acs-tdd-audit` warnings on a `tdd: advisory` milestone. This is honest about the discipline I retroactively applied (tests existed; phase tracking did not happen at AC-promote moment). Future cycles should advance phase at promote-time to keep history tighter.
- **Cross-repo deploy ergonomics.** The deploy step (push rituals branch → merge to rituals main → push main → `/plugin update` → `/reload-plugins`) is a five-step manual cycle each time the fixture changes. M-079 went through this twice (initial deploy + critical-path.md cleanup). For high-iteration milestones, this would become friction; deferring per gap G-090's adjacent direction (CI-side drift check that doesn't require local plugin install) might combine well with a deploy-helper script.
- **`frontmatterField` helper has a v1 limitation:** it parses single-line `description:` only. If a future plugin skill uses block-scalar (`description: |`), my AC-1/AC-2 tests would silently misread it. The kernel-side `parseSkillMarkdown` in `skill_coverage.go` handles both forms; this could be reused. Track-for-later from `wf-review-code`; not blocking M-079.
- **Critical-path.md is intentionally still in the tree.** Its retirement is M-080's act, not M-079's. Removing the document while M-079 still cited it would have left M-079 with broken cross-references. The cleanup commit (07def54) removed the citations from the SKILL.md so M-080 can delete the document without affecting deployed plugin content.
