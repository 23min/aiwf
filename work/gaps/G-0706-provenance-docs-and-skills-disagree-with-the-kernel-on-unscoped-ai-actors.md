---
id: G-0706
title: Provenance docs and skills disagree with the kernel on unscoped ai/ actors
status: open
---
## What's missing

The provenance record gives two rules for a non-human `--actor` that carries a
`--principal` and runs with no active authorization scope, and the kernel enforces
one of them. The kernel refuses the combination at both of its layers: at verb time
`Allow` (`internal/verb/allow.go`) returns `provenance-no-active-scope` when the actor
has no active scope at all, and at check time the `provenance-no-active-scope` rule
(`internal/check/provenance.go`) reports any such commit at error severity.

The record states that refusal — `docs/design/provenance-model.md` §"Composition with
entity FSMs" and its check-rules table, and R-FP-0130 in
`docs/design/legal-workflows-first-principles.md` — and also states the opposite:

- `docs/design/provenance-model.md` §"Worked examples", Example 2: `aiwf add gap
  --actor ai/claude --principal human/peter` with "No scope (the conversation is
  turn-by-turn HITL; the LLM isn't operating autonomously)".
- `internal/skills/embedded/aiwf-authorize/SKILL.md`, the "Tool mode (HITL)" bullet
  and the closing "Don't" list, which prescribe `--actor ai/<id> --principal
  human/<id>` for turn-by-turn and one-off agent edits. The same skill states the
  refusal in its opening paragraph and its findings table.
- `internal/skills/embedded/aiwf-add/SKILL.md` §"Provenance flags", which says to pass
  `--actor ai/<id> --principal human/<id>` in tool mode and, a few lines later, not to
  pass `--actor` at all.

Measured on Linux (go 1.25.11, git 2.54.0) with an `aiwf` binary built from
`716f3b4b5`, in a fresh repository after `aiwf init --skip-hook` with `hosts: []`, no
scope opened, and `b.md` holding a complete gap body (both required sections with
prose):

    $ aiwf add gap --title probe --body-file b.md --actor ai/claude --principal human/peter
    aiwf add: provenance refused: actor "ai/claude" has no active scope authorizing this act (provenance-no-active-scope)
    exit=1

Expected, per Example 2, a gap committed with `aiwf-actor: ai/claude` and
`aiwf-principal: human/peter`. Observed a refusal and no commit.

## Why it matters

No verb that runs the scope gate can produce the trailer set Example 2 promises, so
an assistant following either skill in a consumer repo is refused on its first
mutating verb. Opening a scope does not rescue the documented command: a free-standing
gap carries no reference that reaches a scope, so the same invocation under an active
scope is refused with `provenance-authorization-out-of-scope`.

The provenance design is the normative record of which principal × agent × scope
combinations are legal, and it names this combination both legal and illegal. A reader
cannot learn the rule from the record meant to hold it, and a fix to either side made
from one passage leaves the other passage standing.
