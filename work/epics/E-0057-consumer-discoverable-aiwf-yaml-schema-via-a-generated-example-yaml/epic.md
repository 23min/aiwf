---
id: E-0057
title: Consumer-discoverable aiwf.yaml schema via a generated example.yaml
status: proposed
---

# E-0057 — Consumer-discoverable aiwf.yaml schema via a generated example.yaml

## Goal

Give every aiwf consumer a discoverable, always-fresh reference for the whole
`aiwf.yaml` schema — inside their own repo, without reading aiwf's source. A
config surface a user cannot discover is a feature that effectively does not
exist; today the entire schema is documented only in Go struct doc comments.

## Context

Every `aiwf.yaml` block — `tdd`, `allocate`, `archive`, `guidance`, `worktree`,
`entities`, `areas`, `agents`, `status_md`, `html`, `tree`, `hosts` — is
documented **only** in `internal/config/config.go` struct doc comments. That is
source-level: not reachable via `aiwf <verb> --help`, an embedded skill, the
shipped guidance fragment, or the consumer's `CLAUDE.md`. Design docs are named
by the kernel's "AI-discoverable" principle as an acceptable channel, but design
docs **do not materialize into consumer repos** — so a field documented only
there is discoverable for someone reading aiwf's tree, yet invisible to the
consumer who must author the field. `aiwf init` writes a *minimal* `aiwf.yaml`,
so a fresh repo gives no hint any block exists.

Concrete trigger: the `agents:` model/effort block shipped in v0.24.0. A
consumer can learn it exists only from release notes or by reading aiwf's
source — nothing in their own repo hints at it. This is a systemic hole across
the whole config file, not one field.

This epic addresses [`G-0360`](../../gaps/G-0360-aiwf-yaml-has-no-consumer-discoverable-schema-reference-for-its-blocks.md),
which generalized the retired `G-0288` (the `areas:`-only version, already
`wontfix`) to the entire schema. It coordinates with — but does not absorb —
[`G-0307`](../../gaps/G-0307-top-level-aiwf-yaml-decode-stays-non-strict-only-areas-rejects-unknown-keys.md):
documentation and strict-decode are the two halves of making a field safe to
hand-author, and a documented key that is still silently ignored on a typo is
only half-solved.

## Scope

### In scope

- **A struct-derived schema model + one generator.** Every `yaml:` field across
  the config structs contributes its key path, type, default, and a one-line
  description to a single in-memory model, rendered by one generator into
  commented YAML. This backbone is the load-bearing anti-drift device: the
  reference is *generated from the same structs that decode the file*, so it
  cannot silently diverge from what the loader accepts.
- **A generated, gitignored `aiwf.example.yaml`** at the repo root — the
  always-fresh reference. `aiwf init` and `aiwf update` write/refresh it from the
  generator every run, matching this repo's derived-artifact convention
  (`STATUS.md`, `site/`, materialized `.claude/`). It is machine-owned, so
  regenerating it is always safe.
- **Fresh-repo inline scaffold.** When `aiwf.yaml` does not yet exist, `aiwf init`
  writes it as the fully-commented scaffold (best first-touch onboarding). When
  it already exists, `init`/`update` never touch it — the generated
  `aiwf.example.yaml` sibling carries the reference instead.
- **The `init`/`update` re-run contract, documented in a consumer-facing surface.**
  A one-line `aiwf init --help` reassurance that re-running is idempotent and
  never overwrites an existing `aiwf.yaml`, `.claude/settings.json`
  (consent-gated per ADR-0015), or user git hooks — only derived artifacts
  refresh.
- **An anti-drift test** asserting the generated model covers every `yaml:`
  field on the config structs, so a newly-added block cannot ship undocumented.
- **Coordination with `G-0307`** so the documented key-set and the strict-decode
  key-set derive from one source; a mistyped key errors instead of no-op'ing.

### Out of scope

- **Editing the user's live `aiwf.yaml` after it exists.** The deliberately
  rejected alternative was a marker-managed reference block regenerated *inside*
  `aiwf.yaml` on every `update` (the ADR-0018 guidance-import pattern). Rejected
  in favor of a never-touch-the-user's-config posture: the reference lives in a
  generated sibling the user never owns.
- **Committing `aiwf.example.yaml`.** Gitignored + regenerated, not tracked;
  avoids churn/merge noise on every field addition.
- **Custom/consumer-defined config vocabulary.** The schema is derived from the
  hardcoded Go structs; no external-YAML schema definition.
- **The full strict-decode change itself.** Tracked by `G-0307`; this epic only
  ensures the documented and enforced key-sets share a source.

## Constraints

- **Anti-drift is structural, not vigilance.** The reference is generated from
  the config structs; the completeness of that generation is pinned by a test
  that fails when a `yaml:` field has no schema entry. A hand-kept doc is not an
  acceptable implementation.
- **Never edit a user-owned config file post-creation.** `init`/`update` may
  create `aiwf.yaml` when absent and may write/refresh the generated
  `aiwf.example.yaml`, but must never rewrite an existing `aiwf.yaml`. Consistent
  with aiwf's "no settings edits without consent" grain (ADR-0015).
- **`aiwf.example.yaml` is a derived artifact** — gitignored, regenerated by
  `init`/`update`, never hand-edited.
- **Discoverability is the acceptance bar.** Each shipped surface must be
  reachable from within a consumer repo (`aiwf init --help`, the generated file,
  and/or an embedded skill), not only from aiwf's source tree.

## Success criteria

<!-- Observable outcomes at epic close — not tests. -->

- [ ] After `aiwf init` in a fresh repo, the consumer's own `aiwf.yaml` documents
      every config block with its default and a one-line description.
- [ ] After `aiwf init`/`update` in a repo that already has `aiwf.yaml`, a
      generated `aiwf.example.yaml` at the repo root documents every block, and
      the existing `aiwf.yaml` is byte-unchanged.
- [ ] Every config block listed in the *Context* paragraph is present in the
      generated reference — none omitted — and adding a new `yaml:` field without
      a schema entry fails the test suite.
- [ ] `aiwf init --help` states the re-run is idempotent and lists what is never
      overwritten.
- [ ] The documented key-set and `G-0307`'s strict-decode key-set derive from one
      source, so a key documented in the reference is exactly a key the loader
      accepts.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Description source: parse Go doc comments via `go/ast` vs. an explicit description registry keyed by field path? | no | Settle in the schema-model milestone; `go/ast` reuses the comment already present (single source) but adds tooling; a registry is explicit but duplicates. |
| Defaults source-of-truth: read from the loader's defaults-applier vs. struct zero-values vs. a declared default per field? | no | Settle in the schema-model milestone; must render the *effective* default the loader applies, not a struct zero that lies. |
| Should the generator also back an always-live `aiwf config schema` verb, or is the generated file surface enough for the PoC? | no | Decide during milestone planning; YAGNI unless a second consumer earns the verb. |
| Placement/wording when both surfaces could apply (e.g. `aiwf.yaml` deleted but `aiwf.example.yaml` present)? | no | Milestone-level; define the init/update decision table when wiring the writers. |

## Risks (optional)

| Risk | Impact | Mitigation |
|---|---|---|
| Generated `aiwf.example.yaml` drifts from what the loader accepts | high | Generate from the same structs; pin field-coverage with a test; coordinate the key-set with `G-0307`. |
| Fresh-repo inline `aiwf.yaml` comments age (never refreshed post-init) | low | Accepted by design; the always-fresh `aiwf.example.yaml` sibling is the authority, and a static pointer routes there. |
| Gitignored reference is invisible to a teammate who hasn't run `update` | low | `update` is the documented setup step; the file regenerates on first run. |

## Milestones

<!-- Refined via aiwfx-plan-milestones. Ids assigned at allocation. -->

- `M-0231` — Struct-derived config-schema model + one generator; anti-drift field-coverage test; exports the reusable accepted-key registry · depends on: —
- `M-0232` — `init` inline-scaffolds a fresh `aiwf.yaml`; `init`/`update` write+refresh the gitignored `aiwf.example.yaml`; existing `aiwf.yaml` never touched; `init --help` re-run-safety line · depends on: `M-0231`

The sketched third item — "coordinate the documented key-set with `G-0307`
strict-decode" — is not a standalone milestone: E-0057 puts the strict-decode
change itself out of scope (it is `G-0307`'s), so it cannot ship independently
here. Its substance is folded in as `M-0231`'s exported accepted-key registry
plus a forward note recorded in `G-0307` (the *Coordinate with E-0057* section):
G-0307 consumes the registry and lands the equality test on its side.

## ADRs produced (optional)

<!-- None anticipated; the never-touch-user-config posture and generated-sibling
     decision may warrant one if a reader would later ask "why not the marker
     pattern?" — decide at wrap. -->

## References

- [`G-0360`](../../gaps/G-0360-aiwf-yaml-has-no-consumer-discoverable-schema-reference-for-its-blocks.md) — originating gap (absorbs the retired `G-0288`)
- [`G-0307`](../../gaps/G-0307-top-level-aiwf-yaml-decode-stays-non-strict-only-areas-rejects-unknown-keys.md) — strict-decode; the other half of safe-to-hand-author config
- `internal/config/config.go` — the config structs that are the single source of truth for the schema
- ADR-0015 — settings edits require explicit per-invocation consent (the posture this epic extends to config files)
- ADR-0018 — the marker-managed guidance-import pattern deliberately *not* used here (see *Out of scope*)
