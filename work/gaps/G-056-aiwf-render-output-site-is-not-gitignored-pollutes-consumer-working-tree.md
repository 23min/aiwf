---
id: G-056
title: aiwf render output (site/) is not gitignored; pollutes consumer working tree
status: open
discovered_in: E-14
---

## What's left

The kernel-level reconciliation of the html render output dir landed in `056139d` (May 3, "I3 step 4b — init/update reconcile gitignore for html out_dir"): `aiwf init` and `aiwf update` consult `aiwf.yaml: html.commit_output` and add or remove the `<out_dir>/` line idempotently. Remedies (1) init-writes and (2) update-reconciles from the original gap framing are implemented in `internal/initrepo/initrepo.go`'s `ensureGitignore`, with `htmlOutDirIgnore` and `htmlOutDirCandidates` helpers. The fix predates the gap's filing.

What is still owed: **defense-in-depth at render time.** When `aiwf render --format=html` writes its output and the destination is not covered by `.gitignore`, the verb should print a one-line warning. Cases the init/update fix doesn't catch:

- The consumer hasn't yet run `aiwf init` / `aiwf update` after configuring or changing `html.out_dir`.
- A team's `.gitignore` workflow strips marker-managed blocks (rare but real).
- A consumer pointed `--out` at an ad-hoc path different from `aiwf.yaml: html.out_dir`; reconciliation only knows about the configured value.

## Why it still matters

The original symptom — 100+ generated files in `git status` after one render — recurs in any of the cases above. `aiwf render` is the highest-signal place to surface the warning: the operator is one keystroke away from the misconfiguration and most likely to act on it.

## The real fix

At the end of `aiwf render --format=html`, probe whether the resolved output dir is gitignored (e.g. `git check-ignore -q <out_dir>/` — cheap; respects user-authored entries that don't match the marker block). If not, emit one stderr line:

```
warning: <out_dir>/ is not gitignored; rendered files will appear in `git status`.
         Run `aiwf update` to reconcile, or set `html.commit_output: true` to track them.
```

Advisory output only; no exit-code change. Tested against three states: gitignored (silent), un-ignored (warns), `commit_output: true` (silent — operator opted in).

Once shipped, promote G-056 to `addressed` with the implementing commit as `--by-commit`.

## Out of scope

- A `check` finding for un-gitignored render output. The render verb itself is the right place — the warning fires when the operator is most likely to act on it.
- Whether `site/` is the right default name (it is — matches mkdocs/hugo/jekyll convention).
- Whether render should commit its output (separate decision, controlled by `html.commit_output`).
