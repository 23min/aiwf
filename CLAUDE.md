# CLAUDE.md — aiwf PoC branch

This branch (`poc/aiwf-v3`) builds a small experimental framework called `aiwf`. The framework helps humans and AI assistants keep track of what's planned, decided, and done, by validating a small set of mechanical guarantees about a markdown-and-frontmatter project tree. Read [`docs/pocv3/design/design-decisions.md`](docs/pocv3/design/design-decisions.md) for what the PoC commits to. Read [`docs/pocv3/plans/poc-plan.md`](docs/pocv3/plans/poc-plan.md) for the four sessions of work that produce it. Read [`docs/pocv3/gaps.md`](docs/pocv3/gaps.md) for known defects, rough edges, and their status — check the matrix there before starting work to see what's open or already in flight.

The branch is intentionally isolated from `main`: research documents and the earlier architecture design have been removed here so they do not pollute Claude's working context. They remain on `main` for visitors who want to follow the design trajectory.

---

## Engineering principles

- **KISS — keep it simple.** Prefer the boring solution. Three similar lines beats a premature abstraction. Avoid cleverness — reflection, metaprogramming, deeply nested generics, control-flow tricks — unless the simple version is demonstrably worse.
- **YAGNI — don't build for tomorrow.** No speculative interfaces, no "we might need this later" config knobs, no plugin architectures for a single implementation. Add the second case when it shows up; abstract on the third.
- **No half-finished implementations.** If a feature lands, it lands tested. Stubs and TODOs in shipped code are a smell, not a milestone.
- **Errors are findings, not parse failures.** `aiwf check` loads inconsistent state and reports it; it does not refuse to start. Validation is a separate axis from loading.
- **The framework's correctness must not depend on the LLM's behavior.** Skills are advisory; the pre-push git hook and `aiwf check` are authoritative. If a guarantee depends on the LLM remembering to invoke a skill, it is not a guarantee.

For Go-specific rules (formatting, linting, testing, coverage, error handling, CLI conventions, commit-trailer convention), see `tools/CLAUDE.md`.

---

## Working with the user

- **Q&A / interview format.** When the user says "Q&A", "interview me", or anything similar, present questions or findings **one at a time**, not as a batch. For each item, give:
  1. **Context** — what the question is about and why it matters here.
  2. **Pros / cons** (or whys / why-nots) for each option.
  3. **Risks**, if any.
  4. **Your lean** and the reasoning behind it.
  5. A **numbered list of options** the user can pick from (including "something else").

  Wait for the user's choice before moving to the next item.

---

## What the PoC commits to

These are the load-bearing properties any change must preserve. They are distilled from the research arc (which lives on `main`, not here) and recorded in [`docs/pocv3/design/design-decisions.md`](docs/pocv3/design/design-decisions.md).

1. **Six entity kinds** — epic, milestone, ADR, gap, decision, contract — each with a closed status set and one Go function for legal transitions. Hardcoded; not driven by external YAML.
2. **Stable ids that survive rename, cancel, and collision.** `E-NN`, `M-NNN`, `ADR-NNNN`, `G-NNN`, `D-NNN`, `C-NNN`. The id is the primary key; the slug is just display. Renames preserve the id. "Removal" means flipping status to a terminal value, not deleting the file. Collisions are detected by `aiwf check` and resolved by `aiwf reallocate`.
3. **`aiwf check` runs as a pre-push git hook.** Validation is the chokepoint. The hook is what makes the framework's guarantees real; without it, skills are just suggestions.
4. **`aiwf history <id>` reads `git log`.** No separate event log file. Structured commit trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`) make the log queryable.
5. **Marker-managed framework artifacts in the consumer repo, regenerated only on explicit `aiwf init` / `aiwf update`.** Skills under `.claude/skills/aiwf-*` (gitignored) and git hooks under `.git/hooks/<hook>` (untracked, identified by an `# aiwf:<hook>` marker so user-written hooks are left alone). `aiwf update` is the upgrade verb — it refreshes every artifact the consumer is opted into. Stable across `git checkout` by design.
6. **Layered location-of-truth.** Engine binary lives external (machine-installed via `go install`). Per-project policy and planning state live in the consumer repo. Materialized skill adapters live in the consumer repo but are gitignored.
7. **Every mutating verb produces exactly one git commit.** That gives per-mutation atomicity for free. A failed mutation aborts before the commit.
8. **Acceptance criteria as namespaced sub-elements of milestones; TDD opt-in per milestone.** ACs are not a seventh kind — they're structured sub-elements addressed by composite id `M-NNN/AC-N`, validated by `aiwf check`, with the audit rule "AC `met` requires `tdd_phase: done`" when the milestone is `tdd: required`.

If a proposed change does not preserve one of these, treat it as a kernel-level decision and surface it explicitly — not a quiet refactor.

---

## What is *not* in the PoC

Not in scope, deliberately. None of these blocks PoC value; each can be added later when real friction demonstrates the need.

- An events.jsonl file or any append-only event log.
- A graph projection file or hash chain.
- A monotonic ID counter coordinated across branches.
- A module system or capability registry.
- Multi-host adapter generation (PoC targets Claude Code only).
- A third-party skill registry.
- Tombstones beyond "status = cancelled / wontfix / rejected / retired."
- CRDT primitives, custom merge drivers, server-side hooks.
- GitHub Issues or Linear sync.
- Full FSM-as-YAML.

If you find yourself reaching for any of the above to solve a problem, stop and check [`docs/pocv3/design/design-decisions.md`](docs/pocv3/design/design-decisions.md) §"What's deliberately not in the PoC" — there's almost certainly a simpler way.

---

## How to validate changes

```bash
go test -race ./tools/...                 # unit tests
golangci-lint run                         # linters
go build -o /tmp/aiwf ./tools/cmd/aiwf    # binary builds
```

All three should pass before committing. CI runs all of them on every push.

---

## Working on the PoC

The four sessions are in [`docs/pocv3/plans/poc-plan.md`](docs/pocv3/plans/poc-plan.md). Each session has a clear deliverable and a checkbox list. Mark items as you go; commit per logical step.

The PoC branch is not planned to merge to `main`. Commit directly on the branch; no PR ceremony. Conventional Commits subjects are still useful (`feat(aiwf): ...`, `chore(aiwf): ...`, `docs(poc): ...`).

When in doubt: the smaller change is the right change.
