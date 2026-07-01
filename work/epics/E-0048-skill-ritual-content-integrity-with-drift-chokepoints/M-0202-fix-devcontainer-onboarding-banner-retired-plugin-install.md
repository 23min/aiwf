---
id: M-0202
title: Fix devcontainer onboarding banner (retired plugin install)
status: draft
parent: E-0048
tdd: none
acs:
    - id: AC-1
      title: Devcontainer onboarding drops the retired rituals plugin-install flow
      status: open
    - id: AC-2
      title: Drift chokepoint forbids the retired flow and reconciles the legacy banner pin
      status: open
---
## Goal

The devcontainer onboarding surface — the `.devcontainer/init.sh` post-install
banner and `.devcontainer/README.md` — reflects the current
rituals-materialization contract instead of the retired marketplace/sibling-repo
plugin-install flow. Rituals materialize into `.claude/` when `aiwf init` runs
(ADR-0014, E-0038); the upstream `23min/ai-workflow-rituals` marketplace channel
is archived (ADR-0016, G-0193). The banner and README no longer instruct the
obsolete "install both plugins at PROJECT scope" step or reference the retired
`recommended-plugin-not-installed` doctor warning; they confirm `aiwf init`
already ran and point at `aiwf doctor`'s `rituals:` line for verification. A
mechanical chokepoint keeps the retired instructions from reappearing, and the
one legacy policy that *required* the stale banner literals is reconciled so the
fix and the pin agree. Closes G-0279.

## Acceptance criteria

### AC-1 — Devcontainer onboarding drops the retired rituals plugin-install flow

The `.devcontainer/init.sh` post-install banner and `.devcontainer/README.md`
carry no instruction for the retired flow: no `/plugin marketplace add`, no
`23min/ai-workflow-rituals`, no `/reload-plugins`, no "PROJECT scope" manual
install framing, and no reference to the retired `recommended-plugin-not-installed`
warning. The banner instead states that `aiwf init` (run earlier in the same
script) already materialized the rituals and directs the operator to `aiwf
doctor`'s `rituals:` line to verify. The README's "Reopen in Container" step and
its verification block are corrected to match, and the stale "Cross-repo plugin
testing (rituals repo)" section — which described the archived sibling-repo flow
and cited a CLAUDE.md section that no longer exists — is replaced with an
accurate in-repo authoring note pointing at CLAUDE.md §"Ritual content
authoring".

Mechanical evidence: `PolicyM0202DevcontainerOnboarding` (new, under
`internal/policies/`) passes green on the live tree — it asserts the retired
strings are absent from both files and the banner's `aiwf doctor` / `rituals:`
pointer is present.

### AC-2 — Drift chokepoint forbids the retired flow and reconciles the legacy banner pin

`PolicyM0202DevcontainerOnboarding` is a durable drift chokepoint: it fires if
any retired-instruction string reappears in `.devcontainer/init.sh` or
`.devcontainer/README.md`, or if the banner drops its `aiwf doctor` / `rituals:`
verification pointer. Firing fixtures in `firing_fixtures_multi_site_test.go`
feed synthetic stale trees and assert the policy returns a violation, so the
chokepoint is non-vacuous (satisfies the firing-fixture meta-gate for a policy
added after G-0259). In the same change, `PolicyM0132InitScript`'s obsolete
`bannerLiterals` assertion — the only policy that *required* the retired literals
(`23min/ai-workflow-rituals`, "PROJECT scope", `aiwf-extensions`, `wf-rituals`)
as M-0132's AC-4 banner pin — is removed, so the new chokepoint and the legacy
pin no longer contradict and `PolicyM0132InitScript` passes on the corrected
`init.sh`.
