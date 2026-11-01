---
id: G-0760
title: Comments on the orphan walk's and the oracle's branch filters misdescribe both
status: open
---
## What's missing

`listRitualHeads` (`internal/check/reflog_walk.go`) picks the branches whose
reflogs `WalkOrphanedAICommits` reads, and its comments say it uses the
isolation-escape oracle's filter: its doc comment says it "Matches the existing
oracle's filter (intentionally; same set of refs the rule polices)", and the
comment above the filter says "Same filter the oracle uses
(internal/cli/check/isolation_escape_oracle.go::listRitualBranches). Duplicated
here to keep this helper standalone — the canonical filter is the oracle's; if
the shape changes, both update together." The filter it calls is `ritualShape`,
whose own comment in the same file calls it "a deliberately loose prefix-only
check — looser than branchparse.ParseEntityFromBranch", the filter
`listRitualBranches` applies. The same doc comment lists the shapes it keeps as
`epic/M-NNNN-..., milestone/M-NNNN-..., patch/...`, which describes neither
filter: the oracle wants an `E-` id after `epic/` and returns no entity for
`patch/some-topic` (below), and the walk takes any name under the three
prefixes.

The oracle's own doc comment is off the same way. `listRitualBranches`
(`internal/cli/check/isolation_escape_oracle.go`) documents its filter as the
trunk branch plus `epic/E-NNNN-..., milestone/M-NNNN-..., patch/...`, but the
`ParseEntityFromBranch` it calls (`internal/branchparse`) matches
`^(?:epic/([Ee]-\d+)|milestone/([Mm]-\d+)|patch/([Gg]-\d+))(?:-|$)`, so it keeps
a `patch/` branch only with a `G-` id.

Measured 2026-11-01 in a clone of `main` at `64a8fe45c`, go1.25.12, Linux, with
a probe test added in `internal/check` (package `check`, importing
`internal/branchparse`) that calls both filters:

```go
func TestZZProbeFilters(t *testing.T) {
	for _, b := range []string{"patch/some-topic", "epic/typo", "epic/E-0001-some-epic", "patch/G-0001-fix"} {
		t.Logf("%-22s walk ritualShape=%-5v oracle ParseEntityFromBranch=%q", b, ritualShape(b), branchparse.ParseEntityFromBranch(b))
	}
}
```

```
$ go test -count=1 -run TestZZProbeFilters -v ./internal/check/
    zz_probe_test.go:11: patch/some-topic       walk ritualShape=true  oracle ParseEntityFromBranch=""
    zz_probe_test.go:11: epic/typo              walk ritualShape=true  oracle ParseEntityFromBranch=""
    zz_probe_test.go:11: epic/E-0001-some-epic  walk ritualShape=true  oracle ParseEntityFromBranch="E-0001"
    zz_probe_test.go:11: patch/G-0001-fix       walk ritualShape=true  oracle ParseEntityFromBranch="G-0001"
--- PASS: TestZZProbeFilters (0.00s)
ok  	github.com/23min/aiwf/internal/check	0.005s
```

The walk takes `patch/some-topic` and `epic/typo`; the oracle, which returns no
entity for them, does not.

## Why it matters

The comment above the filter tells a maintainer to keep the walk's filter in step
with the oracle's, while `ritualShape`'s comment says the difference is
deliberate, so that the walk reads the reflogs of malformed ritual branches the
oracle does not index. Following the first comment would remove that coverage;
leaving it means anyone working out which branches the orphan warning covers
from `listRitualHeads`' own documentation gets a set the walk does not read, and
one working out what the oracle indexes from `listRitualBranches`' gets `patch/`
branches it skips. That has happened: the gap-truth audit's finding on G-0060
(`docs/audits/gap-truth-audit.md`) says `isolation_escape_oracle.go:319` "names the
same set" as `ritualShape`.

## Relates to

G-0644 records a false positive in the orphan walk these branches feed.
`docs/audits/structural-sweep-audit.md` lists `reflog_walk.go:146` among
comments "asserting a parity that does not hold".
