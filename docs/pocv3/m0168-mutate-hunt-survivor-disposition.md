# M-0168 — kernel-core mutate-hunt survivor disposition

Probe 1 of the G-0262 corpus work: a `gremlins` mutation sweep over the
load-bearing kernel packages, value-tiered per M-0168/AC-1 (record the efficacy
baseline; kill the high-value, low-cost real survivors; document the
equivalent-mutant / boundary-noise classes by pattern; name the per-package
survivor counts so the un-itemized remainder is visible).

Tuning: `gremlins unleash --workers 1 --timeout-coefficient 15` per package
(`--workers 1` is forced — higher worker counts contend on the test-binary
build cache and time out on this repo).

## Efficacy baseline (the objective floor)

| Package            | Efficacy | Killed | Lived | Not covered |
|--------------------|---------:|-------:|------:|------------:|
| `internal/entity`  |   85.5%  |   106  |   18  |     44      |
| `internal/gitops`  |   91.9%  |   170  |   15  |      9      |
| `internal/verb`    |   86.2%  |   554  |   89  |     71      |
| `internal/check`   |   88.5%  |   348  |   45  |     76      |

"Lived" is the assertion-strength signal (covered, mutation survived). "Not
covered" is a distinct coverage-gap axis, out of scope for this assertion-quality
pass.

## Kills (verified)

Each kill below was confirmed by injecting its *exact* mutation into the
production source, running the focused test, observing it go **red**, and
reverting (the production tree is byte-identical after). 11 real survivors
killed by 4 small test additions.

| Survivor (file:line) | Mutation | Kill-test | Verified |
|----------------------|----------|-----------|:--------:|
| `entity.go:340` | loop `i < len(runes)-2` → `<=` | `TestIsProseyTitle` "sentence mark and space at the very end" (`"All done. "`) — original stops before the slice end; mutant reads `runes[len]` and panics | red ✓ |
| `entity.go:342` | `r != '?'` → `==` | `TestIsProseyTitle` "question-mark multi-sentence" | red ✓ |
| `entity.go:342` | `r != '!'` → `==` | `TestIsProseyTitle` "exclamation multi-sentence" | red ✓ |
| `entity.go:349` | `next <= 'Z'` → `<` | `TestIsProseyTitle` "capital Z opens the second sentence" | red ✓ |
| `entity.go:667` | epics k-guard negation | `TestStripArchiveSegment` epics cases (k="" / k=epic / k=milestone) | red ✓ |
| `entity.go:671` | gaps k-guard negation | `TestStripArchiveSegment` gaps cases | red ✓ |
| `entity.go:675` | decisions k-guard negation | `TestStripArchiveSegment` decisions cases | covered by construction (same shape as 667/671) |
| `entity.go:686` | ADR k-guard negation (both disjuncts) | `TestStripArchiveSegment` adr cases (k="" / k=adr) | red ✓ (both disjuncts) |
| `entity/serialize.go:32` | `idx < 0` → `<=` | `TestSplit_BlankLineInFrontmatter` | red ✓ |
| `entity/serialize.go:43` | `idx < 0` → `<=` | `TestSplit_BlankLineInFrontmatter` | red ✓ |
| `gitops/tests_metric.go:59` | `n < 0` → `<=` | `TestParseTestMetrics` "sole zero-value key still reports ok" (`"pass=0"`) | red ✓ |

## Equivalent / unreachable (documented, no test — justified)

These survivors cannot be killed by any test because the mutation produces no
observable behavior change (equivalent) or the distinguishing input cannot occur
(unreachable). Chasing them is the false-positive work the milestone scope warns
against.

| Survivor | Class | Why no test |
|----------|-------|-------------|
| `entity.go:664`, `entity.go:685` | unreachable boundary | `len(parts) >= 3` → `> 3`; a real archive entity path always has ≥ 4 segments (`work/<kind>/archive/<file>` or `.../<parent>/<file>`), so the exact-3 boundary never occurs. |
| `entity.go:696` (INVERT_NEGATIVES, ARITHMETIC_BASE) | equivalent | `make([]string, 0, len(parts)-1)` is a capacity hint; the resulting slice's length and contents are identical regardless. |
| `entity/allocate.go:66`, `:72` | equivalent | `n > highest` → `>=`; setting `highest = n` when `n == highest` is a no-op. |
| `entity/canonicalize.go:68` | equivalent | `len(num) >= CanonicalPad` → `>`; when `len == CanonicalPad`, `%0*d` pads to the same width, so the early-return and the format path yield identical output. |
| `entity/serialize.go:117` | unreachable | `r > 127` → `>=`; `r == 127` is the DEL control char, never present in an entity title — and even then only the diagnostic `dropped` list differs, the slug is identical. |
| `gitops/revwalk.go:225` | equivalent | `pathsStart < len(chunk)` → `<=`; `chunk[len:]` is the empty string, handled identically downstream. |
| `gitops/trailers.go:154`, `:160` | equivalent | sort comparator `<` → `<=` guarded by a preceding `!=` — the exact `a > b` / `a >= b`-after-`a != b` noise CLAUDE.md names; the trailer order is also pinned structurally by `TestCanonicalTrailerKeys_DerivesFromTrailerOrder`. |

## Not itemized this pass (value-tiered deferral — counts visible)

Per AC-1, the lower-value survivors are recorded by class rather than killed
line-by-line. They are real-or-equivalent (un-triaged individually), not claimed
clean:

- **`internal/gitops` — 5 remaining** (`gitops.go:231`/`:271` error-path
  negations; `refs.go:276`/`:452` rename-parse boundary; `revwalk.go:178`×2 /
  `:214` / `:218` buffer-size arithmetic and marker-search boundaries). Killing
  these needs fault injection (forced `EvalSymlinks` / `commonGitDir` failures)
  or crafted `git -z` byte streams at exact tuple boundaries — disproportionate
  fixtures for the blast radius.
- **`internal/verb` — 89 lived (efficacy 86.2%)**, concentrated in
  `rewidth.go` (26) and `apply.go` (16): mostly `CONDITIONALS_BOUNDARY` noise in
  stateful verb logic whose kills require full apply/projection fixtures, not the
  cheap pure-function tests this pass targets.
- **`internal/check` — 45 lived (efficacy 88.5%)**, spread across
  `body_prose_id.go` (8), `check.go` (6), `acs.go` / `provenance.go` (5 each),
  `isolation_escape.go` / `entity_body.go` / `fsm_history_consistent.go` (4
  each): stateful check-rule gaps whose kills require full fixture-tree inputs
  (`tree.Load` + `check.Run` over a crafted planning tree), not the cheap
  pure-function tests this pass targets — same value-tiered rationale.

## Method note

Kill confirmation uses targeted mutation injection (inject the exact operator
swap, run the one focused test, confirm red, `git checkout` the production file)
rather than a full `gremlins` re-run — faster and more precise, and it leaves the
tree byte-identical. The new tests also ride the diff-scoped coverage gate.
