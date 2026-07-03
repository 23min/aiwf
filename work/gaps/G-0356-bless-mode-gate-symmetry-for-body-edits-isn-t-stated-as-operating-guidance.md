---
id: G-0356
title: Bless-mode gate-symmetry for body edits isn't stated as operating guidance
status: open
---
## What's missing

The shipped gate-discipline guidance (`.claude/aiwf-guidance.md`, materialized
into consumer repos) says every aiwf mutation is its own approval gate, but it
never resolves the structural asymmetry between plain file edits and aiwf
mutating verbs:

- A plain `Edit`/`Write` touches the **working tree only** — reversible, the
  diff is reviewable, and the commit is a *separate later step* gated on its own.
- An aiwf mutating verb (`promote`, `cancel`, `archive`, `reallocate`,
  `edit-body`, `retitle`, `rename`) is **mutation + commit atomically**
  (commitment #7). There is no working-tree window; running the verb *is* the
  commit, so the gate must sit before invocation on stated intent, not on a
  reviewable staged diff.

For **body edits specifically** the asymmetry is avoidable but the guidance
doesn't say so. `aiwf edit-body <id>` bless mode (G-0054 -> M-0060) diffs the
working copy against HEAD and commits whatever body change is already on disk.
Used that way it restores the plain-file rhythm: edit the entity file with a
plain editor (working-tree only, no gate), let the human review the real diff,
then gate one `aiwf edit-body <id>` that commits exactly what they saw. The
`--body-file` mode instead fuses edit-and-commit, so the human approves a
*description* of new content rather than the on-disk diff.

Nothing in the shipped guidance or the `aiwf-edit-body` skill states "prefer
bless mode for the in-conversation human workflow so body edits keep a
review-before-commit window." An assistant reaching for `--body-file` by default
gives up the review window for no reason.

## Why it matters

The gate exists so a human approves a change on the evidence, not on a promise.
`--body-file` degrades that: the human approves the assistant's account of the
new body, sight-unseen, and the verb commits the instant it runs. Bless mode
recovers the exact review-before-commit shape plain files have for free, at zero
extra ceremony — it is strictly the better default for body edits made during a
conversation, and the guidance should say so.

This is operating guidance, not a mechanical gap: there is no chokepoint that
could enforce "prefer bless mode," and none is wanted. The fix is a short rule
in the shipped guidance source (`internal/skills/embedded-guidance/`) and/or the
`aiwf-edit-body` skill body, drawing the plain-edit / verb-commit distinction
and naming bless mode as the default for interactive body edits. G-0054 already
built the affordance; this gap only asks that the *when-to-use-it* be written
down where a consumer assistant will read it.
