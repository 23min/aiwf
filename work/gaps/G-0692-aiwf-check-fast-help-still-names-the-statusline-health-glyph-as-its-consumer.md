---
id: G-0692
title: aiwf check --fast help still names the statusline health glyph as its consumer
status: addressed
addressed_by_commit:
    - 3fdae4b96
---
## What's missing

`internal/cli/check/check.go:60` registers `--fast` with help text that ends
"render-safe (sub-second) for the statusline health glyph and CI pre-flight
(G-0290)", and the `runFast` doc block at `check.go:345-347` repeats it: "the
statusline runs it on a TTL cache to drive the ⚠ health glyph, and CI scripts can
use it as a fast pre-flight." Both describe the wiring G-0290 shipped and ADR-0026
retired. The statusline's health glyph now reads producer-written health files and
spawns no kernel verb, and no CI job invokes the flag.

Measured in the devcontainer (Linux) against the source at `841031efd` and the
shipped `aiwf` on PATH (`v0.35.0`):

```
$ aiwf check --help 2>&1 | grep -- '--fast'
      --fast                run the in-memory content rules (refs, status, ids, cycles, body-prose, ACs) plus tree-discipline, skipping the trunk read / provenance / FSM-history / metrics / contract-validation layer; render-safe (sub-second) for the statusline health glyph and CI pre-flight (G-0290)

$ sed -n 9,12p internal/skills/embedded-statusline/statusline.sh
# The render spawns no kernel verb. The CI probe is the only subprocess that
# can cost anything: it is cached in /tmp with a TTL, so the hot (cache-hit)
# path stays fast and a miss pays one `gh run list`. The health glyph reads
# the producer files under `.claude/` and spawns nothing at all.

$ grep -rn 'aiwf check\|aiwf doctor' .github/workflows/*.yml
.github/workflows/go.yml:344:      - name: Build and run aiwf doctor --self-check
```

A tree-wide search for `--fast` outside `check.go` and test files (`Makefile`,
`.github/`, `scripts/`, `internal/`, `cmd/`, `docs/`) returns only prose: the
`aiwf-check` skill's finding table, ADR-0026, ADR-0030 and four initiative
documents. Nothing invokes the flag; outside `check.go`'s own flag dispatch no
non-test Go code reaches `runFast`; and
`TestStatusline_RenderInvokesNoKernelVerb`
(`internal/policies/statusline_content_test.go:213`) pins that the statusline
spawns no verb at all.

Expected: the flag's help names a consumer that exists. Observed: neither named
consumer invokes it. ADR-0026 itself lists `aiwf status`, `aiwf doctor` and
scripts as the flag's remaining consumers, and the same search finds none of those
invoking it either.

G-0302, whose subject is the contract-config validation `--fast` omits, names the
`runFast` doc block as related stale text; it does not name the help string, which
is the copy an operator and an assistant read.

## Why it matters

`--help` is the kernel's first-line documentation and, by this repo's
discoverability rule, the surface an assistant reads to learn what a flag is for.
This one sends a reader to two consumers that do not exist and away from the record
that explains the current wiring. It also states a constraint owed to them: a
sub-second, render-safe budget. Anyone deciding whether `--fast`'s rule set or
config handling may change — the subject of G-0302 and of G-0691 — reads a latency
contract with a caller that does not exist, and anyone debugging the health glyph
is pointed at a verb the statusline never runs.
