---
id: G-0704
title: aiwf check refuses a non-UTF-8 or multi-document aiwf.yaml as an edit
status: open
priority: medium
discovered_in: E-0094
---
## What's missing

`aiwf check` refuses an `aiwf.yaml` that is not UTF-8 or holds more than one
YAML document, and reports the refusal as an editing failure. The refusal lives
in `aiwfyaml.ReadBytes` (`internal/aiwfyaml/aiwfyaml.go`), which is also the
reader for the `contracts:` block, so read-only callers reach it through
`cliutil.LoadContractsBlock`: `internal/cli/check/check.go`, and
`internal/cli/contract/contract.go`, `recipes.go` and `verify.go`.

Measured on Linux against a clone of the E-0094 epic branch with
`\n---\nextra: 1\n` appended to `aiwf.yaml`:

```
$ aiwf check --root <clone>
aiwf check: reading aiwf.yaml: cannot edit configuration: aiwf.yaml must contain a single YAML document; remove additional documents and retry
exit 3
$ aiwf show E-0094 --root <clone>
exit 0
```

A build of `main` before E-0094 exits 0 on the same `aiwf check`. The main
configuration loader, `config.Load` (`internal/config/config.go`), still
accepts both inputs, so one command reads a file another refuses.

## Why it matters

`aiwf check` is the pre-push hook. A repository whose `aiwf.yaml` carries a
second document or a non-UTF-8 encoding cannot push after upgrading, and the
message tells the operator their *edit* was refused when they ran a read. The
two loaders disagreeing means the rule for what `aiwf.yaml` may contain depends
on which command reads it first.
