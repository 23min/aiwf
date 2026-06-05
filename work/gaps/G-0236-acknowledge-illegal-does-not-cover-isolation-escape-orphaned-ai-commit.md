---
id: G-0236
title: acknowledge-illegal does not cover isolation-escape-orphaned-ai-commit
status: open
---
## What's missing

`aiwf acknowledge-illegal <sha>` requires the target SHA to be reachable from HEAD via `git merge-base --is-ancestor`. The check is load-bearing for three of the five rules that consume the resulting `aiwf-force-for` trailer set — `fsm-history-consistent` variants and `isolation-escape` proper all operate on HEAD-reachable history, so an ack on an unreachable SHA could never match the rule's per-SHA exemption check anyway.

But the **`isolation-escape-orphaned-ai-commit`** rule (M-0161/AC-5 / G-0205) operates on **reflog-discovered orphans** — commits surfaced precisely because they were force-pushed away and are NOT reachable from HEAD. By construction. The rule consumes `ackedSHAs` correctly (`internal/check/reflog_walk.go:275: if ackedSHAs[o.SHA]`), but the verb refuses to record the ack:

```
$ aiwf acknowledge-illegal af1051d1 --reason "..."
aiwf acknowledge-illegal: SHA "af1051d1" is not reachable from HEAD (git merge-base exit 1)
```

The asymmetry: a sovereign-human review verdict that an orphan is benign-rebase-cleanup (not an escape) has no mechanical way to silence the finding. The warning rides along every `aiwf check` forever.

## Why it matters

Parallels G-0214 (which closed the same asymmetry for `forced-untrailered`). Same shape: the rule consumes the ack map; the verb refuses to mint the ack commit. The operator has no path to act on the rule's "ask the operator to review" semantics.

Concrete instance: orphan `af1051d1` on `epic/E-0033` from M-0120/AC-2's work cycle (May 18). The orphan's content re-landed cleanly via `d098614f` on main — same commit subject, the test (`TestADR0011_AC2_SevenDecisionSections` etc.) is in `internal/policies/adr_0011_test.go` today. Force-push was rebase cleanup; the AI's work is in the tree. Verdict: benign. Mechanism to record the verdict: none.

The rule's finding text says "isolation-escape-orphaned-ai-commit" implying operator review; without an ack path, the review verdict can't be recorded mechanically — it lives only in the operator's head until the next `aiwf check` flags the same orphan again.

## Proposed fix shape

Two surfaces to consider:

1. **Loosen the reachability check.** The verb's `git merge-base --is-ancestor` gate predates the orphan rule (which arrived in M-0161/AC-5). The fix: when the target SHA fails the reachability check, ALSO check the reflog for the SHA before refusing. If the reflog contains it (as a force-pushed-away tip), the verb proceeds; the ack commit lands on HEAD (reachable from HEAD by definition) carrying `aiwf-force-for: <orphan-sha>`, and the orphan rule's per-SHA exemption fires.

2. **Separate orphan-ack verb.** A dedicated `aiwf acknowledge-orphan <sha>` that bypasses reachability by design. Cleaner separation of concerns but adds a verb to the surface — the M-0159/AC-4 lift consolidated three subcodes under a single verb specifically to avoid this proliferation.

Option 1 is the minimal-surface fix. The reachability check exists to catch "operator typed a wrong SHA" (the message would silently never fire); adding a reflog fallback preserves that guard for typo-on-active-history while opening the channel for legitimate orphan acks.

## Test surface

When the fix lands:

- Positive test: acknowledge an orphan SHA (set up a fixture repo with a force-pushed tip, then ack it) → next `aiwf check` is silent on that orphan.
- Negative test: acknowledge a SHA that is NEITHER reachable from HEAD NOR in the reflog → verb refuses with a clear error (no silent-typo regression).
- The existing reachable-SHA tests under `internal/cli/acknowledgeillegal/` should keep passing unchanged.

## Workaround

Until the fix lands: the orphan-finding rides along as a perpetual warning per `aiwf check` run. The acknowledgment lives only in the operator's head and (now) in this gap's body. Re-surfacing the orphan-rule warning on every check is the cost.

## Discovered in

G-0218 post-archive cleanup pass on 2026-06-05. Acknowledged the parallel finding `055db369` (promote-on-wrong-branch — also consumes ackedSHAs, no reachability problem because the target was on main) successfully via the same verb; the orphan ack attempt failed with the reachability error above. Same shape G-0214 documented for `forced-untrailered`.
