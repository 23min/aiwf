---
id: M-0078
title: Planning-conversation skills design ADR (placement, tiering, name rationale)
status: done
parent: E-0021
tdd: none
acs:
    - id: AC-1
      title: ADR allocated under docs/adr/ and status proposed
      status: met
    - id: AC-2
      title: 'ADR records placement: rituals plugin, not kernel-embedded'
      status: met
    - id: AC-3
      title: ADR records pure-skill-first tiering rule
      status: met
    - id: AC-4
      title: 'ADR records name worked example: aiwfx-whiteboard with rejected alternatives'
      status: met
    - id: AC-5
      title: ADR cross-references M-0074 skills ADR and CLAUDE.md principles
      status: met
---

# M-0078 — Planning-conversation skills design ADR (placement, tiering, name rationale)

## Goal

Capture the design rationale that shapes the rest of E-0021 as a single ADR — *where* planning-conversation skills live (rituals plugin, not kernel), *when* such skills warrant a backing kernel verb (only when usage shows the synthesis re-deriving the same data), and *what* this skill is named with its rejected alternatives. The ADR is the discoverable artefact future planners will hit when they ask the same questions about a future synthesis skill.

## Context

E-0021's epic spec lists three open questions resolved during milestone planning on 2026-05-08: skill name, kernel-vs-plugin placement, and pure-skill-vs-skill+verb tiering. Each is principle-shaped — the answer applies beyond `aiwfx-whiteboard`. M-0074 (under E-0020) sets the precedent that skill-organisation policy belongs in an ADR, not a project-scoped D-NNN; this milestone files the complementary ADR for *placement and tiering* (M-0074's covers *granularity within a topic*). Together the two ADRs define how skills get organised across the kernel/plugin boundary.

The decisions are locked at planning time. This milestone's job is recording, not deciding — the body content is largely transcription of the rationale the operator and assistant walked through. Status remains `proposed` so the ADR can be revised during M-0079 implementation if the act of building the skill surfaces new constraints.

## Acceptance criteria

### AC-1 — ADR allocated under docs/adr/ and status proposed

ADR is allocated via `aiwf add adr --title "<title>"`, lives at `docs/adr/ADR-NNNN-<slug>.md`, frontmatter sets `status: proposed`. Title (refine at allocation): *"Planning-conversation skills: rituals-plugin placement; pure-skill first, kernel verb only if usage demands it"*.

### AC-2 — ADR records placement: rituals plugin, not kernel-embedded

ADR body articulates the principle "planning-conversation skills go in the rituals plugin; kernel-embedded skills are verb wrappers." Cites the existing pattern (`aiwfx-plan-epic`, `aiwfx-plan-milestones`, `aiwfx-start-milestone`, `aiwfx-wrap-epic` are all plugin-side; `aiwf-status`, `aiwf-history`, etc. are kernel-embedded verb wrappers). Notes that `aiwfx-whiteboard` is a planning conversation, not a verb wrapper, so the principle applies.

### AC-3 — ADR records pure-skill-first tiering rule

ADR body articulates the principle "ship a synthesis function as a pure skill first; promote to a skill+verb pair only when usage shows the skill re-deriving the same structured data on every invocation." Names the deferred follow-on (a `landscape`-style verb behind the skill) and documents the trigger condition for filing it (e.g., "skill repeatedly grovels through prose to extract data that should be structured"). Closes E-0021's success criterion #7.

### AC-4 — ADR records name worked example: aiwfx-whiteboard with rejected alternatives

ADR body uses the `aiwfx-whiteboard` naming choice as the worked example demonstrating the placement and tiering rules in action. Records the rejected alternatives (`recommend-sequence`, `landscape`, `paths`, `focus`, `next`, `survey`, `synthesise-open-work`) with one-line rationale per rejection. The "whiteboard" metaphor's fit-rationale (ephemerality, surfacing-not-deciding, operator-at-the-board) is documented; this is the substantive content of the worked-example section.

### AC-5 — ADR cross-references M-0074 skills ADR and CLAUDE.md principles

ADR body explicitly references M-0074's skills-judgment ADR (the "per-verb default; topical multi-verb when concept-shaped" rule) and frames its own scope as complementary, not overlapping — M-0074 covers *granularity within a topic*; this ADR covers *placement and tiering across kernel/plugin*. Cites CLAUDE.md's *"Kernel functionality must be AI-discoverable"* and *"Framework correctness must not depend on the LLM's behavior"* principles as the source authority for the placement reasoning.

## Constraints

- **No code in this milestone.** Pure ADR authorship. `tdd: none` because there is no test surface — the skill itself ships in M-0079.
- **ADR scope is principle-shaped, not implementation-shaped.** Avoid stuffing this ADR with skill-body content (rubrics, output templates, Q&A flow) — that lives in M-0079 in the SKILL.md body. The ADR articulates *why* and *where*; the skill articulates *what* and *how*.
- **Status remains `proposed`** through M-0079. If M-0079's implementation surfaces a constraint that changes the rationale, edit-body the ADR before promoting. Promotion to `accepted` happens at the E-0021 wrap (in M-0080) or in a follow-on milestone if there's no consensus to ratify yet.
- **No invention of unwritten rules.** The ADR records decisions made on 2026-05-08 in the milestone-planning conversation, with the rationale captured at decision time. New analysis or doctrine belongs in a separate, follow-up ADR.

## Design notes

- ADR allocation uses `aiwf add adr --title "..."` — the verb produces one commit with `aiwf-verb: add` and `aiwf-entity: ADR-NNNN` trailers. The body is then filled via `aiwf edit-body` (one further commit).
- The ADR's body sections (refine at authorship): *Context* (what question is being decided, when, why), *Options considered* (kernel-embedded vs rituals plugin; pure-skill vs skill+verb; name candidates), *Decision* (placement = rituals plugin; tiering = pure-skill-first; name = `aiwfx-whiteboard`), *Consequences* (forces all future planning-conversation skills into the plugin; future `landscape` verb is a separate kernel-side artefact when filed).
- The ADR's "worked example" subsection describes how the three rules cascade: rituals-plugin placement → `aiwfx-` prefix → name candidates evaluated against fit/clarity/PM-jargon-avoidance → `aiwfx-whiteboard` selected for ephemerality + collaborative-surface metaphor.
- Cross-reference to M-0074 lives in the ADR's *Related* section; cross-reference to CLAUDE.md kernel principles lives inline in the rationale prose (with section names quoted for grep-ability).

## Surfaces touched

- `docs/adr/ADR-NNNN-*.md` (new — this milestone's primary deliverable)
- No code changes
- No CLAUDE.md changes (M-0074 owns the *Skills policy* section; this ADR is filed alongside without re-editing CLAUDE.md)
- No skill files (M-0079 owns those)

## Out of scope

- The actual `aiwfx-whiteboard` skill body — ships in M-0079.
- A `landscape` kernel verb — deferred follow-on, possibly a future epic; this ADR only documents the trigger condition for filing it.
- Editing CLAUDE.md's *Skills policy* section — M-0074's scope, and this ADR is complementary not overlapping (so no re-edit needed).
- Promotion of this ADR or M-0074's ADR to `accepted` — both stay `proposed` for now; promotion is a separate decision happening at or after epic wrap.

## Dependencies

- E-0020 / M-0074 — the *Skills judgment ADR* this milestone's ADR cross-references. M-0074 must have allocated its ADR (status `proposed` or later) so AC-5 can cite a real ADR-NNNN id rather than a placeholder. If M-0074 hasn't shipped yet, M-0078 must wait — confirmed at start-milestone.
- No other dependencies.

## Coverage notes

- (filled at wrap)

## References

- E-0021 epic spec — open questions table; success criterion #7.
- M-0074's *Skills judgment ADR* — sibling ADR on skill organisation (granularity within a topic). This ADR is its peer covering placement and tiering.
- `docs/pocv3/design/design-decisions.md` — kernel commitments; informs the placement reasoning (skills are advisory; the kernel verb surface is authoritative).
- CLAUDE.md *Engineering principles* §"Kernel functionality must be AI-discoverable" — primary authority for the placement principle.
- CLAUDE.md *Engineering principles* §"Framework's correctness must not depend on the LLM's behavior" — secondary authority; informs the pure-skill-first rule (skills are advisory; the kernel layer below them must remain authoritative).

---

## Work log

### AC-1 — ADR allocated under docs/adr/ and status proposed

ADR-0007 allocated via `aiwf add adr` with title *"Planning-conversation skills: rituals-plugin placement; pure-skill first, kernel verb only if usage demands it"*; frontmatter `status: proposed` · ADR commit 58e7f7a · test `TestADR0007_AC1_AllocationAndStatus` (`internal/policies/adr_0007_test.go`) · test commit 972b5b1.

### AC-2 — ADR records placement: rituals plugin, not kernel-embedded

Body §Placement names the rule, table of existing pattern cites `aiwfx-plan-epic`, `aiwfx-plan-milestones`, `aiwfx-start-milestone`, `aiwfx-wrap-milestone`, `aiwfx-wrap-epic`, `aiwfx-record-decision`, `aiwfx-release` as plugin-side and `aiwf-status`, `aiwf-list`, `aiwf-history`, `aiwf-contract` as kernel-embedded verb wrappers; `aiwfx-whiteboard` flagged as planning conversation, not verb wrapper · ADR commit d2b1b56 · test `TestADR0007_AC2_PlacementClaims` · test commit 972b5b1.

### AC-3 — ADR records pure-skill-first tiering rule

Body §Tiering articulates the rule, names the deferred follow-on (`aiwf whiteboard` kernel verb — verb shares the skill's name to keep the surface unified), enumerates three trigger conditions for promotion, explicitly cites E-0021 success criterion #7 as closed · ADR commit d2b1b56 (refined 834acf2 for verb-name correction) · test `TestADR0007_AC3_TieringRule` · test commit 972b5b1.

### AC-4 — ADR records name worked example: aiwfx-whiteboard with rejected alternatives

Body §Name documents the three-bullet whiteboard fit-rationale (ephemerality, surfacing-not-deciding, operator-at-the-board) and rejects eight alternatives with one-line rationale per (`recommend-sequence`, `landscape`, `paths`, `focus`, `next`, `survey`, `synthesise-open-work`, `critical-path`) · ADR commit d2b1b56 · test `TestADR0007_AC4_NameWorkedExample` · test commit 972b5b1.

### AC-5 — ADR cross-references M-0074 skills ADR and CLAUDE.md principles

Body §Context frames this ADR as complementary to ADR-0006 (M-0074's skills judgment ADR — granularity *within* a topic; this ADR — placement/tiering *across* kernel/plugin); CLAUDE.md *"Kernel functionality must be AI-discoverable"* and *"Framework correctness must not depend on the LLM's behavior"* cited inline and in References · ADR commit d2b1b56 · test `TestADR0007_AC5_CrossReferences` · test commit 972b5b1.

## Decisions made during implementation

- **Deferred kernel verb is `aiwf whiteboard`, not `aiwf landscape`.** Earlier ADR drafts named the deferred follow-on `aiwf landscape` (matching the noun-shaped tier-data). User correction during self-review: the verb shares the skill's name to keep the surface unified across plugin and kernel. Captured in ADR-0007 §Tiering, §Name §rejected `landscape`, and §Consequences. Not D-NNN-worthy — it's a name clarification within an already-deferred verb that isn't filed yet, and the rationale (unified surface) lives in the ADR itself · commit 834acf2.
- **AC-met required mechanical evidence; backfilled tests after first wrap.** First wrap (commit a68dd2d) marked AC-2..AC-5 `met` based on conversation review of ADR-0007 only. User caught the gap: *"everything should be tested, not assumed via conversation."* Reverted M-0078 to `in_progress` (`--force`, principal-authorised) and ACs to `open`; landed `internal/policies/adr_0007_test.go` (test commit 972b5b1) with one test per AC, scoped to the relevant markdown subsection per CLAUDE.md *Testing* §"Substring assertions are not structural assertions"; verified each test red against a temporary ADR mutation, then green after restoration. Re-promoted ACs and milestone with mechanical evidence backing each. Saved as durable feedback memory (`feedback_ac_mechanical_evidence`) and to be codified into CLAUDE.md as a follow-up patch.

## Validation

- **AC-level tests** — `internal/policies/adr_0007_test.go` (commit 972b5b1) carries one `Test*` per AC scoped to the relevant ADR markdown section. All five pass green; each was verified red via a temporary ADR mutation before restoration (see `## Decisions` and the test commit message body for the per-AC mutation cases).
- `aiwf check` — 0 errors, 0 warnings on M-0078 or ADR-0007.
- `go build -o /tmp/aiwf-m078 ./cmd/aiwf` — clean (exit 0).
- `go test -race ./...` — all packages green (exit 0). The new policy tests run as part of the standard `internal/policies/` suite, so any future contributor breaking an AC-2..AC-5 claim in ADR-0007 fails CI without needing to re-derive the discipline.
- `golangci-lint run ./internal/policies/` — 0 issues (gofumpt-clean after one auto-format).
- `wf-doc-lint` (scoped to M-0078 diff) — 0 findings.
- `wf-review-code` (scoped to first-wrap diff) — verdict `approve`, 0 blocking findings, 2 track-for-later items (table is illustrative-not-exhaustive; deferred-verb name will revisit at filing time per CLAUDE.md *Designing a new verb*). The mechanical-evidence gap that motivated the second wrap was *not* surfaced by `wf-review-code` because the skill checks AC coverage by spec text, not by test existence — this is a follow-up improvement worth filing.

## Deferrals

- Promotion of this ADR to `accepted` is deferred to a separate decision after E-0021 closure. Status remains `proposed` through wrap.

## Reviewer notes

- **Existing-pattern table in ADR §Placement is illustrative, not exhaustive.** The table cites 4 kernel-embedded skills (`aiwf-status`, `aiwf-list`, `aiwf-history`, `aiwf-contract`) and 7 plugin skills (`aiwfx-plan-epic`, `aiwfx-plan-milestones`, `aiwfx-start-milestone`, `aiwfx-wrap-milestone`, `aiwfx-wrap-epic`, `aiwfx-record-decision`, `aiwfx-release`). Kernel-side actually has 13 embedded skills (also `aiwf-add`, `aiwf-authorize`, `aiwf-check`, `aiwf-edit-body`, `aiwf-promote`, `aiwf-reallocate`, `aiwf-rename`, `aiwf-render`, `aiwf-retitle`). The narrowing is deliberate — AC-2's spec text named only the 4, and the table's job is to *demonstrate the principle*, not enumerate every skill. The exhaustive list lives in the source tree (`internal/skills/embedded/`).
- **Status remains `proposed`.** Promotion to `accepted` is a separate decision after E-0021 closes (or later); per spec constraint *"Status remains `proposed` through M-0079. If M-0079's implementation surfaces a constraint that changes the rationale, edit-body the ADR before promoting."* The Deferrals section captures this.
- **`tdd: none` was wrong; tests were backfilled.** The original spec set `tdd: none` on the assumption that an ADR-only milestone has no test surface. That was incorrect — the AC test surface is the body content's structural claims, and a Go test under `internal/policies/` can assert each. The first wrap promoted ACs 2-5 to `met` based on conversation review only; the user correction (*"everything should be tested, not assumed via conversation"*) drove a `--force` reversal of M-0078 and a backfill of tests via `wf-tdd-cycle` discipline. The `tdd:` frontmatter value was left as `none` because no kernel verb yet exists to flip it post-creation; the tests exist regardless and run as part of the `internal/policies/` package suite — `tdd:` is the kernel audit's lever, not the test-existence chokepoint.
- **The `aiwf landscape` → `aiwf whiteboard` correction was caught by user review, not by lint or test.** This is appropriate for a doctrinal ADR — the correctness check is human reading. The mechanical chokepoints (`aiwf check`, doc-lint) wouldn't have flagged the divergence because both names are syntactically valid; consistency between the skill's name and the deferred verb's name is a design judgment. The post-correction value (`aiwf whiteboard`) is now pinned by `TestADR0007_AC3_TieringRule`, so a future regression is mechanical, not silent.
- **The `wf-review-code` skill missed the test-existence gap on first wrap.** It approved the diff because all spec-text claims appeared in the ADR; it did not check whether ACs had test backing. Worth filing as a follow-up improvement to that skill (rituals plugin repo) — the AC-promotion chokepoint should require a test reference before promote-to-met.
