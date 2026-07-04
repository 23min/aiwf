---
id: M-0231
title: Struct-derived aiwf.yaml schema model and commented-YAML generator
status: draft
parent: E-0057
tdd: required
acs:
    - id: AC-1
      title: Schema model enumerates every yaml field across the config structs
      status: open
      tdd_phase: red
    - id: AC-2
      title: Anti-drift test fails when a yaml field has no schema-model entry
      status: open
      tdd_phase: red
---

# M-0231 — Struct-derived aiwf.yaml schema model and commented-YAML generator

## Goal

Build one in-memory model of the `aiwf.yaml` schema derived from the config
structs — every block's key path, type, effective default, and a one-line
description — and one generator that renders it to commented YAML. This is the
anti-drift backbone: because the model is derived from the same structs the
loader decodes, the documentation it produces cannot silently diverge from what
`aiwf.yaml` actually accepts.

## Context

Today the schema is documented only in `internal/config/config.go` struct doc
comments — source-level, invisible to a consumer. Before any user-facing surface
can document the schema (M-0232 writes the files), there must be a single
generator that turns the structs into a reference so the docs are generated, not
hand-kept. This milestone builds that generator and nothing user-facing yet; the
epic's discoverability payoff lands when M-0232 wires it into `init`/`update`.

## Acceptance criteria

<!-- ACs are authored just-in-time at aiwfx-start-milestone via `aiwf add ac
     M-0231 --title "..."` (seeds tdd_phase: red under tdd: required). The
     intended acceptance shape is sketched in Goal / Design notes; it is
     deliberately not frozen into ACs weeks before the work starts. Expected
     shape at start:
       - schema model enumerates every yaml: field across the config structs
       - anti-drift test fails when a yaml: field has no schema entry
       - generator renders each block with its default + description as valid,
         reparseable commented YAML
       - the accepted-key set is exported as a reusable registry (see Design notes) -->

## Constraints

- **Anti-drift is structural, not vigilance.** A test must fail when any `yaml:`
  field on the config structs has no schema-model entry. A hand-maintained model
  is not an acceptable implementation.
- **The generator's output must be valid, reparseable `aiwf.yaml`.** Rendered
  commented YAML must round-trip: uncommenting a block yields a value the loader
  accepts.
- **Effective defaults, not lying zero-values.** A rendered default must match
  what `config.Load` actually applies, not a struct zero that misrepresents
  behavior (see Design notes — defaults source).

## Design notes

- **Locked: expose the accepted-key set as a reusable exported registry.** This
  is the single-source handshake with `G-0307` (strict-decode): G-0307 will
  derive its accepted-key set from this registry rather than a parallel
  allowlist, and lands the equality test on its side. See the *Coordinate with
  E-0057* section in `G-0307`. This milestone only exposes the registry and does
  not implement strict-decode.
- **Locked: description source is an explicit registry, not `go/ast` parsing.**
  `config.go`'s doc comments attach at the struct level, not per field (e.g.
  `TDD`'s comment describes both `RequireTestMetrics` and `Strict` in one prose
  block; most `Config` fields carry no per-field comment at all) — `go/ast`
  field-attachment would resolve empty or wrong for most fields as the file
  stands today, and fixing that would mean an out-of-scope rewrite of existing
  comments serving a different (Go-developer, design-rationale) audience. The
  description is one field on the same unified per-field schema entry (key
  path, type, default, description) — not a parallel side-table that could
  drift from it.
- **Locked: defaults are read from the config package's real accessors, not
  reflected from struct zero-values and not hand-declared.** Instantiate a
  zero-value `Config{}` and call each field's existing getter/constant when one
  exists (`WorktreeDir()`, `HTMLOutDir()`, `EntityTitleMaxLength()`,
  `AllocateTrunkRef()`, `StatusMdAutoUpdate()`, `WireClaudeMd()`); fall back to
  the literal zero value only for fields with no override getter (provably safe
  there — no logic says the zero value is anything but the real default).
  Struct zero-values alone would lie for `StatusMd.AutoUpdate` and
  `Guidance.WireClaudeMd`: their zero is `nil`, but the real default is `true`.
  This milestone also extracts the two named constants those getters are
  currently missing (`DefaultStatusMdAutoUpdate`, `DefaultWireClaudeMd`) so the
  effective default is never a bare literal hiding inside a getter body.

## Surfaces touched

- `internal/config/config.go` — the source-of-truth structs the model reflects
- a new schema/generator unit under `internal/config/` (package layout settled at start)

## Out of scope

- Writing any file or touching `init`/`update` — that is M-0232.
- Implementing strict-decode / rejecting unknown keys — that is `G-0307`.
- An `aiwf config schema` verb — deferred unless a second consumer earns it
  (E-0057 open question); the generated file surface is the PoC target.

## Dependencies

- None. This is the epic's foundational milestone.

## References

- [`E-0057`](epic.md) — parent epic
- [`G-0307`](../../gaps/G-0307-top-level-aiwf-yaml-decode-stays-non-strict-only-areas-rejects-unknown-keys.md) — strict-decode; consumes this milestone's exported registry
- `internal/config/config.go` — schema source of truth

### AC-1 — Schema model enumerates every yaml field across the config structs

### AC-2 — Anti-drift test fails when a yaml field has no schema-model entry

