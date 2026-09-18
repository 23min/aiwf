# Claude artifact compatibility baseline

These inventories pin the complete regular-file output exercised by
`TestClaudeArtifacts_MatchBaseline`: project adapters and manifests, root
instructions, configuration, Git hooks, user settings, and optional statusline.
Each line records an octal permission mode, SHA-256 of the complete file bytes,
and a relative path. `user/` identifies the isolated fixture HOME; `git-hooks/`
identifies the repository's shared hook directory. Lines are sorted lexically.

Six scenarios cover undecided, explicitly enabled, and declined lifecycle hooks;
stored hook consent with agent tiers and guidance opt-out; and statusline
installation with and without settings-wiring consent. The test names, seed
files, configuration, and flags define each input. Personal skills, agents,
templates, settings, Git ignore entries, and root instructions are included so
the comparison also detects damage to user content.

The same inventory must match initial setup, update, and an aiwf-created Git
worktree. Only the user-owned project files are explicitly added to Git before
worktree creation; aiwf must regenerate its ignored artifacts there. User-scope
artifacts and Git hooks are shared by those checkouts.

`.claude/health.aiwf.json` is excluded: its timestamp and diagnostic messages
depend on runtime state and installed tools. No adapter bytes are normalized.
The CLI is stamped `v0.36.0` at build time to make generated version fields
deterministic and exercise tagged-version statusline consent.

## Provenance and maintenance

Captured from the production sources at commit
`488d09bfd44d2ee100f8abd45aa62872925868ad`, before renderer implementation,
using a separate filesystem capture with Node's SHA-256 and file-stat APIs.
The binary was built with:

```sh
go build -ldflags='-X github.com/23min/aiwf/internal/version.Stamp=v0.36.0' -o /tmp/claude-baseline/aiwf ./cmd/aiwf
```

To reproduce a baseline, build that revision in a disposable checkout, apply
the inputs in `TestClaudeArtifacts_MatchBaseline` to isolated consumer repos,
and inventory the files with the format above. The test does not regenerate
expected output and does not derive expectations from current embedded sources.
Review any intentional baseline replacement alongside the output change it
accepts; do not regenerate fixtures just to make a failed compatibility check
pass.
