---
id: G-0191
title: wf-patch and siblings hardcode PR-based flow; project-flow belongs in CLAUDE.md
status: addressed
addressed_by_commit:
    - 1e529e5f
---
## What's missing

`wf-patch`'s skill body hardcodes a PR-based merge flow as the audit-trail mechanism — step 8 is a mandatory "PR gate" that gates merge behind opening a PR, the anti-pattern catalog explicitly forbids *"no need for a PR"* reasoning, and the description names the ritual *"branch-and-PR"*. The skill assumes every consumer project uses GitHub-style PR review as the merge mechanism. This repo doesn't — `CLAUDE.md` §"Working in this repo" is explicit: *"Trunk-based development on `main` for maintainers. Commit directly to trunk; no PR ceremony, no review queue."*

The conflict surfaced in the G-0129 wf-patch session: after committing and pushing the patch branch, the next prescribed step per the skill was *"PR gate. Confirm with the user before opening the PR."* The user (correctly) pointed out that this isn't how the project works.

Two other skills carry softer versions of the same conflation:

- **`wf-review-code`** — description and triggers assume PR context (*"before a PR is opened"*, *"A PR is open and the reviewer wants a structured walkthrough"*). The skill itself is project-flow-agnostic in its actual review checklist, but its framing reads as PR-centric.
- **`wf-doc-lint`** — *"Use before opening a PR that touches code referenced from docs"*. PR is one trigger, not the only one (a periodic doc-hygiene pass is also valid).

Two skills got this right and are the model:

- **`aiwfx-wrap-milestone`** — *"Open the PR if the project's flow is PR-driven. Reference the milestone id in the PR title."* — conditional on the project's flow, named explicitly.
- **`aiwfx-start-milestone`** — has a conditional fallback for projects that land milestones directly on main via PR vs via an epic-integration branch.

The `aiwf-contract` PR reference is legitimate — it's about contributing recipes upstream to the rituals repo, which IS PR-based as an external collaboration mechanism, not a project-flow prescription.

## Why it matters

The kernel rule is *"framework correctness must not depend on LLM behavior."* The same applies one level down at the skills layer: **ritual correctness must not depend on a hardcoded project policy**. A skill that prescribes a merge mechanism is making a project-specific decision on the consumer's behalf. When the consumer's actual policy disagrees (this repo's trunk-based default), the skill produces wrong guidance and the operator has to override it manually each time.

The right separation is:

- **Skills name the moment**: a focused change is ready (`wf-patch`), a review is wanted (`wf-review-code`), docs may have drifted (`wf-doc-lint`).
- **Skills name the structural shape that survives across project flows**: branch, commit, audit trail, gates for commit and push.
- **The project's `CLAUDE.md` (or per-project policy) names the merge mechanism**: PR-vs-trunk-direct, review queue, branch deletion policy.

The current `wf-patch` collapses these into one prescriptive recipe. That's the bug.

## Resolution shape

Edit the three offending skills in place at the **canonical embedded-rituals snapshot** (post-E-0038 / ADR-0014):

- **Primary edit location:** `internal/skills/embedded-rituals/plugins/wf-rituals/skills/<skill>/SKILL.md` — the embedded copy that `aiwf init` / `aiwf update` materializes into the consumer's `.claude/skills/`. This is what actually ships.
- **Paired edit (until G-0182 lands):** `internal/policies/testdata/<skill>/SKILL.md` — the per-AC content-assertion fixture. Edits must land in the same commit per CLAUDE.md §"Cross-repo plugin testing" until G-0182 collapses this duplication onto the embedded snapshot.
- **Upstream sync:** `rituals.lock`-pinned `23min/ai-workflow-rituals` upstream — refresh via `make sync-rituals` after the matching upstream commit lands. `TestRituals_VendoredMatchesUpstream` will block the local commit if the embedded snapshot diverges from the pinned ref. For pure-prose edits like this one, the upstream commit and the in-repo commit are typically authored together.

### Specific edits

1. **`wf-patch`** — replace step 8's "PR gate" with a project-flow-agnostic *"Merge gate. Confirm with the user before merging the patch back to mainline. The mechanism — open a PR, fast-forward main to the patch branch, cherry-pick onto main, etc. — follows the project's `CLAUDE.md` §"Working in this repo" policy."* Replace the *"every patch goes through a branch and PR"* anti-pattern with *"every patch goes through a branch and an explicit merge — the branch is the audit trail; the merge mechanism is project-specific."* Update the description to drop the *"branch-and-PR ritual"* framing.

2. **`wf-review-code`** — change the trigger language from *"before a PR is opened"* / *"A PR is open"* to *"before the change is proposed for merge"* / *"A change is ready for review (in PR form, on a branch, or in any other shape the project uses)."* The actual review checklist is already flow-agnostic.

3. **`wf-doc-lint`** — change *"Use before opening a PR..."* to *"Use before proposing a change for merge..."* — keeps the trigger meaningful without binding to a mechanism.

### Validation

Each edited skill, parsed structurally, contains no prescriptive *"PR"* / *"pull request"* references except (a) conditional mentions of the form *"if the project's flow is PR-driven"*, or (b) framing-only mentions clearly marked as one shape among others. A drift policy under `internal/policies/` (akin to `PolicyFindingCodeAdoption` in shape) could enforce this if the gap class recurs.

## References

- `CLAUDE.md` §"Working in this repo" — *"Trunk-based development on `main` for maintainers"* — the project policy this repo holds.
- `aiwfx-wrap-milestone` — the existing correct pattern (*"Open the PR if the project's flow is PR-driven"*).
- `aiwfx-start-milestone` — same conditional shape.
- [ADR-0014](../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) — embed-and-materialize rituals (E-0038, done); the edit-shape this resolution follows.
- CLAUDE.md §"Cross-repo plugin testing" — the testdata-fixture-plus-embedded-snapshot edit pattern.
- [G-0182](./G-0182-consolidate-testdata-ritual-fixtures-onto-the-embedded-snapshot-dedupe.md) — when this lands, the testdata-side edit in step 1 above goes away and only the embedded snapshot needs editing.

Discovered during the G-0129 wf-patch session (2026-05-31): the skill's step 8 PR gate prescription conflicted with the project's documented trunk-based policy.
