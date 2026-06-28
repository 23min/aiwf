---
id: ADR-0021
title: Sanctioned global area value for inherently-cross-cutting entities
status: accepted
prior_ids:
    - ADR-0020
---
## Context

The area feature is being hardened from a label-only tag into a trustworthy filter for the
1:1 project↔area monorepo, by anchoring each area to a path glob (the oracle) and adding an
`areas.required: true` knob that promotes an untagged entity to a blocking finding. That model
assumes every entity has exactly one project home whose paths the kernel can check against
(the coverage/bijection and mistag checks).

Some entities have no single home by nature: ADRs and decisions are cross-cutting by kind, and
seam contracts (the backend↔frontend payload shape, an external-API seam) belong to no one
project's code — that is precisely what the contract kind is for. Under a blanket
`areas.required` these get forced into one area or blocked, and under the path oracle their
multi-project paths trip the coverage/mistag checks. A seam is, by definition, the not-1:1
residue the clean model can't absorb.

The naive escape — "untagged-and-of-a-cross-cutting-kind = legitimately global" — is rejected:
it makes absence-of-area meaningful again and so cannot distinguish an intentionally global
seam contract from one whose tag was forgotten. That is the exact silent-mislabel state the
`areas.required` knob exists to eliminate.

## Decision

Introduce `global` as an explicit, reserved value of the single-valued `area` dimension — not
an exemption from it. The dimension's value set becomes {declared members} ∪ {`global`}.

- **`areas.required` stays total.** Every entity must carry an area value; forgetting still
  errors. `global` is a satisfying assignment, chosen affirmatively (`area: global` in
  frontmatter), never inferred from absence.
- **The path oracle treats `global` as "claims no path."** A `global` entity is outside the
  domain of the coverage/bijection check and is skipped by mistag detection — a clean domain
  exclusion, not a per-rule special case.
- **One uniform per-entity mechanism, no kind-level global list.** Contracts are bimodal (a
  project-internal contract carries that project's area; only seam contracts are global), so
  the seam/non-seam distinction is necessarily per-entity and cannot be a kind-level rule. The
  same explicit `area: global` tag therefore serves ADR, decision, and seam-contract alike. A
  kind-level default for ADR/decision is deliberately not added now (YAGNI); revisit only if
  typing `area: global` on every ADR proves to be real friction in the dogfood.
- **`global` is a reserved member name.** The kernel forbids a declared `areas.members` entry
  from being named `global`, so a real project cannot shadow the sentinel. Token choice:
  `global` (readable) over `*` (collision-proof but glob-colliding and uglier); the
  reserved-name guard covers the one risk of a project literally named "global".

## The validation surface (a reserved value, accepted everywhere a member is)

`global` is a reserved *value*, so it must be accepted at every site that today validates an
`area` against the declared member set — described here by capability, not by the milestones
that will implement them (that sequencing is a planning concern):

- **The present-⇒-declared check (`area-unknown`)** must treat `global` as known. This is the
  load-bearing site: under `areas.required` that finding escalates to an *error*, so an
  un-whitelisted `global` would be *blocked* under exactly the strict regime this decision
  serves. (This, not the `area-required` present-at-all check, is the real "declared member or
  global" site — `area-required` fires on an *empty* area, which a non-empty `global` already
  satisfies.)
- **The tagging and creation verbs** (`set-area`, `add --area`) must accept `global` as a
  value, not only a declared member.
- **The config validator** rejects a declared member named `global` (the reserved-name guard).
- A single predicate — `IsValidAreaValue(v, members) = v == "global" || isMember(v, members)`
  — centralizes the reserved value across all these sites, so it is defined once, not
  re-litigated per rule.

## Consequences

- **Filtering:** `--area <member>` returns only that member; `global` entities surface under
  `--area global` and in all unscoped views (consistent with "default views never hide").
  Cross-cutting discovery — "everything that touches this seam" — is answered through the
  reference graph (`aiwf show <contract-id>` → `referenced_by`, a maintained reverse-reference
  projection), not the area filter. This is the intended transversal query path and needs no
  new axis.
- **The oracle-dependent checks** (coverage/bijection, mistag) take their domain as
  member-tagged entities only; `global` is excluded from both rather than special-cased per
  rule.
- **Deliberately deferred (a later idea, not built now):** a stronger seam check that requires
  a `global` seam contract's touched paths to lie within the union of the areas it bridges
  (verifying the seam actually sits between X and Y), rather than skipping path checks
  entirely. Noted so it isn't lost; out of scope for the dogfood.
- **Conceptual honesty:** `global` is the named, explicit not-1:1 escape valve. Declaring it
  affirmatively is what prevents the genuinely-cross-cutting minority from masquerading as
  forgotten tags, so the 1:1 invariant stays true for everything else without lying.
