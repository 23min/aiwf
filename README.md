# ai-workflow

> Design research and an experimental PoC for AI-assisted software engineering with structured planning state in the repo.

This repository is two things at once: a body of design research that worked through what an "AI workflow framework" should actually be, and a small working PoC that puts the lightest plausible answer to the test.

It is not currently a usable framework. It is a *thinking-out-loud* about what such a framework should be, plus a four-session implementation that validates the core ideas.

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

## The PoC

The PoC is a deliberately minimal expression of the kernel. It lives on the branch [`poc/aiwf-v3`](../../tree/poc/aiwf-v3):

- A single Go binary `aiwf`, installed via `go install`.
- Six entity kinds — epic, milestone, ADR, gap, decision, contract — each with a closed status set.
- Stable ids (`E-01`, `M-001`, `ADR-0001`, `G-001`, `D-001`, `C-001`) that survive rename, cancel, and collision.
- A small `aiwf check` validator that runs as a pre-push git hook.
- Skills materialized into the consumer repo's `.claude/skills/` directory and gitignored, regenerated only on explicit `aiwf init` / `aiwf update`.
- No event log, no graph projection, no CRDTs, no module system, no registry, no multi-host adapters — yet.

The intent is to validate the core concepts in a few sessions of focused work, use the result on real projects, and iterate based on real friction. Decisions made in the PoC are deliberately reversible: the PoC branch is not planned to merge back to `main`, so a future redesign can take a different shape without paying for the PoC's choices.

The PoC's on-disk format (markdown files with frontmatter, conventional directory layout, structured commit trailers) is simple enough that a future v2 reader could import a v1 repo's state mechanically. The door to a backwards-compatible successor is left explicitly open.

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
