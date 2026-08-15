---
id: G-0587
title: A shipped skill cannot name the docs corpus a review must read
status: open
---
## What's missing

Two current rules collide, and a review pass that reads the project's
commitments cannot satisfy both.

`aiwfx-record-decision` bans a shipped ritual from pointing at a repository
path: a behavioral skill "does not embed a markdown link to a decision record or
design doc under `docs/` (or another non-shipping repo path)", because the
reader may not have that file. The reason is sound — skills materialize into
consumer repositories where this project's paths mean nothing.

A review that audits a specification against what the project has already
committed to must open exactly those files: the ADRs, the decisions, and the
normative documents. A skill instructing that review has to say where they are,
and the only vocabulary available is the paths it is forbidden to name.

The ban is a shipped-surface rule, so it applies to the skill that carries the
instruction rather than to the review itself. A ritual authored in this
repository and never materialized elsewhere would not hit it; one that ships
will.

## Why it matters

The obligation lands on any attempt to ship a corpus-reading review as a skill,
and it is not visible until the skill is written — at which point the natural
phrasing is already the forbidden one.

The resolution is a shape rather than a path: a skill can name what it needs
without citing where this project keeps it. "The project's accepted
architectural decisions" is portable; a directory under `docs/` is not. That
also makes the instruction correct in a consumer repository whose layout differs,
which the ban exists to protect.

Whether the enforcement is mechanical or held at review is unsettled, and it
changes the cost. The `skill-body-id` check scans shipped surfaces for
identifier-shaped tokens; whether a bare directory path trips it has not been
measured. `grep -rn "skill-body-id" internal/check/` and a fixture skill
containing such a path would settle it.

Filed separately from the review pass it blocks because the collision exists
today, independent of that work, and any future skill that must read the
project's own documents meets it.
