---
id: G-056
title: aiwf render output (site/) is not gitignored; pollutes consumer working tree
status: open
discovered_in: E-14
---

## What's missing

`aiwf render --format=html` writes its output to `site/` by default (overridable via `--out` or `aiwf.yaml: html.out_dir`). Neither `aiwf init` nor `aiwf update` adds the configured output directory to the consumer repo's `.gitignore`. A single render run drops 100+ generated files into the working tree — empirically observed: 136 untracked files after one `aiwf render --format=html` invocation in this repo.

The render is read-only at the kernel level (no commit) and is intended for ephemeral inspection or for publishing via a separate channel (CI artifact, GitHub Pages workflow). It is not intended to live in `git status`.

## Why it matters

Generated artifacts polluting `git status` is exactly the problem `.gitignore` exists to solve. Concrete consequences:

- **`git status` becomes unreadable.** 136 untracked files drown the operator's signal. Real working-tree changes (the file the operator is actually editing) get lost in the noise.
- **Accidental commits become easy.** `git add -A` or `git commit -a` (both common reflexes) capture every generated file. Once committed, the render output has to be reverted in a follow-up — and the commit's been made, so the trailer chain now records a noise event.
- **Wraps around to the kernel's own discoverability rule.** A consumer who runs `aiwf render` for the first time should not be punished with a working-tree mess. The "framework correctness must not depend on the LLM remembering" rule applies here: the LLM (or human) shouldn't have to remember to gitignore the output.

## Possible remedies

1. **`aiwf init` writes the render output dir to `.gitignore`.** Default entry is `site/`; if the consumer has set `html.out_dir` in `aiwf.yaml` before init (rare), use that value. Marker-comment block (e.g. `# aiwf:render-out`) so subsequent edits are detectable.
2. **`aiwf update` reconciles the entry.** When `aiwf.yaml: html.out_dir` changes (or the marker block is missing), `aiwf update` rewrites the marker block to match. Idempotent. Output is loud about the change in the same shape as the `tdd.default` migration in [G-055](G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md):

   ```
   .gitignore:
     + site/   (aiwf render output)
   ```

3. **`aiwf render` warns when its output dir is not gitignored.** Belt-and-braces: even if init/update missed it, the verb itself prints a one-line warning the first time the output lands in an un-ignored path. Cheap, high-signal.

The init/update path (1 + 2) is the load-bearing fix. The render-time warning (3) catches the case where the consumer has a custom `.gitignore` workflow that strips marker blocks.

## Out of scope

- Whether `site/` should be the right default name (it is — matches `mkdocs`, `hugo`, `jekyll` convention).
- Whether the render should commit its output (it shouldn't — this is the subject of separate decisions about ephemeral vs. published artifacts).
