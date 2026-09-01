---
id: G-0660
title: The compression lens's gate run cannot go red for the deletions it proposes
status: open
---
## What's missing

`aiwfx-wrap-milestone` step 2's compression question tells the reviewer to
"Apply it, run the gates, and report what actually broke"
(`internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md`,
the Compression bullet). That is a sound oracle for a rewrite and an unsound one
for a removal, and the lens proposes both.

The gates are the ones step 1 enumerates: the test suite, the build, the
project's lint gate, and `aiwf check`. No coverage gate and no mutation harness
is among them. The branch-coverage audit named in the ritual's constraints block
is not a substitute — `wf-tdd-cycle` states it is "agent-performed — a manual
branch-walk, not a tool invocation".

Against that set neither removal the lens can propose has a failing state:

- Deleting a test makes a suite greener, never redder. Nothing in the list fires
  on removed evidence, so "report what actually broke" returns nothing —
  correctly, and with no information in it.
- Removing a guard reds only where some test produces the state the guard
  catches. A green suite after that removal is the surviving-mutant condition:
  evidence the tests are blind, not evidence the guard is dead. The lens is
  licensed to read the null result as a pass.

The oracle that settles both already ships. `wf-vacuity`'s mutation probe is the
instrument, inverted per case: neuter the guard and a green suite is a surviving
mutant rather than a clearance; delete the test, mutate what it claims to pin,
and ask whether anything else goes red — which is what makes "redundant" a
measurement instead of a judgment. That method produced one of the three
findings G-0650 cites for the question's value: two tests carrying no
incremental coverage, demonstrated by running the unit at full statement
coverage without them. It stayed in that gap's evidence and never reached the
rule the gap shipped.

The exclusion already in the bullet does not cover the test case. It puts "tests
pinning distinct rules" out of scope, but whether a test pins a distinct rule is
the disputed judgment, and the same-outcome-clusters question directly above
directs the reviewer to collapse test groups by claimed outcome. The exclusion
excludes the tests the lens has already decided are not load-bearing.

Two smaller defects sit in the same pair of bullets.

**Nothing decides precedence between the compression and over-guarding
questions.** They state opposite defaults, and the text presents that as a
contrast rather than a rule for a cut falling in both scopes. Compression's
constraint list admits "a branch each arm genuinely needs", where *genuinely* is
exactly the judgment the over-guarding question holds must be demonstrated and
not made. A compression cut that touches a guard should inherit the stricter
standard.

**The report shape cannot distinguish a trial that ran from one that did not.**
Step 2's preamble requires naming the command behind the bucketed numstat "so
the next reviewer can re-run it"; the per-cut trial carries no equivalent, so a
reviewer that reasoned and one that ran answer in the same words. That half is
an instance of G-0585, whose site list is declared a floor and whose repair is a
verdict vocabulary with no clear-by-reading value in it.

## Why it matters

Reported downstream from one run of the lens: of what a single compression pass
proposed, half would have removed a working guard and a load-bearing test, each
carrying a plausible argument. Both were stopped by a human reading them, which
is the control the measurement requirement exists to replace.

This is what the rule produces when followed, not a lapse in following it.
G-0650 names "measured, not proposed" as one of four load-bearing properties and
the shipped text carries it; the measurement it names has no failing state for
these two cases.

The tree's other removal-proposing lens is disciplined the way this one is not.
`wf-structural-sweep` runs a whole-program reachability analysis with tests as
roots, then holds that "a reachability hit is a *candidate*, not a verdict",
lists "deleting on the tool's say-so" as an anti-pattern, and keeps discovery
and disposal separate steps. Two lenses in one tree take opposite stances on the
same question, and the one with no mechanical backstop is the permissive one.

Both surfaces ship, so the repair must be project-agnostic — the constraint
G-0650 states for the rules it added. `wf-vacuity` ships alongside the wrap
ritual, so routing to its probe names a ritual every consumer has rather than a
tool a consumer may not.
