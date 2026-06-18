---
id: G-0254
title: Co-Authored-By commit-trailer convention is unstated and unenforced
status: open
---
## What's wrong

CLAUDE.md §"Commit conventions" documents the `aiwf-verb`/`aiwf-entity`/`aiwf-actor`
trailers and the Conventional Commits subject rule, but says **nothing** about the
git `Co-Authored-By` trailer. As a result its presence is inconsistent and depends
on per-session LLM behavior:

- 474 of 4355 commits (~11%) carry `Co-Authored-By`.
- It is mixed even among recent Claude-authored *code* commits: `df73aa00c`
  (the G-0067 coverage gate) carries `Co-Authored-By: Claude Opus 4.8 (1M context)`;
  `fec1db93` (the G-0247 fix) does not.
- The ~89% without it are mostly **structural**: every `aiwf` verb commit
  (`promote`, `add`, `edit-body`, `cancel`, `archive`, …) is written by the
  binary, which emits the `aiwf-*` trailers and never adds `Co-Authored-By`. So
  the real inconsistency is confined to manual `git commit` code commits, where
  the harness default ("add `Co-Authored-By`") meets a silent CLAUDE.md and the
  outcome rides on whatever the model does that session.

Surfaced during the G-0198 wf-patch (2026-06-18): the assistant asserted "precedent
is clear: no `Co-Authored-By`" after sampling a single commit, then reasoned *against*
the trailer from a wrong premise — exactly the drift this gap is about.

## Why it matters

This is a textbook instance of the kernel's own load-bearing anti-pattern:
*"the framework's correctness must not depend on the LLM's behavior."* A commit
convention that is unstated in CLAUDE.md and unenforced by any chokepoint will
drift commit-to-commit, which is precisely what the history shows.

There is also a genuine design question underneath, not just a doc omission:
**does the provenance model's "no co-actor inflation" (human = principal, LLM =
tool) extend to the git `Co-Authored-By` trailer?**

- If **yes**, a `Co-Authored-By: Claude` line *is* co-actor inflation — Claude
  code commits should not carry it, and CLAUDE.md should explicitly override the
  harness default.
- If **no**, git co-authorship is a courtesy separate from aiwf's accountability
  model — then add it *consistently*, and CLAUDE.md should say so.

Resolve as a `D-NNN` (it may touch
[`docs/pocv3/design/provenance-model.md`](../../docs/pocv3/design/provenance-model.md)),
document the outcome in CLAUDE.md §"Commit conventions", then back it with a
chokepoint so it stops depending on LLM recall.

## Enforcement analysis

The enforceability is asymmetric, and that asymmetry is itself an input to the
decision above:

- A mechanical git hook **cannot distinguish AI-authored from human-authored
  commits** — both run under the operator's git identity (`human/peter`).
- A **"must NOT carry `Co-Authored-By`"** rule is therefore trivially enforceable:
  reject any message with an AI co-author line, regardless of who made the commit.
- A **"AI commits MUST carry it"** rule is essentially **unenforceable**
  mechanically: the hook can't tell which commits are AI-made, and it would
  wrongly flag legitimate human commits and binary-authored verb commits (which
  never carry it). That decision would leave the convention LLM-recall-dependent —
  i.e. unfixed.

So only the "no co-author" decision can become a real chokepoint, and it aligns
with the provenance model. Noted as a lean, not pre-decided here.

### Candidate enforcement mechanisms

- **`commit-msg` git hook** — the correct git layer for message-content rules.
  (`pre-commit`, the first instinct, runs before the message is composed and
  receives no message — it inspects the staged tree, not the text, so it cannot
  see the trailer. `prepare-commit-msg` could *auto-inject* the trailer if the
  decision is "must-have".) A `commit-msg` hook fails fast at commit time, but is
  local-only, `--no-verify`-bypassable, and must be installed per clone — so it is
  a UX pre-check, not the guarantee.
- **`aiwf check` rule (pre-push + CI)** — the authoritative chokepoint, consistent
  with aiwf's existing commit-trailer machinery (`internal/check/provenance.go`,
  the trailer-keys policy, the untrailered-entity audit). CI's `aiwf check` always
  runs, catching `--no-verify` bypass and uninstalled-hook clones.
- **Recommended:** both, mirroring aiwf's existing shape-only-at-pre-commit /
  full-check-at-pre-push split — the `commit-msg` hook for fast local feedback,
  the `aiwf check` rule as the always-on guarantee.

## Fix shape

1. Resolve the design question as a `D-NNN` (does "no co-actor inflation" extend to
   git `Co-Authored-By`?).
2. State the decision in CLAUDE.md §"Commit conventions".
3. Back it with a chokepoint per the enforcement analysis (the `aiwf check` rule is
   the load-bearing one; the `commit-msg` hook is the fast local pre-check).
