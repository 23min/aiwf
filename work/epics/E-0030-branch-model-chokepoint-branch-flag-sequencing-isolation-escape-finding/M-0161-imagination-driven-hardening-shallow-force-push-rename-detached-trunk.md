---
id: M-0161
title: 'Imagination-driven hardening: shallow, force-push, rename, detached, trunk'
status: in_progress
parent: E-0030
tdd: required
acs:
    - id: AC-1
      title: aiwf authorize preflight respects configured trunk name
      status: met
      tdd_phase: done
    - id: AC-2
      title: Authorize preflight enforces ritual rung hierarchy
      status: met
      tdd_phase: done
    - id: AC-3
      title: BranchOracle typed errors with per-ref fault tolerance
      status: open
      tdd_phase: done
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

   | Trunk shape | `allocate.trunk` (E2E) ¹ | Local trunk branch |
   |---|---|---|
   | Default | `refs/heads/main` | `main` |
   | GitHub-classic | `refs/heads/master` | `master` |
   | Operator-chosen | `refs/heads/dev` | `dev` |
   | Operator-chosen | `refs/heads/trunk` | `trunk` |

   ¹**E2E fixture shape**: the E2E uses `refs/heads/<X>` for all 4 cells so the fixture is self-contained (no upstream remote setup needed — `git init -b <X>` births the local ref). The orthogonal `refs/remotes/<remote>/<name>` tracking-ref shape is **exhaustively covered by the auxiliary unit table** at [`internal/config/config_test.go::TestTrunkBranchShortName`](../../../internal/config/config_test.go) (10 rows including alternate-remote-upstream). The seam this E2E pins — verb-layer call site reading `cfg.TrunkBranchShortName()` against `opts.CurrentBranch` — is shape-agnostic; both ref shapes reduce to the same short-name via the helper.

   Each scenario: bootstrap temp repo with the named `aiwf.yaml` + local branch; run `aiwf authorize <epic-id> --to ai/alice --branch epic/E-NNNN-foo` from that trunk checkout against the worktree-built binary (no `--force`, no `--reason`); assert exit 0 + `aiwf-branch: epic/E-NNNN-foo` trailer + **NO `aiwf-force:` trailer** (the carve-out is the load-bearing accept, not a silent force-override).

2. **Sabotage-verifiable.** Reverting the call-site change at `internal/verb/authorize.go` (restoring the literal `"main"` at the `currentIsRitualContext := opts.CurrentBranch == "main"` site) makes the 3 non-main scenarios fire `branch-not-found` (with the error text naming M-0104/AC-4 — same shape as the RED phase) — proving the integration test discriminates the new code path. The `main` cell continues to pass under sabotage (the literal matches).

3. **Auxiliary unit test.** `Config.TrunkBranchShortName()` table-driven test covers the canonical shapes (`refs/remotes/<remote>/<name>` → `<name>`; `refs/heads/<name>` → `<name>`) plus degenerate / unparseable cases (empty in, empty out — no panic). Diagnostic, not load-bearing — the E2E is the test set that pins behavioral correctness end-to-end.

4. **Branch-spec cell registration.** 4 positive cells in `internal/workflows/spec/branch/` covering the trunk-name shapes. AC-9 (G-0210) consolidates the catalog.

**Edge cases:**

- Bare-name trunk refs (`refs/heads/master`, not `refs/remotes/origin/master`) parse correctly via the same helper — single source of truth, no fork for "local trunk" vs "tracking trunk" semantics.
- **Empty `allocate.trunk`** → `AllocateTrunkRef()` falls through to `DefaultAllocateTrunk` (`refs/remotes/origin/main`) → `TrunkBranchShortName()` returns `"main"` (the default). This preserves backwards compatibility for repos that never configured the value.
- **Malformed `allocate.trunk`** (no parseable last segment — e.g., `"garbage"` or `"refs/heads/"`) → helper returns `""` → carve-out's left arm fails → preflight falls through to the existing implicit-ritual-current path. No regression on malformed config; no panic.
- Operator on a non-trunk branch (regardless of trunk name) → preflight uses the ritual-current path; this AC does not touch that arm.
- The trunk-name short helper is a pure derivation; it does not query git. The config is the single source of truth, consistent with the rest of `internal/config/`.

**References.**

- [G-0200](../../gaps/G-0200-preflight-main-only-carve-out-generalize-to-trunk-name-from-aiwf-yaml.md) — the gap; closes here
- [`internal/verb/authorize.go`](../../../internal/verb/authorize.go) — the `currentIsRitualContext := opts.CurrentBranch == "main"` call site (line numbers drift; symbol is the durable anchor)
- [`internal/config/config.go`](../../../internal/config/config.go) — where `AllocateTrunkRef()` already lives and where the new short-name helper lands
- [`internal/branchparse/`](../../../internal/branchparse/) — the package the call site already consumes for ritual-branch shapes
- [M-0104](M-0104-aiwfx-start-epic-sequencing-fix-closes-g-0116.md) — the milestone whose Cycle 1 reviewer flagged this layering smell

### AC-2 — Authorize preflight enforces ritual rung hierarchy

**Observable behavior.** `aiwf authorize <id> --to ai/<agent> --branch <ritual-future>` accepts only `(CurrentBranch rung, --branch rung)` pairs from the legal set. Illegal pairs refuse with an actionable error naming the offending shapes and the sovereign override path (`--force --reason "..."`).

**Rung-pair matrix (4 legal + 12 illegal = 16 cells):**

| Current rung | Target rung | Legal? | Notes |
|---|---|---|---|
| trunk | epic | ✅ | `aiwfx-start-epic` from trunk |
| epic | milestone | ✅ | `aiwfx-start-milestone` from epic |
| milestone | patch | ✅ | `wf-patch` from milestone |
| epic | patch | ✅ | `wf-patch` from epic, milestone-skipping |
| trunk | trunk | ❌ | `--branch <trunk-short-name>` from trunk — **upstream-refused** as non-ritual shape (not by this AC's rung-check) |
| epic | trunk | ❌ | `--branch <trunk-short-name>` from epic — **upstream-refused** (same as above) |
| milestone | trunk | ❌ | `--branch <trunk-short-name>` from milestone — **upstream-refused** |
| patch | trunk | ❌ | `--branch <trunk-short-name>` from patch — **upstream-refused** |
| trunk | milestone | ❌ | rung-skip |
| trunk | patch | ❌ | rung-skip×2 |
| epic | epic | ❌ | cross-epic typo |
| milestone | milestone | ❌ | cross-milestone typo |
| patch | patch | ❌ | cross-patch typo |
| milestone | epic | ❌ | up-the-tree |
| patch | milestone | ❌ | up-the-tree |
| patch | epic | ❌ | up-the-tree, skipping milestone |

**Single rung-pair check refuses every illegal cell.** This AC's predicate runs whenever `--branch` is non-empty, **regardless of `BranchExists`**:

```go
currentRung := branchparse.RungOf(opts.CurrentBranch, opts.TrunkShort)
targetRung  := branchparse.RungOf(opts.Branch,        opts.TrunkShort)
if !branchparse.LegalRungPair(currentRung, targetRung) {
    return refusal // names both rungs + override path
}
```

That single check covers ALL 12 illegal cells uniformly:

- The 4 `(X, trunk)` rows refuse because `LegalRungPair(_, "trunk")` is false for every `X` (no legal pair has trunk as its target — AI work on trunk is verboten per ADR-0010).
- The 8 cross-rung-typo / up-the-tree rows refuse because each (current, target) pair is not in the legal set `{(trunk, epic), (epic, milestone), (milestone, patch), (epic, patch)}`.
- The 4 legal pairs accept because their rung-pair IS in the legal set.

**Why "regardless of BranchExists":** the pre-AC-2 verb-layer carve-out only ran when the named `--branch` did not exist locally (`BranchExists=false`); when the trunk's local branch existed and the operator passed `--branch <trunk>`, the verb silently accepted AI work targeting trunk. AC-2 closes that escape by running the rung-pair check on `--branch` whenever it's non-empty.

**Mechanical assertions:**

1. **Combinatorial real-git E2E.** A scenario fan-out under [`internal/cli/integration/authorize_scenarios_test.go`](../../../internal/cli/integration/authorize_scenarios_test.go) exercises all 16 (CurrentBranch rung, --branch rung) combinations as separate scenarios via the M-0159 `RunScenarios` framework. Each scenario bootstraps a temp repo, checks out the relevant current-branch shape, runs `aiwf authorize <id> --to ai/<agent> --branch <target>` against the worktree-built binary, asserts:
   - **4 legal pairs**: exit 0; authorize commit lands with `aiwf-branch:` trailer naming the target; no `aiwf-force:` present.
   - **12 illegal pairs**: exit non-zero; stderr names both the `CurrentBranch` rung and the `--branch` rung; stderr names the `--force --reason "..."` override path; no commit is produced.

2. **One sovereign-override E2E.** A single additional scenario exercises an illegal pair (e.g., epic → epic) plus `--force --reason "cross-epic intentional"` → exit 0; the authorize commit carries both `aiwf-branch:` (the target) AND `aiwf-force:` (the reason). Pins the sovereign-override surface for this AC's gate, per the epic's "override gated, audited, last-resort" commitment.

3. **Sabotage-verifiable.** Reverting the rung-pair check at the carve-out site makes **all 12 illegal cells fire** on "accepted but should refuse" — the pre-AC-2 production accepts (a) the 8 ritual-target illegal cells via the loose "current is ritual + target is ritual" carve-out, and (b) the 4 `(X, trunk)` cells via the `BranchExists=true` bypass that skips the carve-out entirely. Both classes pass-pre-revert and fail-post-revert — single-revert test discrimination.

4. **Branch-spec cell registration.** Each of the 16 rung-pair scenarios registers as a named cell (4 positive + 12 negative) in `internal/workflows/spec/branch/`, plus 1 override cell = 17 cells. AC-9 (G-0210) consolidates the catalog; the cell-coverage drift policy then enforces that each cell has its paired E2E scenario.

5. **Auxiliary unit tests.** The helpers `branchparse.RungOf(branch, trunkShort string) string` and `branchparse.LegalRungPair(currentRung, targetRung string) bool` get unit tests for their derivation shapes. **`RungOf` takes the configured trunk short-name as a parameter** (sourced from `Config.TrunkBranchShortName()` at the call site) so trunk-detection is config-driven, not regex-only: `RungOf("main", "main")` returns `"trunk"`; `RungOf("main", "master")` returns `""` (`main` is not the trunk on a master-repo); `RungOf("epic/E-X-foo", anyTrunk)` returns `"epic"`. The unit tests are diagnostic — they catch a bad helper edit fast at unit level — not the load-bearing evidence; the E2E is what pins behavioral correctness.

**Edge cases:**

- Trunk-name composition with AC-1: the trunk rung is derived from `Config.TrunkBranchShortName()` per AC-1's helper. So `master` (or any other configured trunk name) maps to the `"trunk"` rung. This AC builds on AC-1; AC-1 must land first.
- Unparseable `--branch` (not matching any ritual shape AND not equal to the configured trunk's short name) → `RungOf` returns `""` → `LegalRungPair(_, "")` is false → rung-pair refusal fires. Replaces the old `branch-not-found` carve-out semantics for non-ritual `--branch` values; the rung-pair predicate subsumes the prior check uniformly.
- Detached HEAD → out of scope (AC-7 / G-0207 owns it). The current-rung check here is gated on a parseable symbolic-ref result.
- `--force --reason "..."` bypasses the rung-pair check; the authorize commit then carries `aiwf-force:` per the existing kernel pattern.
- The `(epic, patch)` legal pair encodes the deliberate "patch on epic" shape (a wf-patch cut from the epic branch without an intermediate milestone). The `(milestone, patch)` pair encodes "patch on milestone". Both are legal because both are operator-intentional; neither is a typo class.

**References.**

- [G-0201](../../gaps/G-0201-authorize-preflight-carve-out-accepts-cross-rung-ritual-mismatches.md) — the gap; closes here
- [`internal/verb/authorize.go`](../../../internal/verb/authorize.go) — the carve-out call site
- [`internal/branchparse/`](../../../internal/branchparse/) — where `RungOf` and `LegalRungPair` land
- [M-0105](M-0105-aiwfx-start-milestone-sequencing-alignment.md) — the milestone whose Cycle 1 reviewer flagged this looseness
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

2. **Sovereign-override path stays clean.** A scenario exercises AI escape commit X on ritual branch A (X carrying `aiwf-force: "..."`) + an unrelated ritual ref B failed to resolve at oracle construction. Asserts: `isolation-escape` silent on X (per-commit override takes effect; ref A is healthy so X's branch is correctly identified); `isolation-escape-oracle-failure` fires advisory naming ref B (oracle-failure is orthogonal to the rule's per-commit override — the two ride independent codepaths).

3. **Sabotage-verifiable.** Reverting the typed-error split (collapsing `OracleErrors()` back to "empty means anything") makes the "one-ref-deleted, no escape" scenario fire a missing advisory; the "one-ref-deleted, escape elsewhere" scenario silently misses the advisory while still firing `isolation-escape` (proves the new code is the load-bearing path).

4. **D-0019 decision record.** [D-0019](../../decisions/D-0019-oracle-partial-coverage-fail-shut-correctness-fail-open-coverage.md) records the fail-shut-on-correctness / fail-open-on-coverage choice; allocated at AC-3 cycle start before any RED test landed. The D-0019 id is mirrored into the milestone's `## Decisions made during implementation` section per ritual.

5. **Branch-spec cell registration.** Each of the 7 E2E scenarios above registers as a cell (1 silent-good baseline + 1 escape baseline + 5 oracle-state cells). AC-9 (G-0210) consolidates.

**Edge cases:**

- Corrupted `packed-refs` fails at `git for-each-ref` (whole-repo enumeration), not per-ref. The whole-oracle continues to fail-shut for this case — the D-NNN documents why (no ref enumeration means no rule coverage, period; advisory-per-ref makes no sense without a ref to name).
- Ref deleted between `for-each-ref` and `rev-list` is the TOCTOU case the per-ref tolerance is designed for; covered by the "one-ref-deleted mid-check" scenario.
- Non-ritual refs (`feature/foo`) are filtered before the first-parent index build, per the existing ritual-branch filter; not affected by this AC.
- The advisory-severity choice for `isolation-escape-oracle-failure` matches the M-0125 ratchet pattern (introduce as advisory; tighten to warning/error after one full epic of usage if false-positive rate stays low).
- **`OracleErrors` covers reflog-availability too**: the typed-error contract explicitly includes an `OracleErrReflogDisabled` entry surfaced when `core.logAllRefUpdates=false` is detected at gather time. AC-5 (G-0205 reflog-walk for force-push orphans) composes with this — the reflog-unavailability case rides AC-3's `isolation-escape-oracle-failure` advisory rather than introducing a separate finding code. The shape of the typed error names the affected capability (ref-resolution failure vs reflog-disabled vs shallow-clone per AC-4) so the advisory's hint text can name the specific remediation.

**References.**

- [G-0203](../../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md) + sub-concern N-4
- [`internal/check/isolation_escape.go`](../../../internal/check/isolation_escape.go) — the rule
- [`internal/cli/check/isolation_escape_oracle.go`](../../../internal/cli/check/isolation_escape_oracle.go) — `newGitBranchOracle`
- [`internal/cli/check/provenance.go`](../../../internal/cli/check/provenance.go) — `RunProvenanceCheck` (where oracle-failure handling wires in)
- [M-0106](M-0106-kernel-finding-isolation-escape-closes-g-0099.md) — the milestone that landed the original oracle
- AC-4..AC-7 — the four real-git scenarios depending on this typed-error contract
- [D-0019](../../decisions/D-0019-oracle-partial-coverage-fail-shut-correctness-fail-open-coverage.md) — fail-shut/fail-open decision record (allocated at AC-3 cycle start before any RED test landed)

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
   | `git clone --depth=5`, AI escape inside window | **silent** (oracle fails shut on shallow regardless of depth; the warning carries the operator to unshallow) | fires (warning — depth ≥1 is still shallow per `is-shallow-repository`) |
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
- **Detached-HEAD dangling commits are NOT in scope for this AC**. An AI commit made from detached HEAD that is never tied to a ref appears unreachable from every ritual branch's first-parent index, but the reflog has no force-update event referencing it (the commit was never on a ref to begin with). The reflog-walk finds no orphan event for it; this AC's finding stays silent. AC-7 (G-0207 detached HEAD) owns the dangling-commit-from-detached-HEAD case explicitly; the rule treats it as "no branch info" / KNOWN-GOOD-empty per AC-3's typed-error contract.

**References.**

- [G-0205](../../gaps/G-0205-branchoracle-silent-on-force-pushed-away-violating-commits.md) — the gap; closes here
- [`internal/cli/check/isolation_escape_oracle.go`](../../../internal/cli/check/isolation_escape_oracle.go) — where reflog-walk lands
- [`internal/check/isolation_escape.go`](../../../internal/check/isolation_escape.go) — the rule
- AC-3 (G-0203) — typed-error contract this composes with
- `aiwf acknowledge-illegal` — existing sovereign-override path (composes here)
- AC-9 (G-0210) — catalog refactor that consolidates cells

### AC-6 — BranchOracle resolves renamed branches via SHA fallback

**Scope of closure (honest).** This AC **closes G-0206 for post-AC-6 authorize scopes** — every authorize commit emitted after AC-6 lands carries the `aiwf-branch-sha:` trailer and benefits from rename transparency. **Pre-AC-6 ("legacy") authorize scopes are a documented carve-out**: they lack the SHA trailer; if their bound branch is renamed AND the old name no longer resolves AND there are AI commits on the renamed branch, the rule false-positives via the original G-0206 failure mode. This residual class is not closed here; the architectural completion path (`aiwf scope rebind` verb that records a follow-up SHA trailer on existing scopes) is tracked under **[G-0225](../../gaps/G-0225-legacy-scopes-lack-aiwf-branch-sha-trailer-rename-triggers-false-positive.md)**. Until that verb ships, legacy-scope operators have two workarounds (end-and-re-authorize, or per-commit `aiwf acknowledge-illegal`), documented inline in G-0225.

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

**Trailer-keys policy extension**: `aiwf-branch-sha:` lands in [`internal/policies/trailer_keys.go`](../../../internal/policies/trailer_keys.go) allowlist; the existing trailer-keys policy enforces canonical lowercase trailer-key shape (no value-shape arm). **SHA-shape validation for the trailer's VALUE lives at write-time in `aiwf authorize`** — the verb refuses to emit if the SHA it would record is not exactly 40 lowercase hex chars (the canonical git SHA-1 shape). This keeps the trailer-keys policy single-concern (key naming) and puts the data-shape check at the data's producer.

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
   | Legacy authorize commit, branch renamed | fires (error — **documented legacy carve-out**: pre-AC-6 scopes don't benefit from rename transparency; tracked as [G-0225](../../gaps/G-0225-legacy-scopes-lack-aiwf-branch-sha-trailer-rename-triggers-false-positive.md) for future `aiwf scope rebind` verb) |

2. **Sovereign-override path stays clean.** Existing `aiwf acknowledge-illegal <sha>` silences any remaining false positives (e.g., legacy authorize + rename case); no new override needed.

3. **Sabotage-verifiable.** Removing the SHA-fallback resolution makes the "rename + AI on renamed branch" scenario fire `isolation-escape` (the false-positive failure mode G-0206 names).

4. **Branch-spec cell registration.** 9 scenarios = 9 cells under `internal/workflows/spec/branch/`. AC-9 (G-0210) consolidates.

**Edge cases:**

- **Multi-step renames** (`foo → bar → baz`): SHA still resolves to `baz` (immutable through the chain). Single resolution step suffices.
- **Concurrent renames** (rename happens BETWEEN oracle build and rule run): TOCTOU race, vanishingly rare; documented as a fundamental property of the snapshot-style oracle.
- **`aiwf-branch-sha:` value validation**: the trailer must be a 40-char lowercase hex SHA-1; **the `aiwf authorize` verb refuses to emit on write if the value is malformed** (the trailer-keys policy itself enforces only key naming, not value shape — value-shape lives at the producer per Rust-style "validate at the boundary").
- **Detached HEAD at authorize time**: cannot record `aiwf-branch-sha:` because there's no branch tip. The existing `aiwf authorize` rejection for detached HEAD (AC-7's territory) prevents this from arising — AC-6 assumes a real branch at scope-open.
- **`aiwf authorize --branch` for a NON-EXISTENT branch** (the "future-branch carve-out" from M-0103 + M-0105): no current SHA to record. The trailer is OPTIONAL when the branch doesn't yet exist; rule falls back to name-only resolution until the branch is created. (This is the M-0102/M-0103 future-branch ritual-shape carve-out, unchanged here.)
- **Squat collision** (the trickiest case): rename `foo → bar`, then create new `foo` from an unrelated commit. Name-only resolution would find the squat; SHA-only resolution finds `bar`. Rule prefers SHA: the binding is to whatever SHA was recorded at scope-open, not to whatever label is currently attached to that name. This is the fundamentally correct semantic.

**References.**

- [G-0206](../../gaps/G-0206-branchoracle-false-positive-on-branch-renames-after-authorize.md) — the gap; closes here **for post-AC-6 scopes**
- [G-0225](../../gaps/G-0225-legacy-scopes-lack-aiwf-branch-sha-trailer-rename-triggers-false-positive.md) — legacy-scope carve-out; future `aiwf scope rebind` verb
- [`internal/verb/authorize.go`](../../../internal/verb/authorize.go) — extends with `aiwf-branch-sha:` trailer + write-time SHA-shape validation
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
   | Detached HEAD + `aiwf doctor --format=json` | exit 0; JSON envelope's `findings[]` contains a finding with `code: "detached-head"`, `severity: "advisory"` (asserted structurally against the parsed envelope, NOT via stdout substring) |
   | NOT detached + `aiwf doctor --format=json` | exit 0; JSON envelope's `findings[]` contains NO finding with `code: "detached-head"` (baseline) |

2. **Sovereign-override path is the 3rd row.** Pinned as a positive cell.

3. **Sabotage-verifiable.** Reverting the refined error message reverts to today's flat `branch-context-required` text; the discriminating test fires on missing "detached HEAD" substring **scoped to stderr** (verb-time errors don't carry structured codes/subcodes — see N-6 acknowledgment below; this is the load-bearing substring-discrimination exception). Reverting the `doctor` check removes the `detached-head` entry from the JSON envelope's findings — the structural assertion in the doctor scenarios above fires.

4. **Branch-spec cell registration.** 7 scenarios = 7 cells under `internal/workflows/spec/branch/`. AC-9 (G-0210) consolidates.

**Edge cases:**

- **Detached HEAD on a ritual branch's tip** (`git checkout epic/E-X` → `git switch --detach`): `git symbolic-ref --short HEAD` still exits non-zero (HEAD is detached, not symbolic) — refuse path correct. The operator is working on the branch's content but not "on" the branch; intent is ambiguous; refuse is the right default.
- **`git worktree add --detach`**: creates a worktree with detached HEAD. Same handling; no special case.
- **`aiwf check` during a `git rebase`-in-progress** (detached HEAD + rebase state files): the check proceeds silently. Rebase state is transient; the operator completes the rebase shortly. `aiwf doctor` may surface both `detached-head` AND a hypothetical `rebase-in-progress` advisory if that check exists (it doesn't today; out of scope here).
- **Refined error message in non-English locales**: not localized today; the substring "detached HEAD has no ritual context" is the canonical English form. Localization is a separate gap.
- **Substring-discrimination is the available signal at the verb-time error layer** (no error codes/subcodes today). Per CLAUDE.md "Substring assertions are not structural assertions", the test asserts against **stderr scoped to the error context** (not anywhere in stdout/stderr), which is the tightest pin available without changing the verb's error-shape API. The `aiwf doctor` side of this AC uses the JSON envelope's structured findings array, which IS a structural assertion (per the matrix rows above).

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

**Scope of closure (honest).** This AC **partially closes G-0209** — specifically the **promote-side ordering** failure mode. G-0209 also names the symmetric authorize-side ordering case ("the authorize commit lands on the epic branch instead of main"). That case splits two ways:

- **Authorize with explicit `--branch <ritual>` from a same-rung branch**: refused by AC-2's rung-pair predicate (e.g., on `epic/E-NN-foo` and `--branch epic/E-NN-foo` → (epic, epic) cross-epic-typo, refused at verb-time). ✅ closed by AC-2.
- **Authorize WITHOUT `--branch` from a ritual-current branch** (the implicit-ritual-current path): M-0103/M-0105's existing carve-outs explicitly accept this shape because it supports legitimate cases like "operator on epic/E-NN-foo authorizes child milestone work" (which lands on the epic branch correctly). **The G-0209 case "AI cuts epic/E-NN-foo first, then authorizes E-NN scope on epic branch without --branch"** rides this same carve-out and is **NOT refused** — neither by AC-2 nor by this AC. This is the residual G-0209 case.

**Status of the residual**: tracked as operator-discipline. If it surfaces as a recurring incident class, future work could either (a) extend AC-2 to refuse the implicit-current path when the scope's target-rung doesn't match the current branch's rung, or (b) extend this AC's rule to cover authorize commits as well as promote commits, or (c) add an `aiwf doctor authorize-on-wrong-rung` advisory check. None of those is in scope for M-0161 — the carve-out semantics that allow the implicit-current path are load-bearing for legitimate ritual flows.

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

- [G-0209](../../gaps/G-0209-ritual-step-ordering-is-advisory-only-no-kernel-enforcement.md) — the gap; **partially closes here** (promote-side; authorize-side implicit-current path is residual operator-discipline)
- [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md) — branch model that the rule enforces
- [`internal/check/`](../../../internal/check/) — new rule `promote_on_wrong_branch.go` lands here
- [`internal/cli/check/provenance.go`](../../../internal/cli/check/provenance.go) — wiring into `RunProvenanceCheck`
- AC-1 (G-0200) — composes via `Config.TrunkBranchShortName`
- AC-2 (G-0201) — handles the authorize-side ordering with explicit `--branch` (closed); does NOT cover the implicit-current path (residual)
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

**Net catalog after AC-9 (final)**: 7 (M-0158 retained) + 66 (M-0161 ACs 1–8) + 3 (AC-9's own meta-cells, listed in §"Meta-cell registration" below) = **76 cells**.

**Part 3 — Bijection enforcement (`branchcell.Pin` registry):**

A new **test-only** registry under [`internal/workflows/spec/branch/pin_test_helpers.go`](../../../internal/workflows/spec/branch/) — the `_test_helpers.go` suffix is Go-convention-shaped (matches the `*_test.go` invariant that the file is only compiled into test binaries, never into production builds). Alternative: a `pin.go` file gated by `//go:build testpins` build tag and the M-0161 test Makefile target adds `-tags testpins`. Either shape keeps the registry out of production binaries cleanly — **no `testing.Testing()` runtime guard or production-side panic needed** (the registry simply does not exist outside test compilation).

```go
//go:build testpins
// +build testpins

package branch

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

**Replaces M-0158/AC-5's keyword-set meta-coverage approach.** M-0158/AC-5's claim text is: *"Each branch-cell has at least one test that exercises the cell's view-keyword set"* (the keyword-set heuristic at `internal/policies/m0158_ac5_meta_coverage_test.go`). The bijection meta-test pins a **strictly stronger** claim: *"Each branch-cell has exactly one Pin call from exactly one test function (1:1)"*. The bijection check supersedes the keyword-set check on every axis it covered:

- keyword-set "≥1 test references the cell's keyword" → bijection "exactly 1 Pin references the cell" (covered + tightened).
- keyword-set silently double-mapped cells (the G-0210 signal-weakness) → bijection forbids 2+ Pins per cell (fixes the weakness explicitly).
- keyword-set was orphan-blind → bijection forbids orphan Pins (closes a gap the keyword-set had).

The keyword-set file `internal/policies/m0158_ac5_meta_coverage_test.go` is removed in the same AC-9 commit. M-0158/AC-5's promoted-met status remains valid because the bijection meta-test maintains (and strengthens) every invariant the keyword-set test asserted.

**Part 4 — Drift policy extension:**

The existing M-0158 drift policy is extended to read from the `branchcell.Pin` registry. A cell added to `branch.Rules()` without a paired `Pin` call fails CI; a `Pin` without a corresponding cell fails CI.

**Mechanical assertions:**

1. **Catalog refactor verification.** A test under [`internal/policies/branch_cell_catalog_test.go`](../../../internal/policies/branch_cell_catalog_test.go) asserts:
   - The 9 dropped cells are ABSENT from `branch.Rules()`
   - The 7 M-0158 retained cells are PRESENT
   - The 66 M-0161 AC-1..AC-8 cells are PRESENT
   - The 3 AC-9 meta-cells are PRESENT
   - **Total cell count == 76** (7 retained + 66 AC-1..AC-8 + 3 meta; pinned exact count to catch drift)

2. **Bijection enforcement test.** The 4 invariants above each have a dedicated subtest. Each subtest is sabotage-verifiable:
   - Remove a Pin from a test → "cell with no Pin" subtest fires.
   - Add a Pin for a non-existent cell → "orphan Pin" subtest fires.
   - Add a 2nd Pin to an existing cell → "double-mapping" subtest fires.
   - Add a 2nd Pin from a test function → "overload" subtest fires.

3. **M-0158 keyword-set removal verification.** The file `internal/policies/m0158_ac5_meta_coverage_test.go` is deleted in the AC-9 commit; a structural test asserts the file does not exist (prevents reintroduction).

4. **Drift policy extension.** Fixture tests under [`internal/policies/branch_cell_drift_test.go`](../../../internal/policies/branch_cell_drift_test.go) exercise:
   - Add cell, no Pin → fails CI
   - Remove cell, leave Pin → fails CI
   - Healthy state (1:1 across all 76) → silent

5. **Meta-cell registration.** AC-9 produces three meta-cells in the catalog:
   - `branch-cell-meta-bijection-enforced` (positive — bijection holds across all 76 cells)
   - `branch-cell-meta-pin-orphan-detected` (positive — orphan Pin produces failure)
   - `branch-cell-meta-cell-orphan-detected` (positive — cell with no Pin produces failure)

   These are the 3 meta-cells counted in the **76-cell total**.

**Edge cases:**

- **M-0158 wrap unaffected**: the catalog refactor is additive in the "right direction" (drop 9, add 66) — M-0158's wrap statements about its 7 retained cells stay true; the 9 dropped cells were documented as carrying no mechanical weight already.
- **Test-only files**: `branchcell.Pin` lives at `pin_test_helpers.go` (or under `//go:build testpins` per Part 3) so it is **never compiled into production binaries**. Tests across multiple `_test.go` files can call it. Pins accumulate in a `var pins []pin` package var with a mutex for parallel-test safety. The test-only-file shape replaces the originally-considered runtime `testing.Testing()` guard — build-tag isolation is the canonical Go pattern for this concern and avoids the CLAUDE.md "no new package-level mutable state" tension entirely (the state simply does not exist in production).
- **Bijection meta-test parallelism**: runs serial via `setup_test.go` conventions (registers a non-parallel skip per the M-0091 cap rule; reading the global registry safely is what justifies the skip).
- **Cells added in later milestones** (future): the bijection enforcement catches future drift — anyone adding a cell without a Pin fails CI.

**References.**

- [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md) — the gap; closes here
- [M-0158](M-0158-layer-4-branch-choreography-spec-cells-drift-policy-extension.md) — the milestone with the over-specified catalog (refactor closes here)
- [`internal/workflows/spec/branch/`](../../../internal/workflows/spec/branch/) — package layout
- [`internal/policies/m0158_ac5_meta_coverage_test.go`](../../../internal/policies/m0158_ac5_meta_coverage_test.go) — keyword-set meta-test (REMOVED by AC-9)
- AC-1..AC-8 — sources of the 66 new cells

## Decisions made during implementation

- [D-0018](../../decisions/D-0018-branch-not-found-subsumed-by-rung-pair-illegal-catalog-cleanup-defers-to-ac-9.md) — branch-not-found subsumed by rung-pair-illegal; catalog cleanup defers to AC-9 (AC-2 reviewer pass S-3)
- [D-0019](../../decisions/D-0019-oracle-partial-coverage-fail-shut-correctness-fail-open-coverage.md) — oracle partial-coverage: fail-shut on rule correctness, fail-open on rule coverage (AC-3 cycle start, pre-RED)

