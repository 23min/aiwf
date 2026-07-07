---
id: G-0380
title: STATUS.md lacks a today's-work / since-release activity digest
status: addressed
addressed_by_commit:
    - 1b49940a
---
## What's missing

`aiwf status --format=md` (the source of STATUS.md) has "In flight",
"Roadmap", "Open decisions", "Open gaps", "Warnings", and "Recent
activity" (last 5 commits) sections, but no curated summary of what
happened *today* or *since the project's last release* — a reader has
to scan the raw "Recent activity" tail or `git log` by hand to answer
"what gaps opened or closed today?" or "how much has accumulated since
we last shipped?"

## Why it matters

Anyone opening STATUS.md has no quick answer to "what did we get done
today" or "is there enough unreleased work to justify a release" —
both are derivable from existing commit trailers (`aiwf-verb`,
`aiwf-entity`, `aiwf-to`) but nothing surfaces them today. "Recent
activity" is commit-count-bounded (last 5), not day-scoped, so a busy
day pushes same-day activity off the list entirely, and no section
exists per release-tag range at all.
