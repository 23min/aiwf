---
id: G-0443
title: Design docs cite stale cmd/aiwf/*_cmd.go paths (CLI now internal/cli)
status: addressed
priority: high
discovered_in: M-0274
addressed_by_commit:
    - 03127efe
---
## What's missing

The normative design docs' source/provenance references to CLI verb
implementations still point at the pre-restructure `cmd/aiwf/<verb>_cmd.go`
layout. M-0116 (under E-0032) moved every single-command verb into
`internal/cli/<verb>/<verb>.go`, leaving only `main.go` under `cmd/aiwf/`, but
the doc references were never swept. In active/normative docs:

- `docs/design/legal-workflows-audit.md` — the rule catalog's **Source**
  column cites `cmd/aiwf/*_cmd.go` for 17 distinct verb files across 35 rows
  (`verbs_cmd.go` ×15, plus `archive_cmd.go`, `authorize_cmd.go`,
  `contract_cmd.go`, `import_cmd.go`, `init_cmd.go`, `list_cmd.go`,
  `milestone_cmd.go`, `render_cmd.go`, `retitle_cmd.go`, `rewidth_cmd.go`,
  `schema_cmd.go`, `template_cmd.go`, `update_cmd.go`, `upgrade_cmd.go`,
  `whoami_cmd.go`, `completion_drift_test.go`).
- `docs/design/id-allocation.md` — cites `cmd/aiwf/admin_cmd.go`.
- `docs/design/design-lessons.md` — cites `cmd/aiwf/render_cmd.go` as a
  current location.

The rules' behavior/statement columns are accurate; only the file provenance
is stale. Out of scope: the `docs/archive/pocv3/*` references (Archival tier,
frozen per ADR-0004 — cross-references deliberately unmaintained), and
legitimate historical "moved from `cmd/aiwf/X`" statements (e.g. in
`verb-layer-cleanup.md`), which correctly describe the move.

## Why it matters

The audit catalog's Source column is a provenance map: rule → enforcing file.
A reader (human or LLM) following it to the code hits a nonexistent path for
every CLI-verb rule, defeating the column's purpose. The catalog is
Normative-tier — expected to stay in lockstep with code — so a dead file
attribution across 35 rows is real drift, not a cosmetic nit. Nothing
mechanically checks that the Source column's paths resolve: the catalog's
structural test pins column/id/count shape, not path existence, so this
drifted silently through M-0116 and would drift again on the next verb move.

## Suggested approach

Per-verb re-attribution — each verb moved to a *different*
`internal/cli/<verb>/<verb>.go` path, so it is not a uniform find-replace —
plus the two stragglers in `id-allocation.md` and `design-lessons.md`. Add a
targeted guard to the existing catalog structural test: assert that
Source-column values that are Go file paths resolve on disk, skipping the
column's non-file values (ADR ids, `FSM`/`Verb`, doc names, bare filenames).
Scoping the check to that one unambiguous column keeps it from false-positiving
on prose, historical references, or archival docs — which is why a general
doc-path linter is the wrong tool here.
