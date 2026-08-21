---
id: G-0597
title: Options offered to a human in a decision are not held to the measurement rule
status: open
---
## What's missing

The shipped guidance fragment carries two rules that between them should govern a
decision menu, and they never meet.

*Decide one thing at a time* specifies the **shape** of what is presented: context,
options with pros/cons and risks, a plain lean with its argument, a numbered
pick-list, written in the plainest language that carries the trade. It says nothing
about whether an option has been checked to be a path the human can actually take.

*Nothing is settled except by measurement* carries the verification burden, but binds
**claims written into prose**: it asks for the command, the expected result, the
observed output, and the environment it ran in, and it says to leave an unsettled
claim visibly unsettled by naming the command that would settle it. Nothing states
that an option offered to a human is such a claim, so the burden does not reach the
menu.

An option is a claim — *"this is a real path you can choose."* Between those two rules
an assistant can assemble a menu from a summary, from an entity's own unverified body
prose, or from plausible reasoning, and present unchecked options in exactly the shape
the first rule prescribes. The `aiwfx-whiteboard` ritual, which produces a
first-decision fork from tree state, recognizes one narrow instance and no more: its
anti-pattern 2 requires every verb invocation in its output to resolve to a real
kernel command. The same requirement over the options themselves is absent there too.

## Why it matters

An unchecked option is not merely low-confidence. It is frequently **dead on arrival**,
resting on a premise that is false against the current code or tree state. The human
spends real decision effort ranking a viable option against a fictional one with no
way to tell them apart, and can select the fictional one. Discovery then arrives at
execution time — after the choice, at the point where the cost is highest and the
human has already reasoned onward from a false premise.

The most reliable source of a dead option is an entity's own body prose: a gap
claiming it is subsumed by another, a milestone naming a dependency. That prose is
exactly the stale claim a truth-sweep is convened to check, so sourcing an option from
it launders an unverified claim into a selectable choice.

Observed in a consumer repo during an entity-truth sweep: a three-option disposition
menu offered "fold gap X into gap Y — Y subsumes it", taken from gap X's own body.
The human chose it. Verification then showed the two sat at different pipeline stages
and Y could not structurally subsume X. The recommended option's wording was also
false until checked.

## Resolution shape

One clause on the existing decide-one-thing-at-a-time bullet, following G-0522, which
added the plain-language register to the same bullet on the same argument: the rule
says how to structure a decision and is silent on a property of the options
themselves. Per the guidance's own H3, a clause joining two rules that already ship
costs once, where a new mandate costs per decision.

The clause states that an option is a claim under the measurement rule, and carries
the consequences that do not follow from the measurement rule alone:

- **Only verified options are options.** The numbered list holds paths whose
  load-bearing premise has been checked against the tree or the code. The lean falls
  only among them.
- **An unchecked candidate is a pending check, not a choice.** It is named outside the
  list, with the command that would settle it and a hint of what running that command
  costs, so the human can decide whether the check is worth paying for before it is
  paid. This is the measurement rule's own mechanism — name the command you would have
  run and leave the answer blank — pointed at a menu, so it adds no machinery.
- **A candidate ruled out appears as ruled out, with its disqualifying evidence**,
  rather than being dropped silently. The human otherwise cannot distinguish a path
  considered and killed from one never examined.
- **An option is never sourced from an entity's own body prose or from a summary**
  without checking it against the tree or the code.

Dropping an expensive-to-check candidate silently was considered and rejected: it
buys the same guarantee by reintroducing the information loss the ruled-out clause
exists to prevent. Flagging unverified options inside the numbered list was also
rejected — a flagged option is still selectable, so the guarantee degrades from
"trust the options" to "trust the labels", which returns the verification triage to
the human mid-decision.

Constraints on the landing:

**Where it goes.** `internal/skills/embedded-guidance/aiwf-guidance.md`, not a
consumer's `.claude/aiwf-guidance.md` — the latter is gitignored and materialized from
the former by `aiwf init` / `aiwf update`, so a hand-edit there is overwritten on the
next update and ships nowhere. One edit to the source serves both audiences, since
this repo dogfoods the materialized fragment.

**How it reads.** Imperative and consumer-scoped, citing no real id, no path, and none
of the history above. The failure that motivates the rule belongs here, not on a
surface that materializes into repos where it is context-free noise.

**What evidence closes it.** D-0070 retires prose-presence assertions over shipped
surfaces, naming the always-on guidance fragment in its scope, and M-0313 deletes the
standing corpus — which includes both assertions currently pinning this bullet, the
curated anchor entry and the content test. So the change lands as reviewed prose with
no mechanical pin, by decision rather than by omission, holding the same standing the
fragment's own preamble claims for every rule it carries, and the same standing
G-0522 recorded for the register clause it sits beside. A closing criterion demanding
a test that greps for the new wording would re-introduce what D-0070 retires.
