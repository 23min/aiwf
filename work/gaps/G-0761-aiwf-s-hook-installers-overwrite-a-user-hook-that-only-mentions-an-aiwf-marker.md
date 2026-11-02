---
id: G-0761
title: aiwf's hook installers overwrite a user hook that only mentions an aiwf marker
status: open
priority: high
---
## What's missing

The hook installers decide whether an existing hook is aiwf's own with a
substring test. `ensurePreHook` (`internal/initrepo/initrepo.go`) overwrites a
`pre-push` hook when `strings.Contains(string(existing), preHookMarker)`, and
otherwise moves it to `pre-push.local` ("non-aiwf hook in place: auto-migrate to
pre-push.local (G45)"), or skips with a conflict when that file already exists.
The pre-commit, commit-msg and post-commit installers in the same file test
their markers the same way. `aiwf init`, `aiwf update` and `aiwf worktree add`
(`internal/cli/worktree/worktree.go`, through `initrepo.RefreshArtifacts`) all
reach the four through `refreshArtifacts`. `aiwf doctor`
(`internal/cli/doctor/doctor.go`) decides the same question with the same
`strings.Contains`. So a user's hook that only mentions a marker, in a comment
or a string, is taken for aiwf's own, though the marker constant's comment calls
it "the exact comment line `aiwf init` writes".

The guidance-import marker is matched by line instead: `GuidanceMarkerLineIdx`,
in the same file, matches a line that, trimmed, equals the marker, so that "user
text mentioning the markers is never treated as a block boundary". Its comment
says `aiwf doctor` shares that check, "the same pattern as the hook markers
(HookMarker et al.)"; the hook markers share only the marker string, and each
caller tests it with `strings.Contains`.

Reproduced 2026-11-01 with aiwf built from `main` at `c06ffdaf9` (`vcs.modified=false`), git 2.51.1, Linux, each run in a fresh repository whose `aiwf.yaml` held `hosts: []` and
`guidance: {enabled: false}` before `aiwf init`, and the user's own `pre-push` written
over the one `aiwf init` installed. The two runs differ only in whether one
comment line of that hook mentions the marker:

```
$ printf '#!/bin/sh\n# My team hook. It runs before aiwf'"'"'s own, which carries # aiwf:pre-push.\necho team-check\n' > .git/hooks/pre-push && chmod +x .git/hooks/pre-push
exit 0
$ cat .git/hooks/pre-push
#!/bin/sh
# My team hook. It runs before aiwf's own, which carries # aiwf:pre-push.
echo team-check
$ aiwf doctor 2>&1 | grep -E '^(hook|pre-commit|commit-msg|post-commit):'
hook:         aiwf-managed but malformed (no exec line found); run `aiwf init` to refresh
pre-commit:   ok (resolves to /tmp/claude-1000/-workspaces-aiwf/eb2423dd-a702-4bd8-a308-1404db3d3991/scratchpad/e6v2/bin/aiwf)
commit-msg:   ok (resolves to /tmp/claude-1000/-workspaces-aiwf/eb2423dd-a702-4bd8-a308-1404db3d3991/scratchpad/e6v2/bin/aiwf)
post-commit:  ok (resolves to /tmp/claude-1000/-workspaces-aiwf/eb2423dd-a702-4bd8-a308-1404db3d3991/scratchpad/e6v2/bin/aiwf)
$ aiwf update
Hosts (configured): none
  preserved  aiwf.yaml (legacy actor strip)
  preserved  aiwf.yaml (legacy aiwf_version strip)
  updated    aiwf.example.yaml  (regenerated from the current schema)
  preserved  .gitignore
  updated    .git/hooks/pre-push  (exec `command -v aiwf` check (PATH-relative))
  updated    .git/hooks/pre-commit  (exec `command -v aiwf` check --shape-only (PATH-relative))
  updated    .git/hooks/commit-msg  (exec `command -v aiwf` check --commit-msg "$1" (PATH-relative))
  updated    .git/hooks/post-commit  (exec `command -v aiwf` status --format=md (PATH-relative))

aiwf update: done.

Guidance:
  selected:  not adopted (guidance.packs unset)
  maintenance disabled; installed policy remains in use
  adopted:   none
  installed: none
  ignored:   none
exit 0
$ grep -c team-check .git/hooks/pre-push
0
$ ls .git/hooks | grep pre-push
pre-push
pre-push.sample
$ grep -c team-check .git/hooks/pre-push.local
grep: .git/hooks/pre-push.local: No such file or directory
```

```
$ printf '#!/bin/sh\n# My team hook. It runs before aiwf'"'"'s own.\necho team-check\n' > .git/hooks/pre-push && chmod +x .git/hooks/pre-push
exit 0
$ cat .git/hooks/pre-push
#!/bin/sh
# My team hook. It runs before aiwf's own.
echo team-check
$ aiwf doctor 2>&1 | grep -E '^(hook|pre-commit|commit-msg|post-commit):'
hook:         present but not aiwf-managed (no `# aiwf:pre-push` marker); aiwf check is not running pre-push
pre-commit:   ok (resolves to /tmp/claude-1000/-workspaces-aiwf/eb2423dd-a702-4bd8-a308-1404db3d3991/scratchpad/e6v2/bin/aiwf)
commit-msg:   ok (resolves to /tmp/claude-1000/-workspaces-aiwf/eb2423dd-a702-4bd8-a308-1404db3d3991/scratchpad/e6v2/bin/aiwf)
post-commit:  ok (resolves to /tmp/claude-1000/-workspaces-aiwf/eb2423dd-a702-4bd8-a308-1404db3d3991/scratchpad/e6v2/bin/aiwf)
$ aiwf update
Hosts (configured): none
  preserved  aiwf.yaml (legacy actor strip)
  preserved  aiwf.yaml (legacy aiwf_version strip)
  updated    aiwf.example.yaml  (regenerated from the current schema)
  preserved  .gitignore
  migrated   .git/hooks/pre-push  (existing hook moved to .git/hooks/pre-push.local; aiwf's chain-aware hook now runs it before `command -v aiwf` check)
  updated    .git/hooks/pre-commit  (exec `command -v aiwf` check --shape-only (PATH-relative))
  updated    .git/hooks/commit-msg  (exec `command -v aiwf` check --commit-msg "$1" (PATH-relative))
  updated    .git/hooks/post-commit  (exec `command -v aiwf` status --format=md (PATH-relative))

aiwf update: done.

Guidance:
  selected:  not adopted (guidance.packs unset)
  maintenance disabled; installed policy remains in use
  adopted:   none
  installed: none
  ignored:   none
exit 0
$ grep -c team-check .git/hooks/pre-push
0
$ ls .git/hooks | grep pre-push
pre-push
pre-push.local
pre-push.sample
$ grep -c team-check .git/hooks/pre-push.local
1
```

With the mention, `aiwf doctor` calls the user's hook aiwf-managed and malformed
and says to run `aiwf init`, and `aiwf update` reports `updated` and replaces
it: `team-check` is gone and no `pre-push.local` exists. Without it, `aiwf
doctor` reports the hook as not aiwf-managed, and `aiwf update` moves it to
`pre-push.local`, where `team-check` survives.

The other hooks, `aiwf worktree add`, and the post-commit hook with
`status_md.auto_update: false`, in the same environment (`<scratch>` stands for
the scratch directory):

```
--- pre-commit
$ printf '#!/bin/sh\n# Team hook; aiwf'"'"'s own carries # aiwf:pre-commit.\necho team-check\n' > .git/hooks/pre-commit && chmod +x .git/hooks/pre-commit
exit 0
$ aiwf doctor 2>&1 | grep -E '^pre-commit:'
pre-commit:   aiwf-managed but malformed (no aiwf invocation found); run `aiwf update` to refresh
$ aiwf update | grep -F '.git/hooks/pre-commit'
  updated    .git/hooks/pre-commit  (exec `command -v aiwf` check --shape-only (PATH-relative))
exit 0
$ grep -c team-check .git/hooks/pre-commit; ls .git/hooks | grep -c '^pre-commit.local$'
0
0
--- commit-msg
$ printf '#!/bin/sh\n# Team hook; aiwf'"'"'s own carries # aiwf:commit-msg.\necho team-check\n' > .git/hooks/commit-msg && chmod +x .git/hooks/commit-msg
exit 0
$ aiwf doctor 2>&1 | grep -E '^commit-msg:'
commit-msg:   ok (resolves to <scratch>/e6v2/bin/aiwf)
$ aiwf update | grep -F '.git/hooks/commit-msg'
  updated    .git/hooks/commit-msg  (exec `command -v aiwf` check --commit-msg "$1" (PATH-relative))
exit 0
$ grep -c team-check .git/hooks/commit-msg; ls .git/hooks | grep -c '^commit-msg.local$'
0
0
--- post-commit
$ printf '#!/bin/sh\n# Team hook; aiwf'"'"'s own carries # aiwf:post-commit.\necho team-check\n' > .git/hooks/post-commit && chmod +x .git/hooks/post-commit
exit 0
$ aiwf doctor 2>&1 | grep -E '^post-commit:'
post-commit:  aiwf-managed but malformed (no aiwf invocation found); run `aiwf update` to refresh
$ aiwf update | grep -F '.git/hooks/post-commit'
  updated    .git/hooks/post-commit  (exec `command -v aiwf` status --format=md (PATH-relative))
exit 0
$ grep -c team-check .git/hooks/post-commit; ls .git/hooks | grep -c '^post-commit.local$'
0
0
--- worktree add
$ printf '#!/bin/sh\n# Team hook; aiwf'"'"'s own carries # aiwf:pre-push.\necho team-check\n' > .git/hooks/pre-push && chmod +x .git/hooks/pre-push
exit 0
$ aiwf worktree add patch/some-fix --base main | grep -F 'hooks/pre-push'
  updated    ../../../../.git/hooks/pre-push  (exec `command -v aiwf` check (PATH-relative))
exit 0
$ grep -c team-check .git/hooks/pre-push; ls .git/hooks | grep -c '^pre-push.local$'
0
0
--- post-commit, status_md.auto_update: false
$ printf '#!/bin/sh\n# Team hook; aiwf'"'"'s own carries # aiwf:post-commit.\necho team-check\n' > .git/hooks/post-commit && chmod +x .git/hooks/post-commit
exit 0
$ aiwf doctor 2>&1 | grep -E '^post-commit:'
post-commit:  present (aiwf-managed) but config says off (status_md.auto_update: false); run `aiwf update` to remove
$ aiwf update | grep -F '.git/hooks/post-commit'
  removed    .git/hooks/post-commit  (status_md.auto_update: false (removed previously-installed post-commit hook))
exit 0
$ ls .git/hooks | grep -c '^post-commit$'
0
```

## Why it matters

`aiwf init`, `aiwf update` and `aiwf worktree add` keep a user's own hook by
moving it to `.local` and chaining it from aiwf's. A hook that mentions aiwf's
marker, for example in a comment about how the two relate, is replaced instead,
with no copy kept and the step reported as `updated`, and whatever it ran stops
running; with `status_md.auto_update: false` the post-commit hook is deleted.
`aiwf worktree add` is how this repository's own guidance says to create every
branch, so the loss can come from routine work rather than an upgrade.

`aiwf doctor`, asked about such a hook beforehand, gets it wrong in both
directions. For `pre-push`, `pre-commit` and `post-commit` it calls the hook
aiwf-managed and recommends `aiwf init` or `aiwf update`, the commands that
destroy it. For `commit-msg` it reports `ok`, while the check that refuses
fabricated aiwf trailers is not installed.

## Relates to

G-0472 records the four hook installers as near-identical functions that each
"compare a marker"; the substring test is in each copy.
