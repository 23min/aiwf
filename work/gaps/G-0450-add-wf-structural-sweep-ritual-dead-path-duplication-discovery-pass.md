---
id: G-0450
title: 'Add wf-structural-sweep ritual: dead-path + duplication discovery pass'
status: open
---
## What's missing

There is no invokable ritual for the whole-codebase **structural discovery pass** — the pass that finds dead paths and convergent duplication *before* they calcify. `wf-codebase-health` supplies the judgment *rubric* (the forces and smells), and `wf-review-code` gates a single diff, but neither packages the *discovery mechanism*: the concrete sequence of running a whole-program reachability analysis, reviewing the clone-detector's catalogue, and fanning out a convergence-scoring pass, then triaging the results into gap candidates.

The pass is worth having as a repeatable ritual because its highest-value findings are exactly the class a per-diff review and an exact-clone linter both structurally miss: same-job/different-code duplication, and helper-exists-but-bypassed coupling. Run by hand, it surfaced a several-hundred-LOC duplication tax and a set of dead paths this repo had not otherwise caught.

## Why it matters

- **The findings evaporate otherwise.** A discovery pass run ad-hoc in a conversation leaves nothing behind once the session ends. As a skill it is reproducible and its steps are auditable.
- **Triage-before-delete is a load-bearing step that must be written down.** A whole-program reachability tool flags *retained* dead code as readily as *rot* — code kept deliberately (with a tracked gap or an accepted decision) looks identical to genuine orphan. The ritual must instruct the invoker to check ownership (open gap / decision / coupled spec entry) before proposing any deletion. Skipping this step risks preempting a tracked, possibly-coupled cleanup.
- It is the natural home for the "convergent duplication" lens that `dupl`-style exact matchers cannot reach and that only a reasoner reading structure surfaces.

## Resolution shape

Author a `wf-structural-sweep` ritual (engineering skill, `wf-rituals` plugin) capturing the method in three lenses:

1. **Dead paths** — whole-program reachability analysis; then triage each hit against tracked ownership before proposing removal.
2. **Textual clones** — read the clone-detector's exclusion/known-duplication list as a live catalogue, not merely "is CI green."
3. **Convergent duplication** — fan out the `wf-codebase-health` rubric (optionally per-package) hunting same-job/different-code and bypassed-helper coupling; emit a scorecard + gap candidates, not auto-fixes.

**Method-only, no cadence.** The skill describes *what the pass is and how to run each lens*; *when/how often* to run it stays the operator's judgment (no fixed interval baked into a shipped surface). A "When to use" section names *situations* (new module, a period of fast machine-authored change, a package that feels heavy, before a large refactor) — not a schedule.

**Stack-agnostic, like `wf-codebase-health`.** The three lenses are described conceptually with per-stack tool examples (Go: whole-program `deadcode`, clone detection via the `dupl` linter; other stacks name their equivalents), because `wf-*` rituals materialize into consumer repos of any stack. No stack's tools are the only path.

**Shipped-surface hygiene.** The `SKILL.md` follows the shipped-surface rules — no real entity ids, no filesystem paths, no development history or provenance; illustrative content uses `<prefix>-NNNN` placeholders. Enforced by the `skill-body-id` check.

The skill lands alongside a **referencing structural test** under `internal/policies/` (required by the `skill-edit-structural-test-backstop`), then `aiwf update` materializes it.

## Where to fix

- New `SKILL.md` under `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-structural-sweep/`.
- A referencing structural test under `internal/policies/`.

## Related

- `wf-codebase-health` — the judgment rubric this pass drives; the sweep is its discovery front-end.
- `wf-review-code` — the per-diff gate; the sweep is the whole-codebase complement.
- G-0447 — the convergence-tax findings a run of this sweep produced.
- G-0417 — the retained-dead-code case that motivates the triage-before-delete step.
