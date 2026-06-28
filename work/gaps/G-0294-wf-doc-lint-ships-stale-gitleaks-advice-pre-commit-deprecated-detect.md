---
id: G-0294
title: 'wf-doc-lint ships stale gitleaks advice: pre-commit + deprecated detect'
status: open
---
## Problem

The embedded `wf-doc-lint` ritual (`internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-doc-lint/SKILL.md`) ships two pieces of stale gitleaks advice to consumers:

1. **Pre-commit placement.** It recommends running gitleaks as a *pre-commit* hook (§5, around lines 60/65/66/82). aiwf's own enforcement (G-0291 / G-0292) concluded the opposite: a secret is not exposed until **push**, so the scan belongs at **pre-push** (the real trust boundary) plus an operator-independent **CI** chokepoint — pre-commit just taxes every commit's latency without being the boundary. The dogfooding repo now ships advice it has decided against for itself.

2. **Deprecated subcommand.** The example shows `gitleaks detect --config=.gitleaks.toml --no-banner --no-git`. gitleaks v8.x deprecated `detect` in favour of `gitleaks git` (history) and `gitleaks dir` (filesystem). The example should use the current subcommands.

## Direction

Update the `wf-doc-lint` skill (edit the embedded snapshot — authoring source of truth per ADR-0016) to:

- Recommend the secret-scan as a **pre-push hook + CI job** (operator-independent), framing pre-commit as a latency-taxing non-boundary; the consumer still owns their `.gitleaks.toml`.
- Replace `gitleaks detect --no-git` with `gitleaks dir` (filesystem scan) / `gitleaks git` (history scan) per the consumer's need.
- Optionally cite aiwf's own arrangement (the `gitleaks` CI workflow + the pre-push hook + a fingerprint `.gitleaksignore` for accepted history) as the worked example.

This is consumer-facing ritual content, distinct from aiwf's own gate (which G-0292 already armed) — hence a separate gap rather than folding it into the G-0292 change.

## Provenance

Surfaced during G-0292 (arming aiwf's own gitleaks gate): both the independent code review and the wf-rethink design audit flagged that the shipped `wf-doc-lint` advice contradicts the lesson G-0291 / G-0292 just encoded, and additionally uses the deprecated `gitleaks detect` subcommand.
