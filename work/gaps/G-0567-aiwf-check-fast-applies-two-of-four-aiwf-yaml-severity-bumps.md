---
id: G-0567
title: aiwf check --fast applies two of four aiwf.yaml severity bumps
status: open
priority: high
discovered_in: M-0300
---
## What's missing

`aiwf check` composes four `aiwf.yaml` severity passes over the findings
`check.Run` produced: `ApplyTDDStrict`, `ApplyAreaRequiredStrict`,
`ApplyDocsStrict`, and `ApplyArchiveSweepThreshold`. `aiwf check --fast`
composes two of them — the first and the last. Its own doc comment says it runs
"the aiwf.yaml severity bumps", without qualification, and the omission is not
mentioned in the paragraph that carefully explains which area rules `--fast`
leaves out.

So the two surfaces render different severities for the same finding on the same
bytes whenever `areas.required` or the docs-strict knob is set. `--fast` runs
`AreaUnknown` — it emits the finding — it simply never applies the escalation
the full check applies to it.

## Why it matters

This is the class G-0558 opened and did not close: a cheaper surface disagreeing
with the authoritative gate about the same bytes. The direction is the safer one
of the two — `--fast` under-blocks rather than over-blocks, so a pre-flight
passes where the pre-push hook will later refuse — but an operator reaching for
`--fast` between commits is reading a verdict the gate does not share, which is
the whole reason the fast path was given a way to decline rather than guess.

It also has a second consumer: the statusline runs `--fast` on a TTL cache to
drive its health glyph, so a repo with `areas.required` set shows a clean glyph
while the push is blocked.

Discovered by M-0300's read-path agreement property, which fires on it once
repaired — making this the real violating state that AC's non-vacuity evidence
wants, in place of a stand-in.
