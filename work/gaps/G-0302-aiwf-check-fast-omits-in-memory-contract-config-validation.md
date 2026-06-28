---
id: G-0302
title: aiwf check --fast omits in-memory contract-config validation
status: open
discovered_in: M-0193
---
## Problem

`aiwf check --fast` (M-0193 — the render-safe content-only mode behind the
statusline health glyph, G-0290) skips **all** contract validation. Excluding
the verify half (`contractverify.Run`, which shells external validators) is
correct: it is not render-safe. But the cheap **in-memory** config-
correspondence half (`contractcheck.Run`) is excluded too, and it emits
error-severity findings.

Consequence: in a consumer repo that binds contracts in `aiwf.yaml`, an
error-severity contract-config finding is reported by the full `aiwf check`
(and blocks the pre-push hook) while the statusline health glyph shows clean.
This is a **false-clean** gap only — never a false-lit / always-on indicator —
and has zero effect in this contracts-free kernel repo today.

## Direction

Fold the cheap in-memory `contractcheck.Run` (config-correspondence) into the
`--fast` rule set in `internal/cli/check/check.go` (`runFast`), keeping the
external `contractverify.Run` excluded. Add a contract-fixture test asserting a
contract-config error surfaces under `--fast`. Surfaced by the M-0193 reviewer.
