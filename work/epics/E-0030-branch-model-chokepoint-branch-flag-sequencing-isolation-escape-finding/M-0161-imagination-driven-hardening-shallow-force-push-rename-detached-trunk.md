---
id: M-0161
title: 'Imagination-driven hardening: shallow, force-push, rename, detached, trunk'
status: in_progress
parent: E-0030
tdd: required
acs:
    - id: AC-1
      title: aiwf authorize preflight respects configured trunk name
      status: open
      tdd_phase: red
    - id: AC-2
      title: Authorize preflight enforces ritual rung hierarchy
      status: open
      tdd_phase: red
    - id: AC-3
      title: BranchOracle typed errors with per-ref fault tolerance
      status: open
      tdd_phase: red
    - id: AC-4
      title: BranchOracle detects shallow clones; isolation-escape-shallow-clone fires
      status: open
      tdd_phase: red
    - id: AC-5
      title: Reflog-walk detects orphaned AI commits from force-push
      status: open
      tdd_phase: red
    - id: AC-6
      title: BranchOracle resolves renamed branches via SHA fallback
      status: open
      tdd_phase: red
    - id: AC-7
      title: Detached HEAD behavior pinned across preflight, oracle, check, doctor
      status: open
      tdd_phase: red
    - id: AC-8
      title: Kernel finding promote-on-wrong-branch enforces ADR-0010 ritual ordering
      status: open
      tdd_phase: red
    - id: AC-9
      title: 'Layer-4 spec-table refactor: mechanical-weight catalog + bijection enforcement'
      status: open
      tdd_phase: red
---
## Goal

Close the imagination-driven hardening gaps (G-0200, G-0201, G-0203, G-0204, G-0205, G-0206, G-0207, G-0209, G-0210) using the combinatorial real-git E2E framework M-0159 lands. These scenarios have no in-repo historical evidence — but per the user's M-0159 planning directive *"if we can imagine it, it will happen"*, coverage is mandatory because different operators have different workflows. The user has historically said N to force-push; other operators routinely say Y.

This is **Tier 3** in the E-0030 hardening evidence-priority split:

- **Tier 1 (M-0159)** — combinatorial framework + override convergence + seam coverage, all evidence-backed.
- **Tier 2 (M-0160)** — operational pain regressions, evidence-backed from this repo's incident history.
- **Tier 3 (M-0161, this milestone)** — imagination-driven coverage. No in-repo evidence, but the kernel principle "framework correctness must not depend on the LLM's behavior" requires mechanical pins for every scenario aiwf claims to handle.

## Context

The M-0158 honest-scope audit surfaced these as unmodeled real-world failure modes. The M-0159 history-mining investigation then categorized them as imagination-only (no in-repo evidence). The user's response on whether to drop them: *"don't remove the ones for which we don't have empirical evidence at this time. 'if we can imagine it, it will happen' (just because we didn't run into certain things may be because of the way I work, but other users work differently. I have often said N when asked to force push, for instance, but someone else might think it's OK)"*.

So all imagination-driven scenarios sequence into this milestone, fully covered via the M-0159 framework. The work is real even where the evidence is hypothetical.

## Scope

Gaps consumed (9 total):

- **G-0200** — preflight main-only carve-out hardcodes "main"; generalize to `aiwf.yaml.allocate.trunk`.
- **G-0201** — authorize preflight carve-out accepts cross-rung ritual mismatches; tighten hierarchical predicate.
- **G-0203** — BranchOracle.FirstParentBranches conflates lookup-failed with no-branches; typed errors + fail-shut decision.
- **G-0204** — BranchOracle silent on shallow clones (CI fetch-depth=1); detect + fail-shut or document fallback.
- **G-0205** — BranchOracle silent on force-pushed-away violating commits; reflog-walk or documented limitation.
- **G-0206** — BranchOracle false-positive on branch renames after authorize; reflog-walk for rename events.
- **G-0207** — Detached-HEAD handling untested in preflight and oracle; explicit error or supported path.
- **G-0209** — Ritual step ordering is advisory only; either kernel enforcement or remove the ordering claim from SKILL.md.
- **G-0210** — M-0158 spec table contains 9 documentation-only or duplicate cells; refactor catalog to mechanical-weight-only set.

## Dependencies

- **M-0159** (Tier 1) and **M-0160** (Tier 2) — both must complete first. M-0161 reuses M-0159's E2E framework and may depend on M-0160's reallocate-stress helpers.
- **G-0213** (cellcoverage landmine) — must be closed in M-0159 before any M-0161 rule reads `aiwf-branch` against a resolvability check.

## Out of scope

- New override paths beyond what M-0159 lands.
- Generalizing trunk config beyond named trunks (e.g., arbitrary "current ref is parent") — G-0200's scope is named-trunk only.
- Data-loss scenarios (G-0212) — future-epic.

## Acceptance criteria

<!--
AC seed set (to be allocated via `aiwf add ac` at start-milestone time, one per gap with combinatorial real-git E2E coverage required for each):

1. G-0200 — trunk-name configuration: hardcoded "main" → aiwf.yaml.allocate.trunk; real-git E2E with a non-default trunk name.
2. G-0201 — cross-rung carve-out hierarchical predicate; real-git E2E with epic-ritual + milestone-ritual interaction.
3. G-0203 — BranchOracle typed errors (lookup-failed vs no-branches); rule fails-closed on lookup error; real-git E2E.
4. G-0204 — shallow-clone handling: real-git E2E with git clone --depth=1; either detect + fail-shut or documented fallback path.
5. G-0205 — force-pushed history: real-git E2E with git push --force; either reflog-walk preserves the audit trail or documented limitation surfaces.
6. G-0206 — branch-rename handling: real-git E2E with git branch -m mid-scope; reflog-walk preserves the rename event.
7. G-0207 — detached-HEAD handling: real-git E2E with checkout <sha>; explicit error path or supported flow.
8. G-0209 — ritual step ordering: either kernel-side enforcement OR remove the ordering claim from SKILL.md. Per "no advisory-only floating" directive.
9. G-0210 — M-0158 spec-table catalog refactor: remove documentation-only cells, keep mechanical-weight cells only; structural meta-coverage redesign (branchcell.Pin registry with bijection enforcement) lands alongside.

These 9 are the seed set; aiwfx-start-milestone refines and allocates them.
-->

### AC-1 — aiwf authorize preflight respects configured trunk name

**Observable behavior.** In a repo whose `aiwf.yaml.allocate.trunk` resolves to a non-`main` short name (e.g. `refs/remotes/origin/master` → `master`, or any other operator-chosen trunk), `aiwf authorize <id> --to ai/<agent> --branch <ritual-future-branch>` invoked from a checkout on that trunk **succeeds** without `--force --reason` — the same way today's "main + ritual --branch" carve-out works for `main`-based repos. The verb-layer literal `"main"` at [`internal/verb/authorize.go:300`](../../../internal/verb/authorize.go) is replaced by a configured short-name derivation from [`Config.AllocateTrunkRef()`](../../../internal/config/config.go).

**Mechanical assertions:**

1. **Combinatorial real-git E2E.** A scenario fan-out under [`internal/cli/integration/authorize_scenarios_test.go`](../../../internal/cli/integration/authorize_scenarios_test.go) via the M-0159 framework exercises 4 trunk-name shapes:

   | Trunk shape | `allocate.trunk` value | Local trunk branch |
   |---|---|---|
   | Default | `refs/remotes/origin/main` | `main` |
   | GitHub-classic | `refs/remotes/origin/master` | `master` |
   | Operator-chosen | `refs/remotes/origin/dev` | `dev` |
   | Bare-heads | `refs/heads/trunk` | `trunk` |

   Each scenario: bootstrap temp repo with the named `aiwf.yaml` + local branch; run `aiwf authorize <epic-id> --to ai/alice --branch epic/E-NNNN-foo` from that trunk checkout against the worktree-built binary; assert exit 0 + `aiwf-branch: epic/E-NNNN-foo` trailer.

2. **Sabotage-verifiable.** Reverting the call-site change at `internal/verb/authorize.go:300` (restoring the literal `"main"`) makes all 4 scenarios fire for trunks named other than `main` — proving the integration test discriminates the new code path.

3. **Auxiliary unit test.** `Config.TrunkBranchShortName()` table-driven test covers the canonical shapes (`refs/remotes/<remote>/<name>` → `<name>`; `refs/heads/<name>` → `<name>`) plus degenerate / unparseable cases (empty in, empty out — no panic). Diagnostic, not load-bearing — the E2E is the test set that pins behavioral correctness end-to-end.

4. **Branch-spec cell registration.** 4 positive cells in `internal/workflows/spec/branch/` covering the trunk-name shapes. AC-9 (G-0210) consolidates the catalog.

**Edge cases:**

- Bare-name trunk refs (`refs/heads/master`, not `refs/remotes/origin/master`) parse correctly via the same helper — single source of truth, no fork for "local trunk" vs "tracking trunk" semantics.
- Empty / unparseable `allocate.trunk` → `TrunkBranchShortName()` returns `""` → the carve-out predicate's left arm fails → preflight falls through to the existing implicit-ritual-current path. No regression on malformed config.
- Operator on a non-trunk branch (regardless of trunk name) → preflight uses the ritual-current path; this AC does not touch that arm.
- The trunk-name short helper is a pure derivation; it does not query git. The config is the single source of truth, consistent with the rest of `internal/config/`.

**References.**

- [G-0200](../../gaps/G-0200-preflight-main-only-carve-out-generalize-to-trunk-name-from-aiwf-yaml.md) — the gap; closes here
- [`internal/verb/authorize.go:300`](../../../internal/verb/authorize.go) — the hardcoded `"main"` call site
- [`internal/config/config.go`](../../../internal/config/config.go) — where `AllocateTrunkRef()` already lives and where the new short-name helper lands
- [`internal/branchparse/`](../../../internal/branchparse/) — the package the call site already consumes for ritual-branch shapes
- M-0104 — the milestone whose Cycle 1 reviewer flagged this layering smell

### AC-2 — Authorize preflight enforces ritual rung hierarchy

**Observable behavior.** `aiwf authorize <id> --to ai/<agent> --branch <ritual-future>` accepts only `(CurrentBranch rung, --branch rung)` pairs from the legal set. Illegal pairs refuse with an actionable error naming the offending shapes and the sovereign override path (`--force --reason "..."`).

**Rung-pair matrix (4 legal + 12 illegal = 16 cells):**

| Current rung | Target rung | Legal? | Notes |
|---|---|---|---|
| trunk | epic | ✅ | `aiwfx-start-epic` from trunk |
| epic | milestone | ✅ | `aiwfx-start-milestone` from epic |
| milestone | patch | ✅ | `wf-patch` from milestone |
| epic | patch | ✅ | `wf-patch` from epic, milestone-skipping |
| trunk | trunk | ❌ | `--branch trunk` from trunk (AI on trunk — verboten) |
| epic | trunk | ❌ | `--branch trunk` from epic |
| milestone | trunk | ❌ | `--branch trunk` from milestone |
| patch | trunk | ❌ | `--branch trunk` from patch |
| trunk | milestone | ❌ | rung-skip |
| trunk | patch | ❌ | rung-skip×2 |
| epic | epic | ❌ | cross-epic typo |
| milestone | milestone | ❌ | cross-milestone typo |
| patch | patch | ❌ | cross-patch typo |
| milestone | epic | ❌ | up-the-tree |
| patch | milestone | ❌ | up-the-tree |
| patch | epic | ❌ | up-the-tree, skipping milestone |

**Mechanical assertions:**

1. **Combinatorial real-git E2E.** A scenario fan-out under [`internal/cli/integration/authorize_scenarios_test.go`](../../../internal/cli/integration/authorize_scenarios_test.go) exercises all 16 (CurrentBranch rung, --branch rung) combinations as separate scenarios via the M-0159 `RunScenarios` framework. Each scenario bootstraps a temp repo, checks out the relevant current-branch shape, runs `aiwf authorize <id> --to ai/<agent> --branch <target>` against the worktree-built binary, asserts:
   - **4 legal pairs**: exit 0; authorize commit lands with `aiwf-branch:` trailer naming the target; no `aiwf-force:` present.
   - **12 illegal pairs**: exit non-zero; stderr names both the `CurrentBranch` rung and the `--branch` rung; stderr names the `--force --reason "..."` override path; no commit is produced.

2. **One sovereign-override E2E.** A single additional scenario exercises an illegal pair (e.g., epic → epic) plus `--force --reason "cross-epic intentional"` → exit 0; the authorize commit carries both `aiwf-branch:` (the target) AND `aiwf-force:` (the reason). Pins the sovereign-override surface for this AC's gate, per the epic's "override gated, audited, last-resort" commitment.

3. **Sabotage-verifiable.** Reverting the rung-check at the carve-out site restores flat-union acceptance; all 12 illegal-pair scenarios fire on the missing refusal.

4. **Branch-spec cell registration.** Each of the 16 rung-pair scenarios registers as a named cell (4 positive + 12 negative) in `internal/workflows/spec/branch/`, plus 1 override cell = 17 cells. AC-9 (G-0210) consolidates the catalog; the cell-coverage drift policy then enforces that each cell has its paired E2E scenario.

5. **Auxiliary unit tests.** `branchparse.RungOf` and `branchparse.LegalRungPair` get unit tests for the pure derivation shape (canonical `"epic"`/`"milestone"`/`"patch"`/trunk-short-name/`""`). These are diagnostic — they catch a bad helper edit fast at unit level — not the load-bearing evidence; the E2E is what pins behavioral correctness.

**Edge cases:**

- Trunk-name composition with AC-1: the trunk rung is derived from `Config.TrunkBranchShortName()` per AC-1's helper. So `master` (or any other configured trunk name) maps to the `"trunk"` rung. This AC builds on AC-1; AC-1 must land first.
- Unparseable `--branch` (not matching any ritual shape) → existing rule refuses with the original `branch-not-found` / "must be a ritual branch" error. This AC does not collapse that path.
- Detached HEAD → out of scope (AC-7 / G-0207 owns it). The current-rung check here is gated on a parseable symbolic-ref result.
- `--force --reason "..."` bypasses the rung-pair check; the authorize commit then carries `aiwf-force:` per the existing kernel pattern.
- The `(epic, patch)` legal pair encodes the deliberate "patch on epic" shape (a wf-patch cut from the epic branch without an intermediate milestone). The `(milestone, patch)` pair encodes "patch on milestone". Both are legal because both are operator-intentional; neither is a typo class.

**References.**

- [G-0201](../../gaps/G-0201-authorize-preflight-carve-out-accepts-cross-rung-ritual-mismatches.md) — the gap; closes here
- [`internal/verb/authorize.go`](../../../internal/verb/authorize.go) — the carve-out call site
- [`internal/branchparse/`](../../../internal/branchparse/) — where `RungOf` and `LegalRungPair` land
- M-0105 — the milestone whose Cycle 1 reviewer flagged this looseness
- AC-1 (G-0200) — compose: trunk-name derivation via `Config.TrunkBranchShortName()`

### AC-3 — BranchOracle typed errors with per-ref fault tolerance

**Observable behavior.** `BranchOracle` distinguishes "looked, found nothing" from "tried, lookup failed." Lookup failures accumulate at construction time and surface as a separate advisory finding code [`isolation-escape-oracle-failure`](../../../internal/check/) so silent escapes are mechanically impossible. Per-ref fault tolerance: a single stale or corrupt ref no longer disables the rule for the whole repo — the oracle skips that ref, records the error, proceeds for every other ref that resolved cleanly.

**New oracle API shape:**

```go
// FirstParentBranches returns the ritual branches whose first-parent
// chain reaches sha. An empty/nil return means the commit is on zero
// ritual branches the oracle knows about — KNOWN-GOOD empty, not
// conflated with lookup failure (those go to OracleErrors).
FirstParentBranches(sha string) []string

// OracleErrors returns lookup failures accumulated during construction.
// Empty slice ↔ every ritual ref resolved cleanly. Non-empty slice ↔
// at least one ref's first-parent index could not be built; the rule's
// coverage is incomplete for those refs and the consumer should
// surface an isolation-escape-oracle-failure advisory.
OracleErrors() []OracleErr   // typed; each carries Ref + underlying error
```

**Two finding codes** (vs. the current one):

- **`isolation-escape`** (existing, M-0106) — fires on actual AI-actor escape, error severity per epic decision.
- **`isolation-escape-oracle-failure`** (new) — fires advisory severity per failed ref, naming the ref and the underlying error. Surfaces partial-coverage cases distinctly from silent-good cases.

**Design decision captured as D-NNN**: fail-shut on rule correctness (no false positives from partial info — if a ref's first-parent index is missing, the rule does not fire on commits the missing index would have classified); fail-open on rule coverage (the rest of the rule still runs for every ref that resolved cleanly). The D-NNN records the rationale; `aiwfx-record-decision` invocation lands during the AC-3 cycle.

**Mechanical assertions:**

1. **Combinatorial real-git E2E.** Scenarios under [`internal/cli/integration/isolation_escape_oracle_scenarios_test.go`](../../../internal/cli/integration/) via the M-0159 framework. Each bootstraps a temp repo with a manipulated ref-state, runs `aiwf check` as subprocess, asserts the finding-code matrix:

   | Scenario | `isolation-escape` | `isolation-escape-oracle-failure` |
   |---|---|---|
   | All refs healthy, no escape | silent | silent |
   | All refs healthy, AI escape present | fires (error) | silent |
   | One ritual ref deleted mid-check, no escape elsewhere | silent | fires (advisory, names the ref) |
   | One ritual ref deleted, escape on a healthy ref | fires (error) | fires (advisory) |
   | All ritual refs corrupted | silent (no data to fire against) | fires per ref |
   | Empty repo (no refs at all) | silent | silent |
   | Repo with only non-ritual refs (`feature/foo`) | silent | silent |

2. **Sovereign-override path stays clean.** A scenario exercises an AI escape with `aiwf-force: "..."` on the violating commit + a simultaneous failed ref. Asserts: `isolation-escape` silent (override takes effect); `isolation-escape-oracle-failure` still fires advisory (oracle-failure is orthogonal to the rule's per-commit override).

3. **Sabotage-verifiable.** Reverting the typed-error split (collapsing `OracleErrors()` back to "empty means anything") makes the "one-ref-deleted, no escape" scenario fire a missing advisory; the "one-ref-deleted, escape elsewhere" scenario silently misses the advisory while still firing `isolation-escape` (proves the new code is the load-bearing path).

4. **D-NNN decision record.** A D-NNN entity recording the fail-shut-on-correctness / fail-open-on-coverage choice lands via `aiwfx-record-decision` during the AC-3 cycle. The D-NNN id is mirrored into the milestone's `## Decisions made during implementation` section per ritual.

5. **Branch-spec cell registration.** Each of the 7 E2E scenarios above registers as a cell (1 silent-good baseline + 1 escape baseline + 5 oracle-state cells). AC-9 (G-0210) consolidates.

**Edge cases:**

- Corrupted `packed-refs` fails at `git for-each-ref` (whole-repo enumeration), not per-ref. The whole-oracle continues to fail-shut for this case — the D-NNN documents why (no ref enumeration means no rule coverage, period; advisory-per-ref makes no sense without a ref to name).
- Ref deleted between `for-each-ref` and `rev-list` is the TOCTOU case the per-ref tolerance is designed for; covered by the "one-ref-deleted mid-check" scenario.
- Non-ritual refs (`feature/foo`) are filtered before the first-parent index build, per the existing ritual-branch filter; not affected by this AC.
- The advisory-severity choice for `isolation-escape-oracle-failure` matches the M-0125 ratchet pattern (introduce as advisory; tighten to warning/error after one full epic of usage if false-positive rate stays low).

**References.**

- [G-0203](../../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md) + sub-concern N-4
- [`internal/check/isolation_escape.go`](../../../internal/check/isolation_escape.go) — the rule
- [`internal/cli/check/isolation_escape_oracle.go`](../../../internal/cli/check/isolation_escape_oracle.go) — `newGitBranchOracle`
- [`internal/cli/check/provenance.go`](../../../internal/cli/check/provenance.go) — `RunProvenanceCheck` (where oracle-failure handling wires in)
- M-0106 — the milestone that landed the original oracle
- AC-4..AC-7 — the four real-git scenarios depending on this typed-error contract
- D-NNN (to-be-created) — fail-shut/fail-open decision record

### AC-4 — BranchOracle detects shallow clones; isolation-escape-shallow-clone fires

**Observable behavior.** `newGitBranchOracle` detects shallow-clone state at construction time via `git rev-parse --is-shallow-repository`. When the repository is shallow:

- Construction returns a typed `OracleErrShallow` accumulated into `OracleErrors()` (per AC-3's typed-error contract).
- The oracle's per-SHA branch map is left empty — a shallow rev-list index is structurally incomplete and would produce silent false-negatives for every commit beyond the shallow boundary. The `isolation-escape` rule sees empty oracle data → silent (no false positives).
- The CLI gather layer surfaces a new finding [`isolation-escape-shallow-clone`](../../../internal/check/) at **warning** severity (NOT advisory — total coverage failure is louder than AC-3's per-ref partial-failure advisory).

**Hint text** on the warning names the remediation directly:

> "isolation-escape coverage is incomplete: this repository is a shallow clone (rev-list returns commits only within the shallow boundary). Unshallow with `git fetch --unshallow`, or in CI use `actions/checkout@vN` with `fetch-depth: 0`. After unshallowing, re-run `aiwf check`."

**Mechanical assertions:**

1. **Combinatorial real-git E2E.** Scenarios under [`internal/cli/integration/isolation_escape_shallow_scenarios_test.go`](../../../internal/cli/integration/) via the M-0159 framework. Each bootstraps a temp repo with a specific clone shape and asserts the finding-code matrix:

   | Scenario | `isolation-escape` | `isolation-escape-shallow-clone` |
   |---|---|---|
   | Full clone, no AI escape | silent | silent |
   | Full clone, AI escape present | fires (error) | silent |
   | `git clone --depth=1`, AI escape beyond HEAD-1 | silent (rule can't see it) | fires (warning, names remediation) |
   | `git clone --depth=1`, no AI escape | silent | fires (warning — coverage incomplete regardless) |
   | `git clone --depth=5`, AI escape inside window | fires (error) | fires (warning — depth >1 is still shallow per `is-shallow-repository`) |
   | `git fetch --unshallow` after `--depth=1` | (depends on escape) | silent (shallow flag cleared) |

2. **Sovereign-override scenario.** Shallow clone + AI escape beyond HEAD-1 that ALSO carries `aiwf-force: "..."` → `isolation-escape` silent (override would take effect IF the rule could see the commit — but the shallow boundary hides it; the override is structurally moot here); `isolation-escape-shallow-clone` STILL fires (warning — orthogonal to per-commit override; operator is told to unshallow to see the full picture).

3. **Sabotage-verifiable.** Reverting the `is-shallow-repository` detection at oracle construction makes the "shallow + AI escape beyond HEAD-1" scenario either:
   - (a) miss the warning silently (proves the new finding is the load-bearing path); OR
   - (b) build an incomplete index without the typed error, so `isolation-escape` runs with garbage data — exactly the silent-escape failure mode G-0204 names.

   The discriminating test fires either way.

4. **Branch-spec cell registration.** Each of the 6 scenarios + 1 override = 7 cells under `internal/workflows/spec/branch/`. AC-9 (G-0210) consolidates the catalog.

**Edge cases:**

- `git clone --depth=N` for any N ≥ 1 is shallow per `is-shallow-repository`. The finding fires whenever shallow; no partial-depth special-casing — shallow is shallow.
- A repo just unshallowed via `git fetch --unshallow` reports not-shallow on subsequent oracle construction. No caching; detection re-runs each time.
- **Partial clones** (sparse + blobless, e.g., `git clone --filter=blob:none`) are NOT shallow. `is-shallow-repository` returns false; the oracle's behavior there is the existing behavior (out of scope; if partial-clone produces its own silent-escape class, file a follow-up gap).
- **Worktree repos**: each worktree inherits the parent's shallow state. Detection runs in the worktree's git directory which delegates to the parent's `shallow` file. Worktree-rooted `aiwf check` sees the parent's shallow status — correct.
- Empty repo (no refs at all): `is-shallow-repository` returns false; oracle constructs with empty maps; no findings. Same as today.
- The decision to keep `isolation-escape` silent (rather than fire with garbage data) on shallow is deliberate fail-shut-on-correctness — matches AC-3's D-NNN architectural choice. The new finding's job is to surface the coverage gap; the existing rule's job is to be silent when it cannot make a confident classification.

**References.**

- [G-0204](../../gaps/G-0204-branchoracle-silent-on-shallow-clones-ci-fetch-depth-1.md) — the gap; closes here
- [`internal/cli/check/isolation_escape_oracle.go`](../../../internal/cli/check/isolation_escape_oracle.go) — `newGitBranchOracle` (where detection lands)
- [`internal/check/isolation_escape.go`](../../../internal/check/isolation_escape.go) — the rule
- AC-3 (G-0203) — typed-error contract this composes with (`OracleErrors() []error`)
- AC-9 (G-0210) — catalog refactor that consolidates cells
- M-0125 severity-ratchet pattern — `isolation-escape-shallow-clone` starts at warning per the established cadence

### AC-5 — Reflog-walk detects orphaned AI commits from force-push

**Observable behavior.** A new gather-layer component walks `git reflog` for each ritual branch, identifies force-update events, and extracts the orphaned commit SHAs. The `RunProvenanceCheck` trailer scan is extended to inspect orphaned SHAs (via `git log -1 <sha>` per orphan) for `aiwf-actor: ai/...` + `aiwf-entity: ...` trailers. When a match is found, fires a new [`isolation-escape-orphaned-ai-commit`](../../../internal/check/) finding at **warning** severity, naming the orphaned SHA, the ritual branch it was orphaned from, and the reflog entry's date.

**Hint text** on the warning:

> "AI-actor commit `<sha>` was orphaned by a force-push on `<branch>` at `<reflog date>`. The kernel cannot determine from the orphan alone whether the commit was on the correct branch at the time of force-push — the rewrite removes the audit trail. Inspect with `git reflog show <branch> | grep <sha>` and either restore the commit (`git update-ref refs/heads/<branch> <pre-push-sha>` or cherry-pick onto the correct branch) or, if the force-push was deliberate sovereign cleanup, acknowledge via `aiwf acknowledge-illegal <sha>`."

**Composition with existing surfaces:**

- The existing `aiwf acknowledge-illegal <sha>` verb silences the warning per its existing mechanism (writes an empty commit with `aiwf-force-for: <sha>` + human actor + reason); no new override path needed.
- Composes with AC-3's `OracleErrors()` typed-error contract: if the reflog itself is disabled (`core.logAllRefUpdates=false`), detection accumulates an `OracleErrReflogDisabled` entry surfaced as AC-3's `isolation-escape-oracle-failure` advisory (no new finding code for this orthogonal mode).

**Mechanical assertions:**

1. **Combinatorial real-git E2E.** Scenarios under [`internal/cli/integration/isolation_escape_force_push_scenarios_test.go`](../../../internal/cli/integration/) via the M-0159 framework. Each bootstraps a temp repo, performs the specific force-push shape, runs `aiwf check`, asserts the finding-code matrix:

   | Scenario | `isolation-escape` | `isolation-escape-orphaned-ai-commit` |
   |---|---|---|
   | Baseline: full clone, no force-push, AI escape exists | fires (error) | silent |
   | Baseline: full clone, no force-push, no escape | silent | silent |
   | Force-push orphans AI commit (any branch) | silent (orphan unreachable) | fires (warning, names SHA + branch + date) |
   | Force-push orphans non-AI commit | silent | silent (no AI trailers; rule does not fire) |
   | Force-push orphans AI commit, then `aiwf acknowledge-illegal <sha>` | silent | silent (override takes effect) |
   | Force-push orphans AI commit, reflog entry expired via `git reflog expire --expire-unreachable=now` | silent | silent (no audit trail to walk) |
   | Reflog disabled (`core.logAllRefUpdates=false`), force-push happens | silent (no reflog), PLUS `isolation-escape-oracle-failure` advisory fires (per AC-3 composition) | — |

2. **Sabotage-verifiable.** Removing the reflog-walk step from the gather layer makes the "force-push orphans AI commit" scenario fire on a missing warning. The acknowledged variant continues to silence (proving the acknowledge-illegal composition is not the load-bearing path).

3. **Branch-spec cell registration.** 7 scenarios = 7 cells under `internal/workflows/spec/branch/`. AC-9 (G-0210) consolidates.

**Edge cases:**

- **CI cannot detect orphans.** Reflog is local-only by git design; CI workflows fetch fresh clones without the preserved reflog. Documented limitation: operator-side `aiwf check` (pre-push hook) is the chokepoint, not CI. Surfaced in the hint and in CLAUDE.md.
- **Reflog horizon** (`gc.reflogExpire` default 90 days, 30 days for unreachable): orphan detection is time-bound. Acceptable; documented.
- **Git version variation in reflog format**: parsing path uses `git reflog show <branch> --pretty='%H %gs'` (subject contains `forced-update` or `update by push`) — portable across git ≥ 2.0.
- **Force-push to a remote ref without local fetch**: doesn't appear in the local reflog. Out of scope — kernel polices local state; the operator's machine is the truth source.
- **Bare repos / `core.logAllRefUpdates=false`**: composes with AC-3 via `OracleErrReflogDisabled`; `isolation-escape-oracle-failure` advisory fires; no false-positive silence.
- **Force-push that doesn't orphan anything** (e.g., a force-push that's a no-op against an already-aligned ref): reflog records the event but no SHA is unreachable; detection finds no orphans; both findings silent. Same as today.

**References.**

- [G-0205](../../gaps/G-0205-branchoracle-silent-on-force-pushed-away-violating-commits.md) — the gap; closes here
- [`internal/cli/check/isolation_escape_oracle.go`](../../../internal/cli/check/isolation_escape_oracle.go) — where reflog-walk lands
- [`internal/check/isolation_escape.go`](../../../internal/check/isolation_escape.go) — the rule
- AC-3 (G-0203) — typed-error contract this composes with
- `aiwf acknowledge-illegal` — existing sovereign-override path (composes here)
- AC-9 (G-0210) — catalog refactor that consolidates cells

### AC-6 — BranchOracle resolves renamed branches via SHA fallback

**Observable behavior.** The `aiwf authorize --branch <name>` verb is extended to record TWO trailers on the authorize commit:

- `aiwf-branch: <name>` (existing — the branch's short name at scope-open time)
- `aiwf-branch-sha: <sha>` (new — the branch's tip SHA at scope-open time; the authorize commit IS the at-open marker, so "at-open" is implicit)

The `isolation-escape` rule's scope-branch resolution becomes:

1. Read `aiwf-branch:` (name) and `aiwf-branch-sha:` (SHA, if present) from the scope's authorize commit.
2. Resolve the scope's bound branch:
   - **If `<sha>` trailer present**: look up the current ritual branch whose first-parent index contains `<sha>` (via the oracle, extended with a `BranchOfSHA(sha string) string` query). SHA is unambiguous and survives renames; this is the primary path.
   - **Else (legacy authorize, no SHA trailer)**: resolve `<name>` to a current ritual branch. Name-only path is backwards-compatible for pre-AC-6 authorize commits.
3. Get the AI commit's branch from the oracle (existing).
4. Compare resolved scope branch to AI commit's branch; fire `isolation-escape` on mismatch (existing).

**Net effect:** a `git branch -m oldname newname` rename is transparent to the rule — the scope's binding follows the renamed branch via SHA reachability. No false positives on legitimate ritual renames.

**Backstop**: if NEITHER the name NOR the SHA resolves to a current ritual branch (branch deleted entirely, SHA orphaned by reflog GC), surface AC-3's `isolation-escape-oracle-failure` advisory naming the scope's bound branch as unreachable. The `isolation-escape` rule stays silent (fail-shut on correctness — no false positive when binding lost).

**Trailer-keys policy extension**: `aiwf-branch-sha:` lands in [`internal/policies/trailer_keys.go`](../../../internal/policies/trailer_keys.go) allowlist; existing kernel trailer-keys policy continues to enforce canonical lowercase shape.

**Mechanical assertions:**

1. **Combinatorial real-git E2E.** Scenarios under [`internal/cli/integration/isolation_escape_rename_scenarios_test.go`](../../../internal/cli/integration/) via the M-0159 framework. The matrix:

   | Scenario | `isolation-escape` |
   |---|---|
   | No rename, AI commit on correct branch (baseline) | silent |
   | No rename, AI commit on wrong branch (baseline) | fires (error) |
   | Rename `foo → bar`, AI commit on `bar` (the renamed-to) | silent (SHA resolves to `bar`; matches) |
   | Rename `foo → bar`, AI commit on a *different* branch `baz` | fires (error) (SHA resolves to `bar`; doesn't match `baz`) |
   | Rename `foo → bar → foo` (rename then rename back) | silent (SHA still resolves to `foo`; matches AI's branch) |
   | Branch deleted entirely (SHA orphaned) | silent + `isolation-escape-oracle-failure` advisory (AC-3 composition) |
   | Squat collision: rename `foo → bar`; create new `foo` from unrelated SHA; AI commit on `bar` | silent (SHA wins; resolves to `bar`) |
   | Legacy authorize commit (no `aiwf-branch-sha:` trailer), no rename, AI on correct branch | silent (name-only path; backwards-compatible) |
   | Legacy authorize commit, branch renamed | fires (error — false positive on legacy commits; documented limitation: pre-AC-6 scopes don't benefit from rename transparency) |

2. **Sovereign-override path stays clean.** Existing `aiwf acknowledge-illegal <sha>` silences any remaining false positives (e.g., legacy authorize + rename case); no new override needed.

3. **Sabotage-verifiable.** Removing the SHA-fallback resolution makes the "rename + AI on renamed branch" scenario fire `isolation-escape` (the false-positive failure mode G-0206 names).

4. **Branch-spec cell registration.** 9 scenarios = 9 cells under `internal/workflows/spec/branch/`. AC-9 (G-0210) consolidates.

**Edge cases:**

- **Multi-step renames** (`foo → bar → baz`): SHA still resolves to `baz` (immutable through the chain). Single resolution step suffices.
- **Concurrent renames** (rename happens BETWEEN oracle build and rule run): TOCTOU race, vanishingly rare; documented as a fundamental property of the snapshot-style oracle.
- **`aiwf-branch-sha:` value validation**: the trailer must be a 40-char hex SHA at canonical shape; the trailer-keys policy enforces.
- **Detached HEAD at authorize time**: cannot record `aiwf-branch-sha:` because there's no branch tip. The existing `aiwf authorize` rejection for detached HEAD (AC-7's territory) prevents this from arising — AC-6 assumes a real branch at scope-open.
- **`aiwf authorize --branch` for a NON-EXISTENT branch** (the "future-branch carve-out" from M-0103 + M-0105): no current SHA to record. The trailer is OPTIONAL when the branch doesn't yet exist; rule falls back to name-only resolution until the branch is created. (This is the M-0102/M-0103 future-branch ritual-shape carve-out, unchanged here.)
- **Squat collision** (the trickiest case): rename `foo → bar`, then create new `foo` from an unrelated commit. Name-only resolution would find the squat; SHA-only resolution finds `bar`. Rule prefers SHA: the binding is to whatever SHA was recorded at scope-open, not to whatever label is currently attached to that name. This is the fundamentally correct semantic.

**References.**

- [G-0206](../../gaps/G-0206-branchoracle-false-positive-on-branch-renames-after-authorize.md) — the gap; closes here
- [`internal/verb/authorize.go`](../../../internal/verb/authorize.go) — extends with `aiwf-branch-sha:` trailer
- [`internal/check/isolation_escape.go`](../../../internal/check/isolation_escape.go) — scope-branch resolution extended
- [`internal/cli/check/isolation_escape_oracle.go`](../../../internal/cli/check/isolation_escape_oracle.go) — adds `BranchOfSHA(sha)` query
- [`internal/policies/trailer_keys.go`](../../../internal/policies/trailer_keys.go) — registers `aiwf-branch-sha:`
- AC-3 (G-0203) — typed-error contract this composes with
- AC-9 (G-0210) — catalog refactor that consolidates cells
- M-0103, M-0105 — future-branch carve-out unchanged here

### AC-7 — Detached HEAD behavior pinned across preflight, oracle, check, doctor

**Observable behavior.** Detached HEAD state has explicit, tested behavior across all four surfaces:

1. **Preflight (`aiwf authorize --to ai/<agent>`)** refuses with a refined error message naming detached state and the override path:

   > "refused: detached HEAD has no ritual context. Checkout a ritual branch (epic/E-NNNN-<slug> or milestone/M-NNNN-<slug>) and rerun, or use `--force --reason "..."` to override."

   No new error code — the existing `branch-context-required` refusal path stays; only the message text refines. (Verb-time errors don't carry codes/subcodes; the discriminating signal is the substring "detached HEAD has no ritual context".)

2. **Oracle (BranchOracle construction)** proceeds normally. Detached HEAD doesn't affect `git for-each-ref refs/heads/`; ritual branches index unchanged. An AI commit made FROM detached HEAD lands as a dangling commit (no ref points at it) → its SHA is not in any ritual branch's first-parent index → oracle returns empty → `isolation-escape` rule treats as "no branch info" and stays silent. (Composes with AC-3's typed-error contract: empty branch set is KNOWN-GOOD empty, not lookup-failed.)

3. **`aiwf check` from a detached HEAD worktree** succeeds with no degradation. The root-resolution chain in `internal/repoutil/` walks up the filesystem to find the repo root; detached state doesn't affect the walk.

4. **`aiwf doctor`** surfaces a new [`detached-head`](../../../internal/cli/doctor.go) check at advisory severity when invoked from detached HEAD, with the same actionable text as the preflight refusal. Lets operators discover the state proactively rather than via a verb refusal.

**Mechanical assertions:**

1. **Combinatorial real-git E2E.** Scenarios under [`internal/cli/integration/detached_head_scenarios_test.go`](../../../internal/cli/integration/) via the M-0159 framework. Each bootstraps a temp repo, performs `git checkout <sha>` to detach, asserts:

   | Scenario | Expected outcome |
   |---|---|
   | Detached HEAD + `aiwf authorize <id> --to ai/<agent> --branch epic/...` | refused; stderr contains "detached HEAD has no ritual context" |
   | Detached HEAD + `aiwf authorize <id> --to ai/<agent>` (no --branch) | refused; same message |
   | Detached HEAD + `aiwf authorize <id> --to ai/<agent> --branch epic/... --force --reason "intentional"` | succeeds; commit carries `aiwf-force:` trailer |
   | Detached HEAD + `aiwf check` (no AI commits) | exit 0, no false findings |
   | Detached HEAD + AI commit made dangling | `isolation-escape` silent (oracle returns empty for dangling SHA per AC-3 KNOWN-GOOD path) |
   | Detached HEAD + `aiwf doctor` | exit 0 with `detached-head` advisory in output |
   | NOT detached + `aiwf doctor` | exit 0, no `detached-head` advisory (baseline) |

2. **Sovereign-override path is the 3rd row.** Pinned as a positive cell.

3. **Sabotage-verifiable.** Reverting the refined error message reverts to today's flat `branch-context-required` text; the discriminating test fires on missing "detached HEAD" substring. Reverting the `doctor` check makes the "detached + doctor" scenarios fail on missing advisory.

4. **Branch-spec cell registration.** 7 scenarios = 7 cells under `internal/workflows/spec/branch/`. AC-9 (G-0210) consolidates.

**Edge cases:**

- **Detached HEAD on a ritual branch's tip** (`git checkout epic/E-X` → `git switch --detach`): `git symbolic-ref --short HEAD` still exits non-zero (HEAD is detached, not symbolic) — refuse path correct. The operator is working on the branch's content but not "on" the branch; intent is ambiguous; refuse is the right default.
- **`git worktree add --detach`**: creates a worktree with detached HEAD. Same handling; no special case.
- **`aiwf check` during a `git rebase`-in-progress** (detached HEAD + rebase state files): the check proceeds silently. Rebase state is transient; the operator completes the rebase shortly. `aiwf doctor` may surface both `detached-head` AND a hypothetical `rebase-in-progress` advisory if that check exists (it doesn't today; out of scope here).
- **Refined error message in non-English locales**: not localized today; the substring "detached HEAD has no ritual context" is the canonical English form. Localization is a separate gap.

**References.**

- [G-0207](../../gaps/G-0207-detached-head-handling-untested-in-preflight-and-oracle.md) — the gap; closes here
- [`internal/cli/authorize/authorize.go`](../../../internal/cli/authorize/authorize.go) — `currentBranch` helper (error message refinement)
- [`internal/cli/check/isolation_escape_oracle.go`](../../../internal/cli/check/isolation_escape_oracle.go) — oracle (no code changes; behavior pinned by E2E)
- [`internal/cli/doctor.go`](../../../internal/cli/doctor.go) — new `detached-head` doctor check
- AC-3 (G-0203) — typed-error contract this composes with
- AC-9 (G-0210) — catalog refactor that consolidates cells

### AC-8 — Kernel finding promote-on-wrong-branch enforces ADR-0010 ritual ordering

**Observable behavior.** A new finding [`promote-on-wrong-branch`](../../../internal/check/) fires at **warning** severity when an entity-activating promote commit lands on a branch other than the entity's parent branch. Per ADR-0010, sovereign acts must land on the parent branch BEFORE the ritual branch is cut.

**Activating promotes covered:**

- `aiwf promote E-NNNN active` (epic activation) → expected branch: trunk (per AC-1's `Config.TrunkBranchShortName()`)
- `aiwf promote M-NNNN in_progress` (milestone activation) → expected branch: parent epic branch (`epic/E-XXXX-<slug>`, looked up via `Tree.ByID(parent).Slug()`)
- `aiwf promote G-NNNN active` (gap activation) → no branch expectation (gaps don't get branch-cut semantics today)

**Non-activating promotes** (e.g., `epic.active → done`, `milestone.in_progress → done`, ADR `proposed → accepted`, D-NNN `proposed → ratified`): silent. The rule focuses narrowly on the ritual-ordering failure mode G-0209 names.

**Note on authorize-side ordering**: the authorize-on-wrong-branch failure mode named in G-0209 is **already caught by AC-2's rung-pair predicate**: an authorize from a same-rung branch (e.g., on `epic/E-NN-foo` and `--branch epic/E-NN-foo`) is refused as (epic, epic) cross-epic-typo. So this AC handles only the promote-on-wrong-branch case; the authorize case is structurally prevented upstream.

**Hint text** on the warning:

> "`aiwf promote E-NN active` landed on `epic/E-NN-foo`, not on the parent branch (`main`). The ADR-0010 branch model requires sovereign promotes on the parent branch BEFORE the ritual branch is cut. If the order was deliberate (e.g., re-activating an entity from its branch), use `aiwf acknowledge-illegal <sha>` to silence."

**Sovereign override paths:**

- `aiwf acknowledge-illegal <sha>` silences post-hoc (existing mechanism; same shape as AC-5/AC-6 composition).
- `--force --reason "..."` on the promote commit suppresses per-commit (existing override pattern with `aiwf-force:` trailer).

**Mechanical assertions:**

1. **Combinatorial real-git E2E.** Scenarios under [`internal/cli/integration/promote_wrong_branch_scenarios_test.go`](../../../internal/cli/integration/) via the M-0159 framework. The matrix:

   | Scenario | `promote-on-wrong-branch` |
   |---|---|
   | Baseline: epic activating promote on trunk | silent |
   | Baseline: milestone activating promote on parent epic branch | silent |
   | Wrong: epic activating promote on `epic/E-X-...` branch | fires (warning) |
   | Wrong: milestone activating promote on `milestone/M-Y-...` branch | fires (warning) |
   | Wrong: milestone activating promote on trunk (skipping parent epic) | fires (warning, names expected parent epic branch) |
   | Wrong: epic activating promote on detached HEAD | fires (warning; composes with AC-7) |
   | Non-activating promote on wrong branch (e.g., `epic.active → done` on `epic/E-X`) | silent (out of rule's domain) |
   | Sovereign: wrong-branch promote + `aiwf acknowledge-illegal <sha>` | silent |
   | Sovereign: wrong-branch promote + `aiwf-force: "..."` trailer | silent |

2. **Sabotage-verifiable.** Removing the rule from `RunProvenanceCheck` makes the "wrong-branch promote" scenarios fire on missing warning. Sovereign-override scenarios continue to silence (proving overrides are not the load-bearing test path).

3. **Branch-spec cell registration.** 9 scenarios = 9 cells under `internal/workflows/spec/branch/`. AC-9 (G-0210) consolidates.

**Edge cases:**

- **Trunk-name composition** with AC-1: trunk identity from `Config.TrunkBranchShortName()`. The rule honors `aiwf.yaml.allocate.trunk`.
- **Parent epic branch resolution**: milestone's expected branch derives from `Tree.ByID(parent)` → epic's id + slug → canonical pattern `epic/E-XXXX-<slug>`. **Fail-shut if parent lookup fails**: rule treats as "cannot determine expected branch" → surfaces AC-3's `isolation-escape-oracle-failure` advisory rather than firing a false positive.
- **Legacy promotes** (pre-AC-8, on the current repo's history): the rule scans full history; legacy promotes on wrong branches will fire warnings on first push. Per M-0125 ratchet pattern, finding ships at warning (not error) so operators have one full epic to acknowledge historicals via `aiwf acknowledge-illegal` before any tightening decision (recorded as a future D-NNN).
- **Re-activating cases** (e.g., `epic.done → active` if ever allowed): the activating-transition list is keyed on source-status pairs (`proposed → active`, `draft → in_progress`); re-activations with different source statuses are outside the rule's domain.
- **ADR / D-NNN / contract promotes**: no branch ordering expectation; rule silent. These entities don't have branch-cut semantics.

**References.**

- [G-0209](../../gaps/G-0209-ritual-step-ordering-is-advisory-only-no-kernel-enforcement.md) — the gap; closes here
- ADR-0010 — branch model that the rule enforces
- [`internal/check/`](../../../internal/check/) — new rule `promote_on_wrong_branch.go` lands here
- [`internal/cli/check/provenance.go`](../../../internal/cli/check/provenance.go) — wiring into `RunProvenanceCheck`
- AC-1 (G-0200) — composes via `Config.TrunkBranchShortName`
- AC-2 (G-0201) — handles the authorize-side ordering failure mode upstream
- AC-3 (G-0203) — composes via BranchOracle for branch resolution + typed-error fail-shut
- AC-7 (G-0207) — composes via detached-HEAD edge case
- AC-9 (G-0210) — catalog refactor that consolidates cells

### AC-9 — Layer-4 spec-table refactor: mechanical-weight catalog + bijection enforcement

**Observable behavior.** The layer-4 branch-choreography spec catalog under [`internal/workflows/spec/branch/`](../../../internal/workflows/spec/branch/) is refactored to a mechanical-weight-only set with bijection enforcement between cells and tests.

**Part 1 — M-0158 catalog refactor:**

Drop the 9 documentation-only / duplicate cells per G-0210's enumeration:
- 5 legal-non-override doc-only cells: `branch-cell-3, 5, 6, 9, 11`
- 2 legal-AND-override cells (semantic dupes of overrides): `branch-cell-8, 10`
- 2 override-named cells (semantic dupes of corner cases): `branch-cell-override-cherry-pick`, `branch-cell-override-force-amend`

Keep the 7 mechanical-weight cells:
- 5 illegal: `branch-cell-1, 2, 4, 7, 12`
- 2 standalone overrides: `branch-cell-override-preflight`, `branch-cell-override-f-nnnn-waiver`

**Part 2 — Add M-0161 cells (66 net new):**

| AC | Cells added | Shape |
|---|---|---|
| AC-1 (trunk-name config) | 4 | 4 positive trunk-name shapes |
| AC-2 (rung-pair predicate) | 17 | 4 positive + 12 negative + 1 override |
| AC-3 (oracle typed errors) | 7 | 7 oracle-state scenarios |
| AC-4 (shallow clones) | 7 | 6 scenarios + 1 override |
| AC-5 (force-push orphans) | 7 | 7 reflog-state scenarios |
| AC-6 (branch rename) | 9 | 9 rename-state scenarios |
| AC-7 (detached HEAD) | 7 | 7 detached-state scenarios |
| AC-8 (promote-on-wrong-branch) | 9 | 9 promote-ordering scenarios |
| **M-0161 subtotal** | **66** | |

**Net catalog after AC-9**: 7 (M-0158 retained) + 66 (M-0161 added) = **73 cells**.

**Part 3 — Bijection enforcement (`branchcell.Pin` registry):**

A new package-level registry under [`internal/workflows/spec/branch/pin.go`](../../../internal/workflows/spec/branch/):

```go
// Pin registers a cell ↔ test binding. Called from a test's setup
// (typically inside the scenario's Setup or directly in the
// TestFunction body). The bijection meta-test enforces 1:1
// correspondence at CI time.
func Pin(cellID string, testFunctionName string)
```

Every E2E scenario added by AC-1..AC-8 calls `branchcell.Pin(cellID, "<testName>")` at scenario setup. The package accumulates pins; the bijection meta-test reads them.

**The bijection meta-test** under [`internal/policies/branch_cell_bijection_test.go`](../../../internal/policies/branch_cell_bijection_test.go) enforces four invariants:

1. **Every cell in `branch.Rules()` has at least one Pin** — no documented-but-unpinned cells.
2. **Every Pin references a cell that exists in `branch.Rules()`** — no orphan pins.
3. **No cell has 2+ Pins** — no double-mapping (fixes G-0210's "test signal weakness" concern).
4. **No test function pins 2+ cells** — one test = one cell, no overload.

Replaces M-0158/AC-5's keyword-set meta-coverage approach. The keyword-set test under `internal/policies/m0158_ac5_meta_coverage_test.go` is removed in the same commit.

**Part 4 — Drift policy extension:**

The existing M-0158 drift policy is extended to read from the `branchcell.Pin` registry. A cell added to `branch.Rules()` without a paired `Pin` call fails CI; a `Pin` without a corresponding cell fails CI.

**Mechanical assertions:**

1. **Catalog refactor verification.** A test under [`internal/policies/branch_cell_catalog_test.go`](../../../internal/policies/) asserts:
   - The 9 dropped cells are ABSENT from `branch.Rules()`
   - The 7 M-0158 retained cells are PRESENT
   - The 66 M-0161 new cells are PRESENT
   - **Total cell count == 73** (pinned exact count to catch drift)

2. **Bijection enforcement test.** The 4 invariants above each have a dedicated subtest. Each subtest is sabotage-verifiable:
   - Remove a Pin from a test → "cell with no Pin" subtest fires.
   - Add a Pin for a non-existent cell → "orphan Pin" subtest fires.
   - Add a 2nd Pin to an existing cell → "double-mapping" subtest fires.
   - Add a 2nd Pin from a test function → "overload" subtest fires.

3. **M-0158 keyword-set removal verification.** The file `internal/policies/m0158_ac5_meta_coverage_test.go` is deleted in the AC-9 commit; a structural test asserts the file does not exist (prevents reintroduction).

4. **Drift policy extension.** Fixture tests under `internal/policies/branch_cell_drift_test.go` exercise:
   - Add cell, no Pin → fails CI
   - Remove cell, leave Pin → fails CI
   - Healthy state (1:1 across all 73) → silent

5. **Meta-cell registration.** AC-9 produces three meta-cells in the catalog:
   - `branch-cell-meta-bijection-enforced` (positive)
   - `branch-cell-meta-pin-orphan-detected` (positive)
   - `branch-cell-meta-cell-orphan-detected` (positive)

   These bring the actual total to **76 cells**. (The 73 figure above is the AC-1..AC-8-contributed total; AC-9 adds its own 3 meta-cells.)

**Edge cases:**

- **M-0158 wrap unaffected**: the catalog refactor is additive in the "right direction" (drop 9, add 66) — M-0158's wrap statements about its 7 retained cells stay true; the 9 dropped cells were documented as carrying no mechanical weight already.
- **Test-only files**: `branchcell.Pin` lives in production code (`internal/workflows/spec/branch/pin.go`) so tests across multiple `_test.go` files can call it. Pins accumulate in a `var pins []pin` package var with a mutex for parallel-test safety.
- **`branchcell.Pin` test-mutating-production-code concern**: the pin registry is read-only after construction in production paths; only tests write. A production import-side accidental write is caught by a one-line build-time guard: the writer function does `if !testing.Testing() { panic(...) }`. (Per CLAUDE.md "No new package-level mutable state" — the exception is documented; the registry's mutation IS test infrastructure, not production state.)
- **Bijection meta-test parallelism**: runs serial via `setup_test.go` conventions (registers a non-parallel skip per the M-0091 cap rule; reading the global registry safely is what justifies the skip).
- **Cells added in later milestones** (future): the bijection enforcement catches future drift — anyone adding a cell without a Pin fails CI.

**References.**

- [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md) — the gap; closes here
- M-0158 — the milestone with the over-specified catalog (refactor closes here)
- [`internal/workflows/spec/branch/`](../../../internal/workflows/spec/branch/) — package layout
- [`internal/policies/m0158_ac5_meta_coverage_test.go`](../../../internal/policies/) — keyword-set meta-test (REMOVED by AC-9)
- AC-1..AC-8 — sources of the 66 new cells

