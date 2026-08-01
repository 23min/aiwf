---
id: G-0496
title: The history:ok escape opens on comments that are not the directive
status: addressed
priority: high
addressed_by_commit:
    - 8e8115c0c
---
## What's missing

The `//history:ok` escape in `comment-history-attrition` opens on comments that are not the directive, silencing a blocking pre-push gate.

`hasHistoryOK` searches for the marker *anywhere* in a comment line and treats whatever follows as the reason. `addedCommentLines` then applies the result to the **whole comment group**: one matching line exempts every line in the group, changed or not. Two consequences, both measured against the real detector with a control arm:

| source | violations |
| --- | --- |
| a doc comment plus `// There used to be a third arm.` | 1 |
| the same, where the doc comment reads `// F is documented; see the history:ok escape in CLAUDE.md.` | 0 |
| the same, where the first line reads `//history:okay` | 0 |

The first row is the control: the phrase is in the trigger set and fires normally. In the second, prose that merely *names* the escape suppresses a genuine finding on a different line. In the third, a longer word opening with the marker's letters is read as the directive with `ay` for a reason — which also defeats the neighbouring rule that a bare marker is not an escape, since appending any single non-space character converts one into an escape carrying a garbage reason.

The two are not equally settled. The suffix collision is unambiguously wrong: no reading of the convention makes `//history:okay` an escape. The prose-mention hole is entangled with a deliberate design choice — mid-comment placement is *pinned* as supported (`{"marker mid-line with reason", "// see below //history:ok supported older release", true}`), so closing it means revisiting that pin rather than correcting an oversight.

A third question sits underneath both: whether the exemption should stay group-scoped. A legitimate escape on a legacy-format note currently silences every other line in the same comment group, which is how one true annotation hides an unrelated defect.

## Why it matters

This is the escape hatch of a gate that blocks pushes. A hole in it is silent in the direction that costs something: the comment lands, the push succeeds, and nothing reports that the rule was skipped. The failure is invisible precisely because the gate's job is to say nothing when the tree is clean.

The prose-mention form is reachable by accident rather than by intent. Documentation about the escape is exactly the kind of comment that names it — this repo's own `CLAUDE.md` and skill bodies discuss the marker in prose, and the same sentence in a Go doc comment would open the hole without anyone choosing to.

## Resolution shape

The suffix collision has a worked answer already in the tree. The sibling `test-executable-write` policy resolved the identical class in `hasExecOK`: require the marker to open the comment, then require whitespace between the marker and the reason, so that a longer word reads as a different comment rather than as the directive. Applying the whitespace half here is a contained change and needs no decision.

The prose-mention hole needs the mid-line placement question settled first:

- **Keep mid-line placement.** The pin stays; the hole narrows only as far as requiring the marker to open a comment *line* within the group, which still admits prose that begins by naming it.
- **Require the marker to open the comment**, matching the sibling. Closes the hole outright, and retires the pinned mid-line case as unsupported.

The group-scoped exemption is a separate call: narrowing it to the annotated line (and the line below, as the sibling does) makes one annotation stop covering unrelated lines, at the cost of repeating the marker on a multi-line legacy note.

Whichever way each goes, the cases belong in the existing matcher table rather than only in the seam test, since that table is where the convention is currently stated.

## Where to fix

- `internal/policies/comment_history_attrition.go` — `hasHistoryOK`, the matcher; and `addedCommentLines`, where the group-scoped exemption is applied.
- `internal/policies/comment_history_attrition_test.go` — the matcher table, including the pinned mid-line case that any placement change has to revisit.
- `internal/policies/test_executable_write.go` — `hasExecOK` and `exemptLines` are the worked precedent for both halves; whatever lands here should leave the two markers behaving the same way, or say why they differ.
