---
id: G-0183
title: aiwf has no install path for its aiwf-aware Claude Code statusline
status: open
---
## What's missing

aiwf materializes every framework artifact it ships — verb skills, ritual
skills, role agents, entity templates, git hooks — into a consumer repo via
`aiwf init` / `aiwf update`. The one artifact it does **not** ship is
`.claude/statusline.sh`, even though that script is the only aiwf-*aware*
surface among them: it derives the in-flight epic/milestone/gap from the
ritual branch shapes (`milestone/M-…`, `epic/E-…`, `patch/[Gg]-…`), resolves
them against the `work/epics` / `work/gaps` layout, colors each id by its
frontmatter `status:`, counts sibling ritual worktrees (`+N⎇`), and renders
the context-window token ball. A downstream aiwf consumer has no path to
obtain it short of copying the file out of this repo by hand.

Three things block a clean "just ship it":

1. **Activation crosses a documented stance.** A statusline does nothing until
   `"statusLine"` is wired into a Claude Code settings file — and aiwf has a
   firm, written stance (doctor.go comment + CLAUDE.md) that it *never* edits
   `settings.json`. This is the first artifact aiwf would ship that needs a
   settings edit to function. Every other artifact activates passively.

2. **Portability defects.** `tac` (token transcript walk) is absent on stock
   macOS, and the ahead/behind sync parse depends on a literal tab surviving
   in the source file (`${counts%%<TAB>*}`) — which dies under editor
   tab→space, copy-paste, or a retab. Both fail soft (zeroed tokens, dropped
   sync indicator) but silently produce wrong output on Darwin / after any
   reflow.

3. **Lifecycle mismatch.** Skills/hooks use materialize-and-byte-refresh
   (clobbered every `aiwf update`). A statusline is a tweakable cosmetic;
   clobbering a user's customizations on every upgrade is hostile. It needs a
   scaffold-once-user-owns lifecycle — the lone exception to the refresh set.

## Why it matters

The statusline is the only aiwf HUD, and its value is currently locked inside
the framework's own dev repo. Downstream consumers running the rituals get no
at-a-glance view of which entity their branch maps to, its status color, CI
state, token budget, or parallel-worktree activity — exactly the situational
awareness the script was built to provide. "Copy it out by hand" also orphans
maintenance: the portability fixes above never reach a hand-copied snapshot,
and there is no drift signal telling a consumer their copy is stale.

## Direction (for the epic)

Design worked out in conversation; recorded here so it survives into planning:

- **Ship vehicle:** embed the script in the binary (`go:embed`, like the
  rituals) but **exclude it from the unconditional refresh set** — embedded so
  bug fixes travel with the binary and `aiwf doctor` can detect on-disk drift;
  scaffold-once so the user's tweaks are never clobbered.
- **Install:** an opt-in `--statusline` flag on `aiwf init` and `aiwf update`
  (a flag, not a new verb — no new skill-coverage surface). Writes the script
  only if absent; never clobbers. Bare `aiwf update` never touches it.
- **Scope (`--scope project|user`, default `project`):** the script is
  stateless and cwd-relative, so one copy serves any number of
  worktrees/sessions. Host → project-scope (`<repo>/.claude/`, gitignored,
  relative command path) to avoid touching the developer's real global config.
  Devcontainer → user-scope (`~/.claude/`) is better: the container home is
  disposable and one install covers every worktree (gitignored project-scope
  files don't survive `git worktree add`, forcing per-worktree re-installs).
  No env-sniffing magic — the choice is explicit; `aiwf doctor` *advises*
  user-scope when it detects a container.
- **Activation (the stance amendment):** wiring requires explicit
  per-invocation consent — an interactive `[y/N]` confirm when a TTY is present
  (aiwf's first interactive prompt), or an explicit `--wire-settings` flag when
  not (the common Claude-via-Bash path, where the approving human sees the flag
  on the command). Project-scope wires into `.claude/settings.local.json`
  (personal, gitignored — never the shared `settings.json`, which would force a
  broken statusline on teammates who lack the gitignored script). User-scope
  wires into `~/.claude/settings.json`. Refuse to clobber a pre-existing
  `statusLine` key; idempotent if already ours. Amend the documented stance to
  "aiwf does not edit settings without explicit per-invocation consent" — record
  via ADR + CLAUDE.md + the doctor.go comment.
- **Portability:** `tac "$f"` → `tail -r "$f" 2>/dev/null || tac "$f"`; replace
  the literal-tab sync parse with `read -r ahead behind <<<"$counts"`
  (IFS-driven, whitespace-agnostic, Linux + Darwin identical, survives retab).
- **doctor:** when the statusline is installed, advisory-only reports for
  missing `jq` (load-bearing) / `gh` (CI segment) with platform install hints
  (`brew` vs `apt-get`, branched on `runtime.GOOS`), installed-but-not-wired
  state (prints the snippet), embedded-vs-on-disk drift, and the container
  user-scope nudge.

Suggested milestone sequence for the epic: (1) portability fixes; (2) ADR for
the settings stance amendment (gates wiring); (3) embed + `--statusline`
scaffold with `--scope`; (4) consented wiring into settings(.local).json;
(5) doctor block.
