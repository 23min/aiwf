---
id: G-0105
title: TestProvenanceCheck_UntrailedEntityCommit asserts obsolete warning-severity substring format
status: open
---
# Problem

`cmd/aiwf/provenance_check_test.go:100` asserts the substring `"warning provenance-untrailered-entity-commit"` against the rendered output of `aiwf check`. The current renderer emits the severity *after* the code in parentheses — `"provenance-untrailered-entity-commit (warning) × 1 — …"` — so the substring no longer matches and the test fails with a misleading message:

```
provenance_check_test.go:101: expected warning severity; got:
  provenance-untrailered-entity-commit (warning) × 1 — commit <SHA> touched G-0001 with no aiwf-verb: trailer
```

The first assertion in the same test (line 95) checks for `"provenance-untrailered-entity-commit"` (without the leading "warning") and passes — only the severity-format follow-up at line 101 is wrong.

# Reproduction

```
go test -count=1 -race -run TestProvenanceCheck_UntrailedEntityCommit ./cmd/aiwf/...
```

Fails on `main` (HEAD `cd87ced` at the time of filing) without any local changes.

# Root cause

The kernel's check-renderer was updated to emit `<code> (<severity>) × N — <detail>` (the format visible in every other `aiwf check` invocation today). The test never got updated to match. `t.Errorf` (not `t.Fatalf`) means the failure is non-blocking inside the test but still surfaces as `FAIL` on the package.

# Scope of the fix

Single-line change on `cmd/aiwf/provenance_check_test.go:100`:

```go
if !strings.Contains(out, "(warning)") || !strings.Contains(out, "provenance-untrailered-entity-commit") {
```

Or, tighter:

```go
if !strings.Contains(out, "provenance-untrailered-entity-commit (warning)") {
```

The second form is preferred — it pins the severity-to-code association the test is asserting, instead of treating the two substrings as independent.

Grep confirms `cmd/aiwf/provenance_check_test.go:100` is the only call site in the repo with the obsolete format. No other tests need updating.

# Why this wasn't caught earlier

Pre-commit hook only runs `go test ./internal/policies/...`. The full suite runs on pre-push and in CI. This test was likely failing in CI for some window before today — worth a quick CI history check before closing, to date the regression.

# Suggested resolution

`wf-patch` ritual: branch `fix/provenance-check-severity-format`, single-line edit, commit, push, PR. ~10 minutes including verification.

# Why this isn't urgent

The failing test fires `t.Errorf`, not `t.Fatalf`, so the assertion that actually matters (the finding code appears in the output) still runs and still passes. The misleading-severity-format-match is the only false signal. CI may already be red on `main` because of this — worth confirming and prioritizing accordingly.
