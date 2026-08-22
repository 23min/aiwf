---
id: G-0608
title: Negative regression pins over shipped surfaces are a class D-0070 does not name
status: open
discovered_in: M-0313
---
## What's missing

D-0070 names two surviving classes over shipped surfaces: cross-document
relationship checks, and the trigger phrases that drive skill dispatch. A third
shape exists in the tree and fits neither — the negative regression pin, which
asserts a shipped surface does **not** contain something.

The instance is `TestNoReintroducedDeadVerbForms_ContractsAndSkill`, which
guards against a retired CLI form reappearing in documents that teach the CLI.
Its needles are command spellings rather than prose, and the property it pins is
that an obsolete surface stays obsolete — closer to a contract than to a
reading. But it is a phrase assertion by shape, so the ban fires on it.

Resolved for now by dropping the one shipped-surface site from its watch list
and keeping the two archive-doc sites, which are outside D-0070's scope. That
keeps the rule intact at the cost of a guard: the shipped `aiwf-contract` skill
is no longer checked for the dead form.

Note the asymmetry that makes this awkward to settle from the existing rules: a
positive phrase assertion can drift into vacuity as the prose is reworded, which
is the failure D-0070 measured, while a negative one cannot — it either finds the
forbidden string or it does not.

## Why it matters

The dropped guard is a real if small loss, recorded here rather than left
silent. More importantly the class needs a disposition: either D-0070 gains a
third exception and the ban a third exemption map, or negative pins over shipped
surfaces are ruled out deliberately and the coverage they gave is either
replaced by a derived check or accepted as gone.

A derived replacement looks reachable for this instance — asserting that every
`aiwf <verb> <positional>` shape a shipped skill demonstrates resolves against
the live command tree would catch a dead form without enumerating dead forms,
and would be a relationship check rather than a phrase assertion.
