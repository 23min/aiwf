---
id: G-0756
title: A forced supersession re-point leaves a stale supersedes on the old successor
status: open
---
## What's missing

`aiwf promote <adr> superseded --superseded-by <new> --force --reason "..."`
re-points an ADR's `superseded_by` and writes the reciprocal `supersedes` onto
the new successor (`reciprocalSupersedesOp`, `internal/verb/promote.go`), but
never reads the old successor: `promoteClaimPaths`, in the same file, names only
the re-pointed ADR and the new one. The old successor's `supersedes` keeps
listing the ADR.

`aiwf check` does not report the leftover. `adrSupersessionMutual`
(`internal/check/check.go`) checks one direction, as `README.md` documents it —
"If A.superseded_by = B, then B.supersedes ⊇ {A}" — so a `supersedes` entry the
other ADR does not confirm passes. `docs/design/legal-workflows-audit.md`
describes the rule as covering both: R-AUDIT-0097 says an ADR "claiming to
supersede another via `supersedes:` while the other doesn't have
`superseded-by:` (or vice versa) fires `adr-supersession-mutual`", and
R-AUDIT-0158 that the two fields "must be mutually consistent; checked by
`adr-supersession-mutual`".

Reproduced 2026-11-01 with aiwf built from `main` at `c06ffdaf9`
(`vcs.modified=false`), git 2.51.1, Linux, with the hooks `aiwf init` installs,
in a fresh repository holding `ADR-0001`, `ADR-0002` and `ADR-0003`, each
promoted to `accepted`:

```
$ aiwf promote ADR-0001 superseded --superseded-by ADR-0002
aiwf promote ADR-0001 accepted -> superseded
exit 0
$ aiwf promote ADR-0001 superseded --superseded-by ADR-0003 --force --reason 'correct the pointer'
aiwf promote ADR-0001 superseded -> superseded
exit 0
$ grep -H -A1 -E '^(superseded_by|supersedes):' docs/adr/ADR-000[123]-*.md
docs/adr/ADR-0001-adr-one.md:superseded_by: ADR-0003
docs/adr/ADR-0001-adr-one.md----
--
docs/adr/ADR-0002-adr-two.md:supersedes:
docs/adr/ADR-0002-adr-two.md-    - ADR-0001
--
docs/adr/ADR-0003-adr-three.md:supersedes:
docs/adr/ADR-0003-adr-three.md-    - ADR-0001
$ aiwf check
archive-sweep-pending (warning) × 1 — 1 terminal entities awaiting `aiwf archive --apply`. Set `archive.sweep_threshold` in aiwf.yaml to escalate to blocking past N
provenance-untrailered-scope-undefined (warning) × 1 — no upstream configured and no --since <ref>; provenance audit, body-section gate and instruction-file fence skipped
terminal-entity-not-archived (warning) × 1 — entity ADR-0001 has terminal status "superseded" but file is still in the active tree; awaiting `aiwf archive --apply` sweep

3 findings (0 errors, 3 warnings)
run `aiwf check --verbose` for each warning's location and remediation hint
exit 0
```

The forced commit changed `docs/adr/ADR-0001-adr-one.md` and
`docs/adr/ADR-0003-adr-three.md` only (`git show --stat`). `ADR-0002` still lists
`ADR-0001` under `supersedes`, which names only `ADR-0003`, and no
`adr-supersession-mutual` finding fires.

## Why it matters

The forced re-point is how an operator corrects a supersession that named the
wrong successor, and it leaves that wrong successor still saying it replaced the
ADR. Two ADRs then claim one ADR's place in the record, and `aiwf check` gives no
sign of it. The rule is a warning, so a push passes either way; what is missing
is the signal the audit rows say the rule gives.

## Relates to

G-0255 recorded the same one-sided write on a first supersession and is
addressed; this is the re-point. G-0442 records that `aiwf promote` re-points
`superseded_by` after its transition only this way, with `--force`.
