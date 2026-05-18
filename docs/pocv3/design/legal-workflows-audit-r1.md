# Legal-workflow audit catalog — Pass A, R1 (pristine extraction)

> **Methodology:** ADR-0011. **Milestone:** M-0121. **Status:** snapshot taken 2026-05-18 for Pass B independence.
>
> **Use this file for Pass B (M-0122).** This is the **pristine Pass A extraction** — §§1-9 capture what each kernel surface claims as a legality rule, with citations. It contains no Pass C reconciliation work and no interpretive amendments to facet statements.
>
> The working catalog at `legal-workflows-audit.md` (R2) contains the same §§1-9 evidence *plus* a §10 consolidation/dedup pass *plus* Revision-2 interpretive amendments incorporating external-review findings (FSM tree-invariant commitment, state-aware CancelTarget endorsement, conditional-severity schema, sovereign rationale, etc.). Those interpretive moves are properly Pass C concerns; R2 captures them ahead of time but is **off-limits for Pass B**.
>
> The methodology ADR (ADR-0011) commits Pass B to first-principles derivation **independent** of Pass A. R1 is what Pass B may consult (the source-of-truth extractions); R2 is what Pass C reconciles against (with both Pass B's first-principles catalog and R1 as inputs).
>
> Anything below this header is byte-identical to R2's §§1-9, sans this R1-specific framing. Diff R1 against R2 to see exactly what Pass C-shaped material has crept into the working catalog.

---

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

**Status (in this R1 file):** Extraction complete across all nine sources. R1 stops here. The dedup / consolidation work and the Revision-2 interpretive amendments live exclusively in R2 (`legal-workflows-audit.md`) and are out-of-bounds for Pass B per the methodology ADR.

