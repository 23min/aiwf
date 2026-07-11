---
id: G-0403
title: mutate-hunt silently reports empty results for a subpackage + /... pattern
status: open
---
## What's missing

`.github/workflows/mutate-hunt.yml`'s `pkg_pattern` input takes a Go
package pattern and passes it straight through to `gremlins unleash
... <pkg_pattern>`. The input's own description gives two example
shapes: a specific package (`./internal/version`) and the whole tree
(`./internal/...`) — both work correctly. The natural third shape an
operator reaches for — a **specific subpackage plus the wildcard
suffix** (`./internal/stresstest/...`) — is silently wrong.

Confirmed directly, twice, with the locally-installed `gremlins`
binary (`go install github.com/go-gremlins/gremlins/cmd/gremlins@latest`,
v0.6.0):

- `gremlins unleash --dry-run ./internal/entity/...` hard-errors:
  `go: warning: "./internal/entity/.../..." matched no packages` /
  `no packages to test` / `ERROR: failed to gather coverage`. Gremlins
  appears to append its own `/...` to whatever pattern it's given,
  producing a doubled, invalid suffix.
- `gremlins unleash --dry-run ./internal/stresstest/...` does **not**
  error. It prints `Gathering coverage... done in ~1s` then `No
  results to report.` — a clean-looking, zero-exit-code "nothing
  found," indistinguishable from a genuinely fully-covered package
  with no mutable code. For a package the size of
  `internal/stresstest`, that's impossible.
- The fix in both cases: drop the trailing `/...` —
  `gremlins unleash --dry-run ./internal/stresstest` (no suffix)
  immediately finds dozens of real `RUNNABLE` mutants across the
  package.
- The workflow's own literal default, `./internal/...` (top-level,
  with the suffix), is unaffected — it gathers coverage across the
  whole tree correctly. The bug is specific to a **subpackage path
  combined with the wildcard suffix**, not wildcards in general.

This was hit for real: a `workflow_dispatch` run of `mutate-hunt`
against `./internal/stresstest/...` (scoping mutation testing to the
epic that had just landed a large batch of new stress-testing code)
completed in 27 seconds reporting nothing — a result that read as
"clean" until manually inspected via `gh run view --log`.

## Why it matters

The silent-empty case is strictly worse than the hard-error case: a
malformed top-level wildcard at least fails loudly and obviously.
Scoping the pattern to a subpackage — the exact use case
`workflow_dispatch`'s manual-cadence design assumes ("after a
substantive test-suite change" to *some part* of the codebase, per
the workflow's own header comment) — produces a result that looks
identical to "ran cleanly, nothing to report," inviting false
confidence that a package was checked when it wasn't.

## Direction

Worth a real decision, not assumed. Candidate directions:

- **Normalize the input mechanically** — strip any trailing `/...`
  from `pkg_pattern` in the workflow before invoking `gremlins`
  (a one-line `sed`/bash transform), so both spellings behave
  identically regardless of which one an operator types. Removes the
  footgun without relying on anyone remembering a documentation
  caveat — consistent with this repo's general preference for
  mechanical guarantees over operator discipline.
- **Document the caveat** — extend the input's `description` field to
  explicitly warn against combining a subpackage with `/...`. Cheaper,
  but leaves the trap in place for anyone who doesn't read closely
  (this session included).
- **Validate and fail loudly** — after gremlins reports zero mutants
  for a non-empty package pattern, treat that as a workflow failure
  rather than a clean run, so a silently-wrong pattern can't be
  mistaken for "nothing to find." More robust but more code to
  maintain in the workflow.

## Scope

`.github/workflows/mutate-hunt.yml`'s `pkg_pattern` input handling
(and/or its description text). Possibly worth a small upstream report
to `go-gremlins/gremlins` too, since the inconsistency (hard error for
one shape, silent empty for another) looks like a real gremlins-side
bug in its own pattern-normalization logic, not just a docs gap on
this repo's side.