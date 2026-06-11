---
id: G-0244
title: aiwfx-release lacks CI-green precondition; vuln + lint red on main
status: addressed
addressed_by_commit:
    - 4108a87ccacb9b3449840f960b240dae02542bba
---
## What's missing

The `aiwfx-release` ritual's pre-release checks (step 1) cover
"on `main`" and "tree-clean" and "no failing tests *locally*"
— but never "is CI green on the target commit?" The skill
asserts *"Releases ride on green commits only"* in its
Constraints section, but provides no procedural gate that
verifies green before the tag is created and pushed.

The discovery case (this session, 2026-06-11): v0.12.0 shipped
against pre-existing red CI on `main`. Two jobs in
`.github/workflows/go.yml` had been failing across the last 10+
commits (back to E-0030's wrap commit `8cdd0434`):

- **`vuln`** — govulncheck flags `gitops.BlobReader.Read`
  (via `io.ReadFull`) and `status.RenderWorktreeViews` (via
  `fmt.Fprintln`) as reaching stdlib `x509.Certificate.
  VerifyHostname` / `x509.HostnameError.Error` — symbols with
  CVEs in older Go patch levels. CI's `go-version` pin in the
  workflow file lags behind the stdlib backport that fixes
  these.

- **`lint`** — six pre-existing findings: 3 `gocritic`
  (two `appendAssign` in `internal/cli/integration/
  authorize_scenarios_test.go`, one `rangeValCopy` in
  `internal/cli/integration/branch_scenarios_helpers_test.go`),
  2 `gofumpt` (formatting in `internal/policies/
  m0162_ac4_pin_call_shape_test.go` and `internal/policies/
  trailer_order_matches_constants.go`), 1 `govet` (shadowed
  `err` in `internal/policies/m0162_ac2_build_tag_test.go`).

The local `go test -race ./...` and `golangci-lint run`
checks the operator runs pre-commit don't catch the same set:
- The vuln check requires `govulncheck`, which isn't in the
  default local-validation set named in CLAUDE.md §"How to
  validate changes."
- The lint check does catch the 6 findings locally, but the
  operator has been ignoring them as "pre-existing on main" and
  not introduced by the current patch — exactly the rationale
  this gap closes.

## Why it matters

A release tag is the durable consumer-facing artifact. If
`tagged ⇔ CI green` doesn't hold, downstream consumers can't
trust the tag as evidence of release-readiness. The skill's
Constraints section already asserts the invariant; the
procedure didn't enforce it.

The discipline failure compounded: the operator (Claude this
session) ran the ritual, saw local tests green, and crossed the
tag gate without checking CI. The aiwfx-release skill walks an
LLM through this exact sequence and didn't surface the missing
step.

This is the structural complement of G-0242: there, the rule
was implicit in skill bodies and didn't survive `/compact`;
here, the procedural step was implicit in *"CI must be green"*
prose and wasn't part of the executable workflow. Same failure
shape, different ritual.

## Direction

Three coordinated fixes in one wf-patch:

**Fix 1 — lint findings.** Apply gofumpt to the two named files.
Refactor the two `appendAssign` sites (`amended := append(...)`)
to capture into the right slice or use a different pattern.
Rename the shadowed `err` in the test to avoid the collision.
Six mechanical edits across four files.

**Fix 2 — vuln findings.** Per CLAUDE.md §Dependencies, the CI
workflow pins `go-version: "1.25"` and the floor in `go.mod` is
`go 1.24`. The govulncheck output points at stdlib symbols
fixed in a newer 1.25.x patch — bumping the pin to the latest
patched 1.25.x picks them up without changing the consumer
floor. (If the CVEs aren't fixed in any 1.25.x, the alternative
is bumping to 1.26 in CI; same pattern.) Verify by re-running
govulncheck locally against the bumped toolchain.

**Fix 3 — aiwfx-release skill.** Add a sub-step to step 1
("Pre-release checks") naming the CI-green check explicitly:

```
- Most recent `go.yml` run on the target commit is green.
  Run: `gh run list --workflow=go.yml --branch=main --limit 1`.
  If red, stop and resolve before crossing the Commit gate.
  Releases ride on green commits — the Constraints section
  asserts this; this step is where the assertion binds.
```

Plus a corresponding mention in the Constraints section so the
two surfaces (pre-release checks + Constraints) name the same
chokepoint.

## Test surface

- **Fix 1 (lint):** the existing `golangci-lint run` CI job
  goes green on the patch's HEAD. No new test required —
  the lint job IS the test.
- **Fix 2 (vuln):** the existing `vuln` CI job goes green.
  Same shape — the workflow IS the test.
- **Fix 3 (skill):** structural assertion only — the skill body
  contains the new step text and the Constraints reference.
  Not mechanical; same advisory shape as G-0242's CLAUDE.md
  edit. Verification is the operator running the ritual on
  the next release and the step being there.

The patch's release closure verifies fix 3 in practice — it
goes through aiwfx-release with the new step and either catches
the next red-CI release attempt or doesn't.

## Source

v0.12.0 release, 2026-06-11. Tag shipped against red CI
because the ritual's procedural step was missing; surfaced
during post-release verification when the operator (Claude)
ran `gh run list` after the push and saw the failures.

## Closing this gap

When the wf-patch lands all three fixes, this gap promotes to
`addressed` with `--by-commit <sha>`. The v0.12.0 tag is then
yanked from origin and re-cut at the new green HEAD, per the
release-process discussion in the same session.
