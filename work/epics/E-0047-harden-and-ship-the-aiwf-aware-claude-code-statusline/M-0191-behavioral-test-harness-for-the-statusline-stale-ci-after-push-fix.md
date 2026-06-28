---
id: M-0191
title: Behavioral test harness for the statusline + stale-CI-after-push fix
status: in_progress
parent: E-0047
tdd: required
acs:
    - id: AC-1
      title: Behavioral test runs statusline.sh against a fixture and asserts real output
      status: met
      tdd_phase: done
    - id: AC-2
      title: CI segment shows pending when the run's headSha differs from local HEAD
      status: met
      tdd_phase: done
    - id: AC-3
      title: Statusline cache key includes HEAD sha so a push invalidates a stale CI result
      status: open
      tdd_phase: green
---
## Deliverable

A behavioral test harness for `.claude/statusline.sh` (G-0187), plus the stale-CI-after-push fix (G-0189) as its first target.

**Harness (G-0187).** A Go test that writes a known-shape transcript fixture + a temp git repo, streams a stub stdin JSON through `exec.Command("bash", scriptPath)`, strips ANSI from the rendered output, and asserts the *segment shapes* from real output (token count, sync ahead/behind, CI segment). This replaces the regex-over-source assertions in `internal/policies/statusline_content_test.go` (which never run the script — the `||` binding bug nearly shipped because of exactly that) with assertions that exercise behavior.

**Stale-CI fix (G-0189).** The CI segment compares the latest run's `headSha` against local `git rev-parse HEAD`; on mismatch it renders `… ci` (gray, pending) instead of the previous run's stale `✓`. HEAD is folded into the cache key so a push auto-invalidates.

## Why combined (per the epic)

The harness proves itself by catching and fixing the clearest statusline bug; the stale-CI fix is its first behavioral target. Every later milestone (M2–M4) asserts against this harness.

### AC-1 — Behavioral test runs statusline.sh against a fixture and asserts real output

`TestStatusline_M0191_AC1_RendersRealSegments` runs `.claude/statusline.sh` end-to-end against a hermetic temp git repo (wired to a bare upstream, one commit ahead), a JSONL transcript fixture, and a stubbed `gh` on PATH; it strips ANSI and asserts the rendered segments — token count (`6k`), contiguous branch+sync (`main↑1`), repo name, and the CI glyph. This exercises behaviour rather than grepping the source — the axis G-0187 named missing. The structural M-0153 source-grep tests stay alongside; they guard cross-platform / reflow-robustness shapes a single-OS run cannot exercise.

### AC-2 — CI segment shows pending when the run's headSha differs from local HEAD

`TestStatusline_M0191_AC2_StaleCIShowsPending`: a CI run whose `headSha` differs from local HEAD renders the gray stale-pending glyph `… ci`, not the run's verdict (`✓`). The test fails against the pre-fix code (which showed the stale `✓`). The fix queries `--json …,headSha`, compares against `git rev-parse --verify HEAD`, and renders `…` on mismatch; the main-fallback (`m:` prefix) skips the check, covered by `TestStatusline_M0191_MainFallbackSkipsStaleness`.

### AC-3 — Statusline cache key includes HEAD sha so a push invalidates a stale CI result

`TestStatusline_M0191_AC3_CacheKeyIncludesHEAD`: HEAD is folded into the CI cache key, so a new commit invalidates a cached verdict instead of the 45s TTL serving the pre-commit result. The test caches a `✓` for commit A, commits B, and asserts the next render re-fetches and shows `✗` (not the stale `✓`). The positive cache path — a same-HEAD re-render within the TTL serving the cached verdict — is pinned by `TestStatusline_M0191_CacheHitServesWithinTTL`.

## Work log

- **AC-1 / AC-2 / AC-3 — met.** Harness + stale-CI fix landed in `64ae762c` (`fix(statusline): render stale CI as pending; add behavioral harness`). Five behavioral tests in `internal/policies/statusline_behavioral_test.go` exercise the three ACs plus the cache-hit and main-fallback paths. The fix is mirrored into the embedded copy `internal/skills/embedded-statusline/statusline.sh`, kept byte-identical per the M-0155 drift test (`TestM0155_AC1_StatuslineEmbedded`).

## Validation

- `go test ./internal/policies/ -run 'TestStatusline_M0191|TestM0155_AC1'` — green: AC-1/2/3 + CacheHitServesWithinTTL + MainFallbackSkipsStaleness + embed drift.
- `make check-fast` — exit 0 (vet + lint + full test suite). Full `make ci` runs at the wrap-merge into the epic branch.
- Human-verified renders: `✓ ci` (success@HEAD), `… ci` gray (stale), `✗ ci` (failure@HEAD), `→ ci` (in-progress), and `✓ ci` on an unborn branch (no spurious `…`).

## Reviewer notes

- An independent fresh-context reviewer approved the diff with no blocking findings; three advisories were addressed inline — `git rev-parse --verify` for the unborn-branch edge, AC-1 strengthened to the contiguous `main↑1` assertion, and a positive cache-hit test — plus a main-fallback test covering the `expected_sha=""` arm the reviewer flagged as otherwise unexercised.
- The structural M-0153 source-grep tests were deliberately kept (not replaced): they assert cross-platform / reflow-robustness shapes (`tail -r || tac`, default-IFS sync parse, `GIT_OPTIONAL_LOCKS` ordering) a single-OS behavioral run cannot exercise. The harness complements them; it does not supersede them.
- Branch-model correction: the milestone-activation promote was initially placed on the milestone branch; it was relocated to the epic branch per ADR-0010, clearing the `promote-on-wrong-branch` finding before wrap.
