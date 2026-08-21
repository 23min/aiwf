---
id: G-0599
title: The shipped-surface rule's repo-context half has no chokepoint
status: open
---
## What's missing

The skills policy states that a shipped surface carries only imperative,
consumer-scoped instruction, and cites no real entity id, no filesystem path, and no
development history, provenance narrative, or rationale. One clause of that rule is
mechanized: the `skill-body-id` check walks every shipped Markdown surface and fires
at error severity on a real id or a non-canonical placeholder. The rest — the
filesystem path, and repo-development context generally — is held at review.

The unenforced half does not hold. G-0598 records roughly twenty citations across
eleven shipped skills pointing at named sections of this repo's `CLAUDE.md`: a
filesystem path and repo-development context both, accrued under a rule that forbids
them.

## Why it matters

The measured outcome is the argument. A rule written down and enforced at review over
a surface this repo edits constantly did not survive: the drift accumulated across
years and eleven skills without a reviewer catching it, and it stayed invisible
because these citations resolve correctly here and break only in a consumer tree.

Deleting the citations without a chokepoint leaves the generating pressure intact. The
phrasing is natural — a skill author reaching for the authority behind a rule writes
"per CLAUDE.md §X" because that is where the rule lives from where they sit. A cleanup
that does not change what the next author will reach for buys one pass and no more.

## Resolution shape

A shape rule alongside `skill-body-id`, rejecting a `CLAUDE.md` section citation in a
shipped surface while permitting the generic form — a reference to the consumer's own
project rules, naming no section, as the builder agent card already uses.

The scanner this reuses was read before this was filed. `skill_body_id.go` already
walks every `*.md` under the three embedded authoring roots whole-file, frontmatter
included, and is inert in a consumer repo because those directories are absent there.
A sibling rule is a pattern plus a finding-code constant declared in the closed set,
reusing that walk. One wrinkle to settle in the implementation: the existing scan masks
non-prose link carriers to preserve its doc-link carve-out, so a citation written as a
markdown link would pass unless this rule opts out of the mask.

A ban rather than a mandate, deliberately, per the guidance's own H3: it costs once, at
the moment someone writes the forbidden shape, where a mandate to cite correctly would
cost per skill forever. It is also consistent with D-0070, which retires tests
asserting that shipped prose *contains* a phrase; this asserts prose does *not* contain
a shape, which is what `skill-body-id` already does at the same severity over the same
files.

Sequencing: this fires pre-push at error severity over exactly the surfaces that
currently violate it, so landing it before the cleanup blocks every push. The cleanup
lands first, or both land together.

Worth settling in the same pass: whether the rule covers `CLAUDE.md` alone or any
repo-local document path in a shipped surface. The narrow form addresses what was
measured; the broad form also catches a citation into `docs/` that a consumer cannot
resolve, which the two mis-targeted provenance citations G-0598 records show is a
live shape.
