---
id: M-0175
title: Area grouping in status, render roadmap, and render html
status: in_progress
parent: E-0043
depends_on:
    - M-0171
tdd: required
acs:
    - id: AC-1
      title: a shared Partition helper groups items by area, complement always last
      status: met
      tdd_phase: done
    - id: AC-2
      title: status groups in-flight and planned epics by area when areas declared
      status: met
      tdd_phase: done
    - id: AC-3
      title: render roadmap groups its epics by area when areas declared
      status: open
      tdd_phase: done
    - id: AC-4
      title: render --format=html groups epic sections by area, asserted DOM-structurally
      status: open
      tdd_phase: red
    - id: AC-5
      title: empty declared areas suppressed; default complement always shown
      status: open
      tdd_phase: red
    - id: AC-6
      title: with no areas block, all three surfaces render exactly as today
      status: open
      tdd_phase: red
---
## Goal

Add per-area grouping to the three presentation surfaces — `status`, `render roadmap`, and `render --format=html` — so each renders a section per declared area, with untagged entities collected under the `aiwf.yaml: areas` `default:` display label.

## Context

M-0171 exposes each entity's effective area; the filter milestone narrows a view to one area. This milestone partitions a *whole* view into per-area sections — the readable payoff of the feature for a monorepo or a co-developed tool. It is the heaviest milestone (three surfaces) and may be split at `aiwfx-start-milestone` if the grouping helper + three renderers exceed a single focused cycle.

## Acceptance criteria

### AC-1 — a shared Partition helper groups items by area, complement always last

A generic `areagroup.Partition[T]` helper (new leaf package — the single source of the partition logic for all three surfaces) groups items by effective area: declared `aiwf.yaml: areas.members` in order, each carrying its items; the complement (items whose effective area is "" untagged OR a non-member value — the M-0172 `area-unknown` check is the mis-tag backstop) is always appended last under `areas.default`, or a built-in `Uncategorized` fallback when unset.

Evidence: `internal/areagroup/areagroup_test.go` — `TestPartition_Basic` (members order + complement-includes-undeclared), `TestPartition_DefaultLabelFallback` (fallback + configured label). 100% statement coverage on the helper.

### AC-2 — status groups in-flight and planned epics by area when areas declared

`aiwf status` (text and `--format=md`) partitions the In-flight and Roadmap **epic** sections per area when an `areas` block is declared; flat (today's output) when not. `StatusEpic` gained an additive `area` field (so JSON consumers can group too); the text/md renderers do the visual grouping via the shared helper. Recent activity, decisions, gaps, warnings, and health stay flat (cross-cutting). Grouping is skipped under `--area` (filter and grouping are alternative views).

Evidence: `TestRenderStatusText_GroupsByArea`, `TestRenderStatusMarkdown_GroupsByArea` (structural ordering: area heading before its epics); `TestRunRoadmap_GroupsByAreaViaDispatcher` covers the dispatcher seam for roadmap; the status dispatcher seam is exercised in M-0174's `TestRunStatus_AreaViaDispatcher` (filter) plus these grouping tests.

### AC-3 — render roadmap groups its epics by area when areas declared

`aiwf render roadmap` groups epics into per-area sections (area at `##`, epics demoted to `###`, milestone tables riding along) when an `areas` block exists; flat (epics at `##`) and **byte-identical to before** when not. `Render` stays the flat entry point; `RenderGrouped` and `Render` share one `render()`.

Evidence: `internal/roadmap/area_test.go` — `TestRenderGrouped_ByArea` (h2 area / h3 epic + ordering), `TestRenderGrouped_EmptyComplementShown`, `TestRenderGrouped_NoAreasMatchesFlat` (RenderGrouped(nil,"") == Render); `TestRunRoadmap_GroupsByAreaViaDispatcher` (the dispatcher's grouped + flat arms).

### AC-4 — render --format=html groups epic sections by area, asserted DOM-structurally

`aiwf render --format=html` groups in-flight epics into `<section class="area-group" data-area="…">` containers (resolver partitions into `StatusData.InFlightAreas`; the template branches on it via a shared `{{define "statusEpic"}}`). Asserted **structurally by containment**: the test scopes to each area-group container's sibling bounds and asserts the right epic is *inside* it and not in another area's container — catching a wrong-area or missing-container regression (verified by the reviewer's area-swap sever-probe). Flat DOM (no areas block) is unchanged.

Evidence: `internal/cli/integration/area_grouping_html_test.go` — `TestRun_RenderHTML_GroupsByArea` (per-area containment + complement), `TestRun_RenderHTML_FlatWithoutAreas` (no `area-group` containers without an areas block). Non-vacuity confirmed by severing the resolver grouping branch → test RED.

### AC-5 — empty declared areas suppressed; default complement always shown

Resolves the epic's Open Question 2: a declared area with zero entities is suppressed; the untagged complement is **always** rendered (with a "(none)" / "No epics in this area." placeholder when empty), so a grouped view always surfaces where un-triaged work lives. Baked into `areagroup.Partition` and honored by all three renderers.

Evidence: `TestPartition_SuppressesEmptyDeclared` + `TestPartition_AlwaysShowsComplement` (helper); `TestRenderStatus_EmptyComplementAlwaysShown` (status); `TestRenderGrouped_EmptyComplementShown` + `TestRenderGrouped_EmptyDeclaredSuppressed` (roadmap); the complement assertion in `TestRun_RenderHTML_GroupsByArea` (html).

### AC-6 — with no areas block, all three surfaces render exactly as today

Zero-migration: with no `areas` block declared, `status`, `render roadmap`, and `render --format=html` all render exactly as before. Each renderer guards on an empty member set and takes the pre-M-0175 flat path.

Evidence: `TestRenderStatusText_FlatWithoutAreas` (status); `TestRenderGrouped_NoAreasMatchesFlat` (roadmap byte-identity) + the flat arm of `TestRunRoadmap_GroupsByAreaViaDispatcher`; `TestRun_RenderHTML_FlatWithoutAreas` (html — no area-group containers).

## Constraints

- **One grouping helper, three consumers** — single source of truth for the partition logic; the three renderers format it, they don't each re-derive it.
- **Read-only.** No mutation.
- **Zero-migration parity** — untagged trees render byte-equivalently to today (modulo the optional default-complement heading when an `areas` block exists).

## Out of scope

- The `--area` filter (separate milestone; filter narrows, grouping partitions — they compose but ship separately).
- Grouping any surface beyond status / roadmap / html (e.g. `show`).

## Dependencies

- M-0171 — effective-area exposure on the loaded model.

## References

- [E-0043 epic](epic.md) · [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md)

## Work log

### AC-1 — Partition helper
new `internal/areagroup` leaf package; generic `Partition[T]` (members order, suppress-empty-declared, complement always last under default/fallback). · tests: 4 unit, 100%.

### AC-2 — status grouping
`StatusEpic.Area` (additive); `writeStatusEpicsText` / `writeStatusEpicsMarkdown` group via the helper when `AreaMembers` set; `Run` loads `cliutil.ConfiguredAreas` (skipped under `--area`). · tests: text + md ordering + dispatcher.

### AC-3 — roadmap grouping
`Render` → shared `render()`; new `RenderGrouped` (area h2 / epic h3); `writeEpicSection(level)`. Dispatcher chooses grouped vs flat by `ConfiguredAreas`. · tests: unit + dispatcher + flat byte-identity.

### AC-4 — html grouping
`StatusData.InFlightAreas` + `StatusAreaView`; resolver partitions; `status.tmpl` branches via `{{define "statusEpic"}}`; `area-group`/`data-area` DOM containers. · tests: containment-structural + flat.

### AC-5 — empty-area policy
suppress empty declared, always show complement — in the helper, exercised in all three renderers. · tests: helper + per-surface.

### AC-6 — zero-migration
each renderer guards on empty member set → flat path. · tests: per-surface flat (roadmap byte-identical).

The phase timeline is in `aiwf history M-0175/AC-N`; not duplicated here.

## Decisions made during implementation

- **One milestone, not split.** The spec invited a split; the work is cohesive (one generic helper + three thin consumers, ~400 LOC), so it stayed one milestone with AC-by-AC TDD. Confirmed with the user.
- **AC-5 = suppress empty declared; always show the default complement** (resolves epic Open Question 2). Confirmed with the user. Rationale: an unused declared area is noise; the untagged complement is always worth surfacing (un-triaged work). The complement folds in undeclared-but-tagged values too — the M-0172 `area-unknown` check is the mis-tag backstop, so grouping need not invent a third bucket.
- **Grouping target = epic sections** across all three surfaces; decisions/gaps/recent/health stay flat. Epics are the workstream unit; grouping the auxiliary sections multiplies output for little value.
- **Automatic when an `areas` block exists** (per spec), not flag-gated; flat otherwise (zero-migration). `--area` (filter) suppresses grouping.
- **AC-4 asserted by containment, not a full HTML parse.** The codebase deliberately carries no HTML-parse dependency (`golang.org/x/net/html` is absent; existing htmlrender tests use scoped section-slicing). The containment assertion scopes to each `area-group` container's sibling bounds and verifies the epic is *inside the correct one* — CLAUDE.md's sanctioned "or an equivalent" structural assertion. The reviewer confirmed it catches wrong-container regressions via an area-swap sever-probe.

## Validation

- `go test ./internal/...` — all packages pass.
- `golangci-lint run ./internal/...` — 0 issues.
- `go test ./internal/policies/` — pass (added `internal/areagroup` to the layering-tier map as a tier-7 leaf; canonical-width fixture ids in `areagroup_test.go` for the narrow-id policy).
- Branch-coverage: every changed production line exercised; new functions ~100%. Determinism verified (byte-identical across two renders).
- Independent fresh-context review: APPROVE — every AC verified by measurement (binary exercised on all three surfaces, grouping severed to confirm tests go RED, flat byte-identity proven against a stashed pre-M-0175 binary, merged coverage). Three non-blocking findings (area-heading color-independence, html complement-slice bound, suppressed-area assertion anchoring) fixed inline.

## Deferrals

- None. Grouping `show` and grouping the decisions/gaps sections are explicitly out of scope (epic design), not deferrals.

## Reviewer notes

- The area-grouping partition is one generic `areagroup.Partition[T]` consumed by status (`StatusEpic`), roadmap (`*entity.Entity`), and html (`StatusEpicView`) — generics are the right DRY here (three concrete item types, one ordering/empty-policy). The complement deliberately includes undeclared-tagged items (not a separate group); the M-0172 check is the mis-tag backstop.
- The status text area heading carries a `▸ ` marker so the grouping is legible in piped / no-color output (the G-0080 "glyph is content, not style" principle); markdown uses a `**label**` divider to keep the h2→h3 epic hierarchy; html uses a `data-area` container.
- The epic E-0043 open-question table reconciles at the epic wrap: Open Question 1 (gap derive-on-omit) was decided in M-0173; Open Question 2 (empty-area handling) is decided here (suppress empty declared, always show complement).

