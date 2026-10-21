---
id: G-0714
title: Guidance installer errors for corrupt records and file-system faults name no fix
status: open
discovered_in: M-0352
---
## What's missing

When `aiwf update` cannot install project guidance, its guidance report prints the installer's error on a `blocked:` line, and that error is expected to name the action that clears it. The errors for a corrupt installation record and for file-system faults name only the cause. They are written in `internal/projectguidance/install.go`: `readInstallFile` (`is not a regular file`, `reading %s`), `ownedBytes` (`has malformed project guidance markers`), `readRecord` (`reading %s`, `malformed %s`, `contains an unowned path`), `readInstallRecords` (`invalid ownership digest`, `invalid recovery digest`, `invalid recovery temporary path`, `recovery temporary file %s was edited`) and the write path (`creating parent for`, `retiring`, `installing`, `removing completed recovery record`, `encoding`). The same holds for a per-file detection failure, `inspecting guidance candidate %s: %w` in `internal/projectguidance/detect.go`, and for the temporary-directory errors in `internal/projectguidance/retrieve.go` (`creating temporary guidance directory`, `removing temporary guidance directory`). The corrupt-record case is measured below; the file-system fault messages are read from the source, not measured.

Measured with a build of the E-0095 epic branch at M-0352/AC-2, in a scratch repository with an isolated HOME and Git configuration, a local guidance source listing `sample/base`, and `aiwf.yaml` `hosts: [codex]` with that source:

```
$ aiwf update --root repo            # installs sample/base, exit 0
$ echo 'not json' > repo/.guidance/.aiwf-owned
$ aiwf update --root repo            # exit 0
Guidance:
  selected:  sample/base
  adopted:   none
  installed: cannot inspect installed policy: project guidance installation conflict: reading .guidance/.aiwf-owned: invalid character 'o' in literal null (expecting 'u')
  ignored:   none
  blocked:   project guidance installation conflict: reading .guidance/.aiwf-owned: invalid character 'o' in literal null (expecting 'u')
```

Expected: the `blocked:` line names what to do, as it does for a retrieval failure, incompatible personal delivery, a handwritten legacy import or a locally edited file.

## Why it matters

A maintainer who meets one of these errors is told what went wrong but not how to recover. For a corrupt `.guidance/.aiwf-owned` or recovery record, the recovery is not obvious. Deleting the record, measured the same way, gives `installed: none` and `blocked:   project guidance installation conflict: .guidance/packs/sample/base/guide.md is foreign or locally edited; preserve your changes in project.md or reconcile the generated file before retrying` on the next update, although that file is byte-identical to the source's copy, so the obvious repair trades one blocker for another whose advice does not fit. Every later `aiwf update` repeats the same blocked line, so guidance stays frozen until the maintainer works out the repair from the source.
