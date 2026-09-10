---
id: G-0671
title: The changelog audit does not see a delta in a named kernel surface
status: open
discovered_in: M-0330
---
## What's missing

`internal/policies/changelog_completeness.go` reads one surface class. It scans
commits touching `internal/skills` and asks whether the entity behind each is
cited under `[Unreleased]`. That is the citation half of G-0529's direction, and
the class both recorded release omissions came from.

The other half is unbuilt: a delta in a surface the kernel already enumerates —
a finding code, a top-level Cobra verb, an `aiwf.yaml` key, an exit code — that
the release notes do not describe. Those are computable, but not by this
audit's machinery: the answer comes from comparing the enumerated set at the
base release against the set at HEAD, which is list comparison, where the
shipped-tree check reads commit trailers. No part of the existing audit reaches
them, so the surfaces most likely to change a consumer's scripted usage are the
ones nothing checks.

A second reading falls in the same hole. An epic that reached `done` having
shipped only compiled behaviour touches nothing under `internal/skills`, so it
is reported by nothing. Per-epic coverage today is a side effect of the
shipped-tree scan rather than a property in its own right.

## Why it matters

The two halves catch different failures and neither subsumes the other. G-0529
records the case this one misses: E-0075 wrapped with an entry that omitted a
user-visible refusal, noticed by a human afterwards and tracked as G-0509. The
epic *was* cited — the entry existed and was thin. A citation check passes on
exactly that shape, and the named-surface comparison is what would not have.

The consequence is the one G-0529 already names and this closes only half of: a
version is immutable by this project's own rule, so a surface that ships
undescribed in vX.Y.Z stays undescribed there. A consumer whose script breaks on
a changed exit code finds no entry, and the absence reads the same as the change
never having happened.
