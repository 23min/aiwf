---
name: aiwfx-handoff
description: Emits a short paste-ready "compaction prime" block for `/compact` — a volatile-first handoff carrying the live reasoning thread, the gotchas found this session, and the exact next action, that points into the tree for everything git and aiwf already reconstruct. Fires on request anywhere ("give me a handoff", "prime the compact", "where are we for /compact", "hand off before I compact", "compaction prime"), and is referenced at the milestone and AC boundary by the start- and wrap-milestone rituals. Read-only over the planning tree — no mutation, no commit, no file written.
---

# aiwfx-handoff

Priming a `/compact` is otherwise a compose-from-scratch task done at an unpredictable moment — whenever tokens run high. This skill removes the compose step: it emits a short (≤10-line) paste-ready block the operator drops straight into `/compact <paste>`. The single paste is irreducible — the skill cannot call the harness command — but composing the block is not, and that is the friction this removes.

The block is **volatile-first**: it carries only what git and aiwf cannot reconstruct, and points into the tree for everything they can.

## Block format

Emit this shape — six lines, volatile payload first, pointers second. The ids are placeholders; fill them from the live milestone:

```
Continue E-NNNN / M-NNNN — <milestone one-liner>.
Just landed: <AC-N / what's done>.  Next: <AC-N — the exact next action>.
State lives in the tree: `aiwf show M-NNNN` · `aiwf status` · `aiwf history M-NNNN`.
Branch M-NNNN/<slug>: <clean | WIP: what's uncommitted>.
Watch out: <gotcha found this session — approach X rejected because Y; don't re-open G-NNNN>.
Decisions this session: <ADR-NNNN / D-NNNN, or "none">.
```

Lines 5–6 are the payload — the half `aiwf status` cannot reconstruct. Lines 1–4 are pointers, not re-summaries.

## Volatile vs durable — the one rule

Carry the **volatile** half; **point into** the tree for the durable half.

- **Volatile — belongs in the block.** The live reasoning thread, a gotcha found this session (an approach tried and rejected, a trap to avoid, a finding not to re-open), the exact next action, and the decisions taken this session. This is what evaporates at `/compact` and cannot be re-derived from committed state.
- **Durable — point, don't re-summarize.** Current epic and milestone, AC status, recent history, branch state, open findings. `aiwf show <M>`, `aiwf status`, and `aiwf history <M>` already reconstruct all of it from what is committed.

Re-summarizing the durable half creates a second source of truth that rots — the reason these prompts balloon. If a draft runs past ~10 lines, that is the tell it is duplicating committed state; replace the excess with an `aiwf show` pointer. The cap is a symptom of doing it right, not an arbitrary limit.

## Cadence

- **On request, anywhere.** The description's trigger phrases fire the skill mid-conversation — whenever the operator asks, or sees context growing before the next natural pause. This is the primary affordance; it is not a special mode.
- **Milestone boundary — always emit.** A milestone close is infrequent and a natural compact point, so the wrap-milestone ritual references this skill there unconditionally.
- **AC boundary — on request, or when context is visibly long.** Every-AC emission is noise, since compaction is not per-AC; the start-milestone ritual references this skill at the AC boundary for exactly the on-demand case.

## Anti-patterns

- *Writing a `HANDOFF.md`.* A handoff is ephemeral, consumed immediately into the `/compact` argument. A file just adds a stale surface for no gain — emit inline only, never persist.
- *Re-summarizing `aiwf status`.* If the block restates committed state, it has become the rotting second source the one rule forbids. Point with `aiwf show` instead.
- *Composing the `/compact` argument for the operator.* `/compact` stays human-driven; the paste is theirs to make. The skill drafts the block, it does not drive the harness.
- *Running past ~10 lines.* The length cap is the signal you crossed from volatile payload into durable duplication — cut back to a pointer.
