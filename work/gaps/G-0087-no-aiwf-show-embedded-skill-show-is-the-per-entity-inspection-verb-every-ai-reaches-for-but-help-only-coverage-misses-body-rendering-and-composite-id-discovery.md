---
id: G-0087
title: no aiwf-show embedded skill; show is the per-entity inspection verb every AI reaches for, but --help-only coverage misses body-rendering and composite-id discovery
status: open
discovered_in: M-0074
---

## What's missing

`internal/skills/embedded/aiwf-show/` does not exist. The skill-coverage policy (M-0074, `internal/policies/skill_coverage.go`) lists `show` in `skillCoverageAllowlist` with the rationale `"deferred — see G-087 (a per-entity inspection skill warrants its own design pass; --help covers the surface mechanically in the meantime)"`.

`aiwf show <id>` is the canonical per-entity inspection verb — every AI assistant reaches for it when a user names an entity ("show me G-NNN", "what does M-0007 look like?"). Its surface is rich:

- Composite ids: `aiwf show M-NNN/AC-N` returns just that AC's record. Composite-id discovery isn't obvious from `--help`.
- Body sections: the verb returns frontmatter + acs + recent history + active findings + referenced_by. JSON form additionally carries body (a map of section-heading slug to prose). Knowing which body sections exist per kind (epic goal/scope/out_of_scope; milestone goal/acceptance_criteria; etc.) is something a skill body should enumerate.
- Per-AC payloads: history events carry `tests {pass,fail,skip,total}` when the commit had an aiwf-tests trailer. Discoverable from json output but not from --help text.

`--help` covers the surface mechanically — the verb is invokable, the flags self-explain — but does not surface the *conceptual* shape: when to reach for show vs. history vs. status, what the JSON envelope's body field carries per kind, the composite-id usage pattern.

## Why it matters

The skill-coverage policy that landed in M-0074 makes every absent skill explicit via the allowlist. `show` is the only entry whose rationale is "deferred" rather than "ops verb / trivially documented" — i.e., the only entry where the absence is acknowledged as a real gap rather than a deliberate scoping decision.

Per the kernel principle *"kernel functionality must be AI-discoverable"*: an AI assistant routing a user prompt against the skill set today does not have a `show` skill to match. Instead it relies on:

1. Reading `aiwf show --help` mid-conversation (high latency, breaks flow).
2. Pattern-matching from prior sessions or context (unreliable, drifts as the verb evolves).
3. Falling back to verb-less command construction (compose multiple `aiwf check` / `git log` / `cat work/...` calls).

None of these are the discovery shape the kernel principle commits to.

## Fix shape

Allocate a milestone (under E-NN TBD) that ships `internal/skills/embedded/aiwf-show/SKILL.md`. Body shape (suggested, refine at start-milestone):

1. *What it does* — frontmatter + acs + history + findings + referenced_by; JSON also carries body.
2. *When to use* — composite-id discovery (`aiwf show M-NNN/AC-N`), body-render branches per kind, comparing show vs. history vs. status decision criteria.
3. *Recipes* — `aiwf show E-NN`, `aiwf show M-NNN --format=json --pretty`, `aiwf show M-NNN/AC-N`, JSON-piping examples.
4. *Output* — text default shape + JSON envelope shape (named body keys per kind).

Once the skill ships, the allowlist entry for `show` is removed; the `skillCoverageAllowlist` rationale flow returns to "ops verb / trivially documented" only. The drift guard in `TestRunSkillCoverageChecks_FullDriftFiresAllAxes` doesn't need updating — it tests the policy mechanism, not the specific verb roster.

## References

- M-0074 — added the skill-coverage policy that allowlists `show` with this gap as the deferral pointer.
- ADR-0006 — *Skills policy: per-verb default; topical multi-verb when concept-shaped; no skill when --help suffices* — documents the four-case judgment rule. Show currently falls under "no skill (deferred)"; this gap names when "deferred" needs to be reconsidered.
- `internal/policies/skill_coverage.go` — `skillCoverageAllowlist["show"]` rationale references this gap by id.
