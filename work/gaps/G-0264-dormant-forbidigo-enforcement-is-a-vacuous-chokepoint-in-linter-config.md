---
id: G-0264
title: Dormant forbidigo enforcement is a vacuous chokepoint in linter config
status: addressed
discovered_in: M-0167
addressed_by_commit:
    - 337c0869
---
## What's missing

The `.golangci.yml` `forbidigo` rules that enforce CLAUDE.md's "library code
never panics or calls `os.Exit`" guarantee use call-form patterns — `^panic\(`
and `^os\.Exit\(` — that match **zero** call sites under golangci-lint v2.12.2.
forbidigo matches the bare qualified function name (`panic`, `os.Exit`) without
the trailing call paren, so the `\(` form never fires. Confirmed empirically: a
temporary library `panic("probe")` in `internal/verb` produced `0 issues`.

The rules likely matched full call-text under an older forbidigo and silently
went dormant on a toolchain bump. A documented load-bearing enforcement now
guards nothing.

## Why it matters

This is the G-0259 pathology — a chokepoint that reads as a guarantee in the
enforcement table while detecting nothing — in a **surface the firing-fixture
meta-gate does not cover**. That meta-gate (`firing_fixture_presence`) scans
`internal/policies/*.go` for `Policy: "<id>"` construction lines; it has no
visibility into golangci-lint config rules. There is **no firing test for any
linter-config rule**, so a dormant `forbidigo` / `depguard` / etc. rule is
invisible until something it should have caught slips through.

## Fix direction

Two parts, mirroring G-0259's two-pronged shape:

1. Correct the dormant patterns to the form forbidigo v2 matches (`^panic$`,
   `^os\.Exit$`), verified against a probe.
2. Add a firing test for the linter-config rules — a fixture carrying a library
   `panic` / `os.Exit`, run through `golangci-lint`, asserting the rule flags it
   — so the rule cannot silently die again. This is the firing-evidence
   mechanism for the linter-config surface, parallel to the firing fixtures
   M-0166 adds for Go policies.

## Source

Discovered while running `wf-rethink` on M-0167/AC-1 (the no-time-now forbidigo
migration). The rethink's empirical forbidigo testing surfaced that the existing
`panic` / `os.Exit` patterns match nothing under v2.12.2. Same class as G-0259
(vacuous chokepoints); companion to G-0262 (corpus-wide vacuity audit).
