---
name: aiwf-worktree
description: Use whenever a ritual, subagent dispatch, or ad-hoc fix needs a new git worktree in this repo. Runs `aiwf worktree add` to create the checkout and materialize the selected hosts' aiwf artifacts; bare Git worktree creation does not perform that setup.
---

# aiwf-worktree

Use `aiwf worktree add` to create a checkout and materialize its selected hosts' artifacts in one command. Bare `git worktree add` does not materialize aiwf's ignored artifacts. For an existing checkout created another way, run `aiwf update` there and inspect `aiwf doctor` before relying on its installation; copied artifacts may be missing or stale.

## When to use

Any time you are about to create a git worktree for branch work in this repo — starting an epic, starting a milestone, a one-off patch branch, or a subagent that needs an isolated checkout. Use this instead of a bare `git worktree add`.

## What to run

```bash
# Create a NEW branch off a base ref, at the default in-repo placement
aiwf worktree add <branch> --base <base-ref>

# Create a worktree at an explicit path (sibling directory, any custom location)
aiwf worktree add <branch> <path> --base <base-ref>

# Reuse an EXISTING local branch (omit --base; it only applies to new branches)
aiwf worktree add <branch>

# Enter the checkout in the invoking shell only after creation succeeds
checkout=$(aiwf worktree add <branch> --print-path) && cd "$checkout"
```

- `<branch>` is required. When it does not already exist as a local branch, aiwf creates it fresh starting from `--base` (default: HEAD). When it already exists, aiwf reuses it and `--base` is rejected as a usage error — you cannot re-point an existing branch's start.
- `<path>` is optional. Omit it to resolve to the configured worktree-placement directory plus the branch name; pass it explicitly for a sibling directory or any other location. An explicit path is honored verbatim — it is never redirected back in-repo, even if it points outside the repo.
- `--print-path` suppresses every other output and prints only the resulting absolute path to stdout on success, nothing on failure. Use this mode for shell composition; don't parse the normal ledger output for the path.

The shell example changes the invoking shell's directory. Follow the calling ritual's {{aiwf:host_label}} worktree-entry instructions for session and tool working directories, instruction loading, and wrap behavior. Verify the checkout root and branch before mutations.

## What aiwf does

1. Runs `git worktree add`, surfacing any git failure directly (branch already checked out elsewhere, path already exists, etc.) — never reports success on a failed creation.
2. Materializes each selected host's supported skills and templates, plus role agents where supported. Root guidance follows the configured wiring opt-outs. Host selection and refresh use the same pipeline as `aiwf update`, targeted at the new checkout.
3. Prints the resulting absolute path (or, under `--print-path`, only the path).

Run `aiwf doctor --root <path>` to inspect the selected hosts' installation. Successful disk checks do not establish that an assistant session loaded the files.

## Don't

- Use `aiwf worktree add` for repo branch work so selected-host setup is included.
- Don't expect this verb to change your shell's current directory — no child process can `chdir` its parent. Use the guarded shell example above and follow the calling ritual's host-specific entry instructions.
- Don't pass `--base` when reusing an already-existing branch — aiwf rejects the combination rather than silently ignoring the flag.
- Don't parse the normal (non-`--print-path`) output for the path in a script — that output includes a materialization ledger; `--print-path` is the stable, script-safe surface.
