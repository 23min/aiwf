---
id: G-0500
title: Duplicate id reachable via edit-body over a moved entity file
status: open
discovered_in: M-0284
---
## What's missing

A guard that tells "this entity has no record anywhere" apart from "this entity's
record is at another path". Both look identical from the path the verb was handed:
the file on disk is absent from HEAD.

The two need opposite answers. A never-committed entity is `aiwf edit-body --body-file`'s
sanctioned input — it is how such an entity gets committed at all, and bless mode
refuses that shape and redirects there. An entity whose committed copy sits at a
different path must be refused, because the write lands beside that copy.

Measured, with a gap moved by a plain `mv`:

    aiwf edit-body G-0001 --body-file <new>   ->  exit 0
    HEAD:  work/gaps/G-0001-probe-gap.md      ->  id: G-0001
           work/gaps/G-0001-moved-by-hand.md  ->  id: G-0001

`aiwf check` reports no error locally, because the loader reads the working tree and
sees one file, so the pre-push hook passes it. A fresh clone reports `ids-unique`.

## Why it matters

M-0284 closed this for every verb carrying the claim-side guard: the guard refuses a
target entity whose path HEAD does not record. `edit-body` cannot take that guard,
and it is the verb every other refusal message points the operator at — so the
recommended recovery route is the one that stays open.

The damage is a record carrying one id twice, invisible to the local check that
would otherwise block the push.

## Scope

The lookup is by id rather than by path: does HEAD hold this entity anywhere? A
filename-convention scan of the entity's own directory is the cheap candidate,
since kernel ids ride in the filename; reading blobs to match frontmatter `id:` is
the precise one. Which is right is the question this gap decides.

Out of scope: the claim-side guard's own scoping, settled in M-0284, and the
same-state comparison, which is about content rather than location.

## References

- `internal/verb/editbody.go` — both modes, and why neither takes the claim guard
- `internal/verb/claimguard.go` — the guard that closes this for other verbs
- M-0284 — where the class was closed for guard-carrying verbs