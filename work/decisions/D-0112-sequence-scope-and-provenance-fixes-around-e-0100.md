---
id: D-0112
title: Sequence scope and provenance fixes around E-0100
status: accepted
relates_to:
    - E-0100
    - M-0365
    - M-0366
    - ADR-0056
    - G-0460
    - G-0751
    - G-0752
    - G-0755
    - G-0776
    - G-0779
    - G-0786
    - G-0787
    - G-0788
    - G-0790
    - G-0791
    - G-0792
    - G-0793
---
> **Date:** 2026-11-28 · **Decided by:** human/peter

## Question

Epic E-0100 writes ADR-0056, ADR-0057 and ADR-0058 as verifier-kit spines before any of them
is built, and plays each spine against the `aiwf` on `main`. Its milestones run in their own
session on the epic branch. Fixes for the scope and provenance gaps its first milestone,
M-0364, found run in another session on `main`. When should those fixes land, so that the two
sessions do not change the same behaviour at the same time, and which open gaps wait for
E-0100's later milestones or for the ADR-0056 build?

The dependency runs both ways, which is what makes it non-obvious. Some fixes would decide what
an ADR or a later E-0100 milestone decides, while the ADR build epics that follow E-0100 need
these gaps fixed before their conform runs can pass.

## Decision

- Between an E-0100 milestone's audit of a behaviour and its conform run, make no fix to that
  behaviour on `main`.
  - Until M-0365 wraps, that covers: code that reads or writes scopes; `aiwf check`'s provenance
    and `isolation-escape` rules; how verb commits and their trailers are written; how
    `aiwf history` reads commits; how check treats cherry-picks and merges;
    `aiwf acknowledge illegal`; and the git hooks aiwf installs.
  - From M-0365's wrap until M-0366 wraps, it covers M-0366's ground: what a verb commits, push,
    a lost push reply followed by `pull --rebase`, a branch cut, a merge, what check reads after
    each, and squash.
- Fix G-0793 now, by refusing a reason that contains a line break. Tell the E-0100 session
  before it merges, and hold the merge until M-0365 wraps if that session asks to record
  today's behaviour first.
- Triage G-0779, G-0791 and G-0792 when M-0365 wraps, together with the gaps M-0365 files. Fix
  those whose fix lies off M-0366's ground then, and the rest when M-0366 wraps.
- Decide G-0790, G-0787 and G-0788 when the ADR-0056 build is planned, after E-0100 wraps: does
  ADR-0056 change so that a pause names its scope, and should the opening commit record more
  than the id?
- Fix G-0776, G-0460 and G-0786 in the ADR-0056 build epic. ADR-0056 itself closes G-0751,
  G-0752 and G-0755.
- Tell the E-0100 session about this decision once, while M-0365 is in its audit step. After
  that, take each E-0100 milestone's new gaps at its wrap.

## Reasoning

E-0100 needs gaps filed, not fixed. M-0365's and M-0366's AC-4 each have a divergence that an
open gap already describes cite that gap. What needs gaps fixed is the build epic that starts
from each spine, which E-0100's Goal makes done only when `spine conform` passes against the
binary. So no E-0100 milestone waits on a fix, and holding fixes back costs E-0100 nothing.

A fix landed mid-milestone does cost it. Each audit records today's behaviour against a named
commit, and the milestone's conform run later plays the spine against `main` as it then is. If
the behaviour changes in between, the conform run's divergences no longer match what the audit
recorded, and classifying them means first working out which binary each came from. That is
the reason for the rule, and why it follows each milestone's audit instead of covering all of
E-0100.

What each held gap touches, as read on 2026-11-28 in E-0100's files on the epic branch, where
they stay until E-0100 merges:

- G-0779 changes `isolation-escape`, which M-0365's audit covers under cherry-pick and amend
  (rows C26, C62 and H26 under `spec/operation/audit/`).
- G-0791 concerns the order in which check judges merged history. M-0365 models merges and
  what check reads, and ADR-0056 and ADR-0057 each read a repeat by its first copy in history
  order.
- G-0792 changes how the scope readers and `aiwf history` pick commits, and M-0365 models both
  readers. Squash is on M-0365's list of cases, and M-0364's audit leaves squash to M-0366 (its
  question 15).
- G-0790 and G-0787 come from a pause commit that names no scope, which ADR-0056's Decision
  keeps: "Only the opening commit carries it." G-0788 comes from an opening commit that records
  only the id. A fix to any of the three decides what ADR-0056 decides. M-0365 takes the
  identity's shape from ADR-0056, and M-0366's amended commit carries the scope ids M-0364's
  spine decides, so the question waits for E-0100 to finish, not for M-0365 alone.
- G-0793's fix refuses input. It changes no behaviour M-0365 recorded, since M-0365's runs use
  one-line reasons, and it takes no position on ADR-0058's trailer groups. Folding the lines
  instead would choose an encoding, which belongs to M-0366.

Alternatives:

- Freeze every fix in the area until E-0100 wraps. Rejected: G-0793's failure is silent, a
  pause or an end that does nothing, and it would stay live through M-0365 and M-0366, which
  have no end date.
- Hand the area to the ADR build epics and stop fixing it here. Rejected: none of those epics
  exists yet, so nothing would be fixed until each is planned.
- Keep fixing gaps as they arrive. Rejected: it changes behaviour while M-0365 audits it, the
  collision between the two sessions that this decision exists to prevent.

## Consequences

- G-0779, one of the four gaps first taken up from M-0364, waits until M-0365 wraps.
- Three hand-overs depend on the E-0100 session: its notice of this decision, its answer on
  holding G-0793's fix, and each milestone wrap, which also shows as a promote to `done` on the
  epic branch.
- G-0776's fix stays staged in `.claude/worktrees/patch/G-0776-scopes-follow-reallocate` until
  the ADR-0056 build. Its base is behind `main` in files it touches, so it needs a rebase then.
- Work outside these areas is unaffected.
- When a later E-0100 milestone audits behaviour this decision does not name, the same rule
  applies to it.
