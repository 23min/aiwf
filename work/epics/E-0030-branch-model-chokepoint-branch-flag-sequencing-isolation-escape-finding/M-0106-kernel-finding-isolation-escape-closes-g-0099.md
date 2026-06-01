---
id: M-0106
title: Kernel finding isolation-escape (closes G-0099)
status: done
parent: E-0030
depends_on:
    - M-0102
    - M-0103
tdd: required
acs:
    - id: AC-1
      title: AI commit on main while scope binds epic/X fires isolation-escape
      status: met
      tdd_phase: done
    - id: AC-2
      title: AI commit on epic/Y while scope binds epic/X fires isolation-escape
      status: met
      tdd_phase: done
    - id: AC-3
      title: AI commit on worktree-vs-branch mismatch fires isolation-escape
      status: met
      tdd_phase: done
    - id: AC-4
      title: AI commit on epic/X while scope binds epic/X stays silent
      status: met
      tdd_phase: done
    - id: AC-5
      title: AI commit on epic/X while scope is paused stays silent
      status: met
      tdd_phase: done
    - id: AC-6
      title: Human cherry-pick (committer != actor + marker) stays silent
      status: met
      tdd_phase: done
    - id: AC-7
      title: Human merge of epic/X into main (first-parent) stays silent
      status: met
      tdd_phase: done
    - id: AC-8
      title: Violating commit amended with aiwf-force + human/ actor stays silent
      status: met
      tdd_phase: done
    - id: AC-9
      title: AI commit on entity with no opened scope stays silent
      status: met
      tdd_phase: done
    - id: AC-10
      title: Per-commit firing — one finding per violating commit
      status: met
      tdd_phase: done
    - id: AC-11
      title: Warning severity; check exits 0 with findings reported
      status: met
      tdd_phase: done
    - id: AC-12
      title: Finding hint text names both override paths
      status: met
      tdd_phase: done
    - id: AC-13
      title: Typed Code descriptor lands in internal/check/ per G-0129 pattern
      status: met
      tdd_phase: done
---

## Goal

Add a kernel finding `isolation-escape` that fires at `aiwf check` (pre-push) when an AI-actor's commits violate [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s branch convention — i.e., when commits made under an active AI scope land on a branch that doesn't match the scope's `aiwf-branch:` trailer. Cherry-picks of `ai-actor` commits to other branches are recognized as sovereign re-authorship (committer ≠ actor + cherry-pick marker in body) and suppressed. Per-commit signal at warning severity at first land. Closes [G-0099](../../gaps/G-0099-worktree-isolation-parent-side-precondition.md) fully.

## Context

The session-layer PreToolUse hook (already landed) denies the `isolation: "worktree"` Agent kwarg, preventing one failure mode. M-0102 + M-0103 prevent another: AI dispatch without a named branch is refused at the verb. This milestone adds the *third* layer — post-hoc detection of drift that slips through both gates (e.g., a subagent that escapes its assigned branch via `cd ..`, `git -C <other-path>`, or `git checkout main` from inside the worktree, or a manual cherry-pick that violates the scope-branch coupling).

Together the three surfaces give defense in depth: pre-dispatch (session-layer hook), at-dispatch (preflight), and at-push (kernel finding). The finding is the unbypassable layer for *unintentional* drift — it fires regardless of which dispatch path the parent used. *Intentional* sovereign overrides remain available via the documented `--force` and cherry-pick paths (see epic body §"Sovereign override surface").

The finding polices AI-actor commits only (per ADR-0010's sovereignty principle); human-actor commits, including manual cherry-picks between branches, are not policed. The cherry-pick of an `ai/X` commit is recognized as sovereign re-author and suppressed.

## Pre-decided design

Per E-0030 §"Design decisions" and §"Corner cases":

- **Finding scope:** per-commit. Each violating commit fires its own `isolation-escape` finding.
- **Severity:** warning at first land. Tighten to error after one full epic of usage, gated on the false-positive rate being acceptable. Severity transition is recorded as a D-NNN at the time of tightening; this milestone does not pre-commit the timing.
- **Detection algorithm** (per-commit, idempotent):
  1. Filter commits in the check range to those carrying `aiwf-actor: ai/...` AND `aiwf-entity: <id>` trailers.
  2. For each candidate commit `C`:
     - Walk back from `C` through the commit's ancestor chain (or the entity's commit history, whichever the existing `aiwf history` machinery uses) to find the most recent `aiwf-verb: authorize` + `aiwf-scope: opened` commit on `C.aiwf-entity` whose `aiwf-actor: human/...` opened a scope that was *active* at `C`'s time. If no active scope existed at `C`'s time, skip (corner case 11 — no binding, no finding).
     - Read that authorize commit's `aiwf-branch:` trailer. If absent (legacy commits pre-M-0102), skip (non-retroactive per epic §"Out of scope").
     - Check first-parent reachability: `git merge-base --is-ancestor <C> <scope.aiwf-branch>` AND NOT (`<C>` reachable from `main --first-parent` *unless* `<scope.aiwf-branch>` is itself `main`, which by ritual shape it never is). If reachability matches the bound branch, silent. If `C` is on `main` directly or on a different ritual branch, fire `isolation-escape` against `C` with `aiwf-entity` cited and the scope's authorize-commit SHA as evidence.
  3. Before firing, apply the **cherry-pick suppression check**: if the commit's committer email differs from the actor's encoded email (extracted via the same role-to-email mapping `cliutil.ResolveActor` uses, or by reading the commit's `Co-Authored-By:` line if present) AND the commit body contains the regex `\(cherry picked from commit [0-9a-f]{7,40}\)` (the marker `git cherry-pick -x` writes by default), suppress the finding — this is corner case 8 (sovereign re-author). The hint text on the finding names the suppression path.
- **Override path** (other than cherry-pick): the violating commit's author amends it with an `aiwf-force: "<reason>"` trailer + `human/` actor. The existing trailer-shape rule (`gitops.TrailerForce` requires `value != ""` after trim; `gitops.TrailerActor` requires `human/` for paths that elevate via `--force`) gates the amend. M-0106's finding rule reads `aiwf-force:` on the violating commit and suppresses when present + actor is `human/`.
- **Paused-scope handling:** commits made while a scope is `paused` still ride the bound branch (the scope's `aiwf-branch:` is what was recorded at `opened`; `paused` doesn't change the binding). This is corner case 6 — the finding is silent when the scope is paused and the commit's branch matches.
- **Hint text** (one-liner per `internal/check/hint.go` pattern):
  > *"`isolation-escape`: AI-actor commit on a branch that doesn't match the active scope's recorded `aiwf-branch:`. Override paths: (a) re-author via `git cherry-pick -x` (preserves marker + changes committer); (b) amend the commit with `--trailer 'aiwf-force: <reason>'` and a `human/` actor. See E-0030 epic body §"Sovereign override surface" for the audit trail each path produces."*
- **Finding-code constant naming:** `CodeIsolationEscape = check.Code{ID: "isolation-escape", Class: check.ClassBranchChoreography}` — a new class added to the spec, matching the ADR-0011 layer-4 carve-out. The class constant lands in `internal/check/` per the existing typed-code pattern (and per G-0129 if that lands first; this milestone aligns with whichever pattern is current).

## Out of scope

- Rituals updates (M-0104 / M-0105).
- Author iteration — the finding fires only on AI-actor commits.
- Non-aiwf commits — only commits carrying an `aiwf-entity:` trailer are inspected.
- Retroactive enforcement — only commits made under active scopes after this milestone lands are policed (the `aiwf-branch:` trailer must be present on the active scope's authorize commit; pre-M-0102 scopes don't have it and the rule skips).
- Spec-cell registration in `internal/workflows/spec/branch/` — that's the consolidation milestone's work.

## Dependencies

- **M-0102** — provides the `aiwf-branch:` trailer the finding reads from.
- **M-0103** — the preflight chokepoint that this finding is the post-hoc complement of. The finding's algorithm assumes M-0103 has been preventing the bad-dispatch case at the source, so its caseload is "dispatch was OK at the time; the subagent then drifted."

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0106` time. AC seed set, organized by corner case (epic §"Corner cases"):

Positive (rule fires on these — illegal cells):
1. AI-actor commit on main while scope's `aiwf-branch:` is `epic/E-NN-X` → `isolation-escape` fires (corner case 4).
2. AI-actor commit on `epic/E-NN-Y` while active scope's `aiwf-branch:` is `epic/E-NN-X` → `isolation-escape` fires (corner case 7).
3. AI-actor commit on a worktree path that doesn't match its branch (subagent did `git checkout main` from inside the worktree) → `isolation-escape` fires; the test fixture verifies the cause is branch identity, not path (corner case 12).

Negative (rule silent — legal cells):
4. AI-actor commit on `epic/E-NN-X` while scope's `aiwf-branch:` is `epic/E-NN-X` → silent (corner case 5).
5. AI-actor commit on `epic/E-NN-X` while scope is paused → silent (corner case 6).
6. Human cherry-pick of `ai/X` commit from `epic/E-NN-X` onto main (committer ≠ actor + marker in body) → silent (corner case 8).
7. AI-actor commit on `epic/E-NN-X`, then `--no-ff` merge of `epic/E-NN-X` into main; check the merge commit (which is human-actor) → silent (corner case 9 — the merge commit's actor is human; first-parent reachability puts the ai-actor commit behind the merge, not on main's first-parent).
8. Violating commit amended with `aiwf-force: "<reason>"` + `human/` actor → silent (corner case 10).
9. AI-actor commit on entity with no scope ever opened → silent (corner case 11).

Mechanical:
10. Per-commit firing (one finding per violating commit, not aggregated).
11. Warning severity (not error). The check exits 0 with findings reported.
12. Finding hint text matches the one-liner above and names both override paths.
13. The finding's typed `Code` descriptor lands in `internal/check/` per the existing pattern (or G-0129's, whichever is current at landing time).

These ACs cover the catalog rows 4–12 from the epic body; rows 1–3 are M-0103's territory.
-->

### AC-1 — AI commit on main while scope binds epic/X fires isolation-escape

An AI-actor work commit on `main` while the active scope's `aiwf-branch:` trailer names a ritual epic branch (e.g. `epic/E-0001-engine`) fires `isolation-escape`. The detection: per-commit, walk back to the most recent `aiwf-scope: opened` commit on the same entity, read its `aiwf-branch:`, and compare to the commit's actual branch via the [`BranchOracle.FirstParentBranches`](../../../internal/check/isolation_escape.go) call.

**Pinned by:** [`TestIsolationEscape_AC1_AICommitOnMainFires`](../../../internal/check/isolation_escape_test.go) — asserts exactly 1 finding with `Code = CodeIsolationEscape.ID`, `Severity = SeverityWarning`, `EntityID = E-0001`, and a `Message` naming both the actual branch (`main`) and the bound branch (`epic/E-0001-engine`). Sabotage-verified by inverting the silent-condition — test fires on every regression that flips the comparison.

### AC-2 — AI commit on epic/Y while scope binds epic/X fires isolation-escape

The detection is "equals bound branch", not "is a ritual shape" — landing on a different epic branch (`epic/E-0002-other`) while bound to `epic/E-0001-engine` fires. This guards against a regression that treated all ritual shapes as equivalent and only fired on commits to `main`.

**Pinned by:** [`TestIsolationEscape_AC2_AICommitOnDifferentRitualBranchFires`](../../../internal/check/isolation_escape_test.go) — asserts the wrong branch name (`epic/E-0002-other`) appears in the finding's message.

### AC-3 — AI commit on worktree-vs-branch mismatch fires isolation-escape

The G-0099 "subagent did `git checkout main` from inside its assigned worktree" scenario is detected via the same branch-identity comparison as AC-1. The rule does NOT validate filesystem paths or worktree metadata — only branch identity. The worktree dimension is a fixture variation of AC-1, documented explicitly so a future reader connects this rule's coverage to G-0099's original failure mode.

**Pinned by:** [`TestIsolationEscape_AC3_WorktreeBranchMismatchFires`](../../../internal/check/isolation_escape_test.go) — fixture mirrors AC-1 but explicitly comments the worktree-escape scenario.

### AC-4 — AI commit on epic/X while scope binds epic/X stays silent

The base "no escape" case: when the oracle confirms the commit rides the bound branch, no finding. The `slices.Contains(actualBranches, bound)` guard short-circuits before the fire path.

**Pinned by:** [`TestIsolationEscape_AC4_AICommitOnBoundBranchSilent`](../../../internal/check/isolation_escape_test.go) — asserts zero findings.

### AC-5 — AI commit on epic/X while scope is paused stays silent

The pause event does NOT change the binding — `aiwf-branch:` is recorded at `opened` and remains the scope's bound ref through pause/resume cycles. A commit on the bound branch during the paused phase is silent (corner case 6 from the epic body). The algorithm naturally handles this: the bound-branch index keys only on `aiwf-scope: opened` events, so pause/resume commits don't shift the binding; the existing "rides bound branch" check applies as for AC-4.

**Pinned by:** [`TestIsolationEscape_AC5_AICommitOnBoundBranchPausedScopeSilent`](../../../internal/check/isolation_escape_test.go) — fixture: open → pause → AI commit on bound branch → zero findings.

### AC-6 — Human cherry-pick (committer != actor + marker) stays silent

When a human runs `git cherry-pick -x <ai-sha>` to land the AI's commit on a different branch, the resulting commit carries the original AI's trailers (so it looks like an escape) but the committer flipped to the human and the body carries the `(cherry picked from commit <sha>)` marker. Both signals together = sovereign re-author; the rule suppresses the finding.

The rule receives the cherry-pick set via the `cherryPicked map[string]bool` parameter on `RunIsolationEscape`. The gather layer is responsible for identifying cherry-picks; the rule trusts the gather signal.

**Pinned by:**
- [`TestIsolationEscape_AC6_CherryPickReAuthorSilent`](../../../internal/check/isolation_escape_test.go) — fixture: AI commit on `main` (would normally fire) is flagged via `cherryPicked` → zero findings.
- [`TestIsolationEscape_AC6_NonCherryPickStillFires`](../../../internal/check/isolation_escape_test.go) — lower-bound guard: nil/empty `cherryPicked` does NOT silently suppress everything. A regression that treated missing info as "is a cherry-pick" would convert the rule to a no-op; this guard catches that class.

Sabotage-verified by dropping the suppression block — the AC-6 cherry-pick test fires.

**Limitation:** the gather-side implementation (CLI: parse `git log --format=%ce/%B`, compare committer vs. expected actor email, regex-check the body for the cherry-pick marker) is NOT in this milestone. The rule-side seam is fully tested; the CLI seam passes `nil` for `cherryPicked` from `RunProvenanceCheck`. AC-6 suppression therefore fires under test fixtures but not against real git history yet. Filed as [G-0202](../../gaps/G-0202-isolation-escape-cherry-pick-gather-side-implement-cli-detection.md) — a follow-up that completes the end-to-end seam.

### AC-7 — Human merge of epic/X into main (first-parent) stays silent

When a human merges `epic/E-0001-engine` into `main` via `git merge --no-ff`, two kinds of commits land:

- The **merge commit** itself is human-actor — the rule's `strings.HasPrefix(actor, "ai/")` filter skips it.
- The **AI commits behind the merge** are reachable from `epic/E-0001-engine` via first-parent, not from `main` via first-parent (that's the `--no-ff` semantic). The oracle returns `epic/E-0001-engine` for them; they match the bound branch → silent.

Both paths handled by the existing algorithm; no new code for AC-7.

**Pinned by:** [`TestIsolationEscape_AC7_HumanMergeFirstParentSilent`](../../../internal/check/isolation_escape_test.go) — multi-commit fixture with the merge commit + AI work commits, oracle reflects first-parent semantics, zero findings.

### AC-8 — Violating commit amended with aiwf-force + human/ actor stays silent

When the operator amends a violating commit with `git commit --amend --trailer 'aiwf-force: <reason>'` AND flips `aiwf-actor:` to `human/<id>`, the rule's `ai/` prefix filter skips it. The aiwf-force trailer is the audit signal that records the sovereign override; the rule does not need to inspect it because the actor filter already excludes the commit from consideration. The companion provenance rule `provenance-force-non-human` independently enforces that `aiwf-force:` requires a `human/` actor, so a tampered amend (keep `ai/` + add force) would surface there, not as a false-negative here.

**Pinned by:** [`TestIsolationEscape_AC8_ForceAmendedCommitSilent`](../../../internal/check/isolation_escape_test.go) — fixture: amended commit on `main` (would be an escape) carries `human/peter` + `aiwf-force` → zero findings.

### AC-9 — AI commit on entity with no opened scope stays silent

When an AI-actor commit is made on an entity that has no `aiwf-scope: opened` event in the inspected commit window, the rule is silent. This handles the bootstrap case (the entity has no scope ever opened) and the post-merge case (the gather window doesn't include the opener). The rule polices branch-binding violations, not "AI commit without authorization" — the companion provenance rule `provenance-no-active-scope` handles the latter.

**Pinned by:** [`TestIsolationEscape_AC9_NoScopeOpenedSilent`](../../../internal/check/isolation_escape_test.go) — fixture: AI commit on an entity (E-0002) with no opener in the commit list → zero findings.

### AC-10 — Per-commit firing — one finding per violating commit

When multiple AI commits violate the binding, the rule fires ONE finding per commit — not an aggregate per entity. The user wants the cardinality so each escaped commit is individually addressable (e.g. `git rebase -i` per-commit amends to add `aiwf-force`).

**Pinned by:**
- [`TestIsolationEscape_AC10_PerCommitFiring`](../../../internal/check/isolation_escape_test.go) — three violating commits → three findings, each mentioning its own SHA in the message.
- Implicit anchor in [`TestIsolationEscape_AC1_AICommitOnMainFires`](../../../internal/check/isolation_escape_test.go) — `len(findings) != 1` would fail if the algorithm aggregated.

### AC-11 — Warning severity; check exits 0 with findings reported

The finding is `SeverityWarning` (not `SeverityError`). `aiwf check`'s exit-code mapping (see `internal/check/check.go` and the CLI layer) exits 0 when only warnings are present; the warning surfaces in stdout/JSON but does not block pre-push. A future tightening to error severity is a D-NNN decision after one epic of usage; this milestone does not pre-commit the timing.

**Pinned by:** [`TestIsolationEscape_AC11_WarningSeverityCheckExitsZero`](../../../internal/check/isolation_escape_test.go) — explicit `Severity == SeverityWarning` assertion. The single anchor point for a future severity flip; updating it (when the flip happens) is deliberate.

### AC-12 — Finding hint text names both override paths

The hint text in [`internal/check/hint.go`](../../../internal/check/hint.go) names both sovereign override paths an operator can take when they hit `isolation-escape`:

1. **Cherry-pick re-author** — `git cherry-pick -x <sha>` preserves the marker and changes the committer; the rule then suppresses (AC-6).
2. **Force amend** — `git commit --amend --trailer 'aiwf-force: <reason>'` plus `aiwf-actor: human/<id>` records the sovereign override; the rule then suppresses (AC-8).

The hint also points at the epic body's "Sovereign override surface" section so an operator following the hint lands on a single place that documents the audit trail each path produces.

**Pinned by:** [`TestIsolationEscape_AC12_HintTextNamesBothOverridePaths`](../../../internal/check/isolation_escape_test.go) — asserts 4 markers in the hint (`cherry-pick -x`, `aiwf-force`, `human/`, `Sovereign override surface`). Sabotage-verified by replacing the hint with a placeholder — all 4 markers missing → test fires on each.

### AC-13 — Typed Code descriptor lands in internal/check/ per G-0129 pattern

The finding's `CodeIsolationEscape` descriptor lives at [`internal/check/isolation_escape.go`](../../../internal/check/isolation_escape.go) as a typed `codes.Code` value with `ID: "isolation-escape"` and the new `Class: codespkg.ClassBranchChoreography`. The class is added to [`internal/codes/codes.go`](../../../internal/codes/codes.go) as a third enum value (distinct from `ClassStructural` and `ClassLegality`) — the layer-4 carve-out per ADR-0011 that lets consumers enumerate branch-policing findings independently of structural integrity / legality codes. M-0106 is the first member of the new class; future branch-choreography findings declare themselves the same way.

**Pinned by:**
- [`TestIsolationEscape_AC13_TypedCodeDescriptor`](../../../internal/check/isolation_escape_test.go) — asserts `CodeIsolationEscape.ID == "isolation-escape"` and `Class == ClassBranchChoreography`. Sabotage-verified by flipping Class to `ClassStructural` → test fires.
- [`TestIsolationEscape_AC13_ClassBranchChoreographyDistinct`](../../../internal/check/isolation_escape_test.go) — asserts pairwise distinctness of the three Class enum values; a regression that collides them fires the test. Sabotage-verified by explicit collision (`ClassBranchChoreography Class = ClassStructural`) → test fires.
- [`TestRunProvenanceCheck_AC13_IsolationEscapeWired`](../../../internal/cli/check/isolation_escape_test.go) — AST-level assertion that `RunProvenanceCheck` contains a call to `check.RunIsolationEscape`. Sabotage-verified by dropping the wire-up → test fires.

## Work log

The milestone was wrapped once at commit `a44999fb` (initial wrap), then a retrospective subagent review (dispatched at the user's request) surfaced 10 findings — 5 blocking — that forced a reopen and 3 follow-up fix-cycles. The honest record below covers all 8 cycles (4 original + retrospective + 3 fix).

### Cycle 1 — AC-13 scaffold

Implementation landed at commit `ffa5c76f`. Added `ClassBranchChoreography` enum value to `internal/codes/codes.go`; created `internal/check/isolation_escape.go` with the typed `CodeIsolationEscape` descriptor, the `BranchOracle` interface, and a skeleton `RunIsolationEscape` that returns nil. Wired `RunIsolationEscape` through `RunProvenanceCheck` at `internal/cli/check/provenance.go` with a **nil oracle** — this was the F-1 mistake (see below). 3 tests pin AC-13. No reviewer pass.

### Cycle 2 — AC-1 + AC-3 + AC-4 + AC-9 (core algorithm)

Implementation landed at commit `5a51530c`. Built the per-commit algorithm: filter to AI commits with entity trailers, walk back through chronologically-prior commits to find the most recent `aiwf-scope: opened` event on the same entity, read its `aiwf-branch:`, compare to the oracle's reported branch set. Skip if no scope (AC-9), if scope has no `aiwf-branch:` (legacy pre-M-0102), if oracle returns unknown, or if commit rides bound branch (AC-4). Otherwise fire one finding. 8 tests. No reviewer pass.

### Cycle 3 — AC-2 + AC-5 + AC-10 + AC-11 + AC-12 (variants + mechanical + hint)

Implementation landed at commit `03ceea8b`. AC-2 (different ritual branch) and AC-5 (paused scope) satisfied by the existing algorithm without code change. AC-10 (per-commit firing) pinned by a 3-violation fixture. AC-11 (warning severity) anchored. AC-12 (hint text) added the canonical hint to `internal/check/hint.go`. No reviewer pass.

### Cycle 4 — AC-6 + AC-7 + AC-8 (suppression cases)

Implementation landed at commit `b9d35376`. AC-7 and AC-8 are natural-behavior pins (no code change). AC-6 adds a `cherryPicked map[string]bool` parameter to `RunIsolationEscape`; commits with SHAs in the set are suppressed. CLI wires `nil` for cherryPicked — the second half of the F-1 mistake. Initial wrap at commit `a44999fb`. **No reviewer pass at any of the 4 cycles** — this is what triggered the retrospective.

### Retrospective review (subagent)

User asked whether a reviewer subagent had been dispatched on the wrap. The honest answer was no — M-0106 was the largest milestone of the epic and got zero reviewer passes. User dispatched a retrospective. The subagent (path: `Agent` tool, agentType `reviewer`, model sonnet) returned 10 findings — 5 blocking, 1 fix-in-place, 2 track-for-later, 1 gap-worthy, 1 about the rationalization itself. Verdict: **request-changes; milestone cannot remain `done` while claiming "watertight."**

The 5 blocking findings:

- **F-1**: The CLI wires `RunIsolationEscape(commits, nil, nil)` — `oracle == nil` early-returns at line 105, emitting zero findings in production. G-0202 covered the cherry-pick half of the gather-side; the oracle half had no gap filed. The rule was shipped-disabled. *Caught by the retrospective only because the reviewer specifically inspected the call site.*
- **F-2**: Three reachable branches of `RunIsolationEscape` were misclaimed as covered in the original wrap body. Actual coverage was 7/10 arms, not 9.
- **F-3**: The implementation did not track `aiwf-scope-ends:` events. After the entity reached terminal status, the rule still used the dead opener's binding — a known false-positive class shipped without a gap, contradicting the spec's pre-decided design at line 86.
- **F-4**: `TestIsolationEscape_AC11_WarningSeverityCheckExitsZero` claimed an end-to-end assertion the unit test could not make. It passed for trivial reasons (because F-1 means findings is always empty, exit-code is trivially 0).
- **F-10**: The original wrap's Reviewer notes section minimized the gap with the rationalization *"sabotage probes were exhaustive — 7 total."* The retrospective found 4-5 issues the rationalization claimed wouldn't surface.

Non-blocking findings: F-5 (AC-3 test indistinguishable from AC-1 — documentation pin only), F-6 (cherryPicked parameter rationale was post-hoc; real reason was avoiding `scope.Commit` churn), F-7 (AC-12 hint test is circular tautology — same author wrote hint, substrings, and sabotage probe), F-8 (AC-13 wire-up test is AST-only; depends on F-1 fix to have a real seam test), F-9 (BranchOracle conflates "lookup failed" with "no branches" — silent escape risk).

### Cycle 5 — F-1 + F-8 fix (oracle wire-up + end-to-end seam tests)

Implementation landed at commit `5b413e22`. Built `gitBranchOracle` at `internal/cli/check/isolation_escape_oracle.go` — lists local refs via `git for-each-ref refs/heads/`, filters to main + ritual shapes via `branchparse`, runs `git rev-list --first-parent` per surviving branch, indexes sha → branch-list. Wired into `RunProvenanceCheck` replacing the nil oracle. Added two end-to-end integration tests at `internal/cli/check/isolation_escape_test.go`: fires-on-violating-commit (real fixture: authorize-bound-to-epic-branch + AI commit on main → fires through full chain) and silent-on-bound-branch-commit (symmetric). Sabotage-verified by re-disabling the oracle — the seam test fires on missing finding.

### Cycle 6 + 7 — F-2, F-3, F-4, F-5, F-7 (scope-end + uncovered branches + test/doc tightening)

Implementation landed at commit `45b4896f`. Extended `openerRecord` to carry `endedAt` (chrono position of first `aiwf-scope-ends: <opener-sha>` trailer; -1 = never ended). Per-AI-commit search now predicates "active at C's time" = opened-before AND (never-ended OR ended-after) per spec line 86. Mirrors `provenance.go`'s `buildEndedAtIndex` pattern. Sabotage-verified by dropping the end-check — `TestIsolationEscape_F3_AICommitAfterScopeEndedSilent` fires.

Three new tests for the uncovered F-2 arms: empty-entity opener skip, AI-actor authorize-verb skip (sabotage-verified), AI commit predating every opener.

Renamed `TestIsolationEscape_AC11_WarningSeverityCheckExitsZero` → `TestIsolationEscape_AC11_SeverityIsWarning` per F-4. The exit-code half is now genuinely pinned by Cycle 5's CLI seam tests.

Tightened AC-3 docstring per F-5 to call out the documentation-pin nature explicitly. Added F-7 caveat to AC-12 docstring about the circular-test shape.

### Cycle 8 — F-6 + F-9 (decision + gap)

Filed **D-0017** ([`work/decisions/D-0017-...md`](../../decisions/D-0017-isolation-escape-cherrypicked-param-shape.md)) recording the `cherryPicked` parameter shape decision with the **honest** rationale (the wrap-body's "rule-specific" + "would touch every consuming rule" claims were rationalization; the real reason was "avoiding scope.Commit churn was the path of least diff"). Names the conditions for revisiting (second consumer or 3+ nil args at a call site).

Filed **G-0203** ([`work/gaps/G-0203-...md`](../../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md)) for F-9: `FirstParentBranches` conflates lookup-failed with no-branches, allowing silent escape on hostile-named branches.

## Decisions made during implementation

- **`ClassBranchChoreography` as a new enum value** (per pre-implementation Q&A). Distinct from `ClassStructural` and `ClassLegality` so consumers can enumerate branch-policing findings as their own kernel layer (ADR-0011 layer-4 carve-out). Spec-cell elaboration is parked at M-0158.
- **`cherryPicked` parameter shape recorded as [D-0017](../../decisions/D-0017-isolation-escape-cherrypicked-param-shape.md)** (per the retrospective F-6). Three plausible shapes were considered; option 3 (separate `map[string]bool` parameter) shipped. The honest rationale: avoiding `scope.Commit` churn was the path of least diff. The future-maintenance debt is named in D-0017.
- **`BranchOracle` shipped with eager-build, in-memory index** (per Cycle 5). One `git for-each-ref` + one `git rev-list --first-parent` per ritual branch at construction time. The alternative — per-call `git merge-base --is-ancestor` — would have been O(violations × N) at check time, slower for the common silent case.
- **`BranchOracle` construction failure is non-fatal** (per Cycle 5). If `newGitBranchOracle` errors, the rule degrades to silent. Branch-policing is one rule among many; a single-rule failure should not block the entire check pass. The downstream visibility is named in [G-0203](../../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md).
- **Scope-end tracking via existing `aiwf-scope-ends:` mechanism** (per Cycle 6 / F-3). The `endedAt` slot on `openerRecord` is populated from a first sub-pass mirroring `provenance.go`'s `buildEndedAtIndex`. The kernel's one-scope-per-entity-at-a-time semantic means the most-recent-preceding opener determines the binding; if it ended, there is no binding.
- **Per-commit firing, not aggregated** (AC-10).
- **Severity = warning, not error** (AC-11). First land. Tightening to error is a D-NNN decision after one epic of usage. The single test at the anchor point makes the future flip a deliberate edit.
- **The rule polices AI-actor commits only**. Per the ADR-0010 sovereignty principle.
- **AC-6 cherry-pick gather-side deferred to G-0202** (the original deferral) PLUS the gather-side is wired with `nil cherryPicked` from `RunProvenanceCheck` so AC-6 suppression fires only under test fixtures, not against real git history. The rule-side seam is complete; the production end-to-end seam is partial. This was the half of F-1 that G-0202 already covered before the retrospective.

## Validation

- `go test -race -parallel 8 ./...` — green across all packages (verified at each cycle commit).
- `go build ./...` — green.
- `aiwf check` — 0 errors, 0 warnings (post-retrospective wrap).
- Sabotage probes (Cycles 1–8 combined): **10 single-line regressions**, each caught by at least one test. Cycles 5–7 added the F-1 oracle-disabled sabotage, the F-3 scope-end-dropped sabotage, and the F-2-arm-2 authorize-verb-skip sabotage on top of the original 7.
- Branch coverage **measured**, not claimed: 96%+ on `RunIsolationEscape` post-Cycles 5–7. The earlier wrap's "branch-coverage hard rule satisfied" claim is now true.
- End-to-end seam tests at `internal/cli/check/isolation_escape_test.go`: the rule fires through the full `RunProvenanceCheck` chain against fixture repos, and stays silent on bound-branch fixtures. The F-1 failure mode (rule no-op in production) is structurally prevented.
- `wf-doc-lint` scoped to the changeset: clean.
- Trailer hygiene: no `aiwf-verb: feat` on implementation commits.

## Deferrals

- [G-0202](../../gaps/G-0202-isolation-escape-cherry-pick-gather-side-implement-cli-detection.md) — cherry-pick gather-side (committer-vs-actor + body marker derivation). Pre-retrospective filing. AC-6 suppression fires under test fixtures via the `cherryPicked` parameter, but the CLI wires `nil` until this lands. Rule-side seam complete; production end-to-end seam partial.
- [G-0203](../../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md) — `BranchOracle.FirstParentBranches` conflates "lookup failed" with "no branches." Both currently return an empty slice, both currently silence. Filed during Cycle 8 (F-9 from the retrospective). Address when the first incident traces back to oracle-silent-on-failure, or when M-0158's spec-cell consolidation surfaces a richer reachability semantic.

## Reviewer notes

**Retrospective review (subagent, post-initial-wrap):** 10 findings — 5 blocking (F-1, F-2, F-3, F-4, F-10), 4 fix-in-place / track-for-later (F-5, F-6, F-7, F-8), 1 gap-worthy (F-9). The milestone was reopened, the 5 blocking findings were fixed in-milestone (Cycles 5–7), the 2 architectural concerns were captured as D-0017 (F-6) and G-0203 (F-9), and the tracking-only findings (F-5, F-7) became code-comment caveats on the relevant tests. F-8 was effectively dual-resolved — the AST-level wire-up test stays as defense in depth, AND the real end-to-end seam test landed in Cycle 5.

**Third-pass retrospective review (subagent, max-effort opus pass post-N-fixes):** the user requested a third pass with max effort to find what the prior reviewers missed. The reviewer ran empirical measurements (coverage, lint, integration tests) and probed real-world environments (sparse checkouts, worktrees, packed-refs, force-pushes, etc.). Verdict: **APPROVE-WITH-FOLLOW-UPS** — 8 new findings T-1..T-8, **none blocking**. The pattern shifted from substantive defects (first two passes) to minor doc-tightening and test-completeness items.

Disposition of T-1..T-8 (all addressed at commit `d2014aa8`):

- **T-1**: Added `TestRunProvenanceCheck_IsolationEscape_FindingCarriesHint` pinning the AC-12 hint flow through the production `Run` → `ApplyHintsLikeRun` composition (asserts non-empty Hint plus both override-path markers).
- **T-2**: D-0017's brittle "line 257" reference replaced with structural section name.
- **T-4**: `aiwfx-start-milestone` SKILL.md "once M-0106 ships" updated to present tense.
- **T-5**: `aiwfx-start-epic` SKILL.md gained a symmetric one-line mention of M-0106's post-hoc finding in the Principles section.
- **T-6**: Added `TestIsolationEscape_T6_BoundBranchDoesNotExistSilent` documenting the natural-semantic behavior on bound-branch-typo case.
- **T-7**: `TestRunProvenanceCheck_IsolationEscape_FiresOnViolatingCommit` tightened from "at least one finding" to "exactly one finding" via filter-then-count idiom matching `WarningDoesNotMarkErrors`.
- **T-8**: `TestIsolationEscape_NilOracleSilent` docstring rewritten to reflect post-F-1 production behavior.

The reviewer's verdict explicitly noted that the milestone "can stay `done` as-is" — none of T-1..T-8 was blocking. The choice to address all 8 reflects the operator's watertight standard rather than reviewer demand.

**Second-pass retrospective review (subagent, post-fix-wrap):** to verify the fixes hold, a second reviewer pass ran against the fix range (`a44999fb..afc19709`). Verdict: 10 of 10 original findings genuinely closed. One CI-blocking issue surfaced (N-1: gofumpt violation in the new fixture) plus three follow-ups:

- **N-1** (gofumpt blocker) — fixed in this round; `internal/check/isolation_escape_test.go` and a pre-existing `internal/policies/aiwfx_start_epic_test.go` violation both cleaned. `golangci-lint run ./...` now reports 0 issues.
- **N-2** (exit-code seam under-pinned) — added [`TestRunProvenanceCheck_IsolationEscape_WarningDoesNotMarkErrors`](../../../internal/cli/check/isolation_escape_test.go) which drives the production fixture and asserts `check.HasErrors` returns false over the isolation-escape findings — `HasErrors` being the predicate the CLI's exit-code mapping at `internal/cli/check/check.go:195` consumes. The AC-11 docstring updated to cite the new test by name rather than wave at "CLI seam tests."
- **N-3** (duplicate-scope-ends-skip arm uncovered) — added [`TestIsolationEscape_N3_DuplicateScopeEndsFirstWins`](../../../internal/check/isolation_escape_test.go) with a fixture that distinguishes "first wins" from "last wins" outcomes. Sabotage-verified by inverting the skip to overwrite — test fires on a regression that would change semantic behavior, not just statement coverage.
- **N-4** (oracle fail-shut on any single failed ref) — captured as a sub-concern in G-0203, since the gather-side fault-tolerance question is part of the same surface as F-9's typed-error split.

Both N-2 and N-3 are now sabotage-verified pins, not just decoy tests. The second-pass reviewer flagged the original wrap's "every reachable arm has at least one test" claim as slightly overstated; with N-3 in place that claim is now accurate.

**The original wrap's Reviewer notes section was misleading.** It claimed "sabotage probes were exhaustive — 7 total" as substituting for a reviewer pass. The retrospective surfaced 4-5 issues the sabotage probes did not catch (most notably F-1: the rule was no-op in production). This section is the corrected record.

**Branch-coverage hard rule is now satisfied** with measurement, not assertion. Coverage on `RunIsolationEscape` runs >96% with the F-2 / F-3 tests in place. Every reachable arm of the algorithm has at least one test, including the arms the original wrap misclaimed as covered (empty-entity opener skip, AI-actor authorize-verb skip, predates-opener arm, scope-ended arm).

**End-to-end seam is now real, not AST-only.** The `internal/cli/check/isolation_escape_test.go` integration tests drive `RunProvenanceCheck` against fixture repos and assert the finding surfaces (or doesn't) through the full chain. The original AST-only wire-up test (`TestRunProvenanceCheck_AC13_IsolationEscapeWired`) remains as a defense-in-depth assertion against future call-site removal.

**What this milestone now ships:**

- An algorithmically-complete kernel rule that fires correctly on real git history.
- A git-backed oracle implementation with documented degradation behavior (graceful on construction failure, silent on unknown branch).
- Scope-end tracking that aligns the rule with the spec's pre-decided "active at C's time" predicate.
- A documented gather-side limitation (G-0202: cherry-pick suppression fires only in tests until the CLI committer-vs-actor derivation lands).
- A documented oracle-semantics limitation (G-0203: lookup-failed vs no-branches conflation, address when first incident demands).
- An honest decision record (D-0017) of the `cherryPicked` parameter shape.

**E-0030 epic-wrap** will pull this milestone's `CodeIsolationEscape` into the closure summary; the typed-code shape (per G-0129) means a future audit catalog can enumerate `ClassBranchChoreography` findings without source grepping. The honest retrospective lives in this file; the next milestone author reads "discipline matters — don't skip the reviewer pass even when sabotage probes look exhaustive."

**Lesson for future milestones in this epic and beyond:** sabotage probes pin "this specific regression class is caught." Reviewer passes find regressions the implementer didn't anticipate. The two are complementary, not substitutes. The original wrap's rationalization was wrong; this retrospective demonstrated why.

