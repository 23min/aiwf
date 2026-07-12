---
id: G-0302
title: aiwf check --fast omits in-memory contract-config validation
status: open
discovered_in: M-0193
---
## Problem

`aiwf check --fast` (`runFast` in `internal/cli/check/check.go`) bills itself
as the cheap in-memory content-rule pass (refs, ids, cycles, body-prose-id, AC
rules, tree-discipline, area-unknown) but skips **all** contract validation.
Excluding the verify half (`contractverify.Run`, which shells external
validators) is correct — it is not render-safe. But the cheap **in-memory**
config-correspondence half (`contractcheck.Run`) is excluded too, despite
being the same shape of rule as everything else `--fast` already runs, and it
emits error-severity findings.

This gap was originally framed around the statusline health glyph (G-0290 /
M-0193), on the assumption that `--fast` fed the glyph directly. ADR-0026
superseded that architecture: the glyph now reads only `aiwf doctor`-produced
`.claude/health.aiwf.json` files, never runs a check on the render path, and —
per ADR-0026's own "alternatives considered" — deliberately never will
surface tree-content findings (contract-config included) on that glyph; that
axis stays doctor's (install/config health), not check's (tree health). So
the statusline framing is moot and should not motivate a fix here.

The actual, narrower problem: `--fast` is internally inconsistent about what
"cheap in-memory content rule" means, carving out `contractcheck.Run`
alongside the genuinely expensive `contractverify.Run`. Anyone invoking
`aiwf check --fast` directly for a quick tree-health read — a CI pre-flight
step, an operator wanting fast feedback — gets a false-clean result on
contract-config errors that the full `aiwf check` (and the pre-push hook)
would catch. Zero effect in this contracts-free kernel repo today.

Related stale documentation: the code comment at `check.go`'s `runFast` doc
block and the comment near `internal/skills/embedded-statusline/statusline.sh:10`
still describe `--fast` as driving the statusline glyph — both predate
ADR-0026 and should be corrected alongside this fix.

## Direction

Fold the cheap in-memory `contractcheck.Run` (config-correspondence) into the
`--fast` rule set in `internal/cli/check/check.go` (`runFast`), keeping the
external `contractverify.Run` excluded. Add a contract-fixture test asserting
a contract-config error surfaces under `--fast`. While in there, correct the
stale statusline-glyph framing in `runFast`'s doc comment and the dead
`aiwf check --fast` reference in `statusline.sh` to reflect ADR-0026's actual
wiring (doctor-produced health files only).
