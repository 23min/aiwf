---
name: aiwf-authorize
description: Use when a human wants to delegate autonomous work on an entity to an AI assistant or other non-human agent — opening, pausing, resuming, or ending an authorization scope.
---

# aiwf-authorize

The `aiwf authorize` verb opens, pauses, or resumes an **authorization scope** — the kernel's typed lifecycle for "this human authorizes this agent to act on this entity." Without an active scope, the kernel refuses non-human actors at verb time and `aiwf check` flags any commit they produced as `provenance-no-active-scope`.

## When to use

The user says something like *"go ahead and implement E-03"*, *"work on the cache milestone autonomously"*, or *"pause your work on E-09 while I review"*. Authorization is the verb that records that handoff so subsequent agent commits carry trailers proving the human authorized them.

When the user is just dictating individual changes turn-by-turn (*"add a gap that says X"*, *"promote M-001 to in_progress"*), no authorization is needed — the human is the principal, the assistant is a tool, and verbs run with `--actor human/<id>` as if the human typed them. See *Tool vs. agent* below.

## What to run

```bash
aiwf authorize <id> --to <agent>                  # open scope
aiwf authorize <id> --to <agent> --reason "..."   # open scope with optional rationale
aiwf authorize <id> --pause "<reason>"            # pause the most-recently-opened active scope
aiwf authorize <id> --resume "<reason>"           # resume the most-recently-paused scope
```

`<id>` is the entity the scope authorizes work on (the *scope-entity*). `<agent>` is the operator's full id, conventionally `ai/claude`, `bot/ci`, etc. — anything that doesn't start with `human/`.

The verb **requires a human actor**. `aiwf authorize` invoked by an agent refuses with a usage error; only humans can grant authority.

## Tool vs. agent

The provenance model (see [`docs/pocv3/design/provenance-model.md`](../../docs/pocv3/design/provenance-model.md)) distinguishes two modes for non-human operators:

- **Tool mode (HITL).** The human is in the conversation, dictating changes. The assistant is a tool; verbs run with `--actor ai/<id> --principal human/<id>` (or just `--actor human/<id>` if the assistant is invoking on the human's behalf). No scope, no `aiwf-authorized-by:` trailer. The trailer set is `aiwf-actor:` + `aiwf-principal:` only.

- **Agent mode (autonomous).** The human authorized the assistant to operate without per-verb approval. Open a scope with `aiwf authorize <id> --to ai/<id>`. Subsequent verbs run with `--actor ai/<id> --principal human/<id>` and the kernel matches the most-recently-opened active scope, decorating each commit with `aiwf-on-behalf-of: human/<id>` and `aiwf-authorized-by: <auth-sha>`.

The right mode is the one the user signalled. *"implement this end-to-end"* → agent mode. *"add this gap"* → tool mode.

## Lifecycle

A scope's state machine has three states: `active`, `paused`, `ended`.

- **`--to`** opens a new scope in state `active`. The commit's SHA is the scope id; subsequent verbs inside the scope reference it via `aiwf-authorized-by:`.
- **`--pause "<reason>"`** flips the most-recently-opened active scope on `<id>` to `paused`. While paused, agent verbs targeting the scope refuse with `provenance-no-active-scope` until the scope is resumed.
- **`--resume "<reason>"`** flips the most-recently-paused scope back to `active`.
- A `promote <id> <terminal-state>` on the scope-entity (e.g., `aiwf promote E-03 done`) **auto-ends** every active scope on that entity by writing one `aiwf-scope-ends: <auth-sha>` trailer per scope. Ended is terminal — un-cancelling a scope-entity does not resurrect a previously-ended scope. Issue a fresh `aiwf authorize` instead.

## Common refusals

- **`--to` against a terminal scope-entity**: refuses unless `--force --reason "..."` is set. Override is meaningful — it lets a human resurrect work on a cancelled entity by issuing a new authorization (the original ended scope stays ended).
- **`--pause` with no active scope on the entity**: nothing to pause; the verb refuses.
- **`--resume` with no paused scope on the entity**: nothing to resume.
- **Non-human actor**: `aiwf authorize` is human-only. Per the kernel's "force is sovereign" rule, only humans can grant authority. (G23 reserves a future `--allow-force` for delegated force; sub-agent delegation is G22.)

## Standing checks the LLM may surface

After the scope is open, ordinary `aiwf check` rules apply to subsequent agent commits. If the LLM hits one, look at the finding code to know what went wrong:

| Finding | Meaning |
|---|---|
| `provenance-no-active-scope` | the agent ran a verb without a matching active scope. Open one with `aiwf authorize`, or run the verb as the human directly. |
| `provenance-authorization-out-of-scope` | the verb's target entity has no reference path to the scope-entity. Either authorize the right entity or work on something the existing scope already reaches. |
| `provenance-authorization-ended` | the scope was already ended (terminal-promote / revoke). Open a fresh scope with a new `aiwf authorize`. |
| `provenance-authorization-missing` | the `aiwf-authorized-by:` SHA didn't resolve. Usually a typo or a force-pushed reference. |
| `provenance-trailer-incoherent` | the trailer set violated a required-together / mutually-exclusive rule. Often: a non-human actor without `--principal`, or a human actor with a stray `--principal`. |

## What aiwf does

1. Parses `--to` / `--pause` / `--resume` (mutually exclusive).
2. Validates the actor is human; refuses otherwise.
3. For `--to`: confirms the scope-entity is non-terminal (or `--force` is set with a reason).
4. For `--pause` / `--resume`: confirms the source state exists.
5. Writes a single empty-diff commit with the standard trailer block plus `aiwf-scope: opened|paused|resumed`.

## Don't

- Don't open a scope on every verb — scopes are for autonomous work spans, not per-call. For one-off agent-assisted edits, run the verb directly with `--actor ai/<id> --principal human/<id>`.
- Don't try to use `--force` to make agent commits authoritative without a scope. `--force` is a *human* override; agent verbs need a scope to stand up.
- Don't pause/resume scopes on a different entity than the one they're attached to — the verb picks by entity id.
