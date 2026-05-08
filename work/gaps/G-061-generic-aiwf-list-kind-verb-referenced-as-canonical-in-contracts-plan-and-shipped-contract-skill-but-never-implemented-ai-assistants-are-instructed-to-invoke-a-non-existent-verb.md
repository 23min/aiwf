---
id: G-061
title: Generic `aiwf list <kind>` verb referenced as canonical in contracts plan and shipped contract skill, but never implemented; AI assistants are instructed to invoke a non-existent verb
status: addressed
---

## What's missing

Three coupled defects around the unimplemented generic `aiwf list <kind>` verb:

1. **The verb itself.** No `list` command in `cmd/aiwf/`. Users and AI assistants needing to enumerate entities of a kind (e.g. "all open gaps", "all milestones under E-16", "all contracts and their binding state") fall back to `for`-loops over `work/<kind>/` plus `aiwf show` per id, or to filesystem grepping. `aiwf status` is whole-project and unfilterable; `aiwf show <id>` is single-entity. Neither covers the "give me the set" shape.

2. **Documentation drift in the contracts plan.** [`docs/pocv3/plans/contracts-plan.md`](../../docs/pocv3/plans/contracts-plan.md) references `aiwf list contracts` five times (lines 209, 425, 489, 593, 708) as the canonical generic verb, including in the "what's deliberately not included" section, where contract-specific `list` / `status` / `matrix` verbs are explicitly *excluded* on the basis that the generic verb covers these needs. The plan is wrong: the generic verb does not exist, so the contract surface lost a capability without notice.

3. **Skill drift shipped to AI assistants.** [`internal/skills/embedded/aiwf-contract/SKILL.md`](../../internal/skills/embedded/aiwf-contract/SKILL.md) line 33 instructs AI assistants to use `aiwf list contracts` for the "list recipes / contracts" workflow. The skill is materialized into every consumer repo by `aiwf init` / `aiwf update`. Every assistant that consults this skill is told to invoke a verb that returns "unknown command". This is the inverse of the kernel's "AI-discoverability" principle: the skill discovers a kernel capability that isn't there.

## Why it matters

The verb-shape question is non-trivial and warrants thinking before implementation — flag set (`--status`, `--epic`, `--phase`?), output formats (table, JSON, ids-only?), composite-id semantics (does `aiwf list ac --milestone M-NNN` work?), cross-kind shape (`aiwf list --kind milestone` vs. `aiwf list milestones`?). The contracts plan's chosen shape (`aiwf list contracts` — kind name as positional plural) is one option; `aiwf list --kind milestone --status in_progress` is another. A short decision pinning the surface comes before the implementation.

The skill drift is the urgent half: AI assistants have been instructed to invoke a non-existent verb for some unknown number of sessions. The fix path is verb implementation **first**, then the skill becomes truthful; alternatively, an interim patch removing the line from the contract skill would restore truthfulness while the verb is built. Either route should be picked deliberately.

The doc drift is the easy half: once the verb exists with a chosen shape, the contracts plan's references are either correct (matches the shape) or get a one-line fix.

A check rule along the lines of `skill-references-unknown-verb` would prevent recurrence — a regex pass over `internal/skills/embedded/*/SKILL.md` cross-referenced against the registered Cobra verbs at build time. That belongs in a follow-up milestone, not in the gap itself.

