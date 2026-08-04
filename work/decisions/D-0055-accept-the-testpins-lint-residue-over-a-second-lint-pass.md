---
id: D-0055
title: Accept the !testpins lint residue over a second lint pass
status: proposed
---
## Context

`.golangci.yml` declares `run.build-tags`. Adding `testpins` brought the
`testpins`-gated sources — the branch Pin registry, the CLI integration pin
tests, and one policy sabotage test — into the lint surface for the first time,
closing the hole G-0470 reported.

`golangci-lint` lints the union of the declared tags in a single pass rather than
lint each tag configuration separately. A file gated behind `//go:build
!testpins` therefore compiles in no configuration the linter runs, and leaves the
lint surface as a side effect of the tag being declared. The residue is
`internal/cli/integration`'s negated-arm pin files.

The untagged `go vet ./...` in `.github/workflows/go.yml` is the configuration the
negated arm builds under, so the compile-level check still reaches it. What it
loses is the style and correctness surface `golangci-lint` adds over `go vet`:
`staticcheck`, `errcheck`, `gocritic`, `revive`, `gosec` and the rest of the
enabled set.

## Decision

Accept the negated-arm residue. Do not add a second `golangci-lint` invocation
configured without `testpins`.

Rationale:

- The residue is test scaffolding whose whole job is to be referenced by the
  tagged arm. The linters that would reach it police style and misuse in code
  that has neither branches nor callers outside its own pin.
- A second invocation is not a second file's worth of work — `golangci-lint` has
  no per-tag mode, so reaching the negated arm costs a full additional pass over
  the module in `make lint`, in the pre-push hook, and in the `lint` CI job
  alike. That is paid on every push, forever, against a fixed and tiny subject.
- The asymmetry is deliberate rather than accidental: the tagged arm carries the
  registry and the assertions, so it is where a lint finding would mean
  something. Declaring `testpins` puts the linter on the arm that has the
  content.

Rejected alternative: leaving `testpins` undeclared, which was the state G-0470
reported. It trades a large unlinted surface for a small one and is strictly
worse.

## Revisit trigger

Add the second invocation when a **third** `//go:build !testpins` file appears,
or when the negated arm grows past scaffolding into logic with its own branches.
Either signals that the unlinted surface has stopped being a fixed, trivial
constant, which is the only property that makes one lint pass over it a bad
trade.

## References

- G-0470 — the gap this closes, which named the negated arm as an open decision
  rather than deciding it.
- `.golangci.yml` — the operating fact at the site: why the residue exists and
  what still compiles it.
