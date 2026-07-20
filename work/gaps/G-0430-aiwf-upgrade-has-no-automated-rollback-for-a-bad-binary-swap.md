---
id: G-0430
title: aiwf upgrade has no automated rollback for a bad binary swap
status: open
discovered_in: M-0270
---
## What's missing

`aiwf upgrade` delegates the entire fetch/verify/place sequence to a
single `go install <pkg>@<version>` call — aiwf itself never backs up
the previous binary before swapping. If the newly installed binary is
broken, there is no `aiwf upgrade --rollback`; the operator's only
recovery path is manually running `go install <pkg>@<older-tag>`.

## Why it matters

A "cut a release" / "aiwf upgrade" conversation might otherwise assume
automated rollback exists. This is a reasonable minimalist design (not
reinventing a binary installer), not necessarily a defect — but it's
currently an unstated property rather than a deliberate, tracked one.
Whether to build `aiwf upgrade --rollback` is an open question this
gap holds; if the answer is "no, minimalism is correct here," the gap
still gives that judgment a permanent, referenceable home instead of
leaving the absence undocumented.