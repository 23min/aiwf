---
id: G-0523
title: Always-on guidance delivery rides CLAUDE.md discovery and can fail unobserved
status: open
priority: high
---
## What's missing

aiwf's always-on guidance reaches an AI assistant through exactly one channel:
`aiwf init` / `aiwf update` materialize `.claude/aiwf-guidance.md` and maintain a
marker-wrapped `@.claude/aiwf-guidance.md` import in the consumer's root
`CLAUDE.md`. Whether that import is ever read depends on the harness resolving
that particular `CLAUDE.md` as the session's project instructions — a step aiwf
neither controls nor observes.

When it doesn't resolve, nothing surfaces. `aiwf doctor` reports the guidance
healthy, because what it verifies is **materialization**: the file is on disk and
the import marker sits in `CLAUDE.md`. **Delivery** — the guidance actually being
in the agent's context — is a different fact, and no surface reports it.

Observed: a session working in this repo carried a parent directory's
`CLAUDE.md` as its project instructions. This repo's own `CLAUDE.md`, and
therefore the guidance import it wires, never entered context. Every always-on
operating rule was inactive for that session, and `aiwf doctor` reported the
skills surface clean throughout.

## Why it matters

CLAUDE.md's §"Consumer-operating guidance vs repo-development guidance" rests
its entire placement rule on delivery:

> Because this repo dogfoods the same materialized guidance, an operating rule
> placed in the embedded source is followed here *and* shipped — one source, no
> fork.

The shipping half is mechanical. The "followed here" half is an assumption with
no observation behind it, which makes the guarantee unfalsifiable: the repo
cannot distinguish an agent that followed its guidance from one that never
received it. `PolicyM0211GuidanceOperatingAnchors` pins that operating rules stay
*present in the embedded source*; nothing pins that the source reaches a reader.

The blast radius is every always-on rule at once — gate-per-mutation,
AC-mechanical-evidence, the id-shape rules, the code-health priming. A session
that silently loses the fragment loses all of them together, and the failure
looks exactly like an agent choosing not to comply.

## Fix shape

Add a second, mechanical delivery channel that does not depend on `CLAUDE.md`
discovery.

aiwf already ships one. `internal/skills/hooks.go` wires a hook on the
`SessionStart` and `SubagentStart` events, and
`internal/skills/embedded-hooks/worktree-rituals-check.sh` is its script. That
hook fires only when cwd sits inside `.claude/worktrees/`, and it checks
materialization only — so it is structurally blind to precisely this failure,
despite existing to catch a neighbouring one.

Emitting the guidance content through that channel (or a sibling on the same
events) makes delivery independent of which `CLAUDE.md` the harness picked. The
events are already wired, so this needs no consent surface beyond what ADR-0015
already governs. The `CLAUDE.md` import can stay; the point is that it should
not be the only route.

Deliberately excluded: any scheme where the agent attests it read the guidance —
echoing a canary token, confirming in prose. CLAUDE.md §"Engineering principles"
rules that out: a guarantee that depends on LLM behavior is not a guarantee. Fix
the channel; don't ask the model to swear.

## Related

- **ADR-0018** — established the consent model for edits to user-owned files,
  under which the `CLAUDE.md` import is maintained. This gap does not dispute
  that model; it observes that the import alone is not a delivery guarantee.
- **G-0520** — the concrete cost of one such session: a shipped skill's prose
  contradicted the kernel, and three review passes ran without the repo's own
  conventions in context before the omission was noticed by hand.

## Provenance

Observed during the G-0520 patch. The same session carried a different tool's
guidance successfully, delivered via a `SessionStart` hook, while aiwf's import
route silently produced nothing — the two channels ran side by side and only the
hook arrived.
