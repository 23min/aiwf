---
id: G-0402
title: Always-on guidance omits the verb-and-template entity-authoring rule
status: addressed
addressed_by_commit:
    - e334f6ce
---
## What's missing

The always-on guidance fragment (`aiwf-guidance.md`, injected every turn) and the `builder` role-agent card carry no rule that entity files are verb- and template-managed. That an entity is created via `aiwf add`, its body filled from the canonical `.claude/templates/<kind>.md`, its prose edited via `aiwf edit-body`, and its frontmatter changed only through structured verbs — and that a file under `work/` is never hand-written and frontmatter never hand-edited — lives only inside the `aiwf-add` / `aiwf-edit-body` / `aiwf-retitle` skills and the `aiwfx-record-decision` ritual. Those are pull surfaces: an agent reads them only after it has already decided to invoke that verb's skill. The specific anti-pattern — authoring a new entity by copying a neighboring one as a template, which drifts from the canonical shape and silently drops the H1/header — is stated only in `aiwf-add`'s *Locating the rich body template* section.

## Why it matters

The failure mode is precisely the agent not reaching for those skills: it opens the entity sitting next to the one it wants, copies the shape, and hand-edits body, header, or frontmatter — never triggering the skill that would have told it not to. The mechanical backstops (`provenance-untrailered-entity-commit`, `unexpected-tree-file`) do catch a committed hand-edit, but only at check / pre-push time — after the wrong work is done and a round-trip is wasted. The two push surfaces seen every turn without the agent choosing to look — the guidance fragment and the builder card — are silent at exactly the moment the discipline is needed. The fix adds a dedicated verb-and-template-managed bullet to the shipped guidance (pinned by a new `PolicyM0211GuidanceOperatingAnchors` anchor) and a one-line reminder to the builder card.
