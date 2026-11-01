---
id: ADR-0056
title: Name an authorize scope by a random id, not its commit SHA
status: proposed
---
> **Date:** 2026-11-01 · **Decided by:** Peter Bruinsma

## Context

`aiwf authorize <id> --to <agent>` opens a scope, and the [provenance model](../design/provenance-model.md) addresses the scope by that commit's SHA (§"Scope id"). Two trailers carry the address: `aiwf-authorized-by` on each commit an agent makes in the scope, and `aiwf-scope-ends` on the commit that ends it, whether that is `aiwf authorize --end` or the promote or cancel that closes the scope's entity. [ADR-0047](ADR-0047-gate-every-terminal-epic-edge-end-a-scope-by-naming-it.md) names a scope the same way in `--end --scope`.

A rebase that re-creates the opening commit gives it a new SHA, and every pointer keeps the old one. G-0751 records what that does to the agent's commits, and G-0752 what it does to an end. Neither needs a force push, so the model's premise that "SHAs are stable in the absence of force-pushes" does not cover them.

**What a rebase keeps.** Measured 2026-11-01 with git 2.51.1 on Linux: an empty commit with an author date of 2026-10-01 and an `aiwf-` trailer, on a branch rebased onto a `main` that had moved.

| Operation | SHA | Author date | Trailer |
|---|---|---|---|
| `git rebase main` | new | kept | kept |
| `git rebase --reset-author-date main` | new | that day's | kept |

After the rebased branch was force-pushed, a `git clone --no-local` of the remote did not contain the replaced commit (`git cat-file -e` exited 1), though the bare remote still held it, unreachable. Of the opening commit, its message is what both rebases kept.

The research model of today's per-verb commits agrees: with `aiwf-authorized-by` written as the opening operation's identity rather than its SHA, "no host setting refuses the transaction, and under merge-commit and semi-linear hosts every invariant holds in `auth`" (the model's `RESULTS.md`, headline item 5, on the research branch named under References). The model never ends a scope, so for G-0752 the argument is that an end names the same id, which a rebase keeps.

**Alternatives considered:**

- *Keep the SHA, and find the replaced commit's successor when reading.* A clone over the network does not have the replaced commit, so there is nothing to compare its successor with.
- *Compute the id from the opening commit's metadata* (entity, agent, principal, author date). `--reset-author-date` changes the date and breaks the pointer as a rebase breaks a SHA today, and two authorizations of one entity to one agent within a second get the same id.
- *Give every verb's commit an identity.* The research proposal gives each one, because its fold of a transaction's commits changes their SHAs and its push rule then looks each operation up by identity. The fold is not ratified, and with one commit per verb nothing but the scope readers would read such an identity.
- *Write the scope id under a name for an operation's identity.* The behaviour is the same as this decision's, but the name would settle part of an open question in the research, the trailer format for operations (§10, question 2), before the evidence the research asks for. A scope is not its opening operation: the operation creates the scope, as `aiwf add epic` creates an epic, and the two have separate identities.
- *New keys in place of `aiwf-authorized-by` and `aiwf-scope-ends`.* An older aiwf refuses an agent commit written that way, as it refuses the same keys holding a new value (table below), so new keys protect no older clone and add two keys every reader has to know.
- *Write both: the SHA under the old keys, the scope id under new ones.* An older aiwf then reads an end correctly (table below). But every pointer would hold one fact twice, the SHA copy would still break on a rebase, and it would have to be removed later.

**What an older aiwf does with the new shape.** Measured 2026-11-01 with aiwf built from `c06ffdaf9` standing in for a release before this one, git 2.51.1, Linux, in a scratch repository with a real scope on an epic. The newer commits were written by hand with `git commit --no-verify`.

| Commit, as a newer aiwf writes it | What the older aiwf does |
|---|---|
| Opening commit with an added `aiwf-scope-id` | Ignores the trailer. The scope reads active, an agent's verb succeeds and writes the opening commit's SHA, and `aiwf check` reports no error. |
| Agent commit, `aiwf-authorized-by` holding a scope id | `provenance-authorized-by-malformed` at error severity, so its pre-push hook refuses the branch. |
| Agent commit, a new key in place of `aiwf-authorized-by` | `provenance-trailer-incoherent` at error severity. |
| End commit, `aiwf-scope-ends` holding a scope id | The scope reads active, an agent's verb succeeds (exit 0), and `aiwf check` reports no error. |
| End commit holding both the SHA and a scope id | The scope reads ended, and an agent's verb is refused (`provenance-no-active-scope`). |

**Scopes in use.** On 2026-11-01 every scope in this repository's history had ended, each by its epic's promote to `done`, and the maintainer knew of no other project that opens scopes.

**Copies of an opening commit.** Today aiwf reads each copy of an opening commit as a scope of its own, and rebasing the branch that carries it, then merging a branch built on its earlier version, is enough to make one (G-0755). A copy keeps the commit's message, so under this decision it carries the scope id too, and readers need a rule for two opening commits that carry one id.

## Decision

- **A scope has a random id.** `aiwf authorize <id> --to <agent>` generates it and writes it on the opening commit as `aiwf-scope-id: <scope-id>`. Being random, it needs no other clone to allocate it. Its shape always contains a character no SHA contains, so it is never 7–40 hex.
- **Only the opening commit carries it.** Pause and resume commits name no scope and are unchanged, and no other verb's commit gains an identity.
- **The two pointers keep their keys and hold the scope id.** Every `aiwf-authorized-by` and `aiwf-scope-ends` aiwf writes holds the id of the scope it names. A scope opened before this release has no id, and the pointers aiwf writes to it stay its opening commit's SHA.
- **Readers accept both shapes.** A value of 7–40 hex names an opening commit by its SHA, as every pointer already in history does; any other value names the scope whose opening commit carries that id. Pointers already written are never rewritten.
- **One id is one scope.** Every opening commit carrying an id opens the same scope. The first in history order is its opener, and a later copy changes nothing.
- **A scope is shown and named by its id.** `aiwf show` lists a scope by its id, or by its opening commit's SHA when it has none, and `aiwf authorize --end --scope` takes either, with the prefix rule it applies to SHAs.
- **One release ships the readers and the writer together.**

## Consequences

- **A rebase no longer breaks a scope opened from this release on,** nor does any cherry-pick or merge that keeps commit messages, and a copy of such a scope's opening commit no longer opens a second scope. That closes G-0751, G-0752 and G-0755 for those scopes. A squash that keeps only the title loses the trailers, as it loses every trailer today.
- **A scope opened before this release stays breakable until it ends.** The pointers to it remain SHAs, and a rebase that re-creates its opening commit still breaks them. A copy of its opening commit still opens a second scope.
- **An older aiwf misreads an end that this release writes.** It treats the ended scope as active and reports nothing (Context, the older-aiwf table). The agent commits this release writes, it refuses loudly. Every clone that reads a scope therefore needs this release or later, and the release's CHANGELOG entry says so. No scope was in use when this was decided (Context).
- **Commits an older aiwf makes in a new scope name the opening commit's SHA.** This release reads them, so they pass, and a rebase can still break them as it breaks today's.
- **Reading SHA pointers is permanent.** History holds them, so it is not a transition step to remove later.
- **What changes:**
  - the provenance model's sections that address a scope by its opening commit's SHA: §"Scope id", the shape rules for both pointers in the closed-set table, §"Scope states" and the `--end` flag;
  - the shape validation and every reader of the two keys, which a grep for `TrailerAuthorizedBy` and `TrailerScopeEnds` finds, and the finding hints that describe the value as the authorize commit's SHA;
  - `aiwf show`'s scope listing and `aiwf authorize --end --scope`.
- **ADR-0047's decisions stand.** It names a scope by its opening commit's SHA in `--scope` and in the end commit; this ADR changes that address and nothing else ADR-0047 decides.

## Validation

The decision holds while the G-0751 and G-0752 reproductions, re-run on a scope this release opens, report no `provenance-authorization-missing`, keep the ended scope ended and refuse the agent's verb after the rebase, and while G-0755's, re-run the same way, shows one scope and refuses the agent's verb after the pause. Revisit it if a git or host operation that keeps commits turns out to drop or rewrite their trailers, or when the research's question on the trailer format for operations is decided.

## References

- G-0751, G-0752 and G-0755, the defects this decision closes for the scopes it opens.
- [ADR-0047](ADR-0047-gate-every-terminal-epic-edge-end-a-scope-by-naming-it.md), whose `--end --scope` and end commit use the address this ADR changes.
- [The provenance model](../design/provenance-model.md), §"Scope id" and §"Scope states".
- The research on branch `patch/research-collaborative-state`, not merged into `main`:
  - `docs/research/collaborative-state-and-commit-granularity.md`: the fold (§4.2), this step (§9.3) and the open questions (§10, question 2);
  - `docs/research/transactions/model/RESULTS.md`, headline item 5: the model of today's per-verb commits with the pointer written as an identity;
  - `docs/research/id-allocation-under-concurrent-work.md`, design C: collision-free ids, which need no network.
