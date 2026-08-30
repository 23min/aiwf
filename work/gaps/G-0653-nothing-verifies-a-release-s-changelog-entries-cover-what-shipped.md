---
id: G-0653
title: Nothing verifies a release's CHANGELOG entries cover what shipped
status: open
priority: medium
---
## What's missing

Two surfaces are meant to keep a release's CHANGELOG honest, and neither checks
the thing that matters. `.github/workflows/changelog-check.yml`
fires on a pushed `v*` tag and asserts a single grep — that `CHANGELOG.md`
contains a `## [X.Y.Z]` heading matching the tag. It says nothing about what
sits under that heading, so a release section holding one stale line passes
identically to a complete one. `CLAUDE.md` § *Go conventions § Release process*
carries the real rule — "even pure-mechanical patches need a one-line entry" —
with nothing enforcing it.

The workflow's own comment states the intent it does not reach: "Per the
kernel's 'must not depend on the LLM's behavior' doctrine, the check is what
makes changelog discipline real." It makes the *heading* real. Coverage of the
entries under it still depends entirely on whoever cut the release having
remembered.

Cutting v0.34.0 surfaced three shipped changes that reached the release with no
entry: G-0647 (a `feat(ritual)` adding the milestone-spec template's `## Closes`
section and the three rituals that read it), the always-on guidance's "a test
pins a rule, not an input" rule, and G-0628's new `aiwf-promote` skill section.
Every mechanical gate passed. A human asking whether skills and guidance had
been updated is what caught them.

Two of the three rode in on `docs(` commits. That prefix reads as "no
user-visible change" — true in most repos, false in this one: aiwf ships its
guidance and rituals as product, embedded under `internal/skills/`, so a
`docs(guidance)` or `docs(skills)` commit touching that tree changes what every
consumer receives on upgrade. The commit-message convention and the shipped
surface disagree, and nothing reconciles them.

## Why it matters

The release notes are the only artefact a consumer reads to learn what an
upgrade changes. Discipline that lives solely in `CLAUDE.md` prose fails exactly
when a release is large and the wraps that should have written the entries are
distant from the cut — the case where complete notes matter most. v0.34.0 spanned
235 commits.

The failure is silent and unrepairable. Versions are immutable by this project's
own rule, so a change shipped undocumented in vX.Y.Z stays undocumented there
permanently; a later release can describe it only as history. A consumer
diagnosing behaviour that changed under them finds no entry, and the absence is
indistinguishable from the change never having happened.

The near-miss is the shape of the risk: G-0647 alters a template consumers author
their milestone specs against, and it would have landed with the release notes
silent on it.
