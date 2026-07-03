---
title: Adoption-maturity advisor
status: captured
date: 2026-07-03
---

# Adoption-maturity advisor

## Classifier note

This is an initiative document. `initiative` is not yet an official aiwf entity
kind (see `G-0311`), so this file lives under `docs/initiatives/` as an umbrella
capture alongside `agent-agnostic-execution-topology.md`.

This is not an ADR: it does not ratify a decision.
This is not research: it does not primarily survey or prove a thesis.
This is not an exploration: the need is not speculative — aiwf is already large
enough that operators demonstrably under-use it.
This is not a plan: it intentionally avoids epics, milestones, sequencing, and
timeframes.

The purpose is to preserve the shape of a feature-sized concern — a surface that
tells an operator "given the shape of *your* repo, here is aiwf value you are
leaving on the table" — so future epics can be drafted from a coherent center
instead of rediscovering the edges.

## Initiative statement

aiwf should help an operator discover which of its own capabilities a repo would
benefit from adopting, without becoming a nag or a project-management advisor.

The core posture:

> aiwf informs which of its capabilities fit this repo and why; it never mandates
> adoption, and it never opines on the project beyond its own surface.

The practical target is a **fourth information surface**. aiwf already answers
three questions well, and none of them is the one an operator staring at a big
toolbox is actually asking:

```text
aiwf check     Is my planning state internally consistent?     drift / error  -> fix now
aiwf doctor    Is the tooling wired correctly?                  installed / broken -> fix now
--help / skills  What can each verb do?                         reference
(missing)      Given the shape of MY repo, what latent value    opportunity -> consider
               am I leaving on the table?
```

The discoverability principle in `CLAUDE.md` — "kernel functionality must be
AI-discoverable" — guarantees every capability is *reachable* via `--help`,
skills, and docs. But reachability is not the same as **awareness of what is
relevant right now**. Contracts are reachable through `aiwf contract --help`;
nothing today tells an operator with six JSON schemas and zero `contract`
entities "this is the boundary you would pin first." That is the gap this
initiative names.

It is deliberately *not* onboarding (first-run setup, which `aiwf init` owns) and
*not* help (capability reference, which `--help` and skills own). It is an
adoption-maturity advisor: the surface that closes the distance between "aiwf is
installed and I use four verbs" and "aiwf is doing everything it usefully could
for this repo."

## Mission fit

aiwf's mission is to keep durable structural state honest inside the repo:
planning state, references, status transitions, provenance, audit history, and
mechanical validation. It is explicitly not a project-management tool and not a
workflow engine (`docs/research/07-state-not-workflow.md`,
`docs/research/12-operating-model-agnostic.md`).

An adoption advisor fits that mission only if it stays **meta**. Its scope is
aiwf's *own* adoption surface — which aiwf capabilities measurably fit this repo —
not the health of the user's project. It recommends turning on `contract`
verification, `areas`, per-milestone `tdd`, `archive` sweeps, or a `depends_on`
edge. It does not recommend how to structure the user's release process, grade
their code, or manage their backlog. Code quality already has a home
(`code-health` / `wf-codebase-health`); planning-tree *direction* already has one
(`aiwfx-whiteboard`, "what should I work on next"). This advisor answers a
distinct third question: "which aiwf machinery should be switched on."

Held to that boundary, the advisor is squarely mission-aligned: it helps an
operator realize the value aiwf already commits to, expressed as inspectable,
mechanically-grounded findings rather than a wall of documentation.

## Philosophical anchors

- `CLAUDE.md`: **framework correctness must not depend on LLM behavior.** A
  recommendation that must never be *missed* (you have 47 terminal entities and
  no configured sweep) has to be mechanical and complete; an LLM prose survey
  that "usually notices" is not a guarantee. This forces the mechanical-core /
  judgment-wrapper split below.
- `CLAUDE.md`: **KISS / YAGNI.** aiwf is already "big." The advisor must not add a
  speculative rule engine before the rules have earned their place. Skill first;
  mechanize the rules that prove they deserve determinism.
- `CLAUDE.md`: **errors are findings, not parse failures.** aiwf already has an
  *advisory* finding class (`archive-sweep-pending`) that says "this could be
  better" without blocking. Adoption recommendations are the same shape, on a new
  axis — opportunity rather than drift.
- `G-0199` (finding hints must name the exact remediation command): every
  recommendation must name its exact command (`aiwf contract bind <C-id> ...`,
  `aiwf upgrade`), never just describe the fix. The finding → hint → command
  chain is the highest-leverage discoverability surface; the advisor lives or
  dies on it.
- The `aiwf acknowledge` verb already models "I judged this intentional — record a
  sovereign, reasoned, git-tracked exemption." Declining a recommendation should
  reuse that machinery, not invent a parallel one.

The doctrine this sharpens into:

> aiwf measures adoption opportunity mechanically and completely; it frames that
> opportunity with judgment; and it lets an operator decline any recommendation
> permanently, so advice never decays into noise.

## Mechanical measurement, judgment framing

The central design decision. Two extremes are both wrong:

- **Pure skill** (an LLM surveys the repo and suggests things). Cheap and
  flexible, but non-deterministic, untestable, silently incomplete (it *might*
  notice the archive opportunity), and able to hallucinate advice. It violates
  "a guarantee that depends on the LLM remembering is not a guarantee."
- **Pure verb** (`aiwf advise` emits everything). Testable and complete, but it
  can only recommend what it can *mechanically measure* — it cannot reason about
  whether contracts fit an architecture (that needs reading code), and it adds
  kernel surface to a framework already criticized as large.

The resolution is the split aiwf uses everywhere else (mechanical `check` +
advisory wrap-review; mechanical guidance-anchors + human audience-judgment):

- **Facts stay mechanical and complete.** "47 terminal entities, no
  `archive.sweep_threshold`." "Binary three minors behind latest." "Six files
  match `*.schema.json`, zero `contract` entities." "12
  `provenance-untrailered-entity-commit` warnings this month." These are
  measurable, so they belong to a deterministic core that cannot miss them. The
  shape already exists: `doctor --check-latest` reaches the module proxy for
  version currency, and `doctor --write-health` emits machine-readable health.
- **Framing stays judgment.** *Whether* a measured schema boundary is worth a
  contract, *whether* a logic-heavy milestone should have been `tdd: required`,
  how to sequence the suggestions, how to phrase them for a human. That needs the
  actual code read, and cannot be a rule.

Neither layer over-claims: the numbers are trustworthy and exhaustive; the advice
around them is explicitly a judgment the operator can reject.

## The middle path: skill first, mechanize what earns it

A full recommendation engine built up-front would repeat the `worktree create`
mistake the sibling initiative warns against — owning a large surface before
experience says which parts are durable. The smaller mission-aligned first step:

1. **Ship a skill** (`aiwf-advise`) that composes surfaces that already exist
   read-only — `aiwf doctor --check-latest`, the `aiwf.yaml` knobs, `aiwf list`,
   `aiwf check`, `aiwf history` — plus a survey of the actual code. This adds zero
   kernel surface, is fully reversible, and immediately tells us which
   recommendations land.
2. **Graduate the winners.** The class-1 and class-2 recommendations that prove
   high-value and want to be *complete* (never missed) move into a mechanical
   advisory-finding class — a new `check` axis, or a `doctor` section — modeled on
   `archive-sweep-pending`. This dovetails with `G-0289` (doctor surfaces a
   planning-tree check-health summary) and needs `G-0070` (doctor
   `--format=json`) so agents can consume the output.

"Skill first; abstract on the third" is the repo's own ethos. The build order
falls out of the catalog: classes 1–2 are mechanical and graduate; classes 3–4
are where an LLM skill genuinely earns its keep.

## Catalog of recommendations, by signal class

Grounded in the real config knobs (`archive.sweep_threshold`, `areas.*`,
`tdd.*`, `guidance.wire_claudemd`, `tree.strict`, `status_md.auto_update`,
`entities.title_max_length`, `html.*`) and verbs. Organized by signal type,
which also reveals the engine's shape and its build order.

### Class 1 — currency and wiring *(mechanical; mostly latent in `doctor` already)*

- Binary behind latest published → `aiwf upgrade` (`doctor --check-latest`
  already knows this).
- Materialized skills / rituals drift from the embed, or the guidance import is
  missing from `CLAUDE.md` → `aiwf update`.
- Statusline not installed → `aiwf init --statusline`.
- Legacy id widths present → `aiwf rewidth --apply` (one idempotent commit).

### Class 2 — config knobs at default the repo's shape argues against *(mechanical)*

- Terminal entities accumulating with no `archive.sweep_threshold` → set a
  threshold or sweep. (The `archive-sweep-pending` advisory finding is the exact
  precedent to replicate.)
- Clear module/directory structure but `areas.members` empty → adopt areas for
  per-workstream grouping; or areas defined with `paths:` but no `coverage_roots`
  wired; or mistags accumulating while `areas.required` is off.
- `status_md.auto_update` off → a live `STATUS.md` is one flag away.
- `ROADMAP.md` older than the latest entity change → `aiwf render roadmap
  --write`; static site never rendered → publish governance views.
- Grandfathered pre-cap titles present, or `entities.title_max_length` unset →
  `aiwf retitle` cleanup.

### Class 3 — opt-in features the repo would benefit from *(judgment; the skill's real value-add)*

- **Contracts** (the flagship): schema files, wire formats, or golden fixtures
  exist but zero `contract` entities → bind one with a validator recipe. The
  *signal* (schemas exist, no contracts) is measurable; the *judgment* (is this
  boundary worth pinning) is not.
- **TDD**: logic-heavy milestones shipping at `tdd: none` → consider `tdd:
  required` for the invariant-bearing ones, or `tdd.require_test_metrics` to
  capture counts.
- **ADRs**: long-lived architectural choices visible in history with no
  ADR/`D-NNNN` entity → record them.
- **Authorize / provenance**: non-human actors committing without an `authorize`
  scope → open one for cleaner delegation trailers.
- **Gaps**: TODO/FIXME density in code with no matching gap entities → file gaps
  so they are tracked in the planning tree.
- **`depends_on`**: milestones that plainly sequence but carry no edges → make the
  sequencing explicit.

### Class 4 — reality-vs-state hygiene *(advisory, opportunity-framed)*

- Long-open `proposed` ADRs (decisions in limbo) → ratify or reject.
- `in_progress` milestones with no recent history (status has drifted from
  reality).
- Recurring `provenance-untrailered-entity-commit` warnings → edits bypassing
  verbs; route them through `edit-body` / `retitle` to realize `aiwf history`.
- Epics closing with no wrap artifact → the wrap rituals are not being used; ADR
  candidates never harvested.

## Anti-nag design: recommendations must be declinable

The classic linter-nag failure mode is a surface that fires forever, including on
things the operator has deliberately declined, until everyone stops reading it.
Two constraints prevent that, and both are non-negotiable:

1. **Every recommendation is declinable, and the survey respects prior declines.**
   "I considered contracts and chose not to for this repo" must suppress the
   recommendation durably. aiwf already has the exact primitive: `aiwf
   acknowledge` records a sovereign, reasoned, git-tracked "I judged this
   intentional." Recommendations should hang off the same machinery rather than a
   parallel snooze file.
2. **Tone is "consider X because you have Y," never "you are doing it wrong."**
   `tree.strict` is right for some repos and deliberately wrong for others. Each
   recommendation carries its tradeoff, not a verdict. The advisor's severity
   band is strictly *opportunity*; it must never borrow the imperative tone of a
   `check` error or a broken-hook `doctor` line.

## Naming and placement direction

Working names: the skill is `aiwf-advise`; a future graduated verb is `aiwf
advise`. `recommend` reads as slightly more prescriptive than the posture wants;
`advise` better fits "inform, do not mandate." This is not settled.

Placement relative to `doctor` is a genuine fork:

- **Standalone surface** keeps severity models clean. `doctor` mixes "your hook
  is broken — fix now" with installation truth; folding "consider contracts —
  aspirational" into the same report muddies the actionability signal. A separate
  entry point keeps *opportunity* from being read as *breakage*.
- **A `doctor` mode / section** keeps kernel surface smaller and reuses the
  health entry point operators already run. `doctor` already carries
  `--check-latest` and `--write-health`, so class-1 currency signals partly live
  there today.

The lean matches the middle path: start as a **skill** (no new verb at all),
compose existing read-only surfaces, and only if graduation is warranted decide
between a `doctor` section and a standalone `aiwf advise` — informed by which
recommendations proved they wanted determinism.

Boundary against neighbours, stated so they do not bleed:

- `aiwfx-whiteboard` — "what should I *work on* next" over the planning tree
  (which entity). The advisor answers "which aiwf *capability* should I turn on."
- `code-health` / `wf-codebase-health` — the user's code quality. The advisor
  never enters that domain.
- `aiwf doctor` — installation and drift health. The advisor extends past class-1
  currency into config, feature, and hygiene opportunity.
- `aiwf check` — conformance drift (blocking / advisory findings). The advisor's
  class-4 hygiene overlaps but is framed as opportunity, never as a gate.

## Existing aiwf surfaces this touches

### Config and verbs

- `aiwf doctor` (`--check-latest`, `--write-health`) is the closest existing
  surface and the natural graduation host for class-1 signals.
- `aiwf check`'s advisory findings (`archive-sweep-pending`) are the shape a
  mechanical recommendation class should copy.
- The `aiwf.yaml` knobs enumerated in the catalog are the class-2 measurement
  inputs.
- `aiwf acknowledge` is the reuse target for declining a recommendation.
- `aiwf contract`, `aiwf authorize`, per-milestone `tdd`, and `areas` are the
  class-3 features the advisor would surface.

### Open gaps

- `G-0289` doctor surfaces a planning-tree check-health summary: the adjacent
  step that pulls `check`'s finding count into `doctor`; the advisor extends the
  same "one entry point answers the whole health question" instinct.
- `G-0199` finding hints must name the exact remediation command: the bar every
  recommendation must clear — name the command, not just the fix.
- `G-0070` doctor lacks a `--format=json` envelope: prerequisite for
  agent-readable advisor output if class-1/2 recommendations graduate into
  `doctor`.
- `G-0311` no cross-cutting initiative tier above epic: the reason this capture
  lives under `docs/initiatives/` rather than as a first-class entity.

### Current skills

- `aiwfx-whiteboard` is the read-only synthesis precedent — it loads tree state
  via `aiwf status` / `list` / `show` / `history` and writes a gitignored cache.
  `aiwf-advise` would follow the same read-only, no-mutation, no-commit shape.

## Risks and boundaries

### Risk: it becomes a project-management or code-quality advisor

Avoid by holding the meta boundary: the advisor recommends only aiwf's *own*
capabilities. The moment it grades the user's code or opines on their release
cadence, it has left the mission. Code quality is `wf-codebase-health`'s domain;
this surface must not compete.

### Risk: recommendation fatigue

Avoid with the declinable-recommendation design. A recommendation the operator
has acknowledged-as-declined must not resurface. Without durable declines the
surface rots into noise within weeks.

### Risk: an LLM-only advisor is silently incomplete

Avoid by keeping completeness-bearing recommendations mechanical. Anything that
must never be missed measures deterministically; the skill's judgment layer is
additive, never the sole guarantee.

### Risk: recommending something worse for this repo

Avoid by framing every item as an opportunity with its tradeoff, never a verdict.
`tree.strict`, sibling worktrees, and mandatory areas are all legitimately
declined by some repos. "Consider X because Y" leaves the choice with the
operator.

### Risk: over-building the catalog before it earns its place

Avoid by shipping the composing skill first and graduating rules one at a time.
No standalone verb, no finding class, until a specific recommendation has proven
both high-value and worth making complete.

## Open design questions

These are intentionally not answered here.

- `advise` or `recommend`? Skill-only, a `doctor` section, or a standalone verb?
- How is a decline recorded — reuse `aiwf acknowledge`, or a distinct mechanism —
  and what is its target key so the survey can suppress precisely one
  recommendation?
- Which specific class-1/class-2 recommendations are worth graduating into a
  mechanical advisory-finding class, and where do they live (`check` vs
  `doctor`)?
- What is the exact structured shape of an advisor recommendation (id, severity
  band, measured signal, suggested command, tradeoff note)?
- How does the advisor detect class-3 signals (schema files, TODO density,
  architectural decisions in history) without reimplementing a code scanner —
  how much is measured vs. left to the skill's judgment?
- Should the advisor rank/sequence recommendations, or only list them?
- How does it stay quiet on a mature repo — what is the "nothing to advise, you
  are using aiwf well" terminal state?

## Desired future property

A human or AI agent entering an aiwf repo should be able to ask:

```bash
aiwf advise
```

and learn, in one screen:

- which aiwf capabilities this repo measurably would benefit from turning on,
  each with the signal that triggered it and the exact command to adopt it;
- which are judgment calls the operator may reasonably decline, framed as
  opportunity with tradeoff, not as breakage;
- nothing they have already declined;
- and, on a repo already using aiwf well, a clean "nothing to advise."

That is the initiative's center. aiwf should not replace documentation, a linter,
or a project manager. It should make its own latent value legible enough that an
operator never has to already know a capability exists to benefit from it.
