---
id: G-0360
title: aiwf.yaml has no consumer-discoverable schema reference for its blocks
status: addressed
addressed_by:
    - M-0231
    - M-0232
---
## Problem

No consumer-discoverable surface documents the `aiwf.yaml` schema. Every config
block — `tdd`, `allocate`, `archive`, `guidance`, `worktree`, `entities`,
`areas`, `agents`, `status_md`, `html`, `tree`, `hosts` — is documented **only**
in `internal/config/config.go` struct doc comments. That is source-level: not
reachable via `aiwf <verb> --help`, an embedded skill, the shipped guidance
fragment, or the consumer's `CLAUDE.md`.

The kernel principle "Kernel functionality must be AI-discoverable" names the
design docs as an acceptable channel — but **design docs do not materialize into
consumer repos**. So a config field documented only in a design doc is
discoverable for an AI reading the aiwf source tree, yet invisible to the
consumer who must actually author the field. `aiwf.yaml` config is
*consumer-operating* knowledge currently stored only in *repo-development*
surfaces — the wrong side of the line `CLAUDE.md` itself draws.

Concrete trigger: the `agents:` model/effort block (G-0353) shipped in v0.24.0.
A consumer can learn it exists only from the release notes or by reading aiwf's
design docs / source — not from anything in their own repo. `aiwf init` writes a
*minimal* `aiwf.yaml`, so a fresh repo gives no hint the block (or any other)
exists. `--help` on `init`/`update` says nothing about it.

The same blind spot covers `init`/`update`'s own *contract*. That re-running
either verb is idempotent and never overwrites an existing `aiwf.yaml`,
`.claude/settings.json` (consent-gated per ADR-0015), or user git hooks
(marker-managed — chained to `<hook>.local`, not clobbered), while only derived
artifacts are byte-refreshed, is documented only in ADR-0015 and source
comments. A user reasonably reads "init" as a destructive reset and hesitates to
re-run it — a discoverability failure of the same class as the missing schema.

## Why it matters

A configuration surface a user cannot discover is a feature that effectively
does not exist for most consumers. The knob works, is validated, and ships — but
adoption depends on a user stumbling onto a changelog line. This is a systemic
hole across the whole config file, not one field.

## Absorbs G-0288; complements G-0307

This generalizes **G-0288** ("`areas:` config schema has no AI-discoverable doc
surface", discovered in M-0179) from the `areas:` block to the entire
`aiwf.yaml` schema. G-0288 is retired as folded into this gap. It complements
**G-0307** (top-level `aiwf.yaml` decode stays non-strict): documentation and
strict-decode are the two halves of making a config field safe to hand-author —
a documented key that is still silently ignored on a typo is only half-solved.

## Direction

Land a consumer-discoverable reference covering **every** block, not one
instance. Candidate homes (a fork to settle, each with a tradeoff):

- **`aiwf init` scaffolds a fully-commented `aiwf.yaml`** — every block written
  as commented-out lines with its default, the customary self-documenting-config
  pattern. Discoverable in the consumer's own repo, zero extra commands. Wrinkle:
  keeping the commented template fresh across `aiwf update` in a user-editable
  file is harder than a static init-only scaffold (comments the user may have
  edited or deleted); a marker-managed block or a regenerate-only-if-untouched
  rule needs designing.
- **A shipped `aiwf-config` skill** documenting the schema — materializes into
  `.claude/`, refreshed byte-equal on every `aiwf update`, no staleness risk.
  Discoverable by an AI assistant; less so by a human not reading skills.
- **An always-live `aiwf config schema` verb (or richer `--help`)** — the schema
  is derived from the live config structs, so it can never drift. A new verb to
  design and test.

Pick one (or a primary + backstop); cover the full block list above so the whole
schema is reachable, not just the block that prompted this. Sequence with G-0307
so a mistyped key errors instead of no-op'ing once the shape is documented.

**Also document the `init`/`update` re-run contract**, not just the field
schema. The chosen surface — plus a one-line `aiwf init --help` reassurance —
should state that re-running is idempotent and never overwrites an existing
`aiwf.yaml`, `.claude/settings.json`, or user git hooks (only derived artifacts
refresh). This is the same consumer-operating-knowledge-in-a-repo-development-
surface failure, and a single sentence defuses the common "is `init` safe to
re-run?" hesitation that the verb's name invites.

## Scope

The chosen documentation surface + tests asserting it stays in sync with the
config structs (the anti-drift assertion is the load-bearing part — a hand-kept
doc rots). The `aiwf init --help` re-run-safety line. Retiring G-0288.
Coordinating with G-0307's strict-decode work.
