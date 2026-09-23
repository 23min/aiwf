---
id: D-0096
title: Refuse AI co-author trailers behind an opt-in address list
status: accepted
---
> **Date:** 2026-09-20 · **Decided by:** human/peter

## Question

A `Co-Authored-By:` trailer naming the assistant rode some commits and not
others, with nothing written down to say which. Most of the history without it
is structural — every `aiwf` verb commit is written by the binary, which emits
the `aiwf-*` trailers and never adds a co-author — so the drift was confined to
hand-written commits, where the harness default met a silent `CLAUDE.md` and the
outcome rode on whatever the model did that session.

Two questions sat underneath, and only the second was hard. Does the provenance
model's separation of principal from agent reach git's own trailer namespace?
And if it does, is that aiwf's rule to impose on the repos it ships into?

## Decision

- Commits in this repo do not record a non-human agent as a git co-author.
- The refusal ships as a `commit-msg` refusal ground, gated by
  `provenance.refuse_coauthors` in `aiwf.yaml`: a list of addresses, absent by
  default, so a consumer repo is unaffected until it names one.
- `internal/policies/coauthor_trailer_ban.go` judges the same property over a
  commit range in aiwf's own repository. It reads the same config and the same
  matcher, and it normalizes its input the way the hook's does — sharing the
  list alone does not make the two agree, because a trailer value git folds
  across lines is one message the two can read differently.
- The list holds addresses an agent here actually writes, not every agent that
  might exist.

## Reasoning

The kernel already records the agent that ran a verb in `aiwf-actor:`, and keeps
it separate from the principal, who is always human. A `Co-Authored-By` line
naming the assistant is a second, weaker copy of that fact — one nothing
re-derives, so it drifts — and it contradicts the model's reading of an LLM
directed in conversation, which is a tool rather than a co-author.

Shipping the refusal unconditionally was the obvious alternative and is wrong for
the surface it would ride. The grounds already in the `commit-msg` hook are
about aiwf's own trailer grammar, which is aiwf's to define. Whether a *consumer*
writes a co-author line is not: plenty of projects want one, and refusing a
`git commit` is a hard failure whose only escapes are `--no-verify` or disabling
the hook — which costs them every ground that is legitimately the kernel's.
So the rule ships, and naming an address is what arms it.

Keeping it out of the shipped hook entirely was the other alternative, enforced
here through a repo-local hook under `scripts/git-hooks/`. It was the lean until
a second repo wanted the same refusal: that would have put the same shell script
in two places, to drift a line at a time. One implementation behind a list serves
both.

The list is addresses rather than a boolean because nothing can compute the
predicate a boolean would name. `refuse_coauthors: true` means "refuse every
non-human co-author", and all either layer sees is one `Name <address>` pair —
there is no way to tell a bot from a person. So a boolean necessarily degrades
into a denylist compiled into the binary, which cannot be corrected without a
release. A denylist in config is the only form with no false-positive floor,
and a false positive here is a hard `git commit` failure.

Name-patterns lose outright: display names are free text that changes per
harness build, and a pattern on a vendor name matches a human who shares it.
The address is the stable key.

Both layers are kept because the hook runs only where it is installed, and only
as the `aiwf` it shells. A clone that has not run `aiwf init`, a `--no-verify`
commit, and a hook shelling a release older than the refusal all reach a landed
commit past it; the range check covers all three.

The range check is a Go policy rather than an `aiwf check` rule because of where
each one runs, not because of what it can express: `aiwf check` judges commit
ranges elsewhere — `RunProvenanceCheck` and `RunUntrailedAudit` both do — but it
reaches this repo's history only through the pre-push hook, which shells whatever
`aiwf` is installed. CI runs no repo-wide `aiwf check` at all. A policy test is
compiled from the tree under test, so it is the one layer a stale binary cannot
defeat — which is the third of the three routes above.

## Consequences

The refusal binds this repo the moment `provenance.refuse_coauthors` names an
address, which it now does — so a session acting on the harness attribution
default will have its commit refused, and `CLAUDE.md` §"Commit conventions"
states the override so it does not come as a surprise at the hook.

The ban is not retroactive. Commits already carrying the trailer keep it, and the
range check judges only the range it is given, so an ordinary push audits only
what that push adds.

G-0254's own fix shape asked for a cutoff-SHA severity tier of the kind
`trailer-verb-unknown` carries. That is not needed here: a diff-scoped policy
judges a range rather than all of history, so pre-existing commits are outside
the question instead of being excused by a tier.

A denylist fails open on the address nobody has met yet. When an agent's
attribution address changes, or a second agent arrives, the rule stops binding
and nothing signals it — the cost of choosing the form with no false positives.
Adding the address is the whole of the repair.

The range check is aiwf's own CI policy and does not ship, so a consumer repo
gets the hook alone: it binds only where `aiwf init` has wired it, and only as
the `aiwf` on PATH.
