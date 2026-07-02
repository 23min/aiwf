---
id: M-0202
title: Fix devcontainer onboarding banner (retired plugin install)
status: done
parent: E-0048
tdd: none
acs:
    - id: AC-1
      title: Devcontainer onboarding drops the retired rituals plugin-install flow
      status: met
    - id: AC-2
      title: Drift chokepoint forbids the retired flow and reconciles the legacy banner pin
      status: met
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

## Work log

### AC-1 — Devcontainer onboarding drops the retired plugin-install flow
Rewrote the `init.sh` banner and corrected the README ("Reopen in Container"
step 4, the verify block, and the retired "Cross-repo plugin testing" section →
"Ritual authoring"). Both files are clean of every retired marker. · commit
`7340ea43` (content) + `89170432` (recovery-prompt sweep) · verified by
`PolicyM0202DevcontainerOnboarding` green on the live tree.

### AC-2 — Drift chokepoint forbids the retired flow and reconciles the legacy pin
Added `PolicyM0202DevcontainerOnboarding` (forbids the retired strings in both
files, requires the `aiwf doctor` / `rituals:` pointer) with four firing
fixtures; removed the obsolete `bannerLiterals` block + orphaned `sort` import
from `PolicyM0132InitScript`. · commit `7340ea43` · new policy 100% covered,
`TestPolicy_M0132InitScript` green on the corrected `init.sh`.

## Decisions made during implementation

- **`tdd: none` kept** (the skeleton default). The deliverable is doc/policy
  content with no runtime code path to phase through; `advisory` would only emit
  `acs-tdd-audit` warning noise on the `met` promotes. Mechanical evidence is
  still provided per the "AC promotion requires mechanical evidence" rule.
- **The dead `copy-skill-fixture` Makefile target was retired** (operator
  decision, in-flight). The reviewer found this milestone's deletion of the
  README "Cross-repo plugin testing" section broke a comment reference the target
  cited, and the target was already provably dead (its testdata path was removed
  by G-0182, its sibling repo archived by ADR-0016/G-0193). Retiring it folds the
  finding in rather than filing a separate gap for dead code. · commit
  `89170432`.

No ADR or project-decision entity was warranted — no architectural choice or
durable trade-off surfaced.

## Validation

- `make check-fast` (vet + golangci-lint + full `go test`): green — lint `0
  issues`, all packages `ok`.
- `go test -race -parallel 8 ./internal/policies/`: `ok`.
- `PolicyM0202DevcontainerOnboarding`: 100.0% coverage; reddens on the pre-fix
  files (revert-and-test → 8 violations), passes on the corrected tree.
- Firing-fixture meta-gate: `m0202-devcontainer-onboarding` not grandfathered;
  its four fixtures light the construction line.
- `PolicyM0132InitScript`, `PolicyM0132DevcontainerReadme`: green (the four
  required README H2 sections survive).
- `.devcontainer/` retired-marker sweep: clean. No `copy-skill-fixture`
  references remain repo-wide. `make help` parses.

## Deferrals

None. The source gap **G-0279** is closed by this milestone but stays
`status: open` per the E-0048 convention — source gaps are swept to `addressed`
in a single batch at the epic wrap, not per milestone.

## Reviewer notes

Independent fresh-context reviewer returned **APPROVE**, verifying every
load-bearing claim by measurement (non-vacuity via revert-and-test; marker-set
correctness; the M-0132 reconciliation left no orphaned firing fixture and `sort`
is genuinely unused; firing-fixture meta-gate 100%; README structure/accuracy;
scope discipline — the Makefile was untouched at review time). Two non-blocking
findings were addressed inline in commit `89170432`: (T1) retire the provably-dead
`copy-skill-fixture` Makefile target whose comment referenced the README section
this milestone deleted; (T2) sweep the stale `install plugins` failure-mode in the
README recovery prompt → `materialize rituals`.

Informational (no action): the `retiredMarkers` are single-space substrings, so a
line-wrapped `PROJECT\n   scope` in prose would evade the `PROJECT scope` marker —
irrelevant in practice because the markers co-occur and every fragment was removed.
