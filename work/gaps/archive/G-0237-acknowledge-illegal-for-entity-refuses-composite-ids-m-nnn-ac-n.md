---
id: G-0237
title: acknowledge-illegal --for-entity refuses composite ids (M-NNN/AC-N)
status: addressed
addressed_by_commit:
    - 440bf1a3dbf16d756b76aa04e26135397ee40535
---
## What's missing

`aiwf acknowledge-illegal --for-entity` refuses composite ids (`M-NNN/AC-N`).

The verb's `verifySHATouchesEntity` (`internal/verb/acknowledgeillegal.go:147-161`) canonicalizes the supplied `forEntity` via `entity.Canonicalize`. For a composite input `M-0001/AC-1` that returns `M-0001/AC-1`. But the diff-walking side resolves each touched path through `entity.PathKind` + `entity.IDFromPath`, which returns only the parent milestone id (e.g. `M-0001` from `work/epics/E-.../M-0001-foo.md`). The comparison `M-0001/AC-1` vs `M-0001` always misses, so the verb refuses the ack even when the SHA legitimately touched the parent milestone file.

## Why it matters

Acks against acceptance criteria are a real use case. A historical commit that inline-edited an AC body section (the `### AC-N — ...` prose inside a milestone file) fires `provenance-untrailered-entity-commit` against the composite id `M-NNN/AC-N` (per `compositeRoot` rollup in the rule's emission path). The only available ack mechanism — `aiwf acknowledge-illegal --for-entity` — would refuse the operator's binding, even though the SHA mechanically touched the parent file. The operator's only workaround is "ack the parent milestone id instead," which is undocumented and changes the recorded `aiwf-entity` trailer from `M-NNN/AC-N` to `M-NNN`, losing the per-AC granularity the rule's finding shape pinned.

## How to fix

One line: add `compositeRoot(forEntity)` (or `entity.Canonicalize(compositeRoot(forEntity))`) before the loop in `verifySHATouchesEntity` so the verb compares parent-against-parent. The same `compositeRoot` helper the rule's emission side uses (`internal/check/provenance.go`) is the right call. Add two tests: composite-id positive control (`M-NNN/AC-N` succeeds when SHA touched `M-NNN`-foo.md), and a negative control confirming the wrong parent still fails.

If `compositeRoot` is internal to `internal/check`, lift it to `internal/entity` (its natural home) and call from both sides — this also closes the silent-skew failure mode where the two roll-up implementations could drift.

## Source

G-0231 reviewer pass, N5 finding ("composite entity id binding" — surfaced after the kernel-extension diff that added `--for-entity` in the first place).
