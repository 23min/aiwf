---
id: G-0725
title: make install-hooks reports success when it cannot create the hook links
status: open
priority: low
discovered_in: M-0359
---
## What's missing

The `install-hooks` target in `Makefile` runs its `mkdir -p` and two `ln -sfn`
commands as one `;`-joined shell line without `set -e`, so a failed step does not stop the
recipe: it prints its "Symlinked ..." lines and exits 0 whether or not the links were made.

Measured in a throwaway clone of this repository (Linux devcontainer, uid 1000, git 2.54.0) whose
hooks directory is not writable:

```sh
chmod a-w .git/hooks
make install-hooks; echo "make install-hooks exit: $?"
ls -la .git/hooks/pre-commit.local
```

Expected a non-zero exit. Observed:

```
ln: failed to create symbolic link '.git/hooks/pre-commit.local': Permission denied
ln: failed to create symbolic link '.git/hooks/pre-push.local': Permission denied
Symlinked scripts/git-hooks/pre-commit -> .git/hooks/pre-commit.local
Symlinked scripts/git-hooks/pre-push   -> .git/hooks/pre-push.local
Run 'aiwf init' (if not already done) so the chain-aware aiwf hooks call them.
make install-hooks exit: 0
ls: cannot access '.git/hooks/pre-commit.local': No such file or directory
```

A `core.hooksPath` naming a directory that cannot be created gives the same result, with `mkdir`
failing first (`git config core.hooksPath /nonexistent/repo/.git/hooks`):

```
mkdir: cannot create directory ‘/nonexistent’: Permission denied
ln: failed to create symbolic link '/nonexistent/repo/.git/hooks/pre-commit.local': No such file or directory
ln: failed to create symbolic link '/nonexistent/repo/.git/hooks/pre-push.local': No such file or directory
Symlinked scripts/git-hooks/pre-commit -> /nonexistent/repo/.git/hooks/pre-commit.local
Symlinked scripts/git-hooks/pre-push   -> /nonexistent/repo/.git/hooks/pre-push.local
Run 'aiwf init' (if not already done) so the chain-aware aiwf hooks call them.
make install-hooks exit: 0
```

## Why it matters

Commits and pushes skip this repository's own hook checks without a sign:
`scripts/git-hooks/pre-commit` runs the policy suite when a commit stages Go, module, build,
workflow or hook files or anything under `internal/skills/`, and `scripts/git-hooks/pre-push`
runs golangci-lint, gitleaks and the comment-history scan. aiwf's own hooks run the `.local`
hook only when the file exists (`chainPrelude` in `internal/initrepo/initrepo.go`,
`if [ -e "$local_hook" ]`), so with the links absent every commit and push passes over those
checks silently, while the target has reported "Symlinked". aiwf's own checks still run after
the chain: `aiwf check --shape-only` at commit and `aiwf check` at push.

A caller that checks the exit status cannot tell either. `.devcontainer/project/post-create.sh`
runs the target under `set -euo pipefail`, so a container is created with the chain missing.
