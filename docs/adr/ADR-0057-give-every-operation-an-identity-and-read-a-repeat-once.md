---
id: ADR-0057
title: Give every operation an identity and read a repeat once
status: proposed
---
> **Date:** 2026-11-08 · **Decided by:** Peter Bruinsma

## Context

An aiwf operation has no identity of its own. Readers know it by the commit that records it, so two commits recording one operation read as two operations, and nothing reports it.

Today that happens when a verb commit is cherry-picked onto another line and the original line is merged later. In the ten weeks the [fewer-planning-commits research](../research/fewer-planning-commits.md) replayed, 16 operations in this repository are recorded twice (§7), 14 of them by a cherry-pick onto `main` whose original was merged as well ([its history report](../research/fewer-planning-commits/reports/02-history-measurements.md)). The amend mechanism that research recommends puts several operations in one commit, and the failures it leaves end the same way: a push that leaves no remote-tracking ref, a clone made from the working repository, or a branch created while a verb amends records an operation twice (§4.4).

**What aiwf does with an operation recorded twice.** Measured 2026-11-08 with aiwf built from `d50379c2e` (an untracked directory outside the build marked it `+dirty`) and git 2.51.1 on Linux, in a scratch repository with a bare remote. An `aiwf edit-body G-0001` on a side branch was cherry-picked onto `main` after `main` had moved, and the side branch was then merged.

| Reader | Result |
|---|---|
| `aiwf history G-0001` | Two `edit-body` events, one per commit |
| `aiwf check` | `ok — no findings`, exit 0 |
| `git push` through aiwf's pre-push hook | Accepted |

The copy breaks the record, not the tree: both commits make the same change. What reads it as a second operation is every reader that lists operations by commit.

[ADR-0056](ADR-0056-name-an-authorize-scope-by-a-random-id-not-its-commit-sha.md) considered giving every verb's commit an identity and rejected it, because "with one commit per verb nothing but the scope readers would read such an identity". Two readers would read it now, with today's per-verb commits as with amend: the readers that list operations, which can read a repeat once, and a report of the repeat. ADR-0056's scope id is a different identity: "A scope is not its opening operation: the operation creates the scope, as `aiwf add epic` creates an epic, and the two have separate identities."

**What an older aiwf does with an identity trailer.** The same build and scratch repository, with an `aiwf-op-id:` trailer added by `git commit --amend --no-verify` to an `aiwf promote` commit: `aiwf check` exited 0, no finding named the trailer or reported an unknown, incoherent or malformed one, and a hooked push was accepted.

**Alternatives considered:**

- *Refuse a push that adds a repeat, with a human acknowledgment for one that cannot be undone.* It guarantees each pushed operation is recorded once. But a repeat that readers read once harms nothing, each of this repository's 14 cherry-pick repeats would have refused a push until someone acknowledged it, and under amend the copy sits inside an amended commit that no verb can split, so acknowledging would be the only way past it. It also needs a new `aiwf acknowledge` subverb.
- *Find repeats by patch-id, with no identity.* The research found the 16 repeats this way, but an amended commit's patch is its first operation's plus the next one's, so it does not match the single-operation original that amend's failures leave behind. It also costs a patch-id per commit: about 3 s for 534 commits ([patchid-cost.txt](../research/fewer-planning-commits/reflog-rule/out/patchid-cost.txt)).
- *Compute the identity from the operation's content* (verb, entity, actor, author date). `git rebase --reset-author-date` changes the date, so copies of one operation would get different identities, and two identical operations within one second would get the same one; ADR-0056 rejected this for scope ids on the same ground.
- *One identity per commit, not per operation.* An amended commit holds several operations, and a commit-level identity cannot say which of them is repeated.
- *Write the scope id and the operation identity under one key.* A scope is not its opening operation (ADR-0056), and a reader resolving a scope pointer would have to tell which kind of id it holds.

## Decision

- **Every operation has an identity.** A verb generates a random id for each operation it records and writes it with that operation's trailers as `aiwf-op-id: <id>`. It has the scope id's shape (ADR-0056): random, so no other clone has to allocate it, and always containing a character no SHA contains. One generator makes both.
- **The `commit-msg` hook adds one where it is missing.** A message that carries `aiwf-verb` and no `aiwf-op-id`, such as a ritual's `git commit --trailer`, gets one. A message that already carries one keeps it, so `git commit --amend` keeps the identity, as a cherry-pick or a rebase keeps the whole message.
- **One identity is one operation.** Every record carrying an identity records the same operation, and the first in history order is the one that counts. Every reader that lists operations — `aiwf history`, `aiwf show`, `aiwf status` and the render pass — reads a repeated identity once.
- **`aiwf check` warns of a repeat the checked range adds.** A new warning, `operation-recorded-twice`, names the identity and its commits when the identity is recorded more than once in HEAD's history and at most once in the history of the range's base. The range is the provenance audit's: `--since <ref>`, otherwise the upstream, and with neither the warning is skipped with that audit. It never refuses a push.
- **Operations recorded before this release keep no identity.** Readers read them as they do today, and nothing rewrites history to add one.
- **ADR-0056 stands.** An `authorize` commit carries its scope id and its operation identity, and the scope pointers keep naming the scope.

## Consequences

- **A cherry-picked verb commit merged beside its original reads as one operation and draws a warning,** where today it reads as two and draws nothing.
- **The amend decision can rely on it.** An operation that amend's remaining failures record twice reads once and is reported; the amend ADR decides how its record audit treats a repeat.
- **Older releases keep working.** They ignore the trailer (Context), so a clone on an older aiwf reads and pushes commits carrying it. The commits it makes carry no identity, and their repeats go unreported.
- **A commit made with `--no-verify` gets no identity from the hook,** including a wrap merge committed that way, and reads as an operation recorded before this release.
- **A squash that keeps only the title drops the identity,** with every other trailer, as it does today.
- **The repeats already in history stay read twice.** They carry no identity.
- **The cost is one trailer line per operation.** The warning derives from the HEAD walk `aiwf check` makes once per run (`check.WalkHeadCommits`), so it adds no git call.
- **What changes:**
  - a trailer key, and the [provenance model](../design/provenance-model.md)'s trailer table;
  - the verb commit path, so every verb writes the key;
  - the `commit-msg` hook, which today only refuses a message and now also writes one key into it;
  - the history readers (`entityview.ReadHistory` and `entityview.EventFromCommit`, and what is built on them), `aiwf status`'s activity and the render pass;
  - a new `aiwf check` rule, with its entry in the `aiwf-check` skill's findings;
  - the CHANGELOG entry, which says older releases ignore the key.

## Validation

The decision holds while the Context measurement, re-run on this release, shows one `edit-body` event in `aiwf history`, an `operation-recorded-twice` warning naming both commits, and the push accepted; and while a ritual's `git commit --trailer aiwf-verb: …` comes out of the hook carrying an `aiwf-op-id`. Revisit it if a git or host operation that keeps commit messages turns out to drop trailers, or if repeats keep arriving through a path that a refusal would have stopped and readers reading them once does not make harmless.

## References

- [ADR-0056](ADR-0056-name-an-authorize-scope-by-a-random-id-not-its-commit-sha.md), the scope id this decision sits beside, and the alternative it rejected that this decision takes up.
- [The provenance model](../design/provenance-model.md), whose trailer table gains the key.
- The research this decision draws on:
  - [`docs/research/fewer-planning-commits.md`](../research/fewer-planning-commits.md): amend's remaining failures (§4.4), this foundation item (§5, item 3), and the 16 repeats (§7);
  - [`docs/research/collaborative-state-and-commit-granularity.md`](../research/collaborative-state-and-commit-granularity.md): every operation carrying an identity (§4.2) and the multi-operation record (§10, question 2).
