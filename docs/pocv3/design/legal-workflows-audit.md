# Legal-workflow audit catalog (Pass A)

> **Methodology:** ADR-0011. **Milestone:** M-0121. **Status:** in_progress.
>
> This is **Pass A**'s working artifact — the audit of every legality
> statement already encoded somewhere in the repo. Pass B (M-0122)
> derives the same surface independently from first principles. Pass C
> (M-0123) reconciles A and B into the canonical Go spec table. This
> document is **not the spec** and is **not the source of truth** for
> what aiwf permits — it is a structured record of what *each existing
> source claims*, with citations, for Pass C to reconcile against.

## Schema

Every rule is a row in a per-source markdown table with six columns:

| Column | Meaning |
|---|---|
| **Rule id** | `R-AUDIT-NNNN` — sequential within this catalog, starting at 0001 |
| **Source** | One of the nine source identifiers below |
| **Citation** | `file.go:line` for code sources, named anchor (`§Section`, ADR-NNNN§Decision) for prose |
| **Scope** | Which kind, verb, or pair of kinds the rule applies to |
| **Statement** | The legality claim, expressed in one sentence |
| **Severity if violated** | `hard-reject` (verb errors) / `check-error` / `check-warning` / `unenforced` (prose-only, no mechanical chokepoint) |

## Sources (order = audit order, most-mechanical first)

1. **FSM tables** — `internal/entity/transition.go`
2. **Mechanical policies** — `internal/policies/*.go`
3. **Check rules** — `internal/check/*.go`
4. **Cobra verb definitions** — `cmd/aiwf/`, `internal/cli/<verb>/`
5. **ADRs** — `docs/adr/`
6. **Kernel commitments** — `docs/pocv3/design/design-decisions.md`
7. **Repo principles** — `CLAUDE.md`
8. **Skills** — `.claude/skills/`, rituals plugin
9. **Verb help text** — `aiwf <verb> --help`

## Rule extraction

---

### 1. FSM tables — `internal/entity/transition.go`

Per-kind status FSM plus the AC FSM, the TDD-phase FSM, the `CancelTarget` mapping, and the `MilestoneCanGoDone` precondition. Lines below are 1-indexed file positions.

#### Entity status FSM — one rule per `(kind, from-state, to-state)` legal cell

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0001 | transition.go | L14 | Epic | `promote E-NNNN proposed → active` is legal | hard-reject |
| R-AUDIT-0002 | transition.go | L14 | Epic | `promote E-NNNN proposed → cancelled` is legal (via `aiwf cancel`) | hard-reject |
| R-AUDIT-0003 | transition.go | L15 | Epic | `promote E-NNNN active → done` is legal | hard-reject |
| R-AUDIT-0004 | transition.go | L15 | Epic | `promote E-NNNN active → cancelled` is legal (via `aiwf cancel`) | hard-reject |
| R-AUDIT-0005 | transition.go | L16-17 | Epic | `done` and `cancelled` are terminal — no outgoing transitions | hard-reject |
| R-AUDIT-0006 | transition.go | L20 | Milestone | `promote M-NNNN draft → in_progress` is legal | hard-reject |
| R-AUDIT-0007 | transition.go | L20 | Milestone | `promote M-NNNN draft → cancelled` is legal (via `aiwf cancel`) | hard-reject |
| R-AUDIT-0008 | transition.go | L21 | Milestone | `promote M-NNNN in_progress → done` is legal | hard-reject |
| R-AUDIT-0009 | transition.go | L21 | Milestone | `promote M-NNNN in_progress → cancelled` is legal (via `aiwf cancel`) | hard-reject |
| R-AUDIT-0010 | transition.go | L22-23 | Milestone | `done` and `cancelled` are terminal — no outgoing transitions | hard-reject |
| R-AUDIT-0011 | transition.go | L26 | ADR | `promote ADR-NNNN proposed → accepted` is legal | hard-reject |
| R-AUDIT-0012 | transition.go | L26 | ADR | `promote ADR-NNNN proposed → rejected` is legal (via `aiwf cancel`) | hard-reject |
| R-AUDIT-0013 | transition.go | L27 | ADR | `promote ADR-NNNN accepted → superseded` is legal | hard-reject |
| R-AUDIT-0014 | transition.go | L28-29 | ADR | `superseded` and `rejected` are terminal — no outgoing transitions | hard-reject |
| R-AUDIT-0015 | transition.go | L32 | Gap | `promote G-NNNN open → addressed` is legal | hard-reject |
| R-AUDIT-0016 | transition.go | L32 | Gap | `promote G-NNNN open → wontfix` is legal (via `aiwf cancel`) | hard-reject |
| R-AUDIT-0017 | transition.go | L33-34 | Gap | `addressed` and `wontfix` are terminal — no outgoing transitions | hard-reject |
| R-AUDIT-0018 | transition.go | L37 | Decision | `promote D-NNNN proposed → accepted` is legal | hard-reject |
| R-AUDIT-0019 | transition.go | L37 | Decision | `promote D-NNNN proposed → rejected` is legal (via `aiwf cancel`) | hard-reject |
| R-AUDIT-0020 | transition.go | L38 | Decision | `promote D-NNNN accepted → superseded` is legal | hard-reject |
| R-AUDIT-0021 | transition.go | L39-40 | Decision | `superseded` and `rejected` are terminal — no outgoing transitions | hard-reject |
| R-AUDIT-0022 | transition.go | L43 | Contract | `promote C-NNNN proposed → accepted` is legal | hard-reject |
| R-AUDIT-0023 | transition.go | L43 | Contract | `promote C-NNNN proposed → rejected` is legal (via `aiwf cancel`) | hard-reject |
| R-AUDIT-0024 | transition.go | L44 | Contract | `promote C-NNNN accepted → deprecated` is legal | hard-reject |
| R-AUDIT-0025 | transition.go | L44 | Contract | `promote C-NNNN accepted → rejected` is legal (via `aiwf cancel`) | hard-reject |
| R-AUDIT-0026 | transition.go | L45 | Contract | `promote C-NNNN deprecated → retired` is legal | hard-reject |
| R-AUDIT-0027 | transition.go | L46-47 | Contract | `retired` and `rejected` are terminal — no outgoing transitions | hard-reject |

#### Entity FSM — global rules

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0028 | transition.go | L9-11 | All kinds | FSM is one-directional; no "demote" verb exists (markdown edit is the back-out path) | unenforced (convention only) |
| R-AUDIT-0029 | transition.go | L64-82 | All kinds | `ValidateTransition` returns an error for unknown kind, unknown from-state, or any (from, to) not in the FSM table | hard-reject |
| R-AUDIT-0030 | transition.go | L93-103 | All kinds | `IsTerminal(kind, status)` derives terminality from the FSM (state with no outgoing edges) — no parallel hardcoded list | hard-reject (relied on by check rules) |

#### `aiwf cancel` target mapping

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0031 | transition.go | L110-120 | Cancel verb | `aiwf cancel` on Epic or Milestone targets terminal status `cancelled` | hard-reject |
| R-AUDIT-0032 | transition.go | L110-120 | Cancel verb | `aiwf cancel` on ADR, Decision, or Contract targets terminal status `rejected` | hard-reject |
| R-AUDIT-0033 | transition.go | L110-120 | Cancel verb | `aiwf cancel` on Gap targets terminal status `wontfix` | hard-reject |

#### AC FSM — one rule per `(from-state, to-state)` legal cell

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0034 | transition.go | L128 | AC | `promote M-NNNN/AC-N open → met` is legal | hard-reject |
| R-AUDIT-0035 | transition.go | L128 | AC | `promote M-NNNN/AC-N open → deferred` is legal | hard-reject |
| R-AUDIT-0036 | transition.go | L128 | AC | `promote M-NNNN/AC-N open → cancelled` is legal | hard-reject |
| R-AUDIT-0037 | transition.go | L129 | AC | `promote M-NNNN/AC-N met → deferred` is legal (scope-change after the fact) | hard-reject |
| R-AUDIT-0038 | transition.go | L129 | AC | `promote M-NNNN/AC-N met → cancelled` is legal (scope-change after the fact) | hard-reject |
| R-AUDIT-0039 | transition.go | L130-131 | AC | `deferred` and `cancelled` are terminal AC states | hard-reject |
| R-AUDIT-0040 | transition.go | L137-146 | AC | Self-transitions, unknown `from`, and unknown `to` all return false from `IsLegalACTransition` | hard-reject |
| R-AUDIT-0041 | transition.go | L122-126 | AC | `--force --reason` is the documented relaxation path; verb-projection finding `acs-transition` consults `IsLegalACTransition` | check-error (without `--force`) |

#### TDD-phase FSM — one rule per `(from-phase, to-phase)` legal cell

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0042 | transition.go | L160 | AC tdd_phase | `(absent) → red` is legal (entering the cycle for an AC under tdd: required) | hard-reject |
| R-AUDIT-0043 | transition.go | L161 | AC tdd_phase | `red → green` is legal | hard-reject |
| R-AUDIT-0044 | transition.go | L162 | AC tdd_phase | `green → refactor` is legal | hard-reject |
| R-AUDIT-0045 | transition.go | L162 | AC tdd_phase | `green → done` is legal (refactor is optional) | hard-reject |
| R-AUDIT-0046 | transition.go | L163 | AC tdd_phase | `refactor → done` is legal | hard-reject |
| R-AUDIT-0047 | transition.go | L164 | AC tdd_phase | `done` is terminal | hard-reject |
| R-AUDIT-0048 | transition.go | L153-158 | AC tdd_phase | Entering at `green` or later from absent is intentionally disallowed (would bypass `red` and undermine the `met requires done` audit) | hard-reject |

#### Milestone-to-done precondition

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0049 | transition.go | L193-203 | Milestone | `promote M-NNNN in_progress → done` requires that no AC has status `open`; `MilestoneCanGoDone` returns the offending AC ids when not met | check-error (via `milestone-done-incomplete-acs` finding) + verb-projection block |

**Total for §1 — FSM tables: 49 rules**

---

### 2. Mechanical policies — `internal/policies/*.go`

34 policy files; ~22 encode kernel-verb-workflow-legality statements (in scope), ~12 are code-style / CI-hygiene rules (out of scope for this audit, listed separately at the end of this section).

Each in-scope policy fires as a test failure that blocks CI. Severity throughout: **policy-block** (CI test FAIL → push refused or PR blocked, depending on hook).

#### In-scope policies

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0050 | policies/aiwf_promote_epic_active_audit.go | auditUnforcedEpicActivate | Epic promote verb | Automation-shaped source (CI workflow, scripts, Makefiles) must not invoke `aiwf promote E-<id> active` without `--force --reason "..."` — sovereign promotion of an epic to `active` is human-only | policy-block |
| R-AUDIT-0051 | policies/apply_callers_lock.go | PolicyApplyCallersAcquireLock | All mutating verbs | Every `run*` dispatcher in `cmd/aiwf/` that calls `verb.Apply` directly must also call `cliutil.AcquireRepoLock` — concurrent verb invocations against the same repo are forbidden | policy-block |
| R-AUDIT-0052 | policies/authorized_by_via_allow.go | PolicyAuthorizedByWriteSitesUseAllow | Authorize verb / agent acts | Every function that writes `TrailerAuthorizedBy` must reference `Allow(` or `gateAndDecorate` — hand-stamping an authorize SHA without running the allow-rule check is forbidden | policy-block |
| R-AUDIT-0053 | policies/closed_set_status_constants.go | statusValuesPattern | All kinds | Closed-set status literals (FSM state names) must not appear outside `internal/entity/` — that's the single source of truth | policy-block |
| R-AUDIT-0054 | policies/empty_diff.go | PolicyEmptyDiffCommitsCarryMarker | All verbs allowing empty commits | Every Go file in `internal/verb/` containing `AllowEmpty: true` must also reference `TrailerScope` or `TrailerAuditOnly` — empty-diff commits without a marker trailer are forbidden (G24 audit-trail rule) | policy-block |
| R-AUDIT-0055 | policies/finding_hints.go | PolicyFindingCodesHaveHints | Check rule emission | Every finding code emitted via `Finding{Code: "..."}` must have a matching entry in `internal/check/hint.go`'s `hintTable` — orphan codes break the user-facing "what now?" rendering | policy-block |
| R-AUDIT-0056 | policies/findings_have_tests.go | PolicyFindingCodesHaveTests | Check rule emission | Every finding code emitted by the kernel must be referenced from at least one `*_test.go` file — an orphan code is, by definition, untested | policy-block |
| R-AUDIT-0057 | policies/fsm_invariants.go | PolicyFSMInvariants | Entity FSM | Each kind's FSM must have at least one terminal state, no orphan from-states, and `CancelTarget` must return one of the kind's terminal states | policy-block |
| R-AUDIT-0058 | policies/integration_tests_assert_trailers.go | PolicyIntegrationTestsAssertTrailers | Integration tests | Integration tests that invoke a mutating verb via `runBin` must assert the resulting commit's trailers — pure exit-code observation is not sufficient | policy-block |
| R-AUDIT-0059 | policies/no_actor_in_aiwfyaml.go | forbiddenAiwfYAMLFieldNames | aiwf.yaml schema | `actor:`, `principal:`, and related identity-fields are forbidden on any `internal/aiwfyaml/` struct — identity is runtime-derived from `git config user.email` (I2.5 commitment) | policy-block |
| R-AUDIT-0060 | policies/no_dangling_entity_refs.go | PolicyNoDanglingEntityRefs | Markdown content | Markdown links / inline entity ids must resolve to an existing entity (active or archive); dangling refs across renames/cancels are forbidden | policy-block |
| R-AUDIT-0061 | policies/no_hardcoded_entity_paths.go | PolicyNoHardcodedEntityPaths | `internal/policies/` source | Policy tests that read entity files must resolve via `tree.Load` → `Tree.ByID(id)` → `entity.Path`; `filepath.Join` literals naming entity slugs are forbidden (would break under archive sweeps) | policy-block |
| R-AUDIT-0062 | policies/no_history_rewrites.go | PolicyNoHistoryRewrites | Production code | History-rewriting git invocations (`push --force-with-lease`, `reset --hard`, `commit --amend`, `rebase`, `filter-branch`) must not appear in non-test production source — the kernel's audit-trail guarantee depends on history being append-only | policy-block |
| R-AUDIT-0063 | policies/no_role_id_regex.go | rolePatternSubstrings | Production code | Ad-hoc `<role>/<id>` regex construction outside `gitops.roleIDPattern` is forbidden — actor/principal validation has a single canonical regex | policy-block |
| R-AUDIT-0064 | policies/no_signature_bypass.go | signatureBypassSubstrings | Production code | `--no-verify`, `--no-gpg-sign`, `commit.gpgsign=false`, and equivalents must not appear in non-test source — hook bypass defeats the pre-push `aiwf check` chokepoint | policy-block |
| R-AUDIT-0065 | policies/no_silent_fallback.go | PolicyNoSilentFallbacks | Closed-set switches | `switch` statements over closed-set types (`Kind`, `Status`, etc.) must have a non-silent default branch (error / sentinel return); silent fallthrough is forbidden | policy-block |
| R-AUDIT-0066 | policies/no_timestamp_manipulation.go | PolicyNoTimestampManipulation | Production code | `GIT_AUTHOR_DATE` and `GIT_COMMITTER_DATE` env vars must not be set in non-test source — backdating commits corrupts the chronological order every standing rule relies on | policy-block |
| R-AUDIT-0067 | policies/no_trailer_string_composition.go | PolicyNoTrailerStringComposition | Trailer write-sites | `fmt`-format strings that look like synthesized trailer lines (`aiwf-<name>: ...`) are forbidden; trailers must be assembled via `gitops.Trailer{Key, Value}` struct literals so `gitops.ValidateTrailer` fires | policy-block |
| R-AUDIT-0068 | policies/principal_write_sites.go | PolicyPrincipalWriteSitesGuardHuman | Provenance trailers | Every function that writes `TrailerPrincipal` or `TrailerOnBehalfOf` must reference `"human/"` somewhere in its body — principal and on-behalf-of are human-only by design | policy-block |
| R-AUDIT-0069 | policies/read_only.go | readOnlyVerbs | Read-only verbs (check, history, status, render, doctor, show, list) | Read-only verb body-functions must not call `gitops.Commit`, `gitops.Mv`, `gitops.Add`, or `os.WriteFile`; reads are pure functions | policy-block |
| R-AUDIT-0070 | policies/sovereign.go | PolicySovereignDispatchersGuardHumanActor | Sovereign verbs (`--force --reason`, `--audit-only`, `--to` on authorize) | Every cmd dispatcher with a sovereign-act flag pair must reference `"human/"` — sovereign acts must trace to a named human | policy-block |
| R-AUDIT-0071 | policies/trailer_keys.go | PolicyTrailerKeysViaConstants | Trailer references | String literals matching `gitops.Trailer*` constant values are forbidden outside `internal/gitops/` — every other package must reference trailers by symbol | policy-block |
| R-AUDIT-0072 | policies/verbs_validate_then_write.go | PolicyVerbsValidateThenWrite | `internal/verb/` exported functions | Verb functions must not directly call `gitops.Commit/Mv/Add/Restore/CommitAllowEmpty`, `os.WriteFile`, `os.Create`, or `os.Remove` — they return a `*Plan`; `verb.Apply` is the only writer | policy-block |

#### Out-of-scope policies (CI hygiene / test discipline / discoverability)

Listed here so the audit shows we *considered* and explicitly excluded them, not silently missed them. Per ADR-0011 §Scope, this catalog covers kernel-verb workflow legality only; CI / source-style invariants are out of scope.

- `claude_md_test_discipline.go` — CLAUDE.md must have `### Test discipline` section under `## Go conventions`. (Doc structure, not workflow.)
- `config_fields_discoverable.go` — every yaml-tagged field on aiwf.yaml structs must appear in skill, help, CLAUDE.md, or docs/pocv3. (Discoverability, not workflow.)
- `design_doc_anchors.go` — markdown link anchors in design docs must resolve. (Doc hygiene.)
- `discoverability.go` — every finding code must appear in skill/help/CLAUDE.md/docs/pocv3. (Discoverability.)
- `filepath_join_segments.go` — `filepath.Join` args after the first must not embed `/` separators. (Source-style.)
- `no_retry_loops_on_git.go` — production code must not retry-loop around git. (Source-style.)
- `race_parallel_cap.go` — race-mode `go test` invocations carry `-parallel 8`. (CI infra.)
- `skill_coverage.go` — every Cobra verb has a same-named skill or allowlist entry. (Discoverability.)
- `test_setup_presence.go` — every internal/* test-bearing package carries `setup_test.go` with `TestMain`. (Test infra.)
- `tests_real_clone.go` — integration tests use real bare repo + clone, not `git update-ref`. (Test infra.)
- `policies.go` — package docstring only, not a policy. (N/A.)

**Total for §2: 23 in-scope rules + 11 out-of-scope policies acknowledged**

---

### 3. Check rules — `internal/check/*.go`

23 distinct finding codes emitted by the kernel's check engine. Each defines a class of illegal state — when the tree contains a state matching the rule's preconditions, the finding fires. Severity is per-code (and per-context, in some cases — see `acs-tdd-audit`).

Citations are by source file plus the emitting function or condition.

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0073 | check/acs.go | runACsTDDAudit | Milestone AC + tdd policy | An AC with `status: met` under `tdd: required` whose `tdd_phase` ≠ `done` fires `acs-tdd-audit` — TDD discipline requires phase: done before met | check-error (tdd: required) / check-warning (tdd: advisory) |
| R-AUDIT-0074 | check/acs.go | runACsShape | Milestone AC | AC frontmatter must have a non-empty `id` matching `AC-N` shape; missing or malformed fires `acs-shape` / subcode `id` | check-error |
| R-AUDIT-0075 | check/acs.go | runACsShape | Milestone AC | AC `id` must be unique within the milestone; duplicates fire `acs-shape` / subcode `id` | check-error |
| R-AUDIT-0076 | check/acs.go | runACsShape | Milestone AC | AC must have a non-empty `title`; missing fires `acs-shape` / subcode `title` | check-error |
| R-AUDIT-0077 | check/acs.go | runACsShape | Milestone AC | AC `status` must be one of the closed-set values (`open`, `met`, `deferred`, `cancelled`); invalid fires `acs-shape` / subcode `status` | check-error |
| R-AUDIT-0078 | check/acs.go | runACsShape | Milestone AC | AC `tdd_phase` (when present) must be one of (`red`, `green`, `refactor`, `done`); invalid fires `acs-shape` / subcode `tdd-phase` | check-error |
| R-AUDIT-0079 | check/acs.go | runACsBodyCoherence | Milestone AC | Every AC in frontmatter `acs[]` must have a corresponding `### AC-N — <title>` body section; missing fires `acs-body-coherence` / subcode `missing-heading` | check-warning |
| R-AUDIT-0080 | check/acs.go | runACsTitleProse | Milestone AC | AC title that is long / multi-sentence / contains markdown fires `acs-title-prose` — title should be a short label, detail prose belongs in the body | check-warning |
| R-AUDIT-0081 | check/acs.go | runMilestoneDoneACs | Milestone | A milestone with `status: done` whose ACs include any non-terminal status fires `milestone-done-incomplete-acs` | check-error |
| R-AUDIT-0082 | check/archive_rules.go | runArchivedEntityTerminal | All kinds | An entity under `archive/` whose status is not terminal fires `archived-entity-not-terminal` — archive is the structural projection of FSM-terminality (ADR-0004 §Reversal) | check-error |
| R-AUDIT-0083 | check/archive_rules.go | runTerminalEntityArchive | All kinds | An entity with terminal status still in the active tree fires `terminal-entity-not-archived` — awaits `aiwf archive --apply` sweep | check-warning |
| R-AUDIT-0084 | check/archive_rules.go | runArchiveSweepPending | All kinds | N terminal entities awaiting `aiwf archive --apply` fires `archive-sweep-pending` (configurable threshold via `aiwf.yaml` flips to blocking past the threshold) | check-warning (escalates to check-error past `archive.sweep_threshold`) |
| R-AUDIT-0085 | check/entity_body.go | runEntityBodyEmpty | All kinds | An entity body section (typically `## Acceptance criteria`, `## Goal`) that is empty fires `entity-body-empty` with the affected section name | check-warning |
| R-AUDIT-0086 | check/entity_id_narrow_width.go | runEntityIDNarrowWidth | All kinds | An entity id narrower than the canonical 4-digit width (per ADR-0008) fires `entity-id-narrow-width` — parser tolerates on input, renderer always emits canonical | check-warning |
| R-AUDIT-0087 | check/epic_active_drafts.go | runEpicActiveNoDrafts | Epic + child milestones | An epic with `status: active` whose child milestones are all still `status: draft` fires `epic-active-no-drafted-milestones` — an active epic should have at least one non-draft child | check-warning |
| R-AUDIT-0088 | check/check.go (frontmatter validators) | runFrontmatterShape | All kinds | Frontmatter that fails to parse, has wrong shape, or has unknown required fields fires `frontmatter-shape` | check-error |
| R-AUDIT-0089 | check/check.go | runGapResolvedHasResolver | Gap | A gap with `status: addressed` must have a `resolved-by:` frontmatter field pointing to an entity; missing fires `gap-resolved-has-resolver` | check-error |
| R-AUDIT-0090 | check/check.go | runIDsUnique | All kinds | Two entity files with the same canonical id (after width normalization) fires `ids-unique` | check-error |
| R-AUDIT-0091 | check/check.go | runStatusValid | All kinds | A frontmatter `status` value not in the kind's FSM state-set fires `status-valid` | check-error |
| R-AUDIT-0092 | check/check.go | runTitlesNonempty | All kinds | A frontmatter `title:` field that is empty or whitespace-only fires `titles-nonempty` | check-error |
| R-AUDIT-0093 | check/check.go | runRefsResolve | All kinds | Cross-references (frontmatter parent/depends-on/discovered-in/relates-to/linked-adr/resolved-by, body links) pointing to non-existent entities fire `refs-resolve` | check-error |
| R-AUDIT-0094 | check/check.go | runNoCycles | All kinds | A cycle in the entity-reference graph (parent / depends-on edges) fires `no-cycles` | check-error |
| R-AUDIT-0095 | check/check.go | runIDPathConsistent | All kinds | Frontmatter `id:` mismatching the on-disk filename id-prefix fires `id-path-consistent` | check-error |
| R-AUDIT-0096 | check/check.go | runUnexpectedTreeFile | All kinds | Files under `work/` that don't match the expected entity shape fire `unexpected-tree-file` (blocks under `tree.strict: true` in aiwf.yaml; warning otherwise) | check-warning (escalates to check-error under `tree.strict: true`) |
| R-AUDIT-0097 | check/check.go | runADRSupersession | ADR | An ADR claiming to supersede another via `supersedes:` while the other doesn't have `superseded-by:` (or vice versa) fires `adr-supersession-mutual` | check-error |
| R-AUDIT-0098 | check/check.go | runCasePaths | All kinds | An entity file path differing only by case from another (case-insensitive filesystem collision) fires `case-paths` | check-error |
| R-AUDIT-0099 | check/check.go | loadError | All kinds | A frontmatter parse error or file-read failure fires `load-error` | check-error |
| R-AUDIT-0100 | check/provenance.go | runProvenance | All commits | Provenance audit findings: untrailered acts, missing principal, expired scope, etc. (subcodes per condition) | check-error / check-warning depending on subcode |

**Total for §3: 28 rules across 23 finding codes (some codes have multiple sub-rules)**

---

### 4. Cobra verb definitions — `cmd/aiwf/`, `internal/cli/<verb>/`

The Cobra command tree under `cmd/aiwf/` enforces per-verb preconditions before invoking `verb.Apply`. This section captures *headline* per-verb legality (the citation-level audit of every flag pre-check would explode the catalog and is best deferred to M-0123 reconciliation where we have the spec schema to drive it). Citations are by file; per-line refinement happens during reconciliation.

#### Universal mutating-verb rules

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0101 | cliutil/AcquireRepoLock | All mutating verbs | All mutations | Every mutating verb acquires the repo lock (`internal/repolock`) before reading/writing; concurrent invocations against the same repo serialize | hard-reject (ErrBusy) |
| R-AUDIT-0102 | verb/apply.go | All mutating verbs | All mutations | Every mutating verb returns exactly one `*verb.Plan`; `verb.Apply` is the only writer; the plan produces exactly one git commit (or zero if the verb errors out before staging) | hard-reject (kernel invariant) |
| R-AUDIT-0103 | gitops trailer schema | All mutating verbs | All commits | Every mutating-verb commit carries `aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:` trailers; principal/agent acts also carry `aiwf-principal:` (or `aiwf-on-behalf-of:` + `aiwf-authorized-by:`) | check-error (provenance audit) |
| R-AUDIT-0104 | cliutil principal-required guard | All mutating verbs | Non-human actors | When `--actor` is non-human (`ai/...`, `bot/...`), `--principal human/<id>` must be supplied OR an active authorization scope must cover the entity; otherwise the verb errors before any write | hard-reject |
| R-AUDIT-0105 | cliutil sovereign guard | `--force --reason`, `--audit-only` | Sovereign acts | Sovereign-act flags require a `human/...` actor; non-human actors are refused | hard-reject |

#### Per-verb headline rules

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0106 | cmd/aiwf/verbs_cmd.go | add verb | `aiwf add <kind>` | Kind must be one of the six (epic, milestone, adr, gap, decision, contract) plus `ac` subcommand; unknown kind errors | hard-reject |
| R-AUDIT-0107 | cmd/aiwf/verbs_cmd.go | add milestone | `aiwf add milestone` | `--epic <id>` is required AND `--tdd <required\|advisory\|none>` is required at creation time (G-0055 layer 1) | hard-reject |
| R-AUDIT-0108 | cmd/aiwf/verbs_cmd.go | add ac | `aiwf add ac` | The target milestone must be non-terminal (not `done` / `cancelled`); adding an AC to a done milestone errors | hard-reject |
| R-AUDIT-0109 | cmd/aiwf/verbs_cmd.go | add ac --tests | `aiwf add ac --tests "..."` | The `--tests` flag (red-phase metrics) is only legal when the parent milestone is `tdd: required` | hard-reject |
| R-AUDIT-0110 | cmd/aiwf/verbs_cmd.go | add | `aiwf add` --title | Title length capped at `entities.title_max_length` from aiwf.yaml (default 80); over-cap is hard-rejected (G-0102) | hard-reject |
| R-AUDIT-0111 | cmd/aiwf/verbs_cmd.go | promote verb | `aiwf promote <id> <new-status>` | The (kind, from-state, to-state) tuple must be in `entity.AllowedTransitions`; otherwise hard-rejected unless `--force --reason "..."` | hard-reject |
| R-AUDIT-0112 | cmd/aiwf/verbs_cmd.go | promote --phase | `aiwf promote <id>/AC-N --phase <p>` | `--phase` is mutex with positional `<new-status>`; phase transition follows `IsLegalTDDPhaseTransition` | hard-reject |
| R-AUDIT-0113 | cmd/aiwf/verbs_cmd.go | promote E-NNNN active | `aiwf promote E-NNNN active` | Sovereign-act: requires `--force --reason "..."` AND a `human/...` actor; agent acts cannot activate epics (M-0095) | hard-reject |
| R-AUDIT-0114 | cmd/aiwf/verbs_cmd.go | promote M-NNNN done | `aiwf promote M-NNNN done` | Requires all ACs at terminal status (per `MilestoneCanGoDone`); otherwise hard-rejected unless `--force --reason "..."` | hard-reject |
| R-AUDIT-0115 | cmd/aiwf/verbs_cmd.go | cancel verb | `aiwf cancel <id>` | Promotes to the kind's terminal-cancel status (`CancelTarget` mapping); same provenance/sovereign rules as promote | hard-reject |
| R-AUDIT-0116 | cmd/aiwf/verbs_cmd.go | edit-body verb | `aiwf edit-body <id>` | Frontmatter is untouched; body replacement only. `--body-file` must not start with `---` (refused as that would imply frontmatter rewrite) | hard-reject |
| R-AUDIT-0117 | cmd/aiwf/verbs_cmd.go | rename verb | `aiwf rename <id> <new-slug>` | Renames on-disk slug; id preserved; updates references in cross-linked entities | hard-reject |
| R-AUDIT-0118 | cmd/aiwf/verbs_cmd.go | move verb | `aiwf move M-id --epic E-id` | Moves a milestone to a different epic; preserves milestone id; updates parent reference; target epic must exist and be non-terminal | hard-reject |
| R-AUDIT-0119 | cmd/aiwf/verbs_cmd.go | reallocate verb | `aiwf reallocate <id>` | Renumbers entity to a new id (resolving collisions); rewrites every cross-reference in the tree in the same commit | hard-reject |
| R-AUDIT-0120 | cmd/aiwf/retitle_cmd.go | retitle verb | `aiwf retitle <id> "<title>"` | Updates frontmatter title and re-derives on-disk slug from it in one commit (G-0108); new title subject to the title_max_length cap | hard-reject |
| R-AUDIT-0121 | cmd/aiwf/rewidth_cmd.go | rewidth verb | `aiwf rewidth --apply` | Migrates narrow-legacy entity ids to canonical 4-digit width (ADR-0008); idempotent; one commit | hard-reject |
| R-AUDIT-0122 | cmd/aiwf/authorize_cmd.go | authorize verb | `aiwf authorize <id> --to <agent>` | Opens an autonomous-work scope; requires `human/...` actor; refused on terminal scope-entity unless `--force --reason "..."` | hard-reject |
| R-AUDIT-0123 | cmd/aiwf/authorize_cmd.go | authorize verb | `aiwf authorize <id> --pause "..."` / `--resume "..."` | Pause/resume the most-recently-opened active scope on `<id>`; cycles the scope FSM (active → paused → active → ended) | hard-reject |
| R-AUDIT-0124 | cmd/aiwf/archive_cmd.go | archive verb | `aiwf archive --apply` | Sweeps qualifying (terminal-status) entities into per-kind `archive/` subdirs in one commit (ADR-0004); decoupled from FSM promotion | hard-reject |
| R-AUDIT-0125 | cmd/aiwf/init_cmd.go | init verb | `aiwf init` | One-time setup; refuses to run on a repo that already has `aiwf.yaml` (no overwrite-existing behavior) | hard-reject |
| R-AUDIT-0126 | cmd/aiwf/update_cmd.go | update verb | `aiwf update` | Re-materializes embedded skills into `.claude/skills/aiwf-*/`; refuses to clobber user-modified consumer hooks (`# aiwf:<hook>` marker controls regeneration) | hard-reject |
| R-AUDIT-0127 | cmd/aiwf/upgrade_cmd.go | upgrade verb | `aiwf upgrade [--version vX.Y.Z]` | Fetches a newer binary via `go install`, re-execs into `aiwf update`; pre-release pseudo-versions accepted (G43) | hard-reject |
| R-AUDIT-0128 | cmd/aiwf/import_cmd.go | import verb | `aiwf import <manifest>` | Bulk-creates entities from a YAML/JSON manifest; one commit by default. On-collision behavior controlled by `--on-collision <fail\|skip\|update>` (default fail). `--dry-run` validates without writing | hard-reject |
| R-AUDIT-0129 | cmd/aiwf/contract_cmd.go | contract bind | `aiwf contract bind <C-id>` | Adds/replaces a binding in aiwf.yaml; `--validator`, `--schema`, `--fixtures` required (or `--force` to replace existing) | hard-reject |
| R-AUDIT-0130 | cmd/aiwf/contract_cmd.go | contract recipe install | `aiwf contract recipe install <name>` | Installs a validator from the embedded set or `--from <path>`; `--force` required to overwrite an existing declared validator | hard-reject |
| R-AUDIT-0131 | cmd/aiwf/contract_cmd.go | contract recipe remove | `aiwf contract recipe remove <name>` | Removes a declared validator; errors when any contract binding still references it | hard-reject |
| R-AUDIT-0132 | cmd/aiwf/render_cmd.go | render --write | `aiwf render roadmap --write` / `aiwf render --format=html` | When `--write` (roadmap) or HTML output is requested, files are written but only the roadmap commit is created; HTML output is regenerated artifact (gitignored) | hard-reject |

#### Read-only verbs (one rule, summarized)

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0133 | policies/read_only.go + the verbs themselves | check, history, status, render (no --write), doctor, show, list, schema, template, whoami, version, completion, contract verify, contract recipes, contract recipe show | Read-only verbs | These verbs do not take the repo lock; do not commit; do not modify disk. May run concurrently with mutations (worst case is reading a pre-mutation snapshot) | policy-block (per `read_only.go` policy) |

**Total for §4: 33 rules**

---

### 5. ADRs — `docs/adr/`

Eight ratified or proposed ADRs (excluding ADR-0011 — the methodology ADR for this audit itself). Each ADR pins one or more workflow-legality rules. Many of these have already been mechanized in §1 / §2 / §3; this section records the *source-of-decision* citation and any rules not captured upstream.

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0134 | ADR-0001 | §Decision | All kinds at trunk integration | Entity ids are minted at trunk-integration time via a per-kind inbox-state mechanism; preserves stable ids across branch / rename / cancel. **Note:** ADR-0001 is still `proposed` — the inbox-state mechanism is design-only; current behavior is trunk-relative ULID-shaped allocation | unenforced (ADR proposed) |
| R-AUDIT-0135 | ADR-0003 | §Decision | All kinds | Findings are a seventh entity kind (F-NNN). **Note:** ADR-0003 is `accepted` but no implementation yet — findings currently live as ephemeral `aiwf check` outputs, not as on-disk entities | unenforced (deferred implementation) |
| R-AUDIT-0136 | ADR-0004 | §Decision | All kinds | Terminal-status entities live under per-kind `archive/` subdirectories (`work/gaps/archive/`, etc.); the active directory listing reflects what is currently in-flight | check-warning (`terminal-entity-not-archived`) / check-error (under `archive.sweep_threshold`) |
| R-AUDIT-0137 | ADR-0004 | §Decision | All kinds | Movement is decoupled from FSM promotion: `aiwf promote` and `aiwf cancel` flip status only; `aiwf archive` sweeps qualifying entities into per-kind `archive/` subdirs as a single commit per invocation | hard-reject (verb structure) |
| R-AUDIT-0138 | ADR-0004 | §Decision | All kinds | The loader resolves ids across active and archive transparently; cross-references stay live indefinitely | hard-reject (loader contract) |
| R-AUDIT-0139 | ADR-0004 | §Reversal | All kinds | Reversal of archive is deliberately absent — file a new entity referencing the archived one. Archive is one-way once swept | unenforced (no verb; convention only) |
| R-AUDIT-0140 | ADR-0006 | §Decision | Top-level Cobra verbs | Every top-level verb is reachable through an AI-discoverable channel: per-verb skill (default), topical multi-verb skill, no skill (`--help` covers it), or discoverability-priority split | policy-block (`policies/skill_coverage.go`) |
| R-AUDIT-0141 | ADR-0007 | §Placement | Skill placement | Planning-conversation skills (start-milestone, start-epic, wrap-milestone, plan-epic, plan-milestones, wrap-epic) live in the rituals plugin, not the kernel binary | policy-block (`policies/skill_coverage.go`) |
| R-AUDIT-0142 | ADR-0007 | §Placement | Skill placement | Kernel-embedded skills cover the verb-wrapper shape (e.g., aiwf-add, aiwf-status, aiwf-history) | policy-block (`policies/skill_coverage.go`) |
| R-AUDIT-0143 | ADR-0007 | §Tiering | Skill graduation | A skill begins as a pure markdown skill; promotion to a kernel verb requires a trigger condition (E-0021 success criterion #7) | unenforced (governance rule, no chokepoint) |
| R-AUDIT-0144 | ADR-0008 | §Decision | All kinds | Every kernel id (E, M, G, D, C, ADR, AC, F) emits at canonical 4-digit width on output. Parsers tolerate narrower legacy widths (≥1 digit) on input so pre-migration trees still validate | check-warning (`entity-id-narrow-width`); hard-reject for `aiwf rewidth --apply` to migrate |
| R-AUDIT-0145 | ADR-0008 | §Decision | All kinds | Renderers and allocators always emit canonical width; consumers carrying narrow-legacy trees migrate via `aiwf rewidth --apply` (one commit, idempotent) | hard-reject (renderer contract) |
| R-AUDIT-0146 | ADR-0009 | §Decision | Orchestration | Driver and substrate are separated; events are trailer-only (no separate event log). **Note:** ADR-0009 is `proposed` — currently aspirational | unenforced (ADR proposed) |
| R-AUDIT-0147 | ADR-0010 | §Tier 1 | Branch context | Initial entity creation (`aiwf add epic/milestone/gap/decision/contract/adr/ac`), state-announcement transitions (`promote E-NN proposed → active`, `authorize`), and author iteration land on `main` by default | unenforced (convention; not git-enforced) |
| R-AUDIT-0148 | ADR-0010 | §Tier 2 | Branch context | Ritual surfaces create named branches: `epic/E-NN-<slug>`, `milestone/M-NNN-<slug>`, `fix/...`, `patch/...`, `doc/...`, `chore/...`. Once on a ritual branch, all mutations related to that work go there too | unenforced (skill-driven; outside kernel scope per E-0033, into E-0030 scope) |
| R-AUDIT-0149 | ADR-0010 | §Sequencing | Epic open | State-announcement commits (epic proposed → active, authorize scope) must precede the branch cut, not follow it | unenforced (skill-driven) |
| R-AUDIT-0150 | docs/adr/ADR-0011 | (self-reference) | Workflow legality | **Out of audit scope** — ADR-0011 ratifies the methodology under which this catalog is being built. Auditing it here would be circular | N/A |

**Total for §5: 16 rules + 1 self-reference acknowledged (ADR-0011)**

---

## Halfway-point checkpoint

**Status:** Sources 1–5 complete. 133 rules extracted. Sources 6–9 (kernel commitments doc, CLAUDE.md, skills, `--help` text) deferred to post-checkpoint per M-0121's mid-flight review commitment.

**Rule-id range used:** R-AUDIT-0001 through R-AUDIT-0150 (with one out-of-scope note at 0150).

**Coverage summary by source so far:**

| Source | In-scope rules | Out-of-scope / acknowledged |
|---|---|---|
| 1. FSM tables (`transition.go`) | 49 | — |
| 2. Mechanical policies (`policies/*.go`) | 23 | 11 (CI hygiene / test discipline) |
| 3. Check rules (`check/*.go`) | 28 (across 23 finding codes) | — |
| 4. Cobra verb definitions (`cmd/aiwf/`, `cli/`) | 33 | — |
| 5. ADRs (`docs/adr/`) | 16 | 1 (ADR-0011 self-reference) |
| **Subtotal** | **149** | **12** |

(Earlier subtotals had 49+23+28+33+16 = 149. The mismatch with R-AUDIT-0001..0149 sequential ids is because I numbered through 0150 with one acknowledgment row that's not a rule per se.)

**Open questions for review:**

1. **Severity vocabulary** — current set: `hard-reject` (verb errors out), `check-error` (finding blocks pre-push), `check-warning` (advisory), `policy-block` (CI test FAIL), `unenforced` (prose-only or proposed). Is this the right closed set for the spec? Should we differentiate further (e.g. `pre-commit-block` vs `pre-push-block`)?

2. **Per-verb depth** — §4 captures one headline rule per verb with citations at file granularity, not per-flag-precondition. M-0123's reconciliation can drill down. Is that depth right for Pass A, or do you want deeper per-verb extraction now?

3. **ADR-0011 self-reference** — captured as out-of-scope row. Is that the right framing?

4. **Out-of-scope policy list** — §2's 11 acknowledged-out-of-scope items. Should any of them flip in? (`integration_tests_assert_trailers` and `no_actor_in_aiwfyaml` are the borderline cases I marked in-scope.)

5. **The "proposed" ADRs** (ADR-0001 inbox-state, ADR-0009 substrate-vs-driver) — should their rules be captured at all if they're not implemented? Current handling: yes, with severity `unenforced (ADR proposed)`.

---

### 6. Kernel commitments — `docs/pocv3/design/design-decisions.md`

This doc is the canonical *decision source* for most of what §1–§4 mechanizes. Many rules in those earlier sections trace back to a paragraph here. For Pass A, I capture (a) cross-cutting principles not yet rule-shaped elsewhere and (b) frontmatter / body / provenance commitments that are the originating decisions.

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0151 | design-decisions.md | §Cross-cutting | All verbs | Enforcement does not depend on the LLM choosing to enforce — skills are advisory; the pre-push git hook and `aiwf check` are authoritative | hard-reject (architectural commitment) |
| R-AUDIT-0152 | design-decisions.md | §Cross-cutting | All entities | Referential stability: an id once allocated always means the same entity, even after rename, move, or status-change to terminal; the id is the primary key, the slug is just display | hard-reject (loader contract) |
| R-AUDIT-0153 | design-decisions.md | §Cross-cutting | All verbs | Engine is invocable without an AI — every verb takes flags, reads stable input formats, emits a JSON envelope, exits with documented codes | hard-reject (CLI contract) |
| R-AUDIT-0154 | design-decisions.md | §Six entity kinds | All kinds | The six kinds (epic, milestone, ADR, gap, decision, contract) and their closed status sets are hardcoded in Go for the PoC — not driven by external YAML | hard-reject (kernel commitment) |
| R-AUDIT-0155 | design-decisions.md | §Frontmatter schema | All kinds | Every entity has the common required frontmatter fields `id`, `title`, `status` (yaml-validated) | check-error (`frontmatter-shape`) |
| R-AUDIT-0156 | design-decisions.md | §Frontmatter schema | Milestone | Milestone frontmatter requires `parent` (an epic id); checked by `refs-resolve` | check-error |
| R-AUDIT-0157 | design-decisions.md | §Frontmatter schema | Milestone | Milestone optional `depends_on` field is a list of milestone ids; feeds `no-cycles` | check-error |
| R-AUDIT-0158 | design-decisions.md | §Frontmatter schema | ADR | ADR optional `supersedes` (list) and `superseded_by` (single) must be mutually consistent; checked by `adr-supersession-mutual` | check-error |
| R-AUDIT-0159 | design-decisions.md | §Frontmatter schema | Gap | Gap optional `discovered_in` points to a milestone or epic; optional `addressed_by` points to any kind | check-error |
| R-AUDIT-0160 | design-decisions.md | §Frontmatter schema | Decision | Decision optional `relates_to` is a list of any-kind ids | check-error |
| R-AUDIT-0161 | design-decisions.md | §Frontmatter schema | Contract | Contract optional `linked_adrs` is a list of ADRs | check-error |
| R-AUDIT-0162 | design-decisions.md | §Frontmatter schema | All kinds | Timestamps (`created`, `updated`) are deliberately absent from frontmatter — `git log` carries them; putting them in YAML would be redundant state | unenforced (convention; renderers must not synthesize them) |
| R-AUDIT-0163 | design-decisions.md | §Body templates | All kinds | Body sections are starting points, not enforced structure — the kernel guarantees structural and referential stability of frontmatter; prose is the human's responsibility | unenforced (advisory: `entity-body-empty` fires on empty sections but body shape is not validated) |
| R-AUDIT-0164 | design-decisions.md | §Stable ids | All kinds | Ids are sequential within a kind, allocated by scanning the working tree and trunk ref (`allocate.trunk` config, default `refs/remotes/origin/main`) and picking `max + 1` | hard-reject (allocator contract) |
| R-AUDIT-0165 | design-decisions.md | §Stable ids | All kinds | Removals are not deletions: `aiwf cancel <id>` flips status to the kind's terminal value (`cancelled`/`wontfix`/`rejected`/`retired`). The file stays. References stay valid | hard-reject (verb contract) |
| R-AUDIT-0166 | design-decisions.md | §Stable ids | All kinds | The id format is never extended with suffixes (no `M-0007a`/`M-0007b`); collision recovery always renumbers via `aiwf reallocate` | hard-reject (allocator contract) |
| R-AUDIT-0167 | design-decisions.md | §Stable ids | Reallocate verb | `aiwf reallocate` writes `prior_ids: []` in the new entity's frontmatter and an `aiwf-prior-entity: <old-id>` trailer alongside `aiwf-entity:` so both ids' histories remain queryable via `aiwf history` | hard-reject (verb contract) |
| R-AUDIT-0168 | design-decisions.md | §Validation is the chokepoint | All commits | `--no-verify` bypasses the pre-push hook (standard git behavior). The framework does not try to prevent that, but the *default* is that broken state cannot be pushed silently | unenforced (deliberate escape hatch) |
| R-AUDIT-0169 | design-decisions.md | §Contracts | Contract verb | The engine owns orchestration; the user owns validators. aiwf never ships a `cue` or `ajv` binary — validators are declared in `aiwf.yaml.contracts.validators` (name → command + argv template) and the user installs the binary via their toolchain | hard-reject (verb contract) |
| R-AUDIT-0170 | design-decisions.md | §Contracts | Contract verb | Validator availability is a per-machine concern — missing validator produces `validator-unavailable` *warning* by default; `aiwf.yaml.contracts.strict_validators: true` flips it to error | check-warning (escalates to error under config) |
| R-AUDIT-0171 | design-decisions.md | §ACs | Milestone | ACs are first-class but namespaced inside their milestone, addressable as `M-NNN/AC-N`; they are *not* a seventh entity kind | hard-reject (composite-id grammar) |
| R-AUDIT-0172 | design-decisions.md | §ACs | Milestone | `tdd: required \| advisory \| none` — default `none` when absent; opt-in policy on the milestone | hard-reject (closed-set field) |
| R-AUDIT-0173 | design-decisions.md | §ACs | Milestone | When milestone is `tdd: required`, `aiwf add ac` seeds `tdd_phase: red` in the same commit; otherwise `tdd_phase` is absent | hard-reject (verb contract) |
| R-AUDIT-0174 | design-decisions.md | §ACs | Milestone (anti-rule) | NOT a kernel rule: "milestone must have ≥1 AC" — ACs remain optional | (acknowledgment; intentional absence) |
| R-AUDIT-0175 | design-decisions.md | §ACs | Milestone (anti-rule) | NOT a kernel rule: "milestone can't enter `in_progress` without all ACs in `red`" — the kernel guards the *outcome* (`met` requires `done`), not the entry | (acknowledgment) |
| R-AUDIT-0176 | design-decisions.md | §ACs | Milestone (anti-rule) | NOT a kernel rule: global AC allocator — ACs are per-milestone, no global allocator | (acknowledgment) |
| R-AUDIT-0177 | design-decisions.md | §ACs | Milestone (anti-rule) | NOT a kernel rule: AC tombstone beyond status-cancel — cancelled ACs stay in `acs[]` at their original position with status flipped | (acknowledgment) |
| R-AUDIT-0178 | design-decisions.md | §ACs | All AC promote events | All `promote` events on AC carry `aiwf-to: <state>` trailer so the target state is structured | hard-reject (trailer schema) |
| R-AUDIT-0179 | design-decisions.md | §ACs | Composite-id references | Open-target reference fields (`gap.addressed_by`, `decision.relates_to`) accept `M-NNN/AC-N`. Closed-target fields (`milestone.parent → epic`, `adr.supersedes → adr`, etc.) are unchanged | check-error (`refs-resolve`) |
| R-AUDIT-0180 | design-decisions.md | §ACs | `aiwf rename` composite ids | `aiwf rename M-NNN/AC-N "<new-title>"` updates `acs[].title` AND the `### AC-N — <title>` body heading in one commit; dispatch on composite-vs-bare | hard-reject (verb contract) |
| R-AUDIT-0181 | design-decisions.md | §Provenance | All kinds | Operator identity is runtime-derived from `git config user.email`, with `--actor` override; `aiwf.yaml.actor` is removed (I2.5) | hard-reject |
| R-AUDIT-0182 | design-decisions.md | §Provenance | Three-layer trailers | `aiwf-actor:` (operator) + `aiwf-principal:` (accountability, required when actor is non-human) + `aiwf-on-behalf-of:` and `aiwf-authorized-by:` (required-together scope membership) | hard-reject |
| R-AUDIT-0183 | design-decisions.md | §Provenance | Scope FSM | Scope FSM is `active \| paused \| ended`; opened by `aiwf authorize`; pause/resume are first-class transitions; multiple parallel scopes supported | hard-reject |
| R-AUDIT-0184 | design-decisions.md | §Provenance | Scope FSM | Scope end is automatic when the scope-entity reaches a terminal status (`aiwf-scope-ends:` trailer on the terminal-promote commit). Un-canceling a scope-entity does not resurrect ended scopes (strict end-on-terminal) | hard-reject |
| R-AUDIT-0185 | design-decisions.md | §Provenance | Sovereign acts | `--force` is human-only; agent acts cannot use `--force`. Forced acts carry `aiwf-actor: human/...` + `aiwf-force:` trailers ONLY — no principal, no on-behalf-of | hard-reject |
| R-AUDIT-0186 | design-decisions.md | §Provenance | Gating | A verb is allowed iff the entity-FSM transition is legal AND, for non-human actors, at least one active scope's reachability check passes (gating, not containment) | hard-reject |
| R-AUDIT-0187 | design-decisions.md | §Provenance | Human actors | For human actors with no `--principal`, scope checks are skipped — humans need no authorization to act | hard-reject (intentional exemption) |
| R-AUDIT-0188 | design-decisions.md | §One commit per verb | All mutating verbs | Verbs are *validate-then-write*: compute projected new tree in memory, run `aiwf check` against the projection, write only when clean; partial failure leaves the working tree exactly as it was | hard-reject (Apply contract) |
| R-AUDIT-0189 | design-decisions.md | §One commit per verb | All mutating verbs | Verbs only block on findings *introduced* by the projection — pre-existing tree errors do not refuse an unrelated `aiwf add` | hard-reject (verb policy) |

**Total for §6: 39 rules (with 4 "anti-rule" acknowledgments to capture intentional absences)**

---

### 7. Repo principles — `CLAUDE.md`

CLAUDE.md is the project-instructions doc that codifies engineering principles, Go conventions, and operating procedures for human and AI contributors. Most of its content is either (a) a re-statement of rules already captured in `design-decisions.md` (§6) or (b) Go-conventions / test-discipline rules that are out of scope per ADR-0011 §Scope. This section captures CLAUDE.md-specific workflow rules and explicit re-statements that add chokepoint precision.

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0190 | CLAUDE.md | §Engineering principles | All work | KISS / YAGNI / no half-finished implementations — features land tested or not at all; no stubs / TODOs in shipped code | unenforced (review-driven) |
| R-AUDIT-0191 | CLAUDE.md | §Engineering principles | All verbs | Errors are findings, not parse failures — `aiwf check` loads inconsistent state and reports it; it does not refuse to start | hard-reject (loader contract) |
| R-AUDIT-0192 | CLAUDE.md | §Engineering principles | All kernel surfaces | CLI surfaces must be auto-completion-friendly — every verb, sub-verb, flag, and closed-set value is reachable via tab-completion. Drift-prevention test in `internal/policies/` fails CI on un-wired surfaces | policy-block (`policies/skill_coverage.go` + completion-drift test in `cmd/aiwf/completion_drift_test.go`) |
| R-AUDIT-0193 | CLAUDE.md | §Authoring an ADR | ADR | An ADR captures the choice; planning sequences the action. ADR bodies must not contain gate language (*"ratify after X happens"*, *"status remains proposed through Y wraps"*) — those are planning concerns. FSM (`proposed → accepted | rejected`; `accepted → superseded`) plus `aiwf promote` are the only mechanical surfaces that constrain ADR status transitions | unenforced (convention; FSM still applies) |
| R-AUDIT-0194 | CLAUDE.md | §Authoring an ADR | ADR | No bespoke per-ADR test pins on status transitions. Sovereign override (`--force --reason`) remains available when an exceptional ratification path is needed | unenforced (convention) |
| R-AUDIT-0195 | CLAUDE.md | §AC promotion requires mechanical evidence | All AC promotions | Before `aiwf promote M-NNN/AC-<N> met`, there must be a mechanical assertion that fails if the AC's claim breaks — a Go test under `internal/policies/`, a kernel finding-rule, or a fixture-validation script. *"I read the file and it looks right"* is not evidence; it makes the AC's correctness depend on reviewer recall | unenforced (convention; chokepoint is the AC-promote command but mechanical evidence is human-checked) |
| R-AUDIT-0196 | CLAUDE.md | §AC promotion requires mechanical evidence | ACs under `tdd: none` / `tdd: advisory` | The test-discipline obligation applies even under `tdd: none` — the `tdd:` policy controls whether the kernel's `acs-tdd-audit` finding fires; it does not waive the obligation to have a mechanical test | unenforced (convention) |
| R-AUDIT-0197 | CLAUDE.md | §Working in this repo | Maintainer workflow | Trunk-based development on `main` for maintainers — commit directly to trunk; no PR ceremony. Outside contributors propose via GitHub PRs (CONTRIBUTING.md). Validation is mechanized via pre-commit (shape-only), pre-push (full), and CI | unenforced (workflow convention; layered with branch-model ADR-0010) |
| R-AUDIT-0198 | CLAUDE.md | §Working in this repo | Commit subjects | Conventional Commits subjects are mandatory for both direct-to-main and PR paths (`feat(aiwf): ...`, `chore(aiwf): ...`, `docs: ...`) | unenforced (convention; not git-enforced) |
| R-AUDIT-0199 | CLAUDE.md | §Subagent worktree isolation | Agent dispatch | When dispatching a subagent that must work in an isolated git worktree, the parent session bootstraps the worktree via `git worktree add` BEFORE invoking `Agent`. The `isolation: "worktree"` kwarg has been observed to silently drop (G-0099) | unenforced (operational guidance) |
| R-AUDIT-0200 | CLAUDE.md | §Subagent worktree isolation | Agent dispatch (chokepoint) | `.claude/hooks/validate-agent-isolation.sh` (PreToolUse hook on `Agent` tool, registered in `.claude/settings.json`) denies any `Agent` invocation that passes `isolation: "worktree"` with a message pointing at the precondition pattern | hard-reject (hook denies the tool call) |
| R-AUDIT-0201 | CLAUDE.md | §Cross-repo plugin testing | Skills authored as fixtures | When a milestone's deliverable is a `SKILL.md` in the rituals plugin repo, the canonical authoring location during the milestone is a fixture in this repo at `internal/policies/testdata/<skill-name>/SKILL.md`. AC tests assert content claims against the fixture; deployment to the rituals repo happens at wrap | unenforced (workflow guidance) |
| R-AUDIT-0202 | CLAUDE.md | §Cross-repo plugin testing | Drift detection | A drift-check test in this repo compares the fixture against the local marketplace cache (`~/.claude/plugins/cache/ai-workflow-rituals/.../SKILL.md`) and fires if they diverge; skips cleanly when cache is absent (CI without plugin install) | policy-block when cache present; silent when absent |

**Total for §7: 13 rules**

---

### 8. Skills — `.claude/skills/`, rituals plugin

Per ADR-0011 §Scope, rituals-plugin orchestration is out-of-scope for this audit — *"only the kernel can be mechanically verified."* However, skills do encode *advisory* workflow rules that describe the *intended* verb-sequencing patterns. The kernel does not enforce these; reviewers and AI assistants do. For Pass A, this section captures the high-level structure of skill-driven workflows so Pass C knows what skill-described rules to *not* expect in the spec table.

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0203 | `.claude/skills/` + rituals plugin | (meta) | Skill-described workflows | Skills are advisory — the framework's correctness must not depend on the LLM choosing to follow them. Any workflow rule that lives only in a skill (and not in a kernel verb / check / policy) is unenforced from a kernel perspective | unenforced (architectural commitment per ADR-0011) |
| R-AUDIT-0204 | rituals/aiwfx-start-milestone | SKILL.md | Milestone start | `aiwfx-start-milestone` skill prescribes the sequence: preflight (build green, tests green) → `aiwf promote M-NNN in_progress` → branch setup → per-AC TDD cycle → self-review → hand off to wrap. Each step is human-driven; the kernel verifies each individual mutation | unenforced (advisory) |
| R-AUDIT-0205 | rituals/aiwfx-wrap-milestone | SKILL.md | Milestone wrap | `aiwfx-wrap-milestone` skill prescribes: verify completion (`aiwf show`, `aiwf check`, full test) → final code review → doc-lint sweep → finalize wrap-side sections in spec body → `aiwf promote M-NNN done` → render roadmap → stage + commit → push (each with explicit human approval gate) | unenforced (advisory) |
| R-AUDIT-0206 | rituals/wf-tdd-cycle | SKILL.md | TDD cycle | `wf-tdd-cycle` skill prescribes the red → green → refactor → done phase progression with `aiwf promote <AC> --phase <p>` at each transition. Kernel's `acs-tdd-audit` enforces `met` requires `done` outcome; the skill drives the flow that produces that outcome | unenforced (skill drives flow; kernel enforces outcome) |
| R-AUDIT-0207 | rituals/aiwfx-record-decision | SKILL.md | Decision capture | `aiwfx-record-decision` skill prescribes opening a new `D-NNN` entity when a mid-flight decision surfaces during milestone work, and updating the milestone's `## Decisions made during implementation` section to reference it | unenforced (advisory) |
| R-AUDIT-0208 | rituals/wf-patch | SKILL.md | Ad-hoc patch | `wf-patch` skill prescribes a single-focused-change branch (`fix/...`, `patch/...`, `doc/...`, `chore/...`) that merges into main when the patch lands. Aligns with ADR-0010 §Tier 2 | unenforced (skill drives; branch model in ADR-0010) |
| R-AUDIT-0209 | rituals/aiwfx-wrap-epic | SKILL.md | Epic wrap | `aiwfx-wrap-epic` skill prescribes the closure sequence for an epic: verify all child milestones done → `aiwf promote E-NN done` → render roadmap → merge epic branch to main with `--no-ff` to preserve milestone shape | unenforced (advisory) |
| R-AUDIT-0210 | rituals/aiwfx-plan-epic, aiwfx-plan-milestones | SKILL.md | Planning conversations | The two planning skills prescribe interactive planning conversations: scope an epic before allocating, scope milestones with concrete ACs, lock decisions explicitly. Output is one or more `aiwf add ...` invocations | unenforced (advisory) |
| R-AUDIT-0211 | rituals/aiwfx-authorize | SKILL.md | Authorization | `aiwfx-authorize` skill prescribes opening an autonomous-work scope before delegating an entity to an AI agent: `aiwf authorize <id> --to <agent>`. The scope FSM (active/paused/ended) and its gating effect are kernel-enforced (per R-AUDIT-0183, R-AUDIT-0186) | hard-reject for the kernel gate; unenforced for the skill's workflow prescription |

**Total for §8: 9 rules (all advisory; all kernel-enforcement comes from rules elsewhere)**

---

### 9. Verb help text — `aiwf <verb> --help`

The `--help` text for each verb declares its flag set, defaults, and mutex/required-together pairings. Most workflow-legality claims here echo §4's verb rules; this section captures the flag-level constraints that are stated in `--help` and aren't already covered upstream.

| Rule id | Source | Citation | Scope | Statement | Severity if violated |
|---|---|---|---|---|---|
| R-AUDIT-0212 | `aiwf --help` | "Common flags" | All verbs | `--root <path>` defaults to walking up from cwd looking for `aiwf.yaml`; else cwd | hard-reject if no `aiwf.yaml` found and `--root` is not supplied |
| R-AUDIT-0213 | `aiwf --help` | "Common flags" | All mutating verbs | `--actor <role>/<identifier>` defaults to derived from `git config user.email`; override per invocation | hard-reject if format is malformed (`gitops.ValidateTrailer`) |
| R-AUDIT-0214 | `aiwf --help` | "Common flags" | All mutating verbs | `--principal human/<id>` is required when `--actor` is non-human (ai/..., bot/...); forbidden when `--actor` is human/... | hard-reject |
| R-AUDIT-0215 | `aiwf promote --help` | "Flags for promote and cancel" | Promote / cancel | `--audit-only --reason "..."` backfills audit trail when state was reached via manual commit; entity must already be at the target state (no FSM transition); mutex with `--force`; human-only | hard-reject |
| R-AUDIT-0216 | `aiwf promote --help` | top-level | AC promote | `--phase <p>` (for AC tdd_phase) is mutex with positional `<new-status>` | hard-reject |
| R-AUDIT-0217 | `aiwf promote --help` | top-level | AC promote | `--tests "pass=N fail=N skip=N [total=N]"` attaches an `aiwf-tests` trailer in phase mode; recognized keys only; non-negative integers | hard-reject (write-strict trailer schema) |
| R-AUDIT-0218 | `aiwf authorize --help` | top-level | Authorize verb | `--to <agent>` opens scope; refused on terminal scope-entity unless `--force --reason` | hard-reject |
| R-AUDIT-0219 | `aiwf authorize --help` | top-level | Authorize verb | `--pause "<reason>"` pauses the most-recently-opened active scope on `<id>`; `--resume "<reason>"` resumes the most-recently-paused scope | hard-reject |
| R-AUDIT-0220 | `aiwf import --help` | top-level | Import verb | `--on-collision <fail\|skip\|update>` controls behavior when an explicit id already exists; default `fail` | hard-reject (closed-set flag value) |
| R-AUDIT-0221 | `aiwf import --help` | top-level | Import verb | `--dry-run` validates the projection and prints the would-be plans without writing | hard-reject if combined with conflicting write-mode flags |
| R-AUDIT-0222 | `aiwf upgrade --help` | top-level | Upgrade verb | `--check` prints the current/target comparison and exits; does not invoke `go install` | hard-reject if combined with other write-modifying flags |
| R-AUDIT-0223 | `aiwf doctor --help` | top-level | Doctor verb | `--check-latest` hits the Go module proxy for the latest published `aiwf` version; advisory; honors `GOPROXY=off`; network errors print "unavailable" without failing doctor | unenforced (best-effort) |
| R-AUDIT-0224 | `aiwf check --help` | top-level | Check verb | `--format <fmt>` accepts `text` (default) or `json`; `--pretty` indents JSON only when used with `--format=json` | hard-reject (closed-set flag value) |
| R-AUDIT-0225 | `aiwf history --help` | top-level | History verb | `--show-authorization` includes the full `aiwf-authorized-by` SHA on scope-authorized rows (text format only) | unenforced (display-only) |
| R-AUDIT-0226 | `aiwf add ac --help` | top-level | Add AC verb | `--tests "pass=N fail=N ..."` is only legal when the parent milestone is `tdd: required` (already captured at R-AUDIT-0109; restated here for citation completeness) | hard-reject |

**Total for §9: 15 rules**

---

## Grand total

**Pre-dedup totals:**

| Source | In-scope rules | Acknowledged out-of-scope |
|---|---:|---:|
| 1. FSM tables | 49 | — |
| 2. Mechanical policies | 23 | 11 |
| 3. Check rules (23 codes) | 28 | — |
| 4. Cobra verb definitions | 33 | — |
| 5. ADRs | 16 | 1 |
| 6. Kernel commitments doc | 39 | — |
| 7. CLAUDE.md repo principles | 13 | — |
| 8. Skills (advisory only) | 9 | — |
| 9. Verb help text | 15 | — |
| **Grand total** | **225 in-scope** | **12** |

**Rule-id range used:** R-AUDIT-0001 through R-AUDIT-0226 (with one out-of-scope acknowledgment row at 0150).

**Status:** Extraction complete across all nine sources. Dedup pass below.

---

## 10. Consolidated rules (post-dedup)

> **Revision 2 (2026-05-18):** Incorporated external review findings. Material changes from the original dedup pass:
>
> 1. **FSM-as-tree-invariant adopted** (review #3). §10.1's rules now read as history-invariants, not just verb-preconditions. R-RULE-019 is **restated**: the "markdown edit is the back-out path" framing is removed; the rule now says markdown edits that bypass the FSM are illegal, and the back-out paths are reversal entities / `aiwf reallocate` / `--force --reason`. New rule R-RULE-149 records the `fsm-history-consistent` check that closes the chokepoint — implemented in **M-0130** (E-0033); closes **G-0132**.
> 2. **State-aware `CancelTarget` for Contract** (review #1). R-RULE-021 now distinguishes `proposed|accepted → rejected` from `deprecated → retired`. Note: the current code's `CancelTarget` is *not* state-aware — this is a **real bug** tracked as **G-0129, scheduled as M-0127** in E-0033.
> 3. **Reallocate exception restored** (review #4). R-RULE-127's referential-stability claim now explicitly admits reallocate as the documented exception, with history-side resolution via `prior_ids` + `aiwf-prior-entity:` trailers.
> 4. **Sovereign rationale added** (review #5). R-RULE-076's Notes now explain *why* `--force` excludes `aiwf-principal:` — sovereign = personal accountability; delegation is `aiwf authorize` + normal verb.
> 5. **Conditional-severity schema** (review #6). The Severity column now uses notation `base [escalation predicate → escalated]` for config-dependent severities. Affects 4 rules.
> 6. **Self-transitions explicit** (review #8). New R-RULE-150 makes the implicit "no (from, from) edges" rule explicit across all three FSMs (entity, AC, TDD phase).
> 7. **Force + persistent findings clarified** (review #2). New R-RULE-151 names the two-step pattern: `--force --reason` is permission-to-act, not permission-to-leave-broken; the resulting finding is work-tracking; push requires `--no-verify` until the finding is resolved.
>
> Review finding #7 (proposed ADRs ADR-0001, ADR-0009) is a process call outside the catalog's scope; surfaced to the user separately. R-RULE-145..147 remain marked `unenforced (ADR proposed)` until ratified or rejected.
>
> **Honest sweep (post-revision-2, 2026-05-18):** Re-opened verb files only read via `--help` during §4 extraction. Added 5 net-new rules (R-RULE-152..156): `aiwf milestone depends-on` (a real mutating verb missed in §4); `aiwf list --archived` filter semantics; `aiwf schema`/`template` no-consumer-repo relaxation; `aiwf whoami` git-config dependency; the `aiwf-tests:` trailer schema (write-strict / loose-read / aggregation rule). Also added an **enforcement-status legend at the top of §10.1** distinguishing the verb-time chokepoint from the history-walk chokepoint; the history-walk chokepoint is now active (`fsm-history-consistent`, landed in M-0130, closes G-0132).

Many of the 225 per-source rules are **facets** of the same underlying legality claim, enforced at multiple chokepoints. For example, *"a milestone may not transition to done while any AC has status: open"* is independently asserted by:

- R-AUDIT-0049 — `MilestoneCanGoDone` in `transition.go`
- R-AUDIT-0081 — `milestone-done-incomplete-acs` finding in `check.go`
- R-AUDIT-0114 — promote-verb pre-check in `cmd/aiwf/verbs_cmd.go`
- R-AUDIT-0149 (anti-rule acknowledgment) — design-decisions.md confirms this is intentional
- R-AUDIT-0189 — design-decisions.md `validate-then-write` makes the verb-time check authoritative

Five sources, one underlying rule. Pass C reconciliation needs the *underlying rule* in the spec table, not five rows. This section consolidates the 225 facet rules into **~145 unified rules**, each listing every chokepoint that enforces it.

Schema (8 columns; extends the per-source 6-column schema with **Chokepoints** and **Facets**):

| Column | Meaning |
|---|---|
| **Rule id** | `R-RULE-NNNN` — sequential within this section |
| **Category** | One of: FSM / Frontmatter / Refs / Ids / Provenance / Archive / Verb / Trailer / Body / Policy / Workflow / Anti-rule |
| **Scope** | Which kind, verb, or pair of kinds the rule applies to |
| **Statement** | The consolidated legality claim |
| **Chokepoints** | List of every place this rule is enforced (in `kind: location` form) |
| **Severity** | The strongest enforcement severity across all chokepoints |
| **Facets** | The R-AUDIT-NNNN ids that this rule consolidates |
| **Notes** | Any nuances (config-dependent severity, anti-rules, etc.) |

### 10.1 Entity FSM transitions

One rule per (kind, from-state) listing all legal next-states plus the chokepoints that enforce the closure.

**Framing (revision 2):** The FSM constrains the **history of every entity**, not just the preconditions of `aiwf promote` / `aiwf cancel`. Any state change visible in the working tree must trace to an FSM-legal transition in the git history. Direct markdown edits that bypass the verb path are illegal — they leave a state with no FSM-legal predecessor. The `fsm-history-consistent` check (R-RULE-149) closes this chokepoint.

**Enforcement status legend:** "hard-reject" in §10.1 rows means the legality claim is enforced through *two* chokepoints, both now active:

- **Verb-time path:** `aiwf promote` / `aiwf cancel` consult `ValidateTransition`; illegal (from, to) tuples return an error and the verb refuses to commit.
- **History-walk path (`fsm-history-consistent`, R-RULE-149):** Walks `git log` per entity and validates each frontmatter `status:` change against the FSM. Three subcodes partition the violation space disjointly (per D-0008):
  - `illegal-transition` (**error**) — the (from, to) tuple is not in the FSM and the commit carries no `aiwf-force:` trailer.
  - `forced-untrailered` (**error**) — the change matches a sovereign-act shape (e.g., epic `proposed → active`) by a non-human actor without the force trailer. The predicate mirrors M-0095's `requireHumanActorForSovereignAct` verb gate: a `human/` actor OR a non-empty `aiwf-force:` satisfies the discipline.
  - `manual-edit` (**warning**) — the status-change commit lacks an `aiwf-verb:` trailer at all. Warning severity is deliberate: the audit-only backfill (`aiwf <verb> --audit-only --reason "..."`) is the intended cure for state already correct on disk pending acknowledgment, so an error here would block legitimate cooperation. The rule cross-references HEAD's history for `aiwf-audit-only` + `aiwf-entity` commits that are *descendants* of the unacknowledged flip (chrono-aware suppression), so a cherry-picked ack on a parallel branch does not silently clear the warning.

Merge commits are skipped (per D-0010): a merge is not a status flip event in itself; the legality of the branch tips it joins is judged at the tip commits, not at the merge.

Both paths together satisfy the kernel's "correctness must not depend on the LLM choosing to enforce" commitment. §10.1 rows say "hard-reject" — accurate for verb-mediated illegal targets and sovereign-act-shape flips; for pure manual edits the chokepoint is warning-severity by design (an audit-only acknowledgment is the legal back-out, not a hard block on push). Implemented in M-0130 (E-0033); closes G-0132. The existing `provenance-untrailered-entity-commit` (warning) remains as the broader trailer-absence chokepoint covering non-FSM trailers.

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-001 | FSM | Epic `proposed` | Legal targets: `active` (via promote), `cancelled` (via cancel). Any other target is illegal | FSM: transition.go L14; Validator: ValidateTransition; Verb: promote pre-check; Check: status-valid; Policy: aiwf_promote_epic_active_audit (requires `--force --reason` for `active`) | hard-reject | R-AUDIT-0001, 0002, 0029, 0050, 0091, 0111, 0113 | `proposed → active` is sovereign-act (human-only, requires `--force --reason`) per R-AUDIT-0050 |
| R-RULE-002 | FSM | Epic `active` | Legal targets: `done` (via promote), `cancelled` (via cancel). Any other target is illegal | FSM: transition.go L15; Validator + Verb + Check (same as R-RULE-001) | hard-reject | R-AUDIT-0003, 0004, 0029, 0091, 0111 | — |
| R-RULE-003 | FSM | Epic `done`, `cancelled` | Terminal — no outgoing transitions | FSM: transition.go L16-17; Validator; Verb; Check | hard-reject | R-AUDIT-0005, 0029, 0091, 0111 | — |
| R-RULE-004 | FSM | Milestone `draft` | Legal targets: `in_progress` (via promote), `cancelled` (via cancel) | FSM: transition.go L20; Validator; Verb; Check | hard-reject | R-AUDIT-0006, 0007, 0029, 0091, 0111 | — |
| R-RULE-005 | FSM | Milestone `in_progress` | Legal targets: `done` (via promote, with AC precondition), `cancelled` (via cancel) | FSM: transition.go L21; Validator; Verb; Check | hard-reject | R-AUDIT-0008, 0009, 0029, 0091, 0111 | `→ done` requires `MilestoneCanGoDone` precondition (R-RULE-024) |
| R-RULE-006 | FSM | Milestone `done`, `cancelled` | Terminal | (same chokepoints) | hard-reject | R-AUDIT-0010, 0029, 0091 | — |
| R-RULE-007 | FSM | ADR `proposed` | Legal targets: `accepted` (via promote), `rejected` (via cancel) | (same) | hard-reject | R-AUDIT-0011, 0012, 0029, 0091, 0111 | ADR ratification should not be gated by external schedule (CLAUDE.md §Authoring an ADR; R-AUDIT-0193) |
| R-RULE-008 | FSM | ADR `accepted` | Legal target: `superseded` (via promote) | (same) | hard-reject | R-AUDIT-0013, 0029, 0091, 0111 | — |
| R-RULE-009 | FSM | ADR `superseded`, `rejected` | Terminal | (same) | hard-reject | R-AUDIT-0014, 0029, 0091 | — |
| R-RULE-010 | FSM | Gap `open` | Legal targets: `addressed` (via promote), `wontfix` (via cancel) | (same) | hard-reject | R-AUDIT-0015, 0016, 0029, 0091, 0111 | `addressed` requires `resolved-by:` frontmatter (R-RULE-019) |
| R-RULE-011 | FSM | Gap `addressed`, `wontfix` | Terminal | (same) | hard-reject | R-AUDIT-0017, 0029, 0091 | — |
| R-RULE-012 | FSM | Decision `proposed` | Legal targets: `accepted` (via promote), `rejected` (via cancel) | (same) | hard-reject | R-AUDIT-0018, 0019, 0029, 0091, 0111 | — |
| R-RULE-013 | FSM | Decision `accepted` | Legal target: `superseded` (via promote) | (same) | hard-reject | R-AUDIT-0020, 0029, 0091, 0111 | — |
| R-RULE-014 | FSM | Decision `superseded`, `rejected` | Terminal | (same) | hard-reject | R-AUDIT-0021, 0029, 0091 | — |
| R-RULE-015 | FSM | Contract `proposed` | Legal targets: `accepted` (via promote), `rejected` (via cancel) | (same) | hard-reject | R-AUDIT-0022, 0023, 0029, 0091, 0111 | — |
| R-RULE-016 | FSM | Contract `accepted` | Legal targets: `deprecated` (via promote), `rejected` (via cancel) | (same) | hard-reject | R-AUDIT-0024, 0025, 0029, 0091, 0111 | — |
| R-RULE-017 | FSM | Contract `deprecated` | Legal target: `retired` (via promote) | (same) | hard-reject | R-AUDIT-0026, 0029, 0091, 0111 | — |
| R-RULE-018 | FSM | Contract `retired`, `rejected` | Terminal | (same) | hard-reject | R-AUDIT-0027, 0029, 0091 | — |
| R-RULE-019 | FSM-meta | All kinds | FSM is one-directional; no "demote" verb exists. **Markdown edits that bypass the FSM are illegal**, not a documented back-out path. To undo a transition, file a new entity (e.g., a reversal decision); to renumber, use `aiwf reallocate`; to override the FSM at write time, use `--force --reason`. | transition.go L9-11; R-RULE-149 (fsm-history-consistent) | hard-reject (under tree-invariant interpretation; chokepoint active per R-RULE-149) | R-AUDIT-0028 | **Revised in revision 2** — original framing ("markdown is the back-out path") contradicted the tree-invariant interpretation. |
| R-RULE-020 | FSM-meta | All kinds | `IsTerminal(kind, status)` derives terminality from the FSM (state with no outgoing edges); no parallel hardcoded list | transition.go L93-103 | hard-reject (relied on by check rules) | R-AUDIT-0030 | — |
| R-RULE-021 | FSM | Cancel target | `aiwf cancel` targets per (kind, current state). Single-lifecycle kinds: Epic → `cancelled`; Milestone → `cancelled`; Gap → `wontfix`. Multi-lifecycle kinds (state-aware): ADR/Decision `proposed\|accepted` → `rejected`; Contract `proposed\|accepted` → `rejected`, **Contract `deprecated` → `retired`** (the natural lifecycle endpoint; cancel-from-deprecated to rejected would violate the FSM since `deprecated → rejected` is not legal). | transition.go L130-149 (`CancelTarget(kind, currentStatus)`); verb-level `IsTerminal` pre-flight in `Cancel` (promote.go) and reverse-lookup over `CancelTarget` in `CancelAuditOnly` (auditonly.go) keep the verbs in lock-step with the per-kind projection map | hard-reject | R-AUDIT-0031, 0032, 0033 | State-aware mapping landed in **M-0131** (closes **G-0131**): the signature now takes `(kind, currentStatus)` and the Contract case branches `deprecated → retired` vs `proposed\|accepted → rejected`. |

### 10.2 AC FSM + TDD phase

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-022 | FSM | AC `open` | Legal targets: `met`, `deferred`, `cancelled` | transition.go L128; IsLegalACTransition; promote-verb dispatch; acs-transition finding | hard-reject (without `--force --reason`) | R-AUDIT-0034, 0035, 0036, 0040, 0041 | — |
| R-RULE-023 | FSM | AC `met` | Legal targets: `deferred`, `cancelled` (scope-change after the fact) | (same) | hard-reject | R-AUDIT-0037, 0038, 0041 | — |
| R-RULE-024 | FSM | AC `deferred`, `cancelled` | Terminal AC states | (same) | hard-reject | R-AUDIT-0039 | — |
| R-RULE-025 | FSM | TDD phase | `(absent) → red → green → (refactor →) done`; `refactor` is optional; entry at any state other than `red` from absent is disallowed | transition.go L160-164; IsLegalTDDPhaseTransition; promote-verb `--phase` dispatch | hard-reject | R-AUDIT-0042, 0043, 0044, 0045, 0046, 0047, 0048 | — |
| R-RULE-026 | Cross-FSM | Milestone | `promote M-NNN in_progress → done` requires no AC has `status: open`; `MilestoneCanGoDone` lists offenders | transition.go L193-203; runMilestoneDoneACs (`milestone-done-incomplete-acs` finding); promote-verb pre-check | check-error + verb-time block | R-AUDIT-0049, 0081, 0114, 0149 | The check fires on every `aiwf check` pass even after `--force --reason`; the verb-time check is overridable by `--force`. See **R-RULE-151** for the operator pattern when a forced-done milestone leaves the finding in place. |
| R-RULE-027 | Cross-FSM | Milestone (anti-rule) | NOT a kernel rule: "milestone must have ≥1 AC" — ACs are optional | (design-decisions.md §ACs anti-rule list) | acknowledgment | R-AUDIT-0174 | — |
| R-RULE-028 | Cross-FSM | Milestone (anti-rule) | NOT a kernel rule: "milestone cannot enter in_progress without all ACs in red" — kernel guards outcome, not entry | (same) | acknowledgment | R-AUDIT-0175 | — |
| R-RULE-029 | Cross-FSM | TDD audit | Under milestone `tdd: required`, AC `status: met` requires `tdd_phase: done`; fires `acs-tdd-audit` | check/acs.go runACsTDDAudit; CLAUDE.md (mechanical evidence even under tdd:none/advisory) | `check-warning [milestone.tdd == "required" → check-error]` | R-AUDIT-0073, 0145, 0195, 0196 | Conditional-severity schema (revision 2): the base severity is warning; the milestone's `tdd:` policy escalates to error when `required`. |

### 10.3 Frontmatter shape + body

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-030 | Frontmatter | All kinds | Required common fields: `id`, `title`, `status` | check.go `runFrontmatterShape`, `runTitlesNonempty`, `runStatusValid` | check-error | R-AUDIT-0088, 0091, 0092, 0155 | — |
| R-RULE-031 | Frontmatter | All kinds | `status` must be in the kind's closed FSM state-set | check.go `runStatusValid`; closed_set_status_constants policy | check-error + policy-block | R-AUDIT-0053, 0091 | — |
| R-RULE-032 | Frontmatter | All kinds | `title` must be non-empty; length capped at `entities.title_max_length` (default 80; hard-reject over cap) | check.go `runTitlesNonempty`; add/retitle verbs | hard-reject (write-time) / check-error (read-time) | R-AUDIT-0092, 0110 | — |
| R-RULE-033 | Frontmatter | Milestone | Required: `parent` (epic id); optional: `depends_on` (milestone ids) | check.go `runRefsResolve`, `runNoCycles` | check-error | R-AUDIT-0093, 0094, 0156, 0157 | — |
| R-RULE-034 | Frontmatter | Milestone | Required: `tdd: required\|advisory\|none` (default `none` when absent); set at creation time | add-milestone verb required flag; check.go shape check | hard-reject (write-time) | R-AUDIT-0107, 0172 | — |
| R-RULE-035 | Frontmatter | Milestone | `acs[]` items: `id` (`AC-N`, position-stable), `title`, `status` in closed set, optional `tdd_phase` in closed set | check.go `runACsShape` (subcodes: id, title, status, tdd-phase) | check-error | R-AUDIT-0074, 0075, 0076, 0077, 0078 | — |
| R-RULE-036 | Frontmatter | ADR | Optional `supersedes` (list), `superseded_by` (single); mutually consistent | check.go `runADRSupersession` | check-error | R-AUDIT-0097, 0158 | — |
| R-RULE-037 | Frontmatter | Gap | Optional `discovered_in` (milestone or epic); optional `addressed_by` (any kind); `addressed` status requires `resolved-by:` field | check.go `runRefsResolve`, `runGapResolvedHasResolver` | check-error | R-AUDIT-0089, 0093, 0159 | — |
| R-RULE-038 | Frontmatter | Decision | Optional `relates_to` (any-kind ids) | check.go `runRefsResolve` | check-error | R-AUDIT-0093, 0160 | — |
| R-RULE-039 | Frontmatter | Contract | Optional `linked_adrs` (ADR ids) | check.go `runRefsResolve` | check-error | R-AUDIT-0093, 0161 | — |
| R-RULE-040 | Frontmatter | All kinds | Timestamps (`created`, `updated`) deliberately absent — `git log` carries them | design-decisions.md §Frontmatter schema | unenforced (convention) | R-AUDIT-0162 | Renderers must not synthesize them |
| R-RULE-041 | Frontmatter | aiwf.yaml | No identity fields (`actor`, `principal`) on aiwf.yaml structs; runtime-derived from `git config user.email` | policies/no_actor_in_aiwfyaml.go; I2.5 design | policy-block + hard-reject | R-AUDIT-0059, 0181 | — |
| R-RULE-042 | Body | All kinds | Body sections are starting points, not enforced structure; prose is human's responsibility | (design-decisions.md §Body templates) | unenforced (advisory) | R-AUDIT-0163 | `entity-body-empty` finding fires on empty sections but body shape is not validated |
| R-RULE-043 | Body | Milestone | Every frontmatter `acs[]` entry must have a matching `### AC-N — <title>` body heading | check.go `runACsBodyCoherence` | check-warning | R-AUDIT-0079 | — |
| R-RULE-044 | Body | Milestone | AC title that is long / multi-sentence / contains markdown fires advisory | check.go `runACsTitleProse` | check-warning | R-AUDIT-0080 | — |
| R-RULE-045 | Body | All kinds | Empty body section (e.g., `## Acceptance criteria` with no content) fires advisory | check.go `runEntityBodyEmpty` | check-warning | R-AUDIT-0085 | — |

### 10.4 IDs + references + archive

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-046 | Ids | All kinds | Id format `<prefix>-NNNN` canonical 4-digit width on output; parsers tolerate 1-3 digit legacy widths on input (ADR-0008) | check.go `runEntityIDNarrowWidth` (finding); rewidth verb (migration); renderers always emit canonical | check-warning + hard-reject (rewidth) | R-AUDIT-0086, 0121, 0144, 0145 | — |
| R-RULE-047 | Ids | All kinds | Ids are sequential per-kind, allocated by scanning working tree + trunk ref (`allocate.trunk` config); `max + 1` policy | allocator (gitops); design-decisions.md §Stable ids | hard-reject (allocator contract) | R-AUDIT-0164 | — |
| R-RULE-048 | Ids | All kinds | Id is the primary key; slug is just display; renames preserve id | rename verb; design-decisions.md §Cross-cutting | hard-reject (verb contract) | R-AUDIT-0117, 0152 | — |
| R-RULE-049 | Ids | All kinds | Frontmatter `id:` must match the on-disk filename's id-prefix | check.go `runIDPathConsistent` | check-error | R-AUDIT-0095 | — |
| R-RULE-050 | Ids | All kinds | Two entity files with the same canonical id fire | check.go `runIDsUnique` | check-error | R-AUDIT-0090 | — |
| R-RULE-051 | Ids | All kinds | Id format never extended with suffixes (`M-0007a/b`); collision recovery is renumbering via `aiwf reallocate` | design-decisions.md §Stable ids; reallocate verb | hard-reject (allocator contract) | R-AUDIT-0119, 0166 | — |
| R-RULE-052 | Ids | Reallocate verb | Writes `prior_ids: []` in new entity frontmatter + `aiwf-prior-entity:` trailer alongside new `aiwf-entity:` so both histories queryable | reallocate verb | hard-reject | R-AUDIT-0119, 0167 | — |
| R-RULE-053 | Refs | All kinds | All cross-references must resolve to existing entities (active or archive) | check.go `runRefsResolve`; policies/no_dangling_entity_refs.go | check-error + policy-block | R-AUDIT-0060, 0093 | — |
| R-RULE-054 | Refs | All kinds | No cycles in the entity-reference graph (parent / depends_on edges) | check.go `runNoCycles` | check-error | R-AUDIT-0094 | — |
| R-RULE-055 | Refs | Composite ids | Open-target ref fields (`gap.addressed_by`, `decision.relates_to`) accept `M-NNN/AC-N`; closed-target fields unchanged | check.go `runRefsResolve`; design-decisions.md §ACs | check-error | R-AUDIT-0179 | — |
| R-RULE-056 | Refs | AC scope | AC ids are namespaced inside their milestone (`M-NNN/AC-N`); not a seventh entity kind; position-stable | design-decisions.md §ACs | hard-reject (composite-id grammar) | R-AUDIT-0171 | — |
| R-RULE-057 | Refs | Filesystem | An entity file path differing only by case from another (CI-FS collision) fires | check.go `runCasePaths` | check-error | R-AUDIT-0098 | — |
| R-RULE-058 | Refs | Tree | Files under `work/` that don't match expected entity shape fire | check.go `runUnexpectedTreeFile` | `check-warning [aiwf.yaml.tree.strict == true → check-error]` | R-AUDIT-0096 | Conditional-severity (revision 2). |
| R-RULE-059 | Archive | All kinds | Terminal-status entities are swept to per-kind `archive/` subdirs via `aiwf archive --apply` (one commit per invocation); movement decoupled from FSM promotion | ADR-0004; archive verb; archive_rules.go (`archived-entity-not-terminal` blocks, `terminal-entity-not-archived` warns) | hard-reject + check-error + check-warning | R-AUDIT-0082, 0083, 0124, 0136, 0137 | — |
| R-RULE-060 | Archive | All kinds | Loader resolves ids transparently across active and archive | tree.Load; design-decisions.md §Archive (ADR-0004) | hard-reject (loader contract) | R-AUDIT-0138 | — |
| R-RULE-061 | Archive | All kinds | Archive sweep is one-way; reversal is by filing a new entity referencing the archived one | ADR-0004 §Reversal | unenforced (no reversal verb; convention) | R-AUDIT-0139 | — |
| R-RULE-062 | Archive | All kinds | N terminal entities awaiting `aiwf archive --apply` produce advisory; flips to blocking past `aiwf.yaml.archive.sweep_threshold` | check/archive_rules.go `runArchiveSweepPending` | `check-warning [pending_count > aiwf.yaml.archive.sweep_threshold → check-error]` | R-AUDIT-0084 | Conditional-severity (revision 2). |
| R-RULE-063 | Archive | All kinds | "Removal" is status flip (`cancelled`/`wontfix`/`rejected`/`retired`), not file deletion. File stays; references stay valid | design-decisions.md §Stable ids; cancel verb | hard-reject (verb contract) | R-AUDIT-0115, 0165 | — |

### 10.5 Provenance + sovereign acts

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-064 | Provenance | All mutating verbs | Operator identity runtime-derived from `git config user.email`; `--actor` overrides per invocation; `aiwf.yaml.actor` removed (I2.5) | gitops actor derivation; policies/no_actor_in_aiwfyaml.go | hard-reject + policy-block | R-AUDIT-0059, 0181 | — |
| R-RULE-065 | Provenance | All mutating verbs | Three-layer trailer set: `aiwf-actor:` + `aiwf-principal:` (when actor is non-human) + `aiwf-on-behalf-of:` + `aiwf-authorized-by:` (required-together for scope membership) | gitops trailers; provenance check rules; principal_write_sites policy | hard-reject + check-error + policy-block | R-AUDIT-0103, 0182 | — |
| R-RULE-066 | Provenance | Non-human actors | `--principal human/<id>` required when `--actor` is non-human; forbidden when human; OR active scope must cover entity | cliutil principal-required guard; provenance check | hard-reject + check-error | R-AUDIT-0104, 0214 | — |
| R-RULE-067 | Provenance | Trailer write sites | Sites writing `TrailerPrincipal` or `TrailerOnBehalfOf` must reference `"human/"` (human-only by design) | policies/principal_write_sites.go | policy-block | R-AUDIT-0068 | — |
| R-RULE-068 | Provenance | Trailer schema | Trailers via `gitops.Trailer{Key, Value}` struct only; no Sprintf-composed trailer lines | policies/no_trailer_string_composition.go; gitops.ValidateTrailer | policy-block + hard-reject | R-AUDIT-0067 | — |
| R-RULE-069 | Provenance | Trailer keys | String literals matching `gitops.Trailer*` constant values forbidden outside `internal/gitops/` | policies/trailer_keys.go | policy-block | R-AUDIT-0071 | — |
| R-RULE-070 | Provenance | Trailer keys | Ad-hoc `<role>/<id>` regex construction outside `gitops.roleIDPattern` forbidden | policies/no_role_id_regex.go | policy-block | R-AUDIT-0063 | — |
| R-RULE-071 | Scope FSM | Authorize | Scope FSM `active \| paused \| ended`; opened by `aiwf authorize <id> --to <agent>`; refused on terminal scope-entity unless `--force --reason` | authorize verb; design-decisions.md §Provenance | hard-reject | R-AUDIT-0122, 0183, 0218 | — |
| R-RULE-072 | Scope FSM | Authorize | `--pause "..."` pauses most-recently-opened active scope on entity; `--resume "..."` resumes most-recently-paused | authorize verb | hard-reject | R-AUDIT-0123, 0219 | — |
| R-RULE-073 | Scope FSM | All terminal-promotes | Scope ends automatically when scope-entity reaches terminal status (`aiwf-scope-ends:` trailer on the terminal-promote commit). Strict end-on-terminal: un-canceling does not resurrect ended scopes | design-decisions.md §Provenance | hard-reject | R-AUDIT-0184 | — |
| R-RULE-074 | Scope FSM | Provenance gating | A verb is allowed iff entity-FSM transition is legal AND (for non-human actors) at least one active scope's reachability check passes | provenance check; verb dispatch | hard-reject + check-error | R-AUDIT-0186 | Gating, not containment |
| R-RULE-075 | Scope FSM | Human actors | Human actors with no `--principal`: scope checks skipped — humans need no authorization | provenance check; design-decisions.md §Provenance | unenforced exemption (intentional) | R-AUDIT-0187 | — |
| R-RULE-076 | Sovereign | All verbs | `--force` is human-only; non-human actors refused. Forced acts carry `aiwf-actor: human/...` + `aiwf-force:` trailers ONLY — no principal, no on-behalf-of | cliutil sovereign guard; policies/sovereign.go | hard-reject + policy-block | R-AUDIT-0070, 0105, 0185 | **Rationale (revision 2):** Sovereign = personal accountability. A human acting "on behalf of" their team in an emergency is asserting *personal* responsibility for the call; the team backs the decision politically, not provenance-wise. Allowing `aiwf-principal:` on a sovereign act would imply delegation, which contradicts the meaning of sovereign override. If delegation is genuinely needed, that's not sovereign — that's `aiwf authorize <id> --to <agent>` followed by a normal verb call. |
| R-RULE-077 | Sovereign | All verbs | `--audit-only --reason "..."` backfills audit trail when state was reached via manual commit; entity must already be at target state (no FSM transition); mutex with `--force`; human-only; produces empty-diff commit with `aiwf-audit-only:` trailer | promote/cancel verbs; cliutil sovereign guard | hard-reject | R-AUDIT-0054, 0215 | — |
| R-RULE-078 | Sovereign | Epic activate | `promote E-NNNN active` is sovereign: requires `--force --reason "..."` AND `human/...` actor; agent acts cannot activate epics | sovereign verb dispatch; policies/aiwf_promote_epic_active_audit.go (static); policies/sovereign.go (dispatcher); M-0095 runtime rule | hard-reject + policy-block | R-AUDIT-0050, 0113 | — |
| R-RULE-079 | Trailer | Write sites of authorized-by | Must reference `Allow(` or `gateAndDecorate` — hand-stamping authorize SHAs forbidden | policies/authorized_by_via_allow.go | policy-block | R-AUDIT-0052 | — |

### 10.6 Verb behavior + commit invariants

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-080 | Verb | All mutating verbs | Every mutating verb acquires the repo lock (`internal/repolock`) before reading/writing; concurrent invocations serialize | cliutil.AcquireRepoLock; policies/apply_callers_lock.go | hard-reject + policy-block | R-AUDIT-0051, 0101 | — |
| R-RULE-081 | Verb | All mutating verbs | Validate-then-write: verb computes projected new tree in memory, runs `aiwf check` against the projection, only writes when clean; partial failure rolls back | verb.Apply (the only writer); policies/verbs_validate_then_write.go | hard-reject + policy-block | R-AUDIT-0072, 0102, 0188 | — |
| R-RULE-082 | Verb | All mutating verbs | Verbs only block on findings introduced by the projection; pre-existing tree errors do not refuse unrelated verbs | verb.Apply diff logic | hard-reject (verb policy) | R-AUDIT-0189 | — |
| R-RULE-083 | Verb | All mutating verbs | Exactly one git commit per verb (or zero if verb errors before staging). Empty-diff commits require a marker trailer (`aiwf-scope:` or `aiwf-audit-only:`) | verb.Apply; policies/empty_diff.go | hard-reject + policy-block | R-AUDIT-0054, 0102 | — |
| R-RULE-084 | Verb | Read-only verbs | `check, history, status, render` (no `--write`), `doctor`, `show`, `list`, `schema`, `template`, `whoami`, `version`, `completion`, `contract verify/recipes/recipe show` do not take the repo lock; do not commit; do not modify disk | policies/read_only.go | policy-block | R-AUDIT-0133 | — |
| R-RULE-085 | Verb | All mutating verbs | Commit subjects follow Conventional Commits format (`feat(aiwf): ...`, `chore(aiwf): ...`, `docs: ...`); trailers carry `aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:` minimum | CLAUDE.md §Commit conventions; gitops | unenforced subject (convention); hard-reject trailers (schema) | R-AUDIT-0103, 0198 | — |
| R-RULE-086 | Verb | All mutating verbs | Every promote event carries `aiwf-to: <state>` trailer so the target state is structured (I2 trailer schema extension) | gitops; promote verb | hard-reject (schema) | R-AUDIT-0178 | — |
| R-RULE-087 | Verb | History | History rewriting (`push --force-with-lease`, `reset --hard`, `commit --amend`, `rebase`, `filter-branch`) forbidden in non-test production source | policies/no_history_rewrites.go | policy-block | R-AUDIT-0062 | — |
| R-RULE-088 | Verb | All verbs | `GIT_AUTHOR_DATE` / `GIT_COMMITTER_DATE` manipulation forbidden in non-test source | policies/no_timestamp_manipulation.go | policy-block | R-AUDIT-0066 | — |
| R-RULE-089 | Verb | All verbs | Signing/hook bypass (`--no-verify`, `--no-gpg-sign`, `commit.gpgsign=false`) forbidden in non-test source | policies/no_signature_bypass.go | policy-block | R-AUDIT-0064 | `--no-verify` remains a deliberate escape hatch for humans at the git CLI level (R-AUDIT-0168) |
| R-RULE-090 | Verb | All verbs | Retry loops around git invocations forbidden in production code; diagnose and surface | (CI hygiene policy — out of scope per ADR-0011, listed for cross-reference) | policy-block | (out-of-scope list) | — |
| R-RULE-091 | Verb | All verbs (closed-set switches) | `switch` on closed-set types (`Kind`, `Status`) must have non-silent default (error / sentinel return) | policies/no_silent_fallback.go | policy-block | R-AUDIT-0065 | — |

### 10.7 Per-verb headline preconditions

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-092 | Verb | `aiwf add <kind>` | Kind must be one of (epic, milestone, adr, gap, decision, contract) + `ac` subcommand; unknown kind hard-rejected | add verb dispatch | hard-reject | R-AUDIT-0106 | — |
| R-RULE-093 | Verb | `aiwf add milestone` | Requires `--epic <id>` (parent epic) AND `--tdd <required\|advisory\|none>` | add-milestone verb | hard-reject | R-AUDIT-0107, 0172 | — |
| R-RULE-094 | Verb | `aiwf add ac` | Parent milestone must be non-terminal (not `done`/`cancelled`); seeds `tdd_phase: red` when parent is `tdd: required`; otherwise omits the field | add-ac verb | hard-reject | R-AUDIT-0108, 0173 | — |
| R-RULE-095 | Verb | `aiwf add ac --tests` | Only legal when parent milestone is `tdd: required` | add-ac verb; --help | hard-reject | R-AUDIT-0109, 0217, 0226 | — |
| R-RULE-096 | Verb | `aiwf promote --phase` | Mutex with positional `<new-status>` | promote verb | hard-reject | R-AUDIT-0112, 0216 | — |
| R-RULE-097 | Verb | `aiwf edit-body` | Frontmatter untouched; body replacement only. `--body-file` must not start with `---` (refused) | edit-body verb | hard-reject | R-AUDIT-0116 | — |
| R-RULE-098 | Verb | `aiwf rename` | Bare-id form renames on-disk slug, id preserved; composite-id form (`M-NNN/AC-N`) updates `acs[].title` AND body `### AC-N — <title>` heading in one commit | rename verb dispatch | hard-reject | R-AUDIT-0117, 0180 | — |
| R-RULE-099 | Verb | `aiwf move` | Moves milestone to different epic; preserves milestone id; updates parent reference; target epic must exist and be non-terminal | move verb | hard-reject | R-AUDIT-0118 | — |
| R-RULE-100 | Verb | `aiwf retitle` | Updates frontmatter title and re-derives on-disk slug in one commit; new title subject to title_max_length cap | retitle verb | hard-reject | R-AUDIT-0120 | — |
| R-RULE-101 | Verb | `aiwf init` | One-time setup; refuses to run on a repo that already has `aiwf.yaml` | init verb | hard-reject | R-AUDIT-0125 | — |
| R-RULE-102 | Verb | `aiwf update` | Re-materializes embedded skills into `.claude/skills/aiwf-*/`; refuses to clobber user-modified hooks (`# aiwf:<hook>` marker controls regeneration) | update verb | hard-reject | R-AUDIT-0126 | — |
| R-RULE-103 | Verb | `aiwf upgrade` | Fetches newer binary via `go install`; re-execs into `aiwf update`; `--check` is read-only comparison; `--version vX.Y.Z` pins; pre-release pseudo-versions accepted (G43) | upgrade verb | hard-reject | R-AUDIT-0127, 0222 | — |
| R-RULE-104 | Verb | `aiwf import` | Bulk-creates entities from YAML/JSON manifest; one commit by default. `--on-collision <fail\|skip\|update>` (default fail). `--dry-run` validates without writing | import verb | hard-reject | R-AUDIT-0128, 0220, 0221 | — |
| R-RULE-105 | Verb | `aiwf contract bind` | Adds/replaces a binding; `--validator`, `--schema`, `--fixtures` required; `--force` to replace | contract bind verb | hard-reject | R-AUDIT-0129 | — |
| R-RULE-106 | Verb | `aiwf contract recipe install` | Installs validator from embedded set or `--from <path>`; `--force` required to overwrite | contract recipe install | hard-reject | R-AUDIT-0130 | — |
| R-RULE-107 | Verb | `aiwf contract recipe remove` | Removes declared validator; errors when any contract binding still references it | contract recipe remove | hard-reject | R-AUDIT-0131 | — |
| R-RULE-108 | Verb | `aiwf contract` | Engine owns orchestration; user owns validators. `aiwf` never ships `cue`/`ajv`; validators declared in `aiwf.yaml.contracts.validators` (name → command + argv) | design-decisions.md §Contracts | hard-reject (verb contract) | R-AUDIT-0169 | — |
| R-RULE-109 | Verb | `aiwf contract verify` | Per binding: fixtures live under `<fixtures>/<version>/{valid,invalid}/`. Verify runs every `valid/` (must pass) and `invalid/` (must fail); evolve runs historical `valid/` against current schema | contract verify; design-decisions.md §Contracts | hard-reject | (cited via §Contracts; no direct R-AUDIT row) | — |
| R-RULE-110 | Verb | Validator availability | Missing validator produces `validator-unavailable` warning by default; `aiwf.yaml.contracts.strict_validators: true` flips to error | contract verify; design-decisions.md §Contracts | `check-warning [aiwf.yaml.contracts.strict_validators == true → check-error]` | R-AUDIT-0170 | Conditional-severity (revision 2). |
| R-RULE-111 | Verb | `aiwf render --write` / `--format=html` | When `--write` (roadmap) or HTML output requested, files are written but only the roadmap commit is created; HTML output is regenerated artifact (gitignored when `aiwf.yaml.html.commit_output: false`) | render verb | hard-reject | R-AUDIT-0132 | — |
| R-RULE-112 | Verb | `aiwf rewidth --apply` | Migrates narrow-legacy entity ids to canonical 4-digit width (ADR-0008); idempotent; one commit | rewidth verb | hard-reject | R-AUDIT-0121, 0145 | — |
| R-RULE-113 | Verb | `aiwf check` | `--format <text\|json>`; `--pretty` only with `--format=json` | check verb; --help | hard-reject (closed-set flag) | R-AUDIT-0224 | — |
| R-RULE-114 | Verb | `aiwf doctor --check-latest` | Hits Go module proxy for latest published version; honors `GOPROXY=off`; network errors print "unavailable" without failing doctor | doctor verb | unenforced (best-effort) | R-AUDIT-0223 | — |
| R-RULE-115 | Verb | `aiwf history --show-authorization` | Includes full `aiwf-authorized-by` SHA on scope-authorized rows (text format only) | history verb | unenforced (display-only) | R-AUDIT-0225 | — |
| R-RULE-116 | Verb | `--root <path>` (universal) | Defaults to walking up from cwd looking for `aiwf.yaml`; else cwd; hard-reject if no `aiwf.yaml` found and `--root` not supplied | all verbs | hard-reject | R-AUDIT-0212 | — |

### 10.8 Discoverability + skill coverage

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-117 | Discoverability | All top-level verbs | Every top-level Cobra verb is reachable through an AI-discoverable channel (per-verb skill / topical skill / `--help`-only / discoverability-priority split) per ADR-0006 | policies/skill_coverage.go | policy-block | R-AUDIT-0140 | — |
| R-RULE-118 | Discoverability | Skill placement | Planning-conversation skills live in the rituals plugin; kernel-embedded skills are verb-wrapper shape (ADR-0007) | policies/skill_coverage.go | policy-block | R-AUDIT-0141, 0142 | — |
| R-RULE-119 | Discoverability | All Cobra flags / closed-set values | Tab-completion via Cobra's `ValidArgs` / `RegisterFlagCompletionFunc` / `completeEntityIDFlag` (CLAUDE.md §Engineering principles) | `cmd/aiwf/completion_drift_test.go` | policy-block | R-AUDIT-0192 | — |
| R-RULE-120 | Discoverability | All finding codes | Every finding code emitted by the kernel must appear in skill / `--help` / CLAUDE.md / `docs/pocv3/` (one channel an AI assistant routinely consults) | (out-of-scope per ADR-0011 §Scope; listed for cross-reference) | policy-block | (out-of-scope discoverability.go) | — |
| R-RULE-121 | Discoverability | All finding codes | Every finding code has a matching entry in `internal/check/hint.go` `hintTable` (for "what now?" rendering) | policies/finding_hints.go | policy-block | R-AUDIT-0055 | — |
| R-RULE-122 | Discoverability | All finding codes | Every finding code is referenced from at least one `*_test.go` file (proves emission is exercised) | policies/findings_have_tests.go | policy-block | R-AUDIT-0056 | — |
| R-RULE-123 | Discoverability | aiwf.yaml | Every yaml-tagged field on `internal/aiwfyaml/` structs must appear in skill / `--help` / CLAUDE.md / `docs/pocv3/` | (out-of-scope per ADR-0011; listed for cross-reference) | policy-block | (out-of-scope config_fields_discoverable.go) | — |
| R-RULE-124 | Discoverability | All Cobra verbs | Cross-reference symmetry: every `aiwf <verb>` mention in a skill body resolves to a registered top-level verb | policies/skill_coverage.go | policy-block | (subsumed by R-RULE-117) | — |

### 10.9 Architectural commitments

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-125 | Architecture | All verbs | Enforcement does not depend on the LLM choosing to enforce — skills are advisory; pre-push git hook and `aiwf check` are authoritative | (design commitment; chokepoint is the absence of LLM-required code paths) | architectural (no chokepoint per se) | R-AUDIT-0151, 0191, 0203 | — |
| R-RULE-126 | Architecture | All verbs | Engine is invocable without an AI — every verb takes flags, reads stable input formats, emits a JSON envelope, exits with documented codes | CLI contract; design-decisions.md | hard-reject (CLI contract) | R-AUDIT-0153 | — |
| R-RULE-127 | Architecture | All entities | Referential stability: an id once allocated **always means the same entity in the live tree under rename, move, or status-change to terminal**. Reallocate is the documented exception: at merge-collision time, one of the colliding entities is renumbered to free the contested id. The old id no longer resolves in the live tree; history-side resolution is preserved via the new entity's `prior_ids: []` frontmatter and the `aiwf-prior-entity:` trailer on the reallocate commit (so `aiwf history <old-id>` still answers). | loader contract; design-decisions.md §Stable ids; reallocate verb | hard-reject (loader contract) for the rename/move/status case; documented exception for reallocate | R-AUDIT-0152, 0167 | **Revised in revision 2** — original consolidation lost the qualifier and over-claimed. Reallocate is unavoidable given sequential per-kind ids + no suffixes + branch-collision tolerance. |
| R-RULE-128 | Architecture | Six kinds | Hardcoded in Go for the PoC; not driven by external YAML; extensibility deferred until real friction | kernel design; design-decisions.md §Six entity kinds | hard-reject (kernel commitment) | R-AUDIT-0154 | — |
| R-RULE-129 | Architecture | Pre-push hook | `aiwf check` runs as a `pre-push` git hook installed by `aiwf init`; `--no-verify` bypasses (standard git behavior) | init verb; hook script; design-decisions.md §Validation | hard-reject (hook installed) / unenforced (bypass remains) | R-AUDIT-0168 | — |
| R-RULE-130 | Architecture | Layered location-of-truth | Engine binary (machine-installed); per-project policy + planning state (in-repo, git-tracked); materialized skill adapters + git hooks (in-repo, gitignored/untracked, marker-managed) | design-decisions.md §Layered location-of-truth | hard-reject (materialization contract) | (no direct R-AUDIT row; embedded in 6 misc R-AUDIT entries) | — |

### 10.10 Workflow / skill-driven (advisory only)

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-131 | Workflow | Milestone start | Skill-prescribed sequence: preflight → promote in_progress → branch setup → per-AC TDD cycle → self-review → wrap | rituals/aiwfx-start-milestone | advisory | R-AUDIT-0204 | — |
| R-RULE-132 | Workflow | Milestone wrap | Skill-prescribed sequence: verify completion → final code review → doc-lint sweep → finalize wrap-side sections → promote done → render roadmap → commit + push (with human approval gates) | rituals/aiwfx-wrap-milestone | advisory | R-AUDIT-0205 | — |
| R-RULE-133 | Workflow | TDD cycle | Skill drives red → green → refactor → done with `aiwf promote --phase`; kernel enforces outcome (met requires done) | rituals/wf-tdd-cycle | advisory (skill) + hard-reject (outcome) | R-AUDIT-0206 | Hybrid: kernel guards outcome, skill drives flow |
| R-RULE-134 | Workflow | Decision capture | Skill-prescribed: mid-flight decisions open a new `D-NNN` entity; milestone spec's `## Decisions made during implementation` references it | rituals/aiwfx-record-decision | advisory | R-AUDIT-0207 | — |
| R-RULE-135 | Workflow | Patch | Skill-prescribed single-focused-change branch (`fix/...`, `patch/...`, `doc/...`, `chore/...`); merges to main when patch lands | rituals/wf-patch; ADR-0010 §Tier 2 | advisory | R-AUDIT-0208 | — |
| R-RULE-136 | Workflow | Epic wrap | Skill-prescribed: verify all child milestones done → promote done → render roadmap → merge epic branch to main with `--no-ff` | rituals/aiwfx-wrap-epic | advisory | R-AUDIT-0209 | — |
| R-RULE-137 | Workflow | Planning | Two planning skills prescribe interactive planning conversations producing `aiwf add` invocations | rituals/aiwfx-plan-epic, aiwfx-plan-milestones | advisory | R-AUDIT-0210 | — |
| R-RULE-138 | Workflow | Authorize | Skill prescribes opening scope before delegating to an AI agent; scope FSM and gating are kernel-enforced | rituals/aiwfx-authorize | advisory (workflow) + hard-reject (gating: R-RULE-071..074) | R-AUDIT-0211 | — |
| R-RULE-139 | Workflow | Subagent worktree isolation | Parent session bootstraps worktree via `git worktree add` BEFORE invoking `Agent`; chokepoint hook denies `isolation: "worktree"` kwarg | CLAUDE.md §Subagent worktree isolation; `.claude/hooks/validate-agent-isolation.sh` | hard-reject (hook denies) + advisory (workflow) | R-AUDIT-0199, 0200 | — |
| R-RULE-140 | Workflow | Branch context | ADR-0010 two-tier model: state-announcement + author iteration on main; ritual work on named branches (`epic/`, `milestone/`, `fix/`, …) | ADR-0010 §Tier 1, §Tier 2 | advisory (branch model; full chokepoint deferred to E-0030) | R-AUDIT-0147, 0148, 0149, 0197 | Branch choreography moves to E-0030 |
| R-RULE-141 | Workflow | Cross-repo plugin testing | SKILL.md fixtures live under `internal/policies/testdata/<skill-name>/`; drift-check against marketplace cache | CLAUDE.md §Cross-repo plugin testing | policy-block when cache present | R-AUDIT-0201, 0202 | — |
| R-RULE-142 | Workflow | ADR ratification | ADR captures the choice; planning sequences the action. No gate language in ADR bodies. FSM (`proposed → accepted\|rejected`) + `aiwf promote` are the only mechanical surfaces | CLAUDE.md §Authoring an ADR | unenforced (convention) | R-AUDIT-0193, 0194 | — |
| R-RULE-143 | Workflow | AC mechanical evidence | AC promotion requires a mechanical assertion (Go test / kernel finding-rule / fixture-validation script). Applies even under `tdd: none`/`advisory` | CLAUDE.md §AC promotion | unenforced (convention; chokepoint is human review) | R-AUDIT-0195, 0196 | — |
| R-RULE-144 | Workflow | Engineering principles | KISS, YAGNI, no half-finished implementations | CLAUDE.md §Engineering principles | unenforced (review-driven) | R-AUDIT-0190 | — |

### 10.11 Future / deferred (proposed ADRs)

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-145 | Future | All entity allocations | Inbox-state id allocation at trunk integration (ADR-0001 proposed) | (none yet — design only) | unenforced (ADR proposed) | R-AUDIT-0134 | Deferred |
| R-RULE-146 | Future | Findings as entities | Findings become a seventh entity kind (`F-NNNN`) per ADR-0003 | (none yet — accepted ADR but no implementation) | unenforced (deferred implementation) | R-AUDIT-0135 | — |
| R-RULE-147 | Future | Orchestration | Substrate-vs-driver split with trailer-only events (ADR-0009 proposed) | (none yet — design only) | unenforced (ADR proposed) | R-AUDIT-0146 | — |
| R-RULE-148 | Future | Skill tiering | Pure-skill-first; promotion to kernel verb requires trigger condition (ADR-0007 §Tiering) | (governance rule; no chokepoint) | unenforced | R-AUDIT-0143 | — |

### 10.12 Revision-2 additions

Three new rules added in revision 2 (per the external review):

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-149 | FSM-meta | All kinds | **`fsm-history-consistent` check rule** — walks `git log` over all reachable refs via the batched `gitops.BulkRevwalk` helper (M-0137/AC-1); reads frontmatter `status:` at each (commit, path) pair through the long-lived `gitops.BlobReader` cat-file pump (M-0137/AC-2). For each status change in history, verifies the (from-state, to-state) tuple is legal under the entity's FSM and that the commit carries the expected verb / actor / force trailers. **Four disjoint subcodes**: `illegal-transition` (FSM-illegal flip with no `aiwf-force:` — error), `forced-untrailered` (sovereign-act shape by a non-human actor without `aiwf-force:` — predicate mirrors M-0095's `requireHumanActorForSovereignAct` verb gate — error), `manual-edit` (FSM-legal flip without `aiwf-verb:`, cleared by a chrono-descendant `aiwf-audit-only` commit — warning), and `history-walk-error` (the walker hit a real failure reading the named entity's commit history — subprocess crash, blob-read protocol error, context cancelled mid-walk — error). The first three partition the legal-status-change observation space disjointly (per D-0008); `history-walk-error` is orthogonal — it surfaces walker failures so one transient subprocess error doesn't silently wipe the rule's findings (the M-0130 silent-swallow at `FSMHistoryConsistent:71-77`, closed in M-0137/AC-4+5 per CLAUDE.md §Engineering principles "Errors are findings, not parse failures"). Merge commits are skipped for the first three subcodes (per D-0010). This is the chokepoint that makes the FSM a *tree-invariant* rather than just a verb-precondition. | Check: `fsm-history-consistent` in `internal/check/fsm_history_consistent.go` (M-0130) + `internal/check/fsm_history_walker.go` (M-0137) | Per-subcode: `illegal-transition` = error (hard-reject), `forced-untrailered` = error (hard-reject), `manual-edit` = warning (cleared by audit-only acknowledgment), `history-walk-error` = error (one finding per failed (entity, commit) pair; partial findings preserved for the rest of the walk) | (new — no R-AUDIT facets) | **Implemented in M-0130 (E-0033); closes G-0132. Retrofitted in M-0137 (batched walker + history-walk-error subcode + partial-failure preservation); closes the fsm-history slice of G-0149.** The four subcodes partition the observation space: illegal-transition / forced-untrailered / manual-edit cover legal-status-change observations disjointly; history-walk-error covers the walker's own failure modes orthogonally. The existing `provenance-untrailered-entity-commit` (warning) remains as the broader trailer-absence chokepoint covering non-FSM trailers. |
| R-RULE-150 | FSM-meta | All FSMs | **Self-transitions are illegal for every FSM** (entity, AC, TDD phase). `ValidateTransition` enforces this by *absence* — `(from, from)` is never in any `transitions` map; the map lookup returns the allowed-list, the candidate `to == from` is not in it, and the validator errors with "cannot transition to <state>". Applies uniformly to all six entity kinds, the AC FSM, and the TDD-phase FSM. | transition.go (all `transitions` maps); `IsLegalACTransition`; `IsLegalTDDPhaseTransition` | hard-reject | R-AUDIT-0029, 0040 (entity, AC); also implicit in TDD phase FSM | **Added in revision 2** for explicitness. The code already enforces this; the rule was previously stated only for ACs (R-AUDIT-0040) and was implicit for entity / TDD-phase FSMs. |
| R-RULE-151 | Sovereign | `--force --reason` | `--force --reason "..."` is permission-to-act, not permission-to-leave-broken. The verb succeeds, the `aiwf-force:` trailer records the audit. The resulting tree state may carry standing-rule findings that fire on every subsequent `aiwf check` — that is the **work-tracking signal**, not a bug. To push a forced state, the human acknowledges by either (a) resolving the underlying findings via subsequent verbs, or (b) running `git push --no-verify` to bypass the pre-push hook. The check itself never auto-suppresses on the presence of `aiwf-force:` — suppressing would erase the signal that work remains. | sovereign-act chokepoint; `aiwf check` (no force-aware suppression); `--no-verify` (standard git escape hatch) | hard-reject (verb refusal without `--reason`); check-error/check-warning (the finding that persists post-force is unchanged); unenforced (the human's choice to push via `--no-verify`) | R-AUDIT-0070, 0105, 0168, 0185 | **Added in revision 2** to make the two-step pattern explicit. Original catalog implied (incorrectly) that `--force` resolved the underlying issue; it only authorizes the unusual state. |

### 10.13 Honest-sweep additions

Five rules surfaced by re-opening verb files I'd only read via `--help` during §4 extraction, plus the `aiwf-tests:` trailer schema that §6 mentioned but didn't dedicate a rule to. None are R-AUDIT facets; they're net-new entries identified during the post-revision sweep (2026-05-18).

| Rule id | Category | Scope | Statement | Chokepoints | Severity | Facets | Notes |
|---|---|---|---|---|---|---|---|
| R-RULE-152 | Verb | `aiwf milestone depends-on <M-id> --on M-id,M-id` / `--clear` | Mutating verb that sets or clears a milestone's `depends_on` frontmatter list. **Replace-not-append** semantics — `--on` overwrites the existing list, does not extend it. `--on` and `--clear` are mutually exclusive. Forward-compatible with G-0073's cross-kind generalization (`aiwf <kind> depends-on <id> --on <ids>` extends to other kinds without renaming). Parent `aiwf milestone` is non-Runnable (prints help). | cmd/aiwf/milestone_cmd.go `runMilestoneDependsOnCmd`; usage-error check L76-79 | hard-reject | (no R-AUDIT facets — sweep addition) | **Sweep addition (2026-05-18).** Missed in §4; the verb is real and has specific legality rules. The bare `aiwf milestone` with no subcommand exits 0 with help text (CLI convention). |
| R-RULE-153 | Verb | `aiwf list` | Read-only browser. Default semantic: list **non-terminal** entities (forward-compat with ADR-0004). `--archived` widens to include terminal-status entities. Filters: `--kind` (closed-set), `--status` (kind-aware via completion), `--parent` (entity id). `--format text\|json`; `--pretty` only with json. `--no-trunc` overrides terminal-width title truncation. | cmd/aiwf/list_cmd.go `runListCmd`; flag-validation L126-138 | hard-reject (closed-set flag values) | (no R-AUDIT facets — sweep addition) | **Sweep addition.** Subsumed by R-RULE-084 (general read-only) but the `--archived` semantic (default-hide terminal) is its own rule. |
| R-RULE-154 | Verb | `aiwf schema [kind]` / `aiwf template [kind]` | Read-only printers for frontmatter contract (`schema`) and body-section template (`template`). Distinct from other read-only verbs: **do not require a consumer repo** — they print the kernel-embedded schema/template content and exit. No `aiwf.yaml` lookup, no tree load. | cmd/aiwf/schema_cmd.go; cmd/aiwf/template_cmd.go | hard-reject (closed-set kind value when arg supplied) | (no R-AUDIT facets — sweep addition) | **Sweep addition.** §4 captured these as read-only verbs but missed their no-consumer-repo relaxation. |
| R-RULE-155 | Verb | `aiwf whoami` | Read-only; prints the resolved actor (from `git config user.email`, `--actor` override, or `AIWF_ACTOR` env if applicable) and the source it came from. Requires `git` to be on PATH and `git config user.email` to be set; otherwise prints diagnostic and exits non-zero. | cmd/aiwf/whoami_cmd.go | hard-reject (git config dependency) | (no R-AUDIT facets — sweep addition) | **Sweep addition.** Provenance debug primitive; aligns with R-RULE-064 (runtime-derived identity). |
| R-RULE-156 | Trailer | `aiwf-tests:` trailer | **Schema:** loose-read (any key=value tokens tolerated when parsing `aiwf history`'s output of existing commits); **write-strict** at kernel verbs (`--tests "pass=N fail=N skip=N [total=N]"`; recognized keys only — `pass`, `fail`, `skip`, `total`; non-negative integers; unknown keys or non-integer values rejected with usage error). **Aggregation rule:** for an AC, the first commit returned by `aiwf history M-NNN/AC-N` whose trailer is present is authoritative; subsequent commits' trailers are recorded but not aggregated into the AC's "active" metrics. **Stability:** rebase- and amend-stable because aggregation is by history ordering, not by SHA. **Opt-in finding:** `acs-tdd-tests-missing` fires only when `aiwf.yaml.tdd.require_test_metrics: true` AND milestone is `tdd: required` AND AC at `tdd_phase: done` with no `aiwf-tests:` trailer. | promote verb `--tests` flag parser; `aiwf history` trailer reader; check rule (opt-in finding); design-decisions.md §Governance HTML render | hard-reject (write-strict at verb); `check-warning [aiwf.yaml.tdd.require_test_metrics == true → check-warning]` (the opt-in fires the same severity either way; absent the opt-in, no finding) | R-AUDIT-0186 partial (trailer's existence is referenced); sweep addition for the schema rule itself | **Sweep addition.** §6 mentioned the trailer but didn't dedicate a rule to its schema. The write-strict / loose-read split + the aggregation rule are load-bearing for the Tests tab's rendering accuracy. |

**Total for §10.13: 5 rules**

---

## Post-dedup totals

| Section | Consolidated rules |
|---|---:|
| 10.1 Entity FSM transitions | 21 |
| 10.2 AC FSM + TDD phase | 8 |
| 10.3 Frontmatter shape + body | 16 |
| 10.4 IDs + references + archive | 18 |
| 10.5 Provenance + sovereign acts | 16 |
| 10.6 Verb behavior + commit invariants | 12 |
| 10.7 Per-verb headline preconditions | 25 |
| 10.8 Discoverability + skill coverage | 8 |
| 10.9 Architectural commitments | 6 |
| 10.10 Workflow / skill-driven (advisory) | 14 |
| 10.11 Future / deferred | 4 |
| 10.12 Revision-2 additions | 3 |
| 10.13 Honest-sweep additions | 5 |
| **Consolidated total** | **156** |

**Dedup math:** 225 facets → 148 consolidated rules (sections 10.1–10.11; ~34% compression). The 3 revision-2 additions (R-RULE-149/150/151) and the 5 sweep additions (R-RULE-152/153/154/155/156) are *new* rules, not consolidations of existing facets, so they don't enter the dedup ratio. Total catalog: 148 consolidated + 3 revision-2 + 5 sweep = **156 rules**.

Most of the compression came from FSM transitions (49 facet rows → 21 consolidated; each (kind, from-state) row now lists all chokepoints) and from cross-source overlaps (e.g., R-RULE-026's milestone-done precondition consolidates 5 facets across transition.go, check rules, verb pre-checks, and design-decisions.md).

**For the M-0121 ACs:**

- **AC-1** (per-source sections in spec order) — §§1–9 above remain as evidence sections; §10 is the consolidated review surface.
- **AC-2** (all nine audit sources covered) — §§1–9 each have rule rows or explicit no-rules acknowledgment.
- **AC-3** (six-column schema with non-empty fields) — §§1–9 use the 6-column schema; §10 uses an extended 8-column schema (adds Chokepoints + Facets).
- **AC-4** (catalog schema internally consistent) — R-AUDIT-NNNN ids 0001..0226 (with one ack at 0150); R-RULE-NNN ids 001..148 sequential; per-section totals add up.

---

## Grand total

**TBD — extraction in progress.**
