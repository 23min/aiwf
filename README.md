# ai-workflow

> Design research and an experimental PoC for AI-assisted software engineering with structured planning state in the repo.

This repository is two things at once: a body of design research that worked through what an "AI workflow framework" should actually be, and a small working PoC that puts the lightest plausible answer to the test.

It is not currently a usable framework. It is a *thinking-out-loud* about what such a framework should be, plus a small implementation that validates the core ideas.

---

## Why this repo exists

AI-assisted software development has changed what "tracking the work" means. The AI helps plan, design, code, and decide — but it does so across stateless sessions, often on partial context, and without a consistent place to consult what was already settled. Existing approaches cover parts of the problem and miss others: external project-management tools (Linear, Jira, GitHub Projects) keep planning state out of the AI's working set; ADRs alone are too thin for evolving plans; ad-hoc in-repo conventions vary across teams and degrade as work scales.

Concrete symptoms we have hit repeatedly:

- The AI re-plans from scratch each session because it cannot find the current plan.
- Renaming or rescoping a milestone silently breaks references elsewhere in the repo.
- Switching branches changes the rules the AI thinks it should follow.
- Decisions get re-litigated because no one knows whether something was already settled.
- Plans drift faster than they're recorded, and structural state quietly desynchronizes from the code it claims to describe.

The research in this repo takes the problem seriously: what should an AI-aware planning framework actually be, given that the AI is now half the team and that git is already a state-management system? The PoC is the smallest concrete answer the research could justify — a place to validate the ideas in working code on real projects rather than only on paper.

---

## The research

`docs/research/` contains an arc of seven documents (`KERNEL.md`, then `00`–`06`) that walk through the load-bearing problems and how they interact:

- How a totally-ordered event log fights git's branching model.
- Whether the framework should reinvent state management or let git be the time machine.
- Whether a framework is needed at all, or whether ADRs plus a discipline are enough.
- Where discipline can live so it does not depend on the LLM remembering to enforce it.
- Where governance and provenance UX belong, and how the project-shape spectrum (solo↔team, short↔long, regulated↔not) shapes what's needed.
- Where state lives — in repo, outside, or layered — and which model is more successful.
- A concrete PoC build plan that survives all of the above.

The conclusions are distilled into [`docs/research/KERNEL.md`](docs/research/KERNEL.md) (the eight things the framework needs to do and the cross-cutting properties any solution must respect) and [`docs/research/06-poc-build-plan.md`](docs/research/06-poc-build-plan.md) (the smallest concrete shape that delivers them at solo + short-horizon scale).

For visitors trying to follow the trajectory: read `KERNEL.md` first, then skim `06-poc-build-plan.md`. The numbered docs in between are the intermediate reasoning.

---

## What we've learned

Six conclusions distilled from the research arc — each is the answer to a question we did not have a settled answer to when we started:

- **An append-only, totally-ordered event log fights git's branching model.** We walked back from the original event-sourced architecture; the merge story does not work cleanly at any scale.
- **Markdown is the source of truth; git is the time machine.** No separate event log file, no graph projection file, no hash chain. Structured commit trailers make `git log` queryable per entity.
- **Yes, a framework is needed — but a much smaller one than originally scoped.** ADRs plus a discipline is too thin; an event-sourced kernel is too thick. The right shape is a small validator plus a few verbs that produce well-shaped commits.
- **Enforcement must live where the LLM cannot skip it.** Skills are advisory; the LLM may not invoke them. The pre-push git hook and `aiwf check` are authoritative. A guarantee that depends on the LLM remembering is not a guarantee.
- **State must be layered.** Engine binary external (machine-installed). Per-project policy and planning state in-repo. Materialized AI skill adapters in-repo but gitignored, regenerated only on explicit `aiwf init` / `aiwf update`. Each layer lives where its constraints are best served.
- **Referential stability is achievable; semantic stability of prose is not.** The framework guarantees that ids like `E-19` keep meaning the same entity through rename, cancel, and collision. It does not pretend to guarantee that the meaning of the prose stays fixed; that is a property of human and AI understanding.

The full arguments live in `docs/research/`.

---

## The PoC

**Why now:** the conclusions above are settled enough on paper that more documents will not strengthen them. What strengthens them is working code we can use on real projects and learn from. We want to validate the core ideas with a few focused sessions of implementation rather than committing months of engineering to the more ambitious original design. Real friction will tell us what to add next; nothing else is committed to in advance.

The PoC is a deliberately minimal expression of the kernel. It lives on the branch [`poc/aiwf-v3`](../../tree/poc/aiwf-v3):

- A single Go binary `aiwf`, installed via `go install`.
- Six entity kinds — epic, milestone, ADR, gap, decision, contract — each with a closed status set.
- Stable ids (`E-01`, `M-001`, `ADR-0001`, `G-001`, `D-001`, `C-001`) that survive rename, cancel, and collision.
- A small `aiwf check` validator that runs as a pre-push git hook.
- Skills materialized into the consumer repo's `.claude/skills/` directory and gitignored, regenerated only on explicit `aiwf init` / `aiwf update`.
- No event log, no graph projection, no CRDTs, no module system, no registry, no multi-host adapters — yet.

The intent is to validate concepts quickly, iterate from there, and not paint ourselves into a corner. Decisions made in the PoC are deliberately reversible: the PoC branch is not planned to merge back to `main`, so a future redesign is free to take a different shape without paying for the PoC's choices.

**On forward compatibility:** if you adopt the PoC and we later build a successor framework, your repository will still be readable. The PoC's on-disk format (markdown files with frontmatter, a conventional directory layout, and structured commit trailers in `git log`) is simple enough that a future framework can import a PoC-shaped repo mechanically — even if that future framework takes a different internal shape. The door to a backwards-compatible successor is left explicitly open; the PoC is not a dead end for repositories that adopt it.

---

## Layout

```text
docs/
├── research/                # the design arc — start here to understand the project
│   ├── KERNEL.md
│   ├── 00-fighting-git.md
│   ├── 01-git-native-planning.md
│   ├── 02-do-we-need-this.md
│   ├── 03-discipline-where-the-llm-cant-skip-it.md
│   ├── 04-governance-provenance-and-the-pre-pr-tier.md
│   ├── 05-where-state-lives.md
│   └── 06-poc-build-plan.md
├── architecture.md          # earlier design (preserved as historical context)
└── build-plan.md            # earlier build sequence (likewise)
ROADMAP.md                   # earlier stage list (likewise)
tools/                       # source for the PoC binary (active development on poc/aiwf-v3)
```

The earlier `architecture.md`, `build-plan.md`, and `ROADMAP.md` describe a more ambitious design (event-sourced kernel with hash-verified projections) that the research walked back. They are preserved because the reasoning is useful and a future version of the framework may revisit pieces of it. See [`docs/research/00-fighting-git.md`](docs/research/00-fighting-git.md) for why the original direction was reconsidered.

---

## Status

Pre-alpha. The PoC branch is being built; nothing is published yet. Not recommended for production use, or any use, at present.

---

## License

Apache-2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).
