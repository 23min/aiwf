---
id: G-0198
title: branchparse regex accepts prefix-id mismatch (epic/M-..., milestone/E-...)
status: addressed
discovered_in: M-0103
addressed_by_commit:
    - 8f1227c72ac932d49cbb0710eedfcc0382898288
---
## What's missing

`internal/branchparse/branchparse.go` exposes:

```go
var branchEntityPattern = regexp.MustCompile(`^(?:epic|milestone|patch)/([EeMmGg]-\d+)(?:-|$)`)
```

The regex enforces a ritual *prefix* (epic/milestone/patch) and a ritual *id* (`[EeMmGg]-\d+`), but does NOT enforce coherence between them. The following all match and yield an entity id:

| Branch                          | Parsed id |
|---------------------------------|-----------|
| `epic/E-0001-foo`               | `E-0001`  |
| `epic/M-0001-foo`               | `M-0001`  |
| `milestone/E-0001-foo`          | `E-0001`  |
| `patch/M-0042-foo`              | `M-0042`  |
| `patch/E-0042-foo`              | `E-0042`  |

Only the first row is conventionally correct per ADR-0010.

## Why it matters

Today only the `aiwf status --worktrees` correlator consumes the parsed id (`internal/cli/status/worktrees.go`). A hand-created worktree on `epic/M-0001-foo` would silently correlate to milestone M-0001 under an "epic/" worktree — operator-side confusion in status listings.

M-0103's preflight uses `ParseEntityFromBranch != ""` as a *boolean* (does the shape match?). The parsed id is not consumed; prefix-id coherence doesn't affect refusal behavior.

M-0106's planned `isolation-escape` finding compares commit branch names via first-parent reachability — doesn't parse id from branch. Per E-0030 corner case 12: "branch identity, not path, is the load-bearing axis."

## Scope

**Out of E-0030 scope.** No E-0030 milestone deliverable depends on prefix-id coherence in branchparse. The hygiene improvement belongs in a standalone follow-up that owns the worktree correlator's stricter behavior end-to-end.

## Fix shape

Tighten the regex to enforce per-prefix id type:

```go
var branchEntityPattern = regexp.MustCompile(
    `^(?:epic/(E-\d+)|milestone/(M-\d+)|patch/([Gg]-\d+))(?:-|$)`,
)
```

`ParseEntityFromBranch` then returns the first non-empty submatch. Existing tests in `internal/branchparse/branchparse_test.go` cover the positive shapes; add cases for the four mismatch combinations above asserting empty return.

Audit downstream consumers (`internal/cli/status/worktrees.go`, M-0103's preflight, any future M-0106 use) for behavior change. The preflight is unaffected (boolean check); the correlator becomes stricter (silently miscorrelated hand-typo'd branches now return no correlation, surfacing the typo at lookup time).
