---
id: G-0725
title: make install-hooks reports success when it cannot create the hook links
status: open
discovered_in: M-0359
---
## What's missing

The `install-hooks` target in `Makefile` (lines 326-333) runs its `mkdir -p` and two `ln -sfn`
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
failing first (`git config core.hooksPath /Users/nobody/repo/.git/hooks`):

```
mkdir: cannot create directory ‘/Users’: Permission denied
ln: failed to create symbolic link '/Users/nobody/repo/.git/hooks/pre-commit.local': No such file or directory
ln: failed to create symbolic link '/Users/nobody/repo/.git/hooks/pre-push.local': No such file or directory
Symlinked scripts/git-hooks/pre-commit -> /Users/nobody/repo/.git/hooks/pre-commit.local
Symlinked scripts/git-hooks/pre-push   -> /Users/nobody/repo/.git/hooks/pre-push.local
Run 'aiwf init' (if not already done) so the chain-aware aiwf hooks call them.
make install-hooks exit: 0
```

## Why it matters

Commits and pushes skip the kernel's pre-commit and pre-push checks without a sign. aiwf's own
hooks run the `.local` hook only when the file exists (`chainPrelude` in
`internal/initrepo/initrepo.go`, `if [ -e "$local_hook" ]`), so with the links absent every
commit and push passes over the kernel chain silently, while the target has reported "Symlinked".

A caller that checks the exit status cannot tell either. `.devcontainer/project/post-create.sh`
runs the target under `set -euo pipefail`, so a container is created with the chain missing.
