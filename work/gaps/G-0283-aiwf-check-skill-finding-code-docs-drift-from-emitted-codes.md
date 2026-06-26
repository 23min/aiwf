---
id: G-0283
title: aiwf-check skill finding-code docs drift from emitted codes
status: open
---
## What's missing

The `aiwf-check` skill (`internal/skills/embedded/aiwf-check/SKILL.md`) exists to document the finding codes `aiwf check` emits, but its documented set has drifted from the codes the check layer actually produces. Emitted-but-undocumented codes:

- `milestone-done-incomplete-acs` (wired `internal/check/check.go:117`) — notably *referenced by the aiwf-promote skill*, so the check skill is the odd one out.
- `acs-title-prose` (`internal/check/acs.go:224`)
- `id-path-consistent` (`internal/check/check.go:445`)
- `isolation-escape` plus subcodes `isolation-escape-shallow-clone` / `-oracle-failure` / `-orphaned-ai-commit` (`internal/check/isolation_escape.go`). Note CLAUDE.md still describes this finding as "remains open" under E-0019, but it is wired and firing.
- `promote-on-wrong-branch` (`internal/check/promote_on_wrong_branch.go`)
- `git-config-core-worktree-misset` (`internal/cli/check/git_config.go`)

No mechanical chokepoint binds the skill's documented code set to the emitted set, so a new finding code can ship undocumented and nothing notices. The drift is one-directional (omissions only — every documented code is still emitted).

Minor, sweep in the same patch: the `aiwf-list` skill describes ADR-0004 as "(proposed)" and "once that ADR ships"; ADR-0004 is `accepted` and the archive convention has shipped.

## Why it matters

The check skill is the channel an AI assistant (or operator) consults to interpret a finding it didn't recognize — "what does this code mean, how do I fix it." An undocumented code leaves the reader without that guidance, defeating the skill's reason to exist. `isolation-escape` and `promote-on-wrong-branch` are exactly the high-stakes branch-choreography findings where the interpretation guidance matters most, and they are among the undocumented ones.

## Proposed fix

1. **Document the missing codes** in the aiwf-check skill (and fix the aiwf-list ADR-0004 status nit).
2. **Add the missing chokepoint:** a policy test asserting the skill's documented code set is a superset of the emitted `Code*` constants under `internal/check/` plus the typed `codespkg.Code` descriptors, with an explicit opt-out for any deliberately-internal code. This pins the skill to the emission sites so new finding codes can't ship undocumented.
