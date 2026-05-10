---
id: ADR-0005
title: 'Verb hygiene contract: complete, consistent, pre-flighted aiwf verbs'
status: proposed
---

# ADR-0005 — Verb hygiene contract: complete, consistent, pre-flighted aiwf verbs

## Context

aiwf verbs mutate the planning tree: they edit frontmatter, edit body sections, rename slugs, move directories, regenerate ROADMAP.md and STATUS.md, and emit one git commit per invocation. The kernel's existing principles cover *what* verbs do (one mutation, one commit; trailers carry provenance; validation is a separate axis) but not *how thoroughly* they preserve operator state across the mutation.

Three open gaps surfaced during E-0021 milestone planning on 2026-05-08, each instance of a different shape of the same underlying issue:

1. **G-0081** — `aiwf rename E-NN <new-slug>` succeeds even when the rename creates an `ids-unique/trunk-collision` finding that the next `aiwf check` reports. The verb has access to the same checker rule it would later trigger, but does not consult it pre-mutation. Result: the operator is left with a tree that only the next check reveals as broken, and must revert via destructive operations like `git reset --hard`.

2. **G-0082** — `aiwfx-plan-milestones` (and `aiwfx-plan-epic` when used standalone) close their planning conversations and point the operator at the next step without recommending the merge to main that the workflow logically requires. The skill completes its own scope but does not surface the workflow follow-up — leaving settled planning data hostage on a long-lived branch.

3. **G-0083** — `aiwf retitle` updates an entity's frontmatter `title:` field but leaves the body's H1 stale. The frontmatter and body H1 are both surfaces that render the entity's title; mutating one without the other produces a silent divergence the next reader has to manually reconcile via a separate `aiwf edit-body` commit.

Each gap is fixable in isolation. But the pattern across them is missing a name and a contract: **what does an aiwf verb guarantee about the consistency of operator state, beyond the narrow scope of its named mutation?** Today's verbs answer this implicitly, by what they happen to do, with no checked promise. New verbs land without the contract being applied; verb design takes the path of least resistance.

This ADR articulates the contract so that every existing and future verb has a named bar to clear, and so that reviewers and the kernel's own policy tests have a principle to invoke when a verb falls short.

## Decision

aiwf mutating verbs MUST satisfy the **verb hygiene contract**, with three obligations:

### 1. Pre-flight against known finding rules

Before performing its mutation, a verb MUST consult the check rules that would fire on the post-mutation tree state. If applying the mutation would produce a new finding (error or warning) that the verb has visibility into, the verb MUST refuse with a hint pointing at the finding's resolution path, OR proceed only when the operator explicitly opts in via a `--allow-...` flag.

The opt-out flag is the kernel's *"errors are findings, not parse failures"* stance applied to verbs themselves: the operator can override with a documented reason, but the default is the safe path.

Concrete: `aiwf rename` consults `ids-unique/trunk-collision` before renaming; `aiwf add gap --discovered-in <id>` consults `ref-resolves` before adding; `aiwf promote E-NN done` consults `milestones-all-done` before promoting.

### 2. Atomic completeness over consistent surfaces

When a verb mutates one surface that has a *consistent peer* (a related surface the kernel knows about), the verb MUST mutate both atomically in the same commit. The verb's name describes operator intent; the verb's implementation honours that intent across the kernel-known consistent surface, not just one face of it.

Concrete: `aiwf retitle <entity-id>` updates frontmatter `title:` AND body H1 (`# <ID> — <title>`); `aiwf rename <id> <slug>` updates dir name AND any internal cross-refs the kernel maintains; `aiwf milestone depends-on <m-id> ...` updates frontmatter AND any related index surfaces.

When the consistent peer cannot be located (e.g., a hand-edited body H1 that doesn't match the canonical pattern), the verb refuses with a hint OR proceeds only with an explicit `--frontmatter-only` / equivalent opt-out.

### 3. Surface follow-up actions in skills

Where a verb's completion is one step of a larger operator workflow that the kernel knows about, the calling **skill** (not the verb itself) MUST surface the follow-up action with a strong-recommendation prompt — not optional guidance. Default behaviour, explicit decline with reason.

This obligation lives at the skill layer because workflow assembly is the skill's job; the verb's job is the atomic mutation. The skill's prompt is the chokepoint that makes the larger workflow visible.

Concrete: `aiwfx-plan-milestones` ends with the merge-to-main prompt; `aiwfx-wrap-milestone` ends with the merge-to-main prompt for the milestone branch; future planning skills that close at a workflow boundary inherit the same shape.

### Sovereign override

The contract's three obligations are violated when:

- The mutation produces a finding the verb could have foreseen and didn't refuse → bug.
- The mutation hits one surface but leaves a peer stale → bug.
- The skill closes without the workflow prompt → bug.

In all three cases, the operator can override deliberately via the documented `--allow-...` flag (verb cases) or by declining the prompt with a stated reason (skill case). Sovereign overrides require a human actor with a documented reason, per the kernel's provenance-and-sovereignty model.

## Consequences

### Positive

- **Operator state is predictable.** After a mutating verb completes, the tree is in a known-good state OR the verb refused and surfaced the conflict. There is no third option of "verb succeeded, tree is inconsistent."
- **Fewer cleanup commits.** Today's friction shape — verb commits, then operator follow-up commits to fix what the verb left half-done — disappears. One verb invocation, one consistent commit.
- **Reviewers and policy tests have a named bar.** Code review of a new verb can ask: *"does this verb pre-flight? mutate the full consistent surface? close any workflow follow-ups via its caller-skill?"* and expect a yes-or-explicit-no per obligation.
- **The kernel rule "framework correctness must not depend on LLM behavior" gets stronger.** Skills are advisory; the verb-hygiene contract is mechanical. Every obligation has a code-shape implementation.

### Negative

- **Verb implementations become more defensive.** Each mutating verb now needs to reach into the relevant checker rule's API (or duplicate its logic) at pre-flight time. Small duplication risk; manageable with a shared check-runner helper.
- **Refusal-by-default may feel paternalistic.** Operators who *know* the per-branch divergence is intentional now have to type `--allow-trunk-divergence` to proceed. The opt-out flag is the release valve; the default is the safe path.
- **Existing verbs need an audit.** Each open verb (`add`, `promote`, `cancel`, `rename`, `retitle`, `edit-body`, `move`, `reallocate`, etc.) now has a known bar to meet; the audit catalogues which already comply, which need updates, and which need new opt-out flags.

### Implementation

The three open gaps each implement one obligation of the contract:

- **G-0081** implements obligation 1 (pre-flight) for `aiwf rename` against `ids-unique/trunk-collision`.
- **G-0083** implements obligation 2 (atomic completeness) for `aiwf retitle` against the frontmatter ↔ body H1 peer.
- **G-0082** implements obligation 3 (workflow follow-up) for `aiwfx-plan-epic` and `aiwfx-plan-milestones` against the merge-to-main step.

Future verb gaps, when filed, cite this ADR by id. The audit-of-existing-verbs work is itself a follow-up — likely an epic or a small audit milestone — that catalogues the contract's compliance status across the verb surface.

A complementary policy test (`internal/policies/`) MAY be added to enforce part of the contract mechanically — e.g., a check that every non-trivial verb has a pre-flight call. This is out of scope for this ADR but compatible with it.

## References

- G-0081 — pre-flight obligation, instance.
- G-0082 — workflow-follow-up obligation, instance (skill-layer).
- G-0083 — atomic-completeness obligation, instance.
- G-0084 — catalogue / umbrella of the asymmetries this ADR's contract addresses (filed alongside this ADR as the meta-gap; placeholder `G-NNN` resolved to G-0084 in a 2026-05-09 editorial pass).
- CLAUDE.md *Engineering principles* §"Errors are findings, not parse failures" — informs obligation 1's opt-out structure.
- CLAUDE.md *Engineering principles* §"Framework's correctness must not depend on the LLM's behavior" — informs obligation 3's mandate that workflow prompts live as a *strong* recommendation, with the underlying compliance moving toward kernel-level enforcement when usage demonstrates the prompt is being skipped.
- CLAUDE.md *Engineering principles* §"Kernel functionality must be AI-discoverable" — informs the requirement that opt-out flags are documented, named, and surfaced via `--help`.
