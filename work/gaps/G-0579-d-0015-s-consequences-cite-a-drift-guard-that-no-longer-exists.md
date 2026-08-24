---
id: G-0579
title: D-0015's consequences cite a drift guard that no longer exists
status: open
---
## What's missing

D-0015 is `accepted`, and its Consequences state that the embedded skill bodies are
not rewritten to point at the materialized template path because they are "a
drift-checked verbatim snapshot of upstream (M-0148's
`TestRituals_VendoredMatchesUpstream`), so editing them would fail the drift guard."

The named guard was not lost to drift. ADR-0016 retired it deliberately and by name —
its own Consequences read "`TestRituals_VendoredMatchesUpstream` and the
upstream-vs-vendored drift it polices both retire" — and the retirement is complete
in the tree: no `rituals.lock`, no `make sync-rituals` target, no such symbol
anywhere in the repo. D-0015's Consequences are therefore not prose that decayed;
they are one accepted decision's Consequences superseded by another accepted
decision, with nothing propagating the change.

Nothing is left unguarded. The guard policed embedded-against-upstream drift, and
ADR-0016 removed upstream, so there is no second copy left to drift from. G-0345 has
already rewritten those bodies to cite the materialized path, so D-0015 forbids what
the tree has done.

Two further surfaces carry the same retired reason and belong to the same repair:

- `internal/skills/skills.go:154` justifies leaving an upstream-authored skill's name
  alone because it "must stay byte-verbatim per the vendored-snapshot drift guard."
  Same lapsed reason, and this one ships in the binary.
- `docs/adr/ADR-0016-…:65` closes with "**Status: proposed.** Ratification waits on
  the implementing gap producing a credible punch list and the operator confirming
  the GH-repo-archive step is acceptable," while its frontmatter reads
  `status: accepted`. It is the only ADR in the tree carrying an in-body status line,
  so this is a one-off rather than a class. A reader who reaches ADR-0016 from here
  meets a record that contradicts itself about whether it is in force.

## Why it matters

An accepted decision reads as current truth. A reader deciding whether a shipped
skill body may cite the templates directory is told it may not, and pointed at a
guard to verify that against which does not exist.

The Decision section itself is still correct — the four templates do materialize
where it says. Only the Consequences have gone stale. Whether they are corrected in
place or the decision is superseded is the open question, and it is a real one: an
accepted decision's Consequences are part of the record, not a scratch field.

Whichever repair is chosen, ADR-0016's self-contradiction must be settled in the
same pass. Resolving this gap requires reading ADR-0016 to establish that the guard
was retired on purpose, and a reader who does that reaches a record that declares
itself proposed.
