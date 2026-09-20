---
id: G-0701
title: README's installed-hook list omits the commit-msg hook
status: open
---
## What's missing

`README.md` §"Git hooks" enumerates the hooks `aiwf init` and `aiwf update`
install — `.git/hooks/pre-push`, `.git/hooks/pre-commit`, `.git/hooks/post-commit`
— and never mentions `.git/hooks/commit-msg`, which has shipped alongside them
since the commit-msg refusal landed. The same omission appears in the `aiwf init`
comment earlier in the file, which reads "installs pre-push + pre-commit hooks".

Observed: `grep -n 'commit-msg' README.md` returns no hit in the hook
enumeration, while `aiwf init` in a clean repo writes `.git/hooks/commit-msg`
carrying the `# aiwf:commit-msg` marker, with the same `.local` chaining and
auto-migration behavior the three documented hooks describe.

## Why it matters

A consumer reading the hook section cannot learn that a fourth hook is installed,
what marker it carries, or that it chains to `commit-msg.local` the way the
others chain — so a consumer with an existing `commit-msg` hook meets the
auto-migration without having been told it can happen.

The seam widened when `provenance.refuse_coauthors` shipped: `aiwf.example.yaml`
now describes that knob as one "the commit-msg hook refuses" addresses through,
pointing a reader at a hook the README never introduces.
