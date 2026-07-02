---
id: G-0294
title: 'wf-doc-lint ships stale gitleaks advice: pre-commit + deprecated detect'
status: addressed
addressed_by:
    - M-0199
---
## Problem

The embedded `wf-doc-lint` ritual (`internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-doc-lint/SKILL.md`) has several defects around its path-leak / secret-scan content.

### Stale gitleaks advice

1. **Pre-commit placement.** It recommends running gitleaks as a *pre-commit* hook (§5, around lines 60/65/66/82). aiwf's own enforcement (G-0291 / G-0292) concluded the opposite: a secret is not exposed until **push**, so the scan belongs at **pre-push** (the real trust boundary) plus an operator-independent **CI** chokepoint — pre-commit just taxes every commit's latency without being the boundary. The dogfooding repo now ships advice it has decided against for itself.

2. **Deprecated subcommand.** The example shows `gitleaks detect --config=.gitleaks.toml --no-banner --no-git`. gitleaks v8.x deprecated `detect` in favour of `gitleaks git` (history) and `gitleaks dir` (filesystem). The example should use the current subcommands.

### Internal inconsistency around the path-leak check (added by a skill audit)

3. **Count drift.** The body lists *five* checks (path-leak is #5) but the Workflow says "run each of the **four** checks," the output template has four sections, and the `description:` enumerates four — so #5 is silently skipped.

4. **Anti-pattern contradiction.** The anti-pattern ("doc-lint findings ... block-on-zero is too strict") contradicts check #5's own "deserves a real chokepoint" (a blocking pre-commit/CI gate).

5. **Scope mismatch.** The skill's scope is the docs tree (`docs/`), but path-leak scanning is repo-wide (aiwf's gitleaks runs over all committed text / full history).

## Direction

Update the `wf-doc-lint` skill (edit the embedded snapshot — authoring source of truth per ADR-0016):

- **Reframe the path-leak check as a separate "Related: repo-wide secret / path-leak scanning (standalone tool)" section, explicitly NOT one of the four doc heuristics.** This resolves the count (the four heuristics stay four), the anti-pattern (it scopes to the four heuristics; the deterministic standalone tool legitimately gates), and the scope (the standalone scan is repo-wide, the four heuristics are docs-scoped).
- In that section, recommend the secret-scan as a **pre-push hook + CI job** (operator-independent), framing pre-commit as a latency-taxing non-boundary; the consumer still owns their `.gitleaks.toml`.
- Replace `gitleaks detect --no-git` with `gitleaks dir` (filesystem scan) / `gitleaks git` (history scan) per the consumer's need.
- Optionally cite aiwf's own arrangement (the `gitleaks` CI workflow + the pre-push hook + a fingerprint `.gitleaksignore` for accepted history) as the worked example.

This is consumer-facing ritual content, distinct from aiwf's own gate (which G-0292 already armed) — hence a separate gap rather than folding it into the G-0292 change.

## Provenance

Surfaced during G-0292 (arming aiwf's own gitleaks gate): both the independent code review and the wf-rethink design audit flagged that the shipped `wf-doc-lint` advice contradicts the lesson G-0291 / G-0292 just encoded, and additionally uses the deprecated `gitleaks detect` subcommand. The count / anti-pattern / scope facets (3-5) were added after a later audit of the embedded skills against CLAUDE.md.
