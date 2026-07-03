---
id: G-0354
title: aiwf update --statusline has no way to remove or migrate a scope's wiring
status: open
---
## What's missing

`aiwf update --statusline --scope <user|project>` only ever writes to the scope
you name — it has no counterpart for removing a scope's wiring. Concretely:

- It never touches the *other* scope, so switching from project to user scope
  (or vice versa) leaves the old scope's `statusLine` key and script file in
  place. Nothing detects or cleans up the resulting shadow.
- `aiwf doctor` *does* detect the resulting conflict (`precedence: a statusLine
  is wired in BOTH project and user settings — the project key wins and
  shadows the user one`) and correctly identifies which entry is stale, but
  only prints prose telling the operator to fix it by hand — no verb executes
  the fix.
- `--wire-settings` also refuses to overwrite an *existing* `statusLine` key at
  the target scope, even when the existing value is just a stale variant of
  aiwf's own prior write (e.g. `~/.claude/statusline.sh` vs. the
  currently-preferred `$HOME/.claude/statusline.sh`). It prints the
  replacement it wants applied, but never applies it.

End to end, moving a statusline from one scope to the other today requires:
manually `rm`-ing the stale script file, hand-editing the stale scope's
settings JSON to drop the key, and hand-applying whatever replacement
`aiwf update` printed for the target scope. None of that goes through a
documented aiwf verb.

## Why it matters

- `aiwf doctor` is the tool's own health-check surface for exactly this kind
  of drift, but the fix it prescribes has no mechanical path — a human has to
  hand-edit JSON and delete files instead of running an aiwf command. That
  breaks the "aiwf mutations happen through aiwf verbs" pattern the rest of
  the tool follows (promote, cancel, rename, edit-body all avoid hand-edits
  for exactly this reason).
- It's an easy trap to fall into: install at one scope, later switch to the
  other, and both stay wired with no command to resolve it — only `doctor`
  ever notices, and only after the fact.

## Proposed shape

Add a `--remove` flag, using the existing `--scope` flag as the target
(symmetric with how `--scope` already selects where `--statusline` writes):

    aiwf update --scope project --remove   # delete .claude/statusline.sh + strip the project statusLine key
    aiwf update --scope user --remove      # delete ~/.claude/statusline.sh + strip the user statusLine key

Migrating scopes becomes two explicit, single-target commands:

    aiwf update --statusline --scope user --wire-settings
    aiwf update --scope project --remove

This deliberately does **not** collapse into one implicit bidirectional
`--migrate` (`--scope X` = "install X, delete the other one"). The two
directions have very different blast radii — project scope is repo-local,
user scope backs every other repo on the machine/container — so a single
command that silently deletes a scope you didn't name is a worse footgun than
the one it fixes. Two explicit single-scope commands avoid that asymmetry.

`--remove` should refuse to act unless the target actually looks
aiwf-authored — check the script's own version marker
(`# aiwf-statusline version: ... — managed by aiwf`) and/or the settings value
against the known aiwf-written path pattern — and require `--force` to
override, consistent with the `--force` idiom already used elsewhere in the
CLI (`contract bind --force`, `contract recipe install --force`). This avoids
deleting a hand-customized statusLine command someone wired outside aiwf.
