---
id: G-0757
title: Docs say adr-supersession-mutual flags a missing successor; it never does
status: open
---
## What's missing

`adrSupersessionMutual` (`internal/check/check.go`) skips every ADR whose
`superseded_by` is empty, so a superseded ADR that names no successor never gets
an `adr-supersession-mutual` finding. These texts say it does:

- `internal/skills/embedded/aiwf-promote/SKILL.md`: "the
  gap-addressed-has-resolver and adr-supersession-mutual checks fire whenever the
  field is missing".
- `internal/verb/promote.go`, where the verb requires a resolver: "The check
  rules gap-addressed-has-resolver and adr-supersession-mutual surface the
  problem post-hoc".
- `internal/verb/promote.go`, on the helper that enforces the requirement: "so
  the gap-addressed-has-resolver and adr-supersession-mutual warnings cannot be
  reached via the verb".

The `PromoteOptions` comment in the same file and the spec's comment on the cell
named below tie the field to the rule in the same way.

`aiwf promote <adr> superseded` without `--superseded-by` refuses and names the
rule: "requires --superseded-by <adr-id> so the adr-supersession-mutual rule is
satisfied; pass --force to override". With `--force --reason`, the state lands
and the rule never reports it. The gap half of these texts holds:
`gapAddressedHasResolver` reports an addressed gap with no resolver however it
got there.

The legal-workflow spec's `accepted → superseded` cell with
`self.superseded_by == ""` (`internal/workflows/spec/rules.go`) is about the
unforced request, which the verb rejects; the spec says nothing about the state
`--force` leaves. Nor does the code say that state is wrong: `superseded` is the
only status the state machine allows from `accepted`
(`internal/entity/transition.go`), so retiring an accepted ADR that nothing
replaced takes `--force` either way, to `superseded` with no successor or to
`rejected` (`aiwf promote <adr> rejected --force --reason "..."` exited 0 in the
environment below), where a gap has `wontfix`. What is contradicted is these
texts, by `check.go`.

Reproduced 2026-11-01 with aiwf built from `main` at `c06ffdaf9`
(`vcs.modified=false`), git 2.51.1, Linux, with the hooks `aiwf init` installs,
in a fresh repository holding `ADR-0001`, promoted to `accepted`, and a gap
`G-0001`:

```
$ aiwf promote ADR-0001 superseded
aiwf promote: promoting an ADR to "superseded" requires --superseded-by <adr-id> so the adr-supersession-mutual rule is satisfied; pass --force to override
exit 2
$ aiwf promote ADR-0001 superseded --force --reason 'no successor'
aiwf promote ADR-0001 accepted -> superseded
exit 0
$ grep -H -E '^(status|superseded_by):' docs/adr/ADR-0001-*.md
docs/adr/ADR-0001-adr-one.md:status: superseded
$ aiwf promote G-0001 addressed
aiwf promote: promoting a gap to "addressed" requires --by <entity-id> or --by-commit <sha> so the gap-addressed-has-resolver rule is satisfied; pass --force to override
exit 2
$ aiwf promote G-0001 addressed --force --reason 'no resolver'
aiwf promote G-0001 open -> addressed
exit 0
$ aiwf check
terminal-entity-not-archived (warning) × 2 — entity ADR-0001 has terminal status "superseded" but file is still in the active tree; awaiting `aiwf archive --apply` sweep
archive-sweep-pending (warning) × 1 — 2 terminal entities awaiting `aiwf archive --apply`. Set `archive.sweep_threshold` in aiwf.yaml to escalate to blocking past N
gap-addressed-has-resolver (warning) × 1 — gap is marked addressed but addressed_by and addressed_by_commit are both empty
provenance-untrailered-scope-undefined (warning) × 1 — no upstream configured and no --since <ref>; provenance audit, body-section gate and instruction-file fence skipped

5 findings (0 errors, 5 warnings)
run `aiwf check --verbose` for each warning's location and remediation hint
exit 0
```

The gap forced the same way gets `gap-addressed-has-resolver`; the ADR gets no
`adr-supersession-mutual` finding.

## Why it matters

The skill tells an operator, and the code comment tells a maintainer, that a
warning stands behind a forced supersession with no successor, and the verb's
refusal points at the rule that would raise it. None appears: `aiwf check` gives
no sign of the state, while the same override on a gap keeps its warning. Both
rules are warnings, so neither blocks a push; what is missing is the one signal
the texts promise.

## Relates to

G-0756 records the stale `supersedes` that a forced re-point of `superseded_by`
leaves, which the same rule does not report either.
