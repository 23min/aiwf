---
id: G-0112
title: STATUS.md pre-commit regen produces merge conflicts on a derived artifact
status: addressed
addressed_by_commit:
    - 63acc40
---
## What's missing

The pre-commit hook regenerates `STATUS.md` on every commit by running `aiwf status --format=md` over the working tree. Any feature branch that mutates planning state rewrites the file as a side effect; when two such branches merge, both sides have legitimately regenerated `STATUS.md` from their own tree state, producing a conflict on a fully-derived artifact. Git's three-way merge cannot compute the right answer — which is "regenerate from the merged tree" — because that requires re-running the verb, not text-merging two rewrites.

## Why it matters

The conflict tax lands on every merge that combines two planning-touching branches, regardless of the surface either branch was actually editing. The conflict carries no real information — both branches were right about their own state — but the operator still has to open the file, resolve it (typically by deleting one side and trusting the next post-merge commit to regenerate), and push. The friction compounds across the project's life and falls disproportionately on the contributor doing the merge, not the one whose work introduced the regeneration. The artifact's stated purpose (a current snapshot a reader can open on github.com) is also undermined when merges drag stale regenerations forward until the next commit fires the hook again.
