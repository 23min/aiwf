---
id: G-0199
title: finding hints must name the exact remediation command
status: open
---
## What's missing

Roughly half the finding hints in `internal/check/hint.go` describe *what* to fix without naming *which command* does it. Examples:

- `gap-addressed-has-resolver`: "list the resolving milestone(s) in `addressed_by:`..." — should say `aiwf promote <id> addressed --by <entity-id>` or `--by-commit <sha>`
- `titles-nonempty`: "set a non-empty `title:` in the frontmatter" — should say `aiwf retitle <id> "..."`
- `status-valid`: "use one of the allowed statuses listed above" — should say `aiwf promote <id> <status>`
- `adr-supersession-mutual`: "add this ADR to the other ADR's `supersedes:` list" — should say `aiwf promote ADR-NNNN superseded --superseded-by ADR-MMMM`
- `refs-resolve/*`: "check the spelling, or remove the reference" — no command named at all
- `acs-shape/status`: "use one of the allowed AC statuses" — should say `aiwf promote <id>/AC-N <status>`

The ~20 hints that already name a command (e.g. `ids-unique` → `aiwf reallocate`, `entity-body-empty/*` → `aiwf edit-body`, `acs-tdd-audit` → `aiwf promote ... --phase done`) demonstrate the target bar. The ~25 that don't are the gap.

Additionally, no policy test enforces the bar — a new finding can ship a command-free hint without CI catching it.

## Why it matters

Findings are the primary entry point when an LLM diagnoses a check failure. The hint text is the single highest-leverage discoverability surface for "how do I fix this?" An LLM that reads a hint naming a frontmatter field but no command will search `aiwf --help` for a `set` verb, fail to find one, and conclude there's no path — exactly the failure a downstream consumer reported against `gap-addressed-has-resolver`. The LLM never ran `aiwf promote --help` because `promote`'s top-level one-liner doesn't mention resolver fields.

Per CLAUDE.md: "kernel functionality must be AI-discoverable... if an AI assistant has to grep source to learn a kernel capability, the capability is undocumented." The finding → hint → command chain is the shortest path; when hints omit the command, the chain breaks and the LLM falls through to source grep or gives up.

## Scope

1. Sweep all entries in `hintTable` (`internal/check/hint.go`). Each hint must contain a backtick-delimited `aiwf ...` or `git ...` command with placeholder ids, or explain why no single command applies and name the closest verb + `--help`.
2. Add a policy test (`PolicyFindingHintsNameCommand` or similar in `internal/policies/`) that asserts every hint contains at least one backtick-delimited command pattern. New findings that ship a command-free hint fail CI.
3. Single deliverable: one sweep commit updating `hint.go` + one commit adding the policy test.
