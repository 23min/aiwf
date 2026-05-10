---
id: G-0070
title: aiwf doctor has no --format=json envelope; M-0070's recommended-plugin-not-installed finding-code surfaces only as human text. Add JSON envelope when a JSON-consuming caller appears
status: open
discovered_in: M-0070
---

## What's missing

A `--format=json` flag on `aiwf doctor` that emits the kernel's standard JSON envelope (`{ tool, version, status, findings, result, metadata }` per `CLAUDE.md`'s CLI conventions). M-0070's AC-3 spec explicitly references the envelope shape:

> `finding.code: "recommended-plugin-not-installed"`, `finding.data: {plugin, marketplace, install_command}`

Today doctor emits human-readable text only. The `recommended-plugin-not-installed` finding-code string appears verbatim in the output (so a script can grep for it), but the structured `finding.data` payload doesn't exist as a queryable surface — a caller would have to regex the install command out of the prose continuation line.

The full fix is doctor-wide, not specific to the recommended-plugins check: every existing report section (`binary:`, `config:`, `actor:`, `skills:`, `ids:`, `filesystem:`, `hook:`, `pre-commit:`, `render:`, plus the new `plugins:` block) would need to map onto the envelope's `result` and `findings` arrays.

## Why it matters

`aiwf doctor` is the canonical "is this consumer healthy?" surface. As long as it's text-only:

- CI scripts that want to gate on specific finding codes have to parse prose.
- Downstream tooling (a future health dashboard, a webhook-driven alerting flow) can't consume doctor output cleanly.
- Per the kernel principle in `CLAUDE.md` ("CLI surfaces must be auto-completion-friendly… --format=json emits a structured JSON envelope for CI scripts and downstream tools"), doctor is currently inconsistent with the project's own convention.

Deferred (not blocking) because there's no JSON-consuming caller today — the forcing function hasn't appeared. Filed as a gap so it surfaces on `aiwf status` when it does. When implemented, M-0070's AC-3 contract is automatically satisfied without spec changes.
