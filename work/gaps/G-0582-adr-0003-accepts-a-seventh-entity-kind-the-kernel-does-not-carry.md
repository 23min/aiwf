---
id: G-0582
title: ADR-0003 accepts a seventh entity kind the kernel does not carry
status: open
---
## What's missing

ADR-0003 is `accepted`. It adds `finding` as a seventh entity kind, with an id
pattern, a storage path, and a lifecycle.

The kernel carries six. `entity.AllKinds()` returns six, and CLAUDE.md lists "six
entity kinds" first among the load-bearing properties any change must preserve. So an
accepted architectural decision and the kernel's own stated commitments disagree
about how many kinds exist.

## Why it matters

Prose written about per-kind behaviour has to pick a side, and either choice
contradicts a normative source. That is not hypothetical: it surfaced while drafting
a decision about per-kind body sections, where "the six kinds" was the natural
phrasing and would have contradicted an accepted ADR — the phrasing was changed to
derive from the kernel's own kind set instead.

The wider cost is that an accepted ADR is the strongest signal this project has that
a design is settled. One that the code contradicts weakens every other ADR's claim to
be current truth, which is the property the normative tier exists to carry.

Three ways out: implement it, supersede it, or promote it to `rejected` if the
finding kind is no longer wanted. Which one is a decision rather than a cleanup — the
ADR's Context argues a real case for the kind, and nothing has been recorded against
it since.
