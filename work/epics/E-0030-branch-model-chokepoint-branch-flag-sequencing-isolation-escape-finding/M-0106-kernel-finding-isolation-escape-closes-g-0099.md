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
      status: open
      tdd_phase: red
    - id: AC-7
      title: Human merge of epic/X into main (first-parent) stays silent
      status: open
      tdd_phase: red
    - id: AC-8
      title: Violating commit amended with aiwf-force + human/ actor stays silent
      status: open
      tdd_phase: red
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
      status: open
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

### AC-2 — AI commit on epic/Y while scope binds epic/X fires isolation-escape

### AC-3 — AI commit on worktree-vs-branch mismatch fires isolation-escape

### AC-4 — AI commit on epic/X while scope binds epic/X stays silent

### AC-5 — AI commit on epic/X while scope is paused stays silent

### AC-6 — Human cherry-pick (committer != actor + marker) stays silent

### AC-7 — Human merge of epic/X into main (first-parent) stays silent

### AC-8 — Violating commit amended with aiwf-force + human/ actor stays silent

### AC-9 — AI commit on entity with no opened scope stays silent

### AC-10 — Per-commit firing — one finding per violating commit

### AC-11 — Warning severity; check exits 0 with findings reported

### AC-12 — Finding hint text names both override paths

### AC-13 — Typed Code descriptor lands in internal/check/ per G-0129 pattern

