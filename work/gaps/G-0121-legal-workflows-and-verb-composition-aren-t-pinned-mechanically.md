---
id: G-0121
title: Legal workflows and verb composition aren't pinned mechanically
status: open
priority: high
---
## What's missing

The kernel pins **per-entity** legality tightly — six per-kind FSMs in `internal/entity/transition.go`, the AC and TDD-phase FSMs alongside, ~15 cross-cutting rules in `internal/check/`, and ~40 policy tests in `internal/policies/`. What is **not** pinned, and not even declared in one place, is **workflow-level** legality — the multi-step procedures a human or LLM walks through to ship value (start-epic, plan-milestones, start-milestone, TDD cycle, wrap-milestone, wrap-epic, raise-gap, address-gap, archive-sweep, reallocate, retitle, authorize/end-scope, etc.). The procedural shape of each is encoded today only in skill bodies under `.claude/skills/aiwfx-*` and `wf-rituals:*` — a recipe, not a spec.

Four concrete sub-gaps follow from that omission:

1. **No declarative enumeration of blessed workflows.** There is no artifact a contributor (human or LLM) can read to learn "these are the legal sequences of verbs, with their pre- and post-conditions per step." Skills describe one workflow each in prose; nothing cross-links them or asserts they exhaust the legal surface.
2. **No composition tests across verb chains.** Each verb is tested in isolation. Sequences like `promote → rename → reallocate → archive`, or `add ac → promote tdd_phase red→green → cancel`, or `authorize → start-milestone → end-scope mid-flight → resume` are not exercised end-to-end. G-0118 was exactly this class — `reallocate` didn't populate `prior_ids`, which broke the provenance audit on a downstream verb. There are almost certainly more latent.
3. **No tree-level post-condition assertions after verb sequences.** Properties like "after any reachable sequence of legal verbs, no AC carries `met` under a `tdd: required` milestone with `tdd_phase` not `done`," or "after any sequence, no id resolves to two entities across active+archive," or "for every authorize/end-scope pair, scopes never overlap on the same entity" are partially covered as point-in-time check rules, but never asserted **as invariants under arbitrary legal verb composition**. Property-style fuzzing of verb sequences would catch composition bugs that hand-written per-verb tests miss by construction.
4. **No declared branch/worktree choreography.** The verb surface is branch-agnostic by default, but the *legal sequence of mutations* is not branch-agnostic — there is a right answer for "where does `aiwf add epic` get called from?", "when does the feature branch (or worktree) get created relative to id allocation?", "how do allocated ids survive merge to main?", and "is this work patch-shaped (`wf-patch`) or epic-shaped (epic + milestones)?" Today these answers live as scattered conventions in skill prose; a user (or LLM) saying *"implement gap X on a worktree"* has no machine-readable signal about whether to allocate planning entities on main first vs jump straight to a feature branch. The pre-push hook's branch-unawareness — it runs `aiwf check` identically regardless of branch, ignoring legitimately transient mid-branch states like partial promotions and pending archive sweeps — is **one consequence** of the broader choreography concern, not the whole of it.

## Why it matters

The kernel's load-bearing rule is *"framework correctness must not depend on the LLM's behavior"* — but as long as workflow legality lives only in skill prose, correctness of multi-step flows does depend on the LLM (or human) faithfully walking the recipe. Per-verb FSMs guard each step in isolation; they don't guard the **sequence**. The current setup catches malformed individual moves but is structurally unable to catch *"this sequence of individually-legal moves left the tree in a state we never intended,"* or *"this sequence ran from the wrong branch and the planning artifacts are invisible on main until merge."*

Four operational consequences:

- **Composition bugs ship silently** until a downstream consumer trips on them (G-0118 pattern: the bug was filed, fixed, and a class-level test added — but the underlying gap that no integration test exercised the composition remains).
- **Choreography mistakes ship silently.** Allocating an epic on a feature branch instead of main hides it from other consumers of main, and creates id-collision pressure that `aiwf reallocate` is forced to clean up after the fact. Today there is no rule to violate — only habit; the LLM and the human both have to remember which branch each verb belongs on.
- **Skills cannot be safely refactored** without manual end-to-end re-walks of every workflow, because there is no test layer that drives the workflows mechanically across the branch transitions they specify.
- **New contributors (especially LLMs) cannot learn the legal workflow set** without reading every skill and inferring the boundaries. A declarative `legal-workflows.md` — naming each workflow, its entry condition, its sequenced verb calls with pre/post conditions, the branch each step runs from, and the tree-level invariants the workflow preserves — would let integration tests drive each workflow under fuzz, and let skills cite the spec rather than re-describe it.

The proposed shape (deferred to a milestone, not part of this gap): `docs/pocv3/design/legal-workflows.md` enumerates the workflows including their branch/worktree choreography; an `internal/workflows/` test package drives each end-to-end against a temp git repo built from the binary, with multi-branch fixtures exercising the allocate-on-main → branch → merge contract; a property-style fuzz harness composes random legal verb sequences and asserts tree-level invariants hold after each.

## Notes

Sub-gaps 1–3 are substantively covered, just realized differently than proposed: E-0033 built `internal/workflows/spec` (`Rules()`/`AntiRules()`, a closed predicate vocabulary) with real positive/negative per-cell drivers (M-0124, M-0125) that drive the actual binary; M-0130 added the `fsm-history-consistent` tree-invariant check; and E-0062's `cmd/stresstest` verb-sequence walker composes real multi-step verb chains (promote/rename/reallocate/archive/move, plus `milestone tdd` policy flips added by E-0071) against a correctness oracle. The literal proposed artifact (`docs/pocv3/design/legal-workflows.md` plus an `internal/workflows` fuzz harness) was never written — only intermediate audit docs exist — and E-0033 explicitly scoped skill-level ritual choreography (start-epic, wrap-milestone, etc.) out as "advisory only."

**Measured 2026-08-06 — the walker's coverage is narrower than the paragraph above claims.** The `verb-sequence` walker composes real chains and re-checks after every step, which is the right shape. Three limits keep it short of sub-gaps 2 and 3 as stated. *One axis of state*: it moves `status` — the six FSMs — so the reachable space is statuses, not references, areas, priorities, `depends_on` edges or bodies. *One branch*: no sequence crosses a branch boundary, so no reference is ever evaluated in a context different from the one that authored it — every two-branch scenario in the catalog is about contention (concurrent allocation, worktree races, reallocate collisions), not about a reference authored on one branch and judged on another. *One read path*: the oracle runs `aiwf check` alone and judges its findings against a curated baseline — any error-severity finding is a violation, as is any warning outside the baseline. That is an absolute allowlist applied to each state on its own terms, so it does judge what a wider walk reaches; what it cannot do is compare two surfaces, so a disagreement between `aiwf check` and `aiwf check --fast` on the same bytes is invisible to it by construction. The catalog has also drifted from the group it exists for — five scenarios judge workflow legality, eleven judge concurrency or fault injection.

The first measured instance of what that misses: `aiwf check` and `aiwf check --fast` render opposite verdicts on the same bytes in the same working copy (G-0558), a composition no scenario walks. D-0063 records the direction for closing it — widen the walker's mutation space, keep the oracle invariant-shaped, reserve named scenarios for verdicts a document specifies — and names the agreement invariants that reach this class without needing a model of the correct answer.

**Measured 2026-08-08 — what an invariant shape does and does not buy.** It saves an oracle from modelling the correct verdict; it does not save it from modelling the comparison. An agreement property still needs to know what its two sides are compared *on*, and that model is where the cost sits and where the property fails — silently, since a comparison narrowed to where both sides already agree passes on every tree, including the ones carrying the defect it exists for. The order that follows: widen the walker's mutation space first, under the baseline oracle already in place, since that oracle judges each state on its own terms and needs no new seam to do it. Then size a cross-surface agreement property as its own piece of work, against a real violating state — G-0567 is one such disagreement, live and unfixed.

Sub-gap 3's named tree invariant — *no AC `met` under a `tdd: required` milestone with `tdd_phase ≠ done` after any legal verb sequence* — is still not asserted under composition. E-0071 added `milestone tdd` policy flips to the verb-sequence walker, but the walker seeds no ACs, so it structurally cannot reach a met-phaseless-under-required state. The standalone follow-on this gap still tracks is an **AC-composition invariant-fuzz**: seed ACs, compose `add ac` / phase-promote / `milestone tdd` chains, and assert the invariant holds after every step. E-0071 unblocked it — the policy is now flippable mid-sequence by a real verb — but deliberately left it as its own milestone rather than folding it into the tdd-verb work. G-0572 is a live instance of the same invariant, reached without any walk: two legal verbs on two branches and a clean merge, into a state whose only criterion-preserving repair is sovereign.

**The two-branch composition axis, and the oracle it needs.** Both merge-reachable traps on record — G-0572 and G-0575 — were constructed by hand in minutes: two branches, individually-legal verbs on each, one conflict-free merge. Neither is reachable by any single-branch sequence, which is why the walker has never produced one; it composes on one branch. A scenario that forks, runs legal verbs on each side and merges is therefore small, uses machinery that already exists, and is justified by two measured instances rather than by a theory.

Its oracle is the part to get right, and the obvious formulation is wrong. "No error-severity finding after merging legal branches" is false as a property: a merge can legitimately produce a state that is genuinely bad, and the check flagging it is the check working. What both traps actually violate is *repairability* — that any error a merge of legal branches produces has a route out that is not sovereign. In G-0572 every unforced repair is refused; in G-0575 every act on the criterion is refused including the forced ones, and the only exit forces a terminal milestone backwards. The cheapest concrete form of the property is that a finding's own suggested command succeeds from the state that produced it, which fails in both cases — one refused by an FSM, one converging to a NoOp that reports success and changes nothing.

Sub-gap 4 (branch/worktree choreography) got real mechanization for the current single-host model via E-0030 (`--branch` flag, sequencing, isolation-escape finding, `aiwf-branch` trailer), but `docs/initiatives/agent-agnostic-execution-topology.md` (2026-06-30) still names this gap's choreography concern as open in the broader multi-host context. This gap stays open for that remainder and for the never-built declarative workflow enumeration.

## How this gap is judged satisfied

Not by artifacts built, and not by a count of invariants asserted. Both are
discharged by writing more assertions, which is how the prior attempt reported
coverage it did not have: it shipped an oracle seam, two properties, full line
coverage and a purpose-built anti-vacuity check, and was blind to the defect
class it existed for. Meanwhile both traps on record were constructed by hand,
in minutes, by someone composing two branches.

The measure is therefore a falsification test. Once the recorded traps are
repaired, construct a *third* merge-reachable bad state of a different shape — a
different rule, a different pair of verbs — without touching the harness, and
run the harness unmodified. If it stays green, it is pinned to the instances it
was built from and this gap is not satisfied, however much machinery exists. If
it goes red and names the state, the composition axis is genuinely covered.

Two properties make this worth stating rather than assuming. It can fail, which
"every invariant listed is asserted" cannot. And it is cheap relative to what it
judges: each recorded trap took minutes to construct by hand, so the test costs
far less than the harness it is measuring.

One condition on running it: whoever seeds the defect must not have built the
harness, and must not read its scenario list first. Otherwise the seeded state
is drawn from the same distribution the harness was written against, and the
result measures recall of a known list rather than reach beyond it.

**Baseline, measured 2026-08-08.** Five catalog scenarios run `git merge`, and
three seed acceptance criteria, so the raw ingredients are present. None of them
composes the shape the traps take. Every merge in the catalog is engineered to
be textually conflict-free — one merges a trailer-only commit that touches no
file, three merge adds at deliberately disjoint paths, and the fifth drives two
writes at one body field specifically to produce a textual conflict. Their own
comments state the intent. No scenario runs *different* verbs writing *different
frontmatter fields of the same entity*, which is what both recorded traps do and
what makes them merge cleanly while leaving the tree wrong. Reach on this class
is therefore zero by construction rather than by omission.

The machinery is closer than that suggests: `reachability-isolation` already
merges, re-runs `aiwf check`, and judges the result against the baseline
allowlist. What is absent is a scenario whose two sides diverge semantically
rather than by path.

