# Provenance model build plan (I2.5)

**Status:** proposal · **Audience:** PoC iteration I2.5, between I2 (acceptance criteria + TDD) and I3 (governance HTML render).

This plan implements the design in [`provenance-model.md`](../design/provenance-model.md). Read that doc first; this is the step-by-step build sequence.

I2.5 is a coherent kernel pass: identity, accountability, scope lifecycle, the new authorize verb, the standing audit rules, and the rendering integration all touch the same trailer-writer and config layers. Splitting them across iterations would force partial implementations to land in the kernel's `--help` and `aiwf check` output, which violates the AI-discoverability rule.

---

## 0. Preconditions

Land before starting any I2.5 step:

| Prerequisite | Where defined | Why I2.5 needs it |
|---|---|---|
| **I2 step 11 — reverse-reference index on `aiwf show`** | [`acs-and-tdd-plan.md`](acs-and-tdd-plan.md) §11 step 11 | Step 6 (allow-rule composition) calls `Reaches(from, to)` / `ReachesAny(froms, to)` to gate non-human-actor verbs against the scope-entity. Both helpers are built on top of the in-memory reverse-ref index that step 11 produces. Without it, step 6 has no reference graph to query. |
| **I2 steps 1–10 (acceptance criteria + TDD)** | [`acs-and-tdd-plan.md`](acs-and-tdd-plan.md) §11 steps 1–10 | Composite-id grammar (`M-NNN/AC-N`), the `aiwf-to:` trailer, `aiwf show`'s `ShowView`, and the `--force --reason` flag pair are all assumed by I2.5's verb surface and trailer rules. |

If a step in I2.5 is started before its precondition lands, that step's tests will reach for symbols that don't exist (`tree.Reaches`, `entity.ParseCompositeID`, etc.) and the build will fail loudly — but the failure mode is "obvious type error," not "subtle correctness bug." Still, sequencing matters: the prerequisite is one focused commit, not a sweep.

### Within-iteration build order

The numbered steps in §2 are not strictly sequential; several can land in parallel. The actual DAG:

```
1 (identity migration)
   └── 2 (trailer writer extensions)
            ├── 3 (coherence rules)
            │       │
            │       ├── 5 (authorize verb)
            │       │       │
            │       │       ├── 5b (--audit-only, G24)   ← depends on 3 for the mutex rule
            │       │       │
            │       │       └── 6 (allow-rule)            ← also depends on 4 + I2 step 11
            │       │
            │       └── (no further fan-out from 3)
            │
            ├── 4 (scope FSM)
            │       │
            │       ├── 6 (allow-rule, see above)
            │       │
            │       └── 7 (standing rules)                ← also depends on 2
            │               │
            │               └── 7b (G24 trailer audit)   ← independent of the rest of 7; can land in parallel
            │
            └── 5c (Apply lock diagnostic)               ← independent; can land any time after 2
```

After 6 + 7 + 7b land:

```
8 (history rendering)            ← reads from the new trailer set
9 (show envelope additions)      ← reads from scope FSM (step 4) + new trailers (step 2)
10 (documentation + skills)      ← reflects the now-stable verb / flag / finding surface
11 (render handoff to I3)        ← placeholder; actual work in I3
```

**Suggested commit cadence:** 1 → 2 → (3, 4, 5c in any order) → 5 → (5b, 6, 7 in any order) → 7b → 8 → 9 → 10. Each step is one commit; no half-finished implementations across commits.

---

## 1. Site shape (what changes in the codebase)

| Area | Files |
|---|---|
| Trailer writer / parser | `tools/internal/gitops/` |
| Verb surface (new `aiwf authorize`; updated `--actor` / `--principal` flags on existing verbs) | `tools/cmd/aiwf/`, `tools/internal/verb/` |
| Config layer (drop `aiwf.yaml.actor`; runtime-derive identity) | `tools/internal/aiwfyaml/`, `tools/internal/config/` |
| Scope FSM | `tools/internal/scope/` (new package) |
| Allow-rule composition | `tools/internal/verb/allow.go` (new) |
| Standing-rule check codes | `tools/internal/check/provenance.go` (new) |
| `aiwf history` rendering | `tools/cmd/aiwf/history_cmd.go` |
| `aiwf show` envelope | `tools/cmd/aiwf/show_cmd.go` (small additions) |

Reference-graph reachability uses the index built in `acs-and-tdd-plan.md` step 11. That step is a **load-bearing prerequisite for I2.5**; both must be in place before I3.

---

## 2. Build plan

### Step 1 — Identity migration (drop `aiwf.yaml.actor`)

- [ ] In `tools/internal/aiwfyaml/`: remove the `actor` field from the struct and the YAML tag. Round-trip tests updated to confirm an `actor:` key in incoming YAML is ignored (with a deprecation warning during a transition period).
- [ ] In `tools/internal/config/` (or wherever runtime identity is resolved): new `ResolveActor(args []string, gitConfig GitConfig) (Actor, error)` function with precedence `--actor` flag > `git config user.email` > error.
- [ ] `git config user.email` parsing produces `human/<localpart>` by stripping the domain and slugifying the local part using the existing `entity.Slugify` (drops chars per G8 if needed).
- [ ] `aiwf init` no longer writes `actor:` to `aiwf.yaml`. It validates that `git config user.email` is set; refuses to init if not.
- [ ] `aiwf doctor` validates `git config user.email` is set and the derived `<role>/<id>` matches the regex.
- [ ] Tests: precedence order (flag overrides config); missing-email error; malformed-email error; backwards-compat behavior (existing `aiwf.yaml.actor` is ignored with a one-time deprecation note in `aiwf doctor`).

### Step 2 — Trailer writer extensions

- [ ] In `tools/internal/gitops/`: register the new trailer keys: `aiwf-principal`, `aiwf-on-behalf-of`, `aiwf-authorized-by`, `aiwf-scope`, `aiwf-scope-ends`, `aiwf-reason`.
- [ ] Trailer writer accepts the new fields on the existing trailer-set struct; emits in deterministic order (existing trailers first, then I2.5 trailers in the order above).
- [ ] Write-time shape validators per trailer:
  - `aiwf-principal:` and `aiwf-on-behalf-of:` — `<role>/<id>` regex AND role must start with `human/`.
  - `aiwf-authorized-by:` and `aiwf-scope-ends:` — 7–40 hex.
  - `aiwf-scope:` — closed set `{opened, paused, resumed}`.
  - `aiwf-reason:` — non-empty after trim. Carries the free-text rationale for verbs that require one (pause/resume today; future non-force, non-audit-only verbs that grow a reason field). Distinct from `aiwf-force:` (sovereign override) and `aiwf-audit-only:` (backfill rationale, step 5b) — each reason-bearing trailer carries its own semantic.
- [ ] Trailer reader (extending the existing one) tolerates absent fields (for pre-I2.5 commits) and unknown fields (forward compatibility).
- [ ] Tests: round-trip of every new trailer key; shape validation rejects malformed values at write time; reader tolerance on pre-I2.5 fixtures; ordering deterministic.

### Step 3 — Required-together / mutually-exclusive verb-side rules

- [ ] In `tools/internal/verb/` (or a new `verb/coherence.go`): `CheckTrailerCoherence(set TrailerSet) error` returns a typed error citing the specific rule violated.
- [ ] Rules implemented:
  - Required-together: `(on-behalf-of, authorized-by)`; `(principal, non-human actor)`.
  - Mutually exclusive: `(force, on-behalf-of)`; `(principal, human actor)`; `(on-behalf-of, human actor)`.
  - Force human-only: `(force, non-human actor)` is forbidden.
- [ ] Every mutating verb's `Apply` path calls `CheckTrailerCoherence` after assembling the trailer set and before committing.
- [ ] Tests: each rule fires its own typed error; happy-path trailer sets pass; combinations from §2 of the design doc all assert the right rule.

### Step 4 — Scope FSM package

- [ ] New package `tools/internal/scope/` with:
  - `State` enum: `active`, `paused`, `ended`.
  - `Scope` struct: `AuthSHA string`, `Entity string`, `Agent Actor`, `Principal Actor`, `OpenedAt time.Time`, `Events []ScopeEvent`, `State State`.
  - `LoadScope(authSHA string, history []Commit) (Scope, error)` — walks history forward from the authorize commit, applying transitions in commit order, returns the scope's current state and event list.
  - `IsLegalScopeTransition(from, to State) bool` — closed-set FSM.
- [ ] Auto-end derivation: a commit carrying `aiwf-scope-ends: <auth-sha>` ends the named scope. The terminal-promote verb writes this trailer (see step 6).
- [ ] Tests: FSM transitions (legal/illegal pairs); event-replay correctness across multiple pause/resume cycles; auto-end on terminal-promote; un-cancel-after-end does NOT resurrect the scope.

### Step 5 — `aiwf authorize` verb

- [ ] New file `tools/internal/verb/authorize.go`. The verb has three modes:
  - `aiwf authorize <id> --to <agent> [--reason "<text>"]` — open scope.
  - `aiwf authorize <id> --pause "<reason>"` — pause the most-recently-opened active scope for `<id>`.
  - `aiwf authorize <id> --resume "<reason>"` — resume the most-recently-paused scope for `<id>`.
- [ ] Refusal rules: actor must be `human/...`; for `--to`, the scope-entity must not be in a terminal status (overridable with `--force --reason`); for `--pause` / `--resume`, the scope state must be the corresponding source state.
- [ ] Each invocation produces exactly one commit with the trailer set:
  - `--to`: `aiwf-verb: authorize / aiwf-entity: <id> / aiwf-actor: human/... / aiwf-to: <agent> / aiwf-scope: opened`. When `--reason` is supplied (optional for `--to`), append `aiwf-reason: <text>`.
  - `--pause`: `aiwf-verb: authorize / aiwf-entity: <id> / aiwf-actor: human/... / aiwf-scope: paused / aiwf-reason: <text>` (reason required, non-empty after trim).
  - `--resume`: `aiwf-verb: authorize / aiwf-entity: <id> / aiwf-actor: human/... / aiwf-scope: resumed / aiwf-reason: <text>` (reason required, non-empty after trim).
- [ ] Verb-side `CheckTrailerCoherence` validates the assembled set before commit.
- [ ] Tests: open / pause / resume / re-pause / re-resume cycles; refusal on terminal scope-entity without `--force`; refusal on non-human actor; refusal on missing scope state for pause/resume; one-commit-per-invocation; the authorize commit is reachable by SHA in subsequent verb invocations.

### Step 5b — `--audit-only --reason` recovery mode (G24)

Closes the recovery half of [G24](../gaps.md#g24-manual-commits-bypass-aiwf-verb-trailers-no-first-class-repair-path-open). When a mutating verb fails partway through and the operator finishes the work with a plain `git commit`, there is currently no first-class way to backfill the missing audit trail. This step adds that path.

- [ ] New flag pair on `aiwf cancel` and `aiwf promote`: `--audit-only --reason "<text>"`. Mutex with `--force` (force is for *making* a transition; audit-only is for *recording* one that already happened).
- [ ] Behavior: when `--audit-only` is set, the verb skips the FSM legality check, skips the file-mutation step (writes nothing to disk), and produces an empty-diff commit carrying the standard trailer block (`aiwf-verb`, `aiwf-entity`, `aiwf-actor`, `aiwf-to`, plus the new I2.5 trailers as applicable). The trailer additionally carries `aiwf-audit-only: <reason>` so the commit is distinguishable from a normal verb commit at read time.
- [ ] New trailer key `aiwf-audit-only:` registered in `tools/internal/gitops/` (write + read path; reuses the `aiwf-force:` shape — non-empty after trim).
- [ ] Refusal rules: `--reason` required; `--audit-only` requires the entity to *already* be at the named target state (verb refuses if not — the rationale is "this verb only records what's already true"). For composite ids (`M-NNN/AC-N`), the same rule applies to AC status / phase.
- [ ] Verb-side `CheckTrailerCoherence` (step 3) accepts `aiwf-audit-only:` alongside the existing trailers; the mutex `(audit-only, force)` joins the rule set.
- [ ] Provenance: `--audit-only` is itself a sovereign act in the same way `--force` is — kernel refuses non-human actors. `provenance-audit-only-non-human` (error) added to the standing-rule set in step 7.
- [ ] `aiwf history` renders audit-only events with a distinct chip (`[audit-only]`) and the reason inline, mirroring the `--force` rendering convention.
- [ ] Tests: load-bearing scenario from G24 (entity already at `wontfix` after a manual commit; `aiwf cancel <id> --audit-only --reason "..."` produces a properly-trailered empty-diff commit; `aiwf history <id>` now shows the event); refusal when entity is not at the target state; refusal on non-human actor; mutex with `--force`; one-commit-per-invocation.

### Step 5c — Diagnostic instrumentation in `Apply` (G24)

Closes the root-cause-diagnosis half of G24. Today `Apply` treats every commit failure as fatal and surfaces the underlying error verbatim. When the failure is `.git/index.lock` contention from an external process (VS Code's git extension, a file-watcher, a stale lock from a prior crash), the operator gets a generic message and no signal about who's holding the lock.

- [ ] In `tools/internal/verb/apply.go`: when the `git commit` subprocess fails, classify the stderr. Specifically detect `index.lock` (or `.git/index.lock`) substrings and route to a new `applyError` subtype `lockContention`.
- [ ] On `lockContention`, attempt a best-effort lock-holder lookup: `lsof <repo>/.git/index.lock` (Unix only; macOS + Linux). If `lsof` is missing or the lookup fails, fall back to the bare error message — never block the user on diagnostic gathering.
- [ ] Surface a multi-line error: original stderr, the holder PID + process name (when discoverable), and a one-line hint pointing at G24's `--audit-only` recovery path if the user already finished the work manually.
- [ ] **No retry policy.** The kernel does not silently retry on lock contention — silent retries hide real environmental problems and can race against the holder. The operator decides: wait, kill the holder, or use `--audit-only` after a manual commit.
- [ ] Tests: stderr classification (lock vs. other failures); `lsof` success path with a fixture file held by a sleeping subprocess; `lsof` missing / failing gracefully degrades.

### Step 6 — Allow-rule composition + scope-aware verb dispatch

- [ ] New `tools/internal/verb/allow.go`: `Allow(verb Verb, target Entity, actor Actor, scopes []Scope, refIndex ReferenceIndex) AllowResult`. Returns: allowed/denied; the matching scope (if any); and the diagnostic for refusals.
- [ ] For human actors with no `--principal`: scope check is skipped (returns allowed iff entity-FSM allows the verb).
- [ ] For non-human actors: at least one active scope's `scopeAllows` must return true. If multiple match, pick the most-recently-opened deterministically.
- [ ] On allow: `Apply` writes `aiwf-on-behalf-of:` (= scope.Principal) and `aiwf-authorized-by:` (= scope.AuthSHA) into the trailer set.
- [ ] On deny: verb refuses with `provenance-no-active-scope` (typed error → `aiwf check`-shaped finding).
- [ ] Reference-graph reachability uses the I2-step-11 index; new functions in `tools/internal/tree/` if needed: `Reaches(from string, to string) bool`, `ReachesAny(froms []string, to string) bool` (for creation acts).
- [ ] **Scope-entity resolution walks the `aiwf-prior-entity:` chain.** When the scope-entity id from an authorize commit's `aiwf-entity:` trailer no longer matches a current entity (because `aiwf reallocate` renumbered it after the scope was opened), the resolver follows the existing rename-chain forward to the current id before running the reachability check. Reuses the prior-entity chain primitive that `aiwf history` already consults; no new trailer key. Historical authorize commits stay byte-identical, so their SHAs remain valid as `aiwf-authorized-by:` targets.
- [ ] **Scope-end side effect on terminal promote:** when `Apply` is processing a `promote` verb whose target state is terminal for the entity's kind, it queries all active scopes whose scope-entity is the verb's target, and writes one `aiwf-scope-ends: <auth-sha>` trailer per matched scope into the same commit.
- [ ] Tests: every scenario from `provenance-model.md` "Worked examples" (six examples, each with the exact expected trailer set); pivot-to-other-epic + return; scope-end on epic-done; refusal when agent acts outside any active scope.

### Step 7 — `aiwf check` standing rules

- [ ] New file `tools/internal/check/provenance.go` registering the finding codes from `provenance-model.md` §"`aiwf check` rules":
  - `provenance-trailer-incoherent`
  - `provenance-force-non-human`
  - `provenance-actor-malformed`
  - `provenance-principal-non-human`
  - `provenance-on-behalf-of-non-human`
  - `provenance-authorized-by-malformed`
  - `provenance-authorization-missing`
  - `provenance-authorization-out-of-scope`
  - `provenance-authorization-ended`
  - `provenance-no-active-scope`
  - `provenance-audit-only-non-human` (added by step 5b)
- [ ] Each rule walks `git log` once per check pass and indexes by trailer key for O(1) lookup. The authorization-resolution rules (`-missing` / `-out-of-scope` / `-ended`) build a single `authSHA → Scope` map at the start of the pass.
- [ ] Hint table extended with one entry per finding code: link to `aiwf authorize --help` for `-no-active-scope`, link to `aiwf doctor` for `-actor-malformed`, link to `aiwf cancel --audit-only` for `-untrailered-entity-commit` (see step 7b), etc.
- [ ] Tests: per-finding fixture commits (intentionally malformed) under `tools/internal/check/testdata/messy/`; clean fixtures continue to produce zero findings; backwards-compat assertion (pre-I2.5 commits with single `aiwf-actor:` produce no provenance findings).

### Step 7b — Pre-push trailer audit (G24)

Closes the surface-the-gap half of [G24](../gaps.md#g24-manual-commits-bypass-aiwf-verb-trailers-no-first-class-repair-path-open). When a manual commit lands on an entity file without `aiwf-verb:`, the framework currently goes silent — `aiwf history` and `aiwf status` filter it out and the audit trail has an unsignalled hole. This step makes the hole visible at push time.

- [ ] New finding `provenance-untrailered-entity-commit` (warning) in `tools/internal/check/provenance.go`. Trigger: a commit between `@{u}` and `HEAD` (or all of `HEAD` when no upstream exists) touches at least one file under `work/` and carries no `aiwf-verb:` trailer.
- [ ] Detection walks the same `git log` pass step 7 already uses. For each candidate commit, classify the touched paths via the existing `tree.PathKind` helper; ignore commits that only touch non-entity files (`STATUS.md`, `aiwf.yaml`, `.claude/`, etc.).
- [ ] Severity is **warning**, not error: G24's recovery path (`--audit-only`, step 5b) is the user's intended response. Errors would block the push when the entity state is correct; the warning surfaces the audit-trail hole without forcing a synchronous fix.
- [ ] Hint message names the offending commit SHA + file paths and points at `aiwf cancel --audit-only` / `aiwf promote --audit-only` as the repair path.
- [ ] Tests: a fixture branch with one trailered commit + one manual entity commit produces exactly one `provenance-untrailered-entity-commit` finding; a manual commit touching only non-entity files produces zero findings; pre-I2.5 commits already on `main` (i.e., already in `@{u}`) are ignored.

### Step 8 — `aiwf history` rendering

- [ ] In `tools/cmd/aiwf/history_cmd.go`: text formatter renders the actor column with `principal via agent` syntax when `aiwf-principal:` is present; trailing `[scope-id]` chip when `aiwf-authorized-by:` is present (scope-id = first 7 chars of the auth SHA, plus the scope-entity id from a one-time index lookup, e.g., `[E-03 4b13a0f]` or just `[E-03]` when unambiguous in the visible window).
- [ ] Pause/resume events render with `[E-03 paused]` / `[E-03 resumed]` chips; auto-end events (rows carrying `aiwf-scope-ends:`) render `[E-03 ended]`.
- [ ] New flag `--show-authorization` adds an authorization-SHA column.
- [ ] `--format=json` emits the full trailer set; the JSON envelope has explicit fields for each new trailer.
- [ ] Tests: golden text output across the worked-example scenarios; JSON shape covers every trailer; legacy (pre-I2.5) rows render unchanged.

### Step 9 — `aiwf show` envelope additions

- [ ] In `tools/cmd/aiwf/show_cmd.go`: extend `ShowView` with a `scopes []ScopeView` field listing every scope that ever applied to this entity. `ScopeView`: `{auth_sha, agent, principal, opened, state, ended_at, event_count}`.
- [ ] Populated by walking the entity's history once and extracting `aiwf-authorized-by:` SHAs; for each, load the scope via the package from step 4.
- [ ] `aiwf show --help` enumerates the new field. Embedded skill `aiwf-show` (or equivalent) updated.
- [ ] Tests: golden JSON files per kind covering scopes presence/absence; entity that lived through multiple scopes serially renders all of them in chronological order.

### Step 10 — Documentation and embedded skills

- [ ] `aiwf authorize --help` documents the three modes, the `--to` / `--pause` / `--resume` flags, the human-only rule.
- [ ] `aiwf <verb> --help` documents the `--actor` and `--principal` flags wherever they apply.
- [ ] `aiwf check --help` lists the new finding codes.
- [ ] `aiwf doctor` reports `git config user.email` status.
- [ ] New embedded skill: `aiwf-authorize` under `tools/internal/skills/embedded/`. Mentions: when the LLM is a tool vs. an agent (per CLAUDE.md), how to set `--principal` from session context, when to expect `provenance-no-active-scope` vs. `provenance-authorization-out-of-scope`.
- [ ] Existing skills updated where relevant: `aiwf-add`, `aiwf-promote`, `aiwf-history`, `aiwf-show`.
- [ ] Per the AI-discoverability rule (CLAUDE.md): every new flag, trailer key, finding code, and YAML field is reachable through `aiwf <verb> --help` or an embedded skill.

### Step 11 — Render integration (governance HTML)

- [ ] `governance-html-plan.md` §3.3 Provenance tab spec already references this iteration. Per Q3.3 (scope-as-section), the tab renders:
  - Top: scopes table (auth SHA short form, agent, principal, opened, state, ended, event count).
  - Below: chronological timeline with scope chips.
- [ ] Render-side changes are scoped to `governance-html-plan.md` step 6 ("Cross-cutting render details"), which gains the scope-rendering deliverables. No I2.5 work here; this step is a placeholder noting the cross-iteration handoff.

---

## 3. What is NOT in scope

Per `provenance-model.md` §"Open extensions":

| Feature | Tracked as |
|---|---|
| `aiwf revoke <auth-sha>` | G22 |
| Time-bound scopes (`--until`, `--for`) | G22 |
| Verb-set restrictions (`--verbs`) | G22 |
| Pattern scopes (`--pattern`) | G22 |
| Sub-agent delegation | G22 (and Q3.6b's deferred mutually-exclusive pair) |
| Bulk-import per-entity actor attribution | G22 |
| Delegated `--force` (`aiwf authorize --allow-force`) | G23 |
| `aiwf check --explain` mode | future polish; not load-bearing for I2.5 |

YAGNI for the PoC. If real friction shows up, revisit.

---

## 4. Test scenarios

The test surface for I2.5 is large. The load-bearing scenarios — each must pass before the iteration is considered complete:

1. **Solo human direct verb** — single `aiwf-actor:` trailer, no provenance findings.
2. **Human directs LLM** — `aiwf-actor: ai/claude` + `aiwf-principal: human/peter`, no scope.
3. **Open scope, scoped verb, close scope on terminal** — full trailer set; auto-end via `aiwf-scope-ends:`.
4. **Pivot mid-flight** — pause E-03, open E-09, work on E-09, pause E-09, resume E-03; trailer SHAs route correctly per commit.
5. **Out-of-scope refusal** — agent attempts a verb on an entity that doesn't reach the scope-entity; verb refuses; `aiwf check` confirms no malformed commit landed.
6. **Stale authorization SHA** — three sub-cases: typo (missing), wrong-scope (out-of-scope), ended scope (ended). Each fires the correct finding code.
7. **Force is human-only** — LLM `--force` refuses; human `--force` succeeds and writes only `aiwf-actor: human/...` + `aiwf-force:`.
8. **Reallocation preserves authorization references** — when a scope-entity is reallocated (e.g., `M-007 → M-019`), historical authorize commits stay byte-identical (their SHAs remain valid). The standing-rule resolver walks the existing `aiwf-prior-entity:` chain when matching `aiwf-entity:` trailers from authorize commits against current entity ids. Subsequent agent verbs operating under the scope continue to use the same `aiwf-authorized-by:` SHA; the scope-entity reachability check resolves through the prior-entity chain. Test: open scope on M-007 → reallocate M-007 → M-019 → agent verb on a new milestone under M-019 — verb is allowed (chain resolves), no `provenance-authorization-out-of-scope` finding fires.
9. **Multi-clone identity correctness** — second developer clones the repo and runs verbs; their trailers say `human/<their-id>`, not the original committer's.
10. **Backwards compatibility** — pre-I2.5 commits in fixtures produce zero provenance findings; `aiwf history` renders them with their single-actor format unchanged.
11. **G24 audit-only recovery** — entity reaches `wontfix` via a manual commit (no `aiwf-verb:` trailers). `aiwf check` fires `provenance-untrailered-entity-commit` (warning) on push; `aiwf cancel <id> --audit-only --reason "..."` produces a properly-trailered empty-diff commit; `aiwf history <id>` now shows the cancellation event; the warning clears on the next push.
12. **G24 lock-contention diagnostic** — fixture process holds `.git/index.lock`; `aiwf cancel <id>` fails with the multi-line diagnostic naming the holder PID + a hint pointing at `--audit-only`; the kernel does not retry.

---

## 5. Status

| Step | State | Owner |
|---|---|---|
| 1 — Identity migration (drop `aiwf.yaml.actor`) | done | core |
| 2 — Trailer writer extensions | done | core |
| 3 — Required-together / mutually-exclusive rules | done | core |
| 4 — Scope FSM package | done | core |
| 5 — `aiwf authorize` verb | done | core |
| 5b — `--audit-only --reason` recovery mode (G24) | done | core |
| 5c — Diagnostic instrumentation in `Apply` (G24) | done | core |
| 6 — Allow-rule composition + scope-aware dispatch | done | core |
| 7 — `aiwf check` standing rules | done | core |
| 7b — Pre-push trailer audit (G24) | done | core |
| 8 — `aiwf history` rendering | done | core |
| 9 — `aiwf show` envelope additions | done | core |
| 10 — Documentation and embedded skills | done | core |
| 11 — Render integration handoff | proposed | core (executed in I3) |
