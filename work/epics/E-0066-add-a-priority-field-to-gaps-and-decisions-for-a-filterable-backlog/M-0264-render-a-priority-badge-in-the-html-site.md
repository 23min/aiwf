---
id: M-0264
title: Render a priority badge in the HTML site
status: done
parent: E-0066
depends_on:
    - M-0261
tdd: advisory
acs:
    - id: AC-1
      title: aiwf render emits a priority badge for each gap/decision with a value
      status: met
    - id: AC-2
      title: the badge is asserted structurally, not by substring, and human-verified
      status: met
---

# M-0264 — Render a priority badge in the HTML site

## Goal

Surface each gap's and decision's `priority` as a badge in the `aiwf render` HTML site, so the backlog's importance is visible at a glance in the rendered governance views.

## Context

The field exists (field milestone) and is set and filterable via the other surface milestones. This milestone adds the visual read path. The HTML renderer has no generic per-entity metadata/column abstraction to reuse, so the badge is bespoke template work — hence `tdd: advisory`: the deliverable is visual, human-verification is the real gate, and a structural HTML assertion is the mechanical backstop.

## Acceptance criteria

<!-- Seeded via `aiwf add ac`; each starts at tdd_phase: red. -->

### AC-1 — aiwf render emits a priority badge for each gap/decision with a value

### AC-2 — the badge is asserted structurally, not by substring, and human-verified

## Constraints

- The badge appears only for gaps and decisions carrying a value; an unset priority renders nothing (no empty badge).
- AC evidence is a **structural** assertion — parse the HTML and assert the badge inside the entity's section/attribute, not a substring grep (per the repo's "substring assertions are not structural assertions" rule).
- The render is verified by eye against the kernel's own planning tree before the milestone closes; the test does not stand in for the look.

## Design notes

- No column/badge abstraction exists — the `area` tag reaches templates via a bespoke `data-area` construct, not a reusable component. Keep the priority badge minimal and self-contained.

## Surfaces touched

- `internal/htmlrender/` — the template(s) and page-data plumbing for the badge.

## Out of scope

- Text/JSON surfaces (the read-surface milestone); writing the field (the write-surface milestone).
- Sort ordering — G-0420.

## Dependencies

- M-0261 — the field must exist first. Independent of the write and read surface milestones (fixtures set the field directly).

## References

- G-0078 — the ratified design decisions (HTML badge in scope, sort deferred).

## Work log

### AC-1 — aiwf render emits a priority badge for each gap/decision with a value

`Priority` field added to `htmlrender.KindIndexEntry` and `htmlrender.EntityRef`, populated from `entity.Entity.Priority` at all four resolver construction sites — the cmd-side `internal/cli/render.Resolver`'s `KindIndexData` and `entityRef` (production `aiwf render`), and the in-package `defaultResolver`'s `KindIndexData` and `EntityData` (the nil-resolver fallback and the package's own test baseline). `kind_index.tmpl` gained a `Priority` table column rendering `{{if .Priority}}<span class="priority priority-{{.Priority}}">{{.Priority}}</span>{{end}}` per row (gaps.html/decisions.html); `entity.tmpl` gained the same conditional badge next to the status pill on the individual gap/decision detail page. No kind-gate needed — the same "no separate carrying-kind check" mechanism established for the list/status filter (M-0263) applies here: a kind that never carries a priority (epic, milestone, ADR, contract) simply has an always-empty `Priority` field, so its row/page never renders the badge, the identical path an untagged gap/decision takes.

`style.css` gained one token per priority level (`--priority-urgent/high/medium/low`, light and dark) plus four `.priority-<level>` pill-color rules, reusing the existing generic `.status, .phase, .tdd, .scope-state, .policy` pill mechanism (now `.priority` joins that selector) rather than a new bespoke pill implementation — urgent/high sit on the same warm red/orange tokens as `tdd-required`/`tdd-advisory`, medium on the accent, low muted.

commit f1910741, corrected by 4d875c79 (see Independent pre-wrap review below) · tests: 3 new structural tests in `internal/cli/integration/priority_badge_html_test.go` (kind-index rows, entity detail page, non-carrying-kind never renders) plus two assertions added to the package-level `TestRender_FixtureTree_FilesAndLinks` (`internal/htmlrender/htmlrender_test.go`) exercising both `defaultResolver` construction sites directly (detail-page header and kind-index row) · `wf-vacuity` mutation probe: 5/5 mutants killed (kind-index conditional removed, entity-page conditional removed, cmd-side resolver hardcoded to a wrong level, in-package `EntityData` hardcoded to empty, in-package `KindIndexData` reverted to unpopulated) — each confirmed red under the mutation and restored byte-identical via captured content, never `git stash`/`checkout`.

### AC-2 — the badge is asserted structurally, not by substring, and human-verified

Every badge assertion is scoped by containment, not a page-wide substring match: `priorityRowSlice` isolates one entity's own `<tr>...</tr>` inside the `<table class="kind-index">` body (deliberately excluding the sidebar, which also carries `href="<id>.html"` links for epics — a page-wide search false-matched there before this scoping was added, caught while writing the non-carrying-kind test), and `entityHeaderSlice`/`entityHeaderBlock` isolate the detail page's header block between `</h1>` and the first body `<section>`. Both mirror the repo's established `areaGroupSlice`/`sectionByTab` containment-scoping idiom rather than introducing `golang.org/x/net/html` as a new dependency the repo doesn't otherwise carry.

Human verification: rendered the real `aiwf` binary against a git snapshot of this repo's own planning tree (`aiwf set-priority` tagging three real entities — a gap urgent, a gap high, a decision medium — then `aiwf render --format html`), and read the emitted `gaps.html`/`decisions.html`/detail-page/`assets/style.css` output directly — badge present with the right level on tagged rows and the detail-page header, absent (empty `<td></td>`, no empty span) on an untagged gap's row, and the new CSS tokens/rules present in the emitted stylesheet. No browser was available in this environment to confirm rendered color/layout beyond the raw markup and CSS.

commit f1910741 · same test suite as AC-1 (the two ACs share one implementation pass across the same set of files, mirroring M-0263's combined-AC-implementation precedent).

### Independent pre-wrap review

An independent fresh-context reviewer audited the full diff against nine load-bearing claims (Priority wiring at every construction site; conditional-not-empty rendering, measured against a real fixture render; structural vs. substring test scoping, including reproducing the sidebar false-match claim; the mutation-probe kills, by reproducing two of the four itself with a checksum-verified byte-identical restore; CSS token/rule completeness; the no-kind-gate-needed claim, verified against the write-path `CarriesOwnPriority` gates; branch/statement coverage via the diff-scoped `make coverage-gate`; commit trailer/provenance shape, cross-checked against M-0263's own untrailered implementation commit; and the honesty of the human-verification account). **Verdict: REQUEST-CHANGES**, with one blocking and one should-fix finding — both real, both fixed in-review:

- **Blocking:** `defaultResolver.KindIndexData` (`internal/htmlrender/default_resolver.go`) was missing the `Priority: e.Priority` line its own sibling `EntityData` and both cmd-side sites already carried — a genuine asymmetry the AC-1 Work log's "both resolver sites" claim glossed over. Not a production defect (production `aiwf render` always injects the cmd-side resolver; `defaultResolver` is the nil-`Data` fallback used only by this package's own tests), but real drift between four sites that should all agree. Fixed by adding the missing field, a new row-scoped assertion in `TestRender_FixtureTree_FilesAndLinks` exercising that exact site, and a confirming mutation probe (commit 4d875c79).
- **Should-fix (loose end):** the `aiwf-render` `SKILL.md` "Priority badge" documentation section had been drafted during the doc-lint sweep but was left uncommitted at review time. Committed alongside the fix above (4d875c79).

Two non-blocking observations, not requiring further changes: (a) the non-carrying-kind trust boundary — the template gates on `{{if .Priority}}` value, not on kind, so a hand-edited tree with an invalid `priority:` on a non-carrying entity could in principle render a badge — is deliberate and defensible, since such a tree already fails `aiwf check`'s `priority-not-applicable` rule; (b) the human-verification account is honest and appropriately hedged (explicitly discloses no browser was available in this environment to confirm rendered color/layout, only raw markup and CSS), reasonable given the badge reuses the already-shipping `--pill-color` pill machinery rather than introducing new visual risk.

## Decisions made during implementation

None — the design was fully pre-locked by G-0078's ratified decisions and this spec's own Design notes (reuse the existing generic pill mechanism; no new badge abstraction).

## Validation

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test -race -parallel 8 ./...` (`make test-race`, via `make check-fast`) — all packages pass, no flakes on the final sweep.
- `make lint` (full `golangci-lint` set) — 0 issues.
- `make coverage-gate` (diff-scoped statement-coverage audit + firing-fixture meta-gate) — clean.
- `aiwf check` — 0 error findings; 4 pre-existing/expected warnings (`acs-tdd-audit` × 2 — `tdd: advisory` intentionally left `tdd_phase` untracked, per the milestone's own stated tdd policy rationale; `epic-active-no-drafted-milestones` — M-0264 is the epic's last milestone, none remain drafted; `provenance-untrailered-scope-undefined` — no upstream configured for this unpushed branch).
- Manual branch-coverage audit + `wf-vacuity` mutation probe (covering both ACs together, since they share one implementation pass across the same files): 5/5 targeted mutations killed across the implementation and the review-fix — see the Work log entries above for the full list.
- Independent reviewer re-verified all nine load-bearing claims by measurement (build/vet/tests/a real fixture render/reproducing two mutation probes itself/the coverage gate) — REQUEST-CHANGES with one blocking and one should-fix finding, both fixed in-review (see "Independent pre-wrap review" above).
- Human verification against a real snapshot of this repo's own planning tree (see AC-2's Work log entry) — badge present/absent correctly on tagged/untagged real entities; no browser available to confirm rendered color/layout, only raw markup and CSS.

## Deferrals

- (none) — G-0420 (sort ordering by priority) and the text/JSON surfaces (the read-surface milestone, M-0263) were already scoped out at planning time, not discovered mid-implementation; both are named in this spec's own `## Out of scope` and `## References` sections.

## Reviewer notes

- **The independent review caught a real asymmetry the implementer's own narrative missed** (`defaultResolver.KindIndexData` lacked the `Priority` wiring its three sibling construction sites all had) — not a production defect, but a genuine drift the "both resolver sites" claim glossed over. Fixed in-review with a new row-scoped test and mutation probe; see "Independent pre-wrap review" above.
- **The non-carrying-kind trust boundary is deliberate, not an oversight**: the badge template gates on the `Priority` value being non-empty, not on entity kind, matching the same mechanism the list/status filter (M-0263) uses — a malformed hand-edited tree is `aiwf check`'s job to catch, not the renderer's.
- **No browser was available in this environment to visually confirm rendered color/layout** — verification was raw-markup-and-CSS-level (structural HTML assertions plus reading the emitted stylesheet against a real repo snapshot), not pixel-level. The badge reuses the already-shipping `--pill-color` pill mechanism (the same one `.status`/`.tdd`/`.policy` already use), so the visual-regression surface is small, but a human opening `gaps.html` in an actual browser remains the one verification step this session could not perform.

No `wf-rethink` unit applies: this milestone's surface (a `Priority` field threaded through two pre-existing structs and two pre-existing templates, one new CSS token family layered onto the existing generic pill mechanism) is a mechanical extension of the `--area` badge and list/status filter patterns already established in this codebase, not a new module boundary, core abstraction, or data model.

Given both review findings were mechanical (a one-line field-population gap and an uncommitted doc addition) rather than judgment-level, confirmation was mechanical too — re-running build/vet/the full touched-package test suite/lint/the specific mutation probe — rather than a second reviewer dispatch.
