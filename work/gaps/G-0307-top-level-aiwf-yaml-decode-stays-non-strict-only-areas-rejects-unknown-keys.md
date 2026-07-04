---
id: G-0307
title: top-level aiwf.yaml decode stays non-strict (only areas rejects unknown keys)
status: open
prior_ids:
    - G-0305
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

## Coordinate with E-0057 — derive the accepted-key set from its schema registry

E-0057 builds a struct-derived schema model of every valid `aiwf.yaml` key (to
generate a consumer-discoverable `aiwf.example.yaml`). Documentation and this
strict-decode guard are the two halves of "safe to hand-author a config field":
E-0057 tells the user which keys exist; this gap makes a mistyped key fail loud.
A key that is documented but still silently ignored on a typo is only
half-solved.

Sequence this **after** E-0057 and consume its output rather than hand-listing
keys:

- **Derive the accepted-key set from E-0057's exported schema registry**, not a
  parallel allowlist maintained here. Two independent key lists drift — a block
  added to the schema model but forgotten in this allowlist (or the reverse)
  reintroduces exactly the silent-misconfiguration class this guard exists to
  kill.
- **Land the enforcing test here, not in E-0057.** Assert the strict decoder's
  accepted-key set equals the schema registry, so the two cannot fall out of
  sync. That equality test can only exist once strict-decode exists, so it is
  this gap's deliverable — E-0057 only exposes the registry and records this
  instruction.
- The legacy-key allowlist above (`actor:`, `aiwf_version:`) is orthogonal: the
  registry supplies the *current* accepted keys; the legacy tolerance is layered
  on top (allowlist or strip-then-strict).

Soft dependency, not a hard block: this gap *could* ship with a hand-maintained
allowlist, but doing so forfeits the single-source guarantee — so wait for
E-0057's registry.
