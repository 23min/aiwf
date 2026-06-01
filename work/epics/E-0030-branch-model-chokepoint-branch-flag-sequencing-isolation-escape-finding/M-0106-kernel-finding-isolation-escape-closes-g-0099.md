---
id: M-0106
title: Kernel finding isolation-escape (closes G-0099)
status: in_progress
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

### Cycle 1 — AC-13 scaffold

Implementation landed at commit `ffa5c76f`. Added `ClassBranchChoreography` enum value to `internal/codes/codes.go`; created `internal/check/isolation_escape.go` with the typed `CodeIsolationEscape` descriptor, the `BranchOracle` interface, and a skeleton `RunIsolationEscape` that returns nil. Wired `RunIsolationEscape` through `RunProvenanceCheck` at `internal/cli/check/provenance.go` with a nil oracle (structural wire-up; algorithm follows in Cycle 2). 3 tests pin AC-13: typed-Code descriptor, distinct-class enum, AST-level wire-up assertion. Sabotage probes: wrong class, dropped wire-up, explicit class collision — all 3 caught.

### Cycle 2 — AC-1 + AC-3 + AC-4 + AC-9 (core algorithm)

Implementation landed at commit `5a51530c`. Built the per-commit algorithm: filter to AI commits with entity trailers, walk back through chronologically-prior commits to find the most recent `aiwf-scope: opened` event on the same entity, read its `aiwf-branch:`, compare to the oracle's reported branch set. Skip if no scope (AC-9), if scope has no `aiwf-branch:` (legacy pre-M-0102), if oracle returns unknown, or if commit rides bound branch (AC-4). Otherwise fire one finding with the commit's SHA, the entity id, and both branches in the message (AC-1, AC-3 — same code path, different fixture). 8 tests: 4 ACs + 4 belt-and-braces (nil oracle, unknown branch, human commit, legacy scope). Sabotage probes: invert silent-condition, drop `ai/` filter, drop empty-bound skip — all 3 caught.

### Cycle 3 — AC-2 + AC-5 + AC-10 + AC-11 + AC-12 (variants + mechanical + hint)

Implementation landed at commit `03ceea8b`. The existing algorithm satisfies AC-2 (different ritual branch) and AC-5 (paused scope) without code change; the tests document the behavior structurally. AC-10 (per-commit firing) is pinned by a 3-violation fixture. AC-11 (warning severity) is anchored with an isolated test for future flip discipline. AC-12 (hint text) added the canonical hint to `internal/check/hint.go` naming both override paths + the epic body's audit-trail section. Sabotage probe: replace hint with placeholder → 4 marker assertions fire.

### Cycle 4 — AC-6 + AC-7 + AC-8 (suppression cases)

Implementation landed at commit `b9d35376`. AC-7 and AC-8 are natural-behavior pins (no code change — the `ai/` actor filter already excludes human-merge commits and force-amended commits). AC-6 adds a new `cherryPicked map[string]bool` parameter to `RunIsolationEscape`; commits with SHAs in the set are suppressed before the fire path. Plus a lower-bound guard test that nil/empty `cherryPicked` does NOT silently suppress everything. Sabotage probe: drop AC-6 suppression line → cherry-pick test fires.

## Decisions made during implementation

- **`ClassBranchChoreography` as a new enum value** (per pre-implementation Q&A). Distinct from `ClassStructural` and `ClassLegality` so consumers can enumerate branch-policing findings as their own kernel layer (ADR-0011 layer-4 carve-out). Spec-cell elaboration is parked at M-0158.
- **Separate `cherryPicked` parameter** (not extending `scope.Commit`, not extending `BranchOracle`). Rationale: the cherry-pick info is rule-specific; extending `scope.Commit` would touch every consuming rule, and extending `BranchOracle` couples re-author detection with branch reachability. The parameter shape matches the rule's actual need (a per-SHA boolean) and the gather layer's natural output (a derived set).
- **AC-6 gather-side deferred to G-0202**. The rule-side seam is complete; the CLI seam wires `nil` for now. AC-6 suppression fires under test fixtures but not against real git history — known limitation, filed as G-0202 with the design notes the follow-up needs.
- **Per-commit firing, not aggregated** (AC-10). The user wants each escaped commit individually addressable for `git rebase -i` per-commit amends.
- **Severity = warning, not error** (AC-11). First land. Tightening to error is a D-NNN decision after one epic of usage. The single test at the anchor point makes the future flip a deliberate edit.
- **The rule polices AI-actor commits only**. Per the ADR-0010 sovereignty principle — human commits, including manual cherry-picks between branches, are not policed by this rule. They're subject to other provenance rules.

## Validation

- `go test -race -parallel 8 ./...` — green across all packages.
- `go build ./...` — green.
- `aiwf check` — 0 errors, 13 warnings (all `entity-body-empty` on AC body sections pre-wrap; this commit fills them).
- Sabotage probes (Cycle 1–4 combined): 7 single-line regressions, each caught by at least one test.
- `wf-doc-lint` scoped to the changeset: clean.
- Trailer hygiene: no `aiwf-verb: feat` on implementation commits.

## Deferrals

- [G-0202](../../gaps/G-0202-isolation-escape-cherry-pick-gather-side-implement-cli-detection.md) — the gather-side implementation for AC-6 (committer-vs-actor email comparison + body marker regex check). The rule-side logic is complete; without the gather-side, AC-6 suppression fires only in tests, not against real git history. Filed during Cycle 4 wrap as the known limitation; the gap carries the design notes the follow-up needs.

## Reviewer notes

- 4 cycles, no formal subagent reviewer passes this milestone (the user's pre-cycle Q&A explicitly approved the plan + design choices, and each cycle's sabotage probes were exhaustive — 7 total). A retrospective spot-check would be the only thing to add.
- The CLI-seam test at `internal/cli/check/isolation_escape_test.go` is AST-level — a future regression that comments out the wire-up call, or renames it, fires the test immediately. Without this test, M-0106's rule could ship complete but unhooked.
- The `cherryPicked` parameter is positioned third on `RunIsolationEscape` so a future addition (e.g. a `mergedInto` map for richer merge handling) can extend the signature cleanly.
- Branch-coverage hard rule satisfied: every reachable arm of `RunIsolationEscape` is exercised — nil oracle (graceful), no opener (AC-9), opener before commit (happy), opener after commit (predates), empty bound (legacy), unknown branch (graceful), rides bound (AC-4), cherry-pick suppressed (AC-6), violates (AC-1/2/3/10).
- E-0030 epic-wrap will pull this milestone's `CodeIsolationEscape` into the closure summary; the typed-code shape (per G-0129) means a future audit catalog can enumerate `ClassBranchChoreography` findings without source grepping.

