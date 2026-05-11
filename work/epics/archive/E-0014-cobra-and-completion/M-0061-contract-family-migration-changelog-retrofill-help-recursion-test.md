---
id: M-0061
title: Contract family migration + changelog retrofill + help-recursion test
status: done
parent: E-0014
acs:
    - id: AC-1
      title: aiwf contract family migrated to native Cobra
      status: met
    - id: AC-2
      title: Contract subcommand flags wired for completion; opt-out removed
      status: met
    - id: AC-3
      title: Subprocess integration test passes for every contract subcommand
      status: met
    - id: AC-4
      title: CHANGELOG Unreleased retrofilled with E-0014 user-visible changes
      status: met
    - id: AC-5
      title: Help-recursion regression test pins the SetHelpFunc inheritance fix
      status: met
---

## Goal

Carry the E-0014 Cobra migration through the contract family of verbs (`aiwf contract bind|unbind|verify|reset|list|show`), wire their flag completions, pin a subprocess regression test for the family, retrofill the CHANGELOG's Unreleased section with the user-visible E-0014 surface changes, and add a regression test for the help-recursion bug surfaced when the SetHelpFunc-on-root pattern wasn't being inherited by Cobra's auto-generated `help` subcommand. Closes the migration's last family-shaped gap and proves the `aiwf <verb> --help` surface is a stable target across every command tree depth.

## Acceptance criteria

### AC-1 — aiwf contract family migrated to native Cobra

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-061/AC-1` for the actual implementation history._

### AC-2 — Contract subcommand flags wired for completion; opt-out removed

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-061/AC-2` for the actual implementation history._

### AC-3 — Subprocess integration test passes for every contract subcommand

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-061/AC-3` for the actual implementation history._

### AC-4 — CHANGELOG Unreleased retrofilled with E-0014 user-visible changes

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-061/AC-4` for the actual implementation history._

### AC-5 — Help-recursion regression test pins the SetHelpFunc inheritance fix

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-061/AC-5` for the actual implementation history._
