---
id: G-0345
title: Skills cite rich-template paths that don't resolve in consumer repos
status: addressed
addressed_by_commit:
    - 04c3db35
---
## What's missing

Every `aiwf add <kind>` writes only the minimal body skeleton — a new entity is
born with empty section bodies (the create-then-fill two-step; `entity-body-empty`
is the finding that nags until the prose lands). Four kinds have a *rich* body
template to fill from — ADR and decision (via `aiwfx-record-decision`) and epic
and milestone (via `aiwfx-plan-epic` / `aiwfx-plan-milestones`) — but every one of
those skills cites its template by an authoring-relative path ("this plugin's
`templates/adr.md`"). In a consumer repo that template is materialized at
`.claude/templates/<kind>.md`, which is gitignored and therefore invisible to a
scan of the committed tree. No skill (a) names the self-heal — `aiwf update`
re-materializes the templates; (b) flags that the rich template, not the
`aiwf add` skeleton, is what carries the `# <id> — <title>` H1 and the
date / decided-by header; or (c), in the `aiwf-add` skill, tells the author where
the per-kind template lives at all. The gap and contract kinds have no rich
template — for them the skeleton is the whole shape.

## Why it matters

A downstream AI assistant set out to author an ADR, went looking for the template,
couldn't find it ("the template dir isn't shipped"), and fell back to hand-copying
an existing ADR's format. That path drifts from the canonical template, silently
drops the H1 and the date header (both observed missing on the resulting ADR), and
courts an untrailered / hand-picked-id commit — precisely the silent-correctness
class aiwf exists to close. The same authoring-relative-path defect sits in the
epic and milestone planning skills, so the failure generalizes across every kind
that ships a rich template. The fix is discoverability, not mechanism: the
born-empty skeleton stays minimal by design (fattening the scaffold to embed the
rich template was considered and rejected — it duplicates the template into the
kernel and crosses the ritual/kernel layering boundary). Instead, point every
rich-template skill at the materialized `.claude/templates/<kind>.md` path, add the
`aiwf update` self-heal, name the skeleton-vs-template distinction, and give
`aiwf-add` a per-kind template-location note. Scope is ritual + guidance only; a
referencing structural test under `internal/policies/` pins each edited skill.
