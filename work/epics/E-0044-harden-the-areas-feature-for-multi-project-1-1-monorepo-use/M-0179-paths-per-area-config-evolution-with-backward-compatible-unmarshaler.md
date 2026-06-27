---
id: M-0179
title: paths per-area config evolution with backward-compatible unmarshaler
status: in_progress
parent: E-0044
tdd: required
acs:
    - id: AC-1
      title: Dual-form unmarshal parses string, object, mixed, and no-paths members
      status: met
      tdd_phase: done
    - id: AC-2
      title: 'Legacy string-form parity: zero migration at the parse layer'
      status: met
      tdd_phase: done
    - id: AC-3
      title: Schema validation rejects malformed members with clear per-arm errors
      status: met
      tdd_phase: done
    - id: AC-4
      title: rename-area preserves paths through a lossless round-trip
      status: met
      tdd_phase: done
    - id: AC-5
      title: MemberNames is the single source of truth for name-based readers
      status: open
      tdd_phase: red
---
## Goal

Evolve `config.Areas` from a flat label list to label+location — `members: [{name, paths}]` — via a backward-compatible custom unmarshaler that still accepts the legacy `members: [app-a, app-b]` string form. This is the keystone: it gives each area the path oracle that the bijection check and all of Tier 2 depend on.

## Context

The area is label-only today; nothing ties it to where the project's code lives. Without that anchor the kernel can neither verify a tag nor derive one. This milestone adds the optional `paths:` glob per member while preserving zero-migration for every existing string-form config. Everything downstream — bijection coverage, mistag detection, auto-derive — is dead without it.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac <id> --title "..."` at start-milestone. -->

Candidate behaviors to formalize at start-milestone:

- The object form `members: [{name: app-a, paths: ["projects/app-a/**"]}]` parses into the area set with its globs.
- The legacy string form `members: [app-a, app-b]` parses unchanged (no paths; behaves exactly as E-0043).
- The mixed form (some string, some object members) parses.
- `paths` is optional within the object form; an object member with no paths behaves label-only.
- Validation rejects malformed members (non-string in string position, non-list paths, empty name) with a clear error; round-trips clean.
- A spec-sourced test pass enumerates string-only / object-only / mixed / no-paths-object forms (per CLAUDE.md "spec-sourced inputs").

## Constraints

- Backward-compatible and zero-migration — the named top risk of the epic; the dual-form unmarshaler is tested across every form before this milestone closes.
- `area` stays single-valued; `paths` is a per-member list of globs describing the project's location, not a second grouping axis.
- `paths:` describe the consumer's project code, never aiwf's own kind-partitioned entity layout (ADR-0004 untouched).

## Design notes

- Extend the existing custom `Areas.UnmarshalYAML` (it already decodes via `yaml.Node` to reject non-string members) to additionally accept mapping nodes.
- Glob matcher and any new dependency are decided here (open question); prefer an already-vendored matcher; justify any new dep per Go conventions.
- The flat-list → object schema evolution is an ADR candidate, harvested at wrap.

## Out of scope

- Consuming the paths — the bijection check, mistag detection, and auto-derive are their own milestones.

## Dependencies

- None. Foundation for the bijection coverage check and both Tier-2 milestones (mistag detection, auto-derive).

## References

- `internal/config/config.go` — the `Areas` type and `UnmarshalYAML` extended here.

### AC-1 — Dual-form unmarshal parses string, object, mixed, and no-paths members

**Property.** `areas.members` decodes into `Members []Member` from a spec-sourced enumeration
of forms, each landing as the right `Member`, in declaration order:

- string-only: `[app-a, billing]` → two members, `Paths` nil.
- object-with-paths: `[{name: app-a, paths: [projects/app-a/**]}]` → member carries its paths.
- object-no-paths (absent key): `[{name: app-a}]` → `Paths` nil.
- object-explicit-empty: `[{name: app-a, paths: []}]` → `Paths` normalizes to nil (the same
  canonical empty as absent; explicit-empty and absent express the identical "no paths" intent).
- mixed: `[app-a, {name: billing, paths: [svc/billing/**]}]` → string member nil, object member
  carries paths; order `[app-a, billing]` preserved.

A non-`!!str` scalar member (`42`, `true`) is rejected via the explicit `Tag == "!!str"` check,
routing to AC-3 arm a4 rather than silently becoming `Member{Name: "42"}`.

**Mechanical assertion.** `TestAreas_UnmarshalDualForm` (`internal/config/config_test.go`) — a
spec-sourced table (comment cites the form enumeration per CLAUDE.md "spec-sourced inputs"),
one row per form, asserting the full `[]Member` slice (name, paths, order) via go-cmp;
explicit-empty and absent rows both assert `Paths == nil`.

**Vacuity.** Dropping the mapping-node branch sends object members to the `Kind != ScalarNode`
error path, reddening the object/mixed/explicit-empty rows.

**Builder note.** yaml.v3 decodes `paths: []` into a non-nil empty slice; `UnmarshalYAML` must
normalize `len(paths)==0 → nil` for the explicit-empty row to equal the absent row.

### AC-2 — Legacy string-form parity: zero migration at the parse layer

**Property.** A string-only config parses byte-for-byte as E-0043: every member has nil
`Paths`; `MemberNames()` equals the input list in order; `default`/`required` validation
semantics are unchanged.

`config.Write` needs no change. It is the bootstrap writer — create-once at `aiwf init` from an
empty `&config.Config{}` (initrepo.go), and it refuses to overwrite an existing file
(config.go). It is therefore structurally incapable of ever serializing a populated `Areas`: at
init the areas are empty (omitted before marshal); after init the file exists, so it refuses; a
future "edit areas" verb cannot use it either and must go through `aiwfyaml` (AC-4's path). The
`[]string`→`[]Member` change cannot alter init's output (empty `Areas` stays omitted, verified
against yaml.v3 `isZero`), so the existing `initrepo` tests already pin it. No custom
`MarshalYAML` is added — it would be dead code guarding an unreachable branch.

**Owns the existing-test migration.** `internal/config/area_test.go` indexes `cfg.Areas.Members[i]`
as a string (`:36`, `:158`), which will not compile against `[]Member`; this AC migrates those
accessors to `.Members[i].Name` (or `MemberNames()[i]`). The message asserts at `area_test.go:93,98`
(`"not a string"`) are reconciled with AC-3 arm a4's wording in the same pass.

**Mechanical assertion.** `TestAreas_StringFormParity` (new) + the migrated E-0043
`validate()`/`default`/`required` tests staying green. Asserts `MemberNames() == input`, every
`Paths` is nil, and a string-form config with a `default` and with `required: true` validates
exactly as before.

**Vacuity.** If the scalar branch dropped the member's name, coerced `Paths` to a non-nil empty
slice, or changed default/required handling, the parity assertions redden.

### AC-3 — Schema validation rejects malformed members with clear per-arm errors

**Property.** A malformed member is rejected at decode (`UnmarshalYAML`) or at `validate()`
with a distinct, clear error that names the offending member/path; valid shapes pass. Each arm
maps to its firing site — two fire at decode, three at validate; the "names the member/path"
guarantee requires the decode arms to wrap the raw yaml.v3 error with member context rather
than shipping a bare `"cannot unmarshal !!str into []string"`:

- a1 (validate): object member with empty or absent `name` → error naming the empty-name rule.
- a2 (decode): a `paths` value that is not a list (scalar/mapping in `paths` position) → the
  member decode fails; the error is wrapped to name the offending member.
- a3 (validate): a path entry that is empty or whitespace-only, or has leading/trailing
  whitespace → error, mirroring the member-name hygiene rule. String hygiene, inside the
  glob-deferred decision — not glob validation.
- a4 (decode): a member node that is neither a `!!str` scalar nor a mapping (a bare sequence, or
  a non-`!!str` scalar like `42`/`true`) → the generalized `Tag == "!!str"` guard; error wrapped
  to name the offending member.
- a5 (validate): a duplicate name across forms — one string-form `app-a` and one object-form
  `app-a` → duplicate-member error (uniqueness is on the derived name).

Name rules (trim-clean, uniqueness) are enforced across both forms.

**Mechanical assertion.** `TestAreas_RejectsMalformed` (`internal/config/config_test.go`) — a
table with one input per arm a1–a5 (plus the cross-form-duplicate row), each asserting a
non-nil error whose message names the offending member/path (decode arms assert the wrapped
context, not the bare yaml message).

**Vacuity.** Removing any single guard reddens exactly its row.

### AC-4 — rename-area preserves paths through a lossless round-trip

**Property.** Renaming a member in an object-form config rewrites only the renamed member's
`name` and preserves every member's `paths`; bare-string (paths-less) members stay bare; the
renamed member's paths follow its new name; non-renamed members' paths are untouched.

**Plumbing.** Paths must flow to the writer, not merely be writable. `cliutil.ConfiguredAreas`
stays name-only (so the genuinely name-only callers — render, status, setarea, completion —
keep `[]string`); a new `cliutil.ConfiguredAreaMembersFull` helper feeds the rename-area
handler the full `[]config.Member`; `verb.RenameArea`'s param changes `[]string` →
`[]config.Member` with an order-preserving rebuild that renames only the matching `.Name` and
retains each member's `Paths`; the verb maps `config.Member` → `aiwfyaml.AreaMember` at the
`SetAreas` call. `aiwfyaml` keeps its zero-dep-on-config layering via the local `AreaMember`
struct; `marshalAreasBlock` emits object form for members with paths and bare strings for
members without.

**Mechanical assertion.** `TestRenameArea_PreservesPaths` (`internal/cli/integration/`) —
drive `aiwf rename-area app-a application-a` against a fixture whose `aiwf.yaml` declares
`[{name: app-a, paths: [projects/app-a/**]}, {name: billing, paths: [svc/billing/**]}, plat]`.
Parse the rewritten YAML (not a substring grep, per CLAUDE.md "substring vs structural") and
assert: `application-a` carries `[projects/app-a/**]`; `billing` still carries
`[svc/billing/**]`; `plat` is still a bare string.

**Vacuity.** Reverting `marshalAreasBlock` to flat-string emission drops all paths, reddening
every per-member path assertion.

**Determinism.** rename-area's trailer/commit determinism (entities sorted by id, one OpWrite
per entity) is untouched; path emission iterates each member's `Paths` in declaration order, so
the single `aiwf.yaml` OpWrite stays deterministic.

### AC-5 — MemberNames is the single source of truth for name-based readers

**Property.** The name-based read sites (`add --area` validation, the `area-unknown` check, the
grouping resolver, `--area` completion) behave correctly when the config is object-form —
asserted against the absolute expectation, not merely "object-form == string-form" (a broken
`MemberNames()` returning nil would make both forms reject everything identically and a pure
differential test would pass green).

**Mechanical assertion.** `TestObjectFormConfig_NameReadersWork` (`internal/cli/integration/`)
— load an object-form config declaring members `[app-a, billing]` (with paths) and assert the
absolute outcomes: `aiwf add gap --area app-a` is accepted; `aiwf add gap --area app-z` is
rejected (undeclared); `aiwf check` treats an `app-a`-tagged entity as known and an
`app-z`-tagged entity as `area-unknown`; and `MemberNames()` returns `[app-a, billing]` in
declaration order.

**Vacuity.** A `MemberNames()` that returns nil/empty reddens the "app-a accepted" and order
assertions; a name-consuming reader still reading a stale shape fails compile.

**Coverage note.** The grouping resolver (resolver.go:432) and `--area` completion are not
directly asserted here, but the `[]string`→`[]Member` type change makes their switched lines
compile errors, and the diff-scoped coverage gate (G-0067) forces those lines under test at
wrap. Do not `//coverage:ignore` them — write the covering case.

