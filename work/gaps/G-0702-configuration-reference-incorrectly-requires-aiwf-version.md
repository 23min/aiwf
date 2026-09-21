---
id: G-0702
title: Configuration reference incorrectly requires aiwf_version
status: open
discovered_in: M-0346
---
## What's missing

`docs/design/design-decisions.md`, in the `aiwf.yaml` configuration table, marks
`aiwf_version` as required. The loader accepts a configuration without that field.

Observed in the Linux development container on 2026-09-21, from the E-0094
worktree. Command:

```sh
sed -n '337p' docs/design/design-decisions.md
go test ./internal/config -run '^TestLoad_EmptyFileIsOK$' -count=1 -v
```

Expected the documented requirement to agree with the loader. The first command
printed:

```text
| `aiwf_version` | string | yes | Engine version the repo expects (e.g., `0.1.0`). `aiwf doctor` warns on mismatch. |
```

The test creates an empty `aiwf.yaml`, calls `config.Load`, and requires success.
Observed:

```text
--- PASS: TestLoad_EmptyFileIsOK (0.00s)
PASS
```

The conflicting surfaces are the configuration table and
`internal/config/config.go`'s `Load` / `Validate` behavior, exercised by
`internal/config/config_test.go`.

## Why it matters

Maintainers following the configuration reference are instructed to provide a
field that the loader does not require. The normative documentation therefore
misstates the minimum valid project configuration.
