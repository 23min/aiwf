---
id: G-0573
title: The projection guard cannot see the body bytes a verb is about to write
status: open
priority: high
discovered_in: M-0300
---
## What's missing

The verb-time projection guard decides whether a mutation may land by projecting the
post-mutation tree and refusing on newly-introduced error-severity findings. It cannot
see the body bytes a verb is about to write. `entity-body-empty` reads them from disk
(`internal/check/entity_body.go:147`) and stays silent when that read fails, while the
plan carrying the new bytes is applied only after the guard has returned clean. For an
entity being created there is no file at all, so the rule produces nothing the guard
could refuse on.

Measured 2026-08-25 on `main`, in a fresh repo with `tdd.strict: true`:

```
$ aiwf add epic --title "Strict probe epic"
ok — no findings
aiwf add epic E-0001 "Strict probe epic"        exit 0

$ aiwf check
work/epics/E-0001-strict-probe-epic/epic.md:1: error entity-body-empty/epic:
  E-0001 body section `## Goal` is empty
... (x3)                                         exit 1
```

The severity seam is not what fails. Those are errors rather than the default
warnings, so the consumer's `aiwf.yaml` is being read, and
`internal/verb/common.go:146` applies the policy to both sides of the projection diff.

## Why it matters

The guard exists so a verb refuses rather than landing a bad state. A verb that prints
`ok — no findings`, commits, and leaves a tree the pre-push hook refuses has already
written the state by the time anything disagrees. A repo carrying the knob gets this
on every create.
