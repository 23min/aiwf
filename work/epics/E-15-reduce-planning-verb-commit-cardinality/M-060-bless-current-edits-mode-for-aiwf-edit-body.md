---
id: M-060
title: Bless-current-edits mode for aiwf edit-body
status: in_progress
parent: E-15
acs:
    - id: AC-1
      title: Bare aiwf edit-body commits on-disk body changes with edit-body trailers
      status: met
    - id: AC-2
      title: Frontmatter-changed bless invocation refuses with structured-state-verb pointer
      status: met
    - id: AC-3
      title: No-diff invocation refuses cleanly without an empty commit
      status: met
    - id: AC-4
      title: M-058 --body-file mode continues to work unchanged
      status: open
    - id: AC-5
      title: AC sub-section edits land via bless mode on the parent milestone
      status: open
---
## Goal

Add a "bless current edits" mode to `aiwf edit-body` so the natural human workflow — edit the entity file in $EDITOR, then commit through a verb route — has a verb path. Composes with the `--body-file` mode M-058 shipped without breaking it. Closes **G-054**.

## Approach

`aiwf edit-body <id>` (no `--body-file` flag) becomes the bless-mode entry. The verb:

1. Diffs the working-copy of the entity file against HEAD.
2. If nothing changed, refuses with "no changes to commit on `<id>`".
3. Splits both versions into frontmatter + body. If the frontmatter differs, refuses with a clear message pointing at `aiwf promote` / `aiwf rename` / `aiwf cancel` / `aiwf reallocate` for structured-state edits. (The kernel's FSM-on-frontmatter-fields rule still applies; bless mode is body-only by design.)
4. If only the body differs, validates the projected tree (same `acs-body-coherence` rule as today catches malformed AC headings), stages the file, and commits with the standard `edit-body` trailer set.

The existing `--body-file <path>` and `--body-file -` modes stay exactly as M-058 shipped — same trailers, same atomicity, same rules. Only difference: when `--body-file` is absent, bless mode runs instead of refusing on missing flag.

AC sub-section editing (the workflow-4 case from G-054) is covered by bless mode for free: a user edits the prose under `### AC-N — title` in the milestone file, runs `aiwf edit-body M-NNN`, and the verb commits whatever changed. No composite-id resolver needed; no per-section parsing.

## Out of scope

- **Composite-id `aiwf edit-body M-NNN/AC-N`**. Still refused. Editing an AC sub-section is done via bless mode on the parent milestone (workflow 4 from G-054); a sub-section verb is unnecessary once bless mode is in place.
- **`--in-editor` mode** (open `$EDITOR` then commit). Bless mode covers the same use case by trusting the user's existing editor session; an aiwf-spawned editor adds a layer the kernel doesn't need.
- **Partial frontmatter mutations through `edit-body`**. Frontmatter stays the domain of structured-state verbs. If the diff includes any frontmatter change, the verb refuses; the user routes the edit through the right verb instead.

## Implementation notes

The git-diff comparison wants a stable "what's HEAD's version of this path" read. `gitops` already exposes shaped reads (e.g., for the trunk-walk path); a small `gitops.ReadFromHEAD(ctx, root, path) ([]byte, error)` helper (or equivalent) is the right shape — returns nil-bytes when the file is new (so callers can decide whether bless on a new file is valid; for now it is not, since `aiwf add --body-file` covers the new-file case).

The existing `validateUserBodyBytes` helper (extracted in M-058) is reused — body content from the working copy can no more begin with `---` than body content from a `--body-file`. Same shared rule.

### AC-1 — Bare aiwf edit-body commits on-disk body changes with edit-body trailers

### AC-2 — Frontmatter-changed bless invocation refuses with structured-state-verb pointer

### AC-3 — No-diff invocation refuses cleanly without an empty commit

### AC-4 — M-058 --body-file mode continues to work unchanged

### AC-5 — AC sub-section edits land via bless mode on the parent milestone

