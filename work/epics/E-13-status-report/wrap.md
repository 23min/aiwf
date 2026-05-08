# Epic wrap — E-13

**Date:** 2026-05-08 (closure date; the work itself shipped pre-migration — see *Shipping commits* below)
**Closed by:** human/peter
**Integration target:** poc/aiwf-v3
**Epic branch:** *(none — work landed pre-G-038-migration on the working branch directly)*
**Reconciliation:** kernel state reconciled with shipped reality during E-18 wrap-out review (2026-05-08); FSM walked normally — no `--force` used.

## Milestones delivered

- M-048 — Status report: cross-entity summaries + dashboard + time-window views (shipped scope: markdown renderer + Roadmap + Warnings + per-epic mermaid flowcharts; "time-window views" deferred at plan time per [status-report-plan.md](../../../docs/pocv3/plans/status-report-plan.md) §5)

## Shipping commits

The work landed before the G-038 dogfood migration brought entity tracking into the kernel-managed `work/` tree. None of these commits carry `aiwf-verb:` trailers because the trailer convention post-dates them. They are recoverable via `git log` filtered by the affected paths.

- `3a9d0ec docs(pocv3): plan a markdown status report with mermaid + roadmap + warnings` — the plan that drove the work.
- `39f667a feat(aiwf): aiwf status --format=md with mermaid roadmap, planned epics, and warnings list` — the implementation: `cmd/aiwf/status_cmd.go` (`renderStatusMarkdown`, `statusReport.PlannedEpics`, `writeStatusEpicMarkdown`, `Warnings []statusFinding`).
- `703c163 chore(repo): commit STATUS.md + pre-commit hook to keep it fresh` — installed the pre-commit hook that regenerates `STATUS.md` at the repo root via `aiwf status --format=md` on every commit.
- `e51fec6 docs(poc): mark status-report-plan as shipped` — updated the plan doc's frontmatter to record completion.

## Summary

`aiwf status` gained a third renderer (`--format=md`) that emits a self-contained markdown document with header, In-flight epics (each with a mermaid `flowchart LR` of its milestones), Roadmap (proposed epics + their planned milestones), Open decisions, Open gaps, Warnings (the actual list, not just a count), and Recent activity. The pre-commit hook regenerates `STATUS.md` at the repo root on every commit so the snapshot is always current in any client (GitHub web, VSCode, Obsidian, `glow`, `mdcat`) without a server, HTML, or GitHub Pages dependency. No new state — same `tree.Load` + `check.Run` data, third renderer next to `text` and `json`.

## ADRs ratified

- *(none)* — the design decisions ([`status-report-plan.md`](../../../docs/pocv3/plans/status-report-plan.md) §1, §5: extend `aiwf status` rather than add a new verb; markdown rather than HTML; no live updates / no GitHub Pages / no Gantt / no per-kind state-machine diagrams) are codified in the plan doc itself. Lifting them into ADR form would create a parallel record without adding explanatory weight.

## Decisions captured

- *(none as D-NNN entities)* — see *ADRs ratified*. The plan doc carries the rationale.

## Follow-ups carried forward

- **Time-window / filtering views were deferred at plan time** per [`status-report-plan.md`](../../../docs/pocv3/plans/status-report-plan.md) §5 ("Filtering flags (`--epic`, `--since`). The existing report is already small; if it ever gets too long, paginate later."). The milestone title's "time-window views" reflected the original ambition; the plan-doc decision narrowed scope before any code landed. Not filed as a gap because there's no current friction — `aiwf status` output is short enough that paginating would be premature optimization. If a consumer ever reports "the snapshot is too long," the gap can be filed at that point.

## Doc findings

`wf-doc-lint` not run as a scoped sweep — the wrap is retroactive, not a fresh implementation. The plan doc's "Status: shipped" header is the doc-side closure record and was written contemporaneously with the work.

## Handoff

What's ready: `aiwf status --format=md` is a stable surface that any consumer can pipe to a file, commit, render, or feed to a downstream tool. The pre-commit hook keeps repo-root `STATUS.md` honest without operator intervention.

What's deliberately left open: the filtering / time-window / dashboard-evolution axis (see *Follow-ups*). HTML output, `aiwf serve`, GitHub Pages publishing — all explicitly out of scope per the plan doc.

## Note on this wrap's lateness

This wrap was written 2026-05-08, after the work had been shipped and live for some time, as part of the E-18 close-out review. The pattern (kernel-tracked entity status drifting from shipped reality) is one the dogfood migration (G-038) was supposed to eliminate going forward; the residual cases are pre-migration milestones whose status was carried over without the corresponding state. M-048 was the last one in that category for E-13.
