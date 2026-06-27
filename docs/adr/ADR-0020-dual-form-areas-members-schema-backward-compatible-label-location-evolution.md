---
id: ADR-0020
title: 'Dual-form areas.members schema: backward-compatible label+location evolution'
status: proposed
---
## Context

E-0043 shipped `area` as a label-only tag: `aiwf.yaml: areas.members` was a flat list of
strings, each a member label the `area` frontmatter field validates against. E-0044 hardens
the feature for the 1:1 monorepo by anchoring each area to the path glob of the project it
represents (the oracle the bijection/coverage and mistag checks need). That requires evolving
each member from a bare label into label+location — `{name, paths}`.

The named top risk of the epic is that this evolution silently breaks every existing config.
A consumer on `v0.17.0` declares `members: [app-a, app-b]`; that form must keep parsing and
behaving byte-for-byte after the upgrade, with zero migration.

## Decision

Evolve `config.Areas.Members` from `[]string` to `[]Member{Name string; Paths []string}`,
served by a backward-compatible custom unmarshaler. The single in-memory representation is the
list of `Member`; the member-name set every name-consuming reader needs is the derived
`MemberNames() []string` (single source of truth — names are projected, never stored twice).

`Areas.UnmarshalYAML` accepts each member node in either form, plus any mix:

- legacy scalar string (`- app-a`) → `Member{Name: "app-a"}`, `Paths` nil — the E-0043 shape,
  unchanged;
- `{name, paths}` mapping → a member carrying its path globs;
- an explicit `paths: []` and an absent `paths` both normalize to nil (identical "no paths"
  intent).

A bare scalar must be a YAML `!!str`; an unquoted `42`/`true`/`~` is rejected rather than
silently coerced into a member. Decode-time failures (a non-list `paths`, a node that is
neither scalar nor mapping) are wrapped to name the offending member.

`paths` is validated as well-formed strings only (non-empty, trim-clean) at this layer — no
glob matching and no glob-syntax validation. The two writers split by role: `config.Write` is
the create-once bootstrap at `aiwf init` (empty config only); every post-init edit to the
areas block routes through the comment-preserving `aiwfyaml` writer, which emits a bare string
for a paths-less member and the mapping form for a member with paths, so a legacy config
round-trips byte-for-byte.

## Consequences

- **Zero migration.** A `v0.17.0` string-form config parses and behaves exactly as before;
  the object form is purely additive and `paths` is optional even within it.
- **The dual form is the compat window in both directions.** It absorbs the legacy shape
  (backward) and admits future per-member fields via the same mapping branch (forward) — a new
  field is added to `Member` and the mapping decode picks it up, while the scalar branch leaves
  it zero. The `areas.required` bool may later evolve to a per-kind list by the same trick.
- **The writer must preserve paths.** `aiwf rename-area` carries each member's paths through
  the rewrite (M-0179/AC-4); a flat-string writer would silently drop them.
- **Glob matching and its dependency are deferred to M-0180**, the first call site that matches
  a path against a directory; the chosen matcher leans `github.com/bmatcuk/doublestar/v4`, with
  the dependency justified there per Go conventions rather than imported ahead of its consumer.
- **The reserved `global` area value** (a separate, proposed decision) layers on top as a value
  of the entity's `area` field, not a change to this member schema — the two are orthogonal.
- **Known follow-ups** (filed against M-0179): a mistyped member key silently drops paths under
  the config-wide non-strict decode (G-0287, to be hardened when paths become load-bearing in
  M-0180); the `areas:` config schema has no AI-discoverable doc surface (G-0288).
