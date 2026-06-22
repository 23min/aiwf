---
id: ADR-0015
title: Settings.json edits require explicit per-invocation consent
status: accepted
---
## Context

aiwf has documented, in CLAUDE.md and in `internal/cli/doctor/doctor.go`, a
firm stance that it never edits `settings.json`. That commitment kept the
"materialize-and-byte-refresh" lifecycle of every shipped artifact (verb
skills, ritual skills, role agents, entity templates, git hooks) confined
to artifacts a consumer's settings file does not need to know about: each
of those artifacts activates passively — Claude Code discovers it via the
on-disk layout, not via a settings-file flag.

E-0039 introduces an exception. Claude Code's `statusLine` is the only
kernel-shipped artifact whose activation requires a key in a settings
file. Without that key the script is dead weight on disk; with it, the
consumer sees the aiwf HUD (epic / milestone / branch / token-ball / CI).
The original "never edits settings.json" stance, applied literally, would
leave the statusline disabled even after a consumer explicitly requests
`aiwf init --statusline` or `aiwf update --statusline`.

## Decision

aiwf does not edit settings files **without explicit per-invocation
consent**. The decision narrows the original "never edits" stance to a
consent-gated stance.

Two consent mechanisms, both gated strictly to the opt-in `--statusline`
flow:

1. **TTY-interactive consent.** When a TTY is present, the `--statusline`
   verb path prompts `[y/N]` before writing to the settings file. Default
   declines; the operator confirms by typing `y`.
2. **Non-TTY explicit consent.** Without a TTY (CI, the common
   Claude-via-Bash invocation pattern, scripts), aiwf refuses to prompt
   and instead requires the operator to pass `--wire-settings` explicitly.
   The flag's presence on the command line is itself the consent record —
   the approving human sees the flag, the binary sees the flag, the
   commit record sees the flag.

Project scope writes to `.claude/settings.local.json` (personal,
gitignored), not the shared `.claude/settings.json`. The shared file
would force a broken statusline — gitignored script, missing on a
teammate's clone — onto every collaborator's HUD. The local-scope
settings file is the only project-scope target that avoids that footgun.

User scope writes to `~/.claude/settings.json`. There is no
`settings.local.json` at user scope; the user's own settings file is the
intended target.

The interactive prompt is gated strictly to the `--statusline` opt-in
flag. No other aiwf verb gains a prompt as a side effect of this
decision. The kernel maintains its existing non-interactive default for
every other verb.

## Consequences

- **CLAUDE.md and `internal/cli/doctor/doctor.go` prose updates.** Both
  surfaces today state the un-narrowed stance; both are amended by
  M-0154's other ACs in the same milestone as this ADR.
- **M-0156 builds on a ratified decision.** The wiring milestone
  implements the consent flow, the no-clobber rule, and the `.bak`
  before-edit guard against an ADR that is already `accepted`. M-0156
  itself does not re-author this stance prose; that ownership lives here
  and in M-0154's other ACs.
- **First interactive prompt in the kernel.** Until E-0039, aiwf was
  fully non-interactive (its only `term.IsTerminal` use was for width
  detection in `internal/render/term.go`). The TTY-confirm path is a new
  pattern; M-0156 must add a small exported `IsTTY` predicate to
  `internal/render/`. The pattern is deliberately scoped to this one
  flag — generalizing it would re-open the stance.
- **Audit trail.** Every consent-driven settings edit lands as a
  trailered commit; `aiwf history` resolves the edit through its
  `aiwf-verb` / `aiwf-actor` trailers. The settings file's prior content
  is preserved at `.bak` so an operator can revert.
- **No-clobber.** If a settings file already carries a `statusLine` key,
  the verb refuses to overwrite it and prints merge guidance. Idempotent
  re-runs (the key already points at our script) are a no-op.
