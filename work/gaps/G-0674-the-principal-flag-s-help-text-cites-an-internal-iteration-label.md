---
id: G-0674
title: The --principal flag's help text cites an internal iteration label
status: open
discovered_in: M-0332
---
## What's missing

Every verb accepting `--principal` describes the flag as *"the human/<id> the
actor is acting on behalf of (required when --actor is non-human; gates the
verb through the I2.5 allow-rule)"*. `I2.5` is this repo's own iteration
label. It names nothing a consumer can look up, and `aiwf promote --help` is
where they meet it.

The sentence is registered identically at 13 call sites across 12 files under
`internal/cli/` — `add`, `cancel`, `editbody`, `milestone` (twice), `move`,
`promote`, `reallocate`, `rename`, `renamearea`, `retitle`, `setarea`,
`setpriority`. A correction has to reach every one, which is the second half
of the defect: the string has no single home, so the copies can diverge under
a partial edit.

Out of scope: the same label in `internal/cli/cliutil/provenance.go` and
`scopes.go` Go comments. Those are internal and reach no consumer.

## Why it matters

The label class is already retired from shipped skill markdown, but the
chokepoint that keeps it out — the `skill-body-id` check and the
shipped-surface rule it mirrors — scans `internal/skills/embedded`,
`embedded-rituals` and `embedded-guidance` only. Flag help is consumer-facing
output built from Go string literals, so it sits outside every existing
check. Nothing reports it now and nothing will.

The behaviour the sentence describes is real: a non-human actor does need
`--principal`. Only the label misleads, and a reader who tries to resolve it
finds no such concept in any surface aiwf ships. What belongs in its place is
the rule's own name rather than the iteration that introduced it.
