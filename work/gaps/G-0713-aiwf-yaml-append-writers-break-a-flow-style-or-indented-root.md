---
id: G-0713
title: aiwf.yaml append writers break a flow-style or indented root
status: open
discovered_in: M-0351
---
## What's missing

`aiwfyaml`'s append writers add a missing top-level block as column-0 block-style bytes after the existing content: `appendHooks` (`internal/aiwfyaml/hooks.go:118`, reached through `SetHooks`, `hooks.go:42`) and `appendContracts` (`internal/aiwfyaml/aiwfyaml.go:712`). When the root mapping is flow-style or indented, the result no longer parses as one document. `aiwf init` and `aiwf update` reach the hooks writer; `aiwf contract bind`, `aiwf contract recipe install` and `aiwf add` reach the contracts writer, which commits the result.

The refusal shows at `cliutil.SyncHookMaterialization` (`internal/cli/cliutil/hooks.go:83-86`, from `internal/cli/update/update.go:183`) on the run that writes the block, and at `gateAndSyncHookDecisions` (`internal/cli/update/hooks.go:22-25`, from `update.go:180`) on every later run. Both call sites carry a `//coverage:ignore` stating the path is unreachable. `aiwfyaml.go:114-116` reports any second-decode error as a multi-document file.

Measured with aiwf v0.38.0 and with a build of `3218fa8c4`, in a scratch repository with an isolated HOME and Git configuration, `aiwf.yaml` containing `{hosts: [claude-code], guidance: {enabled: false}}`:

```
$ aiwf update --root repo            # exit 3
aiwf update: done.
aiwf update: hook "worktree-rituals-check.sh" — deferred (undecided — run `aiwf doctor`) (new)
aiwf: cannot edit configuration: aiwf.yaml must contain a single YAML document; remove additional documents and retry
$ cat repo/aiwf.yaml
{hosts: [claude-code], guidance: {enabled: false}}

hooks: {}
$ aiwf update --root repo            # exit 3, same message
$ aiwf check --root repo             # exit 3
aiwf check: reading aiwf.yaml: cannot edit configuration: …
```

The same happens for an indented root (`  hosts: [claude-code]\n  guidance:\n    enabled: false\n`), for `aiwf init` on an existing flow-style file, and for `aiwf contract recipe install cue`, which exits 0 and commits the unparseable file. Expected: `aiwf.yaml` stays one valid document carrying the new block's values, and the next `aiwf update` and `aiwf check` exit 0.

## Why it matters

`aiwf check` exits 3 on the written file, so the pre-push hook refuses every push until the file is repaired by hand; commits still pass the shape-only pre-commit check, so the breakage surfaces only at the push.

The refusal tells the user to remove additional documents, but the file has none — it has a parse error — so the advice sends them looking for a `---` that is not there. Meanwhile `config.Load` still reads the file and silently drops the appended block, so commands that only load configuration run without the recorded hook decisions or contracts.

The README presents flow-style and indented roots as supported for guidance edits, so nothing warns a consumer that hook consent or a contract write will break them.
