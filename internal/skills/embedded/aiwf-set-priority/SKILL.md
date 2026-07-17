---
name: aiwf-set-priority
description: Use when the user wants to set, change, or clear a gap or decision's priority — "mark this urgent", "bump the priority", "this isn't that important, lower it", "clear the priority tag". Runs `aiwf set-priority` so the frontmatter mutation rides through a verb route with proper trailers, instead of a hand-edit that triggers a `provenance-untrailered-entity-commit` warning.
---

# aiwf-set-priority

The `aiwf set-priority` verb rewrites the frontmatter `priority:` of a single gap or decision to a closed-set level, or clears it back to unset, in one atomic commit. Priority is a backlog-triage signal — it says how urgently the entity deserves attention, independent of its status.

## When to use

Triggers:
- *"mark this urgent"* / *"this is high priority"*
- *"bump the priority"* / *"lower the priority"*
- *"deprioritize this"*
- *"clear the priority"* / *"unset the priority tag"*

Priority can also be set at creation time with `aiwf add gap|decision --priority <level>` — see the `aiwf-add` skill. Reach for `aiwf set-priority` to change or clear it afterward.

## What to run

```bash
# Set (or change) a gap's or decision's priority
aiwf set-priority <id> <level>

# Clear it back to unset
aiwf set-priority <id> --clear
```

`<level>` is one of the closed set: `urgent`, `high`, `medium`, `low`. `<level>` and `--clear` are mutually exclusive — pass exactly one.

## What aiwf does

1. Looks up the entity by id; refuses an unknown id.
2. Refuses a target whose kind doesn't carry a priority — only gap and decision do.
3. Refuses `<level>` outside the closed set, naming the allowed levels.
4. Refuses a no-op: setting the level it's already at, or `--clear` on an entity that's already unset.
5. Rewrites the `priority:` frontmatter key (or, with `--clear`, drops it entirely — the on-disk shape returns to exactly the unset state) and creates one commit with `aiwf-verb: set-priority`, `aiwf-entity: <id>`, `aiwf-actor: <actor>` trailers.

The change reverses totally through the same verb: a set reverses with `--clear`; a change from one level to another reverses by setting the prior level back.

## Don't

- Don't hand-edit `priority:` in frontmatter to "skip the verb" — `aiwf history` won't show the change and the next `aiwf check` will surface `provenance-untrailered-entity-commit`.
- Don't run `set-priority` on an epic, milestone, ADR, or contract — priority is scoped to gap and decision only; the verb refuses with a clear message naming the two carrying kinds.
- Don't confuse priority with status. Status tracks lifecycle (open, met, done, …); priority tracks urgency. They're independent axes.
