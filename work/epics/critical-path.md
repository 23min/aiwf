# Critical path — recommended sequence and pending decisions

> **Status:** temporary planning artifact. Last synthesised **2026-05-08** during E-20 planning, after a long discussion in which the open landscape was inventoried and ordered. Update by hand, or rewrite when invoked. Replace with a proper kernel feature (a `landscape`/`paths` verb plus a synthesis skill) when usage justifies — see *Why this doc exists* below.
>
> **Audience:** the human operator (and any AI assistant routing through `aiwf-status` and adjacent skills) trying to answer *"what should I do next?"* across 60+ entities, 3 proposed ADRs, several open epics, and 16 open gaps.

## Why this doc exists

`aiwf status` and `aiwf render roadmap` show *state* — what entities exist, what their statuses are. Neither shows *direction* — what to do next, in what order, with which dependencies foregrounded. That synthesis lives in the operator's head and scrolls out of conversation history.

This doc is a holding pattern for that synthesis. Each rewrite is a snapshot of judgement at a point in time; it is not authoritative against the entity tree, which is the truth. When state and this doc disagree, state wins; this doc is wrong and gets rewritten.

## Open work landscape

Categorised by leverage on future work, not by chronology of when they appeared.

### Tier 1 — fixes that compound across every future planning/implementation session

| Item | Kind | Cost | What it unblocks |
|---|---|---|---|
| **G-071** (entity-body-empty rule lifecycle-blind) | gap (bug) | wf-patch or one small milestone | Every future plan-milestones session leaves a clean tree. Closes the 24-warning E-20 backlog and the standing ADR-0002 noise in the same commit. |
| **G-072** (no writer verb for milestone `depends_on`) | gap (kernel asymmetry) | small milestone (one new verb or `--depends-on` flag on `aiwf add milestone`) | Every multi-milestone epic gets first-class DAG edges, FF-mergeable milestone branches, machine-checkable cycle detection. Eliminates the prose-only fallback used in E-20. |
| **G-065** (no `aiwf retitle` verb) | gap (kernel asymmetry) | small milestone | Every scope-change moment can correct the title. Less load-bearing than G-071/G-072 but cheap. |

### Tier 2 — architecturally foundational; ratify and implement before downstream work

| Item | Kind | Cost | What it unblocks |
|---|---|---|---|
| **ADR-0001** (mint entity ids at trunk integration) | ADR (proposed) | ratification + dedicated implementation epic | All future id allocation. Foundational for ADR-0003 (findings inherit), parallel-branch work in general, the agent-orchestration substrate's parallel-cycle pattern. |
| **ADR-0004** (uniform archive convention) | ADR (proposed) | ratification + dedicated implementation epic | All read verbs (list, status, show); makes high-volume kinds tractable. **Naturally retires G-071 case 2** — when terminal entities move to `archive/` subdirs, the body-empty rule can ignore that path tree. E-20 already designed for forward-compat. |
| **ADR-0003** (add finding F-NNN as 7th kind) | ADR (proposed) | ratification + dedicated implementation epic | Cycle-time findings, `aiwf check` escalation, AC-closure chokepoint. Logical successor to ADR-0001 (inherits id model) and ADR-0004 (high-volume kind needs archive). Substrate for E-19 unfreezing. |

### Tier 3 — workflow rituals (low-hanging codification)

| Item | Kind | Cost | What it unblocks |
|---|---|---|---|
| **G-059** (no canonical branch model: epic→milestone hierarchy ↔ git branches) | gap | medium (ADR + skill update) | Every aiwfx-start-milestone / start-epic. We work around it but the workaround is ad-hoc. |
| **G-060** (patch ritual loosely defined) | gap | medium (skill update) | wf-patch operations get a canonical shape; small fixes don't require re-deriving conventions. |
| **G-063** (no start-epic ritual) | gap | medium (new skill or section in existing) | E-20's start would be more uniform; future epics get a defined preflight. |

### Tier 4 — operational debris (tiny fixes, batch when convenient)

| Item | Kind | Cost | Notes |
|---|---|---|---|
| **G-056** (render `site/` not gitignored) | gap | tiny | One-line `.gitignore` edit |
| **G-057** (stray `aiwf` binary in repo root not gitignored) | gap | tiny | Same |
| **G-069** (`aiwf init` ritual nudge hardcodes user-scope CLI form) | gap | small | Fresh-operator onboarding correctness |

### Tier 5 — defer until a forcing function shows up

| Item | Why defer |
|---|---|
| **G-070** (`aiwf doctor --format=json`) | Wait for a JSON consumer to appear |
| **G-067** (wf-tdd-cycle is honor-system advisory) | Larger fix; couples to the agent-orchestration substrate work |
| **G-068** (discoverability policy misses dynamic finding subcodes) | Narrow; activates more strongly once F-NNN ships |
| **G-022** (provenance model extension surface) | Open-ended design question; no forcing function |
| **G-023** (delegated `--force` via `aiwf authorize --allow-force`) | Narrow; needed when a real subagent flow requires bounded sovereignty |
| **G-058** (AC body sections ship empty) | Likely already addressed by E-17 (M-066/M-067/M-068); verify status before adding to active list |

## Already-scheduled epic work

| Item | Status | Notes |
|---|---|---|
| **E-20** (Add list verb, closes G-061) | proposed; M-072/M-073/M-074 ready | Just planned 2026-05-08; this is the next execution target |
| **E-16** (TDD policy declaration chokepoint, closes G-055) | proposed; M-062..M-065 drafted | Closer to execution-ready than E-19; runs any time |
| **E-19** (parallel TDD subagents) | **deferred** | Pending agent-orchestration substrate finish + implementation; original framing preserved in epic body |

## Recommended sequence

**Before E-20's M-072 starts** — Tier 1 cleanup, batch as small milestones or wf-patches:

1. **Fix G-071** (lifecycle-gate `entity-body-empty`). Closes the 24+3 warning baseline. The fix predicate (`entity.IsTerminal(kind, status)`) is already on M-072's plate per the epic spec — but M-072 only *adds* the helper for list to use; G-071's fix would also *consume* it in `internal/check/entity_body.go`. Could roll the consumption into M-072 via a scope addition, or land it as a separate milestone first.
2. **Fix G-072** (depends_on writer verb). Adds `--depends-on` to `aiwf add milestone` and/or a dedicated `aiwf milestone depends-on` verb. Single-milestone scope. Updates `aiwf-add` and `aiwfx-plan-milestones` skills per G-072's body.
3. **Optional: fix G-065** (retitle verb). Same scale; bundle with G-072 if the verb design rhymes.

**Then** — E-20 implementation, with cleaner tooling around it:

4. M-072 → M-073 → M-074 → close G-061 at wrap. The 24 AC warnings become 0 if G-071 was fixed first.

**After E-20** — foundational architectural decisions:

5. **ADR-0001** (id minting). Ratify; file implementation epic.
6. **ADR-0004** (archive convention). Ratify; file implementation epic. Once implemented, G-071 case 2 is solved structurally (terminal entities live in `archive/` paths the rule ignores).
7. **ADR-0003** (finding kind). Ratify; file implementation epic.

**Parallel low-priority track** (any time, by mood):

8. Tier 4 operational debris (G-056/G-057/G-069) as a single wf-patch.
9. **E-16** when TDD-policy declaration is a felt need.

**Future**:

10. Tier 3 workflow rituals (G-059/G-060/G-063) when their friction sharpens.
11. **E-19** unfreezes after the agent-orchestration substrate is itself decomposed and implemented — likely a multi-step path: review and stabilise `agent-orchestration.md`, decompose into one or more implementation epics, those land via the normal cycle, E-19's scope gets rewritten against the new substrate, E-19 starts.

## The first decision

The next concrete fork is between three options for handling Tier 1 fixes:

**A. Fix G-071 + G-072 standalone first (1–2 small milestones), then start M-072.** Cleaner baseline, but adds delay before E-20 implementation. Cost: 2 small milestones; win: every future planning session.

**B. Roll G-071's fix into M-072.** M-072 already adds the `entity.IsTerminal` helper for list to use; consuming the same helper in `internal/check/entity_body.go` is a 5–10 line addition. Adds one AC to M-072 ("`entity-body-empty` rule consults `IsTerminal` and gates on parent-milestone status; G-071 closed"). Marginal scope creep on M-072, fast resolution. G-072 stays standalone.

**C. Start M-072 as planned; defer G-071 and G-072 to standalone fixes after E-20.** Status quo. Lives with the 27 warnings during E-20 implementation. Cheapest for E-20; expensive in cumulative warning noise.

**Lean: B** — bundling G-071 into M-072 is small enough not to disturb the milestone's shape, eliminates the warning baseline immediately, and keeps E-20 moving. G-072 stays a follow-up because its scope is fundamentally different (a new writer verb is its own change).

If keeping M-072 narrowly scoped matters more than warning cleanliness, **A** is the disciplined choice. Estimated extra: ~2 days for the two small milestones.

## Pending decisions awaiting an answer

These are the open Q&A items implied by everything above. None are blocking E-20's M-072 start.

1. **A vs B vs C on Tier 1 bundling** (above). Blocks the *shape* of M-072.
2. **Ratification of ADR-0001 / ADR-0003 / ADR-0004.** Each has been `proposed` since at least 2026-05-07. Once ratified, each spawns its own implementation epic.
3. **Order of ADR implementation epics.** ADR-0001 first (foundational), then ADR-0004, then ADR-0003 — but worth confirming when ratification happens.
4. **Whether G-058 is actually still open.** The ID list still shows it; M-066/M-067/M-068 under E-17 likely closed it. Audit and either retire or refile.
5. **Whether `critical-path.md` should graduate.** When usage justifies, this doc becomes a `landscape`/`paths` kernel verb plus a synthesis skill (see below). Until then, it lives here as a temporary holding pattern.

## Why this doc may not be the long-term answer

The synthesis above wasn't fully mechanical — it had judgement in it (Tier ordering, A/B/C ranking, which gaps to defer). A proper feature would split the data assembly (mechanical, kernel-side) from the synthesis (LLM-side, conversational with Q&A gating).

A plausible end-state: a kernel verb (e.g. `aiwf landscape`) that emits structured JSON over the open work surface — every open gap with `discovered_in`, every proposed ADR with its dependency chain, every open epic with its draft milestones, every cross-reference. A skill (e.g. `aiwfx-recommend-sequence`) consumes that JSON and produces the tiers + recommended sequence + Q&A as conversational output. Same shape as `aiwf-status` (verb produces data, skill narrates).

Until that pattern is built, this doc is the artefact. Update it when the landscape changes; rewrite it when the work shifts; replace it when the feature lands.
