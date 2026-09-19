---
id: G-0504
title: doctor byte-checks only verb skills; ritual and guidance drift read as healthy
status: open
priority: high
---
## What's missing

The planning rituals refresh templates only when a file is missing. A present
but stale template can still be read by step 5 in
`internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md`
and its `aiwfx-plan-milestones` sibling. Both say to run `aiwf update` if the
respective template is missing; neither requires refresh before reading a
present file. This is a source-inspection finding, not a new live-session run.

E-0093 implements the detection portion: `doctor` compares the selected hosts'
verb skills, rituals, supported role cards, templates and guidance against
rendered expectations. M-0342/AC-4 records the checks and severity distinctions;
`TestDoctor_SelectedHostArtifactAbsenceAndDrift` and
`TestDoctor_ReportsBothHostsAndAllConcurrentDrift` cover drift reporting.
This gap remains open for the planning-ritual requirement.

## Why it matters

A planning session can use outdated template structure without running doctor
first. Detection alone does not ensure that the templates read by the rituals
match the installed binary. The resulting planning records can omit current
sections even while the operator believes the generated template is current.

## Resolution shape

Planning rituals need current templates when they read them, including when a
template already exists. Preserve user-owned files and the selected-host setup
contract. Choose the implementation in the follow-up workflow change.

## Where to fix

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md`

## Related

- E-0093 and M-0342 — selected-host drift detection.
- G-0471 — binary-versus-source staleness, a separate axis.
- G-0698 — planning workflow alignment with accepted D-0073.
