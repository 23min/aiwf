---
id: G-0606
title: Prose assertions written as a production policy escape the shipped-prose ban
status: open
discovered_in: M-0313
---
## What's missing

`shipped-prose-assertion` walks test functions only. The same assertion written
as a production policy — reading a shipped surface and comparing it against a
phrase held in the policy's own source — is outside the scan by construction.

The live instance is `m0211-guidance-operating-anchors`, which pins a curated
set of phrase anchors in the shipped guidance fragment. It is not an oversight:
CLAUDE.md documents it as the mechanical backstop keeping a consumer-operating
rule from drifting out of the shipped source, and its firing fixture is real.
So the question is not whether to delete it but whether the class is exempt.

Either the ban should reach production sources and that policy should carry an
exemption naming its ground, or the scan's test-function scope should be stated
as deliberate in the policy's own doc comment. Today it is neither: the scope is
an implementation detail that happens to spare one policy.

## Why it matters

D-0070's whole argument is that an exemption nothing records is worse than a
ledger, because a ledger is reviewable. "Write it as a policy rather than a
test" is exactly such an exemption — available, effective, and invisible.

The cost of leaving it is small today, since one instance exists and it is
documented. It grows the moment a second policy reaches for the shape, because
the precedent will read as sanctioned.
