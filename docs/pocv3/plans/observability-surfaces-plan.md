# Observability surfaces — exploration

**Status:** aspirational exploration — *not scheduled* · **Audience:** a future iteration. A lot of higher-priority kernel work precedes this; this note exists so the thinking isn't lost, not because it is queued.

The seed was a concrete idea — "an aiwf VS Code extension." On inspection that turned out to be one instance of a broader question: aiwf's **legibility layer**. The kernel is the *truth* layer (markdown + frontmatter + the `aiwf check` chokepoint); `aiwf render` is today's *one* legibility surface. This note explores extending that layer — what observability we actually want (state, plans, work, drift, references, provenance), what data already supports it, what's missing, which hosts could present it, and which visual paradigms fit which data.

The factoring that governs everything below: **a legibility surface is a third consumer of the canonical CLI/JSON surface, never a parallel source of truth.** aiwf already serves two consumers of one surface — humans (tab-completion) and AI assistants (`--help` + skills). A web view or editor extension is a third. The moment it grows its own frontmatter parser or its own FSM table it has become a second truth that drifts, and the repo's drift tests exist precisely to kill that.

For the design context that justifies the existing render shape, see [`governance-html-plan.md`](governance-html-plan.md) and [`../design/design-decisions.md`](../design/design-decisions.md). For the provenance model the timeline/swimlane views render, see [`../design/provenance-model.md`](../design/provenance-model.md).

---

## 1. What we want to see

Observability here is not one thing. It decomposes by the *dominant axis* of the data, and that axis dictates the right visual paradigm later (§4):

| Need | Question it answers | Dominant axis |
|---|---|---|
| **State** | "What's in flight right now?" | aggregate / hierarchical |
| **Plans** | "What's sequenced, what's blocked, what's next?" | temporal + dependency |
| **Work** | "How far along is this milestone / these ACs?" | hierarchical + progress |
| **Drift** | "Is the tree consistent? What's broken?" | aggregate / tabular |
| **References** | "What points at what? What's orphaned?" | relational |
| **Provenance** | "Who did what, when, on whose behalf, in what scope?" | **temporal** |

The two that the existing static render serves *poorly* — and that motivated this note — are **references** (rendered today as flat text lists) and **provenance / drift made first-class** (rendered today as counts and a warnings table, not per-entity).

---

## 2. Grounded data-layer audit

Verified on 2026-06-16 against a worktree-built binary run over this repo's own planning tree (494 entities, tree clean). The headline: **observability here is ~70% a data layer that already exists, and the single hardest part — precise per-finding location — is already done.** The gap is mostly *presentation* plus one small missing data primitive.

| Observability data need | Status | Evidence |
|---|---|---|
| Drift findings with **file + line + entity** | ✅ available | `check.Finding` carries `path`, `line` (1-based, resolved by scanning the YAML for the offending field), `entity_id`, `code`, `severity`, `subcode`, `hint` — `internal/check/check.go:74`, `internal/check/locate.go:19`. Load-bearing for editor diagnostics, and free. |
| **`referenced_by`** (reverse refs) | ✅ available | `aiwf show <id> --format=json` returns it; the inverse graph is built once at load (`Tree.ReverseRefs`, `internal/tree/tree.go:50`). Confirmed live: `show E-0040 → referenced_by: [M-0163, M-0164, M-0165]`. |
| Forward edges in the model | ✅ in the model | `parent, depends_on, supersedes, superseded_by, discovered_in, addressed_by, relates_to, linked_adrs, prior_ids` on the entity struct — `internal/entity/entity.go:362`. |
| `depends_on` projected into `list`/`status` JSON | ⚠️ partial | `aiwf list`/`status` rows expose `parent` but **drop `depends_on`** (`internal/cli/list/list.go:24`). It surfaces only in the worktree-specific lens, so milestone sequencing isn't queryable from the obvious place. |
| Full reference graph as one artifact | ❌ absent | No `graph`/`export` verb (confirmed against the verb catalog). Drawing the graph requires `list` then `show` per entity — an N+1 walk. |
| Findings **rollup** (counts by code/severity) in JSON | ❌ absent | JSON emits a flat per-instance array; the by-code collapse exists only in *text* rendering (`internal/render/render.go:111`). |
| Roadmap / sequence as data | ⚠️ partial | `aiwf render roadmap` emits markdown; no JSON DAG / topological projection. |

### What `aiwf render` produces today, and its ceiling

Nine static HTML files: `index.html` (epic rollup + finding *counts*), `status.html` (health summary + warnings table), one page per epic, one per milestone, five per-kind index pages. `html/template` + `go:embed`, **deliberately no JavaScript** ("no JS, no runtime, no external assets").

Its ceiling for observability:

- Dependency edges render as **text lists** (`M-0001 → M-0002`) — no visual graph.
- Findings appear as **counts on index + a warnings table on status**, *not* per-entity markers. There's no "this milestone has 2 problems" inline.
- **Static and JS-free by design** → re-run to refresh; a genuinely interactive graph does not fit the no-JS constraint without rethinking the surface.

---

## 3. The missing data primitives

Three, in priority order. All three pay off identically across every candidate host in §4 — they are presentation-independent, so they are the honest *first* work regardless of which surface wins.

1. **An edges / graph projection.** Either a read-only `aiwf graph --format=json` emitting forward + reverse edges in one shot, *or* (cheaper) add `depends_on` to `list`/`status` rows. This is the one primitive that unblocks any dependency or graph view on any host. A read-only verb is trivially reversible (it mutates nothing), so it clears the "what verb undoes this?" bar; it would need a skill or a `skillCoverageAllowlist` entry per the skills policy.
2. **A findings rollup in JSON.** Counts by `code` and `severity` alongside the flat array, so a dashboard doesn't have to re-derive what the text renderer already computes. Minor; consumers *can* group themselves.
3. **A local-vs-origin delta** (see §8). Not present today in structured form; the most novel and the most work.

---

## 4. Candidate hosts

All three sit on the same (mostly-complete) data layer, so the host choice is not a bet-the-house decision — it is a presentation choice made after the data primitives land.

- **(A) Extend static `aiwf render`.** Add a drift dashboard, per-entity finding markers, and a static SVG dependency graph. Cheapest, reuses everything, publishable as a CI artifact / pages site, zero install. Ceiling: static (re-run to refresh), and "interactive" is bounded by the no-JS rule.
- **(B) `aiwf serve` — a live local dashboard.** Re-queries the kernel per request, watches the tree, real interactive graph + filters. Richest observability. Cost: a new long-running *shape* for a kernel that is otherwise all one-shot verbs. It is read-only, so it does not break the one-commit-per-mutation model — but it is a bigger surface, and YAGNI says don't build it until static refresh genuinely chafes.
- **(C) VS Code extension.** Diagnostics in the Problems panel (nearly free, given file + line per finding), id-navigation everywhere (any `E-/M-/G-/D-/C-/ADR-NNNN` token becomes a link), an ambient entity tree + status-bar item, and it can *embed* render's HTML in a webview for the graph/dashboard. Best for the "while I'm working" loop; VS Code-only.

### Guardrails that bind all three

- **Never load-bearing for correctness.** Like skills, the surface is advisory. `aiwf check` pre-push + CI stay authoritative. Surface down or stale → nothing breaks.
- **All mutations route through verbs.** No hand-editing frontmatter from a UI (that bypasses id allocation and trips `provenance-untrailered-entity-commit`). "Promote" shells `aiwf promote`; "add" shells `aiwf add`.
- **Binary discipline.** Whichever surface computes diagnostics must surface the resolved `aiwf version` — the stale-worktree-binary hazard (the `make diag-aiwf` discipline) applies to a long-running surface just as it does to a terminal.

---

## 5. Visualization paradigms

### There is a standard for provenance: W3C PROV

W3C PROV (PROV-DM / PROV-O) models provenance as three node types — **Entity** (a thing), **Activity** (something that happened over time), **Agent** (who's responsible) — and its canonical visual form is a typed directed graph (entities as ellipses, activities as rectangles, agents as pentagons) with labeled edges. The mapping onto aiwf is almost exact:

| aiwf | W3C PROV |
|---|---|
| entity (E/M/G/D/C/ADR) | Entity |
| a verb commit (`aiwf-verb` + `aiwf-entity`) | Activity |
| actor / principal | Agent |
| `aiwf-on-behalf-of` / `aiwf-authorized-by` | `actedOnBehalfOf` (delegation) |
| `addressed_by`, `supersedes` | `wasDerivedFrom` / `wasInformedBy` |

The delegation edge is the notable one: aiwf's principal × agent model *is* PROV's `actedOnBehalfOf` responsibility chain.

### But match the paradigm to the dominant axis — don't force one graph

The mistake is treating *references* and *provenance* as the same picture. References are **structural/atemporal** → a node-link graph. Provenance is **temporal** → a timeline/swimlane. PROV *models* provenance as a graph, but for a human reading "who did what, when, on whose behalf," a DAG buries the one thing that matters (order + hand-offs) under layout noise.

- **Provenance → actor swimlanes on a time axis.** One lane per actor (`human/...`, `ai/...`, `bot/...`); each verb-commit is a mark on its lane. The **authorize scope** (active/paused/ended FSM) renders as a *shaded band* on the agent's lane — you see the window in which the agent was allowed to act and the verbs that landed inside it. The human↔AI hand-off is a connector at the `authorize` event. `aiwf history` *is* this stream in text already; the swimlane is a direct visual lift. Closest real-world precedent: a distributed-tracing waterfall (Jaeger/Zipkin spans on lanes).
- **References → node-link graph** (this is where SVG / Graphviz lives). Nodes coloured by kind, border by status; edges are the reference fields. Caveat: **494 nodes is a hairball** — scope it (per-epic subgraph, or a 1-hop neighbourhood around a focused entity) rather than rendering the whole tree.
- **Plans / roadmap → Gantt** (or dependency-ordered columns). `depends_on` is a DAG with a time dimension. Precedent: Airflow renders the same DAG run as *both* a graph and a Gantt — one dataset, two paradigms for two questions ("what depends on what" vs "what's the critical path").
- **Drift → a dashboard, not a graph.** Findings are aggregate/tabular: counts by severity, a sortable table (code · entity · file:line), per-entity badges. A calendar/heatmap is the one spatial touch worth having — it shows *when* drift accumulates and *which epics cluster* it.
- **State-at-a-glance → treemap / sunburst** of epic → milestone → AC, sized by AC count or open findings; or just the rollup table render already has.
- **Lineage (`prior_ids` after `reallocate`) → a small version tree** (precedent: VisTrails' history tree).

### "Is it SVG?" — the rendering tech, and the dividing line

SVG is the *output* for several of these, but the *generator* matters, and it ties straight back to the host fork:

- **Hand-rolled SVG from Go** — for swimlanes / Gantt / scope-bands, this is `<rect>` / `<line>` / `<text>` on a time axis: a few hundred lines of pure Go, **no JS, no external dependency.** Fits the existing no-JS static render beautifully. *Recommended for the provenance swimlane.*
- **Graphviz (DOT → SVG)** at build time — for the reference graph, because force-directed/hierarchical *layout* is the hard part you don't want to hand-roll. Cost: a Graphviz dependency (or a weaker pure-Go DOT layout lib).
- **Mermaid** — markdown-native (so is aiwf), with builtin `gitGraph`, `timeline`, `gantt`, and `stateDiagram` — almost a menu for these needs. But client-side Mermaid = JS (breaks no-JS), and build-time `mermaid-cli` = a Node dependency (heavy for a Go binary). Good for a webview host, awkward for static render.
- **d3 / cytoscape.js** — the only way to tame the 494-node reference hairball *interactively* (filter, expand, pan/zoom). Requires the JS host — i.e. the `aiwf serve` or VS Code-webview branch.

**The clean dividing line:** temporal views (swimlane / Gantt / scope-bands) are cheap pure-Go SVG that fit static render *today*; the interactive reference graph is the one thing that genuinely pulls toward a JS host. That line is also a sensible sequencing boundary.

---

## 6. Staged delivery

The work splits along one clean line: **does it need a renderer?** Data that lands in surfaces a human already reads is a win regardless of whether any visual surface is ever built, so it goes first. Pure-JSON plumbing (an edges export, a findings rollup) is built only *alongside* the renderer that consumes it — the rule is *don't ship JSON nobody reads*. The expensive interactive host comes last, and only if the cheap one chafes.

### Phase 1 — Data + CLI (standalone wins; no renderer)

Each item enhances the interactive CLI directly and is independently shippable. These are the near-term candidate gaps.

- **`depends_on` in the primary `aiwf status` + `aiwf list`** (human text + JSON) — today it surfaces only in the `--worktrees` lens. *(wf-patch)*
- **Readiness in `status`** — mark in-flight/draft milestones **ready** (all `depends_on` terminal) vs **blocked** (name the open blocker). Needs the edges above first. *(small)*
- **Local-vs-origin delta in `status`** — "your branch is N ahead, M entities differ from origin." *(small epic: a tree-at-ref loader + diff reusing the existing `BlobReader` / merge-base machinery, plus an ADR for the offline-first "vs last-fetched origin" semantics)*

*Exit criterion:* dependency edges, milestone readiness, and local-vs-origin state are all answerable from the terminal. **This phase is unconditional — do it regardless of whether Phases 2–3 ever happen.**

### Phase 2 — First visual host + the JSON it consumes

Only when a visual surface is actually wanted. Pick the cheapest host (lean: extend the existing static `render`, or a VS Code extension — §4) and build the JSON primitives (edges/graph projection, findings rollup — §3) *as part of* this phase, not before. Ship the cheap views first: the provenance swimlane, scope-bands, and gantt are pure-Go SVG that fit static render's no-JS constraint (§5).

*Exit criterion:* drift, provenance, and roadmap are legible visually — refreshed by re-running `render`, or live in the editor.

### Phase 3 — Interactive graph / richer host (only if Phase 2 chafes)

The interactive full-tree reference graph (the §5 hairball) is the one thing that genuinely pulls toward a JS host (`aiwf serve` or a VS Code webview — §4); d3 / cytoscape and live filtering earn their keep here. Reach for it last, gated on demonstrated need.

Phases 2 and 3 are deliberately **not** committed — they are gated on shown need, per YAGNI. Only Phase 1 is a standing intention.

---

## 7. Comparison to existing process / PM tools

The instinct is to benchmark aiwf against Jira / GitHub Issues / Linear. That is the wrong peer group, and benchmarking against it mostly produces "aiwf is missing everything" — which misreads what aiwf *is*. Those are server-backed work trackers; aiwf is a git-native validator of planning state. Its real peers are **distributed, git-native trackers** (Fossil — a DVCS with built-in tickets + wiki, the closest philosophical cousin; `git-bug`; `git-issue`; Sit), **ADR tooling** (`adr-tools` and friends), and **docs-as-code / plan-as-code** generally. Against *that* group the comparison is sharp; against Jira it is a category error — closer to comparing a type-checker to a CRM.

### What aiwf uniquely adds

1. **Plan-as-code, co-versioned and co-merged with the work.** The plan lives in the repo as markdown + frontmatter. Branch, and the plan branches; merge, and the plan merges — atomically, in one history. In a separate tracker the ticket does not branch with the feature, and the two are hand-synced forever. aiwf has one source of truth, one merge, and `git clone` retrieves everything — no export, no lock-in.
2. **It is a *validator*, not a *tracker*.** This is the category shift. SaaS trackers have configurable workflows enforced server-side by the app; you cannot make "every status change is FSM-legal and every reference resolves" a hard, local, pre-merge gate *that travels with the repo*. aiwf's `aiwf check` pre-push hook + CI is that chokepoint, and the checker itself is in the repo. aiwf is closer to a linter / type-checker for planning state than to a ticket database — the guarantee does not depend on anyone remembering.
3. **Native principal × agent × scope provenance.** SaaS trackers have "assignee" and an audit log, but no concept of *a human authorized an AI agent to act autonomously within this scope, with a typed scope FSM, every act tracing to a named human principal even when a bot ran the verb.* A bot commit is just another actor — no principal/agent/scope separation. As AI does more of the work, "who is accountable for the AI's action" becomes the question, and aiwf models it natively (`actedOnBehalfOf`, in PROV terms). This is the most defensible differentiator in the current AI-assisted-development era.

The deepest structural root of all three is the distributed origin-vs-local model — treated on its own in §8.

### What aiwf is missing (honest)

Almost all of it lies *outside* aiwf's target (solo / small-team / AI-assisted dev keeping plan + code + decisions coherent in one repo), and most is *deliberately* out of scope — but naming it matters:

| Capability | SaaS trackers | aiwf | In aiwf's target? |
|---|---|---|---|
| Multi-user real-time collab, assignment, @-mentions, watchers | ✅ | ❌ (git merge only) | partially — the rough edge |
| External intake (bug reports, support, customer roadmaps, voting) | ✅ | ❌ (commit-only; needs repo access) | no |
| Non-technical stakeholder UI (PM / exec dashboards) | ✅ | ❌ (CLI + static HTML) | no |
| Push notifications ("you've been assigned") | ✅ | ❌ (git is pull-based) | no |
| Portfolio / cross-repo rollup | ✅ | ❌ (per-repo by design) | no |
| Integrations marketplace (Slack, Figma, time-tracking) | ✅ | ❌ (out of scope) | no |
| Estimation / velocity / burndown / sprints | ✅ | ❌ | deliberate values difference |
| Powerful saved queries / full-text search at scale (JQL) | ✅ | ⚠️ basic `list` / `status` + grep | maybe later |

Two deserve more than a shrug:

- **Estimation / velocity is a values difference, not an oversight.** SaaS trackers center throughput forecasting ("will we finish by Friday"). aiwf centers correctness and provenance ("is the plan internally consistent, and who decided what"). aiwf will never produce a burndown — a positioning choice worth stating out loud.
- **Team-scale concurrency is the one latent risk *inside* the target.** Git merge handles concurrency in principle, and even solo operators hit id collisions across worktrees. With more than one human plus multiple AI agents working concurrently, the local-vs-origin reconciliation UX (collisions, merge, "what's the real state") becomes make-or-break — exactly what the local-vs-origin observability delta (§3 item 3, §8) would harden. The observability work is therefore not just ergonomics; it is the mitigation for aiwf's sharpest within-target weakness.

### Net

aiwf adds real, differentiated value in three things: plan-as-code co-merged with the work, a mechanically-enforced validator rather than a tracker, and native human↔AI provenance. What it "misses" is almost entirely the SaaS-collaboration surface it deliberately declined. The honest framing is not "aiwf is behind Jira" but "aiwf is a different category that happens to share the word *project*." The single thing to watch inside its own lane is team-scale concurrency — which loops back to why the local-vs-origin observability is worth building.

---

## 8. The origin-vs-local dimension

aiwf is **distributed like git itself**: every clone carries the full planning state and history; `origin` is a *convention* (the shared trunk), not a *requirement*. This is the deepest structural difference from server-centric trackers, and it has direct observability consequences:

- **Local can diverge from origin and reconverge via merge.** Power: branch-scoped planning, offline work, plan-travels-with-code. Peril: id collisions across un-pushed sibling branches (the trunk-aware allocator unions with `origin/main` but cannot see un-pushed siblings → `ids-unique/trunk-collision`, resolved by `aiwf reallocate`). This is a genuinely novel failure mode that central-counter trackers don't have — aiwf chose distributed-with-collision-detection over a central monotonic counter (explicitly out of scope).
- **Every state/drift/graph view has a local-vs-origin axis.** "What's the state?" has two answers — on my branch vs on trunk. A strong surface shows *both* and the *delta*: "your branch has N entities not yet on origin," "this milestone is `done` locally but `active` on origin," "trunk has advanced M commits since your merge-base." Git tooling does ahead/behind; PM tools structurally cannot, because they have no "local."
- **Provenance has a pushed-vs-local-only distinction.** A commit's trailers are local until pushed; `aiwf history` on a branch shows un-pushed acts. The swimlane could distinguish shared (pushed) from private (local-only) events — a thing no server tracker can express.

This is also the differentiator that most separates aiwf from server-centric trackers (§7 above): in a SaaS tracker the *server* is canonical and your view is a cache; in aiwf your *local working tree* is canonical (mechanically validated) and origin is just the shared rendezvous.

---

## 9. Open questions (to decide when this is scheduled)

1. **Which host first?** Lean: land the §3 data primitives, then **(C)** for the daily-work loop *or* **(A)** for a shareable governance view; treat **(B)** as the surface you reach for only if static refresh chafes.
2. **`aiwf graph` verb vs. enriching `list`/`status`?** A dedicated read-only graph verb is cleaner for a graph consumer; enriching `list` is cheaper and serves more callers. Possibly both.
3. **How much of the local-vs-origin delta belongs in the kernel vs the surface?** Computing ahead/behind and per-entity local/origin status is git work the kernel is well-placed to do once and expose as JSON — but it is new surface area.
4. **Provenance swimlane scope.** Per-entity (one entity's lifecycle) vs per-repo (all actors over time)? The per-entity view is the obvious first cut and maps onto `aiwf history <id>`.
