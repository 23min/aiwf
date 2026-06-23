---
id: M-0173
title: aiwf add --area write path with completion and discovered-in derivation
status: in_progress
parent: E-0043
depends_on:
    - M-0171
tdd: required
acs:
    - id: AC-1
      title: --area writes area into new entity frontmatter
      status: met
      tdd_phase: done
    - id: AC-2
      title: --area rejected when value undeclared or no areas block
      status: met
      tdd_phase: done
    - id: AC-3
      title: --area is invalid for non-root kinds
      status: met
      tdd_phase: done
    - id: AC-4
      title: --area tab-completes declared members
      status: met
      tdd_phase: done
    - id: AC-5
      title: gap derives area from discovered-in when --area omitted
      status: met
      tdd_phase: done
    - id: AC-6
      title: integration seam test covers set, reject, and derive paths
      status: met
      tdd_phase: done
---
## Goal

Add the write path for `area`: an `--area <name>` flag on `aiwf add` (the five root kinds), validated and tab-completed from `aiwf.yaml: areas`, with a gap deriving its area from `discovered_in` when `--area` is omitted. Changing an entity's area reverses through the same surface.

## Context

M-0171 makes the field exist; M-0172's `area-unknown` check finding catches undeclared values at check time. This milestone gives the operator the loud, completion-assisted way to *set* the field at creation — so a carve-out workstream is tagged in the same atomic commit that creates the entity, not by a later hand-edit. The write-time validation is the verb-time twin of the `area-unknown` check, reading the same declared set through the M-0171 accessor.

## Acceptance criteria

### AC-1 — --area writes area into new entity frontmatter

`aiwf add <root-kind> --area <name> ...` (epic, ADR, gap, decision, contract) writes `area: <name>` into the new entity's frontmatter in the same atomic creating commit.

Evidence: a verb test over the root kinds asserting the created entity's frontmatter carries the area (`TestAdd_Area_WritesFrontmatter`); the dispatcher set path (`TestRunAdd_AreaSetViaDispatcher`).

### AC-2 — --area rejected when value undeclared or no areas block

`--area <name>` is rejected with a usage error (exit 2, **no entity created**) when the value is not a member of the declared `aiwf.yaml: areas` set, or when no `areas` block exists. The error names the offending value and the declared set. Validation uses the M-0171 config accessor (`cliutil.ConfiguredAreaMembers` → `cfg.Areas.Members`) — the same declared set the `area-unknown` check reads (single source of truth, no parallel validator).

Evidence: `TestRunAdd_AreaRejected` — undeclared-value and no-block subcases, each asserting `ExitUsage`, no entity file, and the message naming the value and the declared members.

### AC-3 — --area is invalid for non-root kinds

`--area` is not accepted for a milestone, which derives its area from its parent epic and never stores its own. Passing `--area` to a non-root kind errors (usage error, no entity created).

Evidence: `TestAdd_Area_RejectedForMilestone` (verb) + `TestRunAdd_AreaRejectedForMilestone` (dispatcher).

### AC-4 — --area tab-completes declared members

`aiwf add <root-kind> --area <TAB>` completes exactly the declared `areas.members`, wired via Cobra `RegisterFlagCompletionFunc` (`cliutil.CompleteAreaFlag`) the same way other closed-set flags are. The completion-drift policy stays green — the flag is registered for completion.

Evidence: `TestRunAdd_AreaCompletion` (completion returns exactly the declared members); the completion-drift policy `TestPolicy_FlagsHaveCompletion` in `internal/cli/integration/completion_drift_test.go`.

### AC-5 — gap derives area from discovered-in when --area omitted

`aiwf add gap --discovered-in <id>` derives the gap's `area` from the discovered-in entity's **effective** area when `--area` is omitted and that entity has one — an epic carries `area` directly; a milestone target is a two-hop derivation through its parent epic (milestones don't store `area`), via the M-0171 `ResolvedAreaByID` seam. If the discovered-in entity has no effective area, the gap is left untagged. An explicit `--area` always takes precedence over derivation.

Decision: this resolves the epic's Open Question 1 — **derive-on-omit**.

Evidence: `TestRunAdd_GapDerivesArea` (derive-from-epic, derive-from-milestone two-hop, untagged-source, explicit-override ×2); `TestRunAdd_GapDerivesUndeclaredAreaAsIs` pins that derivation copies the effective area verbatim (no re-validation — the `area-unknown` check is the backstop).

### AC-6 — integration seam test covers set, reject, and derive paths

An integration test drives the real dispatcher end-to-end (test-the-seam) so the flag wiring, config validation, and derivation are proven together, not just at the unit layer: `add --area <declared>` (set), `add --area <undeclared>` (reject), and `add gap --discovered-in <id>` (derive).

Evidence: the `internal/cli/integration` dispatcher tests above, all driving `cli.Execute`.

## Constraints

- **Validate against config at write time** using the M-0171 accessor — no parallel validator. This is the verb-time twin of the `area-unknown` check-time finding; both read the same declared set.
- **Reversible by the same verb** — re-running with a different `--area` changes the tag; no bespoke "unset area" verb invented.

## Out of scope

- The `area-unknown` check finding (M-0172, done) and read surfaces (filter/grouping milestones M-0174–M-0175).
- A bulk re-tagging verb across many entities.

## Dependencies

- M-0171 — the `area` field, `aiwf.yaml: areas` block, and config accessor (`ResolvedArea` / `ResolvedAreaByID`).

## References

- [E-0043 epic](epic.md) · [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md)
- M-0172 — the `area-unknown` check finding; this milestone is its write-time twin.

## Work log

Implementation landed as a single `feat` commit on the milestone branch; the per-AC TDD phase timeline (red→green→done→met) is in `aiwf history M-0173/AC-<N>`.

- **AC-1** — `AddOptions.Area` lands as `area:` frontmatter across the root kinds; verb `applyAddOpts`. · `TestAdd_Area_WritesFrontmatter`, `TestRunAdd_AreaSetViaDispatcher`
- **AC-2** — write-time member validation in the cmd layer (`validateAreaMember`), exit 2, no entity, names value + set. · `TestRunAdd_AreaRejected`
- **AC-3** — verb `validateAddOptsForKind` rejects `--area` on a milestone (flag-vs-kind guard). · `TestAdd_Area_RejectedForMilestone`, `TestRunAdd_AreaRejectedForMilestone`
- **AC-4** — `cliutil.CompleteAreaFlag` + `RegisterFlagCompletionFunc("area", ...)`; offers exactly the declared members. · `TestRunAdd_AreaCompletion`, `TestPolicy_FlagsHaveCompletion`
- **AC-5** — gap derive-on-omit via `tr.ResolvedAreaByID(discoveredIn)` (epic direct, milestone two-hop); explicit `--area` wins. · `TestRunAdd_GapDerivesArea`, `TestRunAdd_GapDerivesUndeclaredAreaAsIs`
- **AC-6** — dispatcher seam coverage (set/reject/derive) through `cli.Execute`. · `internal/cli/integration/add_area_test.go`

## Decisions made during implementation

- **AC-5 = derive-on-omit (resolves the epic's Open Question 1).** When `--area` is omitted, a gap derives its area from the `--discovered-in` entity's effective area (epic direct; milestone two-hop through its parent epic); an untagged source leaves the gap untagged; an explicit `--area` always wins. A local behavior decision — recorded here and in the AC-5 body; the epic's open-question table is reconciled at epic wrap.
- **Derivation copies the effective area verbatim — no re-validation.** A derived value is the source's truth; if the source carries an undeclared area (only reachable via hand-edit / import, since the write path validates `--area`), the M-0172 `area-unknown` check is the backstop, not the write path. Pinned by `TestRunAdd_GapDerivesUndeclaredAreaAsIs` so a future re-validation change is a red test.
- **Validation split.** Kind-applicability (a milestone may not carry `--area`) lives in the verb (`validateAddOptsForKind`, alongside the `--tdd`/`--depends-on` guards); the config-dependent member check lives in the cmd layer (`validateAreaMember`, where config is loaded — mirroring how contracts thread config). The cmd skips the member check for a milestone so the verb's clearer kind error wins.
- **Severity / no knob** — n/a here; this is the write path. The check-time twin (M-0172) settled warning-no-knob.

## Validation

- `make check-fast` (go vet + all `internal/...` tests + golangci-lint full set): green.
- `go build ./...` (CGO_ENABLED=0): green.
- `aiwf check` (worktree diag binary): 0 errors (only the benign `provenance-untrailered-scope-undefined` warning — no upstream on the milestone branch).
- Coverage: every new conditional branch traversed; `validateAreaMember` / `ConfiguredAreaMembers` 100%; the one unreachable `CompleteAreaFlag` branch (`ResolveRoot` only errors on `os.Getwd` failure) is `//coverage:ignore`'d with rationale. Vacuity-proven: 7/7 mutation probes (area-write, milestone guard, member-match, no-block guard, derivation, explicit-precedence, derive-as-is) go red. CI diff-scoped coverage-gate confirms on push.
- `make ci` (race + coverage-gate + end-to-end self-check) at the merge boundary: green.

## Reviewer notes

- **Independent two-lens review (wrap step 2).** A fresh-context `reviewer` subagent (`wf-review-code`) returned **APPROVE**, verifying every AC by measurement (running tests, binary smoke of set/reject/derive/two-hop, confirming the single-source-of-truth accessor and the single production write-site, and checking the `//coverage:ignore` justification is honest). `wf-rethink` was not run: the milestone introduces no new package / abstraction / data model — it extends the existing `aiwf add` verb following the established flag→opts→verb and config-in-cmd patterns.
- **Reviewer findings addressed:** (1) the derive-copies-undeclared-as-is interaction was untested → added `TestRunAdd_GapDerivesUndeclaredAreaAsIs`; (2) the AC-4 evidence path was stale (`cmd/aiwf/...`) → corrected to `internal/cli/integration/completion_drift_test.go`. The pre-existing 17-positional-parameter `Run` signature was noted as out of scope (M-0173 added one parameter consistently).

## Deferrals

None. Read-surface filter/grouping is E-0043's subsequent milestones M-0174–M-0175 (out of scope by design). The epic's Open Question 1 is now decided (derive-on-omit); its open-question table is reconciled at epic wrap.
