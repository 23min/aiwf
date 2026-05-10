---
id: G-0063
title: 'No defined start-epic ritual: epic activation is a deliberate sovereign act with preflight + optional delegation, but kernel treats it as a one-line FSM flip'
status: open
---

## What's missing

### The act

"Starting an epic" is a deliberate, sovereign moment in the planning lifecycle. It is the point at which:

- The human commits to the epic's goal/scope/out-of-scope as load-bearing (no longer a draft).
- The milestones drafted under the epic become the queued work.
- The human optionally **delegates authority** for the epic's work to an agent (`aiwf authorize E-NN --to ai/claude`), or stays in the loop themselves.
- The branch shape (per G-0059) becomes meaningful for the first time, since work is about to begin.

Today the kernel treats `aiwf promote E-NN proposed → active` as a one-line frontmatter mutation. There is no preflight, no human-only enforcement, no pairing with `aiwf authorize`, and no skill-level ritual to walk the operator through the act. The wf-rituals plugin's `aiwf-extensions/skills/` directory ships `aiwfx-plan-epic`, `aiwfx-plan-milestones`, `aiwfx-start-milestone`, `aiwfx-wrap-milestone`, `aiwfx-wrap-epic` — but **no `aiwfx-start-epic`**. The implicit model there is "epic-start = first milestone start," which collapses the sovereign act into a side effect. That model is incoherent with the principal × agent × scope provenance model the kernel commits to: a sovereign delegation moment cannot be a side effect.

### Resolved sub-decisions (this discussion)

1. **Delegation is optional.** The ritual prompts but does not require. Human-in-the-loop is a common path; the human-and-agent split is the other.
2. **Verb idiom: two commits, orchestrated by the skill.** `aiwfx-start-epic` invokes `aiwf promote E-NN active` and then (only if delegating) `aiwf authorize E-NN --to ai/<id>`. Bundling promote + authorize into one verb would muddle the "one verb = one logical event = one commit" rule, since authorization opens a *separate* scope record with its own audit trail (unlike the resolver-field flags on `aiwf promote`, which write adjacent fields on the same entity).
3. **Wrap-side timing: scope ends *before* the epic transitions to `done`.** The tail (review, sign-off, merge/PR) is the human's. This changes the existing kernel behavior of "promote-to-terminal auto-ends scopes" — under the new rule, `aiwf promote E-NN done` should refuse while active scopes exist on the epic, requiring the operator to end scopes explicitly first. **This is a kernel behavior change and warrants its own ADR before the work lands.** (See "Open downstream questions" below.)
4. **Body chokepoint generalizes.** E-0017's planned `acs-body-empty` finding (M-0066) should be parameterized over kind, not scoped to ACs only. Empty body sections at status `active`/`in_progress` are a quality gap regardless of kind. **This rescopes E-0017.** (See "Implications for E-0017" below.)

### Preflight checks (proposed)

| Check | Severity | Where |
|---|---|---|
| Epic is in `proposed` (legal `proposed → active` transition) | Refusal | Already enforced by FSM in `aiwf promote` |
| Epic has ≥1 milestone at status `draft` | Warning | New `aiwf check` rule: `epic-active-no-drafted-milestones` |
| Epic body sections (Goal, Scope, Out-of-scope) are non-empty | Warning, escalates to error in strict mode | New `aiwf check` rule, the kind-generalized form of M-0066 |
| No orphaned active scope on the epic from a prior aborted lifecycle | Refusal | Verify the existing scope FSM already covers this |
| `aiwf promote <epic> active` actor is `human/...` | Refusal | **New rule**: epic-active and epic-done are sovereign acts (mirrors `--force` being human-only) |

The first row is already in place. Rows 2–3 are kernel-checkable findings that benefit from being reported regardless of how the operator reached `active`. Row 4 is mostly verification work. Row 5 is the new sovereign-act rule.

### Skill shape — `aiwfx-start-epic`

```
aiwfx-start-epic E-NN
  ├── 1. Preflight conversation
  │     ├── Read epic spec; confirm body sections concrete
  │     ├── List milestones; confirm at least one is drafted
  │     ├── Run `aiwf check`; confirm zero kernel-finding refusals
  │     └── Run project tests/build; confirm green
  ├── 2. Delegation prompt
  │     ├── Ask: "Delegate to an agent? (--to ai/<id>) or stay in the loop?"
  │     └── If delegating, capture target identity
  ├── 3. Sovereign promotion
  │     └── aiwf promote E-NN active                (commit 1, human-only)
  ├── 4. (Optional) Authorization
  │     └── aiwf authorize E-NN --to ai/<id>        (commit 2, human-only)
  ├── 5. Branch decision (per G-059's eventual resolution)
  │     └── git checkout -b epic/E-NN-<slug>        (or whatever G-059 settles on)
  └── 6. Hand-off
        └── Either spawn subagent (delegating mode) or proceed to aiwfx-start-milestone (in-loop mode)
```

The kernel's responsibility is the rules (preflight findings, sovereign-act enforcement). The skill's responsibility is the conversation and orchestration. Subagent spawning is Claude Code surface and out of kernel scope — the kernel only knows about the authorization scope.

## Why it matters

Without this:

- Epic activation drifts. The operator silently chooses what to check; the kernel silently accepts any actor doing it.
- Provenance is incoherent. The framework commits to "principal × agent × scope" provenance, but the moment authority is delegated has no kernel-recognized shape — it's just a separate verb call the operator might or might not make.
- Branch discipline (G-0059) has no anchor — even after G-0059 is resolved, the branch-creation moment has nowhere to live without a start-epic ritual.
- Body-completeness chokepoints (G-0058 / E-0017) are AC-only, leaving epic and milestone bodies to ship empty.
- The kernel repo itself dogfoods none of this (G-0038 territory).

This turn is the canonical instance: I (Claude in this session) promoted E-0017 from `proposed` to `active` with a single verb call, on the long-running PoC branch, with no preflight, no body check, no authorization decision, no human-only enforcement, no branch creation. The user's "wait, why are we not on an epic branch?" was the only signal that anything was missing. That signal lands too late — after the commit, after the state has flipped — and only because the user happened to remember that rituals exist somewhere.

## Implications for E-0017 (in-flight)

E-0017 ("AC body prose chokepoint") is currently scoped to ACs only:

- M-0066 — `aiwf check` finding `acs-body-empty`
- M-0067 — `aiwf add ac --body-file` flag
- M-0068 — `aiwf-add` skill names fill-in-body as required

Sub-decision #4 above generalizes this: empty body sections on any non-draft entity (epic, milestone, AC, gap, contract, ADR, decision) are a quality gap. E-0017 either:

- **(a)** Stays AC-scoped; a follow-up epic generalizes the rule to other kinds. Pro: doesn't blow open the in-flight epic. Con: the rule lives in two places, two milestones do similar work.
- **(b)** Rescopes to "entity-body-empty" generalized over kind; M-0066 becomes the generalized rule, and possibly two new milestones (per-kind body section requirements + per-kind body-file flag for the verbs that don't have one) are added.

This decision should be made before M-0066 starts implementation, not after. **(b)** is more coherent with the design but heavier; **(a)** keeps momentum.

## Open downstream questions (warrant their own decisions/ADRs)

1. **Does `aiwf promote E-NN done` refuse while active scopes exist?** Today it auto-ends them. Sub-decision #3 wants explicit-end-first. This is a kernel behavior change and probably wants an ADR.
2. **Does the same sovereign-act rule apply to other kinds?** Is contract-active sovereign? ADR-accepted? Probably not in the same way (those don't open authorization scopes), but the general principle "kind X transitions to status Y require human actor" might generalize and deserves a decision.
3. **Branch shape** — per G-0059, this gap intentionally does not lock in `epic/E-NN-<slug>` vs. `milestone/M-NNN-<slug>` vs. some other form. The branch slot in `aiwfx-start-epic` step 5 is a placeholder until G-0059 resolves.
4. **Should `aiwfx-start-epic` be added to `aiwf-extensions` upstream, or live in this kernel repo as part of `internal/skills/embedded/`?** The other ritual skills are upstream; consistency suggests upstream. But the kernel-side `aiwf check` rules and provenance enforcement land in this repo regardless.

## Suggested decomposition

This gap is large enough to warrant its own epic with multiple milestones. A first cut:

- **Milestone**: `epic-active-no-drafted-milestones` finding in `aiwf check`.
- **Milestone**: kind-generalized `entity-body-empty` finding (subsumes / replaces M-0066, possibly E-0017 entirely; coordinate with E-0017 first).
- **Milestone**: human-only enforcement on `aiwf promote <epic> active` (and possibly `done`); commits + tests.
- **Milestone**: ADR for "promote-to-terminal does not auto-end scopes" behavior change, and the verb work to implement it.
- **Milestone**: `aiwfx-start-epic` skill (lives in `aiwf-extensions` upstream; this milestone may belong outside this repo).
- **Milestone**: `aiwfx-wrap-epic` updates to reflect the new wrap-timing (also upstream).

That's six milestones — non-trivial. Worth scoping carefully when the epic is planned.

