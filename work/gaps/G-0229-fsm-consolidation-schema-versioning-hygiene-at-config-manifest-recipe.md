---
id: G-0229
title: FSM consolidation + schema-versioning hygiene at config/manifest/recipe
status: open
---
## What's missing

Three small consistency gaps in how the kernel represents closed-set and on-disk schemas:

1. **FSM single-source consolidation.** Today the per-kind state-set lives in two places: `AllowedStatuses` and the `transitions` map. `TestKindFSM_StateSetAgreement` polices drift at test time; the cleaner fix is to derive `AllowedStatuses(kind)` from `transitions[kind]` keys ∪ values at boot, eliminating the second source entirely. Property tests stay (they now assert a property that's structurally true, which is fine).
2. **Recipe frontmatter has no `version:` field.** `internal/recipe/embedded/` ships embedded cue + jsonschema recipes whose shape may evolve. Today there is no versioned knob — a future shape change has no migration path declared. Add an explicit `version: 1` field, validate it on read, fail on unknown values.
3. **Manifest outer decode tolerates unknown fields.** `internal/manifest/manifest.go:74`'s outer decode uses bare `yaml.Unmarshal` (not `KnownFields(true)`) — silently drops unknown top-level fields. The inner `Entry` decode is strict; the outer is not. Tighten to `KnownFields(true)` and ship a v1→v2 migration shim if/when v2 lands (currently `supportedVersion = 1` is checked, but unknown-field silence undermines the version check's reach).
4. **`aiwf.yaml` schema_version decision.** Today the binary version is *de facto* the schema version (parser tied to the build). Either formalize that ("the binary version IS the aiwf.yaml schema version; cross-version reads MUST go through `aiwf update`") in CLAUDE.md, or add a top-level `schema_version:` field with explicit forward-compat rules. Either is a clean answer; the absence of an answer is the gap.

## Why it matters

C4's verdict was Strong precisely because aiwf has explicit migration paths *when* changes occur (`aiwf rewidth` for ADR-0008, Legacy* capture for deprecated fields, semver-tagged releases). The three gaps above are "places where a change *could* occur and the path isn't declared yet" — the cheapest moment to add the version field is before the first change you'd want to migrate.

## Source

`docs/pocv3/health-scorecard-2026-06-04.md` §C1 (move 1: FSM consolidation), §C4 (moves 1–3, refuting evidence on manifest KnownFields).
