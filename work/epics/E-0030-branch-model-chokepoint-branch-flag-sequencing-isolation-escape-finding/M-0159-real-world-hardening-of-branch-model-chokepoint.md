---
id: M-0159
title: Real-world hardening of branch-model chokepoint
status: in_progress
parent: E-0030
depends_on:
    - M-0102
    - M-0103
    - M-0104
    - M-0105
    - M-0106
    - M-0158
tdd: required
acs:
    - id: AC-1
      title: Combinatorial real-git E2E test framework under internal/cli/integration
      status: met
      tdd_phase: done
    - id: AC-2
      title: M-0106 isolation-escape paths covered by real-git E2E integration tests
      status: met
      tdd_phase: done
    - id: AC-3
      title: walkAcknowledgedSHAs lifted to shared helper, three rules consume it
      status: met
      tdd_phase: done
    - id: AC-4
      title: acknowledge-illegal silences isolation-escape and forced-untrailered
      status: met
      tdd_phase: done
    - id: AC-5
      title: trailer-verb-unknown rule consumes ackedSHAs via shared helper
      status: met
      tdd_phase: done
    - id: AC-6
      title: Cherry-pick gather-side detects cherry picked from commit markers
      status: met
      tdd_phase: done
    - id: AC-7
      title: Cellcoverage fixture aiwf-branch values resolve or are exempted
      status: met
      tdd_phase: done
    - id: AC-8
      title: branch-cell-override-f-nnnn-waiver Kind corrected to finding
      status: met
      tdd_phase: done
    - id: AC-9
      title: internal/check/hint.go names canonical override for isolation-escape
      status: met
      tdd_phase: done
---
## Goal

Land the **combinatorial real-git E2E test framework** for the branch-choreography surface, then use it to ship the override-convergence work (G-0208 + G-0214 / G-0196 + G-0202 + G-0213 sequencing constraint). After this milestone the kernel's branch-policing surface is system-pinned, not just unit-pinned; the override surface is symmetric across the three concrete consumers (`fsm-history-consistent`, `isolation-escape`, `trailer-verb-unknown`); and any future operator hitting one of these scenarios discovers a working override path through `aiwf --help` + tab-completion + skill.

This is **Tier 1 evidence-backed work** per the history-mining audit (June 2026):

- M-0106 itself shipped with the kernel finding effectively disabled for 4 cycles (F-1: CLI passed `nil` for the oracle). The seam-vs-layer gap is documented historical evidence.
- `aiwf-force` trailer has been used 12+ times in production (real overrides, not test fixtures). Operators hand-crafting trailers is documented friction; G-0208's UX gap is grounded.
- G-0214 / G-0196: one real consumer already caught by the `acknowledge-illegal` / `forced-untrailered` asymmetry.
- G-0213: the cellcoverage fictional-branch landmine is a sequencing constraint — must be addressed in the same commit set as any branch-resolution rule that M-0159 introduces.

## Context

The M-0158 honest-scope audit surfaced 11 real-world failure modes catalogued as G-0200 through G-0210. The user's directive during M-0159 planning ("third iteration"):

> *This is so critical to get correct ... I want all thinkable scenarios and realistic combinations to be tested, ie combinatorial. What if. It can happen. Verbs can be composed in any way for any reason ... in the worst case, data loss.*

A confidence-audit workflow (June 2026) then surfaced six test-integrity issues across M-0102..M-0106 + M-0158 (one tautological sabotage, one name/assertion contradiction, one SHA-distinctness fake, one acknowledged-tautological, one self-contradictory docstring, one cross-cell match-bleed). Those landed pre-M-0159 in commit `d43c1f27`.

A history-mining subagent investigation then reframed M-0159 priority from "imagination-driven completeness" to "evidence-driven sequencing":

- **Real evidence in history** (squash-merge override `f4ea7329`/`fdc539b8`, 12+ production `aiwf-force` uses, 26 reallocate commits, G-0167/G-0170 incidents) drives this milestone (M-0159) and M-0160.
- **No in-repo evidence** for G-0200/G-0201/G-0203/G-0204/G-0205/G-0206/G-0207/G-0209 — but per the user's "if we can imagine it, it will happen" principle, these are not dropped; they sequence into M-0161 (Tier 3 imagination-driven hardening) so other operators with different workflows still get coverage.

## Scope split (E-0030 hardening epic — three milestones)

This milestone is **M-0159 (Tier 1)**:

- The combinatorial real-git E2E test framework (closes G-0211).
- Override convergence: extend `acknowledge-illegal` to silence isolation-escape via shared helper-lift, with the same lift covering `forced-untrailered` asymmetry (closes G-0208 + G-0214 + G-0196).
- Cherry-pick gather-side CLI implementation (closes G-0202).
- `trailer-verb-unknown` wires to the shared ack-walk helper (third concrete consumer per the audit).
- Cellcoverage fixture branch-resolution fix (closes G-0213 — sequencing constraint).
- M-0158 spec-table Kind="finding" correctness fix (the rules.go uncommitted patch from M-0158).

**M-0160 (Tier 2 evidence-backed operational pain)** covers:

- Reallocate-stress combinatorial test (26 historical incidents).
- G-0167-class trunk-collision regression test (rename detection).
- G-0170-class apply-rollback data-preservation test.

**M-0161 (Tier 3 imagination-driven hardening)** covers:

- G-0200 (trunk config), G-0201 (cross-rung carve-out), G-0203 (BranchOracle typed errors), G-0204 (shallow clones), G-0205 (force-push), G-0206 (branch rename), G-0207 (detached HEAD), G-0209 (ritual step ordering), G-0210 (M-0158 cell catalog full refactor).
- All require combinatorial E2E coverage via the M-0159 framework.
- "If we can imagine it, it will happen" — different operators have different risk tolerances; coverage is mandatory even without in-repo evidence.

## Pre-decided design

**Test discipline (load-bearing).** Every M-0159 AC requires at least one real-git integration test under `internal/cli/integration/`. The test builds aiwf via `buildAiwfBinary`, sets up a real git repo via `tempRepo`, runs verbs as subprocess invocations, and asserts stdout/stderr/exit-code/trailers/envelope output. Rule-level unit tests stay as cheap regression catches but are NEVER substitutes. **No stubs anywhere.** A test body that doesn't exercise its named claim either gets reframed to match what it actually pins, or rewritten to pin the claim — never deleted, never left as a placeholder.

**G-0208 architecture (Path B with modifications, per confidence-audit workflow).** Lift `walkAcknowledgedSHAs` from `fsm_history_consistent.go` into a shared helper at `internal/check/acks.go` (or equivalent). Three concrete consumers: `fsm-history-consistent` (existing), `isolation-escape` (new), `trailer-verb-unknown` (the third user, currently named-but-not-wired in `trailer_verb_unknown.go:25-29`). The CLI gather layer computes the acked-SHA set once and passes it to all three rules. No `--code` flag, no new `aiwf-force-for-code` trailer, no Cobra rename — the rule does the per-rule SHA matching.

**G-0213 cellcoverage fix (sequencing-load-bearing).** Before landing any rule that reads `aiwf-branch:` against a "must resolve" check, the cellcoverage fixture's fictional branch value must be addressed. Per G-0213, three options: create the branch in the fixture setup, sentinel-trailer the fixture for rule exemption, or have the rule fail-open on empty BranchOracle. Decision lands in M-0159 itself (within the rule-adoption AC).

## Out of scope

- M-0160 and M-0161 work (their gaps remain in their respective milestones).
- New verb addition for G-0208 — Path B keeps the surface to one verb (`acknowledge-illegal`); no new verb shipped this milestone.
- Branch-resolution rule (e.g., "aiwf-branch must point to a real ref") — that's M-0161 work, deliberately gated behind the cellcoverage landmine fix landing first.
- Generalized retroactive override for arbitrary kernel codes — only the three concrete consumers above. The architectural primitive (shared walk + per-rule recognition) supports future expansion but no speculative scaffolding lands now.

## Dependencies

- **M-0102 through M-0106 + M-0158** — all `done`. M-0159 hardens what they delivered.
- **Commit `d43c1f27`** — pre-M-0159 patch round (6 test-integrity fixes) landed before this milestone starts.
- **G-0211, G-0213, G-0214 + existing G-0196, G-0202, G-0208** — gaps consumed by this milestone.

## Acceptance criteria

<!--
AC seed set (to be allocated via `aiwf add ac` at start-milestone time, after the AC framing is confirmed with the user):

1. Combinatorial real-git E2E test framework under internal/cli/integration/branch_scenarios_test.go: scenario-table driver, tempRepo helpers (shallow, rename, force-push, detached-HEAD, cherry-pick, amend, merge setups), envelope assertions. (G-0211)

2. M-0106 paths covered by real-git E2E: every existing M-0106 unit-tested scenario gets a parallel integration test that builds the binary, drives subprocess verbs, asserts envelope output. Closes the "shipped disabled" class.

3. walkAcknowledgedSHAs lifted to internal/check/acks.go; consumed by fsm-history-consistent, isolation-escape, and trailer-verb-unknown rules through a single ackedSHAs map[string]bool parameter populated by the CLI gather layer.

4. acknowledge-illegal extended to cover isolation-escape AND forced-untrailered subcodes via the shared helper. Real-git E2E: AI escape → aiwf acknowledge-illegal <sha> --reason → aiwf check silent; AI authorship preserved on original commit. (G-0208 + G-0214 + G-0196)

5. trailer-verb-unknown wired to consume ackedSHAs through the lifted helper. Real-git E2E: historical stray commit acked → check silent. Converts the docstring promise at trailer_verb_unknown.go:25-29 into mechanical truth.

6. Cherry-pick gather-side implemented in the CLI: real (cherry picked from commit <sha>) markers in commit bodies populate the cherryPicked map. Real-git E2E: git cherry-pick -x of an isolation-escape commit → check silent. (G-0202)

7. Cellcoverage fixture branch-resolution decision landed in the same commit set as any new branch-reading rule. (G-0213)

8. M-0158 spec-table Kind="finding" correctness fix (the uncommitted rules.go change addressing the M-0158 wrap miss).

9. internal/check/hint.go updated to name aiwf acknowledge-illegal as the canonical override invocation for isolation-escape findings; substring-tested at integration level.

These 9 are the seed set; aiwfx-start-milestone refines and allocates them.
-->

### AC-1 — Combinatorial real-git E2E test framework under internal/cli/integration

Landed the `Scenario` / `Expectation` / `RunScenarios` driver shape in [`internal/cli/integration/branch_scenarios_test.go`](../../../internal/cli/integration/branch_scenarios_test.go), the per-scenario `ScenarioEnv` (tempRepo + diag-binary + per-test stdout/stderr capture), and the helper library at [`branch_scenarios_helpers_test.go`](../../../internal/cli/integration/branch_scenarios_helpers_test.go) (OpenBoundScope, SimulateAIEscapeCommit, SimulateForcedUntrailedActivate, SimulateStrayVerbCommit, CherryPick, structural trailer-query helpers). Real-git: every scenario runs `git init`, `git checkout -b main`, real-binary verb invocations through subprocess, real commits with real trailers — no in-process mocks.

**Pinned by:** the framework itself plus every scenario set added in AC-2 / AC-4 / AC-5 / AC-6 (a framework without consumers is a framework that doesn't work). Closes G-0211 (framework). The cellcoverage-fixture-side framework hygiene fix (newScenarioEnv upstream-tracking severance) was caught mid-AC-6 — see Decisions made during implementation.

### AC-2 — M-0106 isolation-escape paths covered by real-git E2E integration tests

Every M-0106 unit-tested scenario gained a parallel integration scenario under [`internal/cli/integration/branch_scenarios_ac2_test.go`](../../../internal/cli/integration/branch_scenarios_ac2_test.go) that builds the binary, drives subprocess verbs, and asserts envelope output (code + severity + hint substrings). The "shipped disabled" failure class M-0106 was caught with — the CLI seam passing `nil` for the oracle — would now surface immediately because every scenario fires `RunProvenanceCheck` end-to-end and parses the JSON envelope, not the rule's internal Findings slice.

**Pinned by:** scenarios under AC-2 plus `FindingHintContainsAll` (envelope-side substring helper). Together with `TestRunProvenanceCheck_IsolationEscape_FiresOnViolatingCommit` they form the seam pin M-0106 was missing. M-0106 §AC-12 was updated at M-0159/AC-9 — see AC-9 body for the marker-count drift.

### AC-3 — walkAcknowledgedSHAs lifted to shared helper, three rules consume it

`walkAcknowledgedSHAs` extracted from `fsm_history_consistent.go` into shared [`internal/check/acks.go`](../../../internal/check/acks.go); the CLI gather layer at [`internal/cli/check/check.go`](../../../internal/cli/check/check.go) computes the ackedSHAs map exactly once per check invocation and threads it to all three rules. Concrete consumers: `fsm-history-consistent/illegal-transition` (existing), `isolation-escape` (new wire-up via [`internal/cli/check/provenance.go`](../../../internal/cli/check/provenance.go)), `trailer-verb-unknown` (existing docstring promise converted to mechanical truth in AC-5).

**Pinned by:** `PolicyAcksHelperLift` (asserts every consumer takes the lifted signature; fires if a fourth call site adopts a parallel walk). The "one walk, three consumers" property is mechanical, not advisory. No `--code` flag, no new `aiwf-force-for-code` trailer — per-rule SHA matching, single-trailer surface.

### AC-4 — acknowledge-illegal silences isolation-escape and forced-untrailered

`forcedUntraileredFindings` extended to consume the ackedSHAs map the same way `illegalTransitionFindings` already did; provenance.go's isolation-escape wire-up gained the same threading. Real-git E2E: AI escape commit → `aiwf acknowledge-illegal <sha> --reason "..."` records an empty audit-trail commit carrying `aiwf-force-for: <sha>` + `aiwf-actor: human/...` → next `aiwf check` runs silent on both subcodes. **Original commit's AI authorship preserved** — no history rewrite; structural trailer-query helper added to assert author-preservation invariantly.

**Pinned by:** AC-4 ack-silencing scenarios in `branch_scenarios_ac4_test.go` plus 13 existing `forcedUntraileredFindings(...)` call sites updated to the new signature (extension of `PolicyAcksHelperLift` class 4d catches future drift). Closes G-0196 + G-0214 (the asymmetry these gaps named). The `SimulateForcedUntrailedActivate` helper was hardened against future body-content fixtures during refactor — see Decisions made during implementation.

### AC-5 — trailer-verb-unknown rule consumes ackedSHAs via shared helper

`trailer-verb-unknown` rule wired to the shared ack-walk helper, converting the docstring promise at `trailer_verb_unknown.go:25-29` into mechanical truth. Real-git E2E: historical commit carrying a fabricated `aiwf-verb: implement` value fires the rule; the operator then runs `aiwf acknowledge-illegal <sha> --reason "..."` and the rule goes silent on the next check.

**Pinned by:** four scenarios in [`branch_scenarios_ac5_test.go`](../../../internal/cli/integration/branch_scenarios_ac5_test.go) (silencing happy path, marker-only negative, gap-only negative, baseline positive). Refactor pass added a reachability assertion via `git merge-base --is-ancestor` so the test discriminates against "commit on wrong branch" failure modes that would otherwise pass spuriously. RED+GREEN combined per pre-commit hook constraint.

### AC-6 — Cherry-pick gather-side detects cherry picked from commit markers

`WalkCherryPicks` landed in [`internal/check/cherry_picks.go`](../../../internal/check/cherry_picks.go); the CLI gather layer at provenance.go now computes the cherryPicked map via a single `git log` subprocess walking HEAD-reachable history. **Both-signals contract:** `(cherry picked from commit <sha>)` marker line in body AND committer email != author email. Either signal alone keeps the rule firing (per the docstring's failure-mode enumeration). Closes G-0202 — before this commit, isolation-escape's cherry-pick suppression arm could not fire end-to-end because the gather-side passed `nil`.

**Pinned by:** four real-git scenarios in [`branch_scenarios_ac6_test.go`](../../../internal/cli/integration/branch_scenarios_ac6_test.go). Discoveries during RED: (1) the original topology put OpenBoundScope after `git checkout -b epic/...`, making the opener unreachable from main and causing scenarios to pass spuriously through a different failure mode → **opener-first fixture topology** is now load-bearing for any scenario that switches off bound branch (Decisions made during implementation). (2) SHA collision in deterministic-identity test mode: `-c user.email=X cherry-pick` is silently ignored because git evaluates `GIT_*_EMAIL` env vars at higher precedence than `-c` config → added `RunGitWithExtraEnv` to the testutil package (Decisions made during implementation).

### AC-7 — Cellcoverage fixture aiwf-branch values resolve or are exempted

`CellFixture.AuthorizeScope` at [`internal/cellcoverage/authorized_scope.go`](../../../internal/cellcoverage/authorized_scope.go) now creates the named branch via `git branch <name>` in the fixture's tmp git repo BEFORE invoking `verb.Authorize`, so the stamped `aiwf-branch:` trailer value resolves end-to-end. **G-0213's Option 1** (chosen at AC-7 design call): keeps production rule semantics strict, no production-code coupling to fixture markers, ~few-ms overhead per cell. Options 2 (sentinel trailer) and 3 (rule fail-open on empty BranchOracle) were rejected for coupling/safety reasons.

**Pinned by:** [`TestCellFixture_AuthorizeScope_AIWFBranchTrailerResolves`](../../../internal/cellcoverage/authorized_scope_branch_resolves_test.go) — drives AuthorizeScope end-to-end and asserts the trailer value resolves via `git rev-parse --verify refs/heads/<name>`. Closes G-0213. The HEAD precondition the helper depends on is the caller's prior `verb.Add + verb.Promote` (initrepo.Init itself produces no commits, per its own docstring at line 5) — corrected in the refactor pass after reviewer caught the original docstring claim.

### AC-8 — branch-cell-override-f-nnnn-waiver Kind corrected to finding

`branch-cell-override-f-nnnn-waiver` cell at [`internal/workflows/spec/branch/rules.go`](../../../internal/workflows/spec/branch/rules.go) flipped from `Kind: "gap"` to `Kind: "finding"` per ADR-0003 §"Decision". The patch had been prepared during M-0158 wrap but never committed; the M-0158 audit surfaced it as a carryover. ADR-0003 declares `finding` as the seventh entity kind (stored at `work/findings/F-NNNN-*.md`); the kind itself is not yet implemented in `entity.AllKinds()` (six-kind PoC), but the spec table's job is to catalog the override surface correctly so future consumers reading the catalog see the right surface name.

**Pinned by:** [`TestM0159_AC8_FNNNNWaiverCellKindIsFinding`](../../../internal/policies/m0159_ac8_kind_correction_test.go) — structural assertion: `cell.Kind == entity.Kind("finding")`. RED+GREEN one-shot per pre-commit hook constraint. The fix is a forward-declaration: when the kernel implements the finding kind, the cell already names the correct surface.

### AC-9 — internal/check/hint.go names canonical override for isolation-escape

`isolation-escape` hint at [`internal/check/hint.go`](../../../internal/check/hint.go) now names three sovereign override paths: (a) canonical: `aiwf acknowledge-illegal <sha> --reason "..."` — separate empty commit, no history rewrite, traces via `aiwf history` through the aiwf-force-for trailer; (b) `git cherry-pick -x` re-author; (c) `aiwf-force` trailer amend. acknowledge-illegal is named first because it's the kernel-native canonical: operators reading the rendered hint see the clean verb path before the operator-hostile amend path that G-0208 named as friction.

**Pinned by:** [`TestIsolationEscape_AC12_HintTextNamesAllOverridePaths`](../../../internal/check/isolation_escape_test.go) (unit; renamed `Both → All` at AC-9 — wantSubstrings table grew from 4 to 5 markers) + [`TestRunProvenanceCheck_IsolationEscape_FindingCarriesHint`](../../../internal/cli/check/isolation_escape_test.go) (integration: substring assertion on the rendered Hint of a real finding through `RunProvenanceCheck` → `ApplyHintsLikeRun`). M-0106 spec §AC-12 was updated with an "Extended by: M-0159/AC-9" cross-reference and the marker count corrected from 4 to 5. The F-7 caveat (substring tests are circular tautologies) extends to AC-9 verbatim — see Reviewer notes for the future-tightening note.

## Work log

### Cycle 1 — AC-1 + AC-2 (framework + M-0106 paths)

Combinatorial real-git E2E framework landed alongside the M-0106 path coverage that proved the framework worked end-to-end. Real binary, real subprocess, real envelope assertions. `FindingHintContainsAll` envelope helper introduced for substring assertions against rendered hints.

### Cycle 2 — AC-3 + AC-4 + AC-5 (override convergence)

`walkAcknowledgedSHAs` lifted to `internal/check/acks.go`; CLI gather layer threads ackedSHAs to all three rules. AC-4 wired `acknowledge-illegal` to silence both `isolation-escape` AND `forced-untrailered` via the lifted helper — closes G-0196 + G-0214. AC-5 wired `trailer-verb-unknown` to consume ackedSHAs through the same helper; the docstring promise at `trailer_verb_unknown.go:25-29` is now mechanical. 13 existing `forcedUntraileredFindings(...)` call sites updated atomically; `PolicyAcksHelperLift` class 4d extended to catch future drift.

### Cycle 3 — AC-6 + AC-7 + AC-8 (cherry-pick gather + landmine + carryover)

`WalkCherryPicks` landed in `internal/check/cherry_picks.go`; CLI gather computes cherryPicked map once per check invocation. Closes G-0202. Cellcoverage fixture's fictional-branch landmine fixed at `AuthorizeScope` (Option 1: create the branch in fixture setup) — closes G-0213. M-0158 spec-table Kind="finding" carryover patch committed in the same cycle.

### Cycle 4 — AC-9 (hint canonical)

`isolation-escape` hint at `internal/check/hint.go` extended to name `aiwf acknowledge-illegal` as canonical override path (a); unit + integration substring assertions added; M-0106 §AC-12 spec extended with "Extended by: M-0159/AC-9" cross-reference and marker count corrected from 4 to 5. Reviewer subagent caught one blocker (B-1: M-0106 spec marker-count drift) and three nits (rename Both→All, Errorf rationale extraction, lineage-stack cleanup), all addressed in the same commit.

### Post-cycle — gap closures

G-0196, G-0214, G-0208 mechanically closed via `aiwf promote --by M-0159` after AC-9 landed. G-0211, G-0213 closed earlier as their respective ACs landed.

## Decisions made during implementation

- **Opener-first fixture topology** (AC-6 discovery, framework hygiene). For any scenario that switches off the bound branch, `OpenBoundScope` must run on `main` BEFORE the `git checkout -b epic/...`. Inverting this puts the opener on the ritual branch, making it unreachable from main; isolation-escape then can't determine the bound branch and the scenario passes through a different failure mode (`provenance-authorization-missing`) instead of the failure mode the AC named. This pattern is now load-bearing for any future cross-branch scenario.

- **`RunGitWithExtraEnv` testutil helper** (AC-6 discovery, framework hygiene). Git evaluates `GIT_AUTHOR_EMAIL` / `GIT_COMMITTER_EMAIL` env vars at higher precedence than `-c user.email=X` config overrides. TestMain forces deterministic identity via env vars process-wide; without per-call env overrides, `-c user.email=X cherry-pick` is silently ignored and the cherry-picked commit gets a byte-identical SHA to its source (same parent/tree/author/committer + same-second timestamps). Added [`testutil/proc.go::RunGitWithExtraEnv`](../../../internal/cli/cliutil/testutil/proc.go) so tests can override per-call. General-purpose; not coupled to AC-6.

- **`newScenarioEnv` upstream-tracking severance** (AC-5/AC-6 framework hygiene). `git checkout -B main` after `git init` severs the upstream-tracking ref; scenarios calling `git push` without `--set-upstream` then fail in non-obvious ways. Added `git branch --set-upstream-to=origin/main main` to `newScenarioEnv`. Pre-existing latent bug surfaced by AC-5's stray-verb scenarios.

- **G-0213 mitigation choice: Option 1 over Options 2/3** (AC-7 design call). Create the branch in the fixture's tmp git repo before stamping the trailer. Option 2 (sentinel trailer) was rejected: couples production rule code to a fixture marker. Option 3 (rule fail-open on empty BranchOracle) was rejected: trades a real safety property for fixture convenience. Cost: few-ms `git branch <name>` per cell. Benefit: production rule semantics stay strict; the fixture's `aiwf-branch:` trailer resolves end-to-end against any future branch-resolution rule.

- **RED+GREEN one-shot commit pattern for body-only edits** (AC-8, AC-9). Pre-commit hook runs `go test ./internal/policies/...` which would fail with a RED-only commit when the test under the policies package is the test pinning the AC. Pattern adopted at AC-5 (when the pre-commit hook landed after the milestone start) and extended to AC-8, AC-9 — combine RED + GREEN in one commit, sabotage-verify by reverting only the implementation half, then promote phase red→green→done in normal sequence.

- **M-0106 §AC-12 spec extension shape** (AC-9 reviewer discovery). When a later AC layers on top of an earlier "met" AC's claim, the spec extension goes in the original AC's body as a trailing "**Extended by: <later AC ref>**" line, with the historical prose left intact. Don't retroactively rewrite the earlier AC's claim; route future readers via the cross-reference. (Reviewer's option-b — option-a would have rewritten AC-12's prose to subsume AC-9's addition, conflating the historical pin with the extension.)

- **Two-reviewer subagent pattern at refactor.** Established mid-AC-5 ("did you do a review on subagent?"). Reviewer subagent runs BEFORE commits land, not just at refactor gate. Caught real bugs each time: AC-6 line ref drift (cherry_picks.go cited isolation_escape.go:258 instead of 269); AC-7 docstring inaccuracy (claimed `aiwf init` produced one commit when initrepo.Init never commits); AC-9 blocker B-1 (M-0106 spec marker-count drift) plus three nits. Discipline pin: reviewer-before-commit, not reviewer-after-refactor.

## Validation

- `go test ./...` — 57 packages green, 0 fail. One TempDir RemoveAll flake on first run cleared on retry (known race; documented in session summary, not introduced by M-0159).
- `aiwf check` — 0 errors, advisory warnings only (the 9 entity-body-empty subsection warnings cleared by this body edit; the 3 `terminal-entity-not-archived` warnings + 1 `archive-sweep-pending` aggregate fire post-gap-closure and clear when `aiwf archive --apply` sweeps the closed gaps at wrap).
- Policies suite — all green including the four AC-9-touching policies (`PolicyFindingCodesHaveHints`, `PolicyAcksHelperLift`, `PolicySkillCoverage`, `PolicyFindingCodesAreDiscoverable`).
- Sabotage probes per AC — RED-discrimination confirmed end-to-end on every load-bearing assertion (hint substrings, ack-helper signature, trailer presence, cherry-pick both-signals, branch resolvability).

## Deferrals

None of M-0159's scope deferred. Sibling milestones remain open as planned:

- [M-0160](M-0160-operational-pain-reallocate-stress-trunk-collision-regress-apply-rollback.md) — Tier 2 evidence-backed operational pain (reallocate-stress, trunk-collision regression, apply-rollback). Independent of M-0159.
- [M-0161](M-0161-imagination-driven-hardening-shallow-force-push-rename-detached-trunk.md) — Tier 3 imagination-driven hardening. Consumes the M-0159 framework; was sequencing-blocked behind G-0213 which AC-7 cleared.

The five referenced gaps land as follows: G-0211 (framework) closed by AC-1; G-0213 (cellcoverage landmine) closed by AC-7; G-0196 + G-0214 (acknowledge-illegal asymmetry) closed by AC-3+AC-4; G-0208 (amend UX) closed by AC-3+AC-4+AC-9 via canonical-path routing; G-0212 + G-0215 remain open as scoped (future tightening — kernel-wide nil-pass audit + structural chokepoint policy is the canonical follow-up).

## Reviewer notes

**Reviewer-before-commit pattern established.** Mid-AC-5 the user prompted *"did you do a review on subagent?"* — the discipline is to run the reviewer subagent BEFORE staging the commit, not just at refactor gate. Every subsequent AC's commit landed only after a reviewer pass that produced specific file:line findings. Three of those findings were real bugs (line-ref drift, docstring inaccuracy, spec marker-count drift); a fourth was a defensible-but-overstated rename rationale that the user-side trade-off favored fixing in the same commit.

**Substring-test caveat extends.** The AC-9 hint substring tests are subject to the M-0106/F-7 caveat: author writes hint, substrings, and sabotage probe. The caveat catches "hint removed entirely" regressions but not "wording drifts from what an LLM agent parses for remediation." When the hint becomes machine-consumed (named-fragments struct or golden file), the place to tighten is `TestIsolationEscape_AC12_HintTextNamesAllOverridePaths` plus the integration twin. AC-9's docstring extension makes this explicit so a future tightener finds the right anchor.

**Rename trade-off (`BothOverridePaths` → `AllOverridePaths`).** Reviewer surfaced as a nit but the kept-name comment overstated the rename cost ("widely referenced" was actually 4 sites). Resolved by renaming in the same commit; the post-rename name pins 3 paths and 5 markers, consistent with the layer it now describes. Discipline pin: when an inline-comment rationale is auditable in 30 seconds (`grep -rn`), prefer the rename to the kept-name-with-rationale.

**M-0158 carryover patch landed.** The Kind="finding" patch had been prepared during M-0158 wrap but never committed; the M-0158 wrap-time audit surfaced it explicitly as carryover. AC-8 landed the patch atomically with a structural pin (`TestM0159_AC8_FNNNNWaiverCellKindIsFinding`) so future wraps can't drop the same shape.

**G-0215 carries forward.** Kernel-wide nil-pass audit + structural chokepoint policy ("a CLI gather-layer parameter passing literal nil to a rule must be checked for nil-passing-as-fail-shut intent") remains open. The same failure pattern that produced M-0106's F-1 (oracle wired with nil), AC-6's provenance.go:67 (cherryPicked wired with nil), and a future repeat in any similar wire-up are the same shape: silent fail-open at a load-bearing seam. Tracked for a future milestone, not in-scope for M-0159.

**E-0030 stays active until M-0159 wraps.** Per M-0158's wrap-time decision. With M-0159 wrapping now, the epic can transition: either M-0160/M-0161 stay open under E-0030 (planned), or a follow-up reshuffle splits them into a successor epic. That's a wrap-epic decision, not an M-0159 decision.

