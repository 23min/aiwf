---
id: G-0595
title: Live records carry claims the kernel contradicts, and no check sees them
status: open
priority: high
---
## What's missing

Every live planning record was measured against the kernel and most carry
claims the code contradicts. The inventory, with the command behind each
finding, is
[`docs/initiatives/entity-truth-audit.md`](../../docs/initiatives/entity-truth-audit.md).
That document is the work list; this gap tracks absorbing it.

A second pass re-measured the gaps `TODO.md` orders, at a higher bar and with an
independent auditor per batch:
[`docs/initiatives/gap-truth-audit.md`](../../docs/initiatives/gap-truth-audit.md),
whose evidence file carries the command behind every high-severity finding. Same
tier, same absorption work, so it is a second work list under this gap and not a
second tracker. What it adds is a cause assigned to each finding and a measured
ceiling on how much of the corpus any mechanism can reach — which is what makes
most of the work here body edits rather than new checks.

The audit covered every open gap, every non-terminal ADR and decision, and the
Normative doc tier, at the tree it names. Two thirds of the subjects carry at
least one finding. Nothing here is a parse failure: `aiwf check` reports the
tree clean, because every defect is semantic — prose that resolves, links, and
is false.

The entity tier is largely sound in *subject* and unsound in *support*. Almost
every open gap is still a real gap; what has decayed is the scaffolding around
it — line numbers that moved, counts that drifted, cited symbols that were
renamed, and premises a later decision overtook. Four gaps whose defects had
actually shipped are already promoted, so what remains is editing bodies rather
than closing them.

Six causes account for most of the findings, and each is worth fixing at the
source rather than one record at a time:

- **A retired verb is still cited as live**, across gaps, decisions, ADRs and
  docs, several edited after ADR-0039 retired it. `internal/policies/m0290_retirement_surface_test.go`
  scopes `docs/adr/` out of its scan, so the one mechanical rule for this class
  cannot see the tier carrying most of it.
- **Three ADR pairs reverse each other with no supersession pointer** in either
  direction: ADR-0016 over ADR-0014 §2, ADR-0032 over ADR-0014 §3, and ADR-0032
  narrowing ADR-0027. `adr-supersession-mutual` polices declared supersession
  only; reversal-in-fact has no oracle. ADR-0007 is the model — a See-also
  banner naming what was retired and what survives.
- **Records overtaken by later work and never revisited.** D-0028 is the sharp
  case: it is `accepted` and the kernel does the opposite.
- **Counts that no longer re-derive**, the failure the shipped guidance names.
  `docs/design/growth.md` is the counter-example worth copying: its baseline
  table still reproduces exactly, because every figure names the script behind
  it.
- **Worked examples that no longer run.** `docs/workflows.md` omits several
  preconditions the kernel now enforces; `docs/skill-author-guide.md` teaches
  consumers to author skills in the `aiwfx-` namespace that `aiwf init` writes
  into their `.gitignore`.
- **Records at `proposed` while already load-bearing** — D-0053, D-0056,
  D-0057 — against this repo's own "decision is decision" rule.

Two mechanical follow-ons are cheap and would have caught real findings here:
widening the retirement-surface scan to `docs/adr/`, and a citation checker for
`file:line` references, which is the largest reference class in the inventory.

Out of scope, because it is already tracked: **fixing the Normative doc tier is
[G-0560](G-0560-the-normative-doc-tree-has-drifted-from-the-kernel-it-documents.md)**.
That gap delegates to a 2026-08-06 inventory covering the same tier; this audit
re-derived those entries rather than inheriting them, so the newer document is
the better evidence for the doc-tier work while G-0560 remains its tracker.
This gap owns the entity tier and the cross-cutting causes above.

## Why it matters

An LLM reads the open-gap set, the accepted ADRs and the Normative docs as
current truth and plans against them. A record that is silently false is worse
than an absent one: it produces confident wrong work, and the tree reports
clean while it does. The concrete cost is already visible — one gap was filed
from a stale snapshot a month after its defect was fixed, two decisions send a
reader to file work that shipped, and an accepted decision states a rule the
kernel inverts.

The inventory ages by construction, and that is the point: each entry is either
fixed and deleted, or still true. Left unabsorbed it becomes another dated
document making claims about a tree that has moved on — the failure it exists
to record.
