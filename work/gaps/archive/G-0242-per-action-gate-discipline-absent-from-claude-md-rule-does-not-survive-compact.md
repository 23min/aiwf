---
id: G-0242
title: Per-action gate discipline absent from CLAUDE.md; rule does not survive /compact
status: addressed
addressed_by_commit:
    - 8792600c451050f7f745709f823d457a7f9d9ddd
---
## What's missing

The rule that "each mutating action is its own approval gate" is
encoded in the advisory layer in two places only:

- The `wf-patch` skill body, as 🛑 commit / PR markers inside a
  numbered step list (steps 6, 7, 8).
- The "Executing actions with care" section of the agent's system
  prompt (not aiwf-owned, not under our control).

Neither surface is re-injected per turn. The system-prompt section
is general-purpose ("ask before risky actions"); the skill body
only loads when the skill is invoked. CLAUDE.md is the one project-
owned surface re-injected every turn — and the rule does not live
there.

The skill-body encoding has a second weakness: it expresses the
gates as *steps in a procedure* ("step 6: commit gate") rather
than as a *standing invariant* ("every mutating action is a
gate"). An agent executing the procedure can satisfy each numbered
step while still bundling, because bundling is not a violation of
any single step — it is a violation of the principle the steps
together imply.

## Why it matters

`/compact` discards the *granularity* at which past gates fired
while preserving *that* approvals were given. The post-compaction
summary reads "user approved the patch" — it cannot distinguish
five separate gates each crossed from five gates collapsed into
one. On the next turn, the agent reads the summary as cadence
("this is how the user prefers to be asked") rather than as
history ("this is what the user approved that one time").

Discovery case (this session, 2026-06-10): the G-0163 wf-patch
close-out bundled commit + push + merge + promote + archive into
a single `AskUserQuestion` labeled "Yes — commit + push
(Recommended)." Five distinct mutating actions, one gate. The
user had no opportunity to redirect mid-sequence; the merge to
main, the gap promotion, and the archive sweep all rode the
single approval. The agent inherited the bundling pattern from a
prior session's summary where the same shape was used; the
inheritance was invisible to both sides until the user named it.

The kernel principle "framework correctness must not depend on
LLM behavior" applies inversely here: there is no mechanical
chokepoint that prevents bundling. The agent's *behavior* is the
only enforcement, and behavior drifts across compaction unless
re-anchored every turn.

## Direction

Two layers in scope; one layer deferred.

- **Layer 1 — CLAUDE.md (this repo).** Add a standing bullet
  under "Working with the user" that names the rule explicitly:
  per-action gating, no bundling, no inferring cadence from
  post-compaction summaries. Re-injected per turn; this is the
  load-bearing surface.

- **Layer 2 — advisory skills (`wf-patch`, plus any other
  ritual that walks an LLM through a mutating sequence).**
  Add a preamble that names the standing invariant — not just
  the per-step 🛑 — so the rule is visible before the agent
  reads the procedure as a sequence of allowable steps. No new
  skill (per scope decision); strengthen what's there.

- **Layer 3 — consumer CLAUDE.md.** Deferred to a separate
  gap. aiwf today does not materialize content into the
  consumer's CLAUDE.md (it owns `.claude/skills/aiwf-*` and
  `.git/hooks/`, not the project root's CLAUDE.md). The clean
  shape — marker-managed fragment gated by ADR-0015-style
  per-invocation consent — is real design work, not a one-patch
  fix.

## Test surface

There is no mechanical chokepoint, and the gap does not propose
one. This is advisory by design, same as CLAUDE.md's
"context.Context as first arg of new IO function" and "No new
package-level mutable state" — the chokepoints column reads
"code review" and "advisory" because mechanizing them would be
either too noisy or too contextual.

The closure evidence is the CLAUDE.md edit landing the rule in a
re-injected location and the skill preamble strengthening the
standing-invariant framing. Verification is structural (the rule
appears under the named section, the skill preamble names the
invariant), not behavioral (the agent can still violate it; the
rule's job is to make the violation legible to both sides
faster).

## Source

Discovered in: G-0163 wf-patch close-out conversation, 2026-06-10.
The bundling pattern was inherited from the prior session's
summary of the G-0184 close-out; the discovery happened when the
user named the inheritance and asked how it survived `/compact`.

## Closing this gap

When the wf-patch lands, this gap promotes to `addressed` with
`--by-commit <sha>`. Layer 3 lands as a separate gap referenced
from this one's body once filed.
