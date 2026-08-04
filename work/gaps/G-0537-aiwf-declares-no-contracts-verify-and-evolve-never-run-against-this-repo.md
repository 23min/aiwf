---
id: G-0537
title: aiwf declares no contracts; verify and evolve never run against this repo
status: open
priority: low
---
## What's missing

The contract surface ships two oracles behind one verb. `aiwf contract verify`
runs the verify pass — every fixture under the valid tree must pass the schema,
every fixture under `invalid/` must fail it — and the evolve pass, which holds
historical valid fixtures against HEAD's schema and emits
`evolution-regression` when a schema change invalidates one.

Neither runs against this repo. `work/contracts/` is empty and `aiwf.yaml`
declares no contract binding, so both passes execute only in consumer trees.
What covers them here is `internal/contractverify`'s own test suite, which
drives the verifier against fabricated bindings — that pins the engine, not any
schema aiwf itself depends on.

## Why it matters

The `invalid/`-fixture requirement is the sharpest meta-oracle the repo owns:
requiring every fixture under `invalid/` to fail is what separates a schema that
rejects from one that merely looks strict, and a validation setup missing that
half is vacuous without appearing so. The argument is made in the design docs
and shipped to consumers by the `aiwf-contract` skill, and the repo making it
has no instance of its own to point at.

aiwf has schemas that would earn the treatment. The top-level `aiwf.yaml` decode
is non-strict (G-0307), the `--format=json` envelope is a wire contract
consumers script against, and the health file is read by a statusline the kernel
does not own. Each is a shape crossing a boundary with no fixture library behind
it.

## Resolution shape

One binding, not a sweep. The JSON envelope is the strongest candidate: its
shape is already fixed, consumers already depend on it, and its invalid cases
are easy to name — a missing `status`, a `status` outside the closed set, a
`findings` entry without a code. A first binding also answers something the unit
tests cannot, which is whether authoring a contract against a real schema is as
workable as the documentation says.

## Sequencing

Behind any gap that names a defect. Nothing here is broken; a claim the repo
makes about itself is unexercised.
