---
id: G-0169
title: Wire --format=json into non-FinishVerb verbs and read-display commands
status: open
discovered_in: M-0143
---
## What's missing

M-0143 (D-0013, decision A2) wired `--format`/`--pretty` and a `status:error` envelope into every mutating verb that routes through the shared `cliutil.FinishVerb` / `DecorateAndFinish` chokepoint (14 verbs). Several commands fall outside that chokepoint and still lack a JSON envelope:

- **Mutating, bespoke output path:** `aiwf import` (multi-entity import) and `aiwf rewidth` (id-width migration) emit their own multi-line/multi-commit output rather than routing through `FinishVerb`, so the uniform rollout did not reach them.
- **Read / generate commands:** `aiwf contract recipes`, `aiwf contract recipe show`, and `aiwf render roadmap` have no `--format=json` surface.

These are recorded as `formatExempt` entries (with this gap id) in `TestFormatFlagUniformRollout_AC4` (`internal/cli/integration/format_coverage_test.go`).

## Why it matters

The envelope surface is uniform for the common mutating verbs but not yet complete. A CI consumer scripting `aiwf import --format=json` or `aiwf render roadmap --format=json` gets "unknown flag". Low urgency — no consumer has asked — but it is the remaining gap between "uniform" and "universal".

## Proposed fix shape

For `import`/`rewidth`: either refactor their output onto a shared envelope emitter (the `cliutil.OutputFormat` helpers added in M-0143 are reusable) or give each a bespoke `--format=json` envelope. For the read/generate commands: mirror the read-verb pattern (`--format`/`--pretty` + `render.Envelope`). Remove each command's `formatExempt` entry as it gains the flag — the AC-4 test then enforces it.
