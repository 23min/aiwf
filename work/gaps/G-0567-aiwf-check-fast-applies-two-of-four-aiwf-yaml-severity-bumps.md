---
id: G-0567
title: aiwf check --fast applies two of four aiwf.yaml severity bumps
status: addressed
priority: high
discovered_in: M-0300
addressed_by_commit:
    - 29eb2a94cdb1551e88d92f1dfa20e00ca658f100
---
## What's missing

`aiwf check` composes four `aiwf.yaml` severity passes over the findings
`check.Run` produced: `ApplyTDDStrict`, `ApplyAreaRequiredStrict`,
`ApplyDocsStrict`, and `ApplyArchiveSweepThreshold`. `aiwf check --fast`
composes two of them — the first and the last. Its own doc comment says it runs
"the aiwf.yaml severity bumps", without qualification.

One of the two omissions is a real severity divergence. `--fast` runs
`AreaUnknown`, so it emits the finding and simply never applies the escalation
the full check applies to it. Measured on one tree with `areas.required: true`
and an entity carrying an undeclared area:

```
aiwf check          error area-unknown ...   1 errors    exit 1
aiwf check --fast   warning area-unknown ... 0 errors    exit 0
```

The other omission is not. `--fast` never runs the doc rules at all, so
`ApplyDocsStrict` there has nothing to escalate and the two surfaces share no
finding to disagree about. That is a rule omission, of the same kind as the
trunk, provenance, FSM-history, metrics and contract rules `--fast` also skips
by design.

## Why it matters

The divergence is wider than the two surfaces this gap was opened about. Seven
call sites reach `check.Run`, and their severity-pass counts are four (`check`),
two (`check --fast`), and zero for `status`, `show`, `render`, `doctor`, and the
verb-time projection guard.

The projection guard is the consequential one. It runs bare `check.Run`, so a
verb decides whether to refuse a mutation against unescalated severities.
Measured with `tdd.strict: true`, `aiwf add epic` prints `ok — no findings` and
commits a state the pre-push hook then blocks. A read surface under-reporting is
a nuisance; a write path that reports success while landing a state the gate
refuses is a defect of a different order, and it is tracked separately.

No consumer currently reads `check --fast`. The statusline does not run it — the
health glyph reads `.claude/health.*.json`, written out of band by `aiwf doctor`,
which extracts only the id-collision code from `check.Run`. The `--fast`
reference in the statusline source is a comment left from before that mechanism
replaced it. So repairing `--fast`'s severity passes changes no observed output
today; what makes the repair worth doing is that the fast path is the
pre-commit-adjacent surface an operator is invited to reach for, and that
nothing mechanical keeps any of the seven call sites in agreement about which
passes apply.

The durable artefact is therefore not the two missing calls. It is one shared
severity-application seam plus a policy enumerating `check.Run`'s call sites
against the applier set they are each meant to carry, so the next added pass
cannot silently reach only some of them.
