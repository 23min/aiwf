---
id: G-0581
title: The backticked-verb resolution policy does not walk the guidance source
status: open
---
## What's missing

The skill-coverage policy resolves every backticked `aiwf <verb>` invocation in a
shipped skill body against the CLI's real command surface, so a renamed or retired
verb cannot leave a skill naming something that no longer exists. Its population is
the verb skills and the ritual skills.

The always-on guidance source is not in that population, and it backticks several
verbs.

## Why it matters

The guidance fragment materializes into every consumer repo and sits in an
assistant's context on every turn. That makes it the surface where a dead verb name
would be read most often and questioned least — and it is the one shipped surface
whose verb citations nothing resolves.

Nothing is broken today: every verb the guidance currently names resolves. This is a
missing backstop rather than a live defect, which is why it is filed rather than
fixed in passing. The cost of closing it looks small, since the policy already walks
several roots and this adds one more — but "looks small" is the argument that grows a
policy's population without anyone weighing it, so it gets the same scrutiny as any
other widening.
