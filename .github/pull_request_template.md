<!--
Every PR description must:
  1. Cite the Issue (or Discussion that converged into one). Drive-by PRs without prior conversation will be asked to open one.
  2. Re-assert the issue's acceptance criteria as a checklist and show how each is met.
  3. Note any principles-checklist (CLAUDE.md §"The principles checklist") items at risk and how they were addressed.
  4. Add a `CHANGELOG.md` entry under `[Unreleased]` for any user-visible change. Internal-only PRs should apply the
     `internal-only` label so the CHANGELOG-touch CI check is skipped.

The pre-PR audit is part of the work, not a follow-up. See CLAUDE.md "Pre-PR audit".
-->

Closes #

## Summary

<!-- 1-3 sentences. What changed and why. -->

## Acceptance criteria (from the issue)

<!-- Re-paste the issue's acceptance-criteria bullets, with a note on how each is met (file/test/command). -->

- [ ]
- [ ]
- [ ]

## Principles-checklist conformance

<!-- Walk the checklist; for each item this PR could regress, state how it was preserved. "No risks identified" is acceptable. -->

## CHANGELOG

<!-- Confirm the `[Unreleased]` entry was added, or apply the `internal-only` label and explain why no entry is needed. -->

## Test plan

- [ ] `go test -race ./tools/...`
- [ ] `golangci-lint run`
- [ ] `bash tests/test-install.sh` (if installer or framework sources changed)
