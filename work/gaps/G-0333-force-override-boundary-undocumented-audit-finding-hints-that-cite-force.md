---
id: G-0333
title: Force override boundary undocumented; audit finding-hints that cite force
status: open
priority: medium
---
## What's missing

aiwf's `--force` carries a two-tier override semantics that is load-bearing but
discoverable only by reading verb source:

- **Tier 1 — verb-time local rules (`--force` overrides these).** The FSM
  transition legality (`entity.ValidateTransition` / `IsLegalACTransition` /
  `IsLegalTDDPhaseTransition`) and verb-time preconditions (resolver-required,
  sovereign-act human-only, addressed-by-commit existence). Each is guarded by
  `if !force { … }` (`internal/verb/promote.go:92`/`108`, `internal/verb/ac.go:206`/`237`).
- **Tier 2 — the introduced-error projection gate (`--force` never touches it).**
  Every mutating verb ends with `if check.HasErrors(projectionFindings(...))`, and
  that call is unconditional — `force` is not a parameter to it
  (`internal/verb/promote.go:171`, `internal/verb/ac.go:372`, and identically in
  add / editbody / move / rename / retitle / import / reallocate). A verb refuses
  to write a commit that introduces a new error-severity check finding, force or
  not.

The boundary — "`--force` overrides local FSM / preconditions, not tree-invariant
error findings" — is stated in no AI-discoverable channel (CLAUDE.md,
`docs/design/provenance-model.md`, or `--force --help`), yet it determines
what a sovereign override can and cannot do. This violates the kernel's "kernel
functionality must be AI-discoverable" principle.

## Why it matters

The `wf-tdd-cycle` skill (and G-0297's premise) wrongly claimed a `--force met`
hatch bypasses the `acs-tdd-audit`. It does not: under `tdd: required` the audit is
an error-severity Tier-2 finding, so `aiwf promote <M>/AC-<N> met --force` is
refused (verified: exit 1, state unchanged). Corrected in the skill under M-0199,
and the same misconception may live in the kernel's own finding-hints. That
hint audit is carried by E-0079, which corrects the surfaces claiming force
enforcement the kernel does not perform; the `milestone-done-incomplete-acs` hint
(`internal/check/hint.go:88`) is one of its subjects. What remains here is the
boundary itself: a reader who wants to know which rules `--force` overrides has
no document to consult, and infers it from whichever hint they happen to read.

## Direction

- Document the Tier-1 / Tier-2 `--force` boundary in CLAUDE.md (the provenance /
  force section) and `docs/design/provenance-model.md`; consider a one-line
  `--force --help` note per the AI-discoverability rule.

The finding-hint audit that this gap originally also carried moved to E-0079,
which is correcting the force-enforcement surfaces as a set; auditing the hints
separately would have split one sweep across two owners.

## Provenance

Surfaced during E-0048 / M-0199 (correcting the `wf-tdd-cycle` `--force met` claim)
and while answering a formal-verification query against aiwf v0.20.0.
