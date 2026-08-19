---
id: ADR-0045
title: Keep six entity kinds; findings live in entity bodies
status: accepted
---
> **Date:** 2026-08-19 · **Decided by:** human/peter

> **See also.** This ADR carries the reasoning for ADR-0003's reversal.
> [ADR-0003](ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md) is
> `rejected` and stays readable as the design it turned down.

## Context

ADR-0003 accepted `finding` (F-NNNN) as a seventh entity kind, carrying two
needs: cycle-time findings from parallel TDD subagents that block AC closure
until a human triages them, and check-time findings durable enough to escalate
out of a transient report.

None of it was built. `AllKinds()` returns six, there is no `work/findings/`
tree, and E-0019 — the consuming epic — has carried zero milestones since it was
filed. Two things decide against building it now: the corpus it would land in,
and a property of the kind model ADR-0003 did not weigh.

## Decision

aiwf carries six entity kinds. `finding` is not added.

**Volume.** ADR-0003 reasons from "the branch's existing 66 gaps (52 addressed)"
and projects that findings will be "at least as high-volume as gaps once
cycle-time emission turns on." That premise has grown ninefold.

    aiwf list --archived --format=json | python3 -c \
      "import sys,json,collections; r=json.load(sys.stdin)['result']; \
       print(len(r), collections.Counter(x['kind'] for x in r)['gap'])"
    # on main at 661d9f390, 2026-08-19 -> 1102 entities, 592 gaps

The projection therefore now means at least 592 further entities against a
corpus of 1,102 — half again as large. Every surface that reads the corpus pays
that permanently, and `aiwf check` already spends about 7 seconds over it at
this size.

**Lifetime.** This is the argument that holds at any corpus size. All six kinds
are permanent-with-terminal-status, and none has a lifetime bounded by another
entity's. A finding's usefulness ends when the work that produced it closes: it
is a handover artifact, not a record the project keeps. Modelling it as a kind
grants permanence it does not want, and pays the volume cost above to do so.

**What carries the need instead.** A finding is recorded in the body of the
entity whose lifetime it shares, triaged while that work is live, and at wrap
either repaired, extracted into a gap, decision or ADR, or declined explicitly —
the disposition `## Deferrals` already prescribes for a milestone's punted work.

**What would reopen this.** A need whose lifetime genuinely outlives its
subject and which a body section cannot carry: a finding that must survive its
milestone's archive under a stable id that other records cite. Neither of
ADR-0003's two motivating cases has that shape.

## Consequences

- E-0019 loses the storage model its finding-gated AC closure depended on. Its
  core — bounding a TDD cycle to a subagent invocation so the protocol is
  enforced by the runtime rather than by an LLM remembering rules — is
  untouched. The gating mechanism is open.
- The escalation-from-`aiwf check` case ADR-0003 raises stays unanswered. Such
  findings stay transient.
- The cross-AC, stable-reference and long-form-triage cases ADR-0003 argues
  against frontmatter arrays are real and are accepted as costs, not argued
  away. A body section carries long-form prose and cross-references; it does not
  carry a stable id, and a finding spanning two entities is recorded in one and
  referenced from the other.
- Kernel principle #1 stands unamended at six kinds — which is what the code,
  `CLAUDE.md`, `design-decisions.md`, `overview.md` and `architecture.md`
  already say. This ADR restores their agreement rather than changing anything.
