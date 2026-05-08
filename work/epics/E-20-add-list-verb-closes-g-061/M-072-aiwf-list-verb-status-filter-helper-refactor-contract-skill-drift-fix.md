---
id: M-072
title: aiwf list verb, status filter-helper refactor, contract-skill drift fix
status: done
parent: E-20
tdd: required
acs:
    - id: AC-1
      title: Core flag set works end-to-end
      status: met
      tdd_phase: done
    - id: AC-2
      title: 'JSON envelope: result is array of summary objects'
      status: met
      tdd_phase: done
    - id: AC-3
      title: Default excludes terminal status; --archived includes them
      status: met
      tdd_phase: done
    - id: AC-4
      title: entity.IsTerminal(kind, status) helper added
      status: met
      tdd_phase: done
    - id: AC-5
      title: Closed-set completion wired for --kind and --status
      status: met
      tdd_phase: done
    - id: AC-6
      title: Shared filter helper extracted; status uses it
      status: met
      tdd_phase: done
    - id: AC-7
      title: Status text and JSON goldens unchanged after refactor
      status: met
      tdd_phase: done
    - id: AC-8
      title: contracts-plan and contract-skill drift fixed
      status: met
      tdd_phase: done
    - id: AC-9
      title: Verb-level integration test drives the dispatcher
      status: met
      tdd_phase: done
---

# M-072 — aiwf list verb, status filter-helper refactor, contract-skill drift fix

## Goal

Ship the `aiwf list` verb as the AI's hot-path read primitive over the planning tree, with V1 flags `--kind / --status / --parent / --archived / --format / --pretty`, and refactor `aiwf status`'s kind/status filter slices into a shared helper that `list` also uses so the two verbs cannot drift.

## Context

`aiwf status` already loads the planning tree via `tree.Load` and filters in-memory (`cmd/aiwf/status_cmd.go:204–211`); this milestone reuses that loader and extracts the filter slice loops at `status_cmd.go:259–333` into a shared helper. The `--archived` flag and the "non-terminal-status entities" default are forward-compat with the proposed ADR-0004 (uniform archive convention); kind enumeration reads from `entity.AllKinds` so adding `KindFinding` later picks up automatically. G-061 names the verb-shape question; this milestone locks `aiwf list --kind <K>` (flag form, not positional plural) and applies that shape to the five `aiwf list contracts` references in `docs/pocv3/plans/contracts-plan.md` and the line in the contract skill.

## Acceptance criteria

### AC-1 — Core flag set works end-to-end

### AC-2 — JSON envelope: result is array of summary objects

### AC-3 — Default excludes terminal status; --archived includes them

### AC-4 — entity.IsTerminal(kind, status) helper added

### AC-5 — Closed-set completion wired for --kind and --status

### AC-6 — Shared filter helper extracted; status uses it

### AC-7 — Status text and JSON goldens unchanged after refactor

### AC-8 — contracts-plan and contract-skill drift fixed

### AC-9 — Verb-level integration test drives the dispatcher

## Constraints

- V1 flag set is locked: `--kind`, `--status`, `--parent`, `--archived`, `--format=text|json`, `--pretty`. No additional axes (`--actor`, `--since`, `--has-tdd`, `--ac-status`, `--has-findings`, `--format=md`) — defer until concrete friction earns them.
- Default semantic = "non-terminal-status entities", computed via `entity.IsTerminal(kind, status)`. Same predicate ADR-0004 will use to decide archive moves; designing with the ADR rather than around it means no UX break when ADR-0004 lands.
- `--archived` flag name is locked verbatim from ADR-0004 §"Display surfaces". Do not bikeshed.
- Closed-set completion for `--kind` and `--status` is wired through `cmd.RegisterFlagCompletionFunc`; the existing drift test in `cmd/aiwf/completion_drift_test.go` is satisfied without an opt-out entry.
- Refactor parity is non-negotiable: status text and JSON output are golden-tested. The shared helper lands first with parity tests against the current status output before `buildStatus` is rewritten to call it.
- Test-the-seam rule (per CLAUDE.md): a unit test of the helper alone is necessary but not sufficient. AC-9 requires a verb-level integration test that drives `run([]string{"list", ...})` and asserts the rendered output, not just the helper's return value.

## Design notes

- Verb shape: `aiwf list --kind <K>` (flag form). Decision rationale recorded inline in the epic; do not re-litigate. Positional plural (`aiwf list milestones`) is rejected to avoid per-kind pluralization rules and keep uniformity with the rest of aiwf's verb surface.
- `--parent` accepts any id whose value is referenced as `parent:` by some entity — e.g., `--parent E-13` returns milestones with `parent: E-13`; `--parent M-068` returns ACs (via the composite-id surface) when ACs become listable. V1 reach: epic → milestone, milestone → AC pending the AC-listability decision.
- JSON envelope `result` is `[]Summary` where `Summary = {id, kind, status, title, parent, path}`. No body — that's `aiwf show`. Keeps list cheap for downstream tools and AI consumption.
- No-args `aiwf list`: per-kind counts ("5 epics · 47 milestones · 12 ADRs · 14 gaps · 3 decisions · 1 contract"). Self-describing summary; not a route to "list everything."
- `entity.IsTerminal(kind, status)` is a pure closed-set switch on `entity.Kind` returning `bool`. ADR-0004 §Trigger names this helper by name; this milestone introduces it. One file edit in `internal/entity/transition.go`.
- Drift fix scope: every `aiwf list contracts` mention in `docs/pocv3/plans/contracts-plan.md` (lines 209, 425, 489, 593, 708) and `internal/skills/embedded/aiwf-contract/SKILL.md` line 33 becomes `aiwf list --kind contract`. Other contract-related verb mentions are unchanged.

## Surfaces touched

- `cmd/aiwf/list_cmd.go` (new)
- `cmd/aiwf/status_cmd.go` (refactor: extract filter slices into helper at `status_cmd.go:259–333`)
- `internal/entity/transition.go` (add `IsTerminal`)
- `internal/skills/embedded/aiwf-contract/SKILL.md` (line 33)
- `docs/pocv3/plans/contracts-plan.md` (5 line-level edits)

## Out of scope

- `aiwf-list` skill creation. M-073 owns that.
- Skills-coverage policy. M-074 owns that.
- Implementation of ADR-0003 (finding kind) or ADR-0004 (archive convention). The verb is forward-compatible; neither is a dependency.
- Any AC-listability surface (`aiwf list --kind ac` or composite-id listing). Decided at milestone start if friction earns it; out by default.

## Dependencies

- None on the aiwf side. Builds on existing `tree.Load`, the FSM definitions in `internal/entity/transition.go`, and the Cobra completion infrastructure established in E-14.

## Coverage notes

- `cmd/aiwf/list_cmd.go`: **91.5%** file-level coverage. Per-function:
  `newListCmd 100%`, `runListCmd 75%`, `isKnownKind 100%`, `buildListRows 100%`,
  `buildListCounts 100%`, `unionAllStatuses 100%`, `renderListCountsText 100%`,
  `pluralKindLabel 100%`, `renderListRowsText 90%`.
- `internal/tree/tree.go`'s `FilterByKindStatuses` (the AC-6 helper): **100%**.
- Uncovered lines in `runListCmd` are defensive paths that never fire under
  unit tests: `resolveRoot` failure (only on bad `--root` path), `tree.Load`
  failure (only on disk corruption), and `render.JSON` write errors to
  `os.Stdout` (only on stdout pipe failure). Same precedent as
  `runStatusCmd` in `status_cmd.go`. Marked as defensive rather than
  `//coverage:ignore`'d so a reviewer can see the shape.
- Test count by AC:
  AC-1 — `TestRun_List_CoreFlagsEndToEnd` (5 subtests),
  AC-2 — `TestRun_List_JSONResultIsArrayOfSummaryObjects`,
  AC-3 — `TestRun_List_ArchivedFlag` (3 subtests),
  AC-4 — exhaustive in `internal/entity/transition_test.go::TestIsTerminal_*`,
  AC-5 — `TestNewListCmd_CompletionWiring` (3 subtests),
  AC-6 — `TestSeam_ListAndStatusAgreeOnOpenGaps` + 5 helper subtests,
  AC-7 — `TestRenderStatus_Goldens` (text + JSON byte-equal),
  AC-8 — `TestNoReintroducedDeadVerbForms_ContractsAndSkill` (drift guard),
  AC-9 — same test as AC-1 (the verb-level seam test satisfies both).
- Plus entries to `TestEnvelopeSchemaConformance_AllJSONVerbs` (no-args
  `result` is object; filtered `result` is array) lock the JSON envelope
  shape against future drift.

## References

- E-20 epic spec (this milestone's parent).
- G-061 — names the unimplemented verb and the documentation drift this milestone resolves.
- ADR-0004 (proposed) `docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md` — names `aiwf list` and the `--archived` flag verbatim; default-semantic source.
- E-14 — Cobra and completion. Established the `RegisterFlagCompletionFunc` pattern and `cmd/aiwf/completion_drift_test.go` chokepoint.
- `cmd/aiwf/status_cmd.go:259–333` — the filter slices the shared helper extracts.
- `internal/tree/tree.go:178` — `tree.Load`'s walk; consumed unchanged.

---

## Work log

(filled during implementation)

## Decisions made during implementation

- **AC-4 (`entity.IsTerminal` helper) was already present.** Audit at start-milestone
  found the helper at `internal/entity/transition.go:93` with exhaustive property
  tests (`TestIsTerminal_ExhaustiveOverFSM`, `TestIsTerminal_TerminalSet`). Someone
  added it before this epic, presumably anticipating ADR-0004's needs. M-072's
  consumers (`buildListRows`, `buildListCounts`) wire it in; AC-4 closes by virtue
  of pre-existing work. The milestone spec was correct that the helper is
  needed; just incorrect that it didn't exist.
- **AC-7 ("goldens unchanged") raised the testing bar mid-milestone.** The
  pre-existing `TestRenderStatus*` tests use structural-substring assertions, not
  byte-equal goldens. They survived the AC-6 refactor — that's necessary parity
  evidence — but they do not satisfy the AC's word "goldens" literally. Wrap-pass
  added `TestRenderStatus_Goldens` with byte-equal `cmd/aiwf/testdata/status_*.golden`
  files locked to a deterministic `canonicalStatusReport` fixture. The
  structural-substring tests are kept; the goldens supplement, not replace.
- **AC-8 fixed 7 references in `contracts-plan.md`, not the spec's 5.** Lines 209,
  425, 489, 593, 708 were named; lines 424 and 654 also carried the dead form.
  Fixed all 7 — over-delivered to avoid leaving residual drift.
- **G-086 filed for out-of-scope drift.** `docs/pocv3/contracts.md` carries 5
  more `aiwf list contracts` references in speculative-future-flag context
  (`--drifted`, `--verified-status`, etc.). M-072 AC-8's named scope did not
  cover this file; filed as G-086 rather than a silent inclusion. The
  `TestNoReintroducedDeadVerbForms_ContractsAndSkill` drift guard's `sites`
  list is the natural extension point when G-086 closes.

## Validation

```
$ go test -race ./... 2>&1 | grep -E "FAIL|ok" | tail
ok  	github.com/23min/ai-workflow-v2/cmd/aiwf	167.290s
ok  	github.com/23min/ai-workflow-v2/internal/aiwfyaml	(cached)
ok  	github.com/23min/ai-workflow-v2/internal/entity	(cached)
ok  	github.com/23min/ai-workflow-v2/internal/policies	6.442s
ok  	github.com/23min/ai-workflow-v2/internal/render	(cached)
ok  	github.com/23min/ai-workflow-v2/internal/skills	3.788s
ok  	github.com/23min/ai-workflow-v2/internal/tree	4.071s
ok  	github.com/23min/ai-workflow-v2/internal/verb	34.480s
... (every package green)

$ golangci-lint run ./...
0 issues.

$ aiwf check 2>&1 | tail -1
2 findings (0 errors, 2 warnings)   # both pre-existing, not from this session

$ aiwf doctor 2>&1 | head -8
binary:    (devel) (working-tree build)
config:    ok
actor:     human/peter (from git config user.email)
skills:    ok (13 skills, byte-equal to embed)
ids:       ok (no collisions)
filesystem: case-insensitive
hook:      ok
pre-commit: ok
```

Smoke-tested against the live planning tree:
- `aiwf list` → `4 epics · 7 milestones · 4 ADRs · 28 gaps · 1 decision · 0 contracts`
- `aiwf list --kind milestone --status in_progress` → only M-072
- `aiwf list --parent E-20` → M-072, M-073, M-074
- `aiwf list --kind gap` → 28 rows; `--archived` widens to 83 rows.

## Deferrals

- (none)

## Reviewer notes

- **Read `tree.FilterByKindStatuses` first** (`internal/tree/tree.go`). It's the
  AC-6 chokepoint that lets `aiwf list --kind X --status Y` and `aiwf status`'s
  per-section slices route through one source of truth. The seam test
  `TestSeam_ListAndStatusAgreeOnOpenGaps` would catch any future re-introduction
  of parallel filter logic in either consumer.
- **The verb's V1 flag set is locked** to `--kind / --status / --parent /
  --archived / --format / --pretty`. Future axes (`--actor`, `--since`,
  `--has-tdd`, `--ac-status`, `--has-findings`, `--format=md`) are explicitly
  out-of-scope and earn their place when concrete friction demands them.
- **AC-1 ↔ AC-9 share one test.** `TestRun_List_CoreFlagsEndToEnd` is both
  the core-flag-set test and the verb-level seam test. AC-9's "drives the
  dispatcher" requirement is satisfied by the same code; both ACs cite it.
- **The earlier closure pass was reverted and remediated.** AC-7 originally
  closed against structural-substring proxies (the pre-existing
  `TestRenderStatus*` suite); audit revealed those don't match the AC's word
  "goldens" literally. AC-8 closed without a future-drift guard; the
  skill-coverage policy from M-074 doesn't catch sub-positional drift like
  `aiwf list contracts` (since `list` is now a real verb). Both gaps were
  closed in the wrap pass: byte-equal goldens for AC-7, scoped drift guard
  for AC-8. The history of those reverts is in the git log between the first
  and second `aiwf promote M-072 done` commits.
- **G-086 is a follow-up to M-072 AC-8.** Same drift class, third file
  (`docs/pocv3/contracts.md`). Out of M-072's named scope. Worth a quick
  pickup in a future small milestone.
