---
id: G-0382
title: 'Move logging: config into internal/config, not internal/logger'
status: addressed
discovered_in: M-0237
addressed_by:
    - M-0238
---
## What's missing

M-0237 parses and validates the `logging:` block's three optional keys
(level/format/destination) entirely inside internal/logger's own
YAMLConfig/ResolveConfig, bypassing internal/config's typed Config
struct and its reflection-driven schema registry (Schema(),
fieldDescriptions, AcceptedKeys(), GenerateExample()) that every other
top-level aiwf.yaml block goes through. This was the right call for
M-0237 itself (nothing programmatically rewrites logging:, so
internal/aiwfyaml's surgical-editor shape doesn't fit either — see
ADR-0017's corrected Consequences section), but the block still has no
home in the general schema surface once M-0238 wires the real
aiwf.yaml file-read.

## Why it matters

Without a Logging field on internal/config.Config, the logging: block
never appears in aiwf.example.yaml, never gets a fieldDescriptions
entry, and is invisible to the schema anti-drift test
(TestSchema_EveryFieldHasDescription) — a second, undiscoverable
aiwf.yaml-parsing pathway that diverges from every other block's
convention. M-0238 should add a Logging sub-struct + schema
description to internal/config, with internal/logger consuming the
already-parsed strings and owning only the slog-domain
validation/precedence it already has.