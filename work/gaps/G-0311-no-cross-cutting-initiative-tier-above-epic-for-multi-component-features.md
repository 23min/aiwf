---
id: G-0311
title: No cross-cutting initiative tier above epic for multi-component features
status: open
---
## Problem

aiwf's work hierarchy tops out at the epic (Epic → Milestone → AC). Since the
area feature landed (E-0043 / ADR-0021), an epic is an *area-atom*: one epic
carries exactly one area, and milestones derive their area from the parent epic.
That makes the epic the natural unit of single-component work — but it leaves
**no first-class home for a cross-cutting capability that spans multiple
components/areas**.

A real example (downstream, a Plex/usenet stack): "subtitle support" = stack
(Bazarr) + picker-backend (client) + picker-frontend (indicator). Because
milestones inherit the epic's area, this *cannot* be one epic with three
area-tagged milestones — the kernel forces it into three separate epics wired by
`depends_on`, with no entity that names "the subtitle feature." The mental load
of tracking the capability across those epics — seams, contracts, sequencing — is
the friction that surfaced this gap.

## Evidence this is real and recurring, not speculative

Two larger sibling projects already invent ad-hoc structures for exactly this:

- **Liminara** uses *umbrella epics* — e.g. an umbrella epic with four peer-epic
  children. That is epic-contains-epic: containment a single (even area-relaxed)
  epic cannot express.
- **FlowTime** (a live aiwf-v3 consumer: ~26 epics, 70+ milestones) carries
  single epics spanning 5+ surfaces (e.g. a Matrix Engine rewrite across Rust
  core / API / Contracts / Sim / CLI / UI) and treats "Time Machine" as a
  first-class component every surface depends on.

A structural survey put ~40–50% of each project's active roadmap as cross-cutting
work. When independent projects hand-roll the same missing concept, the concept
is missing.

## A second, equally important role: aspirational capture

Beyond grouping committed work, the same node should capture an **aspirational
idea that has nothing planned under it yet** — a "we might want this" recorded
officially, then either thrown away before any planning or promoted into real
entities (epics, ADRs, decisions). Today that lives as either the messy "epic
with just a slug" hack or as prose buried under `docs/` explorations. A gap can
be the origin of such an idea ("a feature can come out of a gap"). A
derived-status grouping cannot represent this (derive-from-zero = nothing), which
is why the concept wants to be a real entity with its own lifecycle, not a
computed tag.

## Recommended direction (from the design conversation; not yet decided)

- **A seventh kind, tentatively `initiative`, one tier above epic.** Hierarchy
  becomes Initiative → Epic → Milestone → AC.
- **Name it `initiative`, not `feature`.** Every major tool (Azure DevOps, SAFe,
  Jira, Linear) places "Feature" *below* Epic; only "Initiative" (Jira Advanced
  Roadmaps, Linear) names the cross-cutting tier *above* the deliverable. This
  matters for the (speculative, unplanned) aspiration of a GitHub/DevOps adapter:
  the clean map is aiwf `initiative` → Jira/Linear Initiative / DevOps Epic; aiwf
  `epic` → Jira Epic / DevOps Feature; `milestone` → Story; `AC` → Task. "Feature
  above epic" would invert against every adapter target.
- **Lifecycle spanning the whole arc:** `proposed` (aspirational, possibly born
  from a gap, nothing under it) → `active` (epics/ADRs hang off it) → `done`
  (ship-gated: requires member epics terminal, mirroring the epic↔milestone gate)
  | a disposable terminal (`dropped`/`cancelled`) for ideas killed before
  planning.
- **Optional, additive, near-zero ceremony.** Epics work standalone exactly as
  today; an optional `initiative:` parent pointer on the epic is the only wiring.
  `aiwf add initiative "X"` is *less* ceremony than the slug-epic hack — this is
  the load-bearing constraint: it must not slow down single-component work.
- **Out of the area axis.** An initiative is inherently cross-area, so it carries
  no area; its member epics keep their single-area isolation (and per-epic
  `area-mistag` protection). The area model is untouched. This is the
  both-not-either win that `area: global` could not give (global buys grouping by
  surrendering isolation).
- **Progress derived, status coarse.** The rollup ("3 of 6 epics done") is
  computed; the only stored state is the coarse lifecycle, mechanically gated. No
  new parallel source of truth.

This retires three current hacks at once: umbrella-epics, slug-epics, and
ideas-buried-in-docs.

## Open questions for the eventual ADR / epic

- **Cardinality:** start single-valued (one initiative per epic → clean tree
  extension) vs many-to-many (an epic serves several initiatives → graph,
  heavier, trips the deliberate "no graph projection" deferral). Lean:
  single-valued; earn many-to-many on the third real case.
- **Does relaxing E-0043 (let milestones carry their own area) complement or
  compete with this?** The Liminara epics-of-epics evidence says a looser epic is
  *insufficient* (it can't hold peer epics), so the tier-above is needed
  regardless — but whether to *also* relax epic is a separate question.
- **Ship/integration gate:** is the cross-component end-to-end work a capstone
  epic under the initiative, or a property of the initiative itself?
- **Adapter shape** (speculative): if/when a GitHub/DevOps adapter is built,
  confirm the 4-tier mapping above holds.

## Status

Captured as the durable record of a design conversation (2026-06-29). Not yet
decided. Next step when picked up: a `proposed` ADR for "initiative as a tier
above epic / seventh kind" (a kernel-level decision per design-decisions #1, six
entity kinds), then a scoped epic.
