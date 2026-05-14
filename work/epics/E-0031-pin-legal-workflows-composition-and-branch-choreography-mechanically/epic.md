---
id: E-0031
title: Pin legal workflows, composition, and branch choreography mechanically
status: proposed
---
## Goal

Workflow legality — the multi-step procedures contributors walk through to ship value — moves from prose-only recipes in skill bodies to a declarative spec backed by composition integration tests and a verb-sequence fuzz harness. The chokepoint becomes "tests pass under arbitrary legal composition, including branch transitions," not "the recipe author and the recipe reader both remembered the right sequence."

## Context

The kernel pins per-entity legality tightly today (six FSMs, AC and TDD-phase FSMs, ~15 `aiwf check` rules, ~40 `internal/policies/` tests). What it does not pin is the *composition* of those verbs across multi-step workflows — `start-epic → plan-milestones → start-milestone → wf-tdd-cycle → wrap-milestone → wrap-epic`, or `add gap → promote → archive`, or `authorize → start-milestone → end-scope → resume`. Procedural shape lives only in skill bodies under `.claude/skills/aiwfx-*` and `wf-rituals:*` — recipes, not specs. G-0118 (`reallocate` failing to populate `prior_ids`, breaking the provenance audit on a downstream verb) was the canonical instance of the composition-bug class this epic guards against. G-0121 is the kernel gap; this epic is the addressing arc.

## Evidence in flight

Real-world incidents motivating this epic. Each entry surfaces a choreography rule M-0108 should encode in `legal-workflows.md`.

### Subagent dispatch without `aiwf authorize` (M-0091, May 2026)

During the M-0091 bulk-conversion (TestMain + t.Parallel across 24 internal packages), the parent session dispatched a builder subagent to convert 22 packages. The subagent's per-package commits invoked `aiwf edit-body M-0091`, producing trailers with `aiwf-actor: ai/claude` but no `aiwf-principal:` and no `aiwf-on-behalf-of:` — because no `aiwf authorize` scope was open on M-0091. The pre-commit hook (`--shape-only` by design) did not catch it; the violation surfaced at the post-implementation `aiwf check` with 22 `provenance-trailer-incoherent` errors. Retroactive principal-add surfaced the next-in-chain rule (`provenance-no-active-scope`) with no scope to reference. Recovery was a `git filter-branch --msg-filter` pass stripping every `aiwf-*` trailer from those 22 commits, demoting them to plain `chore(test):` commits with no aiwf provenance.

**Rule M-0108 should encode:** parent must run `aiwf authorize <id> --to <agent>` *before* dispatching any subagent that will invoke aiwf mutating verbs (`add`, `promote`, `edit-body`, `rename`, `retitle`, `reallocate`, `cancel`, `move`, `authorize`, `import`, `contract bind`, `contract unbind`). The scope is what licenses the agent's `aiwf-actor: ai/...` and provides the `aiwf-on-behalf-of:` reference. Without it, agent verb commits accumulate provenance debt that requires history rewriting to clear — possible only on unpushed branches.

**Secondary observation for M-0108:** the violation is structurally invisible to pre-commit (`--shape-only` is by design — it keeps the local commit loop fast). Branch-resident commits can carry provenance debt until pre-push surfaces it. The spec should call out this asymmetry so workflow authors know where the actual chokepoint is.

## Scope

### In scope

- Declarative `legal-workflows.md` spec — each workflow's pre-conditions, sequenced verb calls, branch each step runs from, post-conditions, and the tree-level invariants it preserves.
- `internal/workflows/` test package that builds the aiwf binary and drives each spec'd workflow end-to-end against a temp git repo, with multi-branch fixtures exercising the allocate-on-main → branch → merge contract.
- Property-style fuzz harness composing random legal verb sequences and asserting tree-level invariants hold after each.
- Skill-citation discipline — skills under `.claude/skills/aiwfx-*` and `wf-rituals:*` cite the spec workflow they implement; drift-prevention test pins skill ↔ spec correspondence.

### Out of scope

- Pre-push hook branch-awareness (narrow "what's legal on main vs feature branch") — a separate gap if it remains friction after this epic lands.
- Graph projection / hash-chain / events.jsonl (CLAUDE.md banned list).
- Multi-repo workflows or external-tracker sync.
- Plugin-side recipes that don't touch the aiwf verb surface.
- Custom merge drivers, server-side hooks, CRDT primitives.

## Constraints

- **The spec is the source of truth.** Skill bodies that describe a workflow cite it; integration tests execute *against* it (spec drives test cases, not the other way around). Dual-source-of-truth is the failure mode this epic eliminates.
- **Tests build the real binary.** Composition integration tests run the actual `aiwf` binary as a subprocess against a temp git repo — per CLAUDE.md "Test the seam, not just the layer." No unit-level mocking of verb dispatch at this layer.
- **Fuzz seeds are spec-derived.** The verb-sequence fuzz harness generates sequences from the spec's transition graph; it does not encode the graph in Go separately.
- **No half-finished implementations.** A workflow lands in the spec only when at least one integration test pins it. Specs without a mechanical consumer are prose — what this epic exists to move past.
- **KISS / YAGNI.** No declarative workflow DSL (CUE, custom YAML, etc.). The spec is markdown for human readability; the test package consumes structured markdown sections via a small parser. A DSL move can come later if real friction earns it.

## Success criteria

- [ ] `docs/pocv3/design/legal-workflows.md` exists and enumerates every blessed workflow currently under `.claude/skills/aiwfx-*` and `wf-rituals:*` (with branch choreography per step).
- [ ] An `internal/workflows/` test package builds the aiwf binary and drives every workflow listed in the spec end-to-end in CI.
- [ ] The fuzz harness runs `go test -fuzz` on at least one named target (legal verb sequences) and the seed corpus is committed.
- [ ] Every skill listed in the spec cites it (link or section reference); a drift-prevention test pins skill ↔ spec correspondence.
- [ ] G-0118's composition pattern is covered by an integration test in the new harness (regression coverage).

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Workflow granularity — one per skill, one per verb-chain, or one per "user intent"? | no | First milestone (spec authoring) settles. Lean: one per skill, since skills are the LLM/human entry point. |
| Does the fuzz harness include branch transitions or stay in-tree on a single branch? | no | Second milestone (composition tests) settles in-tree fuzz as floor; branch-state fuzz tagged stretch under same milestone with explicit punt-to-follow-on-gap if it overruns. |
| Spec ratified via ADR (binding) or live as a design doc (binding-by-convention)? | no | First milestone settles. Lean: design doc — the test layer is what actually pins; ADR would be ceremony. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Spec drift from skills as both evolve | med | Drift-prevention test (skill cites spec; both name same workflow). |
| Integration tests slow CI | med | Reuse the `aiwf doctor --self-check` pattern; workflows are small (<10 verbs each). |
| Fuzz harness produces equivalent-mutant failures | low | Triage discipline per CLAUDE.md G44 item 3; not all survivors are bugs. |

## Milestones

- [M-0108](work/epics/E-0031-pin-legal-workflows-composition-and-branch-choreography-mechanically/M-0108-author-legal-workflows-md-spec-enumerating-every-blessed-workflow.md) — Author `legal-workflows.md` spec enumerating every blessed workflow · depends on: —
- [M-0109](work/epics/E-0031-pin-legal-workflows-composition-and-branch-choreography-mechanically/M-0109-internal-workflows-test-harness-with-one-workflow-as-seam-test.md) — `internal/workflows/` test harness with one workflow as seam test · depends on: M-0108
- [M-0110](work/epics/E-0031-pin-legal-workflows-composition-and-branch-choreography-mechanically/M-0110-per-workflow-integration-test-coverage-including-g-0118-regression.md) — Per-workflow integration test coverage including G-0118 regression · depends on: M-0109
- [M-0111](work/epics/E-0031-pin-legal-workflows-composition-and-branch-choreography-mechanically/M-0111-skill-citation-discipline-and-skill-spec-drift-prevention-test.md) — Skill-citation discipline and skill-spec drift-prevention test · depends on: M-0108 (parallel with M-0109/M-0110)
- [M-0112](work/epics/E-0031-pin-legal-workflows-composition-and-branch-choreography-mechanically/M-0112-verb-sequence-fuzz-harness-with-spec-derived-seeds.md) — Verb-sequence fuzz harness with spec-derived seeds · depends on: M-0108, M-0109
