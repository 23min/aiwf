---
id: D-0058
title: The pocv3 reference rule's archive skip and allowlist are its shape
status: proposed
---
## Question

A policy test forbids references to the retired `docs/pocv3/` tree, so that a
path nothing resolves to cannot keep being cited as though it did. It carries two
escapes: any directory named `archive` is skipped during the walk, and a small
allowlist names individual files with a per-entry reason.

Escapes on a rule that exists to stop drift invite the obvious suspicion — that
they are where the drift went. The question is whether the two are a weakening of
the rule or its correct shape.

## Decision

**Both escapes are the design, not a defect, and stay.**

The `archive` skip is the archival tier's forget-by-default convention applied to
this rule: a frozen historical snapshot is not fixed when its references move, so
scanning it would report findings nobody intends to act on.

The allowlist is narrower and needs its per-entry reason to stay honest. Its
members name the literal path as their *subject* — a check whose own source
contains the string it searches for, a gap whose narrative is about that path
having gone dangling. Rewriting those would destroy the content.

## Reasoning

The distinction that carries this is between a reference and a mention. The rule
is about references: a citation that a reader would follow and find nothing at.
A file that discusses the path — as a string it searches for, or as the subject
of its own account — is not making a claim about where a reader should go.

No mechanical predicate separates the two. Both are the same characters in the
same tree, and the difference is what the surrounding sentence is doing. A rule
that tried would be guessing, and its errors would fall on exactly the files whose
content is most deliberate.

The alternative — no escapes, rewrite every occurrence — was rejected because it
inverts the cost. It would edit an archived snapshot to satisfy a rule about live
references, and it would flatten the gap narrative that records why the path went
dangling in the first place. That is a rule reshaping its subject matter to
remain enforceable.

An allowlist is not free, and its cost is the one named as cost-per-subject in
the decision on what earns a gap entity: every future legitimate mention has to
be added by hand, and an allowlist that grows stops being an exception list. That
cost is affordable at the current size and is what the trigger below watches.

## Consequences

**Revisit when a second file legitimately needs the literal path.** Not a second
allowlist entry — entries have been added for reasons that did not survive
scrutiny — but a second *legitimate* need, which is evidence the predicate is
mis-drawn rather than that one file is unusual.

**Each entry states its reason, and that is load-bearing.** An allowlist of bare
paths cannot be audited: nobody can tell later which entries were legitimate
mentions and which were somebody silencing a finding. The reason is what makes the
trigger above evaluable.

**The archive skip is not this rule's own convention.** It follows the archival
tier's forget-by-default treatment, so it moves if that convention moves, and it
should not be argued about here in isolation.
