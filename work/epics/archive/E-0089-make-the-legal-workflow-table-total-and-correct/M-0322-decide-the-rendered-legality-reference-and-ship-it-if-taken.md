---
id: M-0322
title: Decide the rendered legality reference and ship it if taken
status: done
parent: E-0089
depends_on:
    - M-0321
tdd: advisory
acs:
    - id: AC-1
      title: The render decision lands before any render code
      status: met
    - id: AC-2
      title: If taken, the committed render matches a fresh render, enforced
      status: met
---
## Goal

Decide whether a human-readable legality reference is generated from the table, and
ship it if the decision says so.

## Context

A reader wanting to know what aiwf permits has two surfaces today: `docs/workflows.md`,
which is narrative and measured three of seven steps working when walked literally,
and the two catalogs under `docs/design/`, which are working papers whose structure
is pinned but whose claims are held by nothing.

Once the table is total and target-bearing it can answer the question directly, and
a generated document can be checked against it. `aiwf render roadmap --write`
provides a precedent for a derived view that is regenerated rather than maintained.
This milestone adds a fresh-render comparison for the legality reference if the
decision is to ship it.

D-0077 deliberately leaves shipping the render open. That is why the decision is
this milestone's first criterion rather than an assumption in its goal — a render
is a new shipped surface with its own maintenance and its own failure modes, and
the argument for it is not settled by the table being total.

## Acceptance criteria

### AC-1 — The render decision lands before any render code

A decision record states whether the legality reference is generated, and its
consequences name what happens to `docs/workflows.md` and to the two catalogs under
`docs/design/` either way. No render code is committed before that record is
accepted.

### AC-2 — If taken, the committed render matches a fresh render, enforced

Where the decision is to ship: a committed document is produced from the table, and
a check fails when the committed bytes differ from a fresh render. Editing the
document by hand reddens the suite. Where the decision is to decline: this criterion
is closed as not applicable, with the decision cited.

## Constraints

- **The decision governs the code, not the reverse.** Building the render first and
  recording the decision after inverts the order this criterion exists to hold.
- **Generated means generated.** A document that is rendered once and then hand-
  maintained is the drift this milestone is meant to end; the comparison check is
  what makes the claim true rather than intended.
- **Do not absorb the catalogs' disposition.** What happens to the two `docs/design/`
  documents is named by the decision, not executed here.

## Design notes

The render can only carry what the table carries. Reasoning — why a contract may go
straight to rejected, why scope reach is a three-edge tree — lives in ratified
decisions and does not become renderable by this work. A render that implies
otherwise would be worse than none, so the shape of what it omits belongs in the
decision.

## Out of scope

- Retiring or archiving the two `docs/design/` catalogs.
- Rewriting `docs/workflows.md`.
- Any render surface beyond the legality table.

## Dependencies

- M-0321 — a render of a table with holes would publish the holes.

## References

- D-0077 — leaves this decision open by design
- `aiwf render roadmap` — the derived-view precedent
- `docs/workflows.md`, `docs/design/legal-workflows-audit.md`,
  `docs/design/legal-workflows-first-principles.md` — the surfaces the decision must
  account for

## Decisions made during implementation

D-0101 — publish the generated reference through repository development tooling;
its scope and documentation consequences are owned by that decision.

## AC-1 observation

Measured on 2026-09-22 in the Linux amd64 devcontainer, on this milestone's branch.
`git log -4 --oneline` showed decision creation at `f4621d379` followed by
acceptance at `d68ec03b9`. At that accepted commit,
`git diff 4cd8342e1 HEAD --name-only` was expected to show planning files only.
It listed only the decision and this milestone spec; no render code was committed.
The renderer's implementation begins after that acceptance. This criterion records
a sequencing observation, not a red/green code property.

## Release note

The repository now provides a generated workflow legality reference showing
declared transitions, their conditions and outcomes, applicability exclusions,
and global restrictions. Its freshness test rejects changes that leave the
committed reference out of sync with the specification. Regenerate it with
`go run ./cmd/workflow-reference` from the repository root.

## Validation

Measured on 2026-09-22 in the Linux amd64 devcontainer with Go 1.25.11, on this
milestone's branch. Source under test is committed at `1de39375c`.

- `make test`: expected all packages to pass; final run exited 0. The initial
  run failed `TestPolicy_LayeringDirection` because the development command
  lacked a tier; the command is registered and the final policy suite passed.
- `make vet`: expected no diagnostics; ordinary, stress-tagged and testpins-tagged
  vet all exited 0.
- `GOLANGCI_LINT_CACHE="$(git rev-parse --absolute-git-dir)/golangci-lint-cache" golangci-lint run --allow-serial-runners`:
  expected no findings; exited 0 with `0 issues.`
- `go build -o bin/aiwf-diag ./cmd/aiwf` and
  `go build -o /tmp/aiwf-m0322-generator ./cmd/workflow-reference`:
  expected successful builds; both exited 0.
- `/tmp/aiwf-m0322-generator --out /tmp/aiwf-m0322-fresh-render.md` followed by
  `cmp docs/reference/workflow-legality.md /tmp/aiwf-m0322-fresh-render.md`:
  expected identical bytes; both exited 0. Repeated generation was identical.
  Running the binary with an unknown flag returned usage exit 2; a destination
  with a missing parent returned internal-error exit 3 and an `--out` remedy.
- `go test -count=1 ./internal/workflows/spec ./cmd/workflow-reference`:
  expected the renderer and generator checks to pass; exited 0. Tests compare
  exact rows within each table, preserve conditions and refusal metadata,
  distinguish empty binary operands from unary operators, escape table content,
  and compare committed bytes to a fresh render. Generator tests cover repeated
  replacement, invocation errors, write failure, help, and cancellation that
  preserves the destination. The focused coverage profile reports 100% statement
  coverage for renderer functions and command `run`; `main`'s exit delegation
  was exercised through the built binary, outside that profile.
- A copied reference with a hand edit was supplied to the freshness test through
  a Go overlay. `go test -overlay <overlay> -count=1 -run '^TestReferenceMatchesCommittedDocument$' ./internal/workflows/spec`
  was expected to fail; it reported `workflow reference is stale` and exited 1.
  The committed document was not modified by this probe.
- `python3 /tmp/aiwf-m0322-probes.py`: expected each injected defect to fail an
  assertion without a compilation error; all probes did. They removed conditions,
  globals, applicability or a declaration; changed a NoOp or requested target;
  wrote empty output; and ignored cancellation. The probe script and logs are
  session-local artifacts, not permanent repository checks.
- Package-scoped `gremlins unleash` with `--workers 1 --timeout-coefficient 15`
  killed every mutant in the new renderer and generator. The spec run also
  exercised existing code: it reported 28 killed, 0 lived and 3 uncovered mutants
  in existing lookup code; the command run reported 3 killed, 0 lived and 0
  uncovered. Diff-scoped runs reported no results and provide no mutation evidence.
- Generated Markdown was inspected directly and converted to HTML with the
  repository's goldmark/GFM dependency. HTML parsing found the expected table
  widths and declaration counts, resolved both document links, and preserved
  `self.tests == ""` visibly. No browser executable or browser tool was available;
  browser visual inspection was not performed.
- `bin/aiwf-diag check --since 4cd8342e1` after the AC promotions: expected no
  errors; exited 0 with 0 errors and 20 warnings. These comprise the existing
  archive and advisory-TDD warnings, the active epic having no remaining draft
  milestone, and advisory-TDD warnings for this milestone's met criteria. No TDD
  phase events were recorded; the milestone uses advisory TDD.

Full race/CI validation was not run at this local milestone boundary.

## Deferrals

None.

## Reviewer notes

Independent code review of the full milestone through `906fa597d`: approve, with
no blocking findings. Independent design review: keep. Explicit enum switches
retain the relationship between named values and labels; shorter ordinal-indexed
arrays add coupling without simplifying the rendering boundary.

The reference's instruction to consider every row for a request clarifies the
reading boundary in D-0101 without introducing a rule-resolution precedence.
Scoped doc-lint: clean for the changed documentation surface.
