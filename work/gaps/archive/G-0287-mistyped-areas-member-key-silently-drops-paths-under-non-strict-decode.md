---
id: G-0287
title: Mistyped areas member key silently drops paths under non-strict decode
status: addressed
discovered_in: M-0179
addressed_by_commit:
    - d944b07d
---
## Problem

The `areas:` block decode (`config.Areas.UnmarshalYAML`) accepts the object-form member
`{name, paths}` via `yaml.Node.Decode`, which is NON-strict — an unknown key in a member
mapping is silently ignored. A typo'd key (`pathz:`, or singular `path:`) therefore drops the
operator's paths with no error: the member decodes paths-less.

This is inert in M-0179 (paths are validated as strings only and never consumed). It becomes
consequential at M-0180+: the bijection/coverage check (M-0180) and mistag detection (M-0181)
key on `paths`, so a silently-dropped `paths:` block yields a false "area has no location"
rather than a config error the operator can fix at load time.

Discovered in M-0179 — both the `wf-rethink` and `wf-review-code` wrap reviews flagged it.
Consistent with the config-wide non-strict `yaml.Unmarshal`, so it is not a new inconsistency.

## Direction

When paths become load-bearing (M-0180), make the member-mapping branch reject unknown keys —
cheapest local form is an explicit `{name, paths}` key allowlist in the `MappingNode` case
(`yaml.Node.Decode` does not expose `KnownFields`). Weigh against the config-wide non-strict
convention: if the project wants strict loading more broadly, that is a larger decision than
this one field.
