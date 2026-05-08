---
id: M-066
title: aiwf check finding entity-body-empty
status: done
parent: E-17
tdd: required
acs:
    - id: AC-1
      title: entity-body-empty (warning) when body section is empty
      status: met
      tdd_phase: done
    - id: AC-2
      title: Severity escalates to error under aiwf.yaml tdd.strict true
      status: met
      tdd_phase: done
    - id: AC-3
      title: Entities with non-empty body prose produce no finding
      status: met
      tdd_phase: done
    - id: AC-4
      title: Bare HTML comments do not satisfy the non-empty requirement
      status: met
      tdd_phase: done
    - id: AC-5
      title: Finding does not retroactively engage acs-tdd-audit
      status: met
      tdd_phase: done
    - id: AC-6
      title: Finding code documented in aiwf-check skill
      status: met
      tdd_phase: done
---

## Rescope note (per G-063, 2026-05-07)

This milestone was originally scoped AC-only as `acs-body-empty`. **Rescoped 2026-05-07** to a kind-generalized finding `entity-body-empty` covering all entity kinds whose body carries load-bearing prose. The rescope was forced by [G-063](../../gaps/G-063-no-defined-start-epic-ritual-epic-activation-is-a-deliberate-sovereign-act-with-preflight-optional-delegation-but-kernel-treats-it-as-a-one-line-fsm-flip.md): the start-epic preflight requires a "non-empty epic body" check, and the cleanest implementation is one rule parameterized by kind rather than two parallel rules. Sub-decision #4 of G-063 governs.

Title, slug, and per-AC titles have all been updated to reflect the generalized scope. The frontmatter title fields were hand-edited (operator-authorized; no `aiwf retitle` verb exists yet — see [G-065](../../gaps/G-065-no-aiwf-retitle-verb-scope-refactors-that-change-an-entity-s-or-ac-s-intent-leave-frontmatter-title-fields-permanently-misleading-only-slug-rename-is-supported.md) for the verb-mechanism gap that this rescope surfaced).

## Goal

Add an `aiwf check` finding `entity-body-empty` that fires for any entity whose load-bearing body section is empty (no non-heading content between the section heading and the next, or EOF; HTML comments do not satisfy non-empty). Warning severity by default; error under `aiwf.yaml: tdd.strict: true` (sharing the same strictness field as [M-065](../E-16-tdd-policy-declaration-chokepoint-closes-g-055/M-065-aiwf-check-finding-milestone-tdd-undeclared-as-defense-in-depth.md)'s `milestone-tdd-undeclared`). This is the load-bearing chokepoint that makes per-kind body-prose intent mechanically enforceable.

## Approach

New rule in `internal/check/`. Per-kind dispatch: each entity kind has a hardcoded list of load-bearing body sections; the rule walks the body, locates each named section by heading, and asserts non-empty content between that heading and the next.

| Kind | Required non-empty body sections |
|---|---|
| epic | `Goal`, `Scope`, `Out of scope` |
| milestone | `Goal`, `Approach`, `Acceptance criteria` |
| AC (sub-element of milestone) | `### AC-N — <title>` body |
| gap | `What's missing`, `Why it matters` |
| adr | `Context`, `Decision`, `Consequences` |
| decision | `Question`, `Decision`, `Reasoning` |
| contract | `Purpose`, `Stability` |

Definition of empty: between the section's heading and the next heading (or EOF), there is no non-whitespace content other than the heading itself. Multiple consecutive blank lines, leading/trailing whitespace, and Windows line endings all count as empty. HTML comments are stripped before the emptiness check (operator intent to defer is not the prose the design specifies).

For ACs, the rule shares the heading-locator from the existing `acs-body-coherence` rule rather than re-parsing the markdown. For top-level kinds, a similar locator scans the body's `## ` headings.

Severity is resolved from `aiwf.yaml: tdd.strict` — the same field that gates M-065's escalation. Single source of truth: both `entity-body-empty` and `milestone-tdd-undeclared` read it; no parallel field, no second config knob.

The grandfather rule is preserved by *not* coupling this to `acs-tdd-audit`: historical entities with empty bodies surface as `entity-body-empty` warnings (so they're visible) but do not retroactively flunk other audits.

## Acceptance criteria

### AC-1 — entity-body-empty (warning) when body section is empty

`aiwf check` against a planning tree containing an entity with at least one empty load-bearing body section emits an `entity-body-empty` finding at warning severity. The rule fires for each entity kind in the per-kind table above. AC bodies use the existing heading locator (`### AC-N — <title>`); top-level kinds scan `## <section>` headings. Definition of empty: between the section heading and the next heading (or EOF), no non-heading non-whitespace content. Multiple blank lines, leading/trailing whitespace, and Windows line endings all count as empty. The finding includes the entity id (composite for ACs), kind, missing section name, file path, and a hint pointing at `aiwf add ac --body-file` (M-067, AC-only) for ACs and to a follow-up gap for the non-AC `--body-file` flags. Implementation: a new rule in `internal/check/`, with per-kind body-section dispatch sharing the heading-locator from the existing `acs-body-coherence` rule.

### AC-2 — Severity escalates to error under aiwf.yaml tdd.strict true

When `aiwf.yaml` contains `tdd.strict: true`, the `entity-body-empty` finding is emitted at error severity instead of warning, regardless of kind. The escalation reads from the same `tdd.strict` field that M-065's `milestone-tdd-undeclared` reads — single source of truth for the project's strictness posture, no parallel field. Tested with two fixtures sharing the same planning tree but differing only in `tdd.strict`; one produces a warning, the other an error. Exit code rises to 1 in the strict case.

### AC-3 — Entities with non-empty body prose produce no finding

For any entity whose load-bearing body sections each contain at least one non-heading line of non-whitespace content, the rule emits no finding. The check is permissive about *what* the prose is — a one-line paragraph, a bullet list, a code block, a single sentence, or rich multi-paragraph detail all clear the rule. The kernel principle "prose is not parsed" applies (per `acs-and-tdd-plan.md:197`); the rule asserts presence, not structure. Tested with several positive fixtures spanning kinds (epic, milestone, AC, gap).

### AC-4 — Bare HTML comments do not satisfy the non-empty requirement

An entity whose load-bearing body section contains only HTML comments (e.g. `<!-- TODO: write this -->` or `<!-- placeholder -->`) is treated as empty — the comment is operator intent to defer, not the prose the design specifies. The rule strips HTML comment blocks before the emptiness check; if nothing non-whitespace remains, the finding fires. Edge case: a single HTML comment followed by real prose passes (the prose is what counts); a single HTML comment with nothing else does not. Tested with both shapes across at least two kinds.

### AC-5 — Finding does not retroactively engage acs-tdd-audit

The grandfather rule from G-055 / G-058 is preserved: for an AC that surfaces `entity-body-empty`, the AC's status / phase fields are not retroactively re-audited against `acs-tdd-audit`. In practice: the historical E-14 milestones (M-049 through M-055), all `met` with empty bodies, will produce `entity-body-empty` warnings per AC but **zero** new `acs-tdd-audit` findings. Same pattern as M-065 / G-055. Top-level kinds do not have an analogous retroactive-audit coupling, so this AC remains AC-scoped in its concern; the assertion is "no new `acs-tdd-audit` findings introduced when adding `entity-body-empty`," which is independent of how many non-AC kinds the rule covers.

### AC-6 — Finding code documented in aiwf-check skill

The `aiwf-check` skill's findings table gains a row for `entity-body-empty`: severity (warning, escalates to error under `tdd.strict: true`), trigger (any load-bearing body section is empty per the per-kind list above), and remediation (write prose for the named section; for ACs, use `aiwf add ac --body-file` from M-067; for other kinds, edit body and run `aiwf edit-body`, until the follow-up gap delivers `--body-file` for those verbs). The discoverability test in `internal/policies/` (per G-021's `PolicyFindingCodesAreDiscoverable`) catches the code at CI time if the row is missing.

## Decisions made during implementation

### D-001 — top-level sections count sub-headings as content; AC bodies require non-heading prose

Surfaced at AC-1 wrap-time. The spec's "no non-heading non-whitespace content" wording read literally would have fired `entity-body-empty/milestone` on every milestone with ACs but no parent-level prose under `## Acceptance criteria` — which is the canonical milestone shape across this repo and the design templates. Decision: top-level `## Section` bodies treat sub-headings as content (a parent is non-empty if it contains anything, headings included); AC `### AC-N` bodies require true leaf prose. The asymmetry matches each level's role — top-level sections are containers, AC bodies are leaf prose. Recorded in [D-001](../../decisions/D-001-entity-body-empty-top-level-sections-count-sub-headings-as-content-only-ac-leaf-bodies-require-non-heading-prose.md).

## Planning notes — AC-6 may collapse into AC-1's discoverability mention

The discoverability policy (G-021) required `entity-body-empty` and its subcodes to appear in `aiwf-check` SKILL.md as soon as the rule literal landed in source. AC-1's cycle therefore added a row that already covers most of AC-6's contract — bare code, all 7 subcodes, severity, escalation note, and per-kind remediation pointers (including the cross-reference to M-067's `aiwf add ac --body-file` for ACs and `aiwf edit-body` for other kinds).

When AC-6 starts: review the row already landed; tighten phrasing if needed; add a worked example or two; verify the discoverability test still passes. The cycle may end up as a contract-pin (no code change beyond minor doc polish) rather than a fresh deliverable. Worth flagging here so AC-6 doesn't drift into bigger scope by re-litigating ground AC-1 already covered.

## Work log

### AC-1 — entity-body-empty (warning) when body section is empty

Landed `internal/check/entity_body.go` with `entityBodyEmpty(t)` walking each entity, reading its body file, and emitting a `Code: "entity-body-empty"` finding for every empty load-bearing section. Per-kind dispatch via the hardcoded `requiredSectionsByKind` map covers all six top-level kinds plus a separate AC-body locator for `### AC-N` sub-elements under milestones. Subcodes per kind (`epic`, `milestone`, `ac`, `gap`, `adr`, `decision`, `contract`) so operators can grep by kind. Top-level emptiness counts sub-headings as content (a milestone's `## Acceptance criteria` is non-empty if it contains AC headings); AC bodies require true non-heading prose — see [D-001](../../decisions/D-001-entity-body-empty-top-level-sections-count-sub-headings-as-content-only-ac-leaf-bodies-require-non-heading-prose.md) for the rationale. HTML comments stripped via regex before the check so `<!-- TODO -->` placeholders don't satisfy the rule (AC-4 preview). Wired into `check.Run` after the AC-related rules; hint entries in `hint.go` for the bare code and each per-kind subcode; a one-row mention in the embedded `aiwf-check` SKILL.md to satisfy the discoverability policy. Tests: initial 10 subcases (firing-per-kind × 7, cancelled-AC-skipped, AC-without-body-heading-skipped, populated negative control). Sanity-checked sound by temporarily neutering the implementation; all 7 firing subcases failed red. One cross-cutting fix needed in `cmd/aiwf/show_cmd_test.go` (its minimal-body fixture milestone now flags). · commit 2e0c90b · tests pass=10 fail=0 skip=0

**Wrap-time tightening (post-AC-1 close).** AC-1 was declared done with the branch-coverage HARD RULE only partly satisfied — file-read errors, frontmatter-parse failures, and two `scanACBodies` heading-clearing arms shipped untested with a "follows the existing acs.go pattern" rationalization that was a bypass, not a precedent. After the user's review pushback ("are you 100% confident?"), four additional tests landed covering every reachable defensive arm (`TestEntityBodyEmpty_FileReadError_SilentlySkipped`, `TestEntityBodyEmpty_FrontmatterParseFailure_SilentlySkipped`, `TestEntityBodyEmpty_ScanACBodies_H2Resets`, `TestEntityBodyEmpty_ScanACBodies_H3NonACResets`); the genuinely-unreachable `requiredSectionsByKind` lookup miss (every Kind is in the map; would only fire on synthetic Kind values) gained a `coverage:ignore-on-miss` comment with rationale. The TDD-discipline slip itself (impl written before any test) is captured separately as [G-067](../../gaps/G-067-wf-tdd-cycle-is-llm-honor-system-advisory-under-load-the-llm-bypasses-red-first-and-the-branch-coverage-hard-rule-without-anything-mechanical-catching-it-m-066-ac-1-cycle-wrote-165-lines-of-impl-before-any-test-existed.md) — a process gap, not a code defect, so handled outside the AC's diff. Final test metrics: pass=14 fail=0 skip=0.

**Backfill follow-up (post-AC-1 close).** Quantified the rule's impact on this repo's tree (62 findings: 61 historical AC bodies in M-049..M-061 plus M-061's empty `## Goal`). Per option G of the noise-handling discussion, M-061's `## Goal` got real prose (one-paragraph synthesis of the milestone's scope) and the 61 ACs got grandfather-stub paragraphs scripted via `tmp/backfill_stubs.py` — each one names where the actual implementation history lives (`aiwf history M-NNN/AC-N`) so the stub is honest acknowledgement, not a silencing trick. 13 `aiwf edit-body` commits landed (one per affected milestone). `aiwf check` after backfill: 0 entity-body-empty findings remain on this repo's tree.

### AC-2 — Severity escalates to error under aiwf.yaml tdd.strict true

Added `Strict bool` to `config.TDD` (yaml `strict`), mirroring the existing `tree.strict` pattern. The escalation lives in a small `check.ApplyTDDStrict(findings, strict bool)` helper that walks the slice in place and bumps every `entity-body-empty` finding from warning to error when strict=true. The helper is the chokepoint for which codes the strict flag covers — today only entity-body-empty; when [M-065](../E-16-tdd-policy-declaration-chokepoint-closes-g-055/M-065-aiwf-check-finding-milestone-tdd-undeclared-as-defense-in-depth.md)'s `milestone-tdd-undeclared` rule lands it joins by adding one case there. Single source of truth per the spec's "no parallel field, no second config knob" rule.

Wiring lives in `cmd/aiwf/main.go`'s check dispatcher: after the rule pipeline runs, it loads the config and calls `check.ApplyTDDStrict(findings, cfg.TDD.Strict)`. The rule emission stays config-agnostic (so render and status callers see the warning by default); strictness escalation is a separate, testable transformation, mirroring how `cfg.Tree.Strict` flows into TreeDiscipline.

Tests: `TestApplyTDDStrict_EscalatesEntityBodyEmpty` covers the bumper in isolation (3 subcases — strict=true bumps, strict=false passes through, nil-slice no-op); `TestCheck_TDDStrict_EscalatesEntityBodyEmpty` exercises the dispatcher seam end-to-end (same scaffolded epic produces exit 0 without `tdd.strict` and `exitFindings` with it). Stub-first RED: function returned no-op; the strict=true subcase failed assertion-red while the no-op cases passed; real implementation made all 4 subcases green. Branch-coverage audit on `ApplyTDDStrict`: 6 reachable arms (strict ✓✗ × loop empty/populated × code-match ✓✗) all exercised. · commit e570c9b · tests pass=4 fail=0 skip=0

### AC-3 — Entities with non-empty body prose produce no finding

Contract-pin AC: no implementation change. AC-1 already implements the permissiveness via `isAllWhitespaceOrHeadings` in `internal/check/entity_body.go` — once any non-heading non-whitespace line appears under a load-bearing heading, the section is non-empty. This AC adds a regression chokepoint that future tightening of the rule must traverse.

`TestEntityBodyEmpty_AcceptsVariedProseShapes` runs 6 prose shapes × 2 levels (top-level milestone `## Approach` + AC-1 leaf body) for 12 subcases. Shapes: single sentence, multi-paragraph, bullet list, numbered list, fenced code block, paragraph + bullets + code (mixed). Each fixture lands the shape as the section content under an otherwise-populated milestone; the assertion is zero findings. The two new fixture builders (`writeMilestoneWithApproachBody`, `writeMilestoneWithACBody`) parameterize over the prose shape so additional shapes (or future cross-kind sweeps) join the table without churn.

TDD posture for a contract-pinning AC: literal RED-first doesn't apply — the implementation already passes — so the discipline is a sanity-check that the test would actually catch a regression. Mutated `isAllWhitespaceOrHeadings` to return `true` unconditionally; all 12 subcases failed red as expected; restored. The mutation-and-restore is a live equivalent of a one-shot mutation test for this AC's specific contract. · commit 620e544 · tests pass=12 fail=0 skip=0

### AC-4 — Bare HTML comments do not satisfy the non-empty requirement

Contract-pin AC: no implementation change. AC-1's diff already shipped `htmlCommentPattern` (a `(?s)<!--.*?-->` matcher) and `stripHTMLComments(body)` which removes every comment block from body bytes before the per-line emptiness walker sees them. Operator-deferred placeholders therefore don't satisfy the rule — the bare-comment body looks empty to `isAllWhitespaceOrHeadings`. AC-4 turns that incidental property into a load-bearing one.

`TestEntityBodyEmpty_HTMLCommentsAreEmpty` covers 6 shapes × 2 levels (12 subcases): single-line comment only / multi-line comment only / two stacked comments / comment with surrounding whitespace (all four → finding fires); comment-then-prose / prose-then-comment (both → no finding, prose still wins). The two-level coverage hits both consumers of `stripHTMLComments` — the top-level `## Approach` walker via `scanH2Sections` and the AC-leaf walker via `scanACBodies` — so a future regression that loses the strip on either path fails here.

Sanity check: replaced `stripHTMLComments` with a passthrough (`return body`). The 8 comment-only subcases failed red as expected (comment text counted as content, no finding fired); the 4 prose-bearing subcases stayed green (correct — real prose makes the section non-empty regardless of strip behavior). Restored, full project suite green. The asymmetric mutation-failure pattern (8 fail, 4 pass) is itself the proof that the test correctly distinguishes the two regression classes. · commit 51659c4 · tests pass=12 fail=0 skip=0

### AC-5 — Finding does not retroactively engage acs-tdd-audit

Contract-pin AC: no implementation change, no new helpers. The grandfather rule from [G-055](../../gaps/G-055-acs-tdd-audit-applied-retroactively-to-pre-existing-met-acs.md) / [G-058](../../gaps/G-058-acs-body-coherence-and-acs-tdd-audit-key-on-different-bodies.md) is preserved by structure: `acsTDDAudit` in `internal/check/acs.go` reads `ac.Status` and `ac.TDDPhase` only; it never opens the body file. `entityBodyEmpty` reads body content and never touches phase. AC-5 turns that structural independence into a regression chokepoint that future couplings must traverse.

`TestEntityBodyEmpty_DoesNotEngageACSTDDAudit` runs two subcases against a `tdd: required` milestone whose AC body is empty:

- `met+done+empty body` — the canonical historical shape (every M-049..M-061 AC after the AC-1 backfill). Asserts `entityBodyEmpty` fires once for the AC body and `acsTDDAudit` produces zero findings. Pins the independence claim directly.
- `met+red+empty body` — sanity case. `acsTDDAudit`'s contract is "met requires done"; flipping the phase to red trips the audit regardless of body content. Without this case, a regression that silenced `acsTDDAudit` entirely (broke its own contract) would slip past subcase 1's zero-count assertion as a false negative.

Sanity check: removed the `if ac.TDDPhase == done { continue }` guard from `acsTDDAudit` (mutating it to fire on every `met` AC regardless of phase). Subcase 1 failed red as expected (audit count = 1, want 0); subcase 2 still passed (count = 1, want 1). The asymmetric failure pattern proves subcase 1 catches a real coupling regression and subcase 2 catches the orthogonal "audit broke entirely" regression. Restored, full project suite green. · commit 8357feb · tests pass=2 fail=0 skip=0

### AC-6 — Finding code documented in aiwf-check skill

Doc-polish AC. AC-1's discoverability-policy compliance had already landed a row in the `aiwf-check` SKILL.md warnings table for `entity-body-empty`. AC-6 reviewed and tightened that row to surface the operator-facing context that landed in M-066's other ACs but never reached the user-visible channel:

- **Asymmetric semantics** (from [D-001](../../decisions/D-001-entity-body-empty-top-level-sections-count-sub-headings-as-content-only-ac-leaf-bodies-require-non-heading-prose.md)) — top-level `## Section` bodies treat sub-headings as content; `### AC-N` bodies require non-heading prose. Without this in the row, operators triaging the `ac` subcode would wonder why the milestone parent's `## Acceptance criteria` section stays silent.
- **Grandfather rule** (from AC-5) — empty-body warnings on historical `met` + `tdd_phase: done` ACs do not retroactively re-engage `acs-tdd-audit`. Reassures operators looking at a flood of pre-rule warnings that no cascading errors will fire.
- **Permissiveness** (from AC-3) — paragraphs, bullets, code blocks, single sentences all clear the rule. Cuts off the most likely operator mistake (thinking the rule wants a particular shape).

The pre-existing material was kept verbatim: per-kind subcode list (all 7 spelled out so the `PolicyFindingCodesAreDiscoverable` substring-match catches each one), HTML-comment treatment, remediation paths via `aiwf edit-body` and `aiwf add ac --body-file`, the [G-066](../../gaps/G-066-aiwf-add-epic-milestone-gap-adr-decision-contract-verbs-lack-body-file-flag-for-in-verb-body-scaffolding-only-aiwf-add-ac-will-gain-it-via-m-067-leaving-the-other-six-entity-creation-verbs-reliant-on-post-add-aiwf-edit-body.md) follow-up pointer for non-AC creation verbs, and the `tdd.strict` severity escalation note.

Sanity check (RED-first analogue for a doc AC): deleted the row entirely. `TestPolicy_FindingCodesAreDiscoverable` failed red on `entity-body-empty/ac` — exactly one violation. Restored with the improved row; policy test green. The asymmetric red pattern surfaced a related blind spot worth noting: the policy's substring matcher only enforces literal `Subcode: "..."` field values discovered by the AST walker. Six of the seven `entity-body-empty` subcodes are derived dynamically (`Subcode: string(e.Kind)`) and don't appear in the AST-discovered code set as required matches. They're documented today only because operator clarity demanded the per-kind list — not because the policy compelled it. To be tracked as a follow-up gap separate from this AC's diff.

No new tests; AC-6 is contract-pin via the existing G-021 chokepoint. · commit 281d3c8 · tests pass=0 fail=0 skip=0

## Validation

Wrap-time checks at 2026-05-07:

- `aiwf show M-066` — 6/6 ACs `[met]` · `phase: done`.
- `aiwf check` — 0 errors, 1 unrelated warning (`provenance-untrailered-scope-undefined`: branch has no upstream per the PoC's local-only convention; not a milestone-quality finding).
- `go test -race ./...` — green across all packages.
- `go build -o /tmp/aiwf-wrap ./cmd/aiwf` — green.
- `golangci-lint run ./...` — 0 issues.
- `wf-doc-lint` (scoped to milestone branch since `poc/aiwf-v3`) — clean. M-066's narrative-doc footprint is empty; new symbols reach AI-discoverable channels via the embedded SKILL.md edit, structurally enforced by `PolicyFindingCodesAreDiscoverable`.

Test additions over the milestone (cumulative):

- `internal/check/entity_body_test.go` — `TestEntityBodyEmpty_FiresPerKind_OneSectionEmpty` (7 kinds), `_CancelledACSkipped`, `_ACWithoutBodyHeadingSkipped`, `_NonEmptyBodyClean`, `_FileReadError_SilentlySkipped`, `_FrontmatterParseFailure_SilentlySkipped`, `_ScanACBodies_H2Resets`, `_ScanACBodies_H3NonACResets`, `_AcceptsVariedProseShapes` (12 subcases), `_HTMLCommentsAreEmpty` (12 subcases), `_DoesNotEngageACSTDDAudit` (2 subcases), `TestApplyTDDStrict_EscalatesEntityBodyEmpty` (3 subcases).
- `cmd/aiwf/check_tdd_strict_test.go` — `TestCheck_TDDStrict_EscalatesEntityBodyEmpty` (dispatcher seam).

Branch-coverage audit (HARD RULE): every reachable arm in `entity_body.go` is exercised by an explicit test. `requiredSectionsByKind` lookup-miss is documented as `coverage:ignore-on-miss` (synthetic Kind values only; tree loader never produces them). `ApplyTDDStrict` 6 arms (strict ✓✗ × loop empty/populated × code-match ✓✗) all hit by the bumper test. Verified mid-cycle by mutation-and-restore on `isAllWhitespaceOrHeadings` (AC-3), `stripHTMLComments` (AC-4), the `acsTDDAudit` phase guard (AC-5), and the SKILL.md row deletion (AC-6) — each mutation produced the expected red pattern in the right subcase set; each was restored before commit.

## Deferrals

None. All six ACs reached `met`. Two follow-up gaps were filed (see Reviewer notes below) but neither represents work this milestone owed and skipped — both are kernel-discipline observations surfaced *during* M-066's implementation that don't block its value.

## Reviewer notes

Two follow-up gaps surfaced during implementation, neither a deferral but both worth tracking:

- [G-067](../../gaps/G-067-wf-tdd-cycle-is-llm-honor-system-advisory-under-load-the-llm-bypasses-red-first-and-the-branch-coverage-hard-rule-without-anything-mechanical-catching-it-m-066-ac-1-gap-cycle-wrote-165-lines-of-impl-before-any-test-existed.md) — `wf-tdd-cycle` is LLM-advisory under load. AC-1 wrote ~165 lines of implementation before any test existed; the slip surfaced at wrap-time review only because the user pushed back ("are you 100% confident?"), not because anything mechanical caught it. AC-1's work-log entry has the full retrospective; the gap captures the process-improvement options. AC-2 onward demonstrated proper RED-first and mutation-and-restore discipline — the slip happened once, was acknowledged, and corrected in-flight.

- [G-068](../../gaps/G-068-discoverability-policy-misses-dynamic-finding-subcodes.md) — `PolicyFindingCodesAreDiscoverable` only enforces literal `Subcode:` field values discovered by the AST walker. Six of the seven `entity-body-empty` subcodes are derived from `string(e.Kind)` and bypass the policy entirely. Documented today by operator-clarity convention (M-066/AC-1 spelled them out for operator triage), not by structural enforcement. Two candidate fixes recorded in the gap body.

Other reviewer-pertinent context:

- **D-001 captures one substantive design call** (top-level sections count sub-headings as content; AC bodies require non-heading prose). The asymmetry exists because `## Acceptance criteria` is a container with sub-headings whereas `### AC-N` is the leaf prose unit. Without this, every AC-bearing milestone in the repo would have flagged its own `## Acceptance criteria` section as empty.
- **The "Planning notes" section's AC-6-may-collapse prediction came true.** AC-6 closed as a doc-polish (no new test, no new code) layered on AC-1's mandatory discoverability row. The prose tightening still added meaningful operator context (asymmetric semantics, grandfather rule, permissiveness paragraph) so the AC wasn't a no-op.
- **One commit-trailer slip on M-066/AC-2** (now closed): the impl commit `e570c9b` carries `aiwf-verb: edit-body` from a stale work-log edit context. Impl/test commits don't normally carry a verb trailer; this was a single-commit accident not corrected (would have required reset/amend). The mistake was acknowledged and AC-3 onward used clean Conventional Commits subject + body without aiwf trailers. Worth flagging because `aiwf history M-066/AC-2` may surface that commit under the edit-body verb filter slightly oddly.
- **Backfill side-deliverable**: 13 `aiwf edit-body` commits backfilled stub prose into 61 historical AC bodies (M-049..M-061) and one milestone Goal section (M-061) per the user's option-G choice during AC-1's noise-handling discussion. Each stub names where the actual implementation history lives (`aiwf history M-NNN/AC-N`) so it's honest acknowledgement rather than silencing. Post-backfill, the kernel repo's `aiwf check` reports zero `entity-body-empty` findings.
