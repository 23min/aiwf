---
id: G-0084
title: Verb hygiene contract is undocumented; G-0081/G-0082/G-0083 lack umbrella
status: open
discovered_in: E-0021
---

# G-0084 — Verb hygiene contract is undocumented; G-0081/G-0082/G-0083 lack umbrella

## What's missing

Three open gaps surfaced during E-0021 milestone planning on 2026-05-08, each touching a different aiwf verb but each instance of the same underlying issue: verbs mutate the planning tree without a checked promise about how thoroughly they preserve operator state across the mutation.

| Gap | Verb | Shape of issue |
|---|---|---|
| **G-0081** | `aiwf rename` | Mutates without pre-flighting against the `ids-unique/trunk-collision` check rule. The verb has access to the same checker it would later trigger; it just doesn't run it pre-mutation. |
| **G-0082** | `aiwfx-plan-epic` / `aiwfx-plan-milestones` | Closes its planning conversation without surfacing the merge-to-main follow-up that the workflow logically requires. The skill completes its own scope but leaves settled data hostage on a long-lived branch. |
| **G-0083** | `aiwf retitle` | Mutates frontmatter `title:` but leaves the body H1 stale. Two surfaces, one mutation, silent divergence. |

Each gap is fixable in isolation. But the pattern across them — *"verbs/skills should not leave the operator in a worse state than they found them"* — is missing a name and a contract. New verbs land without the contract being applied; verb design takes the path of least resistance; subtle state-hygiene violations accumulate.

This gap exists to **make the umbrella explicit**: file an ADR articulating the contract so each of the three open gaps becomes a concrete implementation of a named principle, and so future verb gaps can cite the ADR rather than re-litigating the same shape three more times.

## Why it matters

1. **Verb design today rewards taking the easy path.** Without a named contract, a verb that does the partial-mutation thing (`aiwf retitle`) reads as "complete" because no one checks. With a named contract, the partial form is a known anti-pattern.
2. **Reviewers have no shared bar.** Code review of a new verb today asks ad-hoc questions ("does this need a pre-flight check?", "should this also touch the body H1?"). With a named contract, review can ask the three contract questions in order.
3. **Operator-trust accumulates or erodes.** Each instance of "the verb succeeded but left the tree inconsistent" is a small trust-erosion event. The contract is the operator-facing trust commitment.
4. **The kernel's own principle ladders this.** *"Framework correctness must not depend on the LLM's behavior"* says the kernel layer is authoritative; this gap extends that to *"and authoritative means *complete and consistent*, not just *partially-mutated."*

## The umbrella ADR

**ADR-0005** — *"Verb hygiene contract: complete, consistent, pre-flighted aiwf verbs"* — has been allocated alongside this gap. The ADR articulates the contract with three obligations:

1. **Pre-flight against known finding rules.** Before mutating, the verb consults check rules that would fire on the post-mutation tree. Refuses with hint OR proceeds with explicit `--allow-...` opt-out.
2. **Atomic completeness over consistent surfaces.** When a verb mutates one surface that has a kernel-known consistent peer, it mutates both atomically.
3. **Surface follow-up actions in skills.** Where a verb's completion is one step of a larger workflow, the calling skill (not the verb) prompts the follow-up as a strong recommendation.

The three open gaps each implement one obligation:

- G-0081 → obligation 1 (pre-flight)
- G-0083 → obligation 2 (atomic completeness)
- G-0082 → obligation 3 (workflow follow-up)

This gap closes when ADR-0005 is filed (already done) and the implementing gaps are sequenced for resolution. ADR-0005 ratification follows the kernel's standard cadence — accept after the three gaps' resolution shapes prove the contract's three obligations are workable in practice.

## Resolution shape

This gap is met when:

1. **ADR-0005 exists** (status `proposed` or later) — done.
2. **Each of G-0081, G-0082, G-0083 cites ADR-0005** in its References section as the principle being implemented. Optional but valuable: a small follow-up commit on each of the three gaps' bodies adding the cross-reference.
3. **An audit of existing verbs against the contract** is filed as a follow-up entity (probably a small epic or audit milestone). The audit catalogues which verbs already comply, which need updates, and which need new opt-out flags. This is itself out of scope for this gap; this gap just ensures the audit *is filed*, not that it's executed.
4. **Future verb gaps cite ADR-0005.** When a fourth or fifth instance of the pattern surfaces, the gap body cites ADR-0005 directly rather than re-litigating the principle.

## Out of scope

- **Implementing the three open gaps themselves.** Those are their own work, sequenced separately.
- **The audit of all existing verbs.** Filed as a follow-up; not this gap's job.
- **A policy test enforcing the contract mechanically** (e.g., `internal/policies/verb_hygiene.go` that asserts every mutating verb has a pre-flight call). Compatible with this gap; not required for it. Probably belongs in the audit follow-up.
- **Renaming any of G-0081/G-0082/G-0083** to reflect their relationship to the contract. The titles are already descriptive of the specific case; the umbrella relationship lives in the cross-references.

## Discovered in

- E-0021 milestone planning, 2026-05-08. The three component gaps surfaced over the course of the planning session — first G-0081 (rename collision after a slug rename), then G-0082 (planning closure should default-merge), then G-0083 (retitle leaves H1 stale during the audit-sweep). At the third instance, the meta-pattern became visible and was named.

## References

- **ADR-0005** — *"Verb hygiene contract: complete, consistent, pre-flighted aiwf verbs"* — the umbrella ADR this gap motivates and aligns to.
- **G-0081** — pre-flight obligation, specific instance (rename verb).
- **G-0082** — workflow-follow-up obligation, specific instance (planning skills).
- **G-0083** — atomic-completeness obligation, specific instance (retitle verb).
- CLAUDE.md *Engineering principles* §"Errors are findings, not parse failures" — informs the contract's opt-out structure.
- CLAUDE.md *Engineering principles* §"Framework's correctness must not depend on the LLM's behavior" — informs why obligation 3 (workflow follow-up at skill layer) eventually wants kernel-level reinforcement.
