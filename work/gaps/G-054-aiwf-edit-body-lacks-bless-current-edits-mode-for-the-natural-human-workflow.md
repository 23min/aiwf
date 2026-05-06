---
id: G-054
title: aiwf edit-body lacks bless-current-edits mode for the natural human workflow
status: open
discovered_in: M-058
---
## Problem

`aiwf edit-body` (shipped in M-058) accepts only `--body-file <path>` or `--body-file -` (stdin) — i.e., the caller must supply *new* body content from outside the working copy. The natural human workflow is the inverse: the user opens the entity file in their editor, edits a paragraph or an AC body section in place, saves, and wants the change committed with proper trailers. There is no verb path for "bless what I just edited"; the only routes are:

- Re-type or copy-paste the entire edited body into a temp file and run `aiwf edit-body <id> --body-file /tmp/edit.md`. High friction; loses context if the user already had the file open in their editor.
- Run plain `git commit` and accept the `provenance-untrailered-entity-commit` warning — the very workflow M-058 was supposed to replace.

Net effect of M-058 for human editors: friction up, not down. The skill carve-out is gone; the verb that was supposed to replace it doesn't match the workflow it was supposed to cover.

## Evidence

During M-058's closure review (this conversation, 2026-05-06), I asked "did we build the wrong thing?" and walked through the workflow inventory:

1. Drafting body content elsewhere (LLM session, scripted markdown gen) — covered by `--body-file`.
2. Editing in place in $EDITOR — **not covered**.
3. Append/prepend a single paragraph — **not covered** (replace-only semantics).
4. Update a single AC body section under `### AC-N — title` — **not covered** (composite ids refused; user must rewrite the entire milestone body including all AC headings).

Workflow 1 was over-served (it has a verb route now). Workflows 2/3/4 were under-served — each requires retyping content the user already has on disk.

## Root cause

M-058's ACs had an internal tension we didn't catch at planning time:

- **AC-1** specified the verb shape literally: "aiwf edit-body verb exists; accepts --body-file or stdin."
- **AC-3** specified the user-facing outcome: "aiwf-add skill text removes plain-git body-edit carve-out."

Implementing AC-1 to the letter (which we did) does not deliver AC-3's spirit — the plain-git workflow that the carve-out covered isn't the same shape as `--body-file`. We built what AC-1 specified literally; AC-3 needed a different shape (or a second one) to actually replace the workflow.

A deeper reading: the milestone was framed in verb-flag terms (mirroring M-056's `aiwf add --body-file`), so we built a symmetric verb-flag affordance. The symmetry was a design instinct, not a workflow analysis — and the workflows on each side of the symmetry are different (create-from-scratch vs. modify-in-place).

## Direction

Add a "bless current edits" mode to `aiwf edit-body`:

```
aiwf edit-body <id>          # diff working-copy vs HEAD on the entity file;
                             # if frontmatter changed, refuse (use promote/rename/cancel);
                             # if body changed, stage+commit with edit-body trailers;
                             # if nothing changed, refuse with "no changes to commit".
```

Composes naturally with the existing `--body-file` mode:

- `aiwf edit-body <id>` — bless the on-disk edit (the human workflow).
- `aiwf edit-body <id> --body-file <path>` — supply new content (the AI/script workflow, what M-058 shipped).

Both produce the same trailered commit; both refuse leading-`---` content; both leave frontmatter the domain of structured-state verbs.

Bless mode also covers AC sub-section editing for free: the user edits the prose under one AC heading, runs `aiwf edit-body M-NNN`, and the verb commits whatever changed. No composite-id resolver needed.

## Relationship to M-058 and G-052

M-058 partially addressed G-052 — it covered the AI/script workflow but not the natural human workflow. G-052 itself is correctly marked `addressed` (the skill carve-out is gone, the rule contradiction is resolved), but the broader user-experience promise — *"the verb covers the workflows the carve-out used to cover"* — was not fully delivered.

This gap is the follow-up to M-058's closure review. The fix is additive to what M-058 shipped (no removals, no API breaks), and the existing tests stay valid.

## Considered alternatives

- **Composite-id `aiwf edit-body M-NNN/AC-N --body-file <path>`** for AC sub-sections only. Solves workflow 4 but not 2 or 3. More code (regex-anchored section finder), more failure modes (heading drift, sub-headings inside an AC). The bless-mode shape solves all three workflows with one path.
- **`aiwf edit-body --in-editor`** (open `$EDITOR` with the current body, capture the result, commit). Works but adds an editor-handling layer the kernel doesn't need; bless mode covers the same use case by trusting the user's existing editor session.
- **Restore the carve-out** — admit M-058 was premature, revert AC-3's skill change. Cleanest in "don't ship strictness before the replacement is complete" terms, but undoes work that's substantively useful for workflow 1 (AI/script). Bless mode keeps M-058's value and closes the workflow gap.

## Resolution

M-060 ships bless mode as designed in *Direction* above — `aiwf edit-body <id>` (no `--body-file`) commits whatever the user edited in the working copy of the entity file. This very edit was committed via bless mode against the kernel's own repo, dogfooding the new affordance end-to-end.
