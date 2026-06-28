---
id: G-0305
title: top-level aiwf.yaml decode stays non-strict (only areas rejects unknown keys)
status: open
discovered_in: M-0185
---

## What's missing

A top-level strict-key guard on the `aiwf.yaml` decode. M-0185 added an
areas-block-level guard (`unknownAreasKey` / `knownAreasKeys` in
`internal/config/config.go`) that rejects an unknown key inside the `areas:`
mapping at load — mirroring G-0287's member-level guard. But the top-level
`Config` decode (`config.Load` → `yaml.Unmarshal`) stays non-strict: a typo'd
*block* name (e.g. `araes:` for `areas:`, or `tdd: { stict: true }`) is silently
dropped — the same silent-misconfiguration class the areas-block guard exists to
kill, uncaught one level up and in every sibling block.

## Why it matters

The asymmetry is principled, not merely expedient (per the M-0185 design
review): areas-block keys feed safety *checks* whose silent misconfiguration
produces a false negative *inside a check* — false confidence that a guard is
active; a typo'd top-level block merely makes a feature visibly not take effect,
which is lower-stakes and self-evident. So guarding the `areas:` block more
tightly than the whole config is defensible. But the gap should still close so
config typos fail loud uniformly across the file.

**Constraint for whoever picks this up (from the M-0185 design review):** this is
NOT a trivial `yaml.Decoder.KnownFields(true)` flip. `config.Load` deliberately
tolerates two legacy keys on read during the migration window — `actor:`
(pre-I2.5) and `aiwf_version:` (pre-G47), captured into `LegacyActor` /
`LegacyAiwfVersion` and stripped on `aiwf update`. A naive strict decode would
reject those and break that documented read-tolerance, so the fix must allowlist
the legacy keys (or strip-then-strict) rather than blanket-strict.
