---
id: G-0618
title: Provenance backstop covers 19 of 50 shipped files
status: open
priority: low
---
## What's missing

`PolicySkillEditProvenanceBackstop` requires a commit editing shipped ritual
content to carry an `aiwf-entity:` trailer resolving to a real entity. Its reach
is set in two places: `skillRitualsDir = "internal/skills/embedded-rituals"`
scopes the root, and the diff filter keeps only paths ending `/SKILL.md`
(`internal/policies/skill_edit_provenance_backstop.go`). Everything else is
skipped.

Counted against what `aiwf init` / `aiwf update` materialize into a consumer
repo, that is 19 of 50 files. Outside it: the 19 verb skills under
`internal/skills/embedded`, the 6 entity templates, the 4 role-agent cards, the
always-on guidance fragment, and the statusline script.

Measured on the commit that shipped the gap and contract templates: it added two
template files and edited a verb skill, the guidance fragment and a test, and
would have passed carrying no `aiwf-entity:` trailer at all. The trailer it
carries was written by hand.

## Why it matters

The policy's own header gives its reason as materialization — a file under the
watched root "is materialized into consumer repos by `aiwf init` / `aiwf update`,
so an edit to one reaches consumers directly." That sentence is equally true of
all 31 files outside the filter. The guidance fragment is the sharpest case: it
is not merely materialized but `@`-imported into the consumer's `CLAUDE.md` and
read every turn, so an edit there reaches more readers, more often, than an edit
to any single skill.

The second-order cost is the one that bites. `CLAUDE.md` presents this rule as
"mechanical, not vigilance", and a reviewer who believes a machine is checking
stops checking. For the majority of the surface nothing is, and the gap between
the described guarantee and the implemented one is invisible from either the
rule's prose or a green CI run.

Widening the filter is not free of consequence: the audit is diff-scoped and two
of its three invocations resolve the base to the merge-base with trunk, so it
re-judges a branch's whole life on every run. Any in-flight branch carrying an
untrailered edit to a newly-covered path turns red at once.
