---
id: ADR-0006
title: 'Skills policy: per-verb default; topical multi-verb when concept-shaped; no skill when --help suffices'
status: accepted
---

## Context

The kernel principle *"kernel functionality must be AI-discoverable"* demands that every verb, flag, and closed-set value reach AI assistants through channels they routinely consult: `aiwf <verb> --help`, embedded skills under `.claude/skills/aiwf-*`, `CLAUDE.md`, or design docs cross-referenced from it. The mechanical companion (`internal/policies/skill_coverage.go`, M-074) enforces *that* every verb has skill coverage or an allowlist entry — but not *which* shape the coverage should take. That decision is judgment-shaped: per-verb skills, topical multi-verb skills, no skill at all, and the discoverability-priority case where a topical group should be split.

Three precedents surfaced through E-20:

- **`aiwf-contract`** is a topical multi-verb skill that documents `aiwf contract verify`, `aiwf contract recipes`, `aiwf contract recipe show`, `aiwf contract bind`, `aiwf contract unbind`, `aiwf contract recipe install`, and `aiwf contract recipe remove` together. The user reaches for "contract" as a concept; the verbs follow uniformly.
- **`aiwf-list` and `aiwf-status`** are split despite covering closely related read verbs over the same planning tree. The split was forced by description-match scoring: status's narrative phrasings ("what's next?", "where are we?") and list's filter-shaped phrasings ("every milestone with status X", "filter by parent") attract different prompts. Folding them under one skill diluted the description that should be specific to either.
- **`aiwf-show`** is deliberately absent and tracked by G-087: `--help` covers the surface mechanically, but body-rendering branches and composite-id discovery probably warrant a skill. The right answer isn't "papered over with --help" or "shipped as a stub"; it's "deferred with a tracked follow-up."

Without a written rule, every new verb relitigates the same decision. The skill-coverage policy fires when coverage is missing but offers no guidance on what shape to ship.

## Decision

aiwf adopts the following four-case judgment rule for skill coverage:

### Per-verb skill (default for mutating verbs that carry decision logic)

Every mutating verb whose semantics involve a closed-set choice or a flag-shaped policy decision ships its own skill: `aiwf-add`, `aiwf-promote`, `aiwf-rename`, `aiwf-retitle`, `aiwf-edit-body`, `aiwf-reallocate`, `aiwf-authorize`. The skill enumerates the closed-set (kinds, statuses, phases) and the decision criteria the AI assistant must surface when pairing the verb with a user prompt.

### Topical multi-verb skill (when users reach for the concept, not the verb)

When several verbs collaborate around a single concept and the user's prompts name the concept rather than any one verb, those verbs share one topical skill. **Precedent: `aiwf-contract`**. The skill's description enumerates concept-shaped phrasings ("define a contract", "verify contracts", "list recipes"); its body covers each verb's role in the workflow. This is the right shape only when the verbs share a discovery surface — the user wouldn't ask for `aiwf contract verify` directly without thinking about the contract itself.

### No skill (when --help suffices)

Verbs ship without a skill when:

- The surface is closed-set and trivially documented in `--help` (`aiwf version`, `aiwf whoami`, `aiwf schema`, `aiwf template`).
- The verb is an operator/install-time tool not invoked in everyday flow (`aiwf init`, `aiwf update`, `aiwf upgrade`, `aiwf doctor`, `aiwf import`).
- The verb is a thin convenience wrapper over a skill-covered verb (`aiwf cancel` over `aiwf promote` for terminal transitions).

In each case the policy's allowlist carries a one-line rationale comment in source so the absence is visible, not implicit.

### Discoverability-priority split (when one topical group must split)

When two verbs would naturally fall under one topical skill but their descriptions attract distinct prompt shapes, splitting beats topical bundling. **Precedent: `aiwf-list` and `aiwf-status`**. Both are read-only verbs over the planning tree; bundling them under "aiwf-tree-reads" would dilute the descriptions. The split lets each skill's description focus on its own prompt shape — narrative for status, structured-filter for list. The shared concept (reading the planning tree) shows up in cross-references between skill bodies, not in shared frontmatter.

The split is justified only when the host's description-match scoring would route the wrong prompts to a bundled skill. Defaulting to a split for verbs that share concept surface (the contract verbs) would over-fragment the discovery surface.

## Consequences

- **Mechanical companion:** `internal/policies/skill_coverage.go` enforces that *every* verb has either a skill or an allowlist entry, and that every embedded skill carries valid frontmatter. The judgment rule above explains *which* shape the coverage takes; the policy explains *that* it must exist.
- **Allowlist rationale comments:** every entry in `skillCoverageAllowlist` (in `internal/policies/skill_coverage.go`) carries a one-line rationale categorized by the cases above (ops verb, trivially-documented, mutation-light wrapper, kind-namespace parent, deferred). A reviewer can read the allowlist top-to-bottom and see which case justifies each absence.
- **New verbs:** when a verb is added to the Cobra tree, the design step "what verb undoes this?" (per CLAUDE.md *Designing a new verb*) gains a sibling step: *"which case from ADR-0006 applies, and where does this verb's skill live?"* The skill-coverage policy fails CI if the answer is "nowhere"; the ADR provides the language for the answer.
- **Re-evaluation prompts:** if a deferred entry (currently only `show`) accumulates user friction, file a follow-up gap (the precedent: G-087 for `aiwf-show`); the gap's body explains why `--help` no longer suffices. The policy's allowlist points at that gap until the skill ships.
- **CLAUDE.md `Skills policy` section:** points at this ADR for the *why* and at the policy file for the enforced *what*. The *What's enforced and where* table gains one row pinning the policy to its CI test chokepoint.
- **Status:** `proposed` until the next planning cycle reviews it. The mechanical policy (M-074 AC-1..AC-7) ships independently — its enforcement does not depend on this ADR's ratification.

## References

- E-20 — *Add list verb (closes G-061)* — the epic this ADR was authored within; M-074 hosts the ADR allocation.
- M-074 — *skill-coverage policy, judgment ADR, CLAUDE.md skills section, G-061 closure* — the milestone whose AC-8 this ADR fulfills.
- `internal/policies/skill_coverage.go` — the mechanical companion enforcing AC-2..AC-5 of M-074.
- `internal/skills/embedded/aiwf-contract/SKILL.md` — topical multi-verb precedent.
- `internal/skills/embedded/aiwf-list/SKILL.md` and `internal/skills/embedded/aiwf-status/SKILL.md` — discoverability-priority-split precedent.
- G-087 — follow-up gap for the deferred `aiwf-show` skill, allowlist-referenced.
- CLAUDE.md kernel principles cited verbatim: *"kernel functionality must be AI-discoverable"*, *"the framework's correctness must not depend on the LLM's behavior"*.
