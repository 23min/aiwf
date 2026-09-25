---
id: G-0711
title: BlobReader reports a missing path containing whitespace as a parse error
status: open
discovered_in: M-0333
---
## What's missing

`parseBatchHeader` in `internal/gitops/catfile.go` splits a `git cat-file --batch` header on whitespace and recognizes a not-found answer only as exactly two fields ending in `missing`. Git echoes the request back in that answer (`<input> missing`), so when the requested path is absent at the commit and contains whitespace, the line has more than two fields and the shared `request` returns a parse error instead of `ErrBlobMissing`. Both `BlobReader.Read` and `BlobReader.Stat` go through it. A path with whitespace that does exist reads correctly, since the found header does not repeat the input.

The same defect falsifies the `//coverage:ignore` on the parse-error branch in `request`, which says that branch is "not reproducible against a healthy git cat-file --batch subprocess".

Measured in the devcontainer (`go1.25.11 linux/amd64`, git 2.54.0) at commit `ce082fee7` with a throwaway test in `internal/gitops`: a repository with one commit holding `a b.md`, one `BlobReader`, reads at `HEAD`. Expected `ErrBlobMissing` for each absent path; observed:

```
Read "nospace.md"     missing=true content="" err=gitops: blob missing at commit:path
Read "with space.md"  missing=false content="" err=gitops: cat-file --batch size parse "missing": strconv.Atoi: parsing "missing": invalid syntax
Read "two sp aces.md" missing=false content="" err=gitops: malformed cat-file --batch header: "HEAD:two sp aces.md missing"
Read "a b.md"         missing=false content="hello\n" err=<nil>
Stat "with space.md"  missing=false err=gitops: cat-file --batch size parse "missing": strconv.Atoi: parsing "missing": invalid syntax
```

## Why it matters

`ErrBlobMissing` is the documented signal that a file did not exist at a commit, and `readStatusAt` in `internal/check/fsm_history_walker.go` branches on it to skip the pair. For a path that is absent at the commit and contains whitespace, it gets a hard error instead, and the error names a protocol parse, which points a reader at the wrong layer. The guidance fence in `internal/policies/guidance_fence.go` also reads through the pump, but asks only for paths present at the revision read and for two fixed names, so it does not meet this case.
