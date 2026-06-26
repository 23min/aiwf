---
id: G-0284
title: skill-coverage policy has a namespace-subverb blind spot
status: open
---
## What's missing

`internal/policies/skill_coverage.go` asserts every top-level verb has a skill or an allowlist entry, and that backticked `aiwf <verb>` mentions in skill bodies resolve to a registered verb. But it iterates **top-level commands only**: `findTopLevelVerbs` (around `skill_coverage.go:386`) collects the commands registered in `NewRootCmd` and does not recurse into a command's own subcommands. Consequences:

- **Namespace subverbs get no coverage check.** `milestone depends-on`, and `contract bind` / `unbind` / `verify` / `recipes` / `recipe show|install|remove`, are never checked for skill coverage or an allowlist entry. A new subverb (e.g. a future `aiwf milestone block`) can ship with no skill and the policy stays green.
- **Body-mention resolution stops at the first token.** `checkSkillBodyMentionsResolve` / `aiwfWordRE` (around `skill_coverage.go:162` / `:541`) captures only the first word after `aiwf`, so a skill body writing `aiwf contract bogus-subverb` resolves `contract` and passes — the subverb token is never validated.

Today the `milestone` allowlist rationale hand-waves the concern in prose ("subverb depends-on ... covered by aiwf-add/aiwf-promote"); there is no mechanical enforcement behind it.

## Why it matters

This is the *second* reason the `aiwf milestone depends-on` capability was undiscoverable. Even when an assistant reaches the skill channel, no policy guarantees a subverb is documented somewhere, nor that a skill's reference to a subverb is valid. As more verbs migrate into kind-namespaces (the `milestone` / `contract` pattern is the established shape), the unguarded surface grows with each one — the guard silently covers less of the CLI over time.

## Proposed fix

Extend the policy to recurse into subcommands: every runnable subverb must have skill coverage — a documenting skill or an allowlist entry carrying a one-line rationale — and the body-mention resolver should validate the full `aiwf <verb> <subverb>` path, not just the first token. Keep the existing allowlist mechanism for deliberate "covered by the parent skill" cases, but make the entry a mechanical requirement rather than prose only.
