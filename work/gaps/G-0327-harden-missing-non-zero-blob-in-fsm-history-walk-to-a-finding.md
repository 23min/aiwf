---
id: G-0327
title: Harden missing non-zero blob in FSM history walk to a finding
status: open
discovered_in: M-0216
---
## Context

E-0053 / M-0216 AC-2 replaced the `fsm-history-consistent` rule's per-read
`<commit>:<path>` resolution with a read by blob object id
(`BlobReader.ReadObject`), using the pre/post blob ids that `git log --raw`
emits. Both the new `statusBySHA` closure and the pre-AC-2 `readStatusAt`
return `("", nil)` — the "skip this pair" signal — when the blob read yields
`gitops.ErrBlobMissing`.

## The gap

For an **all-zero** blob id (the absent side of an add/delete) the skip is
correct: the file genuinely does not exist on that side. But `ErrBlobMissing`
can also fire for a **real, non-zero** blob object that is simply absent from
the local store — a corrupt object store, or a partial / blobless clone that
fetched the tree but not the blob. In that case the FSM comparison is silently
skipped and a real FSM-history finding can be hidden.

This is a **pre-existing** fail-open, not introduced by AC-2: the pre-refactor
`readStatusAt` skipped on `ErrBlobMissing` identically (the `<commit>:<path>`
spec resolves the blob OID from the present tree, then reports the absent blob
as missing), so AC-2 faithfully preserved the behaviour — and the
byte-identical claim required it to. Raised by the M-0216 third-pass review
(Finding 2).

## Proposed resolution

Distinguish the two cases at the read site: an all-zero id stays the "absent
side" skip; an `ErrBlobMissing` for a non-zero id becomes a
`fsm-history-consistent/history-walk-error` finding (the channel the walker
already uses for read failures via `historyWalkErrorFindings`), so a degraded
repo surfaces the unreadable history rather than silently passing.

This is a deliberate **behaviour change** on degraded trees (the old binary
skipped silently), so it lands outside any "byte-identical with the
pre-refactor binary" claim — on a healthy tree every non-zero `--raw` blob id
resolves, so there is no observable change there.

## Acceptance sketch

- A fixture repo whose `--raw` walk references a non-zero blob id that is then
  removed from `.git/objects` produces a `history-walk-error` finding rather
  than a silent skip.
- The all-zero add/delete skip is unchanged (no new finding).
