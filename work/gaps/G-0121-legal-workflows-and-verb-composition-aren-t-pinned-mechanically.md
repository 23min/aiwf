---
id: G-0121
title: Legal workflows and verb composition aren't pinned mechanically
status: open
---
## What's missing

The kernel pins **per-entity** legality tightly — six per-kind FSMs in `internal/entity/transition.go`, the AC and TDD-phase FSMs alongside, ~15 cross-cutting rules in `internal/check/`, and ~40 policy tests in `internal/policies/`. What is **not** pinned, and not even declared in one place, is **workflow-level** legality — the multi-step procedures a human or LLM walks through to ship value (start-epic, plan-milestones, start-milestone, TDD cycle, wrap-milestone, wrap-epic, raise-gap, address-gap, archive-sweep, reallocate, retitle, authorize/end-scope, etc.). The procedural shape of each is encoded today only in skill bodies under `.claude/skills/aiwfx-*` and `wf-rituals:*` — a recipe, not a spec.

Three concrete sub-gaps follow from that omission:

1. **No declarative enumeration of blessed workflows.** There is no artifact a contributor (human or LLM) can read to learn "these are the legal sequences of verbs, with their pre- and post-conditions per step." Skills describe one workflow each in prose; nothing cross-links them or asserts they exhaust the legal surface.
2. **No composition tests across verb chains.** Each verb is tested in isolation. Sequences like `promote → rename → reallocate → archive`, or `add ac → promote tdd_phase red→green → cancel`, or `authorize → start-milestone → end-scope mid-flight → resume` are not exercised end-to-end. G-0118 was exactly this class — `reallocate` didn't populate `prior_ids`, which broke the provenance audit on a downstream verb. There are almost certainly more latent.
3. **No tree-level post-condition assertions after verb sequences.** Properties like "after any reachable sequence of legal verbs, no AC carries `met` under a `tdd: required` milestone with `tdd_phase` not `done`," or "after any sequence, no id resolves to two entities across active+archive," or "for every authorize/end-scope pair, scopes never overlap on the same entity" are partially covered as point-in-time check rules, but never asserted **as invariants under arbitrary legal verb composition**. Property-style fuzzing of verb sequences would catch composition bugs that hand-written per-verb tests miss by construction.

Branch-context coupling (what's legal on `main` vs a feature branch) is a related sub-gap: today's pre-push hook just runs `aiwf check` regardless of branch, so workflows that are legitimately transient mid-branch (partial promotions, pending archive sweeps) get no special treatment.

## Why it matters

The kernel's load-bearing rule is *"framework correctness must not depend on the LLM's behavior"* — but as long as workflow legality lives only in skill prose, correctness of multi-step flows does depend on the LLM (or human) faithfully walking the recipe. Per-verb FSMs guard each step in isolation; they don't guard the **sequence**. The current setup catches malformed individual moves but is structurally unable to catch *"this sequence of individually-legal moves left the tree in a state we never intended."*

Three operational consequences:

- **Composition bugs ship silently** until a downstream consumer trips on them (G-0118 pattern: the bug was filed, fixed, and a class-level test added — but the underlying gap that no integration test exercised the composition remains).
- **Skills cannot be safely refactored** without manual end-to-end re-walks of every workflow, because there is no test layer that drives the workflows mechanically.
- **New contributors (especially LLMs) cannot learn the legal workflow set** without reading every skill and inferring the boundaries. A declarative `legal-workflows.md` — naming each workflow, its entry condition, its sequenced verb calls with pre/post conditions, and the tree-level invariants it must preserve — would let integration tests drive each workflow under fuzz, and let skills cite the spec rather than re-describe it.

The proposed shape (deferred to a milestone, not part of this gap): `docs/pocv3/design/legal-workflows.md` enumerates the workflows; an `internal/workflows/` test package drives each end-to-end against a temp git repo built from the binary; a property-style fuzz harness composes random legal verb sequences and asserts tree-level invariants hold after each.
